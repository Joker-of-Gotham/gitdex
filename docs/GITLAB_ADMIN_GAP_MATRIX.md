# GitLab Admin Gap Matrix

This matrix tracks the current GitLab public-surface model in gitdex.

## Partial Mutate

- `merge_requests`
  - Supported: inspect, create, update, close, reopen.
  - Gap: merged-state rollback and full approval-rule parity are not yet automatic.
- `pipelines`
  - Supported: inspect, create, retry, cancel.
  - Gap: run-control actions are explicitly non-reversible.
- `environments`
  - Supported: inspect environments and deployments, stop environment.
  - Gap: broader environment settings parity is incomplete.
- `pages`
  - Supported: inspect Pages settings, domains, and health-oriented views.
  - Gap: DNS and certificate lifecycle still depends on external systems.

## Inspect Only

- `security`
  - Supported: inspect dashboard-style posture and vulnerability-oriented views.
  - Gap: no unified mutate/rollback executor yet.

## Notes

- Capability boundaries, execution policies, and executor schema hints now exist for these surfaces.
- This gives GitLab flows the same boundary/report/diagnostic vocabulary as GitHub, even where executor depth is still catching up.
