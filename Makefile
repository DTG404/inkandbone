.PHONY: dev build install test clean lint audit secrets-scan check

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
	go test ./... -v
	cd web && npm test -- --run

# Lint Go with golangci-lint and web with ESLint
lint: web/node_modules
	golangci-lint run ./...
	cd web && npm run lint

# Dependency vulnerability audit
audit: web/node_modules
	govulncheck ./...
	cd web && npm audit --audit-level=high

# Secrets scan (gitleaks)
secrets-scan:
	command -v gitleaks >/dev/null 2>&1 && gitleaks detect --source . --no-git -v || echo "secrets-scan: gitleaks not installed, skipping"

clean:
	rm -rf ttrpg tmp/ web/dist/

# Run everything: build (web→Go), lint, audit, test, secrets scan
check: build lint audit test secrets-scan
	@echo "=== All checks passed ==="
