---
name: Gitdex Full Capability Upgrade
overview: Comprehensive audit-driven plan to eliminate all simulated/placeholder code, implement every missing Git/GitHub/filesystem/autonomy feature, upgrade the TUI to lazygit/gh-dash caliber with full editor and diff workbench, and wire a true 7x24 autonomous control plane with real persistence — covering 9 phases across approximately 45 implementation tasks.
todos:
  - id: p1-remove-simulated-export
    content: "Phase 1: Replace DefaultExportEngine and memoryQueryStore with ProviderExportEngine/ProviderQueryRouter in api/export.go, api/queries.go, cli/command/api.go, cli/command/export.go"
    status: completed
  - id: p1-remove-simulated-emergency
    content: "Phase 1: Remove simulated fallbacks in emergency/controls.go; implement pause_scope and suspend_capability against real scope registry"
    status: completed
  - id: p1-remove-simulated-autonomy
    content: "Phase 1: Remove simulated fallbacks in autonomy/recovery.go and autonomy/handoff.go; require real TaskStore; delete GenerateHandoffPackage simulated variant"
    status: completed
  - id: p1-remove-simulated-mutations
    content: "Phase 1: Delete SimulatedMutationEngine and SimulatedReleaseEngine; update 4 collab.go call sites to use GitHubMutationEngine"
    status: completed
  - id: p2-multi-remote-index
    content: "Phase 2: Extend localindex.go to index ALL remotes (origin/upstream/fork); add RemoteTopology to RepoContext; show mode badge in TUI"
    status: completed
  - id: p2-clone-and-switch
    content: "Phase 2: Add 'Clone & Switch' action on remote repos; display remote-only vs local-writable mode; show fork/upstream divergence in inspector"
    status: completed
  - id: p3-editor-undo-find
    content: "Phase 3: Add undo/redo stack, find/replace, go-to-line to file editor in filetree.go or new editor.go"
    status: completed
  - id: p3-split-diff
    content: "Phase 3: Add side-by-side diff, hunk-level staging, line-level staging, staged/unstaged toggle in diff_viewer.go"
    status: completed
  - id: p3-multi-buffer
    content: "Phase 3: Add multi-buffer management, buffer tabs, unsaved changes protection, external $EDITOR fallback"
    status: completed
  - id: p4-stash-view
    content: "Phase 4: Create stash.go TUI view with list, apply, pop, drop, show, branch-from-stash"
    status: completed
  - id: p4-tags-view
    content: "Phase 4: Create tags.go TUI view with list, create, delete, push"
    status: completed
  - id: p4-worktrees-view
    content: "Phase 4: Create worktrees.go TUI view with list, create, remove, lock/unlock, switch"
    status: completed
  - id: p4-reflog-view
    content: "Phase 4: Create reflog.go TUI view with scrollable entries and reset-to-point action"
    status: completed
  - id: p4-interactive-rebase
    content: "Phase 4: Create interactive_rebase.go with pick/squash/reword/edit/drop toggles and reorder"
    status: completed
  - id: p4-bisect-view
    content: "Phase 4: Create bisect.go with start/good/bad/skip/reset workflow"
    status: completed
  - id: p4-submodule-view
    content: "Phase 4: Create submodules.go with list, init, update, sync, add"
    status: completed
  - id: p4-commit-graph
    content: "Phase 4: Create commit_graph.go with ASCII graph rendering and compare-two-commits"
    status: completed
  - id: p4-remotes-view
    content: "Phase 4: Create remotes.go with list, add, remove, set-url, fetch, prune"
    status: completed
  - id: p4-conflict-resolver
    content: "Phase 4: Create conflict.go with three-way diff view and accept current/incoming/both per hunk"
    status: completed
  - id: p4-wire-slash-commands
    content: "Phase 4: Wire cherry-pick, reflog, bisect, submodule into TUI slash commands"
    status: completed
  - id: p5-github-releases
    content: "Phase 5: Complete Release CRUD in client.go; create releases.go TUI view with full lifecycle"
    status: completed
  - id: p5-github-discussions-fix
    content: "Phase 5: Fix collab create --type discussion to call CreateDiscussion not CreateIssue"
    status: completed
  - id: p5-github-branch-protection
    content: "Phase 5: Add Branch Protection/Rulesets API methods; display in inspector or dedicated view"
    status: completed
  - id: p5-github-milestones-projects
    content: "Phase 5: Add Milestones, Projects v2, Labels CRUD to client.go; create management views"
    status: completed
  - id: p5-github-workflow-logs
    content: "Phase 5: Add workflow run logs, artifacts, failed-job drill-down to workflows view"
    status: completed
  - id: p5-github-inline-review
    content: "Phase 5: Add inline PR review comments, review threads, check run drill-down to PR detail"
    status: completed
  - id: p5-deployment-actions
    content: "Phase 5: Add deployment environment actions (approve, reject, rollback, promote) to deployments view"
    status: completed
  - id: p6-filesystem-ops
    content: "Phase 6: Add chmod, symlink, archive, patch apply/check/reverse slash commands"
    status: completed
  - id: p6-batch-file-ops
    content: "Phase 6: Add batch rename/copy/move, binary file detection, large file streaming to filetree"
    status: completed
  - id: p7-wire-cruise
    content: "Phase 7: Wire CruiseEngine to daemon startup with --cruise flag; add concurrent execution control, retry, dead-letter, idempotency"
    status: completed
  - id: p7-policy-gates
    content: "Phase 7: Create policy_gate.go with approval thresholds, policy bundles, risk gates, mission windows"
    status: completed
  - id: p7-cruise-tui
    content: "Phase 7: Mount CruiseStatusView and ApprovalQueueView on router; connect to real cruise state"
    status: completed
  - id: p7-daemon-endpoints
    content: "Phase 7: Add cruise status/pause/resume, approvals CRUD, metrics endpoints to daemon"
    status: completed
  - id: p7-llm-planner
    content: "Phase 7: Upgrade planning/compiler to optional LLM-assisted planning producing planning.Plan objects"
    status: completed
  - id: p8-real-persistence
    content: "Phase 8: Default API router uses ProviderQueryRouter; workspace views connect to real stores; delete all non-store constructors from public API"
    status: completed
  - id: p9-visual-polish
    content: "Phase 9: Fix tab hints, wire Ctrl+R refresh, enable mouse, implement Plans drill, update README, delete orphan code"
    status: completed
isProject: false
---

# Gitdex Full Capability Upgrade Plan

## Audit Summary

The codebase has a working TUI shell (Bubble Tea v2 + Lip Gloss + Glamour + Chroma + Huh), real GitHub REST/GraphQL integration, real gitops wrappers around the `git` binary, daemon with webhooks, and LLM-backed autonomy planning. However, large swaths of the product promise remain undelivered:

- **14 simulated/placeholder code paths** in production code (export, queries, controls, handoff, recovery, mutations, discussions-as-issues)
- **Editor** is a minimal textarea (no undo/redo, find/replace, split diff, hunk staging)
- **Git TUI** is incomplete (no stash/tag/worktree/reflog/bisect/submodule/interactive-rebase dedicated views)
- **GitHub** is incomplete (no real Discussions, Releases CRUD, Branch Protection, Milestones, Projects)
- **Filesystem** is incomplete (no chmod, symlink, archive, patch apply features)
- **Autonomy** has simulated fallbacks; CruiseEngine is not wired; no concurrent execution control
- **API/Control plane** has memory-only query store and simulated export engine in default paths
- **Workspace** views (Plans, Tasks, Evidence) are derived from summary data, not backed by real stores

## Phase 1: Remove All Simulated/Placeholder Code

Every `simulated`/`mock` branch in non-test code must be replaced with real store-backed paths.

**Files to modify:**

- [internal/api/export.go](internal/api/export.go) -- Replace `DefaultExportEngine` with `ProviderExportEngine`; remove simulated JSON
- [internal/api/queries.go](internal/api/queries.go) -- Replace `memoryQueryStore` with `ProviderQueryRouter`; delete seeded fake data
- [internal/emergency/controls.go](internal/emergency/controls.go) -- Remove simulated fallback branches in halt/pause/kill; require real stores; implement `pause_scope` and `suspend_capability` against real scope registry
- [internal/autonomy/recovery.go](internal/autonomy/recovery.go) -- Remove simulated assessment/execution fallbacks; require TaskStore
- [internal/autonomy/handoff.go](internal/autonomy/handoff.go) -- Remove `GenerateHandoffPackage` simulated text; route all callers to `GenerateHandoffPackageFromStores`
- [internal/collaboration/mutations.go](internal/collaboration/mutations.go) -- Remove `SimulatedMutationEngine` entirely; all CLI callers must use `GitHubMutationEngine`
- [internal/collaboration/release.go](internal/collaboration/release.go) -- Remove `SimulatedReleaseEngine`; already unused by CLI but should be deleted
- [internal/cli/command/collab.go](internal/cli/command/collab.go) -- Replace 4 call sites of `NewSimulatedMutationEngine` with `NewGitHubMutationEngine`
- [internal/cli/command/api.go](internal/cli/command/api.go) -- Replace `defaultAPIRouter = api.NewMemoryAPIRouter()` with provider-backed router; remove hardcoded `simulated:true` in exchange export

**Rule:** After this phase, `rg -i simulated internal/` on non-test files must return zero matches.

## Phase 2: Unified Repository Context

Current local clone detection is single-remote-URL focused. Upgrade to multi-remote topology.

- **[internal/gitops/localindex.go](internal/gitops/localindex.go)** -- Extend `indexRepo` to index ALL remotes (origin, upstream, fork), not just the first found; store `map[normalizedURL][]WorktreePath` per remote name; add `LookupByAnyRemote(owner, name)` that matches across all remote URLs
- **[internal/state/repo/model.go](internal/state/repo/model.go)** -- Add `RemoteTopology` field: `map[remoteName]remoteURL`; add `IsFork`, `UpstreamURL`, `DefaultBranch`, `ProtectedBranches` fields
- **[internal/tui/app/app.go](internal/tui/app/app.go)** -- In `fetchRepos`, populate `RemoteTopology` from local git and GitHub API; display "remote-only | local-writable" mode badge; add "Clone & Switch" action (`c` key on remote repo)
- **[internal/tui/panes/inspector.go](internal/tui/panes/inspector.go)** -- Show fork/upstream divergence, all remotes, protected branches, mirror status in Context card

## Phase 3: Full Editor & Diff Workbench

Upgrade from textarea to a real terminal editor workbench. Reference: lazygit staging, diffnav side-by-side.

- **[internal/tui/views/filetree.go](internal/tui/views/filetree.go)** -- Major rewrite:
  - Add undo/redo stack (ring buffer of snapshots)
  - Add `/` find and `Ctrl+H` replace within editor
  - Add `Ctrl+G` go-to-line, `Ctrl+F` find-next
  - Add split diff mode: side-by-side (`s` key toggle) using two viewports
  - Add hunk-level staging: parse diff hunks, allow `Space` to stage/unstage individual hunks, `a` for whole file
  - Add line-level staging: `v` visual select mode, `Space` to stage selected lines
  - Add `Ctrl+Z`/`Ctrl+Y` undo/redo
  - Add unsaved changes indicator and confirmation on quit
  - Add word-diff mode toggle
  - Add staged/unstaged diff toggle (`Tab` to switch)
- **New file: [internal/tui/views/editor.go](internal/tui/views/editor.go)** -- Extract editor into dedicated `EditorView` with:
  - Multi-buffer management (open multiple files)
  - Buffer tabs at top
  - Line numbers with current line highlight
  - Syntax highlighting via `render.Code()`
  - External editor fallback (`$EDITOR` launch via `os.Exec`)
- **New file: [internal/tui/views/diff_viewer.go](internal/tui/views/diff_viewer.go)** -- Dedicated diff viewer:
  - Unified and side-by-side modes
  - Word-level diff highlighting
  - Hunk navigation (`]c` next hunk, `[c` prev hunk)
  - Apply/reverse/check patch operations
  - Integration with `gitops.PatchManager`

## Phase 4: Complete Git TUI Views

Each missing git capability gets a dedicated TUI workbench. Reference: lazygit.

- **New file: [internal/tui/views/stash.go](internal/tui/views/stash.go)** -- Stash browser:
  - List all stashes with message/date
  - Actions: apply, pop, drop, show diff, branch from stash
  - Keys: `Enter` show, `a` apply, `p` pop, `x` drop, `b` branch
- **New file: [internal/tui/views/tags.go](internal/tui/views/tags.go)** -- Tag browser:
  - List tags (annotated vs lightweight)
  - Actions: create, delete, push to remote
  - Show tag message and associated commit
- **New file: [internal/tui/views/worktrees.go](internal/tui/views/worktrees.go)** -- Worktree manager:
  - List all worktrees with branch/path/lock status
  - Actions: create, remove, lock/unlock, switch, inspect diff
  - Backed by `gitops.WorktreeManager`
- **New file: [internal/tui/views/reflog.go](internal/tui/views/reflog.go)** -- Reflog viewer:
  - Scrollable reflog entries with action type badges
  - "Reset to this point" action
  - Backed by `gitops.IntegrityChecker.Reflog`
- **New file: [internal/tui/views/interactive_rebase.go](internal/tui/views/interactive_rebase.go)** -- Interactive rebase:
  - List commits with pick/squash/reword/edit/drop toggles
  - Reorder by moving entries
  - Execute rebase
  - Backed by `gitops.BranchManager`
- **New file: [internal/tui/views/bisect.go](internal/tui/views/bisect.go)** -- Bisect workflow:
  - Start/reset bisect
  - Good/bad/skip current commit
  - Show remaining range
  - Backed by `GitExecutor.Run("bisect", ...)`
- **New file: [internal/tui/views/submodules.go](internal/tui/views/submodules.go)** -- Submodule manager:
  - List submodules with status/path/URL
  - Actions: init, update, sync, add
  - Backed by `gitops.RemoteManager`
- **New file: [internal/tui/views/commit_graph.go](internal/tui/views/commit_graph.go)** -- Commit graph:
  - ASCII graph from `git log --graph --oneline --all`
  - Compare two commits side-by-side
  - Navigate branches visually
- **New file: [internal/tui/views/remotes.go](internal/tui/views/remotes.go)** -- Remote manager:
  - List remotes with URLs and fetch/push status
  - Actions: add, remove, set-url, fetch, prune
- **New file: [internal/tui/views/conflict.go](internal/tui/views/conflict.go)** -- Merge conflict resolver:
  - Three-way diff view (ours/theirs/base)
  - Accept current/incoming/both per hunk
  - Mark resolved, continue merge/rebase
- **[internal/tui/views/explorer.go](internal/tui/views/explorer.go)** -- Extend sub-tabs to include: Stash, Tags, Worktrees, Reflog, Remotes (total 12+ tabs, or group under "Git" mega-tab with sub-navigation)
- **[internal/tui/app/commands.go](internal/tui/app/commands.go)** -- Wire `cherry-pick`, `reflog`, `bisect`, `submodule` into TUI slash commands

## Phase 5: Complete GitHub Integration

- **[internal/platform/github/client.go](internal/platform/github/client.go)** -- Add missing API methods:
  - `CreateRelease`, `UpdateRelease`, `PublishRelease`, `DeleteRelease`, `UploadReleaseAsset` (some already exist -- verify and complete CRUD)
  - `ListMilestones`, `CreateMilestone`, `UpdateMilestone`, `DeleteMilestone`
  - `GetBranchProtection`, `UpdateBranchProtection`, `ListRulesets`
  - `ListProjects`, `CreateProject`, `UpdateProject` (Projects v2 GraphQL)
  - `ListLabels`, `CreateLabel`, `UpdateLabel`, `DeleteLabel`
  - `ListCollaborators`, `AddCollaborator`, `RemoveCollaborator`
  - `GetWorkflowRunLogs`, `ListWorkflowArtifacts`, `DownloadArtifact`
  - `ListCheckRuns`, `ListCheckSuites` (already partial -- complete)
  - `ListReviewComments`, `CreateReviewComment` (inline PR review)
- **[internal/platform/github/discussions.go](internal/platform/github/discussions.go)** -- Already has real GraphQL. Fix CLI routing: `collab create --type discussion` must call `CreateDiscussion`, NOT `CreateIssue`
- **New file: [internal/tui/views/releases.go](internal/tui/views/releases.go)** -- Release lifecycle view:
  - List releases (draft/published/prerelease)
  - Create, edit, publish, delete
  - Asset management (upload/download)
- **Extend [internal/tui/views/deployments.go](internal/tui/views/deployments.go)** -- Add deployment actions:
  - Environment status overview
  - Approval/rejection
  - Rollback to previous deployment
  - Promote across environments
- **New file: [internal/tui/views/labels.go](internal/tui/views/labels.go)** -- Label/milestone management
- **Extend PR detail and Issue detail** -- Add inline review comments, review threads, check run drill-down, workflow artifact download

## Phase 6: Complete Filesystem Operations

- **[internal/tui/app/commands.go](internal/tui/app/commands.go)** -- Add slash commands:
  - `/chmod` -- change file permissions
  - `/symlink` -- create symbolic links
  - `/archive` -- create tar.gz/zip archives via `gitops.IntegrityChecker.Archive`
  - `/patch apply` -- apply patch file via `gitops.PatchManager.ApplyPatch`
  - `/patch check` -- dry-run patch via `gitops.PatchManager.ApplyPatch` with `--check`
  - `/patch reverse` -- reverse-apply patch
- **[internal/tui/views/filetree.go](internal/tui/views/filetree.go)** -- Add file operations:
  - Batch rename/copy/move (multi-select mode)
  - File permissions view and edit
  - Binary file detection with hex preview or "open externally" prompt
  - Large file streaming (don't load >1MB into memory for display)

## Phase 7: True 7x24 Autonomous Control Plane

- **Wire CruiseEngine** -- Connect `internal/autonomy/cruise.go` `CruiseEngine` to daemon startup (`internal/cli/command/daemon_hooks.go`); add `--cruise` flag to `daemon run`
- **[internal/autonomy/cruise.go](internal/autonomy/cruise.go)** -- Enhance:
  - Concurrent execution control (repo-level locks via sync.Mutex map)
  - Retry with exponential backoff
  - Dead-letter queue for failed cycles
  - Idempotency keys per action
  - Metrics: cycle count, success rate, mean duration
- **New file: [internal/autonomy/policy_gate.go](internal/autonomy/policy_gate.go)** -- Approval/policy/risk gates:
  - Per-action approval thresholds
  - Policy evaluation against stored policy bundles
  - Risk level gates (auto/manual/blocked)
  - Mission window enforcement (time-of-day, day-of-week)
- **Wire CruiseStatusView and ApprovalQueueView** -- Mount `CruiseStatusView` and `ApprovalQueueView` on the router under Workspace or Dashboard; these views already exist but are orphaned
- **[internal/daemon/server/handlers.go](internal/daemon/server/handlers.go)** -- Add endpoints:
  - `GET /api/v1/cruise/status` -- current cruise state
  - `POST /api/v1/cruise/pause|resume` -- operator control
  - `GET /api/v1/approvals` -- pending approval queue
  - `POST /api/v1/approvals/{id}/approve|reject` -- approval actions
  - `GET /api/v1/metrics` -- health/run-history/failure-triage
- **[internal/planning/compiler/compiler.go](internal/planning/compiler/compiler.go)** -- Upgrade to optional LLM-assisted planning: if LLM is configured, use it to decompose intents into multi-step plans (similar to `autonomy.Planner` but producing `planning.Plan` objects); keep deterministic fallback when LLM unavailable

## Phase 8: Real Persistence Everywhere

- **[internal/cli/command/api.go](internal/cli/command/api.go)** -- Default API router must check for configured storage provider and use `ProviderQueryRouter` + `ProviderExportEngine`; only fall back to error message (not simulated data) if no store configured
- **[internal/cli/command/export.go](internal/cli/command/export.go)** -- Always use `ProviderExportEngine`; remove `DefaultExportEngine` fallback
- **Workspace views** -- Connect Plans/Tasks/Evidence to real `storage.StorageProvider` instead of `syncDerivedWorkspace` summary derivation
- **Recovery/Handoff** -- All paths must use `FromStores` variants; remove non-store constructors from public API

## Phase 9: Visual Polish & UX Refinement

- **Explorer tab chrome** -- Fix "1-3 jump" hint to show actual count (7+ tabs); add grouped mega-tabs (GitHub | Git | Files)
- **Ctrl+R refresh** -- Wire `GlobalKeys.Refresh` in `app.go` Update to re-fetch current view data
- **Mouse support** -- Enable mouse events in Bubble Tea for scroll and click-to-focus
- **Plans "Enter drill"** -- Implement `Enter` handler in `PlansView.Update` to show plan detail
- **Delete orphan code** -- Remove `StatusPane`, `InputPane`, `ModeDetail` inspector enum if unused
- **README** -- Update root README.md to reflect actual shipping scope (currently describes only Story 1.2)

## Dependencies and Build Order

```mermaid
graph TD
    P1[Phase1_RemoveSimulated] --> P2[Phase2_UnifiedRepoContext]
    P1 --> P8[Phase8_RealPersistence]
    P2 --> P3[Phase3_EditorWorkbench]
    P2 --> P4[Phase4_GitTUIViews]
    P2 --> P5[Phase5_GitHubIntegration]
    P3 --> P6[Phase6_FilesystemOps]
    P4 --> P7[Phase7_AutonomyControlPlane]
    P5 --> P7
    P8 --> P7
    P7 --> P9[Phase9_VisualPolish]
    P6 --> P9
end
```



Phase 1 and 8 are prerequisites (real stores everywhere). Phases 2-6 can be parallelized after Phase 1. Phase 7 requires phases 4, 5, 8. Phase 9 is final polish.