# Security and provider data flow

ink & bone is local-first, not inherently offline. The database and uploaded assets stay on the host, while model inputs go wherever the selected AI provider runs.

## Listener and authentication modes

The default listener is `127.0.0.1:7432`. Loopback mode has no login screen and is intended for one user on the same machine. Do not publish that listener through a proxy or tunnel unless the proxy provides an equivalent security boundary.

A non-loopback `-listen` address fails closed unless all of these are configured:

- `TTRPG_AUTH_SECRET` containing at least 32 bytes;
- a valid certificate and key supplied with `-tls-cert` and `-tls-key`;
- at least one exact HTTP(S) origin supplied with repeatable or comma-separated `-allowed-origin` flags.

Browser login exchanges the master secret for an opaque `HttpOnly`, `Secure`, `SameSite=Strict` cookie. Sessions expire after 30 minutes idle or 12 hours absolute. Cookie-authenticated state changes require the session CSRF token. WebSocket upgrades validate the request origin and remain subject to session revocation. Non-browser clients may use the master secret as a Bearer token over TLS.

Maps and portraits are available only through typed, database-backed asset routes. There is no general file server; `/api/files/...` is deliberately absent. This does not replace filesystem permissions: the SQLite file, adjacent backups, and uploaded assets contain private campaign data.

## What model providers receive

For GM narration and AI-powered automation, the request may include the current player action, non-whisper session history, character/campaign state, active objectives, relevant NPC/world notes, combat state, and selected rulebook excerpts. The exact subset depends on the endpoint and automation.

Whispers remain visible in the authenticated browser transcript, but AI-visible message queries exclude them in SQLite before prompt construction. Session exports exclude whispers as well.

- DeepSeek, Anthropic, and OpenRouter requests send prompt data to those cloud services under their respective policies.
- Hybrid mode keeps GM narration on Ollama but sends automation prompts to Anthropic.
- Ollama-only modes send model requests to the configured Ollama service. They remain machine-local only when that Ollama endpoint is itself local and not proxied elsewhere.

API keys and the authentication secret come from environment variables. They are not stored in the application database. Protect process environments, shell history, TLS keys, the database directory, and backups.

## Operational checklist

- Keep the default loopback listener unless remote access is required.
- Use a unique high-entropy master secret and trusted TLS certificate for remote access.
- Allow only the exact browser origins you operate.
- Review the selected provider before uploading copyrighted or sensitive rulebooks.
- Run `make verify`, `make verify-e2e`, and the secret scan before releases.
