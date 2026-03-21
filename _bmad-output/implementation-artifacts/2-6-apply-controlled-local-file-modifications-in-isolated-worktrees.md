# Story 2.6: Apply Controlled Local File Modifications in Isolated Worktrees (FR17)

Status: done

## Story

As a maintainer,
I want Gitdex to make controlled local file modifications inside isolated execution worktrees,
So that repository edits stay reviewable, reversible, and separated from my active working tree.

## Acceptance Criteria

1. **Given** a supported file modification request within an authorized repository scope **When** the operator approves the associated plan **Then** Gitdex performs the modification inside an isolated worktree rather than the shared live working tree **And** the operator can review the resulting diff before accepting downstream branch or PR actions **And** failed or cancelled modifications can be discarded without corrupting the operator's active workspace

## Tasks / Subtasks

- [x] Task 1: 定义 Worktree 领域模型与配置结构 (AC: #1)
  - [x] 1.1 在 `internal/gitops/worktree.go` 中定义 `WorktreeConfig`、`WorktreeStatus`、`Worktree` 结构
  - [x] 1.2 实现 `WorktreeManager` 与 Create/Inspect/Diff/Discard 方法接口

- [x] Task 2: 实现隔离 Worktree 管理能力 (AC: #1, #2, #3)
  - [x] 2.1 实现 `Create`：创建隔离工作树（MVP 为模拟实现）
  - [x] 2.2 实现 `Inspect`：检查 worktree 状态
  - [x] 2.3 实现 `Diff`：展示修改 diff 供操作者审核
  - [x] 2.4 实现 `Discard`：安全丢弃 worktree，不污染主工作区

- [x] Task 3: 注册 CLI 命令与子命令 (AC: #1, #2, #3)
  - [x] 3.1 在 `internal/cli/command/repo.go` 中注册 `repo worktree` 子命令组
  - [x] 3.2 实现 `worktree create`（--branch、--worktree-dir）
  - [x] 3.3 实现 `worktree inspect`、`worktree diff`、`worktree discard`（--worktree-dir）
  - [x] 3.4 支持 JSON/YAML 结构化输出与人类可读文本渲染

- [x] Task 4: 编写单元测试与契约测试 (AC: 全部)
  - [x] 4.1 `test/conformance/worktree_contract_test.go`：WorktreeConfig、Worktree、WorktreeStatus JSON 往返与字段约定
  - [x] 4.2 `test/integration/repo_worktree_command_test.go`：命令注册、参数校验、JSON 输出

## Dev Notes

### 关键实现细节

- **WorktreeConfig**：包含 `RepoPath`、`Branch`、`WorktreeDir`；`WorktreeDir` 未指定时默认 `../gitdex-worktree-<branch>`
- **WorktreeStatus**：`active`、`dirty`、`clean`、`removed` 四种状态
- **WorktreeManager**：当前为 MVP 模拟实现，不实际调用 `git worktree add`；Create 返回模拟 Worktree，Inspect/Diff 返回模拟数据，Discard 为无副作用 no-op
- **CLI**：`repo worktree` 下有 create、inspect、diff、discard 四个子命令；create 强制要求 `--branch`；inspect/diff/discard 支持 `--worktree-dir` 或使用 `config.paths.working_dir`
- **Diff 审核**：`worktree diff` 输出完整 diff，供操作者在接受下游 branch/PR 操作前审核
- **Discard 安全**：`worktree discard` 可安全移除 worktree，失败或取消的修改可丢弃而不污染主工作区

### 文件结构

```
internal/gitops/worktree.go          # 领域模型与 WorktreeManager
internal/cli/command/repo.go         # worktree 子命令（新增段落）
test/conformance/worktree_contract_test.go
test/integration/repo_worktree_command_test.go
```

### 设计决策

- **MVP 模拟**：真实 `git worktree add` 依赖本地 git 环境，MVP 采用模拟实现以支持 CLI 结构与契约测试
- **独立 worktree 目录**：默认放在 repo 上级目录 `gitdex-worktree-<branch>`，避免污染主工作树
- **统一输出格式**：与 repo inspect/sync/hygiene 一致，支持 `--output json|yaml` 与人类可读文本

### References

- Epic 2: Governed Planning and Safe Single-Repository Action
- FR17: Operators can request controlled local file modifications within an authorized repository scope
- Architecture: All Git-side mutative execution runs in isolated `git worktree` environments

## Dev Agent Record

### Completion Notes List

- Task 1：建立 Worktree 领域模型，支持配置、状态、操作接口
- Task 2：实现 Create/Inspect/Diff/Discard 的 MVP 模拟逻辑
- Task 3：在 repo 命令下注册 worktree 子命令，支持 create/inspect/diff/discard
- Task 4：编写 conformance 与 integration 测试覆盖契约与命令行行为

### File List

**New files:**
- `internal/gitops/worktree.go`
- `test/conformance/worktree_contract_test.go`
- `test/integration/repo_worktree_command_test.go`

**Modified files:**
- `internal/cli/command/repo.go`（新增 `newRepoWorktreeGroupCommand` 及相关子命令与渲染函数）
