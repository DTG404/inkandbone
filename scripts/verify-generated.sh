#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary="$(mktemp -d /tmp/inkandbone-generated.XXXXXX)"
cleanup() { rm -rf "$temporary"; }
trap cleanup EXIT INT TERM

cp "$root/internal/api/realtime_gen.go" "$temporary/realtime_gen.go"
cp "$root/web/src/realtime.gen.ts" "$temporary/realtime.gen.ts"

cd "$root"
make generate

diff -u "$temporary/realtime_gen.go" "$root/internal/api/realtime_gen.go"
diff -u "$temporary/realtime.gen.ts" "$root/web/src/realtime.gen.ts"
