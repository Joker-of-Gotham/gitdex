# GitLab Admin Capability Matrix

This file tracks GitLab platform surfaces by executable coverage in gitdex.

## Public API Surfaces

- `merge_requests`
  - Supported: inspect list/single/changes/approvals, create, update, close, reopen.
  - Rollback: compensating only; merged-state reversal is not automatically supported.
- `pipelines`
  - Supported: inspect list/single/jobs/bridges, create, retry, cancel.
  - Rollback: run-control actions are non-reversible and require compensation or manual restore.
- `environments`
  - Supported: inspect list/single/deployments, stop environment.
  - Rollback: compensating only.
- `pages`
  - Supported: inspect settings/domains/domain/health, verify domain.
  - Rollback: compensating/manual depending on external DNS and certificate state.
- `security`
  - Supported: inspect dashboard, vulnerabilities, and policy posture.
  - Rollback: inspect-only.

## Adapter Routing

- Default routing is `API -> browser -> explicit boundary failure`.
- `browser-backed` flows are stubbed and preserve an explicit audit chain instead of faking external completion.

## Notes

- GitLab uses the same capability matrix, boundary model, schema hints, flow integration, and audit/ledger vocabulary as GitHub.
- Remaining breadth gaps are tracked in [GITLAB_ADMIN_GAP_MATRIX.md](/e:/Work/Engineering-Development/Manual-Series/Git-Maunual/Git-Manual-Tool/docs/GITLAB_ADMIN_GAP_MATRIX.md).
