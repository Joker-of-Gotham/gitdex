# Story 4.4: Coordinate Cross-Object Task Context (FR21)

Status: done

## Story

As an operator,
I want to view a task that touches multiple repository and GitHub objects (branch, PR, issue, comment, workflow, deployment) as one coordinated context,
so that I can navigate evidence while preserving cross-links instead of manual lookups.

## Acceptance Criteria

1. **Given** a task that touches multiple repository and GitHub objects **When** the operator inspects the task in Gitdex **Then** the task view shows the linked branch, PR, issue, comment, workflow, and deployment objects as one coordinated context.
2. **Given** cross-linked objects **When** the operator navigates evidence **Then** evidence navigation preserves those cross-links rather than forcing manual lookups.
3. **Given** a task lineage **When** new linked objects are created during the task **Then** they are added to the same task lineage.

## Tasks / Subtasks

- [x] Task 1: Define context domain model (AC: #1)
  - [x] 1.1 Define `LinkType` (blocks, blocked_by, relates_to, duplicate_of, parent_of, child_of)
  - [x] 1.2 Define `ObjectLink` with SourceRef, TargetRef, LinkType, CreatedAt
  - [x] 1.3 Define `TaskContext` with PrimaryObjectRef, LinkedObjects, RelatedTasks, Notes
  - [x] 1.4 LinkType.Valid() for validation
- [x] Task 2: Implement ContextStore interface and MemoryContextStore (AC: #1, #3)
  - [x] 2.1 SaveContext, GetContext, ListContexts, GetByObjectRef
  - [x] 2.2 Index by ContextID and PrimaryObjectRef
  - [x] 2.3 Append links when context exists; create new context when not
- [x] Task 3: Implement `collab link` command (AC: #1, #3)
  - [x] 3.1 Args: source_ref target_ref; --type default relates_to
  - [x] 3.2 Validate LinkType (blocks, blocked_by, relates_to, duplicate_of, parent_of, child_of)
  - [x] 3.3 Create or update TaskContext for source_ref, append ObjectLink
  - [x] 3.4 Text: "Linked source --[type]--> target"
- [x] Task 4: Implement `collab context <object_ref>` command (AC: #1, #2)
  - [x] 4.1 GetByObjectRef; return empty context if not found
  - [x] 4.2 renderContextText: Linked objects (source --[type]--> target), RelatedTasks, Notes
  - [x] 4.3 Structured output with primary_object_ref, linked_objects
- [x] Task 5: Add tests (AC: #1–#3)
  - [x] 5.1 context_test.go: SaveAndGet, GetByObjectRef, ListContexts, LinkType.Valid
  - [x] 5.2 collab_context_command_test.go: link, context, JSON output
  - [x] 5.3 collab_context_contract_test.go: ObjectLink, TaskContext JSON contract

## Dev Notes

### 关键实现细节
- `link` 命令以 source_ref 为主对象，若不存在 TaskContext 则创建；每次 link 追加到 `LinkedObjects`。
- `GetByObjectRef` 仅按 PrimaryObjectRef 查找；一个 object 仅能作为一个 context 的主对象。
- link 命令在 save 前会 GET 现有 context，追加新 link 后写回，实现 AC#3 的 lineage 追加。
- ObjectLink 使用 owner/repo#number 格式，与 CollaborationObject.ObjectRef 一致。

### 文件结构
```
internal/collaboration/context.go         # LinkType, ObjectLink, TaskContext, ContextStore, MemoryContextStore
internal/cli/command/collab.go            # newCollabLinkCommand, newCollabContextCommand, renderContextText
internal/collaboration/context_test.go
test/integration/collab_context_command_test.go
test/conformance/collab_context_contract_test.go
```

### 设计决策
- 单主对象模型：一个 TaskContext 对应一个 PrimaryObjectRef，简化查找。
- 新 link 通过 collab link 显式添加；create/comment 等 mutation 未自动添加 link，可由后续扩展。
- RelatedTasks 为字符串数组，可存储外部任务 ID；当前无自动填充逻辑。

### References
- FR21: Coordinate cross-object task context
- Story 4.1: ObjectRef format, CollaborationObject
