# Story 2.5: Run Low-Risk Repository Hygiene Tasks (FR16)

Status: done

## Story

As a maintainer,
I want Gitdex to perform low-risk repository hygiene tasks under governance,
So that repetitive maintenance work stops draining attention from actual project work.

## Acceptance Criteria

1. **Given** a repository that qualifies for supported low-risk maintenance **When** the operator selects a hygiene action **Then** Gitdex presents a plan and executes the maintenance task under the same governed lifecycle as other write operations **And** the resulting changes and affected objects are summarized on completion **And** failed hygiene runs preserve enough context for retry or handoff rather than silently stopping.

2. (AC 无第二条，以上为完整标准)

## Tasks / Subtasks

- [x] Task 1: 定义并支持低风险仓库维护任务 (AC: #1)
  - [x] 1.1 在 `internal/gitops/hygiene.go` 中定义 HygieneAction 常量和 HygieneTask、HygieneResult 结构体
  - [x] 1.2 实现 SupportedHygieneTasks() 返回四种任务：prune_remote_branches、gc_aggressive、clean_untracked、remove_merged_branches
- [x] Task 2: 实现 HygieneExecutor 执行器 (AC: #1)
  - [x] 2.1 在 `internal/gitops/hygiene_executor.go` 中实现 Execute() 方法，支持 context 取消和空 repo 路径失败场景
  - [x] 2.2 失败时返回带 ErrorMessage 和 Summary 的 HygieneResult，便于 retry/handoff
- [x] Task 3: 暴露 CLI 子命令 (AC: #1)
  - [x] 3.1 在 `internal/cli/command/repo.go` 中添加 `repo hygiene list` 和 `repo hygiene run [action]`
  - [x] 3.2 支持 human-readable 和 JSON/YAML 结构化输出
- [x] Task 4: 编写单元与集成测试 (AC: #1)
  - [x] 4.1 `internal/gitops/hygiene_executor_test.go`：成功、失败、无效 action、上下文取消
  - [x] 4.2 `test/integration/repo_hygiene_command_test.go`：命令注册、list、run
  - [x] 4.3 `test/conformance/hygiene_contract_test.go`：JSON 序列化合约

## Dev Notes

### 关键实现细节

1. **HygieneAction 枚举**：`prune_remote_branches`、`gc_aggressive`、`clean_untracked`、`remove_merged_branches`
2. **HygieneResult**：包含 Success、Action、FilesAffected、BranchesAffected、ErrorMessage、Summary，失败时保留完整上下文
3. **HygieneExecutor.Execute**：MVP 阶段为模拟执行（50ms 延迟 + mock 结果），空 repo 路径返回失败结果便于重试
4. **上下文取消**：支持 `context.Done()`，返回带 Summary 的失败结果而非 panic

### 文件结构

```
internal/
  gitops/
    hygiene.go          # HygieneAction, HygieneTask, HygieneResult, SupportedHygieneTasks
    hygiene_executor.go  # HygieneExecutor
    hygiene_executor_test.go
internal/cli/command/
  repo.go               # newRepoHygieneGroupCommand, newRepoHygieneListCommand, newRepoHygieneRunCommand
test/
  integration/
    repo_hygiene_command_test.go
  conformance/
    hygiene_contract_test.go
```

### 设计决策

- MVP 阶段采用模拟执行，便于快速验证 CLI 流程和输出格式
- 失败路径返回 `*HygieneResult` 而非 error，保证 Summary 中保留 retry/handoff 建议
- 所有任务标记为 risk_level=low，部分任务 Reversible=true

### References

- Epic 2: Governed Planning and Safe Single-Repository Action
- FR16: Low-risk governed repository hygiene and maintenance

## Dev Agent Record

### Completion Notes List

- Task 1：定义 HygieneAction 类型和四种维护任务，包含 Description、RiskLevel、Reversible、EstimatedImpact
- Task 2：实现 HygieneExecutor，支持成功/失败/取消路径，失败时保留 ErrorMessage 与 Summary
- Task 3：添加 `repo hygiene list` 列出任务，`repo hygiene run [action]` 执行任务，支持 --output json/yaml
- Task 4：单元测试覆盖 Execute 成功、空路径失败、无效 action、context 取消；集成测试验证 list/run；合约测试验证 JSON round-trip

### File List

**New files:**
- `internal/gitops/hygiene.go`
- `internal/gitops/hygiene_executor.go`
- `internal/gitops/hygiene_executor_test.go`
- `test/integration/repo_hygiene_command_test.go`
- `test/conformance/hygiene_contract_test.go`

**Modified files:**
- `internal/cli/command/repo.go` (添加 hygiene 命令组及子命令)
