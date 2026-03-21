# Story 8.5: LLM 自主巡航系统

Status: ready-for-dev

## Story

As a 仓库管理者,
I want Gitdex 能基于 LLM 进行 7×24 无人干预自主巡航，自动发现问题并执行维护,
So that 仓库能保持健康状态而无需我持续手动监控。

## 验收标准

1. 启用自主巡航 → LLM 自动扫描仓库状态（PR、Issue、分支、CI、依赖等）
2. LLM 发现可操作项 → 低风险自动执行，中高风险提交审批队列
3. 每周期完成 → 生成巡航报告（发现/操作/待审批/风险评估）
4. 巡航中 → 用户可随时查看巡航阶段、操作列表、审批队列
5. 安全护栏 → 危险操作被拦截，生成告警，等待人工确认
6. 自然语言多步指令 → LLM 生成结构化计划，用户确认后逐步执行
7. 长程规划 → LLM 分析趋势，提出改进建议

## 任务 / 子任务

### 核心引擎

- [ ] T1: 巡航引擎 `internal/autonomy/cruise.go` (AC: #1, #3)
  - [ ] T1.1: `CruiseEngine` struct — ticker, state, activeRepo, config
  - [ ] T1.2: `Start()` — 启动巡航循环（configurable interval，默认 30 min）
  - [ ] T1.3: `Stop()` — 停止巡航
  - [ ] T1.4: `RunCycle()` — 单次巡航周期:
    1. 收集状态（repo state、GitHub 数据）
    2. 构造上下文 → LLM 分析
    3. 解析 LLM 输出 → 行动项列表
    4. 分级 → 自动执行 / 提交审批
    5. 生成报告
  - [ ] T1.5: `GetStatus()` — 返回当前巡航状态
  - [ ] T1.6: 发送 `CruiseReportMsg` 到 TUI 展示

### LLM 行动规划

- [ ] T2: 行动规划器 `internal/autonomy/planner.go` (AC: #6, #7)
  - [ ] T2.1: `ActionPlan` 类型定义:
    ```go
    type ActionPlan struct {
        ID          string
        Description string
        Steps       []PlanStep
        RiskLevel   RiskLevel  // Low, Medium, High, Critical
        Rationale   string
        CreatedAt   time.Time
    }
    type PlanStep struct {
        Order       int
        Action      string      // "git.branch.delete", "github.issue.close", etc.
        Args        map[string]string
        Reversible  bool
        Description string
    }
    ```
  - [ ] T2.2: `PlanFromLLMOutput(raw string) ([]ActionPlan, error)` — 解析 LLM JSON 输出为计划
  - [ ] T2.3: `PlanFromUserIntent(intent string) (*ActionPlan, error)` — 自然语言意图 → 结构化计划
  - [ ] T2.4: LLM System Prompt 模板 — 包含可用 Tool 列表、风险分级规则、输出 JSON schema

### 安全护栏

- [ ] T3: 护栏系统 `internal/autonomy/guardrails.go` (AC: #5)
  - [ ] T3.1: `RiskLevel` 枚举: Low, Medium, High, Critical
  - [ ] T3.2: 操作风险分级表:
    ```
    Low:      清理已合并分支, 更新标签, 关闭过期 Issue, 添加标签
    Medium:   合并 PR, 创建 Release, Stash 操作
    High:     Push (非 force), 创建分支, 修改文件
    Critical: Force push, 删除 protected 分支, 修改仓库设置
    ```
  - [ ] T3.3: `EvaluateRisk(plan ActionPlan) RiskLevel` — 计算整体风险
  - [ ] T3.4: `CheckPolicy(plan ActionPlan) (allowed bool, reason string)` — 策略检查
  - [ ] T3.5: 护栏拦截 → 生成 `GuardrailBlockMsg` 到 TUI
  - [ ] T3.6: 用户可配置: 自动执行阈值（默认 Low）、需审批阈值（默认 Medium+）

### 计划执行

- [ ] T4: 计划执行器 `internal/autonomy/executor.go` (AC: #2, #6)
  - [ ] T4.1: `Execute(plan ActionPlan) ExecutionResult` — 逐步执行计划
  - [ ] T4.2: 每步执行前检查护栏
  - [ ] T4.3: 失败回滚: 如果某步失败，执行已完成步骤的逆操作（如果 reversible）
  - [ ] T4.4: 执行进度: 通过 `ExecutionProgressMsg` 实时通知 TUI
  - [ ] T4.5: Action 到 gitops/github 的映射:
    ```go
    actionMap := map[string]ActionHandler{
        "git.branch.delete":   h.deleteBranch,
        "git.branch.create":   h.createBranch,
        "git.commit":          h.commit,
        "git.push":            h.push,
        "git.fetch":           h.fetch,
        "github.pr.merge":     h.mergePR,
        "github.pr.comment":   h.commentPR,
        "github.issue.close":  h.closeIssue,
        "github.issue.create": h.createIssue,
        // ... 更多操作映射
    }
    ```

### 报告与通知

- [ ] T5: 巡航报告 `internal/autonomy/reporter.go` (AC: #3)
  - [ ] T5.1: `CruiseReport` 类型:
    ```go
    type CruiseReport struct {
        CycleID     string
        StartTime   time.Time
        EndTime     time.Time
        Findings    []Finding     // LLM 发现的问题
        Executed    []ExecutedAction  // 已自动执行的操作
        Pending     []ActionPlan   // 待审批的计划
        Suggestions []Suggestion   // 长程建议
    }
    ```
  - [ ] T5.2: 报告渲染 — Markdown 格式，可在 Chat 视图中查看
  - [ ] T5.3: 报告持久化 — 保存到配置目录

### LLM Tool 定义

- [ ] T6: Tool 封装 `internal/autonomy/tools.go` (AC: #1, #6)
  - [ ] T6.1: 将 gitops 和 github 操作封装为 Tool 定义:
    ```go
    type Tool struct {
        Name        string
        Description string
        Parameters  map[string]ToolParam
        Handler     func(ctx context.Context, args map[string]string) (string, error)
    }
    ```
  - [ ] T6.2: 工具清单生成 — 自动从 Tool 定义生成 LLM system prompt
  - [ ] T6.3: 工具调用解析 — 从 LLM JSON 输出中提取工具调用

### TUI 集成

- [ ] T7: 审批队列视图 `internal/tui/views/approval_queue.go` (AC: #4)
  - [ ] T7.1: 显示待审批的 ActionPlan 列表
  - [ ] T7.2: 选中计划 → 展示详细步骤、风险等级、理由
  - [ ] T7.3: 操作: 批准(Enter) / 拒绝(d) / 修改(e)
  - [ ] T7.4: 批准后 → 提交执行器执行
- [ ] T8: 巡航状态视图 `internal/tui/views/cruise_status.go` (AC: #4)
  - [ ] T8.1: 显示: 巡航状态(运行中/暂停/停止)、当前周期进度、上次报告摘要
  - [ ] T8.2: 操作: 启动/暂停/停止巡航
  - [ ] T8.3: 查看历史报告列表
- [ ] T9: Chat 集成 (AC: #6)
  - [ ] T9.1: 自然语言指令 → 调用 `planner.PlanFromUserIntent`
  - [ ] T9.2: 在 Chat 中展示计划步骤 → 用户确认
  - [ ] T9.3: 确认后 → 调用 executor 执行 → 进度实时显示

### 配置

- [ ] T10: 巡航配置 (AC: 全局)
  - [ ] T10.1: Settings 新增 "巡航" section:
    - `cruise.enabled` (bool, 默认 false)
    - `cruise.interval` (duration, 默认 30m)
    - `cruise.auto_execute_threshold` (risk level, 默认 low)
    - `cruise.approval_threshold` (risk level, 默认 medium)
  - [ ] T10.2: 配置持久化到 `config.yaml`

## Dev Notes

### 架构设计

```
                    ┌──────────────┐
                    │  CruiseEngine│ ← goroutine, ticker-driven
                    │   (cruise.go)│
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
    ┌──────────────┐ ┌──────────┐ ┌──────────────┐
    │ State Collector│ │ Planner │ │  Reporter    │
    │ (repo inspect) │ │(LLM call)│ │(report gen)  │
    └──────┬───────┘ └────┬─────┘ └──────────────┘
           │              │
           ▼              ▼
    ┌──────────────┐ ┌──────────────┐
    │  Guardrails  │ │   Executor   │
    │(risk classify)│ │(step-by-step)│
    └──────────────┘ └──────┬───────┘
                            │
                  ┌─────────┼─────────┐
                  ▼         ▼         ▼
              [gitops.*]  [github.*] [file ops]
```

### 参考架构

1. **Symphony 模式** (Elixir Orchestrator):
   - Polling loop → 检测工作项 → 分配 Agent → 执行 → 报告
   - 状态: `idle → scanning → planning → executing → reporting → idle`
   - 错误处理: retry 3 次 → backoff → 标记失败

2. **OpenAI Agents 护栏模式**:
   - 输入护栏: 检查意图合理性
   - 工具护栏: 执行前检查权限
   - 输出护栏: 验证结果合规性
   - Tripwire: 触发时立即中断

3. **ruflo MCP 工具模式**:
   - Tool 定义: name + description + inputSchema + handler
   - 工具注册: 中心化注册表
   - 工具调用: JSON-RPC 风格

### LLM System Prompt 模板

```
你是 Gitdex 巡航引擎。你的任务是分析仓库状态并提出维护行动计划。

## 当前仓库状态
{repo_state_json}

## 可用操作
{tool_definitions}

## 风险分级规则
- Low: 清理性操作，无数据风险
- Medium: 修改操作，可逆
- High: 推送/创建操作，影响远端
- Critical: 不可逆操作，影响保护资源

## 输出格式
请以 JSON 格式输出行动计划数组:
[
  {
    "description": "操作描述",
    "risk_level": "low|medium|high|critical",
    "rationale": "操作理由",
    "steps": [
      {
        "action": "git.branch.delete",
        "args": {"branch": "feature/old"},
        "description": "删除已合并的旧分支",
        "reversible": true
      }
    ]
  }
]
```

### Project Structure Notes

- 新建 `internal/autonomy/` 包 — 自主巡航核心
- 新建视图在 `internal/tui/views/` 下
- 巡航引擎作为 goroutine 运行，通过 channel 与 TUI 通信
- 使用 `tea.Cmd` 模式将巡航事件发送到 Bubble Tea 消息循环

### References

- [Source: internal/gitops/] — Git 操作基础
- [Source: internal/platform/github/client.go] — GitHub API
- [Source: internal/llm/adapter/] — LLM Provider 接口
- [Source: internal/state/repo/model.go] — 仓库状态模型
- [Reference: symphony/SPEC.md] — 自主编排模式
- [Reference: symphony/elixir/lib/symphony_elixir/orchestrator.ex] — GenServer 巡航循环
- [Reference: openai-agents-python/docs/guardrails.md] — 护栏设计
- [Reference: openai-agents-python/src/agents/] — Agent/Runner/Tool 模式
- [Reference: ruflo/src/infrastructure/mcp/MCPServer.ts] — MCP Tool Provider
- [Reference: agency-agents/integrations/mcp-memory/] — 跨会话记忆

## Dev Agent Record

### Agent Model Used

（待实现时填写）

### Completion Notes List

### File List
