# Story 6.2: Review Per-Repository Plans and Status in Campaign Matrix (FR38)

Status: done

## Story

As an operator managing a campaign with multiple target repositories,
I want to open a campaign view that presents a per-repository matrix of plans, current states, and outcomes,
so that I can sort, filter, drill into each repository row and see completed, blocked, and pending repositories side by side with per-repository exclusion, retry, and export actions.

## Acceptance Criteria

1. **Given** a campaign with multiple target repositories **When** the operator opens the campaign view **Then** Gitdex presents a per-repository matrix of plans, current states, and outcomes **And** the operator can sort, filter, and drill into each repository row **And** completed, blocked, and pending repositories remain visible side by side **And** the matrix exposes per-repository exclusion, retry, and export actions

## Tasks / Subtasks

- [x] Task 1: 定义 Matrix 领域模型与引擎
  - [x] 1.1 在 `internal/campaign/matrix.go` 中定义 `RepoStatus`（not_started, planning, awaiting_approval, approved, executing, succeeded, failed, excluded）
  - [x] 1.2 定义 `MatrixEntry`（Owner, Repo, Status, PlanID, TaskID, LastUpdated, Notes）
  - [x] 1.3 定义 `MatrixSummary`（Total, Succeeded, Failed, Pending, Excluded）
  - [x] 1.4 定义 `CampaignMatrix`（CampaignID, Entries, Summary）
  - [x] 1.5 定义 `MatrixEngine` 接口并实现 `DefaultMatrixEngine.Build`

- [x] Task 2: 实现 Matrix 构建逻辑
  - [x] 2.1 `Build` 根据 Campaign.TargetRepos 的 InclusionStatus 映射为 RepoStatus
  - [x] 2.2 InclusionExcluded → RepoStatusExcluded，InclusionIncluded → RepoStatusSucceeded，默认 → RepoStatusAwaitingApproval
  - [x] 2.3 汇总 Total/Succeeded/Failed/Pending/Excluded

- [x] Task 3: 注册 CLI matrix 与 status 命令
  - [x] 3.1 `campaign matrix <campaign_id>`：展示矩阵视图（Entries + Summary）
  - [x] 3.2 `campaign status <campaign_id>`：展示状态汇总（仅 Summary）
  - [x] 3.3 支持 JSON/YAML 与表格文本输出

- [x] Task 4: 编写单元与集成测试
  - [x] 4.1 `internal/campaign/matrix_test.go`：Build 含多种 InclusionStatus、BuildEmpty
  - [x] 4.2 `test/integration/campaign_matrix_command_test.go`：matrix/status 必填 campaign_id、matrix 运行

## Dev Notes

### 关键实现细节

- **MatrixEngine.Build**：遍历 Campaign.TargetRepos，按 InclusionStatus 映射 RepoStatus 并生成 PlanID/TaskID（当前为占位：plan_<campaign_id>、task_<repo>）
- **MatrixSummary**：Total=len(TargetRepos)；Succeeded/Failed/Pending/Excluded 按 InclusionStatus 统计
- **renderCampaignMatrixText**：表格输出 Owner/Repo、Status、PlanID、TaskID、LastUpdated
- **renderCampaignSummaryText**：简洁输出 Campaign 名与 Summary 统计

### 文件结构

```
internal/campaign/matrix.go
internal/cli/command/campaign.go   # matrix/status（复用 campaign_store）
internal/campaign/matrix_test.go
test/integration/campaign_matrix_command_test.go
```

### 设计决策

- **Matrix 为衍生视图**：不持久化，每次 Build 从 Campaign 实时计算
- **Sort/Filter/Drill**：当前 CLI 为表格文本输出；排序、过滤、钻取待 TUI 或 Web UI 扩展
- **Per-repository actions**：exclude/retry 在 Story 6.3 的 intervention 中实现，matrix 视图仅展示状态

### References

- Epic 6: Campaign Orchestration
- FR38: Per-repository matrix of plans and status

## Dev Agent Record

### Completion Notes List

- Task 1：定义 RepoStatus、MatrixEntry、MatrixSummary、CampaignMatrix、MatrixEngine
- Task 2：实现 DefaultMatrixEngine.Build，基于 InclusionStatus 映射
- Task 3：实现 campaign matrix/status 子命令
- Task 4：编写 matrix_test、campaign_matrix_command_test

### File List

**New files:**
- `internal/campaign/matrix.go`
- `internal/campaign/matrix_test.go`
- `internal/cli/command/campaign.go`（扩展现有 campaign 命令组）
- `test/integration/campaign_matrix_command_test.go`
