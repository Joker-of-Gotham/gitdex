# Story 3.1: Authorize Gitdex Identity and Scope Through GitHub App (FR30)

Status: done

## Story

As an administrator,
I want to authorize Gitdex at repository, installation, organization, or fleet scope through the supported machine identity model,
So that automation operates with explicit boundaries instead of implicit long-lived user power.

## Acceptance Criteria

1. **Given** an administrator connecting Gitdex to GitHub **When** they authorize Gitdex for use **Then** Gitdex uses the supported GitHub App-based identity model and records the granted scope and capability boundary **And** the granted scope is visible and reviewable from within the product **And** the default authorization path does not require a long-lived PAT to operate

## Tasks / Subtasks

- [x] Task 1: 定义 Identity 领域模型与 GitHub App 身份结构 (AC: #1)
  - [x] 1.1 在 `internal/identity/app_identity.go` 中定义 `IdentityType`（github_app、pat、token）、`Capability`、`ScopeType`、`ScopeGrant`
  - [x] 1.2 定义 `AppIdentity` 结构（IdentityID、AppID、InstallationID、OrgScope、RepoScope、Capabilities、ScopeGrants）

- [x] Task 2: 实现 Identity 存储与当前身份管理 (AC: #1, #2)
  - [x] 2.1 定义 `IdentityStore` 接口：SaveIdentity、GetIdentity、ListIdentities、GetCurrentIdentity、SetCurrentIdentity
  - [x] 2.2 实现 `MemoryIdentityStore` 内存存储，支持多身份与当前身份切换
  - [x] 2.3 首次保存身份时自动设为当前身份

- [x] Task 3: 注册 CLI 身份命令 (AC: #1, #2, #3)
  - [x] 3.1 在 `internal/cli/command/identity.go` 中实现 `identity` 命令组
  - [x] 3.2 `identity show`：展示当前身份及 scope、capabilities
  - [x] 3.3 `identity list`：列出所有身份并标记当前
  - [x] 3.4 `identity register`：注册身份，支持 `--type github_app --app-id --installation-id --org-scope --repo-scope`
  - [x] 3.5 github_app 类型强制要求 app-id 与 installation-id，默认授权路径不依赖 PAT

- [x] Task 4: 编写单元测试与契约/集成测试 (AC: 全部)
  - [x] 4.1 `internal/identity/app_identity_test.go`：Save/Get/List/GetCurrent/SetCurrent、Save nil 失败
  - [x] 4.2 `test/conformance/identity_contract_test.go`：JSON snake_case、IdentityType/Capability 值、AppIdentity/ScopeGrant 往返
  - [x] 4.3 `test/integration/identity_command_test.go`：命令注册、show/list 空态、register 校验、GitHub App 注册流程

## Dev Notes

### 关键实现细节

- **IdentityType**：`github_app`（默认）、`pat`、`token`；`identity register --type github_app` 必须提供 `--app-id` 和 `--installation-id`
- **AppIdentity**：包含 AppID、InstallationID、OrgScope、RepoScope、Capabilities、ScopeGrants；注册时默认授予 read_repo、read_issues、read_prs
- **ScopeGrant**：ScopeType（repository、installation、organization、fleet）+ ScopeValue + Capabilities，支持细粒度 scope 记录
- **MemoryIdentityStore**：进程内内存存储，并发安全（sync.RWMutex）；IdentityID 未提供时自动生成 `id_<uuid8>`
- **默认授权路径**：通过 `identity register --type github_app` 注册，无需长生命周期 PAT；PAT/Token 为可选类型

### 文件结构

```
internal/identity/app_identity.go    # 领域模型与 MemoryIdentityStore
internal/cli/command/identity.go     # identity 命令组
internal/identity/app_identity_test.go
test/conformance/identity_contract_test.go
test/integration/identity_command_test.go
```

### 设计决策

- **多身份支持**：允许同一环境配置多个身份（如不同 org 的 GitHub App），通过 SetCurrentIdentity 切换
- **JSON 契约**：AppIdentity、ScopeGrant 使用 snake_case 字段名，保证 API/CLI 输出一致性
- **默认 GitHub App**：与 NFR14 一致，默认生产机器身份为 GitHub App，不假设长生命周期 PAT

### References

- Epic 3: Governance, Policy, Audit, and Emergency Control
- FR30: Administrators can authorize Gitdex at repository, installation, organization, or fleet scope with bounded capabilities
- NFR14: Default production machine identity must be GitHub App; implementation must not assume long-lived PATs

## Dev Agent Record

### Completion Notes List

- Task 1：定义 Identity 领域模型，支持 GitHub App、PAT、Token 三种类型及 scope/capability 边界
- Task 2：实现 IdentityStore 接口与 MemoryIdentityStore，支持多身份与当前身份管理
- Task 3：实现 identity show/list/register 子命令，默认使用 github_app 类型并强制 app-id/installation-id
- Task 4：编写单元测试、conformance 契约测试与 integration 测试

### File List

**New files:**
- `internal/identity/app_identity.go`
- `internal/identity/app_identity_test.go`
- `internal/cli/command/identity.go`
- `test/conformance/identity_contract_test.go`
- `test/integration/identity_command_test.go`

**Modified files:**
- 根命令注册 `identity` 命令组（通常在 `command/root.go` 或等价位置）
