# GitDex V4 Anti-Patterns

> 禁止沿用的旧设计与反模式清单

## 1. 架构级反模式

### AP-001: 三循环强耦合
- **现状**: `Maintain -> Goal -> Creative` 三流程硬编码在 `flow/orchestrator.go`
- **问题**: 流程间共享状态 (`sliceSeq`, `roundSeq`)、执行顺序固死、无法独立测试
- **替代**: Research -> Plan -> Execute -> Review 标准 Agentic 循环

### AP-002: Planner/Helper 散布多文件
- **现状**: `prompt_a.go` ~ `prompt_e.go` 五套独立提示词
- **问题**: 冗余、维护困难、token 浪费
- **替代**: 统一 Planner Prompt + Helper Prompt，BRTR 四段式，各 < 50 行

### AP-003: 静态多面板布局
- **现状**: `view.go` 通过 `lipgloss.JoinHorizontal/Vertical` 手动拼接固定比例面板
- **问题**: 尺寸不自适应、焦点路由困难、格式错位
- **替代**: 单视图全屏 + Tabs 切换 + Sidebar 详情（gh-dash 模式）

---

## 2. 命令执行反模式

### AP-004: 字符串拼接命令
- **现状**: `git commit -m "message"` 通过字符串拼接构造
- **问题**: 特殊字符/换行/空格在不同 shell 解释不一致，whitespace 错误反复
- **替代**: CmdObj Builder + 临时文件传参 (`-F`)

### AP-005: 平台假设
- **现状**: 部分代码假设 Unix shell 可用 (sed/awk/perl)
- **问题**: Windows 上完全不可用
- **替代**: Platform 运行时检测 + 跨平台命令白名单 + 替代建议

### AP-006: 命令验证依赖提示词
- **现状**: 通过提示词告知 LLM "不要使用危险命令"
- **问题**: LLM 不遵守提示词约束
- **替代**: 代码级命令校验 (rejectShellOperators + validActionTypes + preflightAction)

---

## 3. LLM 交互反模式

### AP-007: 无 JSON 修复
- **现状**: `json.Unmarshal` 直接解析 LLM 输出
- **问题**: LLM 常输出 trailing comma、Python constants、缺失引号
- **替代**: 先 `jsonrepair.Repair()` 再解析

### AP-008: 截断用 `...`
- **现状**: 上下文超限时用 `...` 截断
- **问题**: 语义断裂，LLM 无法理解被截断的内容
- **替代**: 语义压缩（保留关键信息、移除冗余细节）

### AP-009: 全局重规划
- **现状**: 任何 action 失败触发全局重规划
- **问题**: 死循环（重规划产出相同失败 action）
- **替代**: 局部修复优先 -> 失败注入上下文 -> 断路器熔断

### AP-010: 无失败记忆
- **现状**: 上一轮失败的命令不会被记住
- **问题**: LLM 反复生成相同的失败命令
- **替代**: failed-pattern memory 窗口，失败命令签名去重

### AP-011: 负面规则提示词
- **现状**: 提示词中大量 "DO NOT"、"NEVER"、"禁止"
- **问题**: LLM 往往忽略负面指令，且浪费 token
- **替代**: 正向描述 + 代码级约束

---

## 4. TUI 交互反模式

### AP-012: 焦点不可追踪
- **现状**: 焦点状态散布在 `model.go` 的多个字段中
- **问题**: 无法确定当前哪个区域有焦点、PgUp/PgDn 失效
- **替代**: 焦点状态机 (view -> section -> widget)

### AP-013: 滚动不一致
- **现状**: 有些区域支持滚动，有些不支持
- **问题**: 用户进入焦点区域后无法滚动
- **替代**: 所有可滚动区域统一 ListViewport + PgUp/PgDn + 鼠标滚轮

### AP-014: 硬编码颜色/尺寸
- **现状**: 部分颜色值和尺寸直接写在代码中
- **问题**: 无法通过配置或主题调整
- **替代**: Styles 集中初始化，从 Theme 派生所有视觉属性

### AP-015: 无空状态处理
- **现状**: 列表为空时显示空白
- **问题**: 用户不知道是加载中还是真的为空
- **替代**: EmptyMessage + Loading Spinner

---

## 5. 配置反模式

### AP-016: 硬编码 URL/路径
- **现状**: `http://localhost:11434` (Ollama)、`https://api.deepseek.com` 等硬编码
- **问题**: 无法自定义端点
- **替代**: 配置字段 + 默认值在 defaults.go

### AP-017: 硬编码二进制名
- **现状**: `"git"`、`"gh"` 字面量
- **问题**: 自定义安装路径不可用
- **替代**: Platform.GitBin/GhBin 运行时解析

### AP-018: 密钥字段歧义
- **现状**: `api_key_env` 既可以是环境变量名也可以是字面 key
- **问题**: 容易误配置
- **替代**: 明确分离 `api_key_env`（环境变量名）和运行时兼容逻辑

---

## 6. 可观测性反模式

### AP-019: 无结构化日志
- **现状**: 部分日志为 printf 风格
- **问题**: 无法机器解析、无法搜索
- **替代**: 结构化 JSON 日志 + trace_id

### AP-020: 无执行追踪
- **现状**: 部分执行路径缺少 trace
- **问题**: 故障时无法追溯
- **替代**: 全链路 trace_id

---

## 执行守则

- 实现 V4 时**逐条检查此清单**
- 代码审查时**将此文件作为 checklist**
- 发现新反模式时**立即追加**
