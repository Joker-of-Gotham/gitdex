# Story 5.1: Define Autonomy Levels for Supported Capabilities (FR23)

Status: done

## Story

As a repository owner managing Gitdex behavior for a repository scope,
I want to configure autonomy settings and assign supported capabilities to explicit autonomy levels and scopes,
so that autonomous behavior cannot exceed the configured scope and level and operators can see those settings before launching tasks.

## Acceptance Criteria

1. **Given** a repository owner managing Gitdex behavior for a repository scope **When** they configure autonomy settings **Then** Gitdex allows them to assign supported capabilities to explicit autonomy levels and scopes **And** those settings are visible to operators before tasks are launched **And** autonomous behavior cannot exceed the configured scope and level

## Tasks / Subtasks

- [x] Task 1: 定义 Autonomy Level 与 Capability 领域模型
  - [x] 1.1 在 `internal/autonomy/levels.go` 中定义 `AutonomyLevel`（manual, supervised, autonomous, full_auto）
  - [x] 1.2 定义 `CapabilityAutonomy`（Capability、Level、Constraints、RequiresApproval）
  - [x] 1.3 定义 `AutonomyConfig`（ConfigID、Name、CapabilityAutonomies、DefaultLevel、CreatedAt）

- [x] Task 2: 实现 Autonomy 存储层
  - [x] 2.1 定义 `AutonomyStore` 接口：SaveConfig、GetConfig、GetActiveConfig、SetActiveConfig、ListConfigs
  - [x] 2.2 实现 `MemoryAutonomyStore`，支持多配置与 active 切换
  - [x] 2.3 copyAutonomyConfig 防外部修改，SaveConfig 时自动生成 ConfigID、CreatedAt

- [x] Task 3: 注册 CLI autonomy 命令组
  - [x] 3.1 在 `internal/cli/command/autonomy.go` 中实现 `autonomy show`、`list`、`set`
  - [x] 3.2 `autonomy show`：展示当前 active 配置
  - [x] 3.3 `autonomy list`：列出所有配置并标记 active
  - [x] 3.4 `autonomy set --capability <cap> --level <level>`：设置能力自治级别
  - [x] 3.5 支持 JSON/YAML 与文本输出

- [x] Task 4: 编写单元、契约与集成测试
  - [x] 4.1 `internal/autonomy/levels_test.go`：SaveGet、GetActiveConfig、ListConfigs、GetConfigNotFound
  - [x] 4.2 `test/conformance/autonomy_contract_test.go`：JSON snake_case 契约、AutonomyLevel 值校验
  - [x] 4.3 `test/integration/autonomy_command_test.go`：命令注册、help、show 空态、set 参数校验、JSON 输出

## Dev Notes

### 关键实现细节

- **AutonomyLevel**：manual、supervised、autonomous、full_auto 四个级别
- **CapabilityAutonomy**：按 capability 配置 Level、Constraints、RequiresApproval
- **MemoryAutonomyStore**：首次 SaveConfig 时若无 active 则自动设为 active；copyAutonomyConfig 深拷贝 CapabilityAutonomies 与 Constraints
- **CLI set**：必须提供 `--capability` 与 `--level`；支持更新已有 capability 或新增；level 校验只接受四枚举值

### 文件结构

```
internal/autonomy/levels.go
internal/cli/command/autonomy.go   # show/list/set（pause/resume/cancel/takeover 在 5.4）
internal/autonomy/levels_test.go
test/conformance/autonomy_contract_test.go
test/integration/autonomy_command_test.go
```

### 设计决策

- **单一 active 配置**：通过 GetActiveConfig / SetActiveConfig 管理，确保启动任务前可读取当前 scope 与 level
- **ConfigID 格式**：`cfg_` + UUID 前 8 位，便于日志与审计
- **JSON 契约**：config_id、capability_autonomies、default_level、requires_approval 等使用 snake_case

### References

- Epic 5: Autonomy, Monitoring, and Handoff
- FR23: Autonomy levels and scopes for supported capabilities

## Dev Agent Record

### Completion Notes List

- Task 1：定义 AutonomyLevel、CapabilityAutonomy、AutonomyConfig
- Task 2：实现 AutonomyStore 与 MemoryAutonomyStore
- Task 3：实现 autonomy show/list/set 子命令
- Task 4：编写 levels_test、autonomy_contract_test、autonomy_command_test

### File List

**New files:**
- `internal/autonomy/levels.go`
- `internal/autonomy/levels_test.go`
- `internal/cli/command/autonomy.go`
- `test/conformance/autonomy_contract_test.go`
- `test/integration/autonomy_command_test.go`
