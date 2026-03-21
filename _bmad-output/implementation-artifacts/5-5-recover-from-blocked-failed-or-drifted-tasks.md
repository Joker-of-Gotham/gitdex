# Story 5.5: Recover from Blocked, Failed, or Drifted Tasks (FR27)

Status: done

## Story

As an operator,
I want Gitdex to move failed, blocked, or drifted tasks into governed recovery states,
so that I can see the failure reason, latest successful step, and next recovery path with traceability.

## Acceptance Criteria

1. **Given** a task that fails, blocks, or diverges from expected state **When** Gitdex evaluates the outcome **Then** it moves the task into a supported retry, reconciliation, quarantine, or equivalent governed recovery state **And** the operator can see the failure reason, latest successful step, and next recovery path **And** recovery behavior preserves traceability rather than starting a new unlinked task

## Tasks / Subtasks

- [x] Task 1: 定义 Recovery 领域模型 (AC: #1)
  - [x] 1.1 在 `internal/autonomy/recovery.go` 中定义 `RecoveryStrategy`（retry、rollback、escalate、skip、manual_intervention）
  - [x] 1.2 定义 `RecoveryRequest`（TaskID、Strategy、MaxRetries、Reason、Actor）
  - [x] 1.3 定义 `RecoveryResult`（Request、Success、Attempts、FinalStatus、Message、RecoveredAt）
  - [x] 1.4 定义 `RecoveryAssessment`（TaskID、FailureType、RecommendedStrategy、RiskLevel、Details）

- [x] Task 2: 实现 RecoveryEngine (AC: #1, #2, #3)
  - [x] 2.1 定义 `RecoveryEngine` 接口：Assess(ctx, taskID)、Execute(ctx, request)
  - [x] 2.2 实现 `DefaultRecoveryEngine`：Assess 根据 task state 推荐策略（blocked→retry、failed→rollback、drifted→manual_intervention）
  - [x] 2.3 Execute 执行策略并写入 history；History(taskID) 按 task 过滤，保证 traceability

- [x] Task 3: 注册 CLI recovery 命令 (AC: 全部)
  - [x] 3.1 `recovery assess <task_id>`：评估恢复选项
  - [x] 3.2 `recovery execute <task_id> [--strategy retry|rollback|escalate|skip|manual_intervention] [--max-retries N] [--reason <reason>]`：执行恢复
  - [x] 3.3 `recovery history [--task_id <id>]`：查看恢复历史
  - [x] 3.4 支持 JSON/YAML 结构化输出与文本渲染

- [x] Task 4: 编写单元测试与集成测试 (AC: 全部)
  - [x] 4.1 `internal/autonomy/recovery_test.go`：Assess、Execute、History（含按 task 过滤）
  - [x] 4.2 `test/integration/recovery_command_test.go`：assess/execute/history 子命令注册、必填 task_id、执行流程

## Dev Notes

### 关键实现细节

- **Assess**：根据 task state（blocked/failed/drifted）推荐策略与风险等级；blocked→retry/low；failed→rollback/medium；drifted→manual_intervention/high
- **Execute**：validRecoveryStrategies 校验；FinalStatus 映射 retry→retrying、rollback→rolled_back、escalate→escalated、skip→skipped、manual_intervention→manual
- **History**：记录每次 Execute 结果；History("") 返回全部，History(taskID) 返回该 task 的恢复历史，保证不启动新 unlinked task

### 文件结构

```
internal/autonomy/recovery.go        # RecoveryStrategy、RecoveryRequest、RecoveryResult、RecoveryAssessment、RecoveryEngine、DefaultRecoveryEngine
internal/cli/command/recovery.go     # recovery assess/execute/history 命令组
internal/autonomy/recovery_test.go
test/integration/recovery_command_test.go
```

### 设计决策

- **Governed recovery**：retry、rollback、escalate、skip、manual_intervention 五种策略覆盖主流场景；quarantine 可由 manual_intervention 或 escalate 表示
- **Traceability**：RecoveryResult 含 Request（含 TaskID）；History 按 TaskID 过滤，恢复与原 task 关联
- **DefaultRecoveryEngine**：内存 taskStates + history slice；Assess 模拟推荐逻辑，实际 task state 由运行时层提供

### References

- Epic 5: Autonomy and Background Operations
- FR27: Failed/blocked/drifted tasks move to governed recovery; operator sees reason and path; traceability preserved

## Dev Agent Record

### Completion Notes List

- Task 1：定义 RecoveryStrategy、RecoveryRequest、RecoveryResult、RecoveryAssessment
- Task 2：实现 RecoveryEngine 接口与 DefaultRecoveryEngine
- Task 3：实现 recovery assess/execute/history 子命令
- Task 4：编写单元测试与 integration 测试

### File List

**New files:**
- `internal/autonomy/recovery.go`
- `internal/autonomy/recovery_test.go`
- `internal/cli/command/recovery.go`
- `test/integration/recovery_command_test.go`

**Modified files:**
- 根命令注册 `recovery` 命令组
