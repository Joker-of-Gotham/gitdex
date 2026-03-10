#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: ./scripts/set-module-path.sh github.com/<user-or-org>/gitdex" >&2
  exit 1
fi

placeholder='github.com/Joker-of-Gotham/gitdex'
module_path="$1"

if [[ ! "$module_path" =~ ^github\.com/[^/]+/[^/]+$ ]]; then
  echo "Expected a full module path such as github.com/<user-or-org>/gitdex" >&2
  exit 1
fi

mapfile -t files < <(rg -l --fixed-strings "$placeholder" go.mod README.md cmd internal test docs .github scripts 2>/dev/null || true)

if [[ ${#files[@]} -eq 0 ]]; then
  echo "No placeholder module path found."
  exit 0
fi

for file in "${files[@]}"; do
  perl -0pi -e "s|\Q$placeholder\E|$module_path|g" "$file"
  printf 'Updated %s\n' "$file"
done

gofmt -w cmd internal test

echo
echo "Replaced $placeholder -> $module_path"
echo "Next: go test ./..."
