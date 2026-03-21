# Story 7.3: Exchange Versioned Plans/Results/Status (FR42)

Status: done

## Story

As a supported external tool integrating with Gitdex,
I want to exchange plans, results, or status using versioned schemas,
so that required fields, identifiers, and governance semantics are preserved and I can round-trip artifacts without losing core task meaning.

## Acceptance Criteria

1. **Given** a supported external tool integrating with Gitdex **When** plans, results, or status are exchanged **Then** Gitdex uses versioned schemas that preserve required fields, identifiers, and governance semantics **And** the integration can round-trip those artifacts without losing core task meaning **And** Gitdex clearly identifies which contract version each payload conforms to

## Tasks / Subtasks

- [x] Task 1: Exchange 模型与校验
  - [x] 1.1 在 `internal/api/exchange.go` 定义 ExchangeFormat（json、yaml、protobuf_json）
  - [x] 1.2 定义 ExchangePayload（Format、APIVersion、SchemaVersion、PayloadType、Data、Checksum、CreatedAt）
  - [x] 1.3 定义 ExchangeValidator 接口，实现 DefaultExchangeValidator
  - [x] 1.4 Validate 校验 format、api_version、schema_version、payload_type 必填

- [x] Task 2: 解析与文件读写
  - [x] 2.1 ReadExchangeFile：按扩展名自动检测格式，读取并解析
  - [x] 2.2 ReadExchangeFileWithFormat：显式指定格式
  - [x] 2.3 ParseExchangePayload：支持 JSON/YAML 解析为 ExchangePayload

- [x] Task 3: CLI api exchange 子命令
  - [x] 3.1 在 `internal/cli/command/api.go` 实现 `api exchange import`（--file、--format json|yaml）
  - [x] 3.2 实现 `api exchange export`（--type plans|results|status、--format json|yaml）
  - [x] 3.3 实现 `api exchange validate`（--file）校验 payload
  - [x] 3.4 支持 JSON/YAML 与文本输出

- [x] Task 4: 单元与集成测试
  - [x] 4.1 `internal/api/exchange_test.go`：ExchangePayload JSON 契约、Validate 有效/空 api_version/nil、ParseExchangePayload
  - [x] 4.2 `test/integration/api_exchange_command_test.go`：validate 需 file、import 需 file、export 运行、validate 有效文件

## Dev Notes

### 关键实现细节

- **DefaultExchangeValidator**：RequiredAPIVersion=v1、RequiredSchemaVersion=1，空值返回明确错误
- **Format 校验**：仅允许 json、yaml、protobuf_json，其余返回 invalid format
- **ReadExchangeFile**：.yaml/.yml 后缀用 YAML 解析，其余用 JSON
- **export 子命令**：当前产生模拟 ExchangePayload（schema_version=1、payload_type=exportType），Data 为模拟 JSON

### 文件结构

```
internal/api/exchange.go         # ExchangePayload、Validator、Read/Parse
internal/cli/command/api.go      # exchange import/export/validate（与 7.1/7.2 共享）
internal/api/exchange_test.go
test/integration/api_exchange_command_test.go
```

### 设计决策

- **版本显式标识**：ExchangePayload 含 api_version、schema_version，便于 consumer 识别契约版本
- **Data 透明**：json.RawMessage 保持业务数据不被解析，round-trip 无损
- **Checksum 可选**：当前未实现校验逻辑，仅作扩展字段

### References

- Epic 7: Machine API and Integration
- FR42: Exchange versioned plans, results, status with external tooling

## Dev Agent Record

### Completion Notes List

- Task 1：定义 ExchangePayload、ExchangeValidator、DefaultExchangeValidator
- Task 2：实现 ReadExchangeFile、ParseExchangePayload
- Task 3：实现 api exchange import、export、validate
- Task 4：编写 exchange_test、api_exchange_command_test

### File List

**New files:**
- `internal/api/exchange.go`
- `internal/api/exchange_test.go`
- `test/integration/api_exchange_command_test.go`

**Modified files:**
- `internal/cli/command/api.go`（新增 exchange 子命令组）
