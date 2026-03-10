#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-dev}"
OUT_DIR="${2:-dist}"

echo "Building gitdex ${VERSION} into ${OUT_DIR}..."
rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"

build_binary() {
  local goos="$1"
  local goarch="$2"
  local output_name="$3"

  echo "  -> ${output_name}"
  CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o "${OUT_DIR}/${output_name}" ./cmd/gitdex
}

build_binary windows amd64 gitdex-windows-amd64.exe
build_binary windows arm64 gitdex-windows-arm64.exe
build_binary linux amd64 gitdex-linux-amd64
build_binary linux arm64 gitdex-linux-arm64
build_binary darwin amd64 gitdex-macos-amd64
build_binary darwin arm64 gitdex-macos-arm64

git archive --format=zip --output "${OUT_DIR}/gitdex-source.zip" HEAD
(
  cd "${OUT_DIR}"
  sha256sum gitdex-* > checksums.txt
)

"$(dirname "$0")/render-release-notes.sh" "${VERSION}" > "${OUT_DIR}/release-notes.md"

echo "Artifacts written to ${OUT_DIR}"
