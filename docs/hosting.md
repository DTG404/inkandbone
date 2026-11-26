# Host a private table

Run the game on an always-on Linux host to close your laptop and return to the same campaign later. The game and SQLite data live on that host. The GM can run on a separate machine. [AI provider setup](providers.md)

## Serve the game over private HTTPS

This systemd example binds directly to a private interface. Substitute the host’s actual private IP, its certificate’s DNS name and the port players can reach. Restrict that port to your players through the private network or tailnet grant. The default loopback launch is intended for one local user. [Listener and authentication modes](security.md#listener-and-authentication-modes)

1. Build the executable using the [local setup](../README.md#run-a-local-table) and install it at `/usr/local/bin/inkandbone`. Create a dedicated service account and group named `inkandbone`.
2. Put a trusted TLS certificate and its key under `/etc/inkandbone/`. Give the service account read access to both and restrict access to the key.
3. Create a root-owned, mode `0600` file at `/etc/inkandbone/game.env`. Set `TTRPG_AUTH_SECRET` to a unique random secret of at least 32 bytes and add the variables for your GM provider. Use systemd environment-file syntax (`NAME=value`). Share the login secret with players through your private credential channel.
4. Save the following as `/etc/systemd/system/inkandbone.service`. Replace the example IP address and origin before starting it.

```ini
[Unit]
Description=Ink & Bone private game table
Wants=network-online.target
After=network-online.target

[Service]
User=inkandbone
Group=inkandbone
StateDirectory=inkandbone
StateDirectoryMode=0700
WorkingDirectory=/var/lib/inkandbone
UMask=0077
EnvironmentFile=/etc/inkandbone/game.env
ExecStart=/usr/local/bin/inkandbone -db /var/lib/inkandbone/ttrpg.db -listen 192.0.2.10:7432 -allowed-origin https://table.example:7432 -tls-cert /etc/inkandbone/server.crt -tls-key /etc/inkandbone/server.key
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now inkandbone.service
```

From a player’s device, open your HTTPS origin, unlock the table and create a disposable session. Submit an action and wait for a GM reply. On a new installation, restart the unit after the turn and reopen that same session. Confirm that the reply and sheet load. On an existing table, arrange a break in play before restarting.

If the browser loads but the GM is silent, check the provider selected at startup and whether the game host can reach it. If the browser cannot connect, check the private grant, listener, certificate name and allowed origin against the URL on the player’s device. Back up the database and adjacent assets together. [Backup and recovery](data-recovery.md)

## Optional services

- [Syrinscape playback](../pilots/syrinscape/README.md) follows the active table’s narrated sounds. The service needs its own subscription token and a same-origin route for playback.
- [Subscription map worker](../deploy/map-worker/README.md) runs Codex under a separate signed-in account. It receives the location, ruleset name and public scene text. The GM provider can stay on the homelab.

Adapt the integration guides’ unit names, service accounts and backend addresses to your installation.
