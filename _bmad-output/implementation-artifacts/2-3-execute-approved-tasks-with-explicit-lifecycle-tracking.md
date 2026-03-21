# Story 2.3: Execute Approved Tasks with Explicit Lifecycle Tracking

Status: done

## Story

As a maintainer,
I want approved single-repository tasks to run through explicit lifecycle states,
so that I can see what is happening, what already completed, and where intervention would occur if the task fails.

## Acceptance Criteria

1. **Given** an approved task for a supported single-repository action **When** execution begins **Then** Gitdex moves the task through explicit lifecycle states from queueing to execution to reconciliation to terminal outcome.

2. **And** each state transition is tied to a stable task identifier and correlation identifier.

3. **And** operators can inspect the latest completed step and current executing step from the terminal.

4. **And** long-running execution shows stage-aware progress after 3s and whether the operator can safely leave after 10s.

## Tasks / Subtasks

- [ ] Task 1: 任务生命周期数据模型 (AC: #1, #2)
  - [ ] 1.1 新增 `internal/orchestrator/task.go`，定义 `Task` 结构体：`task_id`, `correlation_id`, `plan_id`, `status`, `current_step`, `completed_steps`, `started_at`, `updated_at`, `finished_at`
  - [ ] 1.2 定义 `TaskStatus` 枚举：`queued`, `executing`, `reconciling`, `succeeded`, `failed_handoff_pending`, `quarantined`, `cancelled`, `failed_with_handoff_complete`
  - [ ] 1.3 定义 `TaskEvent` 结构体用于记录状态转换事件：`event_id`, `task_id`, `from_status`, `to_status`, `step_sequence`, `message`, `timestamp`
  - [ ] 1.4 定义 `StepResult` 结构体：`sequence`, `action`, `status`(pending/running/succeeded/failed/skipped), `started_at`, `finished_at`, `output`, `error_message`
  - [ ] 1.5 ID 生成：`corr_` 前缀 + hex

- [ ] Task 2: 任务存储层 (AC: #2)
  - [ ] 2.1 新增 `internal/orchestrator/store.go`，定义 `TaskStore` 接口：`SaveTask`, `GetTask`, `ListTasks`, `UpdateTask`, `AppendEvent`, `GetEvents`
  - [ ] 2.2 实现 `MemoryTaskStore`（与 PlanStore 对齐的内存实现）
  - [ ] 2.3 保证 correlation_id 可查询

- [ ] Task 3: 任务执行器 (AC: #1, #3, #4)
  - [ ] 3.1 新增 `internal/orchestrator/executor.go`，定义 `Executor` 结构体
  - [ ] 3.2 `Execute(ctx, task *Task) error`：逐步执行 plan steps，更新 step results 和 task status
  - [ ] 3.3 状态转换顺序：`queued` → `executing` → 逐步执行 → `reconciling` → `succeeded`/`failed_handoff_pending`
  - [ ] 3.4 每步执行前后记录 `TaskEvent`
  - [ ] 3.5 失败时进入 `failed_handoff_pending` 并记录失败步骤
  - [ ] 3.6 模拟执行器（MVP 不连接真实 Git/GitHub 后端，使用 sleep + 日志模拟）

- [ ] Task 4: CLI `gitdex task` 命令组 (AC: #1, #3)
  - [ ] 4.1 `gitdex task start <plan_id>` — 从已批准计划创建任务并开始执行
  - [ ] 4.2 `gitdex task status <task_id>` — 查看任务状态、当前步骤、已完成步骤
  - [ ] 4.3 `gitdex task list` — 列出所有任务
  - [ ] 4.4 支持 text/JSON/YAML 输出格式
  - [ ] 4.5 status 视图渲染包含：任务状态、当前执行步骤、已完成步骤列表、失败原因（如有）
  - [ ] 4.6 在 `root.go` 中注册 task 命令组

- [ ] Task 5: 进度感知 (AC: #4)
  - [ ] 5.1 在 `task status` 输出中显示任务运行时长
  - [ ] 5.2 超过 3s 时显示当前阶段和步骤信息
  - [ ] 5.3 超过 10s 时显示 "safe to leave" 提示（后台可继续）

- [ ] Task 6: 全面测试 (AC: #1-#4)
  - [ ] 6.1 `internal/orchestrator/task_test.go` — 任务模型和状态枚举
  - [ ] 6.2 `internal/orchestrator/store_test.go` — 存储 CRUD 和事件追加
  - [ ] 6.3 `internal/orchestrator/executor_test.go` — 执行器逻辑（正常路径和失败路径）
  - [ ] 6.4 `test/integration/task_command_test.go` — CLI 命令注册
  - [ ] 6.5 `test/conformance/task_lifecycle_contract_test.go` — 生命周期合约
  - [ ] 6.6 运行 `go test ./... -count=1` 全量通过 + `golangci-lint run ./...` 零错误

- [ ] Task 7: 收尾验证 (AC: #1-#4)
  - [ ] 7.1 验证范围：仅包含任务编排骨架，不包含真实 Git/GitHub 后端集成
  - [ ] 7.2 确认与 Story 2.1/2.2 的接口兼容（PlanStore, Plan, ApprovalRecord）
  - [ ] 7.3 更新 sprint-status.yaml

## Dev Notes

### 从前序 Story 学到的

**Story 2.2 关键经验：**
- SaveApproval 应在 Save/UpdateStatus 之前调用以保证一致性
- ExecutionMode 需要验证有效值
- renderReviewText 中 PolicyResult 可能为 nil，需要 guard
- gofmt 格式化对齐需要在每次修改后检查

**Story 2.1 关键经验：**
- MemoryPlanStore 不跨进程持久化（本 Story 同理）
- JSON/YAML 使用 snake_case

### 已有的可复用组件

| 组件 | 路径 | 复用方式 |
|------|------|---------|
| Plan Model | `internal/planning/plan.go` | Task 引用 plan_id |
| PlanStore | `internal/planning/store.go` | 查询计划状态 |
| Reviewer | `internal/planning/reviewer/reviewer.go` | 验证 plan 已 approved |
| Output Format | `internal/cli/output/format.go` | JSON/YAML 输出 |
| Root Command | `internal/cli/command/root.go` | 注册 task 命令 |

### 架构约束

1. **任务状态机**：queued → executing → reconciling → succeeded/failed_handoff_pending/quarantined/cancelled
2. **每次状态转换必须记录 TaskEvent**
3. **correlation_id 贯穿 plan 和 task**
4. **MVP 使用模拟执行器**，不连接真实 Git/GitHub 后端
5. **UX-DR16**：长时间操作在 3s 后显示阶段进度，10s 后显示是否可以安全离开

### 本 Story 新增的文件结构
```
internal/orchestrator/
├── task.go           # Task, TaskStatus, TaskEvent, StepResult
├── task_test.go
├── store.go          # TaskStore 接口 + MemoryTaskStore
├── store_test.go
├── executor.go       # Executor 模拟执行器
├── executor_test.go

internal/cli/command/
├── task.go           # task start/status/list 子命令

test/integration/
├── task_command_test.go

test/conformance/
├── task_lifecycle_contract_test.go
```

### References

- [Source: architecture.md §Task Lifecycle & State Machine]
- [Source: architecture.md §Recovery, Reconciliation, and Compensation]
- [Source: prd.md §FR10, FR11, FR13]
- [Source: ux-design-specification.md §UX-DR16]
- [Source: epics.md §Story 2.3]

## Dev Agent Record

### Agent Model Used

claude-4.6-opus-max-thinking

### Debug Log References

(to be filled during dev-story)

### Completion Notes List

(to be filled during dev-story)

### File List

(to be filled during dev-story)

## Change Log

| Date | Change | Author |
|------|--------|--------|
| 2026-03-18 | Story created | Agent |
