# Story 3.2: Configure Policy Bundles, Risk Tiers, and Execution Boundaries (FR31, FR32, FR36, FR43)

Status: done

## Story

As an administrator,
I want to define policy bundles and execution boundaries for repositories and groups,
So that Gitdex can apply consistent governance without hard-coded or ad hoc decisions.

## Acceptance Criteria

1. **Given** one or more authorized repositories or repository groups **When** an administrator configures Gitdex governance **Then** they can define approval rules, risk tiers, protected targets, data-handling rules, and shared policy defaults **And** those policy decisions apply consistently across command, chat, API, integration, and autonomous entry points **And** policy changes are versioned and traceable for later review

## Tasks / Subtasks

- [x] Task 1: 定义 Policy Bundle 领域模型 (AC: #1, #3)
  - [x] 1.1 在 `internal/policy/bundle.go` 中定义 `PolicyBundle`（BundleID、Name、Version、CapabilityGrants、ProtectedTargets、ApprovalRules、RiskThresholds、DataHandlingRules）
  - [x] 1.2 定义 `CapabilityGrant`、`ProtectedTarget`（TargetType: branch/environment/path）、`ApprovalRule`（ApprovalType: owner/security/release/quorum）、`DataHandlingRule`

- [x] Task 2: 实现 Policy Bundle 存储与版本管理 (AC: #2, #3)
  - [x] 2.1 定义 `PolicyBundleStore` 接口：SaveBundle、GetBundle、ListBundles、GetActiveBundle、SetActiveBundle
  - [x] 2.2 实现 `MemoryBundleStore`，支持多 bundle 与 active bundle 切换
  - [x] 2.3 SaveBundle 时自动设置 Version（默认 1.0.0）、BundleID、CreatedAt，支持版本化与追溯

- [x] Task 3: 注册 CLI 策略命令 (AC: #1, #2, #3)
  - [x] 3.1 在 `internal/cli/command/policy.go` 中实现 `policy` 命令组
  - [x] 3.2 `policy show`：展示当前 active bundle 详情（approval rules、protected targets、capability grants 等）
  - [x] 3.3 `policy list`：列出所有 bundle 并标记当前 active
  - [x] 3.4 `policy create --name <name>`：创建新 bundle 并设为 active
  - [x] 3.5 支持 JSON/YAML 结构化输出与人类可读文本

- [x] Task 4: 编写单元测试与契约/集成测试 (AC: 全部)
  - [x] 4.1 `internal/policy/bundle_test.go`：SaveBundle/GetBundle/ListBundles/GetActiveBundle/SetActiveBundle、Save nil 失败
  - [x] 4.2 `test/conformance/policy_bundle_contract_test.go`：PolicyBundle 字段契约、JSON 往返
  - [x] 4.3 `test/integration/policy_command_test.go`：show 空态、list 空态、create 流程与输出校验

## Dev Notes

### 关键实现细节

- **PolicyBundle**：包含 CapabilityGrants、ProtectedTargets、ApprovalRules、RiskThresholds（map）、DataHandlingRules；支持 approval rules、risk tiers、protected targets、data-handling rules 等治理配置
- **ProtectedTarget**：TargetType 为 branch、environment、path，配合 Pattern 与 ProtectionLevel
- **ApprovalRule**：ActionPattern、RequiredApprovers、ApprovalType（owner、security、release、quorum）
- **DataHandlingRule**：RuleType、Pattern、Action，用于按 scope/retention/sensitivity 定义数据规则
- **MemoryBundleStore**：内存存储，deepCopyBundle 防止外部修改；首次 Save 时自动设为 active
- **CLI**：`policy create` 强制 `--name`；create 后自动 SetActiveBundle；show/list 支持无 bundle 时的友好提示

### 文件结构

```
internal/policy/bundle.go           # 领域模型与 MemoryBundleStore
internal/cli/command/policy.go      # policy 命令组
internal/policy/bundle_test.go
test/conformance/policy_bundle_contract_test.go
test/integration/policy_command_test.go
```

### 设计决策

- **共享 policy defaults**：通过 PolicyBundle 统一管理，支持跨 command、chat、API、integration、autonomous 入口一致应用（当前 CLI 实现为存储层，策略执行层待后续集成）
- **版本化**：PolicyBundle 含 Version 字段，默认 1.0.0；CreatedAt 记录创建时间，便于追溯
- **Active bundle**：同一时刻仅有一个 active bundle，通过 SetActiveBundle 切换，保证执行边界清晰

### References

- Epic 3: Governance, Policy, Audit, and Emergency Control
- FR31: Administrators can define policies for approvals, risk tiers, protected targets, and execution boundaries
- FR32: Gitdex can enforce policy decisions consistently across all entry points
- FR36: Administrators can define data-handling rules by scope, retention, and sensitivity
- FR43: Administrators can apply shared policy bundles and governance defaults across repo groups

## Dev Agent Record

### Completion Notes List

- Task 1：定义 PolicyBundle 及相关结构（CapabilityGrant、ProtectedTarget、ApprovalRule、DataHandlingRule）
- Task 2：实现 PolicyBundleStore 接口与 MemoryBundleStore，支持版本化与 active 管理
- Task 3：实现 policy show/list/create 子命令，展示与创建 bundle
- Task 4：编写单元测试、conformance 契约测试与 integration 测试

### File List

**New files:**
- `internal/policy/bundle.go`
- `internal/policy/bundle_test.go`
- `internal/cli/command/policy.go`
- `test/conformance/policy_bundle_contract_test.go`
- `test/integration/policy_command_test.go`

**Modified files:**
- 根命令注册 `policy` 命令组
