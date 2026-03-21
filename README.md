# Gitdex

Gitdex is a terminal-first control plane for Git, GitHub, and governed automation. It ships as a Go CLI (`gitdex`), an optional daemon (`gitdexd`), and a full-screen Bubble Tea TUI for interactive operations.

## TUI

- **Navigation**: F1–F5 switch Dashboard, Chat, Explorer, Workspace, and Settings; command palette (`Ctrl+K`-style binding per keymap); inspector toggle; theme cycling; **mouse** enabled via Bubble Tea v2 `View.MouseMode` (`MouseModeCellMotion`) so the terminal sends wheel/click events to the app.
- **Refresh**: `Ctrl+R` reloads data for the active view (repos and health on Dashboard; Explorer file tree, GitHub objects, and current file preview; Workspace plans/tasks/evidence from summary or stores; Settings from disk; Chat is unchanged).
- **Explorer**: GitHub-oriented tabs (PRs, issues, workflows, deployments, releases) and **Git** tabs (files, commits, branches) are grouped as mega-tabs; `[` / `]` switch groups; number keys apply within the current group.
- **Workspace**: Plans, tasks, evidence, cruise status, and approval queue; **Enter** on a plan opens a scrollable steps/tasks drill; store-backed plans include compiled steps and linked tasks.
- **Chat**: LLM-assisted session with streaming (interrupt with `Esc` / `Ctrl+C`).

## Git integration

Local repository discovery, file tree and preview, commits, branches, diffs, editing, stash/tags/worktrees/remotes/reflog views, and Git execution through the tool/autonomy layers. Explorer ties local paths to GitHub metadata when a remote is configured.

## GitHub integration

Authenticated client usage for repositories, PRs, issues, Actions runs, deployments, releases, and remote file access when not working from a full local clone.

## Autonomy and daemon

- **Autonomy**: Planned actions, policy/risk awareness, and execution flows used from chat and automation paths.
- **Daemon** (`gitdex daemon run` / `gitdexd run`): HTTP control plane, webhooks, and cruise-style processing aligned with the internal autonomy engine.

## File operations

Create/move/delete, batch renames, patches, staging helpers, and external editor handoff where implemented in the Files explorer tab.

## Quick start

```powershell
go run ./cmd/gitdex --help
go run ./cmd/gitdex doctor
go run ./cmd/gitdex config show
```

Example config: `configs/gitdex.example.yaml`. Precedence: flags > environment (`GITDEX_*`) > repo `.gitdex/config.yaml` > user config > defaults.

## Validation

```powershell
go test ./...
go build ./...
golangci-lint run
```
