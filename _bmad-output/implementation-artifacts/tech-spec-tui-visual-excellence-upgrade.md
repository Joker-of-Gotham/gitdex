---
title: 'Gitdex TUI 极致美化升级'
slug: 'tui-visual-excellence-upgrade'
created: '2026-03-19'
status: 'ready-for-dev'
stepsCompleted: [1, 2, 3, 4]
tech_stack: ['Go 1.26.1', 'Bubble Tea v2.0.2', 'Lipgloss v2.0.2', 'Cobra', 'Viper']
files_to_modify:
  - 'internal/tui/theme/tokens.go'
  - 'internal/tui/theme/styles.go'
  - 'internal/tui/theme/loader.go (NEW)'
  - 'internal/tui/theme/palette.go (NEW)'
  - 'internal/tui/theme/icons.go (NEW)'
  - 'internal/tui/layout/responsive.go'
  - 'internal/tui/layout/columns.go (NEW)'
  - 'internal/tui/app/app.go'
  - 'internal/tui/components/header.go'
  - 'internal/tui/components/composer.go'
  - 'internal/tui/components/spinner.go (NEW)'
  - 'internal/tui/components/progress.go (NEW)'
  - 'internal/tui/components/modal.go (NEW)'
  - 'internal/tui/components/table.go (NEW)'
  - 'internal/tui/components/statusbar.go (NEW)'
  - 'internal/tui/components/cmdpalette.go (NEW)'
  - 'internal/tui/views/view.go'
  - 'internal/tui/views/router.go'
  - 'internal/tui/views/chat.go'
  - 'internal/tui/views/status.go'
  - 'internal/tui/views/cockpit.go (NEW)'
  - 'internal/tui/views/plans.go (NEW)'
  - 'internal/tui/views/tasks.go (NEW)'
  - 'internal/tui/views/evidence.go (NEW)'
  - 'internal/tui/panes/nav_pane.go'
  - 'internal/tui/panes/risk_pane.go'
  - 'internal/tui/panes/inspector.go (NEW)'
  - 'internal/tui/keymap/keymap.go'
code_patterns:
  - 'Elm architecture: Model/Update/View'
  - 'View interface with Init/Update/Render/SetSize/ID/Title'
  - 'Router for view management'
  - 'Theme token system (currently 9 colors, needs expansion)'
  - 'Layout Dimensions with Breakpoint classification'
  - 'FocusArea enum for focus management'
  - 'CommandHandler map for command dispatch'
test_patterns:
  - 'External test packages (*_test)'
  - 'Direct struct construction for testing'
  - 'tea.KeyPressMsg for input simulation'
  - 'String comparison for render output verification'
---

# Tech-Spec: Gitdex TUI 极致美化升级

**Created:** 2026-03-19

## Overview

### Problem Statement

当前 Gitdex TUI 是功能原型级别，存在以下核心问题：

1. **布局单栏**：三栏响应式布局已有计算逻辑（`layout/responsive.go`）但完全未接入渲染，所有视图单列铺满
2. **主题硬编码**：9 个色彩 token 仅覆盖基础语义，组件中 30+ 处直接使用 hex 色值绕过主题系统，无 dark/light 自适应，无用户自定义能力
3. **组件原始**：缺少 Spinner、ProgressBar、Table（lipgloss 原生）、Modal、Tree、CommandPalette 等生产级组件
4. **图标贫乏**：仅使用 6 个基础 Unicode 字符（❯ ● ◆ ✗ ○ ◐），缺乏视觉层次
5. **首屏错误**：默认启动到 Chat 视图而非 UX 规范要求的 Cockpit 仪表盘
6. **交互粗糙**：无焦点动画、无状态过渡、无加载反馈、无操作确认
7. **Panes 闲置**：NavPane、RiskPane、StatusPane 已存在但未接入主应用

### Solution

参照 btop、ratatui、lipgloss、textual、glow 等 10 个顶级 TUI 参考项目的设计模式，执行七大维度的全面升级：

1. **三栏响应式布局** — 接入 `layout.Dimensions` 的 `ShowNav()`/`ShowInspector()`，使用 `lipgloss.JoinHorizontal` 构建 Nav|Main|Inspector 三栏
2. **全 Token 主题 + 可定制** — 扩展至 30+ 语义 token，支持 `AdaptiveColor{Light, Dark}`，内置 5 套风格主题，支持 `~/.config/gitdex/theme.yaml` 用户自定义
3. **生产级组件库** — Spinner、ProgressBar（Blend1D 渐变）、Table（lipgloss table + StyleFunc 行条纹）、Modal（Place 居中 + Layer 合成）、CommandPalette（Ctrl+P 模糊搜索）、StatusBar
4. **Nerd Font 图标体系** — 全面替换为 Nerd Font 图标，状态/文件类型/Git 操作/导航均有专属图标，自动检测降级到 Unicode
5. **Cockpit 首屏** — 新建 CockpitView 作为默认着陆页，融合仓库概览、维度仪表、风险警示、快捷操作
6. **交互打磨** — 焦点边框动画、Spinner 加载、状态过渡（harmonica spring）、键盘快捷键高亮
7. **Panes 集成** — NavPane 接入左栏导航、新建 InspectorPane 右栏检查器

### Scope

**In Scope:**
- 三栏响应式布局（Compact/Standard/Wide 三档）
- 主题系统重构（30+ token、5 套内置主题、dark/light 自适应、用户自定义主题文件）
- 新增 8 个生产级组件（Spinner、ProgressBar、Table、Modal、StatusBar、CommandPalette、Tree、Breadcrumb）
- Nerd Font 图标系统（含 Unicode 降级）
- CockpitView 首屏仪表盘
- PlansView、TasksView、EvidenceView 骨架
- InspectorPane 右栏检查器
- 全部硬编码 hex 色值 Token 化
- 交互动效（Spinner 动画、焦点过渡、渐变进度条）
- 既有组件视觉升级（Header、Composer、ChatView、StatusView）
- 所有新代码的测试文件

**Out of Scope:**
- LLM 集成和实际对话能力
- GitHub API 实际调用
- 后端存储层改动
- Git 命令实际执行
- 网络请求和 Webhook 处理

## Context for Development

### Codebase Patterns

1. **Elm 架构**: 所有 TUI 组件遵循 `Model → Init() → Update(msg) → View()` 模式
2. **View 接口**: `views.View` 接口定义 `Init/Update/Render/SetSize/ID/Title`，Router 管理活跃视图
3. **主题 Token**: `theme.Theme` 持有 `IsDark` 布尔值，通过 `Fg()`/`MutedFg()`/`BorderColor()` 方法返回 `color.Color`
4. **布局分类**: `layout.Classify(w, h)` 返回 `Dimensions`，提供断点、高度类别和各列宽度计算
5. **焦点管理**: `FocusArea` 枚举（`FocusContent`/`FocusComposer`），`toggleFocus()` 切换
6. **命令派发**: `cmdHandlers map[string]CommandHandler`，`/` 前缀识别

### Files to Reference

| File | Purpose |
| ---- | ------- |
| `internal/tui/theme/tokens.go` | 9 个色彩 token + SemanticToken + Theme 结构体 |
| `internal/tui/theme/styles.go` | 18 个预定义 Style + StatusStyle 路由 |
| `internal/tui/layout/responsive.go` | 三档断点 + Dimensions 计算（未接入渲染） |
| `internal/tui/app/app.go` | 主模型，Header+Router+Composer 组合，30+ 处硬编码 hex |
| `internal/tui/components/header.go` | 品牌 + Tab + 提示，8 处硬编码 hex |
| `internal/tui/components/composer.go` | 输入框 + 提交 + 历史，5 处硬编码 hex |
| `internal/tui/views/view.go` | View 接口定义 + 4 个 ViewID 常量 |
| `internal/tui/views/router.go` | 视图路由管理 |
| `internal/tui/views/chat.go` | 消息列表渲染，5 种角色 + 滚动，6 处硬编码 hex |
| `internal/tui/views/status.go` | 仓库状态表格渲染，renderLabel/renderSeverity 硬编码 |
| `internal/tui/panes/nav_pane.go` | 左栏导航（已存在，未接入） |
| `internal/tui/panes/risk_pane.go` | 风险面板（已存在，未接入） |
| `internal/tui/panes/status_pane.go` | 状态面板（已存在，未接入） |
| `internal/tui/keymap/keymap.go` | 全局键绑定 + 列表键绑定 |
| `go.mod` | 依赖清单，需添加 bubbles/spinner、harmonica |

### Technical Decisions

1. **Lipgloss v2 原生组件优先**: Table/List/Tree 使用 `charm.land/lipgloss/v2` 子包（table、list、tree 已内置于 lipgloss v2）
2. **Bubbles Spinner**: 使用 `github.com/charmbracelet/bubbles/v2/spinner` 组件
3. **主题文件格式**: YAML（与项目已有 Viper 一致），路径 `~/.config/gitdex/theme.yaml`
4. **Nerd Font 检测**: 环境变量 `GITDEX_NERD_FONT=1` 或配置文件开关，默认启用 Nerd Font（检测失败时降级 Unicode）
5. **AdaptiveColor**: 使用 `tea.BackgroundColorMsg` 检测 + 每个 token 存储 Light/Dark 双色值
6. **Column Layout**: `lipgloss.JoinHorizontal` 组合三栏，各栏宽度由 `Dimensions` 计算

---

## Implementation Plan

### Phase 1: 主题系统重构（基础层）

- [ ] Task 1: 扩展色彩 Token 和 Palette 系统
  - File: `internal/tui/theme/palette.go` (NEW)
  - Action: 创建 `Palette` 结构体，包含 30+ 语义色彩 token，每个 token 存储 Light 和 Dark 两个 `color.Color` 值
  - Details:
    - 定义 `type Palette struct` 包含以下字段（每个为 `AdaptiveColor` 双值结构）：
      - **基础色**: `Fg`, `MutedFg`, `SubtleFg`, `Bg`, `SurfaceBg`, `ElevatedBg`
      - **品牌色**: `Primary`, `PrimaryMuted`, `Secondary`
      - **语义色**: `Success`, `Warning`, `Danger`, `Info`
      - **焦点**: `FocusBorder`, `FocusBg`
      - **边框**: `Border`, `BorderMuted`, `Divider`
      - **交互**: `Accent`, `AccentMuted`, `Highlight`, `Selection`
      - **特殊**: `DimText`, `CodeBg`, `LinkText`, `Timestamp`
      - **渐变端点**: `GradientStart`, `GradientMid`, `GradientEnd`
    - 定义 `type AdaptiveColor struct { Light, Dark color.Color }`
    - 方法 `(ac AdaptiveColor) Resolve(isDark bool) color.Color`
    - 5 套内置调色板：
      - `DefaultPalette()` — 当前 SignalBlue/FocusCyan 色系（"控制面"风格）
      - `TokyoNightPalette()` — 紫蓝色系冷调
      - `CatppuccinPalette()` — 柔和暖色（Mocha 变体）
      - `DraculaPalette()` — 经典暗色方案
      - `NordPalette()` — 极简北欧蓝色系
  - Notes: 每套内置 Palette 的 Light/Dark 色值必须分别调好，不能只翻转明暗

- [ ] Task 2: 创建 Nerd Font 图标系统
  - File: `internal/tui/theme/icons.go` (NEW)
  - Action: 定义双轨图标集（Nerd Font + Unicode 降级），通过全局开关切换
  - Details:
    - 定义 `type IconSet struct` 包含所有图标字段，分类：
      - **状态**: `Healthy`, `Drifting`, `Blocked`, `Degraded`, `Unknown`, `Running`, `Paused`, `Queued`
      - **Git 操作**: `Branch`, `Commit`, `Merge`, `PullRequest`, `Issue`, `Tag`, `Diff`, `Stash`
      - **文件类型**: `FileCode`, `FileConfig`, `FileDoc`, `FileTest`, `Folder`, `FolderOpen`
      - **导航**: `ChevronRight`, `ChevronDown`, `ArrowUp`, `ArrowDown`, `ArrowLeft`, `ArrowRight`, `Home`, `Back`
      - **UI**: `Spinner` (多帧), `Check`, `Cross`, `Warning`, `Info`, `Question`, `Lock`, `Unlock`, `Eye`, `EyeOff`
      - **面板**: `Dashboard`, `Chat`, `Plan`, `Task`, `Evidence`, `Search`, `Settings`, `Help`
      - **装饰**: `Separator`, `Dot`, `Diamond`, `Star`, `Fire`, `Rocket`
    - `var NerdFontIcons = IconSet{...}` — Nerd Font 图标（例如 Healthy: `""`, Branch: `""`, PullRequest: `""` 等）
    - `var UnicodeIcons = IconSet{...}` — Unicode 降级（例如 Healthy: `"✓"`, Branch: `"⎇"`, PullRequest: `"⇄"` 等）
    - `var Icons = NerdFontIcons` — 全局活跃图标集
    - `func SetNerdFont(enabled bool)` — 切换图标集
    - `func DetectNerdFont() bool` — 检查 `GITDEX_NERD_FONT` 环境变量或配置
  - Notes: Nerd Font 图标码点参考 https://www.nerdfonts.com/cheat-sheet，选用视觉最清晰的变体

- [ ] Task 3: 创建主题加载器
  - File: `internal/tui/theme/loader.go` (NEW)
  - Action: 实现从 YAML 文件加载用户自定义主题的功能
  - Details:
    - 定义 `type ThemeFile struct` 包含可覆盖的 YAML 字段：
      ```yaml
      name: "My Theme"
      colors:
        fg: { light: "#111827", dark: "#F8FAFC" }
        muted_fg: { light: "#334155", dark: "#E5E7EB" }
        primary: { light: "#2C6BED", dark: "#5B8DEF" }
        # ... 所有 Palette 字段
      icons:
        nerd_font: true
      ```
    - `func LoadThemeFile(path string) (*Palette, error)` — 读取 YAML，合并到 DefaultPalette 上（用户只需覆盖想改的字段）
    - `func DefaultThemePath() string` — 返回 `~/.config/gitdex/theme.yaml`
    - `func ThemeSearchPaths() []string` — 搜索路径：`$GITDEX_THEME`, `~/.config/gitdex/theme.yaml`, `./gitdex-theme.yaml`
    - 使用 `go.yaml.in/yaml/v3` 解析（已在 go.mod 中）
  - Notes: 颜色值支持 hex 格式 `"#RRGGBB"` 和命名引用 `"@primary"`

- [ ] Task 4: 重构 Theme 和 Styles 以使用 Palette
  - File: `internal/tui/theme/tokens.go` (MODIFY)
  - File: `internal/tui/theme/styles.go` (MODIFY)
  - Action: 将 Theme 和 Styles 建立在 Palette 之上，消除硬编码色值
  - Details:
    - **tokens.go 改动**：
      - `Theme` 结构体添加 `Palette Palette` 字段
      - 保留 `IsDark bool`
      - `Fg()` 改为 `return t.Palette.Fg.Resolve(t.IsDark)`
      - `MutedFg()` 改为 `return t.Palette.MutedFg.Resolve(t.IsDark)`
      - `BorderColor()` 改为 `return t.Palette.Border.Resolve(t.IsDark)`
      - `FocusBorderColor()` 改为 `return t.Palette.FocusBorder.Resolve(t.IsDark)`
      - 新增方法：`Primary()`, `Success()`, `Warning()`, `Danger()`, `Info()`, `Accent()`, `Surface()`, `Elevated()`, `Divider()`, `DimText()`, `CodeBg()`, `GradientColors() (start, mid, end)`
      - `NewTheme(isDark bool)` 改为 `NewTheme(isDark bool, palette ...Palette)` 可选 Palette 参数
      - `StateTokens` 的颜色改为引用 Palette 而非全局变量
      - 删除旧的顶级 `var Ink, Slate, Cloud, Mist, SignalBlue, FocusCyan, SuccessGrn, WarningAmb, DangerRed`（色值迁移到各 Palette 中）
    - **styles.go 改动**：
      - `NewStyles(t Theme)` 内部所有颜色引用改为 `t.Primary()`, `t.Success()` 等方法
      - 新增 Styles 字段：
        - `Surface lipgloss.Style` — 面板底色
        - `Elevated lipgloss.Style` — 弹出层底色
        - `Divider lipgloss.Style` — 分割线
        - `CodeBlock lipgloss.Style` — 代码块背景
        - `Link lipgloss.Style` — 超链接
        - `Timestamp lipgloss.Style` — 时间戳
        - `Badge lipgloss.Style` — 徽章/标签
        - `BadgeSuccess/Warning/Danger lipgloss.Style` — 语义徽章
        - `GradientBar func(width int) string` — 渐变进度条渲染器
      - `GradientBar` 实现使用 `lipgloss.Blend1D(width, palette.GradientStart, palette.GradientMid, palette.GradientEnd)`
  - Notes: 此任务完成后所有组件均可通过 `Theme.XXX()` 获取色值，不再需要直接引用 hex

### Phase 2: 图标系统 + 布局基础设施

- [ ] Task 5: 扩展布局系统以支持三栏渲染
  - File: `internal/tui/layout/columns.go` (NEW)
  - File: `internal/tui/layout/responsive.go` (MODIFY)
  - Action: 添加三栏布局渲染辅助函数
  - Details:
    - **columns.go (NEW)**:
      - `type ColumnLayout struct { Nav, Main, Inspector string }` — 持有三栏渲染结果
      - `func RenderColumns(dims Dimensions, nav, main, inspector string) string` — 使用 `lipgloss.JoinHorizontal(lipgloss.Top, ...)` 组合三栏
      - 当 `dims.ShowNav()` 为 false 时不渲染 Nav 栏
      - 当 `dims.ShowInspector()` 为 false 时不渲染 Inspector 栏
      - 每栏使用 `lipgloss.NewStyle().Width(w).Height(h)` 约束尺寸
      - Nav 栏右边框：`BorderRight(true).BorderStyle(lipgloss.NormalBorder())`
      - Inspector 栏左边框：`BorderLeft(true).BorderStyle(lipgloss.NormalBorder())`
    - **responsive.go 改动**:
      - 新增 `func (d Dimensions) HeaderHeight() int` 返回 2（品牌行 + 分割线）
      - 新增 `func (d Dimensions) StatusBarHeight() int` 返回 1
      - 新增 `func (d Dimensions) ComposerHeight() int` 返回 3
      - 修改 `ContentHeight()` 为 `Height - HeaderHeight() - StatusBarHeight() - ComposerHeight()`
      - 新增 `func (d Dimensions) AvailableMainWidth() int` — 考虑边框后的主区域净宽

- [ ] Task 6: 扩展键绑定系统
  - File: `internal/tui/keymap/keymap.go` (MODIFY)
  - Action: 添加新视图和组件所需的键绑定
  - Details:
    - 新增键绑定：
      - `CmdPalette` — `ctrl+p` — 打开命令面板
      - `SwitchCockpit` — `f1` — 切换到 Cockpit
      - `SwitchChat` — `f2` — 切换到 Chat（原 f1）
      - `SwitchStatus` — `f3` — 切换到 Status（原 f2）
      - `SwitchPlans` — `f4` — 切换到 Plans
      - `SwitchTasks` — `f5` — 切换到 Tasks
      - `FocusNav` — `ctrl+1` — 聚焦导航栏
      - `FocusMain` — `ctrl+2` — 聚焦主区域
      - `FocusInspector` — `ctrl+3` — 聚焦检查器
      - `ToggleInspector` — `ctrl+i` — 显示/隐藏检查器
      - `CycleTheme` — `ctrl+t` — 循环切换内置主题
    - 更新 `GlobalKeys` 结构体包含新字段
    - 更新 `DefaultGlobalKeys()` 返回新键绑定

### Phase 3: 新增组件库

- [ ] Task 7: 创建 Spinner 组件
  - File: `internal/tui/components/spinner.go` (NEW)
  - Action: 封装 Bubbles spinner 为 Gitdex 风格的加载指示器
  - Details:
    - `type Spinner struct` 包含 `bubbles/spinner.Model`、`label string`、`theme *theme.Theme`
    - `func NewSpinner(t *theme.Theme, label string) Spinner`
    - 默认使用 `spinner.Dot` 样式，颜色取 `t.Primary()`
    - `Update(msg tea.Msg) (Spinner, tea.Cmd)` 委托给内部 spinner
    - `Render() string` 渲染为 `"⠋ Loading..."` 格式，spinner 颜色 = Primary，label 颜色 = MutedFg
    - `SetLabel(s string)` 动态更新标签
    - 提供预设: `NewSpinnerDot`, `NewSpinnerLine`, `NewSpinnerPulse`

- [ ] Task 8: 创建 ProgressBar 组件
  - File: `internal/tui/components/progress.go` (NEW)
  - Action: 创建带渐变色的进度条组件
  - Details:
    - `type ProgressBar struct` 包含 `percent float64`、`width int`、`theme *theme.Theme`、`label string`
    - `func NewProgressBar(t *theme.Theme) *ProgressBar`
    - `SetPercent(p float64)` — 0.0~1.0
    - `SetWidth(w int)` — 进度条像素宽度
    - `Render() string`:
      - 使用 `lipgloss.Blend1D()` 在 `GradientStart → GradientMid → GradientEnd` 之间渐变
      - 填充字符: `█` (Nerd Font) 或 `▓` (Unicode 降级)
      - 空白字符: `░`
      - 右侧显示百分比 `"67%"`
      - 格式: `[████████░░░░] 67%`
    - 可选标签: `SetLabel("Scanning...")` 渲染在进度条上方

- [ ] Task 9: 创建 Table 组件包装器
  - File: `internal/tui/components/table.go` (NEW)
  - Action: 封装 lipgloss table 为主题感知的表格组件
  - Details:
    - `type Table struct` 包含 `headers []string`、`rows [][]string`、`theme *theme.Theme`、`width int`
    - `func NewTable(t *theme.Theme, headers ...string) *Table`
    - `AddRow(cells ...string)` — 添加行
    - `SetRows(rows [][]string)` — 批量设置
    - `Render() string`:
      - 使用 `charm.land/lipgloss/v2/table` 包
      - `table.New().Border(lipgloss.RoundedBorder()).Headers(...).Rows(...).StyleFunc(...)`
      - HeaderRow: Bold, Primary 前景色, Surface 背景色
      - 偶数行: 正常前景色
      - 奇数行: 略微不同的背景 (`SurfaceBg`)
      - Border 颜色取 `Divider`
    - 支持列宽自适应和固定宽度混合模式

- [ ] Task 10: 创建 Modal/Overlay 组件
  - File: `internal/tui/components/modal.go` (NEW)
  - Action: 创建居中弹出层组件
  - Details:
    - `type Modal struct` 包含 `title string`、`content string`、`width, height int`、`theme *theme.Theme`、`visible bool`
    - `func NewModal(t *theme.Theme, title string) *Modal`
    - `Show(content string)` / `Hide()`
    - `IsVisible() bool`
    - `Update(msg tea.Msg) tea.Cmd` — Escape 键关闭
    - `Render(behindContent string) string`:
      - 模态框使用 `Elevated` 背景色 + `FocusBorder` 边框 + `RoundedBorder`
      - 标题栏: `Primary` 前景色 + Bold，居中
      - 使用 `lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, box)` 居中
      - 背景内容使用 `lipgloss.NewStyle().Faint(true)` 降低对比度
    - 预设: `ConfirmModal(title, message string, onYes, onNo func())` 带 Y/N 按钮

- [ ] Task 11: 创建 StatusBar 组件
  - File: `internal/tui/components/statusbar.go` (NEW)
  - Action: 创建底部状态栏（替代当前简单的 hints 行）
  - Details:
    - `type StatusBar struct` 包含 `theme *theme.Theme`、`width int`、`mode string`、`branch string`、`repoState string`、`themeName string`
    - `func NewStatusBar(t *theme.Theme) *StatusBar`
    - `SetMode(m string)` — "NORMAL", "INSERT", "COMMAND"
    - `SetBranch(b string)` — 当前分支名
    - `SetRepoState(s string)` — 仓库状态标签
    - `SetThemeName(n string)` — 当前主题名
    - `Render() string`:
      - 全宽单行，背景色 `SurfaceBg`
      - 左侧: 模式指示器 `[NORMAL]`（背景色按模式变化: NORMAL=Primary, INSERT=Success, COMMAND=Warning）
      - 左中: 分支图标 + 分支名 `  main`
      - 右中: 仓库状态 ` healthy`
      - 右侧: 主题名 + Nerd Font 检测状态
      - 使用 `lipgloss.JoinHorizontal` + 间隔填充

- [ ] Task 12: 创建 CommandPalette 组件
  - File: `internal/tui/components/cmdpalette.go` (NEW)
  - Action: 创建 Ctrl+P 模糊搜索命令面板
  - Details:
    - `type CmdPalette struct` 包含 `input []rune`、`cursor int`、`items []PaletteItem`、`filtered []PaletteItem`、`selected int`、`visible bool`、`theme *theme.Theme`
    - `type PaletteItem struct { Label, Description, Shortcut string; Action func() tea.Cmd }`
    - `func NewCmdPalette(t *theme.Theme) *CmdPalette`
    - `AddItem(item PaletteItem)` — 注册可搜索项
    - `Show()` / `Hide()` / `IsVisible() bool`
    - `Update(msg tea.Msg) tea.Cmd`:
      - 字符输入: 更新搜索文本，重新过滤
      - Up/Down: 在过滤结果中导航
      - Enter: 执行选中项的 Action
      - Escape: 关闭面板
    - `Render() string`:
      - 顶部搜索框: `🔍 ` + 输入文本 + 光标
      - 结果列表: 最多显示 10 项
      - 选中项: `Selection` 背景色高亮
      - 每项显示: 图标 + Label + Description (右对齐) + Shortcut (DimText)
      - 整体使用 Modal 风格居中渲染
    - 模糊匹配: 简单子串匹配（`strings.Contains` 忽略大小写），匹配字符高亮

### Phase 4: 视图升级与新建

- [ ] Task 13: 创建 CockpitView（首屏仪表盘）
  - File: `internal/tui/views/cockpit.go` (NEW)
  - Action: 创建仓库运维仪表盘作为默认着陆视图
  - Details:
    - `type CockpitView struct` 包含 `summary *repo.RepoSummary`、`width, height int`、`theme *theme.Theme`、`table *components.Table`
    - `ID() = ViewCockpit`, `Title() = "Cockpit"`
    - `Render()`:
      - **顶部横幅**: 仓库名 `Owner/Repo` + 整体状态徽章 + 最后扫描时间
      - **维度仪表板**: 使用 `components.Table` 显示 5 个维度，每行包含：
        - Nerd Font 图标 + 维度名
        - 状态标签（彩色徽章）
        - 详情文本
        - 迷你进度指示（healthy=满条绿色, drifting=半条黄色, blocked=红色叉）
      - **风险摘要区**: 按严重度排列风险项，使用 `BadgeDanger`/`BadgeWarning` 渲染严重度
      - **快捷操作区**: 3-4 个常用操作按钮（`[S]can`, `[P]ull`, `[D]octor`, `[H]elp`），使用 `PrimaryAction` 样式
      - **空状态**: 无数据时显示大号图标 + "运行 `gitdex scan` 开始首次仓库扫描" + 提示
    - `Update()`: 处理 `StatusDataMsg` 更新 summary
    - 使用 `lipgloss.JoinVertical` 组合各区块

- [ ] Task 14: 更新 ViewID 常量和 Router
  - File: `internal/tui/views/view.go` (MODIFY)
  - File: `internal/tui/views/router.go` (MODIFY)
  - Action: 添加新视图 ID 并扩展 Router
  - Details:
    - **view.go**: 添加 `ViewCockpit ID = "cockpit"`、`ViewEvidence ID = "evidence"`
    - **router.go**: 无结构改动，但 Router 需支持 5+ 视图的 Tab 显示

- [ ] Task 15: 创建 PlansView 骨架
  - File: `internal/tui/views/plans.go` (NEW)
  - Action: 创建执行计划列表视图的结构框架
  - Details:
    - `type PlansView struct` 包含 `plans []PlanSummary`、`selected int`、`width, height int`、`theme *theme.Theme`
    - `type PlanSummary struct { Title, Status, Scope string; StepCount, CompletedSteps int; RiskLevel string }`
    - `ID() = ViewPlans`, `Title() = "Plans"`
    - `Render()`:
      - 空状态: "没有活跃的执行计划。使用自然语言描述你的意图来创建计划。"
      - 有数据: 列表显示每个计划的标题、状态徽章、进度条、范围描述
      - 选中项展开显示步骤列表
    - `Update()`: Up/Down 导航，Enter 展开/折叠

- [ ] Task 16: 创建 TasksView 骨架
  - File: `internal/tui/views/tasks.go` (NEW)
  - Action: 创建任务列表视图的结构框架
  - Details:
    - `type TasksView struct` 包含 `tasks []TaskItem`、`selected int`、`width, height int`、`theme *theme.Theme`
    - `type TaskItem struct { ID, Title, Status, AssignedPlan string; Priority int }`
    - `ID() = ViewTasks`, `Title() = "Tasks"`
    - `Render()`:
      - 空状态: "没有排队中的任务。"
      - 有数据: 使用 `components.Table` 显示任务列表（ID、标题、状态、所属计划）
      - 状态图标: queued=⏳, running=🔄, done=✓, failed=✗

- [ ] Task 17: 创建 EvidenceView 骨架
  - File: `internal/tui/views/evidence.go` (NEW)
  - Action: 创建证据/审计日志视图的结构框架
  - Details:
    - `type EvidenceView struct` 包含 `entries []EvidenceEntry`、`selected int`、`width, height int`、`theme *theme.Theme`
    - `type EvidenceEntry struct { Timestamp time.Time; Action, Result, Detail string; Success bool }`
    - `ID() = ViewEvidence`, `Title() = "Evidence"`
    - `Render()`:
      - 空状态: "没有执行记录。"
      - 有数据: 时间线样式列表，每条记录显示时间戳 + 操作名 + 结果徽章

### Phase 5: 既有组件视觉升级 + Token 化

- [ ] Task 18: 升级 Header 组件
  - File: `internal/tui/components/header.go` (MODIFY)
  - Action: 消除硬编码色值，添加 Nerd Font 品牌图标，增强 Tab 样式
  - Details:
    - 接收 `*theme.Theme` 参数: `func NewHeader(t *theme.Theme) *Header`
    - 品牌: ` Gitdex ` 使用 `Icons.Dashboard` + `t.Primary()` 前景 + Bold
    - Tab 渲染: 活跃 tab 使用 `t.Primary()` 背景 + `Fg(对比色)` 前景 + 圆角边框
    - 非活跃 tab: `t.MutedFg()` + 对应 Nerd Font 图标（`Icons.Dashboard`, `Icons.Chat`, `Icons.Plan`, `Icons.Task`, `Icons.Evidence`）
    - 右侧提示: `t.DimText()` 前景色
    - 底部分割线: `t.Divider()` 颜色
    - Tab 之间添加 `│` 分隔符，颜色 `t.Divider()`

- [ ] Task 19: 升级 Composer 组件
  - File: `internal/tui/components/composer.go` (MODIFY)
  - Action: 消除硬编码色值，添加状态指示器和视觉增强
  - Details:
    - 接收 `*theme.Theme`: `func NewComposer(t *theme.Theme) *Composer`
    - Prompt 图标: 使用 `Icons.ChevronRight` 替代 `❯`，颜色 `t.FocusBorder()`
    - 聚焦边框: `t.FocusBorder()` + `RoundedBorder()`
    - 非聚焦边框: `t.Border()` + `RoundedBorder()`
    - Placeholder: `t.DimText()` + Italic
    - 光标: 使用 `t.Accent()` 背景色 + 闪烁效果（通过 `tea.CursorBlinkMode`）
    - 添加左侧模式指示器: 命令模式显示 `/` 图标（`t.Warning()` 色），聊天模式显示 `💬` 图标（`t.Info()` 色）
    - 输入文本前景色: `t.Fg()`

- [ ] Task 20: 升级 ChatView
  - File: `internal/tui/views/chat.go` (MODIFY)
  - Action: 消除硬编码色值，使用 Nerd Font 图标和主题 Styles
  - Details:
    - 接收 `*theme.Theme`: `func NewChatView(t *theme.Theme) *ChatView`
    - 角色图标替换:
      - User: `Icons.ChevronRight` + `t.Primary()` + Bold
      - Assistant: `Icons.Diamond` + `t.Success()`
      - System: `Icons.Info` + `t.Info()`
      - Info: `Icons.Info` + `t.MutedFg()`
      - Error: `Icons.Cross` + `t.Danger()` + Bold
    - 时间戳: `t.Timestamp()` 颜色
    - 消息文本: `t.Fg()` 前景色
    - 消息之间添加细分隔线: `t.Divider()` 颜色的 `─` 字符
    - 代码块检测: ``` 包裹的内容使用 `t.CodeBg()` 背景

- [ ] Task 21: 升级 StatusView
  - File: `internal/tui/views/status.go` (MODIFY)
  - Action: 消除硬编码色值，改用 `components.Table` 和主题 Styles
  - Details:
    - 接收 `*theme.Theme`: `func NewStatusView(t *theme.Theme) *StatusView`
    - 仓库标题: `Icons.Branch` + `t.Primary()` + Bold
    - 状态标签: 使用 `theme.Styles.BadgeSuccess/Warning/Danger` 渲染
    - 维度表格: 使用 `components.Table` 替代手写 `Sprintf` 格式化
    - `renderLabel`: 颜色引用 `t.Success()`/`t.Warning()`/`t.Danger()`/`t.MutedFg()`
    - `renderSeverity`: 颜色引用 `t.Danger()`/`t.Warning()`/`t.MutedFg()`
    - 分割线: `t.Divider()` 颜色
    - 空状态: 使用 `Icons.Question` + `t.DimText()` + 引导文案

- [ ] Task 22: 升级 NavPane（接入主应用）
  - File: `internal/tui/panes/nav_pane.go` (MODIFY)
  - Action: 适配新主题系统，准备接入三栏布局
  - Details:
    - 接收 `*theme.Theme`: `func NewNavPane(t *theme.Theme, items []NavItem) *NavPane`
    - 每个导航项添加 Nerd Font 图标
    - 选中项: `t.Accent()` 背景 + `Icons.ChevronRight` 前缀
    - 未选中项: `t.Fg()` 前景
    - 标题: `t.Primary()` + Bold + `Icons.Home` 图标
    - 组分隔线: `t.Divider()` 颜色的虚线

- [ ] Task 23: 创建 InspectorPane（右栏检查器）
  - File: `internal/tui/panes/inspector.go` (NEW)
  - Action: 创建右栏检查器面板
  - Details:
    - `type InspectorPane struct` 包含 `mode InspectorMode`、`riskPane *RiskPane`、`theme *theme.Theme`、`width, height int`、`visible bool`
    - `type InspectorMode int` — `ModeRisk`, `ModeEvidence`, `ModeAudit`, `ModeDetail`
    - `func NewInspectorPane(t *theme.Theme) *InspectorPane`
    - `SetMode(m InspectorMode)` — 切换检查器内容
    - `Toggle()` — 显示/隐藏
    - `Render()`:
      - 标题栏: 模式名称 + 图标
      - ModeRisk: 委托给 `RiskPane.View()`
      - ModeEvidence: 显示最近的执行证据
      - ModeAudit: 显示审计日志
      - ModeDetail: 显示选中项的详细信息
    - 底部快捷键提示: `[1]Risk [2]Evidence [3]Audit`

### Phase 6: 主应用重构

- [ ] Task 24: 重构 app.go — 三栏布局 + 新组件集成
  - File: `internal/tui/app/app.go` (MODIFY)
  - Action: 全面重构主应用以使用三栏布局、新组件和主题系统
  - Details:
    - **Model 结构体改动**:
      - 添加字段: `statusBar *components.StatusBar`、`cmdPalette *components.CmdPalette`、`navPane *panes.NavPane`、`inspectorPane *panes.InspectorPane`、`cockpitView *views.CockpitView`、`plansView *views.PlansView`、`tasksView *views.TasksView`、`evidenceView *views.EvidenceView`、`currentPaletteName string`、`paletteIndex int`
      - 扩展 FocusArea: `FocusNav`, `FocusContent`, `FocusComposer`, `FocusInspector`, `FocusPalette`
    - **New() 改动**:
      - 检测 Nerd Font: `theme.DetectNerdFont()`
      - 加载用户主题: `theme.LoadThemeFile(theme.DefaultThemePath())`
      - 默认 Palette: `theme.DefaultPalette()`
      - CockpitView 作为默认视图: `router := views.NewRouter(views.ViewCockpit, cockpitView, chatView, statusView, plansView, tasksView, evidenceView)`
      - 初始化 NavPane: 导航项 = `[Cockpit, Chat, Status, Plans, Tasks, Evidence]`
      - 初始化 StatusBar、CmdPalette、InspectorPane
      - 注册 CommandPalette 项: 所有 `/` 命令 + 视图切换 + 主题切换
    - **Update() 改动**:
      - 新增 `tea.KeyPressMsg` 分支:
        - `ctrl+p`: 显示/隐藏 CommandPalette
        - `ctrl+t`: 循环切换内置主题（Default → TokyoNight → Catppuccin → Dracula → Nord → ...）
        - `ctrl+i`: 切换 Inspector 可见性
        - `ctrl+1/2/3`: 切换焦点到 Nav/Main/Inspector
        - `f1~f5`: 切换视图（Cockpit/Chat/Status/Plans/Tasks）
      - CmdPalette 可见时优先处理其 Update
      - NavPane 选择事件触发 `SwitchViewMsg`
    - **View() / renderApp() 改动**:
      - 使用 `layout.RenderColumns(dims, navStr, mainStr, inspStr)` 替代单列渲染
      - Nav 栏: `navPane.View()` (仅 Wide 模式)
      - Main 栏: `router.Render()`
      - Inspector 栏: `inspectorPane.Render()` (Standard+Wide 模式)
      - 底部: `composer.Render()` + `statusBar.Render()`（替代旧 `renderHints`）
      - CmdPalette 可见时叠加渲染
    - **消除所有硬编码 hex**:
      - `renderHints()` 整体由 StatusBar 替代
      - `renderHelp()` 使用主题 Styles
      - 删除所有 `lipgloss.Color("#xxxxxx")` 直接引用

- [ ] Task 25: 添加主题循环切换命令
  - File: `internal/tui/app/app.go` (within Task 24)
  - Action: 注册 `/theme` 命令，支持主题浏览和切换
  - Details:
    - `/theme` — 列出所有可用主题
    - `/theme <name>` — 切换到指定主题
    - `Ctrl+T` — 快捷循环切换
    - 切换后：重建 Styles，刷新所有组件的 theme 引用

### Phase 7: 测试 + 回归验证

- [ ] Task 26: 为所有新文件添加测试
  - Files: 每个新增 .go 文件的对应 `*_test.go`
  - Action: 创建全面的单元测试
  - Details:
    - `theme/palette_test.go`: 测试 5 套 Palette 的 Resolve(dark/light)、所有字段非 nil
    - `theme/icons_test.go`: 测试 NerdFont/Unicode 双集合完整性、`SetNerdFont` 切换
    - `theme/loader_test.go`: 测试 YAML 加载、字段覆盖、无效路径错误
    - `layout/columns_test.go`: 测试三栏渲染在 Compact/Standard/Wide 下的输出
    - `components/spinner_test.go`: 测试 Render 非空、SetLabel、Update 委托
    - `components/progress_test.go`: 测试 0%/50%/100% 渲染、宽度自适应
    - `components/table_test.go`: 测试 Header/Rows/Render、空表
    - `components/modal_test.go`: 测试 Show/Hide/IsVisible、Escape 关闭
    - `components/statusbar_test.go`: 测试 SetMode/SetBranch/Render 宽度
    - `components/cmdpalette_test.go`: 测试 AddItem/过滤/选择/执行
    - `views/cockpit_test.go`: 测试空状态/有数据/Render
    - `views/plans_test.go`: 测试空状态/有数据/导航
    - `views/tasks_test.go`: 测试空状态/有数据
    - `views/evidence_test.go`: 测试空状态/有数据
    - `panes/inspector_test.go`: 测试模式切换/Toggle/Render

- [ ] Task 27: 更新既有测试文件
  - Files: `app/app_test.go`, `components/components_test.go`, `views/views_test.go`, `panes/panes_test.go`, `theme/tokens_test.go`
  - Action: 适配新的构造函数签名（添加 theme 参数）和新的默认视图（Cockpit）
  - Details:
    - `app_test.go`: 更新 `New()` 期望，默认活跃视图改为 `ViewCockpit`，添加三栏布局测试
    - `components_test.go`: Header/Composer 构造函数添加 theme 参数
    - `views_test.go`: ChatView/StatusView 构造函数添加 theme 参数，添加 CockpitView 测试
    - `panes_test.go`: NavPane 构造函数添加 theme 参数
    - `tokens_test.go`: 验证 Palette 集成、新方法

- [ ] Task 28: 全量构建和回归验证
  - Action: 运行完整的构建、格式化和测试验证
  - Details:
    - `go build ./...` — 编译通过
    - `go vet ./...` — 无告警
    - `gofmt -l` — 全部格式正确
    - `go test ./...` — 全部通过，0 FAIL，0 [no test files]
    - 确认 `go test ./internal/tui/...` 所有子包通过

---

## Acceptance Criteria

### 主题系统

- [ ] AC 1: Given 默认配置, when 启动 Gitdex TUI, then 使用 DefaultPalette 渲染，所有颜色取自 Palette token，无硬编码 hex
- [ ] AC 2: Given dark 终端背景, when `tea.BackgroundColorMsg{IsDark: true}` 触发, then 所有颜色自动切换到 Dark 变体
- [ ] AC 3: Given light 终端背景, when `tea.BackgroundColorMsg{IsDark: false}` 触发, then 所有颜色自动切换到 Light 变体
- [ ] AC 4: Given 用户在 `~/.config/gitdex/theme.yaml` 中设置 `colors.primary.dark: "#FF0000"`, when 启动 TUI, then Primary 颜色变为红色
- [ ] AC 5: Given 用户按 `Ctrl+T`, when 当前主题为 Default, then 循环切换到 TokyoNight，所有组件颜色立即刷新
- [ ] AC 6: Given 输入 `/theme dracula`, when 命令执行, then 切换到 Dracula 主题，StatusBar 显示主题名

### 图标系统

- [ ] AC 7: Given `GITDEX_NERD_FONT=1` 环境变量设置, when 启动 TUI, then 所有图标使用 Nerd Font 字符
- [ ] AC 8: Given 未设置 Nerd Font 环境变量, when 启动 TUI, then 所有图标使用 Unicode 降级字符，UI 功能不受影响
- [ ] AC 9: Given Nerd Font 启用, when 查看 Cockpit 视图, then 每个维度行有对应的 Nerd Font 图标前缀

### 布局

- [ ] AC 10: Given 终端宽度 80 列 (Compact), when 渲染 UI, then 单栏显示仅 Main 区域，无 Nav 和 Inspector
- [ ] AC 11: Given 终端宽度 120 列 (Standard), when 渲染 UI, then 双栏显示 Main + Inspector，无 Nav
- [ ] AC 12: Given 终端宽度 160 列 (Wide), when 渲染 UI, then 三栏显示 Nav | Main | Inspector，各栏有垂直分隔边框
- [ ] AC 13: Given 终端窗口从 Wide 缩小到 Compact, when `WindowSizeMsg` 触发, then 布局即时重排为单栏，无渲染错误

### Cockpit 首屏

- [ ] AC 14: Given 无参数启动 `gitdex`, when TUI 加载完成, then 默认显示 Cockpit 视图（非 Chat）
- [ ] AC 15: Given 仓库数据已加载, when 查看 Cockpit, then 显示仓库名 + 状态徽章 + 5 维度仪表 + 风险摘要 + 快捷操作
- [ ] AC 16: Given 无仓库数据, when 查看 Cockpit, then 显示引导文案和操作提示，不显示空白

### 组件

- [ ] AC 17: Given Spinner 组件创建, when `Update(spinner.TickMsg)` 持续触发, then Spinner 图标动画旋转，标签显示在旁
- [ ] AC 18: Given ProgressBar 设置 50%, when `Render()` 调用, then 显示半满的渐变色进度条 + `"50%"` 标签
- [ ] AC 19: Given Table 有 3 列 5 行, when `Render()` 调用, then 使用 lipgloss table 渲染带圆角边框的表格，奇偶行交替样式
- [ ] AC 20: Given Modal 调用 `Show("确认删除?")`, when 渲染, then 弹出层居中显示在背景内容之上，背景变暗
- [ ] AC 21: Given Modal 显示中, when 按 Escape, then Modal 关闭，恢复正常视图
- [ ] AC 22: Given StatusBar 设置 branch="main" state="healthy", when 渲染, then 底部单行显示模式指示器 + 分支名 + 状态 + 主题名
- [ ] AC 23: Given CommandPalette 打开, when 输入 "sta", then 过滤显示包含 "sta" 的命令（如 "status", "start"）
- [ ] AC 24: Given CommandPalette 中选中一项, when 按 Enter, then 执行对应操作并关闭面板

### 交互

- [ ] AC 25: Given Composer 获得焦点, when 边框渲染, then 使用 `FocusBorder` 颜色的圆角边框（区别于非焦点状态）
- [ ] AC 26: Given 按 `Tab` 键, when 焦点在 Composer, then 焦点移到 Content 区域，Composer 边框变为普通色
- [ ] AC 27: Given 按 `F1~F5`, when 任意焦点状态, then 切换到对应视图（Cockpit/Chat/Status/Plans/Tasks）
- [ ] AC 28: Given 按 `Ctrl+P`, when 任意状态, then 打开 CommandPalette 覆盖层
- [ ] AC 29: Given 按 `Ctrl+I`, when Inspector 可见, then Inspector 隐藏；再按恢复

### 测试

- [ ] AC 30: Given 全量测试运行, when `go test ./...` 完成, then 0 FAIL，0 [no test files]
- [ ] AC 31: Given 代码格式检查, when 运行 `gofmt -l`, then 无输出（全部格式正确）
- [ ] AC 32: Given 静态分析, when 运行 `go vet ./...`, then 无告警

---

## Additional Context

### Dependencies

需要添加的新依赖：
- `github.com/charmbracelet/bubbles/v2` — Spinner 组件（如已在 go.mod 则使用子包 `spinner`）
- `github.com/charmbracelet/harmonica` — 弹性动画（可选，用于焦点过渡）
- `charm.land/lipgloss/v2/table` — Table 组件（lipgloss v2 内置子包，无需额外 `go get`）
- `charm.land/lipgloss/v2/list` — List 组件（同上）
- `charm.land/lipgloss/v2/tree` — Tree 组件（同上）
- `go.yaml.in/yaml/v3` — YAML 解析（已在 go.mod 中）

### Testing Strategy

- **单元测试**: 每个新增 .go 文件必须有对应 `*_test.go`，使用外部测试包 (`package xxx_test`)
- **主题测试**: 验证 5 套 Palette 的 `Resolve(true)` 和 `Resolve(false)` 均返回非 nil 颜色
- **图标测试**: 验证 `NerdFontIcons` 和 `UnicodeIcons` 的所有字段非空字符串
- **布局测试**: 验证 `RenderColumns` 在 Compact/Standard/Wide 三档下的输出宽度正确
- **组件测试**: 每个组件测试 `Render()` 返回非空字符串、`SetSize()` 不 panic、关键交互行为
- **回归测试**: `go test ./...` 全量通过，包括既有测试的适配
- **构建验证**: `go build ./...` + `go vet ./...` + `gofmt -l` 全部通过

### Notes

**风险项:**
1. Lipgloss v2 的 `table`/`list`/`tree` 子包 API 可能与预期不同 — 需在 Task 9 实施前确认 import 路径和可用性
2. Bubbles v2 spinner 的 `TickMsg` 类型可能与 Bubble Tea v2 不完全兼容 — 需验证 `spinner.Tick` 命令
3. `lipgloss.Blend1D` 可能不是 v2 中的确切 API — 需查阅 lipgloss v2 文档确认渐变 API

**已知限制:**
- PlansView、TasksView、EvidenceView 在本次规格中仅为骨架，数据填充依赖后续 LLM 和 autonomy 集成
- CommandPalette 使用简单子串匹配，未来可升级为真正的模糊匹配算法
- 主题热加载仅支持 `Ctrl+T` 循环切换，不支持文件变更监听

**实施顺序原则:**
- Phase 1 (主题) 必须最先完成，因为所有后续组件都依赖主题系统
- Phase 2 (布局+键绑定) 是 Phase 6 (主应用重构) 的前置
- Phase 3 (组件) 和 Phase 4 (视图) 可以并行，但 Phase 4 的视图可能使用 Phase 3 的组件
- Phase 5 (既有组件升级) 依赖 Phase 1 (主题)
- Phase 6 (主应用重构) 依赖所有前置 Phase
- Phase 7 (测试) 贯穿所有 Phase，每个 Task 完成后立即编写对应测试
