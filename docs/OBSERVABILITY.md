# GitDex V3 观测体系

## Trace

- 每次 suggestion 执行都会生成 `trace_id`。
- `trace_id` 贯穿：
  - TUI 执行日志（oplog detail）
  - `output.txt` 的 step 记录
  - ActionResult `trace` 字段

## Metrics

内置轻量指标（进程内）：

- `llm_calls_total`
- `llm_calls_failed`
- `llm_latency_ms_total`
- `commands_total`
- `commands_succeeded`
- `commands_failed`
- `replan_attempts`
- `provider_available`

## UI 展示

状态栏会显示：

- `ctx[used/max]`
- `cmd[ok/total]`
- `replan[n]`
- `llm[up|down]`

用于快速定位稳定性与可用性问题。

## Failure Taxonomy Dashboard

- 在主界面输入 `/failures` 可输出失败分类看板。
- 分类桶覆盖：`auth_permission`、`network_transient`、`not_found`、`conflict_duplicate`、`validation`、`unknown`。
- 用于识别高频错误分布并指导回归优先级。
