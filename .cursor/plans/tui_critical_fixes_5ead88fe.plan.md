---
name: TUI Critical Fixes
overview: 修正 9 个 TUI 关键问题：头部标签溢出、Inspector 不随上下文变化、文件编辑器需支持 128MB 流畅操作、设置 UX、F6 按键、提交图谱树状可视化、选中高亮覆盖缺口、分支名编码、以及全面修复所有面板的滚动截断问题。
todos:
  - id: fix-header-overflow
    content: "Fix header tab overflow: truncate labels when total width exceeds terminal width; prevent overlap with help text"
    status: completed
  - id: fix-inspector-context
    content: "Make inspector context-aware: add SetCommitDetail/SetBranchDetail/SetPRDetail/SetFileDetail; hook explorer sub-tab changes to update inspector"
    status: pending
  - id: fix-file-editor
    content: "File editor 128MB support: increase maxFilePreviewBytes to 128MiB, chunked syntax highlighting for visible window only, loading indicator, keep file tree visible, vim keybindings"
    status: pending
  - id: fix-settings-ux
    content: "Overhaul settings UX: clear LLM/GitHub/Git sections with descriptive labels, masked API keys, inline validation, direct editing"
    status: pending
  - id: fix-f6-binding
    content: Wire F6 key to Reflog view in GlobalKeys and app.go Update handler
    status: completed
  - id: fix-commit-graph
    content: "Enhance commit graph: per-branch coloring, graph toggle in commits tab, commit detail in inspector"
    status: pending
  - id: fix-selection-gaps
    content: "Fix selection highlight gaps: match FillBlock width to actual panel inner width; style column gutters with surface background"
    status: pending
  - id: fix-branch-encoding
    content: "Sanitize branch names: strip null bytes and control characters after git format parsing"
    status: in_progress
  - id: fix-universal-scroll
    content: "Universal scroll fix: add viewport to plans/tasks/evidence, forward keys to vp in tags/worktrees, add PgUp/PgDn to commits/branches/PRs/issues/deployments/workflows, fix repos linesPerRepo height calc, add detail viewport to commit_log"
    status: pending
isProject: false
---

# TUI Critical Fixes Plan (Revised)

## Issue 1: Header Tab Overlap

**Root cause:** [header.go](internal/tui/components/header.go) line 108 -- `gap = h.width - lipgloss.Width(left) - lipgloss.Width(right)` has no truncation for `left`. When tab labels exceed available width, they overflow and overlap with the right-side help text. Also, 6 tabs are shown but only F1-F5 are bound.

**Fix:**

- When `left` exceeds `h.width - len(right) - 2`, truncate tab labels progressively: first remove help text, then shorten tab labels to abbreviations (e.g. "Dash" "Chat" "Expl" "Work" "Cfg" "Ref")
- Add F6 binding (see Issue 5 below)
- Use `runewidth.Truncate` for safe truncation

## Issue 2: Inspector Not Context-Aware

**Root cause:** [inspector.go](internal/tui/panes/inspector.go) only has `ModeRepoDetail`, `ModeRisk`, `ModeEvidence`, `ModeAudit`. No `SetCommitDetail`, `SetBranchDetail`, `SetPRDetail` etc. `syncChrome()` in [app.go](internal/tui/app/app.go) only runs on main view switch.

**Fix:**

- Add new inspector data structs and setters: `SetCommitDetail`, `SetBranchDetail`, `SetPRDetail`, `SetIssueDetail`, `SetFileDetail`
- Add `OnSubTabChanged` callback to `ExplorerView` so `app.go` can update inspector when user switches sub-tabs
- When user selects a specific commit/branch/PR in the list, update the inspector with that item's details
- Inspector tabs should change contextually

## Issue 3: File Editor -- 128MB Streaming Support

**Root cause:** [app.go](internal/tui/app/app.go) line 2266 defines `maxFilePreviewBytes = 1 << 20` (1 MiB). Any file larger than 1MB gets truncated. The user requires smooth CRUD for files up to 128MB.

**Fix (tiered approach):**

- **Increase limit:** Change `maxFilePreviewBytes` from `1 << 20` to `1 << 27` (128 MiB)
- **Chunked preview rendering:** In [filetree.go](internal/tui/views/filetree.go), do NOT pass the entire 128MB string to Chroma for highlighting. Instead:
  - Split content into lines; store as `[]string` in the view
  - Only syntax-highlight the visible window (viewport offset +/- buffer of ~100 lines) via a `highlightVisibleChunk()` helper
  - The `codeVP` viewport displays the highlighted chunk; on scroll, re-highlight the new visible range
- **Editing large files:** For files under 10MB, load entirely into `textarea`. For files 10MB-128MB, show a warning banner "Large file -- recommended: press `o` for external editor" but still allow inline editing with the `textarea` (Go's string type can handle it)
- **Loading indicator:** Show a spinner/progress message while reading large files (e.g. "Loading 45.2 MB...")
- **Binary detection:** Keep `sniffBinaryBytes = 8192` for binary detection; binary files still show hex dump or "open externally" prompt
- **Keep file tree visible:** The file tree panel must remain on the left during preview/edit (split layout in `renderWorkbench`)
- **Vim keybindings:** Ensure `i`/`a`/`o` enter INSERT, `Esc` returns NORMAL, `:w` saves, `:q` quits, `:wq` save+quit; visual mode indicator in status line

## Issue 4: Settings UX Overhaul

**Root cause:** [settings.go](internal/tui/views/settings.go) settings are confusing. Users cannot easily find/configure LLM providers, API keys, GitHub credentials, or local models.

**Fix:**

- Restructure into clear sections: LLM Configuration, GitHub Configuration, Git Identity, Storage
- Each field: clear label + current value + hint text
- Masked fields (API keys, tokens) show `*`*** with toggle to reveal
- Inline validation for API keys and URLs
- Direct editing on Enter or typing

## Issue 5: F6 Key Not Wired

**Root cause:** [keymap.go](internal/tui/keymap/keymap.go) only defines `f1`-`f5`. Reflog view has no F6 shortcut.

**Fix:**

- Add `SwitchReflog` in `GlobalKeys` with `key.WithKeys("f6")`
- Add handler in `app.go` `Update` for F6

## Issue 6: Commit/Branch Tree Visualization

**Root cause:** [commit_graph.go](internal/tui/views/commit_graph.go) displays raw `git log --graph` text without per-branch coloring.

**Fix:**

- Per-branch coloring with a palette of 6-8 colors
- Parse graph prefix characters and color by branch
- In Commits tab, add `g` key to toggle linear/graph mode
- Selected commit updates inspector

## Issue 7: Selection Highlight Gaps

**Root cause:** [repos.go](internal/tui/views/repos.go) uses hardcoded `width-4` in `FillBlock` that doesn't match real inner width. [columns.go](internal/tui/layout/columns.go) gutter is plain `" "` with no background.

**Fix:**

- Compute selection width from actual panel inner width
- Style gutter with surface background color
- Apply same fix to all views using `FillBlock`

## Issue 8: Branch Name Encoding

**Root cause:** [branch.go](internal/gitops/branch.go) uses `\x00` separators; failed parsing leaks raw null bytes.

**Fix:**

- Sanitize after parsing: strip `\x00`, `%00`, non-printable characters

## Issue 9: Universal Scrolling and Truncation Fix (NEW -- Critical)

**Root cause:** `RenderColumns` in [columns.go](internal/tui/layout/columns.go) line 28-30 applies `Height(h).MaxHeight(h)` to every column. Any view that renders more lines than `ContentHeight()` without internal scrolling gets **hard-clipped** by lipgloss. This affects the majority of views. Detailed audit:

### Category A -- No scroll at all (content silently clipped):


| View                 | File                   | Issue                                                                                         |
| -------------------- | ---------------------- | --------------------------------------------------------------------------------------------- |
| Plans list           | `plans.go` L178-196    | Renders ALL plan cards in a loop, no cursor/scroll/viewport                                   |
| Tasks list+detail    | `tasks.go` L104-115    | Renders ALL task cards, no scroll                                                             |
| Evidence list+detail | `evidence.go` L104-123 | Renders ALL entries, no scroll                                                                |
| Settings body        | `settings.go`          | Section content has no viewport; pgup/pgdown only navigates between sections, not within them |


### Category B -- viewport.Model exists but key events not forwarded (scroll broken):


| View             | File                                  | Issue                                                                    |
| ---------------- | ------------------------------------- | ------------------------------------------------------------------------ |
| Tags detail      | `tags.go` L75-94 `syncDetailViewport` | `vp` exists, but `handleKey` (L134-181) never calls `vp.Update(msg)`     |
| Worktrees detail | `worktrees.go` L64-68                 | Same: `vp` present, `handleKey` (L124-174) never forwards to `vp.Update` |


### Category C -- Manual cursor scroll but missing PgUp/PgDn:


| View             | File                  | Issue                                          |
| ---------------- | --------------------- | ---------------------------------------------- |
| Commits list     | `commit_log.go` L214  | Manual `viewH` windowing, no PgUp/PgDn handler |
| Branches list    | `branch_tree.go` L243 | Manual `viewH` windowing, no PgUp/PgDn         |
| PRs list         | `pulls.go` L194       | Manual `scroll` index, no PgUp/PgDn            |
| Issues list      | `issues.go` L191      | Manual `scroll` index, no PgUp/PgDn            |
| Deployments list | `deployments.go` L110 | Windowed, no PgUp/PgDn                         |
| Workflows list   | `workflows.go` L218   | Windowed, no PgUp/PgDn                         |


### Category D -- Height calculation bugs causing clipping:


| View          | File                     | Issue                                                                                                                                                                                                                |
| ------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Repos list    | `repos.go` L14           | `linesPerRepo=5` but `renderRepoCard` outputs 6 lines when `IsLocal && len(LocalPaths)>0` (extra path line at L235-236). Underestimates card height, causing visible cards to exceed allocated space and get clipped |
| Commit detail | `commit_log.go` L238-257 | `renderDetail` outputs unbounded `render.Code(...)` content with no viewport -- long patches overflow the panel                                                                                                      |


**Comprehensive fix strategy:**

1. **Category A (no scroll):** For each of `plans.go`, `tasks.go`, `evidence.go`: add a `viewport.Model` field; in `SetSize`, set viewport dimensions; in `View()`, set viewport content to the rendered list, return `vp.View()`; in `Update`, forward key/mouse messages to `vp.Update`. For `settings.go`: add a content viewport for the active section detail so long sections can scroll independently of section navigation.
2. **Category B (broken vp forwarding):** In `tags.go` and `worktrees.go` `handleKey`: add a `default` branch that calls `v.vp, cmd = v.vp.Update(msg)` and returns `cmd`, so arrow keys / mouse wheel / PgUp/PgDn reach the viewport.
3. **Category C (missing PgUp/PgDn):** In each of `commit_log.go`, `branch_tree.go`, `pulls.go`, `issues.go`, `deployments.go`, `workflows.go`: add `pgup`/`pgdown` cases in `handleKey` that jump the cursor by `viewH/2` (half-page scroll), clamped to bounds.
4. **Category D (height bugs):** In `repos.go`: change `linesPerRepo` from `5` to dynamically compute based on `renderRepoCard` actual line count (or set to `7` to accommodate all card variants). In `commit_log.go`: wrap `renderDetail` output in a `detailVP viewport.Model` so long patches scroll instead of overflowing.

**Key files to modify:**

- [plans.go](internal/tui/views/plans.go) -- add viewport
- [tasks.go](internal/tui/views/tasks.go) -- add viewport
- [evidence.go](internal/tui/views/evidence.go) -- add viewport
- [settings.go](internal/tui/views/settings.go) -- add section content viewport
- [tags.go](internal/tui/views/tags.go) -- forward keys to vp
- [worktrees.go](internal/tui/views/worktrees.go) -- forward keys to vp
- [commit_log.go](internal/tui/views/commit_log.go) -- add PgUp/PgDn + detail viewport
- [branch_tree.go](internal/tui/views/branch_tree.go) -- add PgUp/PgDn
- [pulls.go](internal/tui/views/pulls.go) -- add PgUp/PgDn
- [issues.go](internal/tui/views/issues.go) -- add PgUp/PgDn
- [deployments.go](internal/tui/views/deployments.go) -- add PgUp/PgDn
- [workflows.go](internal/tui/views/workflows.go) -- add PgUp/PgDn
- [repos.go](internal/tui/views/repos.go) -- fix linesPerRepo height

