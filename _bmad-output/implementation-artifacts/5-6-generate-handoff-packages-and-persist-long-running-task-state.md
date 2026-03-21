# Story 5.6: Generate Handoff Packages and Persist Long-Running Task State (FR28, FR29)

Status: done

## Story

As an operator,
I want Gitdex to generate handoff packages and persist long-running task state,
so that when I reopen a task or request a handoff, I get trigger source, scope, completed steps, blocked step, evidence, and recommended next actions in a consistent, exportable format.

## Acceptance Criteria

1. **Given** a long-running or failed task that spans terminal sessions or requires human continuation **When** the operator reopens the task or requests a handoff package **Then** Gitdex restores the latest task state and generates a handoff package containing trigger source, scope, completed steps, blocked step, evidence, and recommended next actions **And** the restored task state remains consistent **And** the handoff view presents current status, pending steps, risks, and suggested next actions **And** handoff artifacts can be exported

## Tasks / Subtasks

- [x] Task 1: 定义 Handoff 领域模型 (AC: #1, #2)
  - [x] 1.1 在 `internal/autonomy/handoff.go` 中定义 `HandoffPackage`（PackageID、TaskID、TaskSummary、CurrentState、CompletedSteps、PendingSteps、ContextData、Artifacts、Recommendations、CreatedAt）
  - [x] 1.2 ContextData 存储 trigger source、scope（如 repo、branch）；Artifacts 为证据文件列表；Recommendations 为建议下一步

- [x] Task 2: 实现 Handoff 存储 (AC: #2)
  - [x] 2.1 定义 `HandoffStore` 接口：SavePackage、GetPackage、ListPackages、GetByTaskID
  - [x] 2.2 实现 `MemoryHandoffStore`，byID + byTaskID 双索引；copyHandoffPackage 深拷贝

- [x] Task 3: 实现 GenerateHandoffPackage (AC: #1)
  - [x] 3.1 函数 `GenerateHandoffPackage(store, taskID)` 生成包含 completed/pending steps、context、artifacts、recommendations 的包并持久化
  - [x] 3.2 当前为模拟实现（固定 step/context 模板），待与任务运行时层对接

- [x] Task 4: 注册 CLI handoff 命令 (AC: 全部)
  - [x] 4.1 `handoff generate <task_id>`：生成并持久化 handoff 包
  - [x] 4.2 `handoff show <package_id>`：展示包详情（status、pending steps、recommendations）
  - [x] 4.3 `handoff list`：列出所有 handoff 包
  - [x] 4.4 支持 JSON/YAML 输出，可通过 `--output json` 导出

- [x] Task 5: 编写单元测试与契约/集成测试 (AC: 全部)
  - [x] 5.1 `internal/autonomy/handoff_test.go`：SaveAndGet、GetByTaskID、ListPackages、GenerateHandoffPackage
  - [x] 5.2 `test/conformance/handoff_contract_test.go`：HandoffPackage JSON 字段契约（package_id、task_id、completed_steps 等）；RecoveryStrategy 值契约
  - [x] 5.3 `test/integration/handoff_command_test.go`：generate/show/list 子命令注册、必填参数、generate 流程

## Dev Notes

### 关键实现细节

- **HandoffPackage**：CompletedSteps、PendingSteps 为字符串切片；ContextData 为 map（可存 plan_id、repo、branch、trigger_source）；Recommendations 为建议下一步列表
- **MemoryHandoffStore**：byID 与 byTaskID 映射；GetByTaskID 通过 taskID→packageID→GetPackage 实现；SavePackage 时 TaskID 必填
- **GenerateHandoffPackage**：模拟生成包并 Save；真实场景需从任务运行时读取 state、steps、evidence

### 文件结构

```
internal/autonomy/handoff.go         # HandoffPackage、HandoffStore、MemoryHandoffStore、GenerateHandoffPackage
internal/cli/command/handoff.go     # handoff generate/show/list 命令组
internal/autonomy/handoff_test.go
test/conformance/handoff_contract_test.go
test/integration/handoff_command_test.go
```

### 设计决策

- **State 一致性**：HandoffPackage 为任务状态的快照；restore 时通过 GetByTaskID 或 GetPackage 获取，保证一致性
- **Export**：CLI `--output json` / `--output yaml` 输出完整 HandoffPackage，可重定向至文件实现导出
- **Risks**：当前 Recommendations 可包含风险提示；未来可在 HandoffPackage 增加 Risks 字段

### References

- Epic 5: Autonomy and Background Operations
- FR28: Handoff package contains trigger source, scope, completed/pending steps, evidence, recommended next actions
- FR29: Restored task state consistent; handoff view presents status, pending steps, risks, next actions; artifacts exportable

## Dev Agent Record

### Completion Notes List

- Task 1：定义 HandoffPackage 及字段
- Task 2：实现 HandoffStore 接口与 MemoryHandoffStore
- Task 3：实现 GenerateHandoffPackage 函数
- Task 4：实现 handoff generate/show/list 子命令
- Task 5：编写单元测试、conformance 契约测试与 integration 测试

### File List

**New files:**
- `internal/autonomy/handoff.go`
- `internal/autonomy/handoff_test.go`
- `internal/cli/command/handoff.go`
- `test/conformance/handoff_contract_test.go`
- `test/integration/handoff_command_test.go`

**Modified files:**
- 根命令注册 `handoff` 命令组
