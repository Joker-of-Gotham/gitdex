# Story 8.3: 仓库自动发现与选择（gh-dash 模式）

Status: ready-for-dev

## Story

As a GitHub 用户,
I want Gitdex 在配置 PAT/GitHub App 后自动发现我的所有仓库并展示为列表,
So that 我可以快速选择一个仓库进入，开始工作。

## 验收标准

1. 已配置认证 → 自动抓取用户 GitHub 仓库列表，显示为可选条目
2. 每个仓库条目显示: 仓库名、星数、语言、最近更新、本地状态、fork 关系、PR 数、Issue 数
3. 选择本地已存在仓库 → 切换上下文到该仓库，加载完整状态
4. 选择仅远端仓库 → 询问: (a) 克隆到本地 (b) 只读远端模式进入
5. 克隆到本地 → 可自定义目标目录、显示进度、完成后自动进入
6. 只读远端模式 → 可查看但禁用修改/提交操作
7. 仓库列表支持搜索过滤

## 任务 / 子任务

- [ ] T1: 扩展 GitHub client — ListUserRepositories (AC: #1)
  - [ ] T1.1: `ListUserRepositories(ctx, opts)` — 使用 `go-github` 的 `Repositories.List`
  - [ ] T1.2: 分页处理 — 自动遍历所有页
  - [ ] T1.3: 返回 `[]RepoListItem` 含 name, owner, stars, language, updatedAt, fork, defaultBranch
  - [ ] T1.4: 使用 ETag/If-None-Match 做条件请求，减少 API 调用
- [ ] T2: 创建仓库列表视图 `repos.go` (AC: #2, #7)
  - [ ] T2.1: `ReposView` struct — items, filtered, cursor, searchQuery
  - [ ] T2.2: 渲染: 表格布局，列 = 名称 | ⭐ | 语言 | 更新 | 状态 | PR | Issue
  - [ ] T2.3: 搜索: 实时过滤（输入框 + 列表联动）
  - [ ] T2.4: 排序: 按名称/星数/更新时间
  - [ ] T2.5: 状态标记: ✓ 本地存在 / ✗ 仅远端 / ⑂ fork
- [ ] T3: 本地仓库检测 (AC: #2)
  - [ ] T3.1: 配置 `workspace_roots []string` — 用户指定的本地仓库根目录列表
  - [ ] T3.2: 扫描 workspace_roots 下一层/两层目录，检测 `.git` 目录
  - [ ] T3.3: 对检测到的 git 仓库，读取 `remote.origin.url`，与 GitHub 仓库匹配
  - [ ] T3.4: 构建 localPath 映射: `map[string]string` (owner/repo → localPath)
- [ ] T4: 仓库上下文切换 (AC: #3, #6)
  - [ ] T4.1: `RepoContext` 类型定义:
    ```go
    type RepoContext struct {
        Owner, Name, FullName string
        LocalPath             string
        IsLocal               bool
        IsReadOnly            bool
        DefaultBranch         string
    }
    ```
  - [ ] T4.2: `app.Model` 新增 `activeRepo *RepoContext`
  - [ ] T4.3: `SwitchRepoMsg{Repo RepoContext}` 消息触发全局上下文切换
  - [ ] T4.4: 视图标题栏显示当前仓库名（如 `owner/repo` 或 `owner/repo [只读]`）
- [ ] T5: 克隆工作流 (AC: #4, #5)
  - [ ] T5.1: 选择仅远端仓库 → 弹出选择对话: "克隆到本地" / "只读模式进入"
  - [ ] T5.2: 克隆: 目录输入框（默认 `workspace_roots[0]/repo-name`）
  - [ ] T5.3: 调用 `gitops.RemoteManager.Clone(url, targetPath)`
  - [ ] T5.4: 进度: StatusBar 显示 "正在克隆 owner/repo..."
  - [ ] T5.5: 完成: 更新 RepoContext.IsLocal=true，自动进入
- [ ] T6: 只读远端模式 (AC: #6)
  - [ ] T6.1: `RepoContext.IsReadOnly = true` + `RepoContext.IsLocal = false`
  - [ ] T6.2: 所有写操作命令前检查 `isReadOnly`，拒绝并提示
  - [ ] T6.3: 数据加载: 仅通过 GitHub API，不调用 gitops（无本地 .git）
- [ ] T7: Dashboard 集成 (AC: #1)
  - [ ] T7.1: Dashboard 新增 "仓库" 子标签
  - [ ] T7.2: 首次启动: 若无 activeRepo，自动显示仓库列表
  - [ ] T7.3: 已有 activeRepo: 显示当前仓库概览 + "切换仓库" 操作

## Dev Notes

### GitHub API

```
GET /user/repos?sort=updated&per_page=100&page=N
```
- PAT 需要 `repo` scope
- 返回字段: `full_name`, `stargazers_count`, `language`, `updated_at`, `fork`, `open_issues_count`, `default_branch`
- PR 计数需额外调用 `GET /repos/{owner}/{repo}/pulls?state=open&per_page=1` 并读取 `Link` header 获取总数

### 本地检测算法

```
for each root in workspace_roots:
    walk root (depth 1-2):
        if dir contains .git:
            remoteURL = git -C dir config --get remote.origin.url
            normalize(remoteURL) → owner/repo
            localMap[owner/repo] = dir
```

URL 归一化: `git@github.com:owner/repo.git` → `owner/repo`，`https://github.com/owner/repo.git` → `owner/repo`

### 已有基础

- `internal/platform/github/client.go` 已有 `github.NewClient`，使用 `go-github/v84`
- `internal/gitops/remote_manager.go` 已有 `Clone()` 方法
- `internal/state/repo/model.go` 已有 `RepoSummary`、`LocalState` 等类型
- `internal/tui/views/dashboard.go` 已有子标签机制（`renderSubTabs`）

### gh-dash 参考模式

- gh-dash 使用 GraphQL 批量查询 PR/Issue 数量
- gh-dash 的 section 概念: 不同 query 配置不同列表
- Gitdex 简化为单一用户仓库列表 + 搜索过滤

### Project Structure Notes

- `internal/tui/views/repos.go` 新建，实现 `View` 接口
- `internal/platform/config/config.go` 扩展 `FileConfig` 加入 `WorkspaceRoots []string`
- `internal/state/repo/model.go` 新增 `RepoContext` 和 `RepoListItem` 类型

### References

- [Source: internal/platform/github/client.go] — GitHub client 基础
- [Source: internal/gitops/remote_manager.go] — Clone 实现
- [Source: internal/state/repo/model.go] — 仓库状态模型
- [Source: internal/tui/views/dashboard.go] — 子标签机制
- [Reference: gh-dash/internal/tui/ui.go] — TUI 仓库列表布局
- [Reference: gh-dash/internal/data/] — GitHub 数据层
- [Reference: go-github v84 Repositories.List API]

## Dev Agent Record

### Agent Model Used

（待实现时填写）

### Completion Notes List

### File List
