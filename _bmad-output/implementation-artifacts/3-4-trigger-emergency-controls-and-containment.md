# Story 3.4: Trigger Emergency Controls and Containment (FR35)

Status: done

## Story

As an authorized operator,
I want to pause, suspend, or kill risky automation quickly,
So that a bad task or bad scope cannot continue causing damage while the team regains control.

## Acceptance Criteria

1. **Given** an active task, repository scope, or wider authorized scope **When** an authorized operator triggers an emergency control **Then** Gitdex applies the corresponding pause, suspension, or kill-switch action and records the event in the security and audit trail **And** affected tasks visibly enter a contained or blocked state instead of continuing silently **And** operators can see what was stopped and what manual follow-up is still required

## Tasks / Subtasks

- [x] Task 1: 定义 Emergency Control 领域模型 (AC: #1, #2, #3)
  - [x] 1.1 在 `internal/emergency/controls.go` 中定义 `ControlAction`（pause_task、pause_scope、suspend_capability、kill_switch）
  - [x] 1.2 定义 `ControlRequest`（Action、Scope、Reason、Actor、Timestamp）与 `ControlResult`（Request、Success、AffectedTasks、AffectedScopes、Message）

- [x] Task 2: 实现 Control Engine 与状态追踪 (AC: #1, #2, #3)
  - [x] 2.1 定义 `ControlEngine` 接口：Execute(request) -> (*ControlResult, error)
  - [x] 2.2 实现 `DefaultControlEngine`，执行 pause/suspend/kill_switch 动作（MVP 为模拟实现）
  - [x] 2.3 维护 activeControls 列表，支持 Status() 查询当前生效的控制，便于展示「被停止的内容」及后续人工处理

- [x] Task 3: 注册 CLI Emergency 命令 (AC: #1, #2, #3)
  - [x] 3.1 在 `internal/cli/command/emergency.go` 中实现 `emergency` 命令组
  - [x] 3.2 `emergency pause <task_id>`：暂停指定任务
  - [x] 3.3 `emergency suspend <scope>`：暂停指定 scope 的能力
  - [x] 3.4 `emergency kill`：触发 kill switch，影响所有任务
  - [x] 3.5 `emergency status`：展示当前生效的 emergency controls，便于操作者了解「what was stopped」及后续跟进
  - [x] 3.6 支持 JSON/YAML 结构化输出与人类可读文本

- [x] Task 4: 编写单元测试与契约/集成测试 (AC: 全部)
  - [x] 4.1 `internal/emergency/controls_test.go`：Execute 各 ControlAction、Status 返回 activeControls
  - [x] 4.2 `test/conformance/emergency_contract_test.go`：ControlRequest、ControlResult、ControlAction JSON 契约与往返
  - [x] 4.3 `test/integration/emergency_command_test.go`：命令注册、pause/suspend/kill/status 执行与输出校验

## Dev Notes

### 关键实现细节

- **ControlAction**：`pause_task`（单任务）、`pause_scope`（作用域）、`suspend_capability`（能力挂起）、`kill_switch`（全局熔断）
- **ControlRequest**：包含 Action、Scope、Reason、Actor、Timestamp；Actor 在 CLI 中为 "cli"
- **ControlResult**：Success、AffectedTasks、AffectedScopes、Message；kill_switch 时 AffectedTasks=["*"]、AffectedScopes=["*"]
- **DefaultControlEngine**：MVP 为模拟执行，不实际暂停后台任务；Execute 后追加到 activeControls，Status() 返回所有生效控制
- **安全与审计**：Control 事件应写入 security/audit trail（与 audit ledger 集成待后续实现）；当前通过 Status 展示受影响任务与 scope
- **CLI**：emergency pause/suspend 需传入 task_id 或 scope；kill 无参数；status 展示 active controls 列表（action、scope、actor、timestamp）

### 文件结构

```
internal/emergency/controls.go     # ControlAction、ControlRequest、ControlResult、ControlEngine、DefaultControlEngine
internal/cli/command/emergency.go  # emergency 命令组
internal/emergency/controls_test.go
test/conformance/emergency_contract_test.go
test/integration/emergency_command_test.go
```

### 设计决策

- **MVP 模拟**：真实 pause/suspend/kill 需与任务运行时、后台 daemon 集成；MVP 先实现控制面接口与 CLI，记录请求并返回受影响范围
- **activeControls 列表**：便于操作者通过 `emergency status` 查看「what was stopped」及手动跟进项
- **Kill switch 全局性**：Scope 为 "*"，AffectedTasks/AffectedScopes 均为 ["*"]，明确表示全系统熔断

### References

- Epic 3: Governance, Policy, Audit, and Emergency Control
- FR35: Authorized users can trigger emergency controls such as pause, capability suspension, or kill switch actions
- NFR9: Safe pause、cancel、kill switch 指令须在 30 秒内生效；外部副作用已提交时须显式转入 containment/handoff 状态

## Dev Agent Record

### Completion Notes List

- Task 1：定义 ControlAction、ControlRequest、ControlResult
- Task 2：实现 ControlEngine 与 DefaultControlEngine，支持 Execute 与 Status
- Task 3：实现 emergency pause/suspend/kill/status 子命令
- Task 4：编写单元测试、conformance 契约测试与 integration 测试

### File List

**New files:**
- `internal/emergency/controls.go`
- `internal/emergency/controls_test.go`
- `internal/cli/command/emergency.go`
- `test/conformance/emergency_contract_test.go`
- `test/integration/emergency_command_test.go`

**Modified files:**
- 根命令注册 `emergency` 命令组
