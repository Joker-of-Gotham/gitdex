#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: ./scripts/v3-cutover-preflight.sh [--skip-network]

Options:
  --skip-network   Skip `git fetch --all --prune` check.
EOF
}

skip_network=false
for arg in "$@"; do
  case "$arg" in
    --skip-network)
      skip_network=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      usage
      exit 1
      ;;
  esac
done

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

echo "== GitDex V3 Cutover Preflight =="

if [[ -n "$(git status --porcelain)" ]]; then
  echo "Working tree is not clean. Commit or stash changes before cutover." >&2
  exit 1
fi

if [[ "$skip_network" == false ]]; then
  echo "[network] git fetch --all --prune"
  git fetch --all --prune
fi

echo "[1/6] go vet ./..."
go vet ./...

echo "[2/6] go test ./... -count=1"
go test ./... -count=1

echo "[3/6] go test regression packages"
go test ./internal/executor ./internal/llmfactory -run Regression -count=1

echo "[4/6] go build ./..."
go build ./...

echo "[5/6] config diagnostics smoke"
go run ./cmd/gitdex config lint >/dev/null
go run ./cmd/gitdex config explain >/dev/null
go run ./cmd/gitdex config source >/dev/null

echo "[6/6] preflight checks passed"
