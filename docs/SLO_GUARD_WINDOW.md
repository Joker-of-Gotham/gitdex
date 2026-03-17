# GitDex V3 切换后 SLO 守护与观察窗口

本文件定义切换后观察窗口、SLO 指标与处置阈值。

## 1. 观察窗口

- `T+5m`：首轮健康检查（命令执行与 provider 可用性）。
- `T+15m`：短期稳定性检查（replan 是否异常增长）。
- `T+30m`：回归风险检查（whitespace/404/provider unavailable）。
- `T+60m`：切换结论评审（go/hold/rollback）。
- `D+1`：次日复盘与指标确认。

## 2. 核心 SLO（切换窗口）

- **命令成功率**：`>= 90%`（`cmd_ok / cmd_total`）。
- **重规划强度**：每 10 次命令 `replan <= 2`。
- **LLM 可用性**：`>= 95%`，禁止持续 down。
- **关键回归项**：`whitespace/404/provider` 不得复发。

## 3. 数据来源

- TUI 头部指标：`cmd[ok/total] replan[n] llm[up|down]`
- 执行日志：`.gitdex/maintain/output.txt`
- CI 回归任务：`regression` / `cutover-drill`

## 4. 异常处置阈值

- 任一安全类告警：立即停止推进并评估回滚。
- 命令成功率低于阈值 15 分钟以上：进入前滚或回滚决策。
- LLM 可用性连续 5 分钟为 down：优先恢复 provider 路由。

## 5. 观察结论模板

- 结论：`GO` / `HOLD` / `ROLLBACK`
- 证据：指标截图、关键 trace、回归任务结果
- 决策人：Commander + Approver
