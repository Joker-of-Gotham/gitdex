# Story 8.4: 完整仓库操作系统

Status: ready-for-dev

## Story

As a 进入仓库后的操作者,
I want 在 TUI 中完成所有查看、文件编辑、Git 操作和 GitHub 协作操作,
So that 我不需要离开终端就能完成全部仓库维护工作。

## 验收标准

### A. 查看能力

1. 工作树查看 — 目录结构 + 文件内容 + 语法高亮
2. PR 详细视图 — 标题/描述/评论/审查/文件变更/检查状态
3. Issue 详细视图 — 标题/描述/评论/标签/里程碑
4. Commit 历史 — 提交日志、diff 查看、blame 信息
5. 分支树 — 分支列表、合并关系、ahead/behind 状态
6. 上游 fork: 显示上游仓库的 PR 和 Issue

### B. 文件系统操作

7. 创建新文件 — 指定路径和初始内容
8. 编辑文件 — 内联编辑器或调用 `$EDITOR`
9. 保存修改 — 写入磁盘
10. 删除文件 — 确认后删除
11. 文件 diff — git diff 或本地版本对比
12. 搜索 — 文件内容搜索 + 文件名搜索

### C. Git 操作

13. 暂存区管理 — add/reset/restore（文件级/行级）
14. 提交 — commit（支持消息编辑）
15. 分支管理 — 创建/切换/删除/合并/变基
16. 远端操作 — fetch/pull/push/remote 管理
17. Stash — 保存/应用/弹出/列表
18. 日志与 Blame — log/blame/show
19. 标签 — 创建/删除/列表
20. Worktree — 创建/列表/删除
21. 维护 — gc/prune/clean
22. Diff 与 Patch — 生成 diff/应用 patch

### D. GitHub 操作

23. PR 管理 — 创建/审查者/合并/关闭/评论
24. Issue 管理 — 创建/标签/分配/关闭/重新打开/评论
25. Review — 提交审查（approve/request changes/comment）
26. Actions — 查看运行状态/触发工作流
27. Releases — 创建/查看列表
28. Deployments — 查看部署状态

## 任务 / 子任务

### A 区: 查看视图

- [ ] T1: PR 详细视图 `pr_detail.go` (AC: #2)
  - [ ] T1.1: 头部: 标题、作者、状态（open/closed/merged）、标签
  - [ ] T1.2: 描述: Markdown 渲染（使用 glamour）
  - [ ] T1.3: 评论列表: 作者 + 时间 + 内容，可翻页
  - [ ] T1.4: 审查状态: 每位审查者的 approve/request-changes/comment 状态
  - [ ] T1.5: 文件变更: diff 列表，可展开查看
  - [ ] T1.6: 检查状态: CI/CD check 结果
  - [ ] T1.7: 扩展 GitHub client — `GetPullRequestDetail`, `ListPRComments`, `ListPRReviews`, `ListPRFiles`
- [ ] T2: Issue 详细视图 `issue_detail.go` (AC: #3)
  - [ ] T2.1: 头部: 标题、状态、标签、里程碑、分配者
  - [ ] T2.2: 描述: Markdown 渲染
  - [ ] T2.3: 评论列表: 作者 + 时间 + 内容
  - [ ] T2.4: 扩展 GitHub client — `GetIssueDetail`, `ListIssueComments`
- [ ] T3: Commit 历史视图 `commit_log.go` (AC: #4)
  - [ ] T3.1: 日志列表: hash (short) + 作者 + 时间 + 消息（一行）
  - [ ] T3.2: 选中 commit: 展开显示完整消息 + diff
  - [ ] T3.3: Blame: 选中文件 → 行级 blame 信息
  - [ ] T3.4: 调用 `gitops.GitExecutor` 执行 `git log --format=...`、`git show`、`git blame`
- [ ] T4: 分支树视图 `branch_tree.go` (AC: #5)
  - [ ] T4.1: 本地分支列表 + 当前分支标记
  - [ ] T4.2: 远端分支列表
  - [ ] T4.3: 每个分支显示 ahead/behind（相对默认分支或指定分支）
  - [ ] T4.4: 调用 `gitops.BranchManager.List()` 和 `git rev-list --left-right --count`

### B 区: 文件系统操作

- [ ] T5: 文件编辑器 `editor.go` (AC: #7, #8, #9)
  - [ ] T5.1: 简单模式: 调用 `$EDITOR` 或 `$VISUAL` 打开文件
  - [ ] T5.2: 内联模式: 基于 Bubble Tea textarea 的简单编辑（可选，Phase 2）
  - [ ] T5.3: 新建文件: 输入路径 → 创建 → 打开编辑器
  - [ ] T5.4: 保存: 编辑器退出后自动检测变更
- [ ] T6: 文件操作命令 (AC: #10, #11, #12)
  - [ ] T6.1: `/new <path>` — 创建新文件
  - [ ] T6.2: `/edit <path>` — 编辑文件
  - [ ] T6.3: `/rm <path>` — 删除文件（确认对话）
  - [ ] T6.4: `/diff [path]` — 显示 git diff（全部或指定文件）
  - [ ] T6.5: `/search <pattern>` — git grep 搜索
  - [ ] T6.6: `/find <name>` — 文件名搜索

### C 区: Git 操作命令

- [ ] T7: 暂存区命令 (AC: #13)
  - [ ] T7.1: `/add [path|.]` — git add
  - [ ] T7.2: `/reset [path]` — git reset（取消暂存）
  - [ ] T7.3: `/restore [path]` — git restore（撤销修改）
  - [ ] T7.4: `/status` — git status（工作区/暂存区摘要）
- [ ] T8: 提交命令 (AC: #14)
  - [ ] T8.1: `/commit <message>` — git commit -m
  - [ ] T8.2: `/commit` (无参) — 打开编辑器写 commit message
  - [ ] T8.3: `/amend` — git commit --amend
- [ ] T9: 分支命令 (AC: #15)
  - [ ] T9.1: `/branch` — 列出分支
  - [ ] T9.2: `/branch <name>` — 创建分支
  - [ ] T9.3: `/checkout <name>` — 切换分支
  - [ ] T9.4: `/merge <name>` — 合并分支
  - [ ] T9.5: `/rebase <name>` — 变基
  - [ ] T9.6: `/branch -d <name>` — 删除分支
- [ ] T10: 远端命令 (AC: #16)
  - [ ] T10.1: `/fetch [remote]` — git fetch
  - [ ] T10.2: `/pull [remote] [branch]` — git pull
  - [ ] T10.3: `/push [remote] [branch]` — git push
  - [ ] T10.4: `/remote` — 列出远端
- [ ] T11: Stash 命令 (AC: #17)
  - [ ] T11.1: `/stash` — git stash
  - [ ] T11.2: `/stash pop` — git stash pop
  - [ ] T11.3: `/stash list` — git stash list
  - [ ] T11.4: `/stash apply [n]` — git stash apply
- [ ] T12: 其他 Git 命令 (AC: #18, #19, #20, #21, #22)
  - [ ] T12.1: `/log [--oneline] [-n N]` — git log
  - [ ] T12.2: `/blame <path>` — git blame
  - [ ] T12.3: `/tag [name]` — 列出/创建标签
  - [ ] T12.4: `/worktree` — worktree 管理
  - [ ] T12.5: `/gc` — git gc
  - [ ] T12.6: `/clean` — 清理未跟踪文件（确认对话）

### D 区: GitHub 操作命令

- [ ] T13: PR 操作命令 (AC: #23, #25)
  - [ ] T13.1: `/pr create` — 创建 PR（标题/描述/目标分支）
  - [ ] T13.2: `/pr merge <number>` — 合并 PR
  - [ ] T13.3: `/pr close <number>` — 关闭 PR
  - [ ] T13.4: `/pr comment <number> <text>` — 评论
  - [ ] T13.5: `/pr review <number> approve|request-changes|comment` — 提交审查
  - [ ] T13.6: 扩展 GitHub client — `CreatePullRequest`, `SubmitPRReview`
- [ ] T14: Issue 操作命令 (AC: #24)
  - [ ] T14.1: `/issue create` — 创建 Issue（标题/描述/标签）
  - [ ] T14.2: `/issue close <number>` — 关闭 Issue
  - [ ] T14.3: `/issue reopen <number>` — 重新打开
  - [ ] T14.4: `/issue comment <number> <text>` — 评论
  - [ ] T14.5: `/issue label <number> <labels>` — 添加标签
  - [ ] T14.6: `/issue assign <number> <users>` — 分配
- [ ] T15: Actions 和 Release 命令 (AC: #26, #27, #28)
  - [ ] T15.1: `/actions` — 查看工作流运行列表
  - [ ] T15.2: `/actions run <workflow>` — 触发工作流
  - [ ] T15.3: `/release create` — 创建 Release
  - [ ] T15.4: `/deploy` — 查看部署列表
  - [ ] T15.5: 扩展 GitHub client — `TriggerWorkflow`, `CreatePullRequest`

### 只读保护

- [ ] T16: 只读模式保护 (AC: 全局)
  - [ ] T16.1: 命令路由前检查 `app.activeRepo.IsReadOnly`
  - [ ] T16.2: 写操作在只读模式下返回: "当前为只读模式，请克隆到本地后操作"
  - [ ] T16.3: 只读模式下: B 区(除查看外)、C 区、D 区写操作均禁用

## Dev Notes

### 已有 Git 操作基础

`internal/gitops/` 已实现的 Manager:

| Manager | 已有方法 | 可直接映射的命令 |
|---------|---------|----------------|
| `GitExecutor` | `Run()`, `RunWithTimeout()` | 底层执行 |
| `BranchManager` | `List()`, `Create()`, `Delete()`, `Checkout()`, `Merge()`, `Rebase()` | /branch, /checkout, /merge, /rebase |
| `CommitManager` | `Add()`, `Reset()`, `Restore()`, `Commit()`, `Stash*()` | /add, /reset, /restore, /commit, /stash |
| `RemoteManager` | `Clone()`, `Fetch()`, `Push()`, `ListRemotes()` | /fetch, /push, /remote |
| `Syncer` | `FastForward()`, `Push()`, `StashAndPull()` | /pull |
| `PatchManager` | `Diff()`, `Apply()` | /diff |
| `Inspector` | `Inspect()` | /status |
| `HygieneExecutor` | `PruneRemotes()`, `GC()`, `CleanUntracked()` | /gc, /clean |
| `WorktreeManager` | `Create()`, `List()`, `Remove()` | /worktree |

### 已有 GitHub API 基础

`internal/platform/github/client.go` 已有:
- 读: `GetRepository`, `ListOpenPullRequests`, `ListOpenIssues`, `ListWorkflowRuns`, `ListDeployments`, `ListReleases`, `GetCombinedStatus`, `ListCheckRuns`
- 写: `CreateIssue`, `UpdateIssue`, `CreateComment`, `AddLabels`, `SetAssignees`, `CloseIssue`, `ReopenIssue`, `MergePullRequest`, `CreateRelease`

### 需扩展的 GitHub API

```go
// PR 详情
func (c *Client) GetPullRequestDetail(ctx, owner, repo string, number int) (*PullRequestDetail, error)
func (c *Client) ListPRComments(ctx, owner, repo string, number int) ([]Comment, error)
func (c *Client) ListPRReviews(ctx, owner, repo string, number int) ([]Review, error)
func (c *Client) ListPRFiles(ctx, owner, repo string, number int) ([]CommitFile, error)
func (c *Client) CreatePullRequest(ctx, owner, repo, title, body, head, base string) (*PullRequest, error)
func (c *Client) SubmitPRReview(ctx, owner, repo string, number int, event, body string) error

// Issue 详情
func (c *Client) GetIssueDetail(ctx, owner, repo string, number int) (*IssueDetail, error)
func (c *Client) ListIssueComments(ctx, owner, repo string, number int) ([]Comment, error)

// Actions
func (c *Client) TriggerWorkflow(ctx, owner, repo string, workflowID int64, ref string) error
```

### 命令路由设计

`app.go` 的 `executeCommand` 扩展:
```go
switch parts[0] {
case "/add", "/reset", "/restore", "/commit", "/amend":  → gitOpsHandler
case "/branch", "/checkout", "/merge", "/rebase":         → branchHandler
case "/fetch", "/pull", "/push", "/remote":               → remoteHandler
case "/stash", "/log", "/blame", "/tag", "/worktree":     → gitMiscHandler
case "/pr":                                                → prHandler
case "/issue":                                             → issueHandler
case "/actions", "/release", "/deploy":                    → githubMiscHandler
case "/new", "/edit", "/rm", "/diff", "/search", "/find": → fileHandler
}
```

### Project Structure Notes

- 新建视图文件在 `internal/tui/views/` 下
- 命令处理器集中在 `internal/tui/app/commands.go`（新建，从 app.go 提取）
- GitHub client 扩展方法直接添加到 `client.go`
- 所有 gitops 调用通过 `tea.Cmd` 异步执行，结果作为 Msg 返回

### References

- [Source: internal/gitops/] — 完整 Git 操作包
- [Source: internal/platform/github/client.go] — GitHub API client
- [Source: internal/tui/views/pulls.go, issues.go, files.go] — 已有列表视图
- [Reference: lazygit/pkg/gui/] — 分支树、commit 历史展示模式
- [Reference: gh-dash] — PR/Issue 详情渲染
- [Reference: git-scm.com/docs/git] — Git 命令参考
- [Reference: go-github/v84 API docs]

## Dev Agent Record

### Agent Model Used

（待实现时填写）

### Completion Notes List

### File List
