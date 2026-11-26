"""Private, single-table playback rehearsal behind the Ink & Bone tailnet endpoint."""
import live_score
import contextlib
import functools
import hmac
import http.cookies
import json
import os
import sqlite3
import ssl
import threading
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

HERE = Path(__file__).resolve().parent
STATE = Path(os.environ.get("STATE_DIRECTORY", "."))
ORIGIN = os.environ["INK_ORIGIN"]
DRAFT = json.loads((HERE / "rehearsal.json").read_bytes())
TOKEN = (Path(os.environ["CREDENTIALS_DIRECTORY"]) / "syrinscape").read_text().strip()
GAME_BASE = os.environ.get("INK_GAME_BASE", "https://127.0.0.1:7432")
GAME_TLS = ssl.create_default_context(cafile=os.environ.get("INK_GAME_CA", "/etc/inkandbone/backend.crt"))
LOCK = threading.Lock()
PUBLIC = {"index.html": "text/html", "pilot.js": "application/javascript", "style.css": "text/css"}


def provider(path, method="GET", app=False):
    host = "app.syrinscape.com" if app else "syrinscape.com"
    req = urllib.request.Request("https://" + host + "/" + path, method=method,
                                 headers={"Authorization": "token " + TOKEN, "Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=15) as resp:
        return json.load(resp) if method == "GET" else None


@functools.lru_cache(maxsize=1)
def session_id():
    return provider("config/", app=True)["session_id"]


def state(db):
    row = db.execute("select pos, version, delivered, mood from playback").fetchone()
    return dict(zip(("pos", "version", "delivered", "mood"), row, strict=True))


def public_state(row):
    pos = row["pos"]
    return {"pos": pos, "version": row["version"], "scene": DRAFT["beats"][pos] if pos >= 0 else None,
            "next": DRAFT["beats"][pos + 1]["action"] if pos + 1 < len(DRAFT["beats"]) else None}


def send_cue(db, category, ident, verb, version):
    provider(f"online/frontend-api/{category}/{ident}/{verb}/", "POST")
    db.execute("insert into receipts values(?,?,?,?,CURRENT_TIMESTAMP)", (version, category, ident, verb))
    db.commit()


def operate(db, method, expected):
    row = state(db)
    if row["version"] != expected:
        return 409, {"error": "The rehearsal moved in another tab. Reload to rejoin it."}
    pos, version = row["pos"], row["version"]
    if method == "advance" and pos + 1 < len(DRAFT["beats"]):
        db.execute("update playback set pos=pos+1, version=version+1, delivered=0")
    elif method == "restart" and pos == len(DRAFT["beats"]) - 1:
        if row["mood"]:
            send_cue(db, "moods", row["mood"], "stop", version)
        db.execute("update playback set pos=-1,version=version+1,delivered=1,mood=0")
    elif method in ("heard", "resume") and pos >= 0:
        scene = DRAFT["beats"][pos]
        mood = DRAFT["cues"][scene["location"]]["id"]
        restore = False
        if method == "resume":
            live = provider("online/frontend-api/state/")
            restore = not live.get("mood", {}).get(str(mood), {}).get("is_playing")
        if mood != row["mood"] or restore:
            send_cue(db, "moods", mood, "play", version)
            db.execute("update playback set mood=?", (mood,))
        # Claim an event durably before dispatch. A lost provider response must never replay a knock.
        if method == "heard":
            db.execute("update playback set delivered=1")
        db.commit()
        if method == "heard" and not row["delivered"] and scene["event"]:
            send_cue(db, "elements", DRAFT["cues"][scene["event"]]["id"], "play", version)
    else:
        return 409, {"error": "This action is unavailable at the current scene."}
    db.commit()
    return 200, public_state(state(db))


class Handler(BaseHTTPRequestHandler):
    def handle_one_request(self):
        # The game can hand off an in-flight plan to the browser. Its original
        # connection may then close before the shared plan is ready to deliver.
        try:
            super().handle_one_request()
        except (BrokenPipeError, ConnectionResetError):
            self.close_connection = True

    def reply(self, code, body, mime="application/json"):
        raw = json.dumps(body).encode() if mime == "application/json" else body
        self.send_response(code)
        for key, val in {
            "Content-Type": mime, "Content-Length": str(len(raw)), "Cache-Control": "no-store",
            "Referrer-Policy": "strict-origin-when-cross-origin", "X-Content-Type-Options": "nosniff",
            "Content-Security-Policy": "default-src 'self'; script-src 'self' https://syrinscape.com/static/dist/; style-src 'self'; connect-src 'self' https://syrinscape.com https://*.syrinscape.com wss://*.syrinscape.com; img-src 'self' data:; media-src 'self' blob: https://*.syrinscape.com; object-src 'none'; base-uri 'none'; frame-ancestors 'none'",
        }.items():
            self.send_header(key, val)
        self.end_headers()
        self.wfile.write(raw)

    def route(self):
        name = self.path.partition("?")[0].removeprefix("/sound-test/").lstrip("/")
        if self.command == "GET" and (name or "index.html") in PUBLIC:
            name = name or "index.html"
            return self.reply(200, (HERE / name).read_bytes(), PUBLIC[name])
        if name not in {"state", "advance", "heard", "resume", "restart","live/join","live/plan","live/resume","live/heard"}:
            return self.reply(404, {})
        cookies = http.cookies.SimpleCookie(self.headers.get("Cookie", ""))
        session = cookies.get("ttrpg_session")
        if not session:
            return self.reply(401, {})
        req = urllib.request.Request(GAME_BASE + "/api/auth/session",
                                     headers={"Cookie": "ttrpg_session=" + session.value})
        with urllib.request.urlopen(req, context=GAME_TLS, timeout=5) as resp:
            auth = json.load(resp)
        if not auth.get("authenticated") or not auth.get("csrf_token"):
            return self.reply(401, {})
        with contextlib.closing(sqlite3.connect(STATE / "pilot.db")) as db:
            if self.command == "GET" and name == "state":
                return self.reply(200, dict(public_state(state(db)), listenerSession=session_id()))
            if self.command != "POST" or name == "state":
                return self.reply(405, {})
            if self.headers.get("Origin") != ORIGIN or not hmac.compare_digest(
                    self.headers.get("X-CSRF-Token", ""), auth["csrf_token"]):
                return self.reply(403, {})
            size = int(self.headers.get("Content-Length", "0"))
            if not 0 < size <= 256:
                return self.reply(400, {})
            payload = json.loads(self.rfile.read(size))
            if name.startswith("live/"):
                def read_context():
                    url = GAME_BASE + "/api/context"
                    req = urllib.request.Request(url, headers={"Cookie": "ttrpg_session=" + session.value})
                    with urllib.request.urlopen(req, context=GAME_TLS, timeout=5) as response:
                        return json.load(response)
                try:
                    result = live_score.handle(db, name[5:], payload, read_context, provider, session_id, TOKEN)
                except PermissionError:
                    return self.reply(403, {})
                return self.reply(200, result)
            expected = payload["version"]
            if type(expected) is not int:
                return self.reply(400, {})
            with LOCK:
                code, result = operate(db, name, expected)
            self.reply(code, result)

    def handle_request(self):
        try:
            self.route()
        except (ValueError, KeyError):
            self.reply(400, {"error": "Invalid rehearsal request."})
        except (urllib.error.URLError, TimeoutError, sqlite3.Error) as exc:
            print("Upstream or state error:", type(exc).__name__, getattr(exc, "code", ""), flush=True)
            self.reply(502, {"error": "The service could not confirm the sound transition. Reload to reconnect."})

    do_GET = do_POST = handle_request

    def log_message(self, *args):
        pass


if __name__ == "__main__":
    with contextlib.closing(sqlite3.connect(STATE / "pilot.db")) as db:
        db.executescript(live_score.SCHEMA)
        db.executescript("""
          create table if not exists playback(pos integer, version integer, delivered integer, mood integer);
          insert into playback select -1,0,1,0 where not exists(select 1 from playback);
          create table if not exists receipts(version integer, category text, sound_id integer, verb text, at text);
        """)
    ThreadingHTTPServer(("127.0.0.1", int(os.environ.get("INK_SCORE_PORT", "7439"))), Handler).serve_forever()
