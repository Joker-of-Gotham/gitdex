# Epic 8: 全面功能升级 — LLM 集成、仓库发现、完整 Git/GitHub 操作与自主巡航

状态: backlog
创建日期: 2026-03-19

## 概述

在 TUI 骨架、面板合并、焦点导航、配置系统完成的基础上，进入功能全面升级阶段。本 Epic 涵盖五大核心能力：

1. **LLM 实时对话集成** — 将已有 LLM 适配器接入 Chat 视图，支持流式对话
2. **剪贴板与粘贴修复** — 解决终端右键粘贴和复制的兼容性问题
3. **仓库自动发现与选择** — 类 gh-dash 的仓库列表，自动抓取用户 GitHub 仓库，检测本地存在性
4. **完整仓库操作系统** — 查看、文件系统、Git 全操作、GitHub 全操作
5. **LLM 自主巡航系统** — 连续行为规划、无人干预自主运维、安全护栏

## 功能需求覆盖

| 需求 | 覆盖 Story |
|------|-----------|
| FR2 (自然语言对话) | 8.1 |
| FR3 (命令/对话无缝切换) | 8.1 |
| FR4 (仓库状态汇总) | 8.3 |
| FR5 (风险与下一步建议) | 8.3, 8.5 |
| FR14 (本地工作区) | 8.4 |
| FR15 (上游对比与同步) | 8.4 |
| FR16 (仓库维护) | 8.4 |
| FR17 (受控文件修改) | 8.4 |
| FR18 (终端协作对象查看) | 8.4 |
| FR19 (创建/更新协作对象) | 8.4 |
| FR20 (分拣与摘要) | 8.5 |
| FR23 (自主等级) | 8.5 |
| FR24 (持续监控) | 8.5 |
| FR25 (受控任务启动) | 8.5 |
| FR26 (暂停/恢复/取消) | 8.5 |

---

## Story 8.1: LLM 实时对话集成

**As a** Gitdex 用户,
**I want** 在 Chat 视图中输入自然语言即可获得 LLM 的实时流式回复,
**So that** 我可以通过对话获取建议、执行操作、理解仓库状态。

### 验收标准

1. **Given** 用户已在 Settings 中配置 LLM Provider/Model/API Key
   **When** 用户在 Composer 中输入自然语言并提交
   **Then** Chat 视图显示 LLM 的流式逐字回复，包含思考标记和完成标记

2. **Given** LLM 配置缺失或无效
   **When** 用户提交自然语言
   **Then** 系统显示友好错误提示，引导用户前往 Settings 配置

3. **Given** 正在进行流式回复
   **When** 用户按 Esc 或 Ctrl+C
   **Then** 流式回复中断，已收到的内容保留在 Chat 中

4. **Given** 多轮对话
   **When** 用户连续提问
   **Then** 上下文窗口自动管理，保留最近 N 条消息作为上下文

5. **Given** 支持的 LLM Provider（openai/deepseek/ollama）
   **When** 切换 Provider
   **Then** 下次对话使用新 Provider，无需重启

### 任务分解

- [ ] T1: 创建 `internal/llm/chat/session.go` — 会话管理器（上下文窗口、消息历史、token 计数）
- [ ] T2: 在 `app.go` 的 `handleSubmit` 中接入 LLM — 非命令输入走 `StreamChatCompletion`
- [ ] T3: 创建 `StreamResponseMsg` 消息类型 — 流式 chunk → ChatView 逐字渲染
- [ ] T4: ChatView 支持流式渲染 — 追加模式、光标闪烁、完成标记
- [ ] T5: 错误处理 — API 超时、配额耗尽、网络断开的友好提示
- [ ] T6: 中断机制 — Esc/Ctrl+C 取消正在进行的流式请求（`context.WithCancel`）
- [ ] T7: Provider 热切换 — ConfigSaveMsg 触发时重建 Provider 实例

### 技术要点

- 使用已有 `adapter.Provider.StreamChatCompletion()` 接口
- 流式 chunk 通过 `tea.Cmd` 返回 `StreamResponseMsg`，ChatView 增量追加
- 会话上下文使用滑动窗口（默认最近 20 条消息或 4K tokens）
- Provider 实例由 `app.Model` 持有，`ConfigSaveMsg` 时通过 `adapter.NewProviderFromConfig` 重建
- 参考项目: `openai-agents-python/src/agents/models/`、`symphony` 的 Agent Runner 模式

### 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/llm/chat/session.go` | 新建 | 会话管理器 |
| `internal/llm/chat/session_test.go` | 新建 | 会话管理器测试 |
| `internal/tui/views/chat.go` | 修改 | 流式渲染、中断、StreamResponseMsg 处理 |
| `internal/tui/views/messages.go` | 修改 | 新增 StreamResponseMsg 类型 |
| `internal/tui/app/app.go` | 修改 | handleSubmit 接入 LLM、Provider 生命周期 |
| `internal/tui/app/app_test.go` | 修改 | 补充 LLM 路径测试 |

---

## Story 8.2: 剪贴板与右键粘贴修复

**As a** Gitdex 用户,
**I want** 在终端中使用右键粘贴、Ctrl+V 粘贴、以及选择文本右键复制,
**So that** 我可以在 TUI 中自由地复制粘贴内容。

### 验收标准

1. **Given** 用户在终端中选择文本并右键
   **When** 终端发送 paste 事件
   **Then** Composer 正确接收粘贴内容（包括多行文本和特殊字符）

2. **Given** 用户在 Content 区域查看代码或日志
   **When** 用户选择文本并使用终端复制功能
   **Then** 文本正确复制到系统剪贴板（不被 TUI 事件拦截）

3. **Given** Bubble Tea v2 的 bracketed paste 模式
   **When** 应用启动
   **Then** 自动启用 bracketed paste 支持（`tea.EnableBracketedPaste`）

4. **Given** 不同终端环境（Windows Terminal、PowerShell、iTerm、gnome-terminal）
   **When** 用户执行粘贴操作
   **Then** 粘贴行为在所有支持的终端中一致工作

### 任务分解

- [ ] T1: 在 `app.go` 的 `Init()` 中启用 `tea.EnableBracketedPaste`
- [ ] T2: 确保 Composer 的 `Update` 正确处理 `tea.PasteMsg`（已部分实现，需验证多行）
- [ ] T3: Content 区域鼠标事件处理 — 不拦截终端原生选择和复制行为
- [ ] T4: 添加 `tea.EnableMouseCellMotion` 用于鼠标支持（但排除对选择的干扰）
- [ ] T5: 跨平台测试 — Windows Terminal、PowerShell、bash、zsh

### 技术要点

- Bubble Tea v2 通过 `tea.PasteMsg` 处理 bracketed paste
- `tea.EnableBracketedPaste` 作为 `tea.ProgramOption` 在启动时设置
- 鼠标事件的启用不能干扰终端原生的文本选择复制（使用 `tea.EnableMouseCellMotion` 而非 `tea.EnableMouseAllMotion`）
- 参考: `bubbletea/examples/`、`lazygit` 的鼠标处理模式

### 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/tui/app/app.go` | 修改 | Init 启用 BracketedPaste |
| `internal/tui/components/composer.go` | 修改 | 完善 PasteMsg 处理 |
| `cmd/gitdex/main.go` | 修改 | ProgramOption 传入 |
| `internal/tui/app/app_test.go` | 修改 | 粘贴集成测试 |

---

## Story 8.3: 仓库自动发现与选择（gh-dash 模式）

**As a** GitHub 用户,
**I want** Gitdex 在配置 PAT/GitHub App 后自动发现我的所有仓库并展示为列表,
**So that** 我可以快速选择一个仓库进入，开始工作。

### 验收标准

1. **Given** 用户已配置 GitHub PAT 或 GitHub App 认证
   **When** 用户进入 Dashboard 或 Explorer
   **Then** 系统自动抓取用户的 GitHub 仓库列表，显示为可选条目

2. **Given** 仓库列表
   **When** 展示每个仓库
   **Then** 每个条目显示: 仓库名、星数、语言、最近更新时间、本地存在状态（✓ 已克隆 / ✗ 仅远端）、是否有上游 fork、打开的 PR 数、打开的 Issue 数

3. **Given** 用户选择一个本地已存在的仓库
   **When** 按 Enter 进入
   **Then** Gitdex 切换上下文到该仓库，加载其完整状态（git 信息、GitHub 协作对象等）

4. **Given** 用户选择一个仅在远端存在的仓库
   **When** 按 Enter
   **Then** 询问用户: (a) 克隆到本地（可自定义目标目录），(b) 以只读远端模式进入

5. **Given** 用户选择克隆到本地
   **When** 确认克隆目录
   **Then** 执行 `git clone`，进度显示在 StatusBar，完成后自动进入该仓库

6. **Given** 用户选择只读远端模式
   **When** 进入仓库
   **Then** 可以查看代码树、提交历史、PR、Issue 等，但禁用修改/提交类操作

7. **Given** 仓库列表
   **When** 用户输入搜索关键词
   **Then** 实时过滤仓库列表

### 任务分解

- [ ] T1: 扩展 GitHub client — `ListUserRepositories()` 方法（使用 REST API，支持分页）
- [ ] T2: 创建 `internal/tui/views/repos.go` — 仓库列表视图（搜索、排序、状态标记）
- [ ] T3: 本地仓库检测 — 扫描配置的工作目录，匹配 remote URL
- [ ] T4: 仓库上下文切换 — `app.Model` 支持 `ActiveRepo` 状态，所有视图根据当前仓库加载数据
- [ ] T5: 克隆工作流 — 目录选择对话框 → `gitops.RemoteManager.Clone` → 进度反馈
- [ ] T6: 只读远端模式 — GitHub API only，禁用 git 写操作
- [ ] T7: 仓库列表搜索与过滤
- [ ] T8: 将 repos 视图嵌入 Dashboard（作为新子标签）

### 技术要点

- GitHub API: `GET /user/repos`（需 `repo` scope 的 PAT）或 GitHub App 的 installation repos
- 本地检测: 遍历配置的 `workspace_roots` 目录，对每个 git 仓库读取 `remote.origin.url` 进行匹配
- 仓库上下文: `app.Model` 新增 `activeRepo *RepoContext`，包含 owner/name/localPath/isLocal/isReadOnly
- 参考: `gh-dash` 的 `internal/tui/ui.go`、`internal/data/` 数据层
- 参考: `gh-dash` 的 PR/Issue 列表渲染和状态标记模式

### 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/platform/github/client.go` | 修改 | 新增 ListUserRepositories |
| `internal/tui/views/repos.go` | 新建 | 仓库列表视图 |
| `internal/tui/views/repos_test.go` | 新建 | 仓库列表测试 |
| `internal/tui/views/dashboard.go` | 修改 | 嵌入 repos 子标签 |
| `internal/tui/app/app.go` | 修改 | ActiveRepo 上下文管理 |
| `internal/platform/config/config.go` | 修改 | workspace_roots 配置 |
| `internal/state/repo/model.go` | 修改 | RepoContext 类型 |

---

## Story 8.4: 完整仓库操作系统

**As a** 进入仓库后的操作者,
**I want** 在 TUI 中完成所有查看、文件编辑、Git 操作和 GitHub 协作操作,
**So that** 我不需要离开终端就能完成全部仓库维护工作。

### 验收标准

#### 8.4.A: 查看能力

1. 工作树查看 — 目录结构 + 文件内容 + 语法高亮（已有 FilesView 基础）
2. PR 详细视图 — PR 标题/描述/评论/审查/文件变更/检查状态
3. Issue 详细视图 — Issue 标题/描述/评论/标签/里程碑
4. Commit 历史 — 提交日志、diff 查看、blame 信息
5. 分支树 — 分支列表、合并关系、ahead/behind 状态（参考 lazygit 的分支树）
6. 如有上游 fork: 显示上游仓库的 PR 和 Issue

#### 8.4.B: 文件系统操作

1. 创建新文件 — 指定路径和初始内容
2. 编辑文件 — 内联编辑器或调用 `$EDITOR`
3. 保存修改 — 自动写入磁盘
4. 删除文件 — 确认后删除
5. 文件 diff — 基于 `git diff` 或本地版本对比
6. 搜索 — 文件内容搜索（grep）、文件名搜索（find）

#### 8.4.C: Git 操作

1. 暂存区管理 — `git add`、`git reset`、`git restore`（文件级/行级）
2. 提交 — `git commit`（支持消息编辑）
3. 分支管理 — 创建、切换、删除、合并、变基
4. 远端操作 — `fetch`、`pull`、`push`、remote 管理
5. Stash — 保存、应用、弹出、列表
6. 日志与 Blame — `git log`、`git blame`、`git show`
7. 标签 — 创建、删除、列表
8. Worktree — 创建、列表、删除
9. 维护 — `gc`、`prune`、`clean`
10. Diff 与 Patch — 生成 diff、应用 patch

#### 8.4.D: GitHub 操作

1. PR 管理 — 创建 PR、添加评审者、合并/关闭 PR、评论
2. Issue 管理 — 创建 Issue、添加标签、分配、关闭/重新打开、评论
3. Review — 提交审查（approve/request changes/comment）
4. Actions — 查看工作流运行状态、触发工作流
5. Releases — 创建 Release、查看 Release 列表
6. Deployments — 查看部署状态

### 任务分解

- [ ] T1: PR 详细视图 — 评论列表、审查状态、文件变更 diff
- [ ] T2: Issue 详细视图 — 评论列表、标签、里程碑
- [ ] T3: Commit 历史视图 — 日志列表、diff 查看、blame
- [ ] T4: 分支树视图 — 参考 lazygit 的分支展示模式
- [ ] T5: 文件编辑器 — 简单内联编辑 or 调用 $EDITOR
- [ ] T6: 文件系统操作命令 — `/new`、`/edit`、`/rm`、`/search`
- [ ] T7: Git 操作命令集 — 映射到 `internal/gitops` 已有实现
- [ ] T8: GitHub 操作命令集 — 映射到 `internal/platform/github/client.go` 已有实现
- [ ] T9: 扩展 GitHub client — Review 提交、Workflow 触发、Comment CRUD
- [ ] T10: 只读模式保护 — isReadOnly 时禁用写操作命令

### 技术要点

- 已有基础: `gitops.*Manager` 覆盖了大部分 Git 操作
- 已有基础: `github.Client` 覆盖了基本 CRUD
- 需扩展: PR Review 提交（`POST /repos/{owner}/{repo}/pulls/{pull_number}/reviews`）
- 需扩展: Workflow 触发（`POST /repos/{owner}/{repo}/actions/workflows/{workflow_id}/dispatches`）
- 参考: `lazygit/pkg/gui/` 的分支树和 commit 历史展示
- 参考: `gh-dash` 的 PR/Issue 详情渲染
- 参考: `gitops/` 包的 BranchManager、CommitManager、RemoteManager 等

### 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/tui/views/pr_detail.go` | 新建 | PR 详细视图 |
| `internal/tui/views/issue_detail.go` | 新建 | Issue 详细视图 |
| `internal/tui/views/commit_log.go` | 新建 | Commit 历史视图 |
| `internal/tui/views/branch_tree.go` | 新建 | 分支树视图 |
| `internal/tui/views/editor.go` | 新建 | 内联文件编辑器 |
| `internal/platform/github/client.go` | 修改 | 扩展 Review/Workflow/Comment API |
| `internal/tui/app/app.go` | 修改 | 新增操作命令路由 |
| `internal/tui/views/explorer.go` | 修改 | 嵌入新的详细视图 |

---

## Story 8.5: LLM 自主巡航系统

**As a** 仓库管理者,
**I want** Gitdex 能基于 LLM 进行 7×24 无人干预自主巡航，自动发现问题并执行维护,
**So that** 仓库能保持健康状态而无需我持续手动监控。

### 验收标准

1. **Given** 用户启用自主巡航模式
   **When** 系统启动巡航
   **Then** LLM 自动扫描仓库状态（PR、Issue、分支、CI 状态、依赖更新等）

2. **Given** LLM 发现可操作项（如: 过期分支、未处理 Issue、CI 失败等）
   **When** 生成行动计划
   **Then** 低风险操作自动执行（如: 清理已合并分支），中高风险操作提交人工审批队列

3. **Given** 巡航周期运行
   **When** 每个周期完成
   **Then** 生成巡航报告: 发现的问题、已执行的操作、待审批项、风险评估

4. **Given** 自主巡航运行中
   **When** 用户随时查看巡航状态
   **Then** 显示当前巡航阶段、已完成操作列表、待审批队列

5. **Given** 安全护栏
   **When** LLM 尝试执行危险操作（如: force push、删除 protected branch）
   **Then** 操作被护栏拦截，生成告警并等待人工确认

6. **Given** 用户通过 Chat 输入自然语言指令
   **When** 指令涉及多步操作（如: "帮我把这个 Issue 修复并提交 PR"）
   **Then** LLM 生成结构化行动计划，展示步骤，用户确认后逐步执行

7. **Given** 长程规划能力
   **When** LLM 分析仓库状态趋势
   **Then** 能提出改进建议（如: 建议添加 CI 检查、建议更新依赖、建议改进代码结构）

### 任务分解

- [ ] T1: 创建 `internal/autonomy/cruise.go` — 巡航引擎（周期扫描、状态收集）
- [ ] T2: 创建 `internal/autonomy/planner.go` — LLM 行动规划器（意图→结构化计划）
- [ ] T3: 创建 `internal/autonomy/guardrails.go` — 安全护栏系统（风险评估、操作分级）
- [ ] T4: 创建 `internal/autonomy/executor.go` — 计划执行器（步骤执行、回滚）
- [ ] T5: 创建 `internal/autonomy/reporter.go` — 巡航报告生成器
- [ ] T6: LLM Tool 定义 — 将 Git/GitHub 操作封装为 LLM 可调用的 Tool
- [ ] T7: 人工审批队列 — 待审批操作的 TUI 展示和操作
- [ ] T8: Chat 集成 — 自然语言指令 → 结构化计划 → 确认 → 执行
- [ ] T9: 巡航配置 — 巡航间隔、风险等级阈值、自动执行范围
- [ ] T10: 巡航状态视图 — 实时展示巡航进度和结果

### 技术要点

- **架构参考**: `symphony` 的 Orchestrator + polling 模式（Elixir GenServer → Go goroutine）
- **Agent 参考**: `openai-agents-python` 的 Agent/Runner/Tool/Guardrail 模式
- **MCP 参考**: `ruflo` 的 MCP Tool Provider 模式
- **安全护栏**: 参考 `openai-agents-python/docs/guardrails.md` — 输入护栏、输出护栏、工具护栏、tripwire
- **人工介入**: 参考 `openai-agents-python` 的 `needs_approval`、`interruptions`、`RunState.approve/reject`
- **操作分级**:
  - 低风险（自动）: 清理已合并分支、更新标签、关闭过期 Issue
  - 中风险（建议）: 合并 PR、创建 Release
  - 高风险（审批）: Force push、删除分支、修改 protected 设置
- **LLM Tool 接口**: 将已有 `gitops.*Manager` 和 `github.Client` 方法封装为 Tool 定义
- **上下文组装**: 巡航前收集 repo 状态 → 构造 system prompt → LLM 分析 → 结构化输出解析

### 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/autonomy/cruise.go` | 新建 | 巡航引擎 |
| `internal/autonomy/planner.go` | 新建 | LLM 行动规划器 |
| `internal/autonomy/guardrails.go` | 新建 | 安全护栏 |
| `internal/autonomy/executor.go` | 新建 | 计划执行器 |
| `internal/autonomy/reporter.go` | 新建 | 报告生成器 |
| `internal/autonomy/tools.go` | 新建 | LLM Tool 定义 |
| `internal/tui/views/approval_queue.go` | 新建 | 审批队列视图 |
| `internal/tui/views/cruise_status.go` | 新建 | 巡航状态视图 |
| `internal/llm/chat/session.go` | 修改 | 支持 Tool calling |
| `internal/tui/app/app.go` | 修改 | 巡航生命周期管理 |

---

## 依赖关系

```
8.2 (剪贴板) ─── 无依赖，可立即开始
8.1 (LLM 对话) ─── 无依赖，可立即开始
8.3 (仓库发现) ─── 依赖已有 GitHub PAT 配置（Story P3 已完成）
8.4 (仓库操作) ─── 依赖 8.3（需要仓库上下文）
8.5 (自主巡航) ─── 依赖 8.1 + 8.4（需要 LLM + 完整操作能力）
```

## 建议执行顺序

1. **Sprint 1**: 8.2（剪贴板）+ 8.1（LLM 对话）— 并行，基础能力
2. **Sprint 2**: 8.3（仓库发现）— 核心交互流
3. **Sprint 3**: 8.4（仓库操作）— 完整功能
4. **Sprint 4**: 8.5（自主巡航）— 智能化

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| LLM API 延迟影响 TUI 响应性 | 中 | 流式渲染 + 异步 cmd + 超时控制 |
| GitHub API Rate Limit | 高 | 缓存机制 + conditional request (ETag) + GraphQL 批量查询 |
| 自主巡航误操作 | 高 | 三级风险分类 + 人工审批队列 + 操作可回滚 + kill switch |
| 终端粘贴兼容性差异 | 低 | bracketed paste 标准 + 多终端测试矩阵 |
| 大型仓库性能 | 中 | 分页加载 + 虚拟滚动 + 后台预加载 |
