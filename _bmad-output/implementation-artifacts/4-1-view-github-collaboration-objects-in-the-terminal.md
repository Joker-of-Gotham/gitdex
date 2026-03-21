# Story 4.1: View GitHub Collaboration Objects in the Terminal (FR18)

Status: done

## Story

As a maintainer,
I want to view GitHub collaboration objects (issues, PRs, reviews, workflows, deployments) in a unified terminal interface,
so that I can navigate from summary rows into detailed object views without losing repository context.

## Acceptance Criteria

1. **Given** an authorized repository with active GitHub collaboration objects **When** the maintainer opens the collaboration view **Then** the terminal surfaces issues, PRs, reviews, workflows, and deployment state in a unified, navigable interface.
2. **Given** an authorized repository **When** the maintainer lists or shows objects **Then** the operator can move from summary rows into detailed object views without losing repository context.
3. **Given** collaboration objects **When** the maintainer uses list or show commands **Then** rich TUI and text-only mode expose the same object information hierarchy.

## Tasks / Subtasks

- [x] Task 1: Define collaboration object domain model (AC: #1)
  - [x] 1.1 Define `ObjectType` (issue, pull_request, discussion, release, check_run)
  - [x] 1.2 Define `CollaborationObject` with repo, number, title, state, author, assignees, labels, body, timestamps
  - [x] 1.3 Define `ObjectFilter` for type, state, labels, assignee, author, milestone, repo filtering
  - [x] 1.4 Define `ObjectRef()` for stable reference (owner/repo#number)
- [x] Task 2: Implement ObjectStore interface and MemoryObjectStore (AC: #1, #2)
  - [x] 2.1 SaveObject, GetObject, ListObjects, GetByRepoAndNumber
  - [x] 2.2 Thread-safe in-memory implementation with byID and byRepoNo indices
  - [x] 2.3 matchFilter logic for ObjectFilter
- [x] Task 3: Implement `collab list` command (AC: #1, #2, #3)
  - [x] 3.1 Support --type (issue, pr, discussion), --state (open, closed, all), --repo, --label
  - [x] 3.2 Text output: table with #, Type, State, Title, Repo
  - [x] 3.3 Structured output (JSON/YAML) with objects array
- [x] Task 4: Implement `collab show <owner/repo#number>` command (AC: #2, #3)
  - [x] 4.1 parseObjectRef for owner/repo#number format
  - [x] 4.2 Full object details including body, labels, assignees, URL
  - [x] 4.3 Text and structured output modes
- [x] Task 5: Add conformance and integration tests (AC: #1–#3)
  - [x] 5.1 Unit tests: objects_test.go for MemoryObjectStore
  - [x] 5.2 Integration tests: collab_command_test.go (list, show, help)
  - [x] 5.3 Conformance: collab_contract_test.go (JSON snake_case, round-trip)

## Dev Notes

### 关键实现细节
- `CollaborationObject` 使用 `object_id`（UUID）、`owner/repo#number` 作为唯一引用；PR 的 ObjectRef 为 `owner/repo#pr/N`。
- `parseCollabObjectType` 支持 `pr` 和 `pull_request` 别名。
- `renderCollabListText` 输出表格，标题截断为 28 字符；`renderCollabShowText` 输出完整详情并包含 Body 块。
- JSON/YAML 输出通过 `clioutput.WriteValue` 与全局 `--output` 一致。

### 文件结构
```
internal/collaboration/objects.go     # ObjectType, CollaborationObject, ObjectFilter, ObjectStore, MemoryObjectStore
internal/cli/command/collab.go       # newCollabListCommand, newCollabShowCommand, parseObjectRef, renderCollab*
internal/collaboration/objects_test.go
test/integration/collab_command_test.go
test/conformance/collab_contract_test.go
```

### 设计决策
- 使用 in-memory store 作为默认实现，便于 CLI 独立运行；后续可换为 GitHub API 实现。
- 对象引用格式 `owner/repo#number` 与 GitHub 约定一致。
- 文本与结构化输出共享同一数据源，保证信息等价。

### References
- FR18: View GitHub collaboration objects in terminal
- `internal/cli/output` for format handling
