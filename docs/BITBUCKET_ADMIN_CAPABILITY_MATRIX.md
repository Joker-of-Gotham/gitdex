# Bitbucket Admin Capability Matrix

This file tracks Bitbucket platform surfaces by executable coverage in gitdex.

## Public API Surfaces

- `pull_requests`
  - Supported: inspect list/single/activity/diffstat, create, update, decline, reopen.
  - Rollback: compensating only; merge reversal is not automatic.
- `pipelines`
  - Supported: inspect list/single/steps, create, stop, rerun.
  - Rollback: run-control actions are non-reversible and require compensation or manual restore.
- `deployments`
  - Supported: inspect environments and history.
  - Rollback: inspect-only/limited compensation depending on downstream deployment tooling.
- `branch_restrictions`
  - Supported: inspect, create, update, delete.
  - Rollback: reversible for prior repository policy snapshots.
- `webhooks`
  - Supported: inspect, create, update, delete.
  - Rollback: reversible for repository-side webhook state; downstream delivery remains operator-managed.
- `repository_variables`
  - Supported: inspect, create, update, delete.
  - Rollback: compensating; restoring secured values depends on previous plaintext source material.

## Adapter Routing

- Default routing is `API -> browser -> explicit boundary failure`.
- `browser-backed` flows are stubbed and preserve audit metadata, driver identity, and manual completion requirements.

## Notes

- Bitbucket uses the same capability matrix, boundary model, schema hints, flow integration, and audit/ledger vocabulary as GitHub.
- Remaining breadth gaps are tracked in [BITBUCKET_ADMIN_GAP_MATRIX.md](/e:/Work/Engineering-Development/Manual-Series/Git-Maunual/Git-Manual-Tool/docs/BITBUCKET_ADMIN_GAP_MATRIX.md).
