# GitDex V3 切换后前滚处置手册

当前滚动策略用于“切换已完成但出现可修复故障”的场景，目标是最短时间恢复服务，而不是立即回退版本。

## 1. 适用场景

- 单点缺陷：某类命令持续失败（例如 GitHub API 参数不兼容）。
- 配置缺陷：某 provider 可用但路由错误。
- 体验缺陷：TUI 局部交互异常，但主流程仍可运行。

## 2. 快速分诊（10 分钟内）

1. 收集故障证据：
   - 错误日志（含 `trace_id`）
   - `.gitdex/maintain/output.txt` 对应执行段
   - TUI 头部指标快照
2. 归类影响范围：
   - P0：主链路不可用或安全风险
   - P1：主链路可用但高频失败
   - P2：可绕过的体验问题
3. 决策：
   - P0：优先评估回滚
   - P1/P2：走前滚热修

## 3. 前滚热修流程

1. 建立热修分支：
   - `git checkout -b hotfix/v3-<issue-id> main`
2. 修复并本地验证：
   - `./scripts/v3-cutover-preflight.sh --skip-network`
   - Windows 使用 `.\scripts\v3-cutover-preflight.ps1 -SkipNetwork`
3. 发起紧急 PR（必须包含故障根因与验证结论）。
4. 合入后打补丁 tag 并发布（例如 `v3.0.1`）。
5. 再次进入 SLO 观察窗口（至少 60 分钟）。

## 4. 退出条件

- 关键 SLO 回到阈值内。
- 回归场景复测通过。
- 观察窗口内无新增同类告警。
