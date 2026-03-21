# Story 5.2: Monitor Authorized Repositories (FR24)

Status: done

## Story

As a repository owner,
I want Gitdex to monitor authorized repositories continuously or on an approved schedule,
so that supported maintenance or governance scenarios are evaluated and observations surface as tasks, summaries, or recommended next actions.

## Acceptance Criteria

1. **Given** a repository with supported background monitoring enabled **When** Gitdex runs continuously or on an approved schedule **Then** it evaluates only the explicitly supported maintenance or governance scenarios for that scope **And** the resulting observations are surfaced as tasks, summaries, or recommended next actions **And** disabled or unauthorized scenarios are not executed

## Tasks / Subtasks

- [x] Task 1: 定义 Monitor 领域模型 (AC: #1)
  - [x] 1.1 在 `internal/autonomy/monitor.go` 中定义 `MonitorConfig`（MonitorID、RepoOwner、RepoName、Interval、Checks、Enabled）
  - [x] 1.2 定义 `MonitorEvent`（EventID、MonitorID、RepoOwner、RepoName、CheckName、Status、Message、Timestamp）
  - [x] 1.3 定义 `MonitorEventFilter` 用于 ListEvents 筛选

- [x] Task 2: 实现 Monitor 存储 (AC: #1)
  - [x] 2.1 定义 `MonitorStore` 接口：SaveMonitorConfig、GetMonitorConfig、ListMonitorConfigs、AppendEvent、ListEvents、RemoveMonitorConfig
  - [x] 2.2 实现 `MemoryMonitorStore`，线程安全内存存储

- [x] Task 3: 注册 CLI monitor 命令 (AC: 全部)
  - [x] 3.1 `monitor add --repo owner/repo [--interval 5m]`：添加仓库监控
  - [x] 3.2 `monitor list`：列出监控配置
  - [x] 3.3 `monitor events [--repo owner/repo] [--limit N]`：列出最近事件
  - [x] 3.4 `monitor remove <monitor_id>`：移除监控
  - [x] 3.5 支持 JSON/YAML 结构化输出与文本表格

- [x] Task 4: 编写单元测试与集成测试 (AC: 全部)
  - [x] 4.1 `internal/autonomy/monitor_test.go`：SaveAndGet、ListMonitorConfigs、AppendAndListEvents、RemoveMonitorConfig
  - [x] 4.2 `test/integration/monitor_command_test.go`：命令注册、help、add 必填 repo、add+list 流程、list 空态

## Dev Notes

### 关键实现细节

- **MonitorConfig**：RepoOwner/RepoName 表示 scope；Interval 如 5m、1h 表示调度间隔；Checks 数组表示支持的检查场景；Enabled 控制是否执行
- **MonitorEvent**：Status 为 ok/warning/critical；按时间倒序 ListEvents（最近优先）
- **MemoryMonitorStore**：configs map + events slice；AppendEvent 时自动生成 EventID、Timestamp

### 文件结构

```
internal/autonomy/monitor.go           # MonitorConfig、MonitorEvent、MonitorStore、MemoryMonitorStore
internal/cli/command/monitor.go        # monitor add/list/events/remove 命令组
internal/autonomy/monitor_test.go
test/integration/monitor_command_test.go
```

### 设计决策

- **Scope**：通过 RepoOwner/RepoName 限定监控范围；Checks 为空时表示未配置具体 scenario，实际执行逻辑由后续调度器集成
- **事件存储**：内存存储，按 MonitorID/RepoOwner/RepoName 过滤；Limit 默认 100，支持分页
- **禁用场景**：MonitorConfig.Enabled=false 的监控不会被调度执行（存储层支持，执行层待后续集成）

### References

- Epic 5: Autonomy and Background Operations
- FR24: Gitdex evaluates only supported maintenance or governance scenarios; disabled/unauthorized scenarios are not executed

## Dev Agent Record

### Completion Notes List

- Task 1：定义 MonitorConfig、MonitorEvent、MonitorEventFilter
- Task 2：实现 MonitorStore 接口与 MemoryMonitorStore
- Task 3：实现 monitor add/list/events/remove 子命令
- Task 4：编写单元测试与 integration 测试

### File List

**New files:**
- `internal/autonomy/monitor.go`
- `internal/autonomy/monitor_test.go`
- `internal/cli/command/monitor.go`
- `test/integration/monitor_command_test.go`

**Modified files:**
- 根命令注册 `monitor` 命令组
