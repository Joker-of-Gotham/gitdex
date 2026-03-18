# GitDex V4 Adoption Matrix

> 资料 -> 设计决策映射矩阵

## 采纳状态说明

- **ADOPT**: 立即采纳，直接影响 V4 实现
- **TRIAL**: 试验性采纳，需在实现中验证
- **DEFER**: 延后采纳，非 V4 核心路径
- **REJECT**: 明确拒绝，不适用于 GitDex

---

## 1. TUI 架构采纳矩阵

| 模式/技术 | 来源 | 状态 | GitDex 目标模块 | 理由 |
|-----------|------|------|----------------|------|
| ProgramContext 集中状态 | gh-dash B1-001 | ADOPT | tui/context | 消除组件间状态传递，统一屏幕尺寸/配置/主题 |
| Styles 集中初始化 | gh-dash B1-002 | ADOPT | tui/context | 避免样式散落各处，主题切换只需重新初始化 Styles |
| Section 接口 | gh-dash B1-003 | ADOPT | tui/components/section | 视图组件统一协议：ID/Title/Update/View/GetCurrItem |
| Table + ListViewport | gh-dash B1-004/005 | ADOPT | tui/components/table | 列宽策略+行级滚动+PgUp/PgDn，替代当前 areas_tree |
| Sidebar (glamour) | gh-dash B1-006 | ADOPT | tui/components/sidebar | Markdown 渲染详情面板，viewport 滚动 |
| Tabs + Carousel | gh-dash B1-007/012 | ADOPT | tui/components/tabs | 视图切换导航，溢出处理 |
| Footer 动态帮助 | gh-dash B1-008 | ADOPT | tui/components/footer | 上下文感知键位提示，替代当前静态帮助 |
| KeyMap 集中化 | gh-dash B1-009 | ADOPT | tui/keys | 全局+视图键位，配置覆盖，冲突检测 |
| 单视图全屏布局 | gh-dash B1-016 | ADOPT | tui/ui.go | 替代当前多面板硬编码布局 |
| zone-based 鼠标 | diffnav B3-002 | ADOPT | tui/ui.go | 鼠标事件精确路由到焦点区域 |
| Diff 缓存 | diffnav B3-003 | TRIAL | tui/views/git | 大 diff 缓存避免重复计算 |
| 文件树组件 | diffnav B3-004 | ADOPT | tui/views/workspace | 文件树浏览组件 |

## 2. 命令执行采纳矩阵

| 模式/技术 | 来源 | 状态 | GitDex 目标模块 | 理由 |
|-----------|------|------|----------------|------|
| CmdObj Builder | lazygit B2-001/002 | ADOPT | executor/cmdobj | 命令对象化替代字符串拼接 |
| Platform 抽象 | lazygit B2-003 | ADOPT | executor/platform | 运行时 OS 检测、shell 适配 |
| ICmdObjRunner 接口 | lazygit B2-004 | ADOPT | executor/ | 可测试的执行器接口 |
| 文件传参 (-F) | git-commit A1-002 | ADOPT | executor/tempfile | 彻底解决 whitespace |
| 命令预检查 | current runner.go | ADOPT | executor/preflight | binary/auth/network 快速失败 |
| shell 操作符拒绝 | current runner.go | ADOPT | executor/validate | 保留，防止注入 |
| 密钥脱敏 | current runner.go | ADOPT | executor/redact | 保留，增强 |
| Object-Action 语义 | octo.nvim B4-001 | TRIAL | executor/schema | 资源+动作的统一命令语义 |

## 3. LLM/Agent 采纳矩阵

| 模式/技术 | 来源 | 状态 | GitDex 目标模块 | 理由 |
|-----------|------|------|----------------|------|
| jsonrepair 自动修复 | kaptinlin/jsonrepair A5-003 | ADOPT | llm/parser | LLM 输出 JSON 自动修复 |
| BRTR 提示词结构 | DAIR-AI B5-002 | ADOPT | llm/prompt | Background-Role-Task-Requirements 四段式 |
| EASYTOOL 精简工具 | NAACL 2025 C-001 | ADOPT | llm/prompt | 精简工具描述提升准确率 |
| MCP Tool Schema | MCP A6-001/002 | ADOPT | flow/tools | 标准化工具定义 |
| Research->Plan->Execute->Review | Cluster444 B5-004 | ADOPT | flow/orchestrator | 替代旧三循环 |
| Context 分区预算 | C-005 | ADOPT | llm/budget | 输入分区预算+语义压缩 |
| 断路器/熔断 | SRE D-010 | ADOPT | flow/circuit | 重复失败停止 |
| 多模型路由 | current config | ADOPT | llm/router | planner/helper 独立 provider |
| ReAct 模式 | C-006 | TRIAL | flow/ | Reason+Act 交替模式 |
| Self-Refine | C-010 | DEFER | flow/ | 自我反馈迭代，V4 后期考虑 |

## 4. 配置与可观测性采纳矩阵

| 模式/技术 | 来源 | 状态 | GitDex 目标模块 | 理由 |
|-----------|------|------|----------------|------|
| 配置优先级链 | lazygit/12-Factor | ADOPT | config | Defaults < Global < Project < Env < CLI |
| YAML 嵌套配置 | gh-dash B1-014 | ADOPT | config | 统一 YAML 结构 |
| 结构化日志 | 12-Factor D-009 | ADOPT | observability | JSON 结构化日志 |
| trace_id 全链路 | SRE D-010 | ADOPT | observability | UUID 贯穿请求链 |
| Provider 健康状态机 | current | ADOPT | llm/ | up->degraded->down 状态机 |
| SLO 定义 | SRE D-010 | TRIAL | observability | 延迟/成功率/重规划率 SLO |

## 5. GitHub 全能力面采纳矩阵

| 能力 | API 通道 | 状态 | 理由 |
|------|---------|------|------|
| Issues CRUD | gh CLI (REST) | ADOPT | 核心功能 |
| PRs CRUD + Review | gh CLI (REST) | ADOPT | 核心功能 |
| Releases CRUD | gh CLI (REST) | ADOPT | 核心功能 |
| Actions 管理 | gh CLI (REST) | ADOPT | 核心功能 |
| Notifications | gh CLI (REST) | ADOPT | 核心功能 |
| Labels 管理 | gh CLI (REST) | ADOPT | 核心功能 |
| Secrets/Variables | gh CLI (REST) | ADOPT | 运维功能 |
| Discussions | gh API (GraphQL) | TRIAL | 需 GraphQL |
| Projects v2 | gh API (GraphQL) | TRIAL | 需 GraphQL |
| Pages | gh API (REST) | TRIAL | 低优先级 |
| Codespaces | gh CLI (REST) | DEFER | 低优先级 |
| Rulesets | gh API (REST) | DEFER | 管理员功能 |
| Org Admin | gh API (REST) | DEFER | 管理员功能 |

## 6. 质量与安全采纳矩阵

| 实践 | 来源 | 状态 | 理由 |
|------|------|------|------|
| 核心模块单测 >= 85% | Go best practices | ADOPT | 质量底线 |
| 三平台 CI | GitHub Actions | ADOPT | Win/Linux/macOS |
| 命令注入防护 | OWASP D-008 | ADOPT | 安全底线 |
| 凭据最小暴露 | 12-Factor | ADOPT | 安全底线 |
| 日志脱敏 | current | ADOPT | 保留增强 |
| 混沌测试 | SRE | TRIAL | 网络抖动/限流测试 |
| 供应链审计 | Go vuln | ADOPT | govulncheck |
