# V3 BRANCH PROTECTION

本文件定义 GitDex V3 的分支治理与合入门禁，适用于 Big Bang 切换期间与后续稳定期。

## 保护策略

- 受保护分支：`main`（必须启用分支保护规则）。
- 禁止直接 push 到 `main`，仅允许通过 Pull Request 合入。
- 必须至少 1 名代码评审通过（建议 2 名，含 1 名模块负责人）。
- 必须通过 CI（Linux/Windows/macOS 三平台门禁）。
- 必须启用“禁止过期审批自动合并”（有新提交需重新评审）。

## 必选状态检查

- `go test ./... -count=1`
- `go build ./...`
- 回归门禁（whitespace/404/provider）
- cutover drill（手动触发的三平台演练）

## 合入约束

- PR 必须包含风险说明与回滚策略（至少引用 `docs/ROLLBACK_PLAYBOOK.md`）。
- 涉及执行层、配置层、TUI 交互层的改动，必须附测试或回归说明。
- 禁止将密钥、token、凭据文件直接提交到仓库。

## 紧急修复例外

- 仅在生产阻断且负责人审批后允许走紧急流程。
- 紧急修复仍需：
  - 通过最小回归集；
  - 补齐事后 PR 与 RCA；
  - 在 24 小时内补齐正式评审。

