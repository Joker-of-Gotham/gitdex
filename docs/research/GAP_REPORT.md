# GitDex V4 Gap Report

> GitDex 当前状态与目标架构差距报告

## 差距概览

| 维度 | 当前状态 | 目标状态 | 差距等级 |
|------|---------|---------|---------|
| TUI 架构 | 静态多面板拼接 | 单视图全屏 + Tabs + Sidebar | P0 |
| 命令执行 | 字符串拼接 + exec.Command | CmdObj Builder + Platform + TempFile | P0 |
| JSON 解析 | 直接 Unmarshal | jsonrepair + fallback chain | P0 |
| 工作流 | Maintain/Goal/Creative 三循环 | Research->Plan->Execute->Review | P0 |
| 提示词 | 5 套冗余提示词 | 2 套精简 BRTR 提示词 | P0 |
| 幂等性 | 基础签名去重 | 资源预检查 + 失败记忆 + 熔断 | P1 |
| 上下文 | 无分区预算 | 分区预算 + 语义压缩 + TUI 展示 | P1 |
| GitHub 能力 | Issues/PRs/Releases 基础 | 全 GitHub 功能面覆盖 | P1 |
| 组件化 | 部分组件化 | 全组件化 + Section 接口 | P1 |
| 键鼠交互 | 不一致 | 统一焦点状态机 + PgUp/PgDn/滚轮 | P1 |
| 配置 | Viper 基础 | 优先级链 + 迁移 + Schema | P2 |
| 可观测性 | 基础 oplog | 结构化日志 + trace_id + 指标 | P2 |
| 测试 | 基础单测 | 全矩阵(单元/集成/E2E/混沌) | P2 |
| 安全 | 基础脱敏 | 全面审计 + 沙箱 + 白名单 | P2 |
| CI/CD | 无 CI | 三平台 CI + 灰度发布 | P2 |

---

## P0 差距详情（阻塞性）

### GAP-001: TUI 架构完全重建

**当前**:
- `internal/tui/model.go`: 定义 Page/FocusZone 枚举，混合 UI 状态
- `internal/tui/view.go`: 手动拼接 Left/Right 面板
- `internal/tui/update.go`: 单一 Update 函数处理所有消息
- 组件: `areas_tree.go`, `suggestion_card.go` 等独立组件但无统一协议

**目标**:
- `internal/tui/context/context.go`: ProgramContext 集中状态
- `internal/tui/context/styles.go`: Styles 集中样式
- `internal/tui/components/section/section.go`: Section 统一接口
- `internal/tui/components/table/`: Table + ListViewport
- `internal/tui/components/sidebar/`: Sidebar + glamour
- `internal/tui/components/tabs/`: Tabs + Carousel
- `internal/tui/components/footer/`: Footer 动态帮助
- `internal/tui/keys/keys.go`: 集中 KeyMap
- `internal/tui/views/agent/`: Agent 视图
- `internal/tui/views/git/`: Git 视图
- `internal/tui/views/workspace/`: Workspace 视图
- `internal/tui/views/github/`: GitHub 视图
- `internal/tui/views/config/`: Config 视图

**工作量**: ~15 新文件, ~8 重建文件

### GAP-002: 命令执行层重构

**当前**:
- `internal/executor/runner.go`: 1399 行单文件，混合命令解析/执行/校验/分类
- 命令通过 `parseCommand()` 字符串拆分
- 临时文件管理未抽象

**目标**:
- `internal/executor/cmdobj.go`: CmdObj Builder
- `internal/executor/platform.go`: Platform 检测
- `internal/executor/tempfile.go`: 临时文件管理
- `internal/executor/runner.go`: 精简为分发逻辑
- `internal/executor/preflight.go`: 预检查逻辑
- `internal/executor/classify.go`: 错误分类

**工作量**: ~6 新/重构文件

### GAP-003: JSON 解析鲁棒性

**当前**: 直接 `json.Unmarshal`，失败触发重规划
**目标**: `jsonrepair.Repair()` -> `json.Unmarshal` -> 结构化错误

**工作量**: 引入依赖 + 修改 4-5 个解析点

### GAP-004: 工作流引擎

**当前**:
- `internal/flow/orchestrator.go`: RunMaintainRound/RunGoalRound/RunCreativeRound
- `internal/flow/maintain.go`, `goal.go`, `creative.go`: 三个独立流程
- 状态机: idle -> analyzing -> executing -> refreshing

**目标**:
- `internal/flow/orchestrator.go`: Research -> Plan -> Execute -> Review 循环
- `internal/flow/plan.go`: Plan 持久化
- `internal/flow/circuit.go`: 断路器/熔断
- 废除旧三循环

**工作量**: ~4 文件重构/新建

### GAP-005: 提示词精简

**当前**:
- `internal/llm/prompt/builder.go`: 复杂的 prompt 构建
- 多套 prompt 文件，token 超标

**目标**:
- 统一 Planner Prompt < 50 行
- 统一 Helper Prompt < 30 行
- BRTR 四段式结构
- MCP Tool Schema 标准化

**工作量**: ~3 文件重构

---

## P1 差距详情（重要但非阻塞）

### GAP-006: 幂等与反死循环

**当前**: 基础 action 签名去重 + idempotency key
**缺失**: 资源存在性预检查（已有部分）、失败模式记忆窗口、连续相同错误熔断

### GAP-007: 上下文预算

**当前**: 基础 token 估算
**缺失**: 分区预算、失败块优先级、语义压缩、TUI 展示 used/max

### GAP-008: GitHub 全能力面

**当前**: Issues/PRs/Releases/Labels/Secrets/Variables 基础操作
**缺失**: Discussions, Projects v2, Pages, Codespaces, Rulesets, Org Admin

### GAP-009: TUI 组件标准化

**当前**: `areas_tree.go`, `suggestion_card.go` 等独立组件
**缺失**: 统一 Section 接口、Table 组件、Search/Input/Autocomplete

### GAP-010: 键鼠一致性

**当前**: 部分区域支持滚动，焦点路由不一致
**缺失**: 统一焦点状态机、PgUp/PgDn 全区域生效、鼠标滚轮精确路由

---

## P2 差距详情（质量提升）

### GAP-011: 配置系统
- 缺少配置优先级链 (Defaults < Global < Project < Env < CLI)
- 缺少配置迁移机制
- 缺少配置 Schema 验证

### GAP-012: 可观测性
- 缺少全链路 trace_id
- 缺少核心指标面板
- 缺少 SLO 定义

### GAP-013: 测试矩阵
- 缺少 E2E 测试
- 缺少跨平台 CI
- 缺少混沌测试

### GAP-014: 安全合规
- 缺少命令白名单完善
- 缺少路径沙箱
- 缺少供应链审计

### GAP-015: 发布流程
- 缺少灰度发布
- 缺少功能开关
- 缺少回滚预案

---

## 依赖关系

```
GAP-001 (TUI) <-- GAP-009 (组件) <-- GAP-010 (键鼠)
GAP-002 (命令) <-- GAP-003 (JSON) <-- GAP-006 (幂等)
GAP-004 (工作流) <-- GAP-005 (提示词) <-- GAP-007 (上下文)
GAP-008 (GitHub) <-- GAP-002 (命令)
GAP-011~015 依赖 GAP-001~010 完成
```

## 执行优先级

1. GAP-002 + GAP-003 (命令+JSON) — 解决 whitespace 和死循环根因
2. GAP-004 + GAP-005 (工作流+提示词) — 重建核心引擎
3. GAP-001 (TUI) — 重建交互层
4. GAP-006 + GAP-007 (幂等+上下文) — 稳定性提升
5. GAP-008~010 (GitHub+组件+键鼠) — 功能完善
6. GAP-011~015 (质量/安全/发布) — 工程化
