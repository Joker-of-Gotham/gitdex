# GitDex V3 发布门禁（Big Bang）

## 一票否决项

- 任一平台（Windows/Linux/macOS）主流程失败。
- 出现路径逃逸、符号链接越界、密钥明文日志泄露。
- 存在“失败后无恢复动作”的黑盒错误。
- 存在无限重规划或高频重试风暴。

## 必须通过项

### 架构门禁
- 契约层独立可用，跨层 DTO 不再依赖内部私有结构。
- 执行层采用 runtime + adapter 分层，具备前置检查与失败分类。

### 稳定性门禁
- 关键回归场景（whitespace/404/provider）全通过。
- auto/cruise 模式闭环执行完成率达标。

### 可观测门禁
- trace_id 贯穿 LLM -> flow -> runtime -> UI。
- 核心指标可见：延迟、成功率、重试、可用性。

### 体验门禁
- 所有核心面板支持键盘与鼠标滚动。
- 关键文本不使用省略号截断（完整换行）。

### 文档门禁
- 架构宪章、借鉴规则、RFC 模板、切换与回滚手册齐备。
- 已落地并可执行：
  - `docs/V3_BRANCH_PROTECTION.md`
  - `docs/BIG_BANG_CUTOVER_RUNBOOK.md`
  - `docs/ROLLFORWARD_PLAYBOOK.md`
  - `docs/ROLLBACK_PLAYBOOK.md`
  - `docs/SLO_GUARD_WINDOW.md`
  - `docs/CHAOS_DRILL_SUITE.md`

## 切换清单

1. 冻结主干并确认回滚 tag。
2. 执行全量测试与回归集。
3. 执行三平台 CI 并复核工件。
4. 发布前观察窗口（短期 smoke + 指标观察）。
5. Big Bang 切换执行。
6. 切换后 SLO 观察窗口。
