# Story 7.1: Submit Structured Intents/Plans/Tasks Through Machine API (FR40)

Status: done

## Story

As an authorized integration client,
I want to submit structured intents, plans, or tasks to Gitdex via a machine API,
so that Gitdex validates the payload against a versioned contract, creates governed tasks, and keeps them visible to terminal operators with stable identifiers.

## Acceptance Criteria

1. **Given** an authorized integration client **When** it submits a supported structured intent, plan, or task payload **Then** Gitdex validates the payload against a versioned contract and creates the corresponding governed task **And** invalid or out-of-scope payloads are rejected with explicit, machine-readable error responses **And** the created task remains visible to terminal operators using the same identifiers and task lineage

## Tasks / Subtasks

- [x] Task 1: API 请求/响应与路由模型
  - [x] 1.1 在 `internal/api/endpoints.go` 定义 APIRequest（RequestID、Endpoint、Method、Payload、APIVersion、Timestamp）
  - [x] 1.2 定义 APIResponse（StatusCode、Payload、Errors、Timestamp）、APIError（Code、Message、Field）
  - [x] 1.3 定义 Endpoint、HandlerFunc、APIRouter 接口

- [x] Task 2: MemoryAPIRouter 实现
  - [x] 2.1 实现 MemoryAPIRouter：Register、Handle、ListEndpoints
  - [x] 2.2 注册 POST /api/v1/intents、/api/v1/plans、/api/v1/tasks
  - [x] 2.3 handleSubmitIntent/Plan/Task：空 payload 返回 400 + machine-readable error；有效 payload 返回 201 + id/accepted/created_at
  - [x] 2.4 未知 endpoint 返回 404；handler 异常返回 500

- [x] Task 3: CLI api 命令组
  - [x] 3.1 在 `internal/cli/command/api.go` 实现 `api submit`（--endpoint intents|plans|tasks、--payload JSON）
  - [x] 3.2 实现 `api endpoints` 列出可用端点
  - [x] 3.3 支持 JSON/YAML 与文本输出

- [x] Task 4: 单元、契约与集成测试
  - [x] 4.1 `internal/api/endpoints_test.go`：APIRequest/Response JSON 契约、Handle intent 成功、空 payload 400、ListEndpoints、NotFound 404
  - [x] 4.2 `test/conformance/api_contract_test.go`：APIRequest/Response/Error JSON 字段契约、HandleIntent 201
  - [x] 4.3 `test/integration/api_command_test.go`：api 命令注册、submit 必需 endpoint、endpoints、submit intents/plans/tasks

## Dev Notes

### 关键实现细节

- **APIRequest/Response**：使用 json.RawMessage 保持 payload 结构不变，RequestID 透传
- **normalizePath**：空路径默认 `/api/v1/`，自动补前缀保证统一路由
- **Handler 错误处理**：返回 *APIResponse 而非 error，便于 400/404/500 统一语义
- **空 payload 校验**：len(req.Payload)==0 时返回 400，Errors 含 code/message/field

### 文件结构

```
internal/api/endpoints.go      # API 路由、Request/Response、intents/plans/tasks handlers
internal/cli/command/api.go    # submit、endpoints（query/get/exchange 在 7.2/7.3）
internal/api/endpoints_test.go
test/conformance/api_contract_test.go
test/integration/api_command_test.go
```

### 设计决策

- **版本化**：APIVersion 常量 "v1"，便于后续扩展版本化 contract
- **幂等性**：每次 submit 生成新 UUID id，无幂等键（后续可按 request_id 去重）
- **JSON 契约**：request_id、status_code、errors、api_version 使用 snake_case

### References

- Epic 7: Machine API and Integration
- FR40: Submit structured intents, plans, tasks via machine API

## Dev Agent Record

### Completion Notes List

- Task 1：定义 APIRequest、APIResponse、APIError、Endpoint、APIRouter
- Task 2：实现 MemoryAPIRouter 与 intents/plans/tasks handlers
- Task 3：实现 api submit、api endpoints
- Task 4：编写 endpoints_test、api_contract_test、api_command_test

### File List

**New files:**
- `internal/api/endpoints.go`
- `internal/api/endpoints_test.go`
- `internal/cli/command/api.go`
- `test/conformance/api_contract_test.go`
- `test/integration/api_command_test.go`
