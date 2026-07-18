.PHONY: dev build install test e2e clean lint audit secrets-scan check verify verify-e2e generate

GO_VERSION ?= 1.26.5
NODE_VERSION ?= 24.18.0
GOLANGCI_LINT_VERSION ?= v2.12.2
GOVULNCHECK_VERSION ?= v1.3.0
GITLEAKS_VERSION ?= v8.30.1
GO_PACKAGES := ./cmd/... ./internal/... ./web

generate:
	go generate ./internal/api

# Run Go server (air hot reload) + Vite dev server concurrently
dev:
	cd web && npm run dev &
	air

# Install npm dependencies (idempotent — uses lockfile)
web/node_modules:
	cd web && npm ci

# Build production binary (React first, then Go with embedded assets)
build: web/node_modules
	cd web && npm run build
	go build -o ttrpg ./cmd/ttrpg

# Install binary to ~/bin (preserves the ttrpg wrapper script)
install: build
	mkdir -p ~/bin
	cp ttrpg ~/bin/ttrpg-bin
	@echo "Installed to ~/bin/ttrpg-bin"

# Run all Go tests and web tests
test:
	go test $(GO_PACKAGES) -v
	cd web && npm test -- --run

# Build the current binary, then run Playwright smoke, security, and reliability tests.
e2e: build
	cd e2e && npm test

# Reproduce the checks enforced by CI from a clean checkout.
verify:
	cd web && npm ci && npm run build
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*' -not -path './.worktrees/*' -not -path '*/node_modules/*'))" || { gofmt -l $$(find . -name '*.go' -not -path './.git/*' -not -path './.worktrees/*' -not -path '*/node_modules/*'); exit 1; }
	./scripts/verify-generated.sh
	golangci-lint run $(GO_PACKAGES)
	go vet $(GO_PACKAGES)
	go test $(GO_PACKAGES)
	govulncheck $(GO_PACKAGES)
	cd web && npm test -- --run
	cd web && npm run lint
	cd web && npm audit --audit-level=high
	@binary=$$(mktemp /tmp/inkandbone-verify.XXXXXX); trap 'rm -f "$$binary"' EXIT; go build -o "$$binary" ./cmd/ttrpg
	@if command -v gitleaks >/dev/null 2>&1; then gitleaks detect --source . --no-git -v; else echo "verify: gitleaks not installed, skipping"; fi

# Build and exercise the maintained browser suite with disposable runtime state.
verify-e2e:
	@set -eu; \
	binary=$$(mktemp /tmp/inkandbone-e2e.XXXXXX); \
	port=$$(node -e 'const n=require("node:net").createServer();n.listen(0,"127.0.0.1",()=>{console.log(n.address().port);n.close()})'); \
	child=''; \
	cleanup() { test -z "$$child" || { kill "$$child" 2>/dev/null || true; wait "$$child" 2>/dev/null || true; }; rm -f "$$binary"; }; \
	trap cleanup EXIT INT TERM; \
	cd web && npm ci && npm run build; \
	cd ..; \
	go build -o "$$binary" ./cmd/ttrpg; \
	cd e2e && npm ci; \
	INKANDBONE_E2E_BINARY="$$binary" INKANDBONE_E2E_PORT="$$port" npm test & child=$$!; \
	wait "$$child"; \
	child=''

# Lint Go with golangci-lint and web with ESLint
lint: web/node_modules
	golangci-lint run $(GO_PACKAGES)
	cd web && npm run lint

# Dependency vulnerability audit
audit: web/node_modules
	govulncheck $(GO_PACKAGES)
	cd web && npm audit --audit-level=high

# Secrets scan (gitleaks)
secrets-scan:
	command -v gitleaks >/dev/null 2>&1 && gitleaks detect --source . --no-git -v || echo "secrets-scan: gitleaks not installed, skipping"

clean:
	rm -rf ttrpg tmp/ web/dist/

# Run everything: build (web→Go), lint, audit, test, secrets scan
check: build lint audit test secrets-scan
	@echo "=== All checks passed ==="
