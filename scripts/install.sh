#!/usr/bin/env bash
set -euo pipefail

echo "Installing gitdex..."
go install ./cmd/gitdex
echo "Done. Run 'gitdex' to start."
