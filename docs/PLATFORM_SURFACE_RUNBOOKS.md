# Platform Surface Runbooks

This file complements the capability matrices with operator-facing runbooks for the surfaces that currently have executor coverage.

## GitHub Security Surfaces

### `dependabot_config`

- Capability: structured `.github/dependabot.yml` lifecycle.
- Supported operations: inspect, mutate, validate, rollback.
- Coverage and boundary: `full` for file-backed config, with posture details split across repository settings.
- Rollback and compensation: reversible to the previous file snapshot.
- Unattended risk: medium; mutations are approval-gated and validated through schema, deterministic re-encode, and no-op diff detection.
- Adapter conditions: `api-backed` by default, `gh-backed` or `browser-backed` only when the API route is unavailable.

### `dependabot_posture`, `secret_scanning_settings`, `secret_scanning_alerts`, `code_scanning_tool_settings`, `code_scanning_default_setup`, `codeql_setup`

- Capability: repository security posture and alert/default-setup administration.
- Supported operations: inspect, mutate where public API exists, validate, rollback-or-compensation.
- Coverage and boundary: `partial_mutate` when GitHub exposes only a subset of the full UI surface.
- Rollback and compensation: compensating rollback; some UI-only posture remains outside direct repository CRUD.
- Unattended risk: high; surfaces are approval-required and not scheduler-safe for unattended mutation.
- Adapter conditions: API first, then `gh-backed`, then `browser-backed` with explicit manual completion.

## GitHub Artifact And Notification Surfaces

### `packages`

- Capability: package, version, asset, restore, and registry metadata inspection.
- Supported operations: inspect package, inspect versions, inspect assets, delete version, restore version.
- Coverage and boundary: `partial_mutate` because registry-specific metadata is uneven across public APIs.
- Rollback and compensation: compensating restore semantics using package identity and restore endpoints when available.
- Unattended risk: high for delete and restore, low for inspect.
- Adapter conditions: API-backed where registry endpoints exist; adapter fallback remains auditable.

### `notifications`, `email_notifications`

- Capability: repository watch state, thread subscription, inbox views, repo-level mark-read.
- Supported operations: inspect repo/thread/watch surfaces, mutate subscription state, validate mark-read state.
- Coverage and boundary: `partial_mutate`; account-wide notification preferences stay in `adapter` or `inspect_only`.
- Rollback and compensation: compensating because notification delivery state may change between operator actions.
- Unattended risk: medium; muted/watch state changes are approval-required.
- Adapter conditions: API-backed for repository and thread surfaces, adapter fallback for anything outside public repository admin APIs.

## GitHub Actions, Codespaces, And Copilot

### `actions`

- Capability: workflow usage, artifacts, caches, repo policy, allowed actions, permissions, and selected run-control flows.
- Supported operations: inspect usage/artifacts/caches/policy, mutate permissions and allowed-actions state, trigger selected run-control actions.
- Coverage and boundary: `partial_mutate` because dispatch/rerun/cancel are inherently non-reversible.
- Rollback and compensation: policy changes are reversible; run-control actions are compensating or manual-restore only.
- Unattended risk: high for run-control and policy changes.
- Adapter conditions: API-backed first; adapter fallback is recorded in the ledger with rollback kind and boundary reason.

### `codespaces`

- Capability: repo policy, list, devcontainer inspect, prebuild inspect, create/start/stop/delete.
- Supported operations: inspect codespaces and repo policy, mutate lifecycle controls.
- Coverage and boundary: `partial_mutate`; deleted codespaces cannot be recreated as an exact automatic rollback.
- Rollback and compensation: compensating rollback only.
- Unattended risk: high for lifecycle actions, low for inspect.
- Adapter conditions: API-backed where public endpoints exist, adapter fallback otherwise.

### `copilot_seat_management`, `copilot_code_review`, `copilot_coding_agent`

- Capability: seat management, billing/metrics inspection, content exclusions, org-scoped admin surfaces with public coverage.
- Supported operations: inspect public billing/metrics/exclusions surfaces, mutate supported exclusions and seat assignments.
- Coverage and boundary: `partial_mutate`; non-public or plan-gated UI remains boundary-labeled.
- Rollback and compensation: compensating rollback with operator review for org-scoped changes.
- Unattended risk: high because seat and policy changes are approval-required.
- Adapter conditions: API-backed only for public surfaces; non-public areas explicitly fall through to boundary handling instead of fake CRUD.

## GitLab And Bitbucket Parity Surfaces

### GitLab

- Capability: `merge_requests`, `pipelines`, `environments`, `pages`, `security`.
- Supported operations: public API inspect/mutate/validate/rollback for exposed surfaces.
- Coverage and boundary: public API parity only; broader GitHub-like breadth is documented in `GITLAB_ADMIN_GAP_MATRIX.md`.
- Rollback and compensation: mostly compensating; `security` remains inspect-leaning and may require manual restore.
- Unattended risk: non-inspect mutations remain approval-required and non-scheduler-safe.
- Adapter conditions: `api-backed` when tokened, otherwise `browser-backed` stub with explicit manual recovery.

### Bitbucket

- Capability: `pull_requests`, `pipelines`, `deployments`, `branch_restrictions`, `webhooks`, `repository_variables`.
- Supported operations: public API inspect/mutate/validate/rollback for exposed repository admin surfaces.
- Coverage and boundary: public API parity only; remaining gaps are documented in `BITBUCKET_ADMIN_GAP_MATRIX.md`.
- Rollback and compensation: reversible for branch restrictions and webhooks, compensating for pipelines, deployments, and repository variables.
- Unattended risk: medium-to-high for mutate and rollback flows.
- Adapter conditions: `api-backed` when tokened, otherwise `browser-backed` stub with audit-chain preservation.

## Operator Checklist

- Check `flow-report` for health, approval state, dead-letter queue, locks, observe-only mode, escalation timestamp, and recovery path.
- Check `platform-mutation-ledger` for adapter label, rollback kind, diagnostics, and boundary reason before retrying or compensating.
- Use `H` to recover unattended execution from observe-only mode, then use `R`, `X`, or `C` on the selected step.
- Treat every `browser-backed` mutation as manual completion required even if a stub result exists in the ledger.
