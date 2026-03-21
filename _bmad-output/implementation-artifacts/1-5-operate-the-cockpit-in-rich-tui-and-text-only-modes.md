# Story 1.5: Operate the Cockpit in Rich TUI and Text-Only Modes

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a terminal-first operator,
I want the Gitdex cockpit to work in rich TUI and text-only modes with the same core semantics,
so that I can operate Gitdex reliably across machines, shells, and terminal capabilities.

## Acceptance Criteria

1. **Given** supported terminal environments across Windows, Linux, and macOS **When** the operator opens Gitdex in rich TUI or text-only mode **Then** both modes provide the same core operator flows for navigation, repository selection, state viewing, and next-action discovery.

2. **Given** any primary interaction in the cockpit **When** the operator uses only the keyboard **Then** all primary interactions are keyboard operable with visible focus and non-color-only status signaling.

3. **Given** terminal widths of compact (80-99 columns), standard (100-139 columns), and wide (140+ columns) **When** the cockpit is rendered **Then** the cockpit adapts layout without hiding critical task or risk information.

4. **Given** the terminal typography and density configuration **When** titles, panels, dense tables, and annotations are rendered **Then** they remain readable across the supported 80/100/140-column regression matrix.

5. **Given** the `--no-tui` flag or non-TTY stdout **When** Gitdex starts **Then** it runs in text-only mode with the same information hierarchy as rich TUI (paragraph + table + status block + command prompt).

6. **Given** the TUI cockpit is active **When** the operator navigates between views **Then** the Calm Ops Cockpit information architecture is followed: cockpit-first (not chat-first), with main workspace for current object, persistent risk/evidence inspector, and bottom-anchored unified input.

## Tasks / Subtasks

- [x] Task 1: 引入 Bubble Tea v2 / Lipgloss v2 / Bubbles v2 依赖并搭建 TUI 应用骨架 (AC: #1, #5)
  - [x] 1.1 执行 `go get charm.land/bubbletea/v2 charm.land/lipgloss/v2 charm.land/bubbles/v2`，更新 `go.mod`
  - [x] 1.2 创建 `internal/tui/app/app.go`，实现顶层 `tea.Model`：`Init()`, `Update()`, `View()`
  - [x] 1.3 实现 TTY 检测逻辑：使用 `golang.org/x/term` 的 `term.IsTerminal()` 判断是否启用 TUI
  - [x] 1.4 非 TTY 或 `--no-tui` 标志时使用 `tea.WithoutRenderer()` 或直接走文本输出路径
  - [x] 1.5 创建 `internal/cli/command/cockpit.go`，注册 `gitdex cockpit` 命令，启动 TUI 程序
  - [x] 1.6 在 `root.go` 中注册 cockpit 命令

- [x] Task 2: 实现语义设计令牌系统与主题支持 (AC: #2, #4)
  - [x] 2.1 创建 `internal/tui/theme/tokens.go`，定义语义色彩令牌：`neutral/info/success/warning/danger/focus/muted`
  - [x] 2.2 映射终端颜色：Ink (#111827), Slate (#334155), Signal Blue (#2C6BED), Focus Cyan (#0F9FB5), Success Green (#1C8C5E), Warning Amber (#B7791F), Danger Red (#C44536)
  - [x] 2.3 实现暗色/亮色主题切换：使用 `tea.RequestBackgroundColor` + `tea.BackgroundColorMsg` 检测终端背景色
  - [x] 2.4 创建 `internal/tui/theme/styles.go`，使用 Lipgloss v2 定义可复用样式（标题、面板、表格、状态标签、注释）
  - [x] 2.5 实现密度层级：page-title / panel-title / object-title / body / dense-data / annotation
  - [x] 2.6 确保所有状态颜色都配对文本标签或符号，不单独依赖颜色传达信息

- [x] Task 3: 实现响应式布局引擎 (AC: #3, #4)
  - [x] 3.1 创建 `internal/tui/layout/responsive.go`，监听 `tea.WindowSizeMsg` 追踪终端宽高
  - [x] 3.2 定义布局断点：Compact (80-99 列单列) / Standard (100-139 列双列) / Wide (140+ 列三列)
  - [x] 3.3 定义高度断点：Short (<32 行隐藏非必要区域) / Normal (32-51) / Tall (52+)
  - [x] 3.4 实现布局管理器：根据当前断点动态切换面板排列
  - [x] 3.5 Wide 模式：导航/队列 | 主工作区 | 右侧检查器 三列布局
  - [x] 3.6 Standard 模式：主工作区 | 抽屉式检查器 双列布局
  - [x] 3.7 Compact 模式：单列 + 全屏面板切换

- [x] Task 4: 实现 Calm Ops Cockpit 核心面板 (AC: #1, #6)
  - [x] 4.1 创建 `internal/tui/panes/status_pane.go`，复用 `appstate.Assembler` 展示仓库状态摘要
  - [x] 4.2 创建 `internal/tui/panes/risk_pane.go`，显示 material risks 和 next actions 列表
  - [x] 4.3 创建 `internal/tui/panes/nav_pane.go`，实现仓库选择和导航队列
  - [x] 4.4 创建 `internal/tui/panes/input_pane.go`，实现底部锚定的统一输入区（命令+聊天统一入口）
  - [x] 4.5 各面板实现 `tea.Model` 接口，支持独立的 `Init/Update/View` 生命周期
  - [x] 4.6 面板间通过 `tea.Cmd` 消息传递状态变更，不直接耦合

- [x] Task 5: 实现键盘导航与焦点管理 (AC: #2)
  - [x] 5.1 创建 `internal/tui/keymap/keymap.go`，定义全局和面板级键绑定
  - [x] 5.2 实现焦点环：Tab/Shift+Tab 在面板间轮转焦点，当前焦点面板高亮边框
  - [x] 5.3 面板内使用方向键/j/k 导航列表项
  - [x] 5.4 实现 `?` 快捷键显示当前可用键绑定帮助覆盖层
  - [x] 5.5 实现 `q` / `Ctrl+C` 退出，`Esc` 返回上级
  - [x] 5.6 确保所有焦点状态有可见的非颜色指示器（如 `>` 前缀、`[*]` 标记、边框加粗）

- [x] Task 6: 实现文本模式输出（text-only parity）(AC: #1, #5)
  - [x] 6.1 创建 `internal/tui/presenter/text_presenter.go`，将 `RepoSummary` 渲染为结构化纯文本
  - [x] 6.2 文本模式信息层级：状态摘要段落 → 维度状态表 → 风险列表 → 下一步动作 → 命令提示
  - [x] 6.3 复用 `internal/cli/output` 的 `WriteValue` 进行 JSON/YAML 结构化输出
  - [x] 6.4 非 TTY 检测自动切换到文本模式，`--no-tui` 标志强制文本模式
  - [x] 6.5 确保文本模式与 TUI 模式输出相同的状态标签、风险信息和 next actions

- [x] Task 7: 补齐测试与回归验证 (AC: #1-#6)
  - [x] 7.1 `internal/tui/app/app_test.go`：TUI 应用初始化和消息处理测试
  - [x] 7.2 `internal/tui/theme/tokens_test.go`：令牌映射和主题切换测试
  - [x] 7.3 `internal/tui/layout/responsive_test.go`：断点计算和布局选择测试
  - [x] 7.4 `internal/tui/keymap/keymap_test.go`：键绑定映射和冲突检测测试
  - [x] 7.5 `internal/tui/presenter/text_presenter_test.go`：文本模式输出完整性测试
  - [x] 7.6 `test/integration/cockpit_command_test.go`：cockpit 命令注册和启动测试
  - [x] 7.7 `test/conformance/tui_text_parity_test.go`：TUI 与文本模式信息等价性一致性测试
  - [x] 7.8 执行 `go test ./... -count=1` 全通过 + `golangci-lint run ./...` 零错误

- [x] Task 8: 控制范围，避免提前实现后续能力 (AC: #1-#6)
  - [x] 8.1 本 Story 只实现座舱骨架和状态查看流程，不实现计划审批/任务执行/campaign 矩阵
  - [x] 8.2 不实现 chat 集成的实际 LLM 调用流（输入面板只做 UI 占位）
  - [x] 8.3 不实现守护进程通信或 WebSocket 实时推送

## Dev Notes

### 前序 Story 关键经验

**Story 1.4 经验教训：**
- `RepoRoot` vs `RepositoryRoot` 路径问题：优先使用 `app.RepoRoot`，为空时 fallback 到 `app.Config.Paths.RepositoryRoot`
- 输出纯净性：避免在 JSON 输出路径使用 `fmt.Printf`，rate limit 警告重定向到 `stderr`
- API 命名清晰度：方法名需明确传达语义（如 `EstimateOpenIssueCount` 而非 `ListRecentIssues`）
- 测试覆盖：每个新 API 方法都需要显式测试

**Story 1.3 经验教训：**
- LLM 配置需同时支持环境变量和配置文件，通过 `appFn` 闭包传递 `bootstrap.App`
- `TaskContext` 所有字段需互斥锁保护，提供 getter 方法
- 输出格式优先级：flag → env → config
- Chat JSON 不应包含冗余字段

### 已有基础设施（必须复用）

| 组件 | 路径 | 用途 |
|------|------|------|
| Bootstrap | `internal/app/bootstrap/bootstrap.go` | 应用初始化，提供 `App` 实例 |
| Config | `internal/platform/config/config.go` | 配置加载，含 `IdentityConfig`, `LLMConfig` |
| Output | `internal/cli/output/format.go` | `WriteValue` 进行 JSON/YAML 输出 |
| Status Assembler | `internal/app/state/assembler.go` | 组装 `RepoSummary` |
| Repo Model | `internal/state/repo/model.go` | `RepoSummary`, `StateLabel` 等模型 |
| Git State | `internal/platform/git/state.go` | `ReadLocalState()` |
| GitHub Client | `internal/platform/github/client.go` | GitHub API 查询 |
| Identity | `internal/platform/identity/github_app.go` | GitHub App 身份认证 |
| Session Context | `internal/app/session/context.go` | 共享任务上下文 |
| Input Parser | `internal/cli/input/parser.go` | 输入分类（命令/自然语言） |
| Root Command | `internal/cli/command/root.go` | 命令树根，注册子命令 |

### 架构约束（必须遵守）

1. **TUI 只消费 read models**：`internal/tui/` 只能通过 `internal/app/` 和 `internal/state/` 读取数据，不能直接调用 GitHub API 或 git 操作
2. **CLI/TUI/API 共享同一 read model**：不允许 TUI 产生与 CLI 不同的治理语义
3. **TUI 不直接读写数据库**：所有数据访问通过 app 层 facade
4. **消息传递架构**：面板间通过 Bubble Tea 的 `tea.Cmd` 和 `tea.Msg` 通信，不直接耦合
5. **JSON/YAML 字段命名**：外部字段统一 `snake_case`，时间统一 RFC3339 UTC，枚举统一 `lower_snake_case`

### 技术栈（本 Story 引入）

| 库 | 版本 | 导入路径 | 用途 |
|----|------|----------|------|
| Bubble Tea | v2.0.2 | `charm.land/bubbletea/v2` | TUI 框架 |
| Lipgloss | v2.0.2 | `charm.land/lipgloss/v2` | 终端样式 |
| Bubbles | v2.0.0 | `charm.land/bubbles/v2` | 可复用 TUI 组件 |
| x/term | latest | `golang.org/x/term` | TTY 检测 |

### Bubble Tea v2 关键 API 变更

- `View()` 返回 `tea.View` 而非 `string`：`v := tea.NewView("content"); v.AltScreen = true; return v`
- 键盘事件使用 `tea.KeyPressMsg` / `tea.KeyReleaseMsg`，通过 `msg.String()` 匹配
- 鼠标事件拆分为 `tea.MouseClickMsg`, `tea.MouseWheelMsg` 等
- 使用 `tea.RequestBackgroundColor` + `tea.BackgroundColorMsg` 检测终端背景色
- `tea.WithoutRenderer()` 禁用渲染器（用于非 TTY / 文本模式）
- Bubbles v2 使用 functional options：`viewport.New(viewport.WithWidth(80))`
- Bubbles v2 使用 getter/setter：`SetWidth()`, `Width()` 而非直接字段访问
- 暗色/亮色主题：`help.DefaultStyles(msg.IsDark())`

### 目录结构（本 Story 新建）

```
internal/tui/
├── app/
│   ├── app.go          # 顶层 tea.Model，组合面板
│   └── app_test.go
├── panes/
│   ├── status_pane.go  # 仓库状态摘要面板
│   ├── risk_pane.go    # 风险和下一步动作面板
│   ├── nav_pane.go     # 导航/仓库选择面板
│   └── input_pane.go   # 底部统一输入面板
├── presenter/
│   ├── text_presenter.go      # 文本模式渲染器
│   └── text_presenter_test.go
├── theme/
│   ├── tokens.go       # 语义设计令牌
│   ├── styles.go       # Lipgloss 样式定义
│   └── tokens_test.go
├── layout/
│   ├── responsive.go   # 响应式布局引擎
│   └── responsive_test.go
└── keymap/
    ├── keymap.go       # 键绑定定义
    └── keymap_test.go
```

### 非颜色状态信号映射

| 状态 | 颜色 | 文本标签 | 符号 |
|------|------|----------|------|
| Healthy | Success Green | `[healthy]` | `✓` |
| Drifting | Warning Amber | `[drifting]` | `~` |
| Blocked | Danger Red | `[blocked]` | `!` |
| Degraded | Danger Red | `[degraded]` | `▼` |
| Unknown | Muted/Slate | `[unknown]` | `?` |

### Project Structure Notes

- `internal/tui/` 是架构文档明确定义的 TUI 代码目录，包含 `app/`, `panes/`, `presenter/`, `keymap/` 子目录
- `internal/cli/command/cockpit.go` 是命令入口，连接 CLI 层和 TUI 层
- 文本模式 presenter 放在 `internal/tui/presenter/` 而非 `internal/cli/output/`，因为它属于 TUI 层的降级输出

### References

- [Source: _bmad-output/planning-artifacts/architecture.md — Lines 157-191: TUI foundation]
- [Source: _bmad-output/planning-artifacts/architecture.md — Lines 258-259: rich TUI / text-only 语义等价]
- [Source: _bmad-output/planning-artifacts/architecture.md — Lines 333-348: Operator Experience Plane]
- [Source: _bmad-output/planning-artifacts/architecture.md — Lines 819-924: 项目目录结构]
- [Source: _bmad-output/planning-artifacts/architecture.md — Lines 1310-1314: TUI 组件边界]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR1: 语义设计令牌]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR2: 排版与密度]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR3: 响应式终端布局]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR4: Calm Ops Cockpit 信息架构]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR12: 动作层级]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR15: 导航模式]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR17: 键盘与焦点 / 非颜色信号]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR19: 文本模式等价性]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR20: UX 回归矩阵]
- [Source: _bmad-output/planning-artifacts/epics.md — Story 1.5 验收标准]

## Dev Agent Record

### Agent Model Used

claude-4.6-opus-max-thinking

### Debug Log References

- `go test ./... -count=1` — 29 packages passed, zero failures
- `golangci-lint run ./...` — zero errors (after gofmt fixes)
- `go build ./...` — successful compilation

### Completion Notes List

- **Task 1:** Bubble Tea v2.0.2, Lipgloss v2.0.2, Bubbles v2.0.0 installed via `charm.land/` import paths. TUI app skeleton in `internal/tui/app/app.go` with `Init/Update/View` lifecycle. TTY detection via `golang.org/x/term`. `gitdex cockpit` command with `--no-tui` flag. Non-TTY auto-falls back to text mode.
- **Task 2:** Semantic design tokens in `internal/tui/theme/tokens.go` with 9 colors (Ink, Slate, Cloud, Mist, SignalBlue, FocusCyan, SuccessGrn, WarningAmb, DangerRed). State tokens map with labels and icons. Lipgloss v2 uses `color.Color` type (not `lipgloss.Color` type). Theme detects dark/light via `tea.BackgroundColorMsg`. Density tiers: PageTitle, PanelTitle, ObjectTitle, Body, DenseData, Annotation.
- **Task 3:** Responsive layout in `internal/tui/layout/responsive.go`. Width breakpoints: Compact (0-99), Standard (100-139), Wide (140+). Height: Short (<32), Normal (32-51), Tall (52+). MainWidth/NavWidth/InspectorWidth computed proportionally. ShowNav only in Wide; ShowInspector in Standard+.
- **Task 4:** Four panes: StatusPane, RiskPane, NavPane, InputPane. Each implements Init/Update/View. StatusPane shows dimension table with cursor navigation. RiskPane shows risks and next actions. NavPane shows navigation items with selection. InputPane is bottom-anchored unified input. All panes communicate via `tea.Msg`.
- **Task 5:** Keymap in `internal/tui/keymap/keymap.go`. Global keys: q/Ctrl+C quit, ? help, Tab/Shift+Tab focus cycle, Esc back, r refresh. List keys: j/k/arrows navigate, Enter select. No duplicate key bindings verified by test. Focus ring advances with Tab, skips hidden panes. Non-color focus: `>` prefix, focused border color.
- **Task 6:** Text presenter in `internal/tui/presenter/text_presenter.go`. Renders: repo info → overall label → dimension table → risks → next actions. Uses shared `theme.TokenForState` for consistent icon/label mapping. Fallback to text mode for non-TTY and `--no-tui`. Structured output (JSON/YAML) reuses `clioutput.WriteValue`.
- **Task 7:** 6 test files, 29 packages pass. Tests cover: token mapping, theme switching, breakpoint classification, key bindings, text rendering, TUI/text parity conformance, cockpit command registration.
- **Task 8:** Scope limited to cockpit skeleton and status viewing. No plan approval, task execution, campaign matrix, chat LLM calls, or daemon communication implemented.

### File List

**New files:**
- `internal/tui/app/app.go`
- `internal/tui/app/app_test.go`
- `internal/tui/theme/tokens.go`
- `internal/tui/theme/tokens_test.go`
- `internal/tui/theme/styles.go`
- `internal/tui/layout/responsive.go`
- `internal/tui/layout/responsive_test.go`
- `internal/tui/keymap/keymap.go`
- `internal/tui/keymap/keymap_test.go`
- `internal/tui/panes/status_pane.go`
- `internal/tui/panes/risk_pane.go`
- `internal/tui/panes/nav_pane.go`
- `internal/tui/panes/input_pane.go`
- `internal/tui/presenter/text_presenter.go`
- `internal/tui/presenter/text_presenter_test.go`
- `internal/cli/command/cockpit.go`
- `test/integration/cockpit_command_test.go`
- `test/conformance/tui_text_parity_test.go`

**Modified files:**
- `internal/cli/command/root.go` — register cockpit command
- `go.mod` / `go.sum` — added bubbletea/v2, lipgloss/v2, bubbles/v2, x/term
