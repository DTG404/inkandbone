# Development and verification

This file is the canonical supported-toolchain and verification reference. Version values come from `go.mod`, `.nvmrc`, package manifests, and the pinned Makefile variables.

## Supported toolchain

| Tool | Version |
|---|---|
| Go | 1.26.5 |
| Node.js | 24.18.0 |
| React / React DOM | 19.2.x |
| TypeScript | 6.0.x |
| Vite | 8.1.x |
| golangci-lint | v2.12.2 |
| govulncheck | v1.3.0 |
| gitleaks | v8.30.1 |

The repository currently embeds 57 ordered SQL migration files and seeds 14 supported game systems. Add new migrations; never rewrite an applied migration.

## Commands

```bash
make dev          # Go/Vite development servers
make build        # production frontend plus embedded Go binary
make test         # Go and frontend unit suites
make verify       # generated drift, format, lint, vet, tests, audits, build, secrets
make verify-e2e   # disposable binary/database plus maintained Playwright suite
make generate     # regenerate realtime Go/TypeScript contracts
```

Use `npm ci`, not `npm install`, for reproducible frontend and E2E installs. The maintained browser suite lives only under `e2e/`; its coverage inventory is in [testing/e2e-coverage-matrix.md](testing/e2e-coverage-matrix.md).

CI runs `make verify` and `make verify-e2e` as separate required jobs. A local release candidate should pass both commands from a clean dependency install.

## Change checklist

- Run `gofmt` on changed Go files and ESLint on changed TypeScript.
- Add tests before changing behavior; preserve route, realtime, and accessibility contracts.
- Run `make generate` after editing `contracts/realtime.json`, then verify no generated drift.
- Add a new numbered migration for schema/data changes.
- Update this file when the supported toolchain or canonical verification command changes.
