# GitDex V3 Big Bang 切换手册

本手册用于一次性切换到 V3 主执行链路，覆盖切换前并行验证、切换步骤、验收点与失败处置入口。

## 1. 角色与职责

- **切换指挥（Commander）**：统一发令、确认门禁、决定前滚或回滚。
- **执行负责人（Operator）**：执行脚本、打标、发布、记录时间线。
- **观察员（Observer）**：盯盘 SLO 指标与异常告警，记录证据。
- **审批人（Approver）**：在切换前后给出 go/no-go 决策。

## 2. 前置输入

- 已通过 `docs/V3_RELEASE_GATES.md` 的全部一票否决项与必须通过项。
- 已确认回滚基线 tag（例如 `v3-pre-cutover-YYYYMMDD-HHMM`）。
- 已冻结主干写入窗口（切换期间禁止并发合入）。

## 3. Big Bang 前并行验证

### 3.1 本地预检（必须）

- macOS/Linux：
  - `./scripts/v3-cutover-preflight.sh`
- Windows：
  - `.\scripts\v3-cutover-preflight.ps1`

### 3.2 CI 并行预检（必须）

- 手动触发 `cutover-drill` workflow。
- 期望结果：`ubuntu-latest`、`macos-latest`、`windows-latest` 三平台全部通过。

### 3.3 数据对账（必须）

- 契约对账：`suggestion/action` 均带协议版本字段（`version`）。
- 配置对账：`gitdex config lint/explain/source` 均可成功输出。
- 观测对账：执行日志含 `trace=`，并可在 TUI 头部看到 `cmd/replan/llm` 指标。

## 4. 一次性切换步骤（主路径）

1. 同步主干并确认无脏工作区：
   - `git fetch --all --prune`
   - `git checkout main && git pull --ff-only`
2. 打回滚保护 tag（切换前）：
   - `git tag -a v3-pre-cutover-YYYYMMDD-HHMM -m "pre v3 cutover backup"`
   - `git push origin v3-pre-cutover-YYYYMMDD-HHMM`
3. 执行预检脚本（本地）并确认通过。
4. 合入 V3 切换提交（禁止 squash 丢失审计历史）：
   - 推荐 `git merge --ff-only <v3-ready-branch>`
5. 推送主干并记录切换时间戳。
6. 进入切换后观察窗口（见 `docs/SLO_GUARD_WINDOW.md`）。

## 5. 切换后验收（T+60 分钟内）

- 命令成功率、重规划、LLM 可用性达到门槛。
- 高频回归项（whitespace/404/provider）未出现回归。
- 核心链路可用：`maintain -> goal -> execute -> refresh`。
- 如出现重大故障，执行：
  - 前滚：`docs/ROLLFORWARD_PLAYBOOK.md`
  - 回滚：`docs/ROLLBACK_PLAYBOOK.md`

## 6. 禁止项

- 未通过预检直接切换。
- 无回滚 tag 直接发布。
- 切换期间强推主干。
- 出现安全红线（密钥泄露、路径逃逸）后继续推进。
