# Contributing

## Before you open a pull request

- read [README.md](README.md) for the product surface
- read [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md) for local setup
- check [SECURITY.md](SECURITY.md) first if the issue is security-sensitive

## Local setup

Requirements:

- Git
- Go
- Ollama

Install and verify:

```bash
git --version
go version
ollama --version
```

Run the local quality gate:

```bash
go vet ./...
go test ./...
go build ./...
```

Recommended before larger pull requests:

```bash
go test -race ./...
```

Windows equivalents:

```powershell
.\build.ps1 -Target test
.\build.ps1 -Target build
```

## Branch and commit guidance

- keep changes scoped to one clear concern
- prefer short-lived branches
- write commit messages that explain intent, not just file movement
- avoid bundling generated artifacts into source commits

Recommended commit sequence for larger work:

1. repository scaffolding and ignore rules
2. core source changes
3. docs, assets, and repository metadata

## Coding expectations

- preserve the local-first product direction
- keep AI reasoning and execution paths inspectable
- do not reintroduce dead code, hidden flows, or unused assets
- add tests when behavior, parsing, or state transitions change
- keep compatibility notes for `.gitmanualrc` and `GITMANUAL_*` only where they remain intentional

## Pull request checklist

- tests pass locally
- docs are updated if behavior changed
- screenshots or SVG placeholders are updated if the README surface changed
- no local caches, `bin/`, or `dist/` content are included
- release-related changes are reflected in [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) or [docs/PUBLISHING_TO_GITHUB.md](docs/PUBLISHING_TO_GITHUB.md) when relevant

## Release process

`gitdex` releases are created by pushing a `v*` tag. The GitHub `Release` workflow builds six binaries, generates `gitdex-source.zip`, writes `checksums.txt`, and publishes the release automatically.

See:

- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)
- [docs/PUBLISHING_TO_GITHUB.md](docs/PUBLISHING_TO_GITHUB.md)
