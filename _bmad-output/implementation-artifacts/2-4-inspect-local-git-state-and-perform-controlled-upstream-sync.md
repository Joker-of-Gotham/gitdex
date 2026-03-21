# Story 2.4: Inspect Local Git State and Perform Controlled Upstream Sync

Status: done

## Story

As a maintainer,
I want to inspect my repository state and run controlled upstream synchronization workflows,
so that I do not have to manually diagnose divergence or hand-compose the safest sync path.

## Acceptance Criteria

1. **Given** a repository with local and remote branch state **When** the operator asks Gitdex to inspect or sync with upstream **Then** Gitdex presents branch state, diffs, divergence, and the recommended sync action before any write occurs.

2. **And** supported sync actions run in a governed mode with previewable impact.

3. **And** conflict or blocked scenarios are surfaced with a clear explanation and safe next step.

## Tasks / Subtasks

- [ ] Task 1: Git 状态检查数据模型 (AC: #1)
  - [ ] 1.1 新增 `internal/gitops/inspect.go`，定义 `RepoInspection` 结构体：local_branch, remote_branch, ahead, behind, has_uncommitted, has_untracked, divergence_state
  - [ ] 1.2 定义 `DivergenceState` 枚举：`synced`, `ahead`, `behind`, `diverged`, `detached`, `no_upstream`
  - [ ] 1.3 定义 `SyncRecommendation` 结构体：action, risk_level, description, previewable

- [ ] Task 2: Git 状态检查器 (AC: #1)
  - [ ] 2.1 新增 `internal/gitops/inspector.go`，定义 `Inspector` 结构体
  - [ ] 2.2 `Inspect(ctx, repoPath) (*RepoInspection, error)` — 使用 git 命令获取分支状态、ahead/behind、uncommitted changes
  - [ ] 2.3 `Recommend(inspection) *SyncRecommendation` — 基于 divergence state 生成同步建议
  - [ ] 2.4 MVP 使用 `os/exec` 调用 git 命令，不依赖 libgit2

- [ ] Task 3: 受控同步操作 (AC: #2, #3)
  - [ ] 3.1 新增 `internal/gitops/syncer.go`，定义 `Syncer` 结构体
  - [ ] 3.2 `Preview(ctx, repoPath, action) (*SyncPreview, error)` — 预览同步影响
  - [ ] 3.3 `Execute(ctx, repoPath, action) (*SyncResult, error)` — 执行同步操作（作为治理任务）
  - [ ] 3.4 `SyncPreview` 包含：affected_files, merge_strategy, conflict_risk
  - [ ] 3.5 `SyncResult` 包含：success, files_changed, conflicts, error_message
  - [ ] 3.6 冲突场景返回清晰的说明和安全下一步

- [ ] Task 4: CLI `gitdex repo sync` 命令 (AC: #1, #2, #3)
  - [ ] 4.1 `gitdex repo inspect` — 显示仓库状态、分支、divergence、建议
  - [ ] 4.2 `gitdex repo sync --preview` — 预览同步影响
  - [ ] 4.3 `gitdex repo sync --execute` — 执行受控同步（需要计划+审批流程）
  - [ ] 4.4 支持 text/JSON/YAML 输出格式
  - [ ] 4.5 冲突时显示清晰的 blocker 说明和建议

- [ ] Task 5: 全面测试 (AC: #1-#3)
  - [ ] 5.1 `internal/gitops/inspect_test.go` — 数据模型测试
  - [ ] 5.2 `internal/gitops/inspector_test.go` — 检查器逻辑测试
  - [ ] 5.3 `internal/gitops/syncer_test.go` — 同步器测试
  - [ ] 5.4 `test/integration/repo_sync_command_test.go` — CLI 命令注册
  - [ ] 5.5 `test/conformance/repo_inspection_contract_test.go` — 合约测试
  - [ ] 5.6 运行 `go test ./... -count=1` + `golangci-lint run ./...`

- [ ] Task 6: 收尾验证 (AC: #1-#3)
  - [ ] 6.1 验证范围：MVP inspect + sync 骨架，不含完整 worktree 隔离（Story 2.6）
  - [ ] 6.2 确认与 planning/orchestrator 的接口兼容
  - [ ] 6.3 更新 sprint-status.yaml

## Dev Notes

### 从前序 Story 学到的

- Executor 需要并发保护（Story 2.3）
- SaveApproval 应在 Save 之前（Story 2.2）
- 0-steps 计划需要验证（Story 2.3）
- gofmt 格式化需要每次修改后检查

### 架构约束

1. **所有写操作必须经过 plan → policy → approval → execute 流程**
2. **sync 是受治理操作**，不能直接执行 git pull/merge
3. **冲突场景必须 surface 而非静默处理**
4. **MVP 使用 os/exec 调用 git，不引入 libgit2**

### References

- [Source: architecture.md §Repository & Collaboration Operations]
- [Source: prd.md §FR14, FR15]
- [Source: epics.md §Story 2.4]

## Dev Agent Record

(to be filled)

## Change Log

| Date | Change | Author |
|------|--------|--------|
| 2026-03-18 | Story created | Agent |
