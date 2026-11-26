# Subscription map worker

Ink & Bone can commission a tactical illustration and named map zones through the signed-in Codex CLI on a private game server. The Go service sends the location, ruleset name, and public scene text through a Unix socket. The worker runs the built-in image generation tool in a new, read-only Codex session. The game stores the returned image under its data directory and saves the zones with the map in SQLite. The GM and location detector continue using the game's configured narrative and automation providers.

## VPS setup

1. Create a dedicated system account named `inkandbone-map` with a private home at `/var/lib/inkandbone-map`. Install the Codex 0.172.0 Linux CLI and its matching code-mode host under `/opt/inkandbone/map-tools/0.172.0/`. The service unit pins this path.
2. Authenticate Codex with ChatGPT as that account and verify with `codex login status` using that binary and home. Only the worker account should be able to read or update its login.
3. Install `worker.py` and `zones.json` under `/opt/inkandbone/map-worker/` as root-owned, read-only application files. Install both supplied systemd units. The socket permits the game group `inkandbone` to request illustrations.
4. Run `systemctl daemon-reload` and `systemctl enable --now inkandbone-map.socket`. Add `Environment=INK_MAP_WORKER_SOCKET=/run/inkandbone-map.sock` to the game service and restart the game at a turn boundary.
5. Open a played session, click **Generate Map**, open **Zones**, reveal a named region, and reload the session. Verify the image and reveal state persist. An automatic map is triggered when the detector identifies a new named location and the Generate Maps automation is enabled.

The account's subscription limits and login expiry apply. Inspect the instance journal with `journalctl -u 'inkandbone-map@*'` if a map fails. A configured worker failure is reported to the game. Installations without a configured socket retain their original SVG generator.
