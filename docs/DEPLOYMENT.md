# Deployment And Release Design

This document describes how `gitdex` is shipped, verified, and presented on GitHub.

## Release model

`gitdex` is a CLI/TUI product. The primary deployment target is GitHub Releases, not a server runtime.

Each `v*` tag triggers a release pipeline that publishes:

- `gitdex-windows-amd64.exe`
- `gitdex-windows-arm64.exe`
- `gitdex-linux-amd64`
- `gitdex-linux-arm64`
- `gitdex-macos-amd64`
- `gitdex-macos-arm64`
- `gitdex-source.zip`
- `checksums.txt`

## GitHub Actions layout

### CI

File: `.github/workflows/ci.yml`

Checks:

- `go vet ./...`
- `go test ./...`
- `go build ./...`
- `golangci-lint`
- `go test -race ./...`

### Release

File: `.github/workflows/release.yml`

Flow:

1. trigger on `v*` tag push
2. run `go vet`
3. run `go test -race ./...`
4. run `./scripts/build.sh <tag> dist`
5. publish all release assets to GitHub Releases

### CodeQL

File: `.github/workflows/codeql.yml`

Purpose:

- baseline static security scanning on pushes, pull requests, and a weekly schedule

## Local build helper

`scripts/build.sh` is the single release-asset builder. It creates the same filenames that GitHub Releases will expose and also writes:

- `dist/release-notes.md`
- `dist/checksums.txt`

Local example:

```bash
./scripts/build.sh v1.0.0 dist
```

## Release notes model

`scripts/render-release-notes.sh` generates the release body used by GitHub Actions. This keeps the release copy aligned with the actual asset set and avoids hand-edited drift.

## Repository metadata

Recommended GitHub repository settings:

- description:
  `AI-native Git workflow for local repositories with visible context, memory, raw output, and execution flow.`
- website:
  `https://github.com/Joker-of-Gotham/gitdex#readme`
- topics:
  `git`, `tui`, `ollama`, `local-first`, `observability`, `developer-tools`, `terminal-ui`, `ai-workflow`

These settings are applied in the GitHub web UI after the repository exists.

## Community health

The repository is expected to ship:

- `LICENSE`
- `CODE_OF_CONDUCT.md`
- `CONTRIBUTING.md`
- `SECURITY.md`
- issue templates
- pull request template

This aligns the GitHub sidebar and repository guidance with a public product repository.

## Release verification checklist

After a tag push:

1. confirm the `Release` workflow passed
2. open the GitHub release page
3. verify all six binaries exist
4. verify `gitdex-source.zip` exists
5. verify `checksums.txt` exists
6. verify the release body matches the product surface
