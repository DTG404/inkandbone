import hashlib
import json
import sqlite3
import unittest
from unittest.mock import patch

import live_score as score


class LiveScoreTest(unittest.TestCase):
    def setUp(self):
        self.db = sqlite3.connect(":memory:")
        self.addCleanup(self.db.close)
        self.db.executescript(score.SCHEMA)
        self.ctx = {
            "campaign": {"id": 6},
            "session": {"id": 4, "campaign_id": 6},
            "recent_messages": [],
        }
        self.addCleanup(patch.stopall)
        patch.object(score, "SCOPE", "all").start()
        self.playing = 0
        self.calls = []

    def provider(self, path, verb="GET"):
        if verb == "GET":
            return {"mood": {str(self.playing): {"is_playing": True}}}
        self.calls.append(path)
        if "/moods/" in path:
            self.playing = int(path.split("/")[-3]) if path.endswith("/play/") else 0

    def turn(self, mid, text):
        self.ctx["recent_messages"].append(
            {"id": mid, "role": "assistant", "content": text}
        )
        return hashlib.sha256(text.encode()).hexdigest()

    def run_op(self, name, **args):
        return score.handle(
            self.db,
            name,
            {"session": 4, **args},
            lambda: self.ctx,
            self.provider,
            lambda: "listener",
            "unused",
        )

    def test_claim_once_restore_only_ambience_and_reject_stale_turn(self):
        text = "The fans hum. A bell rings."
        cues = [
            dict(kind="mood", place="hall", anchor="fans hum", sound=12, title="Fan"),
            dict(
                kind="event", place="hall", anchor="bell rings", sound=45, title="Bell"
            ),
        ]
        digest = self.turn(1, text)
        with patch.object(score, "direct", return_value=cues):
            plan = self.run_op("plan", digest=digest)
            self.assertEqual(
                [i["anchor"] for i in plan["cues"]], ["fans hum", "bell rings"]
            )
            self.assertEqual(self.calls, [])
            for slot in (0, 1, 1):
                self.run_op("heard", message=1, slot=slot)
            self.playing = 0
            self.run_op("resume")
            self.turn(2, "No perceptible change.")
            self.run_op("heard", message=1, slot=1)
        self.assertEqual(
            self.calls,
            [
                "online/frontend-api/moods/12/play/",
                "online/frontend-api/elements/45/play/",
                "online/frontend-api/moods/12/play/",
            ],
        )

    def test_brief_hush_is_not_durable_silence(self):
        text = "The scratching stops. For a heartbeat it is quiet. The fan hums."
        quiet = dict(
            kind="quiet",
            place="alley",
            anchor="The scratching stops.",
            silentThroughEnd=False,
            endEvidence="The fan hums.",
        )
        with patch.object(score, "model", return_value={"cues": [quiet]}):
            self.assertEqual(
                score.direct(self.db, 6, 4, text, "Zay", "unused", "dnd5e"), []
            )

    def test_sustained_silence_survives_reconnect(self):
        text = "You seal the airlock. You hear only silence inside."
        cue = dict(
            kind="quiet",
            place="airlock",
            anchor="You seal the airlock.",
            silentThroughEnd=True,
            endEvidence="You hear only silence inside.",
        )
        self.db.execute("insert into score_room values(6,4,'outside',12)")
        self.playing = 12
        digest = self.turn(5, text)
        with patch.object(score, "model", return_value={"cues": [cue]}):
            self.run_op("plan", digest=digest)
            self.run_op("heard", message=5, slot=0)
        with patch.object(score, "model", side_effect=AssertionError("already scored")):
            self.run_op("resume")
        self.assertEqual(self.calls, ["online/frontend-api/moods/12/stop/"])
        self.assertEqual(self.playing, 0)

    def test_reopen_rechecks_old_zero_mood_and_never_replays_an_event(self):
        text = "Silence falls for a heartbeat. A fan whirs and a drop splashes."
        self.turn(18, text)
        old = dict(
            kind="quiet", place="hive", anchor="Silence falls", sound=0, title=""
        )
        self.db.execute("insert into score_place values(6,'hive',12,'Hive')")
        self.db.execute("insert into score_room values(6,4,'hive',0)")
        self.db.execute("insert into score_turn values(6,4,18,?)", (json.dumps([old]),))
        self.db.execute("insert into score_checkpoint values(6,4,18)")
        mood = dict(kind="mood", place="hive", query="fan", anchor="A fan whirs")
        with patch.object(score, "model", return_value={"cues": [mood]}) as model:
            self.run_op("resume")
            self.run_op("resume")
            self.assertEqual(model.call_count, 1)
        self.assertEqual(self.calls, ["online/frontend-api/moods/12/play/"])

    def test_remembered_place_uses_its_original_mood(self):
        self.db.execute("insert into score_place values(6,'market',72,'Market')")
        direction = {
            "kind": "mood",
            "place": "Market",
            "query": "crowd",
            "anchor": "fans hum",
        }
        with (
            patch.object(score, "model", return_value={"cues": [direction]}),
            patch.object(
                score, "search", side_effect=AssertionError("unexpected new search")
            ),
        ):
            cues = score.direct(
                self.db, 6, 4, "The same fans hum.", "Zay", "unused", "dnd5e"
            )
        self.assertEqual([c["sound"] for c in cues], [72])

    def test_sparse_phrase_includes_plain_playable_sounds_and_deduplicates(self):
        def search(_, q, *args, **kwargs):
            if q == "bell chime":
                return [{"id": 1, "title": "Cathedral warning chime"}]
            return [
                {"id": 1, "title": "Cathedral warning chime"},
                {"id": 2, "title": "Copper bell"},
            ]

        with patch.object(score, "search", side_effect=search):
            choices = score.candidates(
                "unused", "bell chime", "Oneshots", "OneshotElement"
            )
        self.assertEqual([row["id"] for row in choices], [1, 2])

    def test_busy_ranker_preserves_remembered_mood_and_unambiguous_event(self):
        self.db.execute("insert into score_place values(6,'market',72,'Market')")
        intent = {
            "cues": [
                dict(
                    kind="mood",
                    place="market",
                    anchor="The market hums",
                    query="crowds",
                ),
                dict(kind="event", place="market", anchor="A bell rings", query="bell"),
            ]
        }
        with (
            patch.object(score, "model", side_effect=[intent, TimeoutError()]),
            patch.object(
                score,
                "search",
                return_value=[
                    {"id": 45, "title": "Bell"},
                    {"id": 46, "title": "Magic bells"},
                ],
            ),
        ):
            cues = score.direct(
                self.db,
                6,
                4,
                "The market hums. A bell rings.",
                "Zay",
                "unused",
                "dnd5e",
            )
        self.assertEqual([c["sound"] for c in cues], [72, 45])

    def test_empty_game_transcript_can_be_null(self):
        self.ctx["recent_messages"] = None
        self.assertEqual(self.run_op("resume"), {"cues": []})

    def test_quote_preserves_rendered_text_and_requires_unique_evidence(self):
        self.assertEqual(
            score.narration_quote("The vendors CALL aloud.", "vendors call"),
            "vendors CALL",
        )
        self.assertEqual(
            score.narration_quote("Rain falls. Rain falls.", "rain falls"), ""
        )
        self.assertEqual(
            score.narration_quote("The doors stay closed.", "a bell rings"), ""
        )

    def test_reopen_with_no_saved_room_retries_ambience_without_events(self):
        self.turn(20, "You hear the surf.")
        self.db.execute("insert into score_checkpoint values(6,4,20)")
        self.db.execute("insert into score_turn values(6,4,20,'[]')")
        cue = dict(
            kind="mood", anchor="hear the surf", place="shore", sound=12, title="Surf"
        )
        with patch.object(score, "direct", return_value=[cue]) as director:
            self.run_op("resume")
        self.assertTrue(director.call_args.args[-1])
        self.assertEqual(self.calls, ["online/frontend-api/moods/12/play/"])

    def test_switch_to_empty_campaign_stops_sound_and_preserves_the_first_room(self):
        self.db.execute("insert into score_room values(6,4,'harbor',12)")
        self.db.execute("insert into score_place values(6,'harbor',12,'Surf')")
        self.playing = 12
        self.run_op("join")
        self.ctx.update(
            campaign={"id": 7},
            session={"id": 5, "campaign_id": 7},
            score_ruleset="dune",
        )
        self.run_op("join", session=5)
        self.run_op("resume", session=5)
        self.assertEqual(self.playing, 0)
        self.db.execute("insert into score_place values(7,'harbor',47,'Spice port')")
        text = "The harbor engines roar."
        digest = self.turn(20, text)
        cue = dict(kind="mood", place="harbor", anchor="engines roar", query="engines")
        with patch.object(score, "model", return_value={"cues": [cue]}) as model:
            self.run_op("plan", session=5, digest=digest)
        self.assertEqual(model.call_args.args[1]["ruleset"], "dune")
        self.run_op("heard", session=5, message=20, slot=0)
        self.assertEqual(self.playing, 47)
        self.ctx.update(
            campaign={"id": 6}, session={"id": 4, "campaign_id": 6}, recent_messages=[]
        )
        self.turn(10, "You hear the surf.")
        self.db.execute("insert into score_checkpoint values(6,4,10)")
        self.run_op("resume")
        self.assertEqual(self.playing, 12)
        with self.assertRaises(PermissionError):
            self.run_op("heard", session=5, message=20, slot=0)

    def test_other_campaign_and_unseen_anchor_cannot_play(self):
        self.ctx["campaign"]["id"] = 7
        with self.assertRaises(PermissionError):
            self.run_op("join")
        with patch.object(
            score,
            "model",
            return_value={
                "cues": [
                    {
                        "kind": "event",
                        "place": "hive",
                        "query": "knock",
                        "anchor": "A knock rings out",
                    }
                ]
            },
        ):
            self.assertEqual(
                score.direct(self.db, 6, 4, "All is quiet.", "Zay", "unused", "dnd5e"),
                [],
            )
