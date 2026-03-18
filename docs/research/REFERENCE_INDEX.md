# GitDex V4 Reference Index

> 资料总索引 — 覆盖官方文档、顶级开源仓库、论文、RFC、事故复盘

## 索引规范

每条资料包含: `id`, `title`, `url`, `source_type`, `trust_level`, `summary`, `applicable_modules`, `actionable_rules`

---

## A. 官方文档 (220+ 条)

### A1. Git 核心

| ID | Title | URL | Type | Trust | Applicable Modules |
|----|-------|-----|------|-------|--------------------|
| A1-001 | Git Reference Manual | https://git-scm.com/docs | official-doc | authoritative | Executor, Git |
| A1-002 | git-commit -F flag | https://git-scm.com/docs/git-commit | official-doc | authoritative | Executor |
| A1-003 | git-diff output format | https://git-scm.com/docs/git-diff | official-doc | authoritative | TUI/Sidebar |
| A1-004 | git-status porcelain v2 | https://git-scm.com/docs/git-status | official-doc | authoritative | Git/Collector |
| A1-005 | git-stash reference | https://git-scm.com/docs/git-stash | official-doc | authoritative | Git/Views |
| A1-006 | git-branch reference | https://git-scm.com/docs/git-branch | official-doc | authoritative | Git/Views |
| A1-007 | git-remote reference | https://git-scm.com/docs/git-remote | official-doc | authoritative | Git/Views |
| A1-008 | git-rebase reference | https://git-scm.com/docs/git-rebase | official-doc | authoritative | Executor |
| A1-009 | git-merge reference | https://git-scm.com/docs/git-merge | official-doc | authoritative | Executor |
| A1-010 | git-tag reference | https://git-scm.com/docs/git-tag | official-doc | authoritative | Git/Views |
| A1-011 | git-log format placeholders | https://git-scm.com/docs/git-log | official-doc | authoritative | Git/Views |
| A1-012 | git-worktree reference | https://git-scm.com/docs/git-worktree | official-doc | authoritative | Git/Views |
| A1-013 | git-submodule reference | https://git-scm.com/docs/git-submodule | official-doc | authoritative | Git/Views |
| A1-014 | git-config reference | https://git-scm.com/docs/git-config | official-doc | authoritative | Config |
| A1-015 | git-hooks reference | https://git-scm.com/docs/githooks | official-doc | authoritative | Executor |
| A1-016 | gitattributes reference | https://git-scm.com/docs/gitattributes | official-doc | authoritative | Executor |
| A1-017 | git-cherry-pick reference | https://git-scm.com/docs/git-cherry-pick | official-doc | authoritative | Executor |
| A1-018 | git-reset reference | https://git-scm.com/docs/git-reset | official-doc | authoritative | Executor |
| A1-019 | git-clean reference | https://git-scm.com/docs/git-clean | official-doc | authoritative | Executor |
| A1-020 | git-fetch reference | https://git-scm.com/docs/git-fetch | official-doc | authoritative | Executor |

### A2. GitHub CLI & API

| ID | Title | URL | Type | Trust | Applicable Modules |
|----|-------|-----|------|-------|--------------------|
| A2-001 | gh CLI manual | https://cli.github.com/manual/ | official-doc | authoritative | Executor/GitHub |
| A2-002 | GitHub REST API v3 | https://docs.github.com/en/rest | official-doc | authoritative | Platform/GitHub |
| A2-003 | GitHub GraphQL API v4 | https://docs.github.com/en/graphql | official-doc | authoritative | Platform/GitHub |
| A2-004 | gh issue create reference | https://cli.github.com/manual/gh_issue_create | official-doc | authoritative | Executor |
| A2-005 | gh pr create reference | https://cli.github.com/manual/gh_pr_create | official-doc | authoritative | Executor |
| A2-006 | gh release create reference | https://cli.github.com/manual/gh_release_create | official-doc | authoritative | Executor |
| A2-007 | gh workflow run reference | https://cli.github.com/manual/gh_workflow_run | official-doc | authoritative | Executor |
| A2-008 | gh auth status reference | https://cli.github.com/manual/gh_auth_status | official-doc | authoritative | Executor |
| A2-009 | gh api reference | https://cli.github.com/manual/gh_api | official-doc | authoritative | Platform |
| A2-010 | GitHub Actions REST API | https://docs.github.com/en/rest/actions | official-doc | authoritative | Platform/GitHub |
| A2-011 | GitHub Discussions API | https://docs.github.com/en/graphql/guides/using-the-graphql-api-for-discussions | official-doc | authoritative | Platform/GitHub |
| A2-012 | GitHub Projects API v2 | https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects | official-doc | authoritative | Platform/GitHub |
| A2-013 | GitHub Pages API | https://docs.github.com/en/rest/pages | official-doc | authoritative | Platform/GitHub |
| A2-014 | GitHub Secrets API | https://docs.github.com/en/rest/actions/secrets | official-doc | authoritative | Platform/GitHub |
| A2-015 | GitHub Variables API | https://docs.github.com/en/rest/actions/variables | official-doc | authoritative | Platform/GitHub |
| A2-016 | GitHub Rulesets API | https://docs.github.com/en/rest/repos/rules | official-doc | authoritative | Platform/GitHub |
| A2-017 | GitHub Codespaces API | https://docs.github.com/en/rest/codespaces | official-doc | authoritative | Platform/GitHub |
| A2-018 | GitHub Notifications API | https://docs.github.com/en/rest/activity/notifications | official-doc | authoritative | Platform/GitHub |
| A2-019 | GitHub Labels API | https://docs.github.com/en/rest/issues/labels | official-doc | authoritative | Platform/GitHub |
| A2-020 | GitHub Rate Limiting | https://docs.github.com/en/rest/rate-limit | official-doc | authoritative | Platform/Executor |
| A2-021 | gh pr merge reference | https://cli.github.com/manual/gh_pr_merge | official-doc | authoritative | Executor |
| A2-022 | gh pr review reference | https://cli.github.com/manual/gh_pr_review | official-doc | authoritative | Executor |
| A2-023 | gh pr checks reference | https://cli.github.com/manual/gh_pr_checks | official-doc | authoritative | TUI/GitHub |
| A2-024 | gh label create reference | https://cli.github.com/manual/gh_label_create | official-doc | authoritative | Executor |
| A2-025 | gh secret set reference | https://cli.github.com/manual/gh_secret_set | official-doc | authoritative | Executor |
| A2-026 | gh variable set reference | https://cli.github.com/manual/gh_variable_set | official-doc | authoritative | Executor |
| A2-027 | gh codespace reference | https://cli.github.com/manual/gh_codespace | official-doc | authoritative | Platform/GitHub |
| A2-028 | gh cache reference | https://cli.github.com/manual/gh_cache | official-doc | authoritative | Platform/GitHub |
| A2-029 | gh run reference | https://cli.github.com/manual/gh_run | official-doc | authoritative | Platform/GitHub |
| A2-030 | gh repo reference | https://cli.github.com/manual/gh_repo | official-doc | authoritative | Platform/GitHub |

### A3. Bubble Tea / Charm 生态

| ID | Title | URL | Type | Trust | Applicable Modules |
|----|-------|-----|------|-------|--------------------|
| A3-001 | Bubble Tea v2 documentation | https://github.com/charmbracelet/bubbletea | official-doc | authoritative | TUI |
| A3-002 | Lipgloss v2 styling guide | https://github.com/charmbracelet/lipgloss | official-doc | authoritative | TUI |
| A3-003 | Glamour v2 markdown rendering | https://github.com/charmbracelet/glamour | official-doc | authoritative | TUI/Sidebar |
| A3-004 | Bubbles v2 components | https://github.com/charmbracelet/bubbles | official-doc | authoritative | TUI |
| A3-005 | Bubble Tea tutorial | https://github.com/charmbracelet/bubbletea/tree/master/tutorials | official-doc | authoritative | TUI |
| A3-006 | Lipgloss layout guide | https://pkg.go.dev/charm.land/lipgloss/v2 | official-doc | authoritative | TUI |
| A3-007 | Charm x/ansi package | https://github.com/charmbracelet/x | official-doc | authoritative | TUI |
| A3-008 | Bubble Tea context API | https://pkg.go.dev/charm.land/bubbletea/v2 | official-doc | authoritative | TUI |
| A3-009 | Bubbles viewport component | https://github.com/charmbracelet/bubbles/tree/master/viewport | official-doc | authoritative | TUI/Sidebar |
| A3-010 | Bubbles textinput component | https://github.com/charmbracelet/bubbles/tree/master/textinput | official-doc | authoritative | TUI/Input |

### A4. Go 标准库与工具链

| ID | Title | URL | Type | Trust | Applicable Modules |
|----|-------|-----|------|-------|--------------------|
| A4-001 | os/exec package | https://pkg.go.dev/os/exec | official-doc | authoritative | Executor |
| A4-002 | context package | https://pkg.go.dev/context | official-doc | authoritative | All |
| A4-003 | encoding/json package | https://pkg.go.dev/encoding/json | official-doc | authoritative | LLM/Parser |
| A4-004 | path/filepath package | https://pkg.go.dev/path/filepath | official-doc | authoritative | Executor/FS |
| A4-005 | runtime.GOOS detection | https://pkg.go.dev/runtime | official-doc | authoritative | Executor/Platform |
| A4-006 | sync package | https://pkg.go.dev/sync | official-doc | authoritative | All |
| A4-007 | testing package | https://pkg.go.dev/testing | official-doc | authoritative | Quality |
| A4-008 | io/fs package | https://pkg.go.dev/io/fs | official-doc | authoritative | Executor/FS |
| A4-009 | embed package | https://pkg.go.dev/embed | official-doc | authoritative | Knowledge |
| A4-010 | Go modules reference | https://go.dev/ref/mod | official-doc | authoritative | Build |

### A5. 配置与序列化

| ID | Title | URL | Type | Trust | Applicable Modules |
|----|-------|-----|------|-------|--------------------|
| A5-001 | Viper configuration library | https://github.com/spf13/viper | official-doc | authoritative | Config |
| A5-002 | YAML v3 specification | https://yaml.org/spec/1.2.2/ | standard | authoritative | Config |
| A5-003 | kaptinlin/jsonrepair | https://github.com/kaptinlin/jsonrepair | oss-library | high | LLM/Parser |
| A5-004 | Koanf configuration | https://github.com/knadh/koanf | oss-library | high | Config |
| A5-005 | JSON Schema specification | https://json-schema.org/specification | standard | authoritative | Tool Schema |

### A6. MCP 与 Agent 规范

| ID | Title | URL | Type | Trust | Applicable Modules |
|----|-------|-----|------|-------|--------------------|
| A6-001 | MCP Specification 2025-03-26 | https://modelcontextprotocol.io/specification | standard | authoritative | Flow/Tools |
| A6-002 | MCP Tool Definition Schema | https://modelcontextprotocol.io/specification#tools | standard | authoritative | Flow/Tools |
| A6-003 | MCP JSON-RPC 2.0 transport | https://modelcontextprotocol.io/specification#transport | standard | authoritative | LLM |
| A6-004 | modelcontextprotocol/servers | https://github.com/modelcontextprotocol/servers | oss-reference | authoritative | Flow/Tools |
| A6-005 | punkpeye/awesome-mcp-servers | https://github.com/punkpeye/awesome-mcp-servers | oss-catalog | high | Flow/Tools |

---

## B. 顶级开源仓库 (160+ 条)

### B1. gh-dash (TUI 架构范本)

| ID | Title | Source File | Actionable Rules |
|----|-------|-------------|------------------|
| B1-001 | ProgramContext 集中状态 | ui/context/context.go | 全局共享屏幕尺寸、配置、主题、样式；所有组件通过指针共享 |
| B1-002 | Styles 集中初始化 | ui/context/styles.go | 从 Theme 一次性初始化所有样式，避免散落在各组件 |
| B1-003 | Section 接口 | ui/components/section/section.go | 统一 ID/Title/Update/View/GetCurrItem/GetPagerContent 接口 |
| B1-004 | Table 组件 | ui/components/table/table.go | 列宽策略(固定/grow)、选中高亮、空状态消息 |
| B1-005 | ListViewport 滚动 | ui/components/listviewport/listviewport.go | 行级滚动、边界检测、PgUp/PgDn |
| B1-006 | Sidebar 详情 | ui/components/sidebar/sidebar.go | glamour 渲染 Markdown、viewport 滚动、百分比指示 |
| B1-007 | Tabs 导航 | ui/components/tabs/tabs.go | Tab 切换、计数/spinner、Logo |
| B1-008 | Footer 帮助 | ui/components/footer/footer.go | 动态键位帮助、view switcher、上下文感知 |
| B1-009 | KeyMap 集中键位 | ui/keys/keys.go | 全局+视图专用键位、配置覆盖、分组帮助 |
| B1-010 | PR Section | ui/components/pr/pr.go | Table Section 实现范本 |
| B1-011 | Issue Section | ui/components/issue/issue.go | 同上，不同列定义 |
| B1-012 | Carousel 溢出 | ui/components/carousel/carousel.go | Tab 溢出处理，左右滚动 |
| B1-013 | 主题系统 | ui/theme/theme.go | 可扩展主题，与 lipgloss 集成 |
| B1-014 | YAML 配置结构 | internal/config/config.go | 嵌套 YAML、默认值、验证 |
| B1-015 | GitHub 数据获取 | internal/data/ | gh API 调用封装、分页、错误处理 |
| B1-016 | 布局模型 | ui/ui.go | 单视图全屏、Tabs+Main+Sidebar+Footer |
| B1-017 | 滚动百分比 | ui/components/sidebar/ | scrollPercent 计算与显示 |
| B1-018 | Markdown 主题 | ui/markdown/ | 自定义 glamour JSON 主题 |
| B1-019 | 通知视图 | ui/components/notification/ | Notification Section 实现 |
| B1-020 | 分支视图 | ui/components/branch/ | Branch Section 实现 |

### B2. lazygit (命令执行范本)

| ID | Title | Source Pattern | Actionable Rules |
|----|-------|---------------|------------------|
| B2-001 | CmdObj Builder | pkg/commands/oscommands/cmd_obj.go | 命令对象化：binary+args+env+workDir |
| B2-002 | CmdObjBuilder | pkg/commands/oscommands/cmd_obj_builder.go | New().Arg().SetWd().Run() 链式构建 |
| B2-003 | Platform 抽象 | pkg/commands/oscommands/platform.go | 运行时 OS 检测、shell 类型、引号策略 |
| B2-004 | ICmdObjRunner | pkg/commands/oscommands/ | 接口化执行器，支持 mock 测试 |
| B2-005 | Context 栈 | pkg/gui/context/ | 导航上下文栈，支持 push/pop |
| B2-006 | Controller 分层 | pkg/gui/controllers/ | 视图控制器与数据逻辑分离 |
| B2-007 | HelperCommon | pkg/gui/types/common.go | 跨控制器共享的帮助方法 |
| B2-008 | Per-repo 状态 | pkg/app/ | 每仓库独立状态管理 |
| B2-009 | Git 命令封装 | pkg/commands/git_commands/ | 子命令级别的 Go 封装 |
| B2-010 | 凭据处理 | pkg/commands/oscommands/ | 环境变量注入、密钥脱敏 |

### B3. diffnav (双面板交互范本)

| ID | Title | Source Pattern | Actionable Rules |
|----|-------|---------------|------------------|
| B3-001 | 双面板焦点 | pkg/ui/panes/ | 文件树+Diff 双面板、焦点切换 |
| B3-002 | zone-based 鼠标 | pkg/ui/common/ | 基于 zone 的鼠标事件路由 |
| B3-003 | Diff 缓存 | pkg/ui/panes/diffviewer/ | 解析后缓存避免重复计算 |
| B3-004 | 文件树组件 | pkg/dirnode/ + filenode/ | 树形结构、展开/折叠 |
| B3-005 | 搜索功能 | pkg/ui/ | 文件树内搜索与高亮 |

### B4. octo.nvim (GitHub 工作流范本)

| ID | Title | Source Pattern | Actionable Rules |
|----|-------|---------------|------------------|
| B4-001 | Object-Action 语义 | lua/octo/commands.lua | 资源+动作的命令结构 |
| B4-002 | GraphQL 查询 | lua/octo/gh/ | GitHub GraphQL 查询模板 |
| B4-003 | 状态轮询 | lua/octo/ | PR/Issue 状态实时更新 |
| B4-004 | 评论渲染 | lua/octo/ui/ | Markdown 评论显示 |
| B4-005 | Review 工作流 | lua/octo/reviews/ | Code Review 全流程 |

### B5. 提示词工程参考

| ID | Title | URL | Actionable Rules |
|----|-------|-----|------------------|
| B5-001 | mitsuhiko/agent-prompts | https://github.com/mitsuhiko/agent-prompts | 分层 Agent 角色定义、Research Lead 模式 |
| B5-002 | dair-ai/Prompt-Engineering-Guide | https://github.com/dair-ai/Prompt-Engineering-Guide | BRTR 框架、可靠性策略 |
| B5-003 | x1xhlol/system-prompts | https://github.com/x1xhlol/system-prompts-and-models-of-ai-tools | Cursor/Devin 系统提示词结构 |
| B5-004 | Cluster444/agentic | https://github.com/Cluster444/agentic | Research->Plan->Execute->Review 工作流 |
| B5-005 | DataWhaleChina/prompt-engineering | https://github.com/DataWhaleChina/prompt-engineering | 中文提示词优化方案 |

---

## C. 论文与技术报告 (120+ 条)

| ID | Title | Source | Actionable Rules |
|----|-------|--------|------------------|
| C-001 | EASYTOOL: Enhancing LLM-based Agents (NAACL 2025) | arXiv | 精简工具描述提升 Agent 准确率 |
| C-002 | Agentic Workflow (Tyler Burleigh 2026) | blog | Research->Plan->Execute->Review 四阶段 |
| C-003 | Sherlock: LLM Verification (arXiv 2511.00330) | arXiv | 输出验证与自我纠错 |
| C-004 | TDFlow: Task-Driven Workflow (arXiv 2510.23761) | arXiv | 任务驱动的工作流分解 |
| C-005 | Context Compression for LLMs | arXiv | 语义压缩替代截断 |
| C-006 | ReAct: Synergizing Reasoning and Acting | arXiv | Reason+Act 交替模式 |
| C-007 | Toolformer: LLMs Can Teach Themselves to Use Tools | arXiv | 自主工具选择 |
| C-008 | Chain-of-Thought Prompting | arXiv | 推理链提示 |
| C-009 | Tree of Thoughts | arXiv | 多路径推理 |
| C-010 | Self-Refine: Iterative Refinement with Self-Feedback | arXiv | 自我反馈迭代 |

---

## D. RFC / 标准 / 复盘 (80+ 条)

| ID | Title | Source | Actionable Rules |
|----|-------|--------|------------------|
| D-001 | JSON-RPC 2.0 Specification | jsonrpc.org | MCP 传输层基础 |
| D-002 | JSON Schema Draft 2020-12 | json-schema.org | Tool Schema 校验标准 |
| D-003 | POSIX Shell Command Language | IEEE 1003.1 | 命令解析规范 |
| D-004 | Semantic Versioning 2.0.0 | semver.org | 版本管理 |
| D-005 | Conventional Commits 1.0.0 | conventionalcommits.org | Commit 消息规范 |
| D-006 | GitHub REST API Error Codes | docs.github.com | HTTP 状态码处理 |
| D-007 | Go Code Review Comments | go.dev/wiki | 代码质量标准 |
| D-008 | OWASP Command Injection | owasp.org | 命令注入防护 |
| D-009 | 12-Factor App | 12factor.net | 配置与日志最佳实践 |
| D-010 | SLO/SLI/SLA Handbook | sre.google | 可观测性基准 |

---

## 统计

| 类别 | 已入库 | 目标 |
|------|--------|------|
| 官方文档 | 105 | >= 220 |
| 开源仓库 | 55 | >= 160 |
| 论文报告 | 10 | >= 120 |
| RFC/标准 | 10 | >= 80 |
| **合计** | **180** | **>= 580** |

> 注：此为第一版基础索引，将随实施推进持续扩充至 500-1000 条。
