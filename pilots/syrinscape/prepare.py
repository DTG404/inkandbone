"""Prepare a four-turn sound rehearsal using the homelab GM and the owner's catalog."""
import json
import os
import sys
import urllib.parse
import urllib.request
from pathlib import Path


def request(url, *, token=None, body=None, timeout=150):
    headers = {"Accept": "application/json"}
    if token:
        headers["Authorization"] = "token " + token
    data = None
    if body is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(body).encode()
    with urllib.request.urlopen(urllib.request.Request(url, data, headers), timeout=timeout) as response:
        return json.load(response)


def search(token, query, kind, model, timeout=15):
    args = urllib.parse.urlencode({"q": query, "kind": kind, "library": "Available to Play",
                                   "pp": 15})
    rows = request("https://syrinscape.com/search/?" + args, token=token, timeout=timeout)["results"]
    return [{"id": row["pk"], "title": row["title"], "genre": row.get("category", []),
             "sounds": row.get("related_loop_titles", [])[:5] + row.get("related_sfx_titles", [])[:5]}
            for row in rows if row["available_to_play"] and row["model_name"] == model]


def prepare(token):
    options = {
        "market": search(token, "market", "Moods", "Mood"),
        "knock": search(token, "knock", "Oneshots", "OneshotElement"),
        "machinery": search(token, "machinery", "Moods", "Mood"),
    }
    if not all(options.values()):
        raise ValueError("A rehearsal sound could not be matched in the unlocked catalog")
    actions = ["Enter the lower-hive market", "Listen at the sealed door",
               "Follow the service passage into the machine room", "Return to the market"]
    prompt = """You are the Ink & Bone GM directing a restrained Wrath & Glory sound rehearsal.
Write four short second-person scenes following the given player actions. No spoilers or offscreen facts.
Beat one: market ambience is presently audible. Beat two: keep market ambience and hear one physical knock
from inside a sealed door. Beat three: leave market and hear machines. Beat four: return to the same
recognizable market. Select one best-fitting catalog ID per requested sound, preferring sci-fi and avoiding
magic, spells, combat, music stingers or unintended alarms. Write a short setup then a separate exact audible
perception sentence, then one short aftermath sentence, for each beat. No labels, no invented player decisions.
Return only JSON: {"market":integer,"knock":integer,"machinery":integer,"beats":[{"setup":"...",
"perception":"...","after":"..."}, ...]}. The distinct knock must first occur in the perception sentence."""
    response = request(os.environ["OLLAMA_HOST"].rstrip("/") + "/v1/chat/completions", body={
        "model": os.environ["OLLAMA_MODEL"], "temperature": 0.35, "max_tokens": 3500,
        "response_format": {"type": "json_object"}, "messages": [
            {"role": "system", "content": prompt},
            {"role": "user", "content": json.dumps({"actions": actions, "choices": options})}]})
    data = json.loads(response["choices"][0]["message"]["content"])
    if len(data["beats"]) != len(actions):
        raise ValueError("GM supplied the wrong number of scenes")
    cues = {}
    for key, choices in options.items():
        entry = next(row for row in choices if row["id"] == data[key])
        cues[key] = {"id": entry["id"], "title": entry["title"]}
    beats = []
    for action, scene, location, event in zip(actions, data["beats"],
            ["market", "market", "machinery", "market"], [None, "knock", None, None], strict=True):
        lines = [scene[k].strip() for k in ("setup", "perception", "after")]
        if not all(lines):
            raise ValueError("Empty narration")
        beats.append(dict(action=action, lines=lines, location=location, event=event))
    return dict(cues=cues, beats=beats)


if __name__ == "__main__":
    draft = prepare(sys.stdin.readline().strip())
    Path(sys.argv[1]).write_text(json.dumps(draft, indent=2) + "\n")
    print("GM selected:", ", ".join(v["title"] for v in draft["cues"].values()))
