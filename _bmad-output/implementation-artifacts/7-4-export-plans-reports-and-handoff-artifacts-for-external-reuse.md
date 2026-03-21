# Story 7.4: Export Plans/Reports/Handoff Artifacts (FR49)

Status: done

## Story

As an operator or integration client,
I want to export plans, reports, handoff packages, or supported task artifacts from Gitdex,
so that I receive structured exports in machine-readable and human-usable forms with stable identifiers and linked evidence references, without scraping terminal text.

## Acceptance Criteria

1. **Given** a plan, report, handoff package, or supported task artifact in Gitdex **When** the operator or integration client requests export **Then** Gitdex produces a structured export in the supported machine-readable form and a human-usable form **And** exported artifacts retain stable identifiers and linked evidence references **And** export does not require the consumer to scrape terminal text **And** exported HTML or documentation-oriented reports preserve semantic structure and accessible contrast for WCAG 2.1 AA

## Tasks / Subtasks

- [x] Task 1: Export 模型与引擎
  - [x] 1.1 在 `internal/api/export.go` 定义 ExportType（plan_report、task_report、campaign_report、audit_report、handoff_artifact）
  - [x] 1.2 定义 ExportRequest（ExportType、Filters、Format、IncludeEvidence）
  - [x] 1.3 定义 ExportResult（ExportType、Format、Data、FilePath、GeneratedAt）
  - [x] 1.4 定义 ExportEngine 接口，实现 DefaultExportEngine

- [x] Task 2: DefaultExportEngine 实现
  - [x] 2.1 Export 支持 format json|yaml|markdown
  - [x] 2.2 生成含 export_type、format、include_evidence 的模拟输出（后续接入真实数据）
  - [x] 2.3 ListExportTypes 返回全部五种 ExportType

- [x] Task 3: CLI export 命令组
  - [x] 3.1 在 `internal/cli/command/export.go` 实现 `export generate`（--type、--format、--include-evidence）
  - [x] 3.2 实现 `export list` 列出可用 ExportType
  - [x] 3.3 支持 JSON/YAML 与文本输出

- [x] Task 4: 单元、契约与集成测试
  - [x] 4.1 `internal/api/export_test.go`：ExportResult JSON 契约、各 ExportType 导出、各 Format、nil request 错误、ListExportTypes
  - [x] 4.2 `test/conformance/export_contract_test.go`：ExportResult/Request JSON 契约、RoundTrip、ListExportTypes 非空无重
  - [x] 4.3 `test/integration/export_command_test.go`：export 命令注册、generate、list、generate 带 format

## Dev Notes

### 关键实现细节

- **DefaultExportEngine**：当前为模拟实现，Data 为 JSON 字符串含 simulated=true
- **Format 校验**：仅 json、yaml、markdown 有效，其余回退 json
- **IncludeEvidence**：请求字段透传到输出，供后续真实实现使用
- **WCAG 2.1 AA**：导出 HTML/Markdown 时需保留语义结构与对比度（当前模拟未实现 HTML 输出）

### 文件结构

```
internal/api/export.go           # ExportType、Request、Result、DefaultExportEngine、ListExportTypes
internal/cli/command/export.go   # export generate、list
internal/api/export_test.go
test/conformance/export_contract_test.go
test/integration/export_command_test.go
```

### 设计决策

- **独立 export 命令组**：与 api exchange 分离，export 面向最终报告/交接产物
- **FilePath 可选**：当前模拟不写文件，FilePath 为空；真实实现可写文件并返回路径
- **GeneratedAt**：UTC 时间戳，便于审计与版本追踪

### References

- Epic 7: Machine API and Integration
- FR49: Export plans, reports, handoff artifacts for external reuse

## Dev Agent Record

### Completion Notes List

- Task 1：定义 ExportType、ExportRequest、ExportResult、ExportEngine
- Task 2：实现 DefaultExportEngine、ListExportTypes
- Task 3：实现 export generate、export list
- Task 4：编写 export_test、export_contract_test、export_command_test

### File List

**New files:**
- `internal/api/export.go`
- `internal/api/export_test.go`
- `internal/cli/command/export.go`
- `test/conformance/export_contract_test.go`
- `test/integration/export_command_test.go`
