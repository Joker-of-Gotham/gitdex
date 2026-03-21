# Story 2.2: Review, Approve, Reject, Edit, or Defer a Plan

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a repository operator,
I want to review and control a plan before it runs,
so that governed actions only proceed when I understand and accept the proposed change.

## Acceptance Criteria

1. **Given** a structured plan awaiting operator review **When** the operator inspects the plan **Then** they can approve, reject, edit, or defer it from within the terminal.

2. **And** the review surface includes linked evidence, current blockers, and the next actionable path.

3. **And** the review result is recorded as part of the task's traceable lifecycle.

4. **And** supported tasks can specify an execution mode (observe, recommend, dry-run, execute) that constrains what happens after approval.

## Tasks / Subtasks

- [ ] Task 1: 审批数据模型 (AC: #1, #3)
  - [ ] 1.1 在 `internal/planning/plan.go` 新增 `ApprovalAction` 枚举：`approve`, `reject`, `edit`, `defer`
  - [ ] 1.2 新增 `ApprovalRecord` 结构体：`record_id`, `plan_id`, `action`, `actor`, `reason`, `previous_status`, `new_status`, `created_at`
  - [ ] 1.3 新增 `ExecutionMode` 枚举：`observe`, `recommend`, `dry_run`, `execute`
  - [ ] 1.4 在 `Plan` 结构体中新增 `ExecutionMode` 和 `DeferredUntil` 字段
  - [ ] 1.5 所有新结构体/字段使用 `snake_case` JSON/YAML 标签

- [ ] Task 2: 审批存储层 (AC: #3)
  - [ ] 2.1 在 `PlanStore` 接口新增 `SaveApproval(record *ApprovalRecord) error`
  - [ ] 2.2 在 `PlanStore` 接口新增 `GetApprovals(planID string) ([]*ApprovalRecord, error)`
  - [ ] 2.3 在 `MemoryPlanStore` 实现上述方法，使用 `sync.RWMutex` 保护
  - [ ] 2.4 审批记录 ID 生成使用 `approval_` 前缀 + hex

- [ ] Task 3: 状态转换验证 (AC: #1, #2)
  - [ ] 3.1 新增 `internal/planning/transitions.go`，定义合法状态转换规则
  - [ ] 3.2 `review_required` → `approved` / `blocked` / `draft` (defer) / `review_required` (edit) 是合法转换
  - [ ] 3.3 `blocked` 状态不能直接 approve（需要先 edit 降低风险或手动 override）
  - [ ] 3.4 `approved` / `executing` / `completed` 状态不能再 approve/reject
  - [ ] 3.5 验证函数返回操作者可读的错误信息

- [ ] Task 4: 审批业务逻辑 (AC: #1, #2, #3, #4)
  - [ ] 4.1 新增 `internal/planning/reviewer/reviewer.go`，定义 `Reviewer` 结构体
  - [ ] 4.2 `Approve(ctx, planID, actor, reason) error`：验证状态、转换为 `approved`、记录审批
  - [ ] 4.3 `Reject(ctx, planID, actor, reason) error`：验证状态、转换为 `blocked`、记录审批、附带拒绝原因
  - [ ] 4.4 `Defer(ctx, planID, actor, reason) error`：验证状态、转换回 `draft`、记录审批
  - [ ] 4.5 `Edit(ctx, planID, actor, edits PlanEdits) error`：验证状态、应用编辑、重新评估策略、记录审批
  - [ ] 4.6 `PlanEdits` 结构体支持修改 scope（branch）和 execution mode
  - [ ] 4.7 每次审批操作均记录 `ApprovalRecord`，包含 previous/new status

- [ ] Task 5: CLI `gitdex plan` 审批子命令 (AC: #1, #2, #4)
  - [ ] 5.1 `gitdex plan review <plan_id>` — 展示完整审查视图（步骤、风险、策略、证据、阻断、下一步动作）
  - [ ] 5.2 `gitdex plan approve <plan_id> [--reason <reason>] [--mode <mode>]` — 审批通过
  - [ ] 5.3 `gitdex plan reject <plan_id> --reason <reason>` — 驳回（reason 必填）
  - [ ] 5.4 `gitdex plan defer <plan_id> [--reason <reason>]` — 延迟处理
  - [ ] 5.5 `gitdex plan edit <plan_id> [--branch <branch>] [--mode <mode>]` — 修改并重新评估
  - [ ] 5.6 所有子命令支持 text/JSON/YAML 输出格式
  - [ ] 5.7 `review` 视图渲染包含：Evidence Refs（如果有）、Policy Verdict Bar、Current Blockers、Next Actionable Path
  - [ ] 5.8 在 `root.go` 中将新子命令注册到 plan 命令组

- [ ] Task 6: 全面测试 (AC: #1-#4)
  - [ ] 6.1 `internal/planning/transitions_test.go` — 状态转换规则覆盖
  - [ ] 6.2 `internal/planning/reviewer/reviewer_test.go` — 审批逻辑测试（approve/reject/defer/edit）
  - [ ] 6.3 `test/integration/plan_review_command_test.go` — CLI 审批命令集成测试
  - [ ] 6.4 `test/conformance/plan_approval_contract_test.go` — 审批合约一致性测试（JSON 字段、状态转换）
  - [ ] 6.5 运行 `go test ./... -count=1` 全量通过 + `golangci-lint run ./...` 零错误

- [ ] Task 7: 收尾验证 (AC: #1-#4)
  - [ ] 7.1 验证 Story 范围不超出审批流程（不含实际执行、不含持久化数据库）
  - [ ] 7.2 检查不存在未处理的 LLM 常见错误（幻觉 API、错误类型断言等）
  - [ ] 7.3 确认所有新代码与 Story 2.1 的接口兼容
  - [ ] 7.4 更新 sprint-status.yaml 标记 Story 完成

## Dev Notes

### 从 Story 2.1 学到的

**Story 2.1 关键经验：**
- `MemoryPlanStore` 是内存存储，不跨进程持久化 — 本 Story 同样使用内存存储
- `plan compile` 需要在 Git 仓库内运行，否则 fail fast
- `PolicyResult` 已经包含 verdict、reason、explanation、risk_factors、required_approvals
- `PlanStatus` 已有 6 个值：draft, review_required, approved, blocked, executing, completed
- JSON/YAML 使用 `snake_case`，时间戳使用 RFC3339 UTC

**Story 1.5 关键经验：**
- Lipgloss v2 的 `lipgloss.Color` 不是类型而是函数，使用 `color.Color`
- `tea.View` 使用 `Content` 字段不是 `String()` 方法
- 早期返回模式需要在 `KeyPressMsg` case 内部检查，不要在外层阻止其他消息

### 已有的可复用组件

| 组件 | 路径 | 复用方式 |
|------|------|---------|
| PlanStore | `internal/planning/store.go` | 扩展接口 |
| Plan Model | `internal/planning/plan.go` | 新增字段 |
| PolicyEngine | `internal/policy/engine.go` | Edit 后重新评估 |
| Output Format | `internal/cli/output/format.go` | JSON/YAML 输出 |
| Plan Commands | `internal/cli/command/plan.go` | 新增子命令 |
| Helpers | `internal/cli/command/helpers.go` | `effectiveOutputFormat` 等 |

### 架构约束

1. **审批是路由系统而非二元开关**：repo owner / security / release / quorum approval 类型
2. **所有写操作必须经过**：intent → context → structured_plan → policy → approval → queue → execute
3. **审批输出必须进入事件流和审计链**
4. **blocked 状态不能直接跳到 approved** — 必须 edit 降低风险或使用显式 override
5. **UX-DR12 动作层级**：approve 是 Primary、reject/defer/edit 是 Secondary、inspect evidence 是 Utility
6. **UX-DR13 反馈模式**：成功反馈包含可追踪链接、错误包含失败原因和恢复入口

### 本 Story 新增的文件结构
```
internal/planning/
├── plan.go           # 新增 ApprovalAction, ApprovalRecord, ExecutionMode
├── transitions.go    # 状态转换验证规则
├── transitions_test.go
├── store.go          # 扩展 PlanStore 接口
├── reviewer/
│   ├── reviewer.go   # 审批业务逻辑
│   └── reviewer_test.go

internal/cli/command/
├── plan.go           # 新增 review/approve/reject/defer/edit 子命令

test/integration/
├── plan_review_command_test.go

test/conformance/
├── plan_approval_contract_test.go
```

### 审批操作的状态机

```
review_required ──approve──→ approved
review_required ──reject───→ blocked
review_required ──defer────→ draft
review_required ──edit─────→ review_required (重新评估策略后)

blocked ──edit──→ review_required (修改后重新评估)
blocked ──!approve──→ error "blocked plans cannot be directly approved"
```

### References

- [Source: _bmad-output/planning-artifacts/architecture.md §Approval Routing]
- [Source: _bmad-output/planning-artifacts/architecture.md §Task Lifecycle & State Machine]
- [Source: _bmad-output/planning-artifacts/architecture.md §Policy, Risk, and Approval Architecture]
- [Source: _bmad-output/planning-artifacts/prd.md §FR9, FR10, FR12, FR13]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md §UX-DR9, UX-DR12, UX-DR13]
- [Source: _bmad-output/planning-artifacts/epics.md §Story 2.2 定义]

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
