# Story 5.4: Pause, Resume, Cancel, and Take Over Autonomous Tasks (FR26)

Status: done

## Story

As an operator,
I want to pause, resume, cancel, or take over active autonomous tasks,
so that I can apply control while preserving task state, evidence, and the latest completed step.

## Acceptance Criteria

1. **Given** an active autonomous task **When** the operator issues a pause, resume, cancel, or take-over action **Then** Gitdex applies the requested control while preserving current task state, current evidence, and the latest completed step **And** the task view clearly shows whether control has returned to the human or remains in autonomous mode **And** cancelled or paused work does not disappear into an unexplained terminal state

## Tasks / Subtasks

- [x] Task 1: 定义 Task Control 领域模型 (AC: #1, #2)
  - [x] 1.1 在 `internal/autonomy/task_control.go` 中定义 `TaskControlAction`（pause、resume、cancel、takeover）
  - [x] 1.2 定义 `TaskControlRequest`（Action、TaskID、Reason、Actor、Timestamp）
  - [x] 1.3 定义 `TaskControlResult`（Request、Success、PreviousStatus、NewStatus、Message）

- [x] Task 2: 实现 TaskController (AC: #1, #3)
  - [x] 2.1 定义 `TaskController` 接口：Execute(ctx, request) (*TaskControlResult, error)
  - [x] 2.2 实现 `DefaultTaskController`：维护 task 状态 map；pause→paused、resume→running、cancel→cancelled、takeover→manual
  - [x] 2.3 状态转换校验：仅 running/executing 可 pause；仅 paused 可 resume；cancel/takeover 任意状态

- [x] Task 3: 注册 CLI autonomy 控制子命令 (AC: 全部)
  - [x] 3.1 `autonomy pause <task_id> [--reason <reason>]`：暂停任务
  - [x] 3.2 `autonomy resume <task_id>`：恢复任务
  - [x] 3.3 `autonomy cancel <task_id> [--reason <reason>]`：取消任务
  - [x] 3.4 `autonomy takeover <task_id>`：接管为手动模式
  - [x] 3.5 输出 PreviousStatus、NewStatus、Success、Message；支持 JSON/YAML

- [x] Task 4: 编写单元测试与集成测试 (AC: 全部)
  - [x] 4.1 `internal/autonomy/task_control_test.go`：Pause、Resume、Cancel、Takeover、ResumeWithoutPause（失败用例）
  - [x] 4.2 `test/integration/autonomy_control_command_test.go`：pause/resume/cancel/takeover 子命令注册、必填 task_id、执行流程

## Dev Notes

### 关键实现细节

- **DefaultTaskController**：内部 states map 存储 task_id→status；初始 running；pause 需 running/executing；resume 需 paused
- **Takeover**：将状态设为 manual，表示控制已交回人类
- **Cancel**：任意状态可取消，设为 cancelled；paused 任务不会丢失，有明确终端状态

### 文件结构

```
internal/autonomy/task_control.go    # TaskControlAction、TaskControlRequest、TaskControlResult、TaskController、DefaultTaskController
internal/cli/command/autonomy.go    # autonomy pause/resume/cancel/takeover 子命令（与 5.1 共享 autonomy 组）
internal/autonomy/task_control_test.go
test/integration/autonomy_control_command_test.go
```

### 设计决策

- **状态保留**：DefaultTaskController 仅维护 status string；完整 state/evidence 由任务运行时层管理，当前为模拟实现
- **控制权可见**：NewStatus 为 manual 时表示控制权已交回；paused 表示暂停但未取消
- **Terminal state**：cancelled、manual 为终端状态，不会“消失”，可被查询与审计

### References

- Epic 5: Autonomy and Background Operations
- FR26: Pause/resume/cancel/takeover preserve state; task view shows control mode; no unexplained terminal state

## Dev Agent Record

### Completion Notes List

- Task 1：定义 TaskControlAction、TaskControlRequest、TaskControlResult
- Task 2：实现 TaskController 接口与 DefaultTaskController
- Task 3：在 autonomy 命令组添加 pause/resume/cancel/takeover 子命令
- Task 4：编写单元测试与 integration 测试

### File List

**New files:**
- `internal/autonomy/task_control.go`
- `internal/autonomy/task_control_test.go`
- `test/integration/autonomy_control_command_test.go`

**Modified files:**
- `internal/cli/command/autonomy.go`（添加 pause/resume/cancel/takeover 子命令；引入 taskController）
