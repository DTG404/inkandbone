# Data backup and recovery

The default database is `~/.ttrpg/ttrpg.db`; `-db PATH` selects another file. Uploaded assets live beside the selected database. Stop ink & bone before copying or replacing live data.

## Automatic repair backup

Before a pending repair migration touches a file-backed database, startup creates a consistent SQLite copy with `VACUUM INTO`, validates it with `PRAGMA integrity_check`, sets mode `0600`, and publishes it beside the database as:

```text
ttrpg.db.backup-<unique suffix>
```

If backup creation or validation fails, the repair does not run. Startup errors after a validated backup include its path. Ordinary non-repair migrations do not create this backup.

Repair migration `056_integrity_repair.sql` moves recoverable rows with invalid ownership into `orphaned_records` before removing broken live references. Each row records the source table, source ID, JSON payload, and reason. Treat this table as recovery evidence; do not blindly insert its payloads into production tables.

## Validate a copied database

Never experiment on the live file. With the server stopped:

```bash
cp ~/.ttrpg/ttrpg.db.backup-UNIQUE /tmp/inkandbone-recovery.db
chmod 600 /tmp/inkandbone-recovery.db
sqlite3 /tmp/inkandbone-recovery.db 'PRAGMA integrity_check; PRAGMA foreign_key_check;'
```

`integrity_check` must print exactly `ok`. `foreign_key_check` must print no rows. Inspect quarantined records if present:

```bash
sqlite3 -header -column /tmp/inkandbone-recovery.db \
  'SELECT source_table, source_id, reason FROM orphaned_records ORDER BY id;'
```

Then start the application against the copy on loopback and inspect campaigns, sessions, characters, assets, and recent messages:

```bash
ttrpg -db /tmp/inkandbone-recovery.db -listen 127.0.0.1:7432
```

## Restore

1. Stop every ink & bone process using the live database.
2. Preserve the current database and adjacent assets as a separately named incident copy.
3. Validate and exercise the candidate recovery copy as above.
4. Copy the validated candidate to a new file in the live data directory and set mode `0600`.
5. Point `-db` at that new file for a final verification run.
6. Only after verification, update the launcher to use the recovered path. Keep the original and automatic backup until the recovery is accepted.

Do not expose a database or backup through HTTP. Typed asset routes intentionally cannot download it.
