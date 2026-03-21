# Story 4.2: Create, Update, and Respond to Collaboration Objects (FR19)

Status: done

## Story

As a maintainer,
I want to create, update, and respond to collaboration objects (issues, PRs, comments) through Gitdex,
so that I can execute supported GitHub actions through the CLI and have results recorded with object linkage.

## Acceptance Criteria

1. **Given** an authorized repository and a supported issue, pull request, comment, or review action **When** the maintainer submits that action through Gitdex **Then** Gitdex executes it through the supported GitHub integration path and records the resulting object linkage.
2. **Given** a mutation request **When** any policy or scope restriction applies **Then** the restriction is explained before the write occurs.
3. **Given** a successful mutation **When** the operation completes **Then** the completion summary includes the affected GitHub object references.

## Tasks / Subtasks

- [x] Task 1: Define mutation domain model (AC: #1)
  - [x] 1.1 Define `MutationType` (create, update, comment, close, reopen, label, assign, merge)
  - [x] 1.2 Define `MutationRequest` with type, object type, repo, number, title, body, labels, assignees
  - [x] 1.3 Define `MutationResult` with success, message, object reference
  - [x] 1.4 Define `MutationEngine` interface
- [x] Task 2: Implement SimulatedMutationEngine (AC: #1, #3)
  - [x] 2.1 executeCreate: auto-increment number, persist to ObjectStore
  - [x] 2.2 executeComment: increment CommentsCount
  - [x] 2.3 executeClose / executeReopen: update State
  - [x] 2.4 executeUpdate: title, body, labels, assignees
  - [x] 2.5 Unsupported type returns error message (AC: #2)
- [x] Task 3: Implement `collab create` command (AC: #1, #3)
  - [x] 3.1 Required flags: --type, --repo, --title; optional --body
  - [x] 3.2 parseCollabObjectType for issue, pr, discussion
  - [x] 3.3 renderMutationResultText for success/failure with object ref
- [x] Task 4: Implement `collab comment` and `collab close` / `collab reopen` (AC: #1, #3)
  - [x] 4.1 comment: args owner/repo#number, --body required
  - [x] 4.2 close/reopen: args owner/repo#number
  - [x] 4.3 Result includes object reference on success
- [x] Task 5: Add tests (AC: #1–#3)
  - [x] 5.1 mutations_test.go: Create, Comment, Close, Reopen, UnsupportedType
  - [x] 5.2 collab_mutations_command_test.go: create then show, close then reopen, required flags
  - [x] 5.3 collab_mutations_contract_test.go: JSON snake_case, round-trip

## Dev Notes

### 关键实现细节
- `SimulatedMutationEngine` 使用 `ObjectStore` 持久化，create 时从现有对象中取最大 number+1 分配新号。
- `executeCreate` 生成 URL：`https://github.com/{owner}/{repo}/issues/{number}`。
- 失败时 `MutationResult.Success=false`，`Message` 说明原因（如 object not found、number required）。
- `renderMutationResultText` 在成功时输出 `owner/repo#N - Title`。

### 文件结构
```
internal/collaboration/mutations.go       # MutationType, MutationRequest, MutationResult, MutationEngine, SimulatedMutationEngine
internal/cli/command/collab.go            # newCollabCreateCommand, newCollabCommentCommand, newCollabCloseCommand, newCollabReopenCommand
internal/collaboration/mutations_test.go
test/integration/collab_mutations_command_test.go
test/conformance/collab_mutations_contract_test.go
```

### 设计决策
- 当前为 simulated 实现，未调用真实 GitHub API；便于离线/测试。真实集成时替换 `SimulatedMutationEngine` 即可。
- comment 不存储评论内容，仅递增 `CommentsCount`；后续可扩展为 CommentStore。
- 策略/范围限制在失败时通过 Message 说明，未单独做 pre-flight 检查界面。

### References
- FR19: Create, update, respond to collaboration objects
- Story 4.1: ObjectStore and CollaborationObject
