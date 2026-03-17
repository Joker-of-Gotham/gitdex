# GitDex V3 回滚手册与演练流程

当切换后出现不可接受风险（安全、数据破坏、主链路不可用）时，使用本手册执行受控回滚。

## 1. 回滚触发条件（任一满足即触发）

- 出现密钥泄露、路径逃逸、权限越界等安全红线。
- 命令失败率持续高于阈值且前滚无法快速止损。
- 自动/巡航模式出现不可恢复死循环。

## 2. 回滚策略

采用**可审计回滚**：基于“切换前保护 tag”生成回滚提交，不使用强推主干。

## 3. 回滚执行步骤

1. 确认切换前保护 tag（示例：`v3-pre-cutover-YYYYMMDD-HHMM`）。
2. 创建回滚分支：
   - `git checkout -b rollback/v3-YYYYMMDD-HHMM main`
3. 执行回滚提交（示例）：
   - `git revert --no-edit v3-pre-cutover-YYYYMMDD-HHMM..HEAD`
4. 验证：
   - `./scripts/v3-cutover-preflight.sh --skip-network`
   - 或 `.\scripts\v3-cutover-preflight.ps1 -SkipNetwork`
5. 提交紧急 PR 并合入主干。
6. 打应急发布 tag（例如 `v2.9.rollback1`）并发布。

## 4. 演练流程（建议每月一次）

1. 在演练分支模拟故障注入。
2. 按本手册执行一次完整回滚。
3. 记录耗时、失败点、改进项。
4. 更新手册与检查清单。

## 5. 注意事项

- 禁止 `push --force` 覆盖主干历史。
- 禁止跳过验证直接发布回滚。
- 回滚后仍需执行 SLO 观察窗口，确认稳定恢复。
