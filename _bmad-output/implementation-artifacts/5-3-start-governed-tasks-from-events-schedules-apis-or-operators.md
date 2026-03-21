# Story 5.3: Start Governed Tasks from Events, Schedules, APIs, or Operators (FR25)

Status: done

## Story

As an operator or system,
I want Gitdex to normalize webhooks, schedules, API requests, or operator actions into governed task envelopes,
so that the originating trigger source remains visible in task lineage and audit history, and webhook ingestion is asynchronous and replay-safe.

## Acceptance Criteria

1. **Given** a supported webhook, schedule, API request, or operator action **When** it triggers Gitdex work **Then** Gitdex normalizes that trigger into the same governed task envelope **And** webhook ingestion is asynchronous and replay-safe **And** the originating trigger source remains visible in task lineage and audit history

## Tasks / Subtasks

- [x] Task 1: 定义 Trigger 领域模型 (AC: #1, #3)
  - [x] 1.1 在 `internal/autonomy/trigger.go` 中定义 `TriggerType`（event、schedule、api、operator）
  - [x] 1.2 定义 `TriggerConfig`（TriggerID、TriggerType、Name、Source、Pattern、ActionTemplate、Enabled、CreatedAt）
  - [x] 1.3 定义 `TriggerEvent`（EventID、TriggerID、TriggerType、SourceEvent、ResultingTaskID、Timestamp）用于 lineage/audit

- [x] Task 2: 实现 Trigger 存储 (AC: #1, #3)
  - [x] 2.1 定义 `TriggerStore` 接口：SaveTrigger、GetTrigger、ListTriggers、EnableTrigger、DisableTrigger、AppendTriggerEvent、ListTriggerEvents
  - [x] 2.2 实现 `MemoryTriggerStore`，configs map + events slice；AppendTriggerEvent 记录 trigger→task 映射

- [x] Task 3: 注册 CLI trigger 命令 (AC: 全部)
  - [x] 3.1 `trigger add --type <event|schedule|api|operator> --name <name> [--pattern cron] [--action template] [--source id]`：添加触发器
  - [x] 3.2 `trigger list`：列出触发器（含 Enabled 状态）
  - [x] 3.3 `trigger enable <trigger_id>` / `trigger disable <trigger_id>`：启用/禁用
  - [x] 3.4 `trigger events [--trigger id] [--limit N]`：显示触发历史（含 SourceEvent、ResultingTaskID）
  - [x] 3.5 支持 JSON/YAML 结构化输出

- [x] Task 4: 编写单元测试与集成测试 (AC: 全部)
  - [x] 4.1 `internal/autonomy/trigger_test.go`：SaveAndGet、EnableDisable、AppendAndListEvents
  - [x] 4.2 `test/integration/trigger_command_test.go`：add/list/enable/disable/events 子命令注册、help、add 必填 type/name、add 流程、list 空态

## Dev Notes

### 关键实现细节

- **TriggerType**：event（webhook）、schedule（cron）、api（API 调用）、operator（人工操作）四种来源
- **TriggerConfig**：Pattern 对 schedule 为 cron 表达式；ActionTemplate 为动作模板；Source 为源标识（如 webhook URL 或 API 路径）
- **TriggerEvent**：每次触发产生事件，记录 SourceEvent、ResultingTaskID，实现 lineage 与 audit
- **MemoryTriggerStore**：ListTriggerEvents 按 triggerID 过滤、时间倒序；limit 默认 50

### 文件结构

```
internal/autonomy/trigger.go         # TriggerType、TriggerConfig、TriggerEvent、TriggerStore、MemoryTriggerStore
internal/cli/command/trigger.go     # trigger add/list/enable/disable/events 命令组
internal/autonomy/trigger_test.go
test/integration/trigger_command_test.go
```

### 设计决策

- **Governed task envelope**：TriggerEvent 将 trigger 与 resulting task 关联，所有来源归一为相同事件结构
- **Replay-safe**：AppendTriggerEvent 追加写入；EventID 唯一；实际 webhook 去重由上层 webhook 接收器实现
- **Source 可见**：TriggerEvent.SourceEvent、TriggerConfig.Source 保留触发来源，供 lineage/audit 使用

### References

- Epic 5: Autonomy and Background Operations
- FR25: Webhook/schedule/API/operator triggers normalize to governed task envelope; trigger source visible in lineage/audit

## Dev Agent Record

### Completion Notes List

- Task 1：定义 TriggerType、TriggerConfig、TriggerEvent
- Task 2：实现 TriggerStore 接口与 MemoryTriggerStore
- Task 3：实现 trigger add/list/enable/disable/events 子命令
- Task 4：编写单元测试与 integration 测试

### File List

**New files:**
- `internal/autonomy/trigger.go`
- `internal/autonomy/trigger_test.go`
- `internal/cli/command/trigger.go`
- `test/integration/trigger_command_test.go`

**Modified files:**
- 根命令注册 `trigger` 命令组
