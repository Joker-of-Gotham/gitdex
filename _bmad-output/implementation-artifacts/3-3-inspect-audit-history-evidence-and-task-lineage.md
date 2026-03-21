# Story 3.3: Inspect Audit History, Evidence, and Task Lineage (FR33, FR34)

Status: done

## Story

As an operator or administrator,
I want to inspect the full audit trail and lineage for governed actions,
So that I can explain who triggered what, under which policy, with which evidence, and with what result.

## Acceptance Criteria

1. **Given** a governed task or action in Gitdex **When** an operator opens the audit and evidence view **Then** Gitdex shows trigger source, plan, policy result, approvals, lifecycle history, outcome state, and linked evidence **And** the operator can navigate from a task to related plans, reports, and handoff artifacts without leaving the terminal **And** the audit history remains append-only and queryable over time

## Tasks / Subtasks

- [x] Task 1: 定义 Audit 领域模型与事件类型 (AC: #1, #3)
  - [x] 1.1 在 `internal/audit/ledger.go` 中定义 `EventType`（plan_created、plan_approved、plan_rejected、task_started、task_succeeded、task_failed、policy_evaluated、emergency_control、identity_registered 等）
  - [x] 1.2 定义 `AuditEntry`（EntryID、CorrelationID、TaskID、PlanID、EventType、Actor、Action、Target、PolicyResult、EvidenceRefs、Timestamp）
  - [x] 1.3 定义 `AuditFilter` 支持按 EntryID、EventType、TaskID、CorrelationID、时间范围查询

- [x] Task 2: 实现 Append-Only Audit Ledger (AC: #3)
  - [x] 2.1 定义 `AuditLedger` 接口：Append、Query、GetByCorrelation、GetByTask、GetByEntryID
  - [x] 2.2 实现 `MemoryAuditLedger`，entries 仅追加，不支持修改或删除
  - [x] 2.3 Append 时自动生成 EntryID（若为空）、Timestamp（若为空），保证可查询与追溯

- [x] Task 3: 注册 CLI Audit 命令 (AC: #1, #2)
  - [x] 3.1 在 `internal/cli/command/audit.go` 中实现 `audit` 命令组
  - [x] 3.2 `audit log [--limit N]`：展示最近审计条目，默认 limit=20，按时间倒序
  - [x] 3.3 `audit show <entry_id>`：展示单条条目详情（trigger、plan、policy result、evidence refs 等）
  - [x] 3.4 `audit trace <correlation_id>`：按 correlation_id 展示完整 lineage，支持从任务导航到相关 plans/reports
  - [x] 3.5 支持 JSON/YAML 结构化输出与人类可读文本

- [x] Task 4: 编写单元测试与契约/集成测试 (AC: 全部)
  - [x] 4.1 `internal/audit/ledger_test.go`：Append、Query、GetByCorrelation、GetByTask、GetByEntryID、时间过滤、Append nil 失败
  - [x] 4.2 `test/integration/audit_command_test.go`：命令注册、log/show/trace 行为，支持 SetAuditLedgerForTest 注入
  - [x] 4.3 `test/conformance/audit_contract_test.go`（若存在）：AuditEntry 字段契约与 JSON 结构

## Dev Notes

### 关键实现细节

- **AuditEntry**：包含触发来源（Actor、Action）、PlanID、TaskID、PolicyResult、EvidenceRefs、生命周期事件类型（EventType），满足 AC 中的 trigger、plan、policy result、approvals、lifecycle history、outcome state、linked evidence
- **EventType**：plan_created/approved/rejected、task_started/succeeded/failed、policy_evaluated、emergency_control、identity_registered 等
- **MemoryAuditLedger**：使用 slice 追加 + byID map 索引；Query 支持 EntryID、EventType、TaskID、CorrelationID、FromTime、ToTime 过滤
- **audit trace**：通过 GetByCorrelation 获取同一 correlation 下的完整事件链，支持从任务导航到相关 plans、reports
- **SetAuditLedgerForTest**：供集成测试注入自定义 ledger，验证命令行为

### 文件结构

```
internal/audit/ledger.go           # AuditEntry、AuditLedger、MemoryAuditLedger
internal/cli/command/audit.go      # audit 命令组
internal/audit/ledger_test.go
test/integration/audit_command_test.go
test/conformance/audit_contract_test.go  # 若存在
```

### 设计决策

- **Append-only**：AuditLedger 仅支持 Append，无 Update/Delete，满足审计完整性要求
- **CorrelationID 与 TaskID**：支持按业务关联（correlation）或任务（task）追溯，便于导航
- **EvidenceRefs**：条目可携带证据引用列表，支持链接到 plans、reports、handoff artifacts

### References

- Epic 3: Governance, Policy, Audit, and Emergency Control
- FR33: Gitdex can record complete audit trails for governed actions, approvals, policy evaluations, and security-relevant events
- FR34: Operators and administrators can inspect audit history, evidence, and task lineage
- Architecture: Audit system must be append-only; audit ledger pairs with evidence store

## Dev Agent Record

### Completion Notes List

- Task 1：定义 AuditEntry、EventType、AuditFilter，覆盖 trigger、plan、policy、approvals、lifecycle、evidence
- Task 2：实现 AuditLedger 接口与 MemoryAuditLedger，append-only，支持 Query/GetByCorrelation/GetByTask
- Task 3：实现 audit log/show/trace 子命令，支持 limit 与 correlation lineage 导航
- Task 4：编写单元测试与 integration 测试，支持 ledger 注入

### File List

**New files:**
- `internal/audit/ledger.go`
- `internal/audit/ledger_test.go`
- `internal/cli/command/audit.go`
- `test/integration/audit_command_test.go`

**Modified files:**
- 根命令注册 `audit` 命令组
