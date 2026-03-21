# Story 7.2: Query Task/Campaign/Audit State Through API (FR41)

Status: done

## Story

As an authorized integration client,
I want to query task, campaign, or audit-friendly state from Gitdex via the API,
so that I receive structured state with stable identifiers, explicit status semantics, and the latest relevant evidence references in a form consistent with the terminal.

## Acceptance Criteria

1. **Given** an authorized integration client querying Gitdex **When** it requests task, campaign, or audit-friendly state **Then** Gitdex returns structured state with stable identifiers, explicit status semantics, and the latest relevant evidence references **And** active, blocked, and completed states are distinguishable **And** query responses align with the same state model shown in the terminal

## Tasks / Subtasks

- [x] Task 1: 查询模型与 QueryRouter
  - [x] 1.1 在 `internal/api/queries.go` 定义 QueryType（task_status、campaign_status、audit_log、plan_status）
  - [x] 1.2 定义 QueryRequest（QueryType、Filters、Pagination、SortBy、SortOrder）
  - [x] 1.3 定义 QueryResult（Items、TotalCount、Page、PerPage）
  - [x] 1.4 定义 QueryRouter 接口：Query、GetResource

- [x] Task 2: MemoryAPIRouter 查询扩展
  - [x] 2.1 memoryQueryStore 存储 tasks、campaigns、audit、plans 种子数据
  - [x] 2.2 Query 支持按 QueryType 过滤，分页，matchFilters
  - [x] 2.3 GetResource(endpoint, id) 支持 tasks|campaigns，404 返回 machine-readable error
  - [x] 2.4 BuildQueryRequest 从 CLI 参数构建 QueryRequest（type、filter key=value）

- [x] Task 3: CLI api query / get
  - [x] 3.1 在 `internal/cli/command/api.go` 实现 `api query`（--type tasks|campaigns|audit、--filter key=value）
  - [x] 3.2 实现 `api get`（--endpoint tasks|campaigns、--id <id>）
  - [x] 3.3 支持 JSON/YAML 与文本输出

- [x] Task 4: 单元与集成测试
  - [x] 4.1 `internal/api/queries_test.go`：QueryRequest/Result JSON 契约、BuildQueryRequest、Query、GetResource 成功、NotFound
  - [x] 4.2 `test/integration/api_query_command_test.go`：query 带 filter、campaigns、audit、get task/campaign

## Dev Notes

### 关键实现细节

- **memoryQueryStore**：单例 getGlobalQueryStore，种子 task_001、camp_001、audit_001
- **matchFilters**：遍历 item JSON 字段，与 filters 键值匹配；fmtStr 处理 string/float64/bool
- **分页**：from=(Page-1)*PerPage，to=from+PerPage，越界返回空切片
- **StoreTask**：供 submit handlers 将创建的任务写入 query store（当前未接入）

### 文件结构

```
internal/api/queries.go           # QueryRequest/Result、memoryQueryStore、Query、GetResource
internal/cli/command/api.go       # query、get（与 7.1 共享）
internal/api/queries_test.go
test/integration/api_query_command_test.go
```

### 设计决策

- **Query 与 Handle 分离**：Query 走 QueryRequest 通道，GetResource 走 endpoint+id，复用 APIResponse
- **Filters 语义**：key=value 仅做等值匹配，未提供字段视为通过
- **状态区分**：items 中含 status 字段，client 可解析 running/active/completed 等

### References

- Epic 7: Machine API and Integration
- FR41: Query task, campaign, audit state through API

## Dev Agent Record

### Completion Notes List

- Task 1：定义 QueryType、QueryRequest、QueryResult、QueryRouter
- Task 2：扩展 MemoryAPIRouter 支持 Query、GetResource、memoryQueryStore
- Task 3：实现 api query、api get
- Task 4：编写 queries_test、api_query_command_test

### File List

**New files:**
- `internal/api/queries.go`
- `internal/api/queries_test.go`
- `test/integration/api_query_command_test.go`

**Modified files:**
- `internal/cli/command/api.go`（新增 query、get）
