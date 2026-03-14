# Bitbucket Admin Gap Matrix

This matrix tracks the current Bitbucket public-surface model in gitdex.

## Partial Mutate

- `pull_requests`
  - Supported: inspect, create, update, decline, reopen.
  - Gap: merged pull requests cannot be automatically unmerged.
- `pipelines`
  - Supported: inspect, create, stop, rerun.
  - Gap: run-control actions are non-reversible.
- `deployments`
  - Supported: inspect deployment environments and history.
  - Gap: broader deployment-policy lifecycle is still incomplete.
- `branch_restrictions`
  - Supported: inspect, create, update, delete.
  - Gap: some policy-composition parity still requires manual review.
- `webhooks`
  - Supported: inspect, create, update, delete.
  - Gap: downstream delivery rollback remains operator-managed.
- `repository_variables`
  - Supported: inspect, create, update, delete.
  - Gap: secured-value restore still requires prior plaintext.

## Notes

- Capability boundaries, execution policies, and executor schema hints now exist for these surfaces.
- This allows diagnostics, reports, and operator-facing boundary explanations to stay consistent with GitHub.
