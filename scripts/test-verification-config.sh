#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
makefile="$root/Makefile"
workflow="$root/.github/workflows/ci.yml"

require() {
  local pattern=$1
  local file=$2
  if ! grep -Eq "$pattern" "$file"; then
    echo "missing pattern '$pattern' in ${file#"$root/"}" >&2
    exit 1
  fi
}

require '^verify:' "$makefile"
require '^verify-e2e:' "$makefile"
require '^GO_VERSION[[:space:]]*\?=[[:space:]]*1\.26\.5$' "$makefile"
require '^NODE_VERSION[[:space:]]*\?=[[:space:]]*24\.18\.0$' "$makefile"
require '^GOLANGCI_LINT_VERSION[[:space:]]*\?=[[:space:]]*v2\.12\.2$' "$makefile"
require '^GOVULNCHECK_VERSION[[:space:]]*\?=' "$makefile"
require '^GITLEAKS_VERSION[[:space:]]*\?=[[:space:]]*v8\.30\.1$' "$makefile"
require '^GO_PACKAGES[[:space:]]*:=[[:space:]]*\.\/cmd/\.\.\.[[:space:]]+\.\/internal/\.\.\.[[:space:]]+\.\/web$' "$makefile"
require 'INKANDBONE_E2E_BINARY=.*npm test' "$makefile"
require 'cd web && npm ci && npm run build' "$makefile"
require '^[[:space:]]+- web/node_modules$' "$root/.golangci.yml"
require '^[[:space:]]+- e2e/node_modules$' "$root/.golangci.yml"
require '^permissions:$' "$workflow"
require '^[[:space:]]+contents:[[:space:]]+read$' "$workflow"
require 'go-version:[[:space:]]*"?1\.26\.5"?' "$workflow"
require 'node-version:[[:space:]]*"?24\.18\.0"?' "$workflow"
require 'version:[[:space:]]*v2\.12\.2' "$workflow"
require 'run:[[:space:]]+make verify$' "$workflow"
require 'run:[[:space:]]+make verify-e2e$' "$workflow"
require 'gitleaks/v8@v8\.30\.1' "$workflow"

if [[ $(grep -c 'web/package-lock\.json' "$workflow") -lt 2 || $(grep -c 'e2e/package-lock\.json' "$workflow") -lt 2 ]]; then
  echo "both CI jobs must cache the web and e2e npm lockfiles" >&2
  exit 1
fi

if grep -Eq '\|\|[[:space:]]*echo|pull-requests:[[:space:]]+write|@latest|npm install([^[:alnum:]]|$)' "$workflow"; then
  echo "workflow contains an unpinned, bypassed, or over-privileged command" >&2
  exit 1
fi

first_verify_command=$(awk '/^verify:/{getline; sub(/^[[:space:]]+/, ""); print; exit}' "$makefile")
if [[ "$first_verify_command" != 'cd web && npm ci && npm run build' ]]; then
  echo "verify must build embedded frontend assets first on a clean checkout" >&2
  exit 1
fi

require 'golangci-lint run \$\(GO_PACKAGES\)' "$makefile"
require 'go vet \$\(GO_PACKAGES\)' "$makefile"
require 'go test \$\(GO_PACKAGES\)' "$makefile"
require 'govulncheck \$\(GO_PACKAGES\)' "$makefile"
