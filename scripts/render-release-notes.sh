#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-v1.0.0}"

cat <<EOF
# gitdex ${VERSION}

\`gitdex\` is an AI-native Git TUI for local repositories. It keeps repository state, context budget, memory, raw model output, reasoning, and execution history visible in one inspectable workflow surface.

## Highlights

- First-run language selection and runtime language switching with \`L\`
- Inspectable \`Workflow\`, \`Timeline\`, \`Context\`, \`Memory\`, \`Raw\`, \`Result\`, and \`Thinking\` panels
- Clear separation between view-only advisories and executable Git actions
- Persistent repository memory and structured context budgeting
- Local-first Ollama model selection with optional verifier support

## Included Assets

- \`gitdex-windows-amd64.exe\`
- \`gitdex-windows-arm64.exe\`
- \`gitdex-linux-amd64\`
- \`gitdex-linux-arm64\`
- \`gitdex-macos-amd64\`
- \`gitdex-macos-arm64\`
- \`gitdex-source.zip\`
- \`checksums.txt\`

## Quick Start

1. Install Git, Go, and Ollama
2. Pull a local model such as \`qwen2.5:3b\`
3. Start \`gitdex\` inside a real Git repository
4. Choose a language, pick a model, and inspect the first analysis round

## Notes

- \`gitdex\` is local-first and expects Ollama for AI-backed suggestions
- Legacy config compatibility for \`.gitmanualrc\` and \`GITMANUAL_*\` is still available
- README, release assets, and GitHub workflows are aligned for the public \`${VERSION}\` release
EOF
