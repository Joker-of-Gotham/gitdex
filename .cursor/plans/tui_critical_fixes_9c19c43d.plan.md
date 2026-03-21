---
name: TUI Critical Fixes
overview: "Fix 7 critical TUI issues: header tab overflow/overlap, inspector context not updating per sub-tab, settings editing UX, F6 keybinding, commit graph tree visualization, selection highlight gaps, and file editor responsiveness."
todos:
  - id: fix-header-overflow
    content: "Fix header tab overflow: truncate labels when total width exceeds terminal width; prevent overlap with help text"
    status: completed
  - id: fix-inspector-context
    content: "Make inspector context-aware: add SetCommitDetail/SetBranchDetail/SetPRDetail/SetFileDetail; hook explorer sub-tab changes to update inspector"
    status: completed
  - id: fix-file-editor
    content: "Fix file page: add loading indicator, enforce size limit, keep file tree visible in split layout, improve vim key bindings"
    status: completed
  - id: fix-settings-ux
    content: "Overhaul settings UX: clear LLM/GitHub/Git sections with descriptive labels, masked API keys, inline validation, direct editing"
    status: completed
  - id: fix-f6-binding
    content: Wire F6 key to Reflog view in GlobalKeys and app.go Update handler
    status: completed
  - id: fix-commit-graph
    content: "Enhance commit graph: per-branch coloring, graph toggle in commits tab, commit detail in inspector"
    status: completed
  - id: fix-selection-gaps
    content: "Fix selection highlight gaps: match FillBlock width to actual panel inner width; style column gutters with surface background"
    status: completed
  - id: fix-branch-encoding
    content: "Sanitize branch names: strip null bytes and control characters after git format parsing"
    status: completed
isProject: false
---

# TUI Critical Fixes Plan

## Issue 1: Header Tab Overlap (Fig 1)

**Root cause:** [header.go](internal/tui/components/header.go) line 108 -- `gap = h.width - lipgloss.Width(left) - lipgloss.Width(right)` has no truncation for `left`. When tab labels exceed available width, they overflow and overlap with the right-side help text. Also, 6 tabs are shown but only F1-F5 are bound.

**Fix:**

- When `left` exceeds `h.width - len(right) - 2`, truncate tab labels progressively: first remove help text, then shorten tab labels to abbreviations (e.g. "Dash" "Chat" "Expl" "Work" "Cfg" "Ref")
- Add F6 binding (see Issue 5 below)
- Use `runewidth.Truncate` for safe truncation

## Issue 2: Inspector Not Context-Aware (Fig 4)

**Root cause:** [inspector.go](internal/tui/panes/inspector.go) only has `ModeRepoDetail`, `ModeRisk`, `ModeEvidence`, `ModeAudit`. There is no `SetCommitDetail`, `SetBranchDetail`, `SetPRDetail` etc. The inspector always shows the same repo-level data regardless of which Explorer sub-tab is active.

Additionally, `syncChrome()` in [app.go](internal/tui/app/app.go) only runs on main view switch, NOT on explorer sub-tab changes. The `ExplorerView` in [explorer.go](internal/tui/views/explorer.go) has no callback to update the inspector.

**Fix:**

- Add new inspector data structs and setters:
  - `SetCommitDetail(hash, author, date, message, stats string)` -- shown when viewing commits
  - `SetBranchDetail(name, upstream, ahead, behind, lastCommit string)` -- shown when viewing branches
  - `SetPRDetail(number int, title, state, author, reviews, checks string)` -- for PR view
  - `SetIssueDetail(number int, title, state, labels, assignees string)` -- for Issues
  - `SetFileDetail(path, size, language, lastModified string)` -- for file preview
- Add `OnSubTabChanged` callback to `ExplorerView` so `app.go` can update inspector when user switches between Files/Commits/Branches/PRs/etc.
- When user selects a specific commit/branch/PR in the list, update the inspector with that item's details
- Inspector tabs should change contextually: in commit view show "Commit | Diff | Stats", in PR view show "PR | Reviews | Checks"

## Issue 3: File Page Freeze (Fig 3)

**Root cause:** File loading via `loadFileEditContent` in [app.go](internal/tui/app/app.go) reads up to the preview limit in a `tea.Cmd`. The freeze is likely from large files or the edit mode not properly receiving focus. Also the file tree disappears when previewing -- the file content takes over the entire main area without a split layout.

**Fix:**

- Add a loading indicator while file content is being read
- Ensure the file tree panel remains visible on the left when previewing/editing (split layout, not replacement)
- Improve vim-like key handling: ensure `i`/`a`/`o` enter INSERT mode, `Esc` returns to NORMAL, `:w` saves, `:q` quits editor, `:wq` save+quit
- Add visual mode indicator in status line: `-- NORMAL --` / `-- INSERT --`

## Issue 4: Settings UX Overhaul (Fig 5)

**Root cause:** The settings view in [settings.go](internal/tui/views/settings.go) has editing capability but the UX is confusing. Users cannot find where to configure:

- LLM provider/model/API key/endpoint
- GitHub token/username/email/host
- Local model selection (Ollama)

**Fix:**

- Restructure the settings into clear, user-friendly sections with descriptive labels:
  - **LLM Configuration**: Provider (openai/anthropic/ollama/deepseek), Model name, API Key (masked), Endpoint URL, Temperature
  - **GitHub Configuration**: Host (github.com), Username, Email, Personal Access Token (masked), SSH Key Path
  - **Git Identity**: Committer name, Committer email, GPG signing key
  - **Storage**: Backend type, Database path
- Each field should show a clear label, current value, and hint text
- Masked fields (API keys, tokens) should show `*`*** with a toggle to reveal
- Add inline validation: show red "Invalid" or green "Valid" status for API keys and URLs
- When entering a section, expand it to show all fields with direct editing (Enter or just start typing)

## Issue 5: F6 Key Not Wired

**Root cause:** [keymap.go](internal/tui/keymap/keymap.go) `DefaultGlobalKeys()` only defines `f1`-`f5` bindings. The Reflog view is registered as the 6th tab in the router but has no F6 shortcut.

**Fix:**

- Add `SwitchReflog` binding in `GlobalKeys` struct with `key.WithKeys("f6")`
- Add handler in `app.go` `Update` to call `m.switchView(views.ViewReflog)` on F6
- If more views are added later (beyond F6), use the command palette as fallback

## Issue 6: Commit/Branch Tree Visualization (Fig 6)

**Root cause:** [commit_graph.go](internal/tui/views/commit_graph.go) just runs `git log --graph --oneline --all --decorate` and displays the raw text. While this shows ASCII graph characters, it's not colorized per-branch like lazygit. Also the "normal" commits view in `commit_log.go` is purely linear with no graph lines at all.

**Fix:**

- Enhance the commit graph colorizer to assign colors per branch/author (cycle through a palette of 6-8 distinct colors)
- Parse the graph prefix characters (`*`, `|`, `/`, `\`, `-`) and color them based on which branch they belong to
- In the regular Commits tab, add a `g` key to toggle between linear and graph mode
- Selected commit should show details in the inspector (see Issue 2)
- Branch names in decorations should be colored distinctly from commit messages

## Issue 7: Selection Highlight Gaps (Fig 7, Fig 8)

**Root cause:** Two issues:

1. In [repos.go](internal/tui/views/repos.go) line 243, `FillBlock(..., width-4, ...)` uses a hardcoded offset that may not match the actual inner width computed by `SurfacePanel` via `GetHorizontalFrameSize()`
2. In [columns.go](internal/tui/layout/columns.go), the gutter between columns is a plain `" "` space with no background styling, creating a visible seam between panes

**Fix:**

- In `renderRepoCard`: compute selection width from the actual panel inner width, not `width-4`. Use `lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0,1).GetHorizontalFrameSize()` to get the real frame offset
- In `RenderColumns`: style the gutter space with the surface background color so there is no visible seam between panes
- Apply the same fix to all views that use `FillBlock` with selection: branch list, commit list, stash list, tags list, etc.

## Additional: Branch Name Encoding (Fig 4)

**Root cause:** Branch names show `%00` and raw hex sequences. The git format uses `\x00` as separator in [branch.go](internal/gitops/branch.go) via `--format=%(refname:short)%x00%(objectname)%x00...`. If the split fails (e.g. binary data in ref), raw null-encoded text leaks to display.

**Fix:**

- Add a sanitization step after parsing: strip any remaining `\x00`, `%00`, or non-printable characters from branch names before display
- If a branch name contains control characters after parsing, log a warning and display a cleaned version

