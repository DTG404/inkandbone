# Syrinscape playback

The normal game uses its active campaign, session, ruleset and public GM narration to score perceived sounds. Location moods are saved per campaign. The game has one active table and the subscription supplies one shared playback session. Switching tables releases the previous table's mood. The optional rehearsal at `/sound-test/` controls that same subscription.

## Hosted setup

1. Install the files from this directory in `/opt/inkandbone/sound-test/`. The sound service uses Python 3.11 or newer and its standard library. Install `inkandbone-sound-test.service` as a systemd unit and `game-live-score.conf` as a drop-in for the game service.
2. Set `INK_ORIGIN` to the game's private HTTPS origin in `/etc/inkandbone/sound.env`. Set `OLLAMA_HOST` and `OLLAMA_MODEL` there to the configured homelab OpenAI-compatible endpoint and model. Provision the subscription token as the root-only file `/etc/inkandbone/syrinscape-token`. The unit passes it through systemd credentials.
3. Route `/sound-test/` and its subpaths on the game origin to the sound service at `127.0.0.1:7439`. The service validates game sessions through `https://127.0.0.1:7432` using `/etc/inkandbone/backend.crt`. Override `INK_GAME_BASE` and `INK_GAME_CA` if the private backend uses different settings.
4. Reload systemd, start and enable the sound service, then restart the game. Both units use `INK_SCORE_CAMPAIGN_ID=all`. Setting it to a single numeric ID on both limits playback to that campaign.
5. Back up the game database and the sound service's `pilot.db` using SQLite's online backup facility. The sound database holds location assignments and event receipts.

Run `python3 -m unittest test_live_score` in this directory and `make verify` and `make verify-e2e` from the repository root.

For a live browser check, install the dependencies in `e2e` with `npm ci`. Run `node e2e/live-score.mjs fixtures.json` with `INKANDBONE_E2E_BASE_URL` pointing to a disposable deployment and its login in `TTRPG_AUTH_SECRET`. Supply a JSON array of existing audible test sessions, for example `[{"cid":2,"sid":3,"rule":"dnd5e","title":"Forest"}]`. An optional `action` submits a real player turn and checks for a GM reply and a dispatched sound cue. The walkthrough changes the active table and shares the configured subscription's playback.
