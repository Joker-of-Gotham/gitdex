# GitHub Admin Capability Matrix

This file tracks GitHub platform surfaces by real executor coverage, not by UI label presence.

## Fully or Broadly Executable

- `rulesets`
- `branch_rulesets`
- `actions_secrets_variables`
- `codespaces_secrets`
- `dependabot_secrets`
- `dependabot_config`
- `webhooks`
- `deployment`
- `environments`
- `pr_review`
- `deploy_keys`
- `dependabot_alerts`
- `code_scanning`
- `codeql_analysis`
- `private_vulnerability_reporting`
- `protection_rules`
- `push_protection`

## Partial Mutate

- `actions`
  - Supported: workflow/runs inspection, repository Actions permissions, allowed-actions policy, workflow enable/disable, dispatch, rerun, cancel.
  - Gap: dispatch and run control do not offer full rollback.
- `release`
  - Supported: release inspect/create/update/delete, notes generation, publish draft, asset upload/list/delete.
  - Gap: asset restore still depends on recoverable source bytes or a downloadable previous asset.
- `pull_request`
  - Supported: inspect pull requests, create/update/close/reopen, merge, enable/disable auto-merge.
  - Gap: merged pull requests cannot be automatically unmerged as rollback.
- `pages`
  - Supported: config/build history/latest build/health/domain inspect, create/update/delete config, build/rebuild trigger, external DNS validation.
  - Gap: full DNS registrar and certificate lifecycle still depends on external DNS state and limited API feedback.
- `packages`
  - Supported: inspect package/version/latest version, delete, restore.
  - Gap: richer package settings and all registry-specific admin surfaces are not uniform across APIs.
- `notifications`
  - Supported: repo subscription, thread subscription, repo inbox, global inbox, participating inbox, mark read.
  - Gap: account-level notification preference CRUD is outside repository admin REST flows.
- `email_notifications`
  - Supported: repo/thread subscription state.
  - Gap: full account-level email-routing preferences are not repository admin mutations.
- `advanced_security`
  - Supported: summary, configuration, repository security posture, automated security fixes.
  - Gap: not every UI subsetting is exposed as CRUD REST.
- `security`
  - Supported: aggregate `security_and_analysis` inspect/update.
  - Gap: this is an aggregate executor, not a one-to-one mirror of every security screen.
- `grouped_security_updates`
  - Supported: repository-level posture toggle plus `dependabot_config`.
  - Gap: detailed grouping policy is mainly in `dependabot.yml`.
- `dependabot_version_updates`
  - Supported: repository-level posture toggle plus `dependabot_config`.
  - Gap: detailed cadence/policy is mainly in `dependabot.yml`.
- `copilot_code_review`
  - Supported: billing/seats/metrics/content exclusions inspect, content exclusion mutate.
  - Gap: broader org policy depends on plan-gated or non-public surfaces.
- `copilot_coding_agent`
  - Supported: billing/seats/metrics/content exclusions inspect, content exclusion mutate.
  - Gap: broader coding-agent admin parity is not fully public.
- `copilot_seat_management`
  - Supported: billing/seats inspect, add/remove users, add/remove teams.
  - Gap: wider Copilot org policy remains split across other surfaces.
- `check_runs_failure_threshold`
  - Supported: ruleset-backed inspect/mutate.
  - Gap: GitHub does not expose a dedicated standalone threshold resource.
- `codespaces`
  - Supported: list/single/devcontainers inspect, create, start, stop, delete.
  - Gap: deleted codespaces cannot be recreated automatically through rollback; account-level preferences are outside repo scope.

## Inspect Only

- `dependency_graph`
  - Supported: SBOM export / dependency graph inspection.
  - Gap: repository-level mutation is not exposed through GitHub REST.
- `copilot_autofix`
  - Supported: inspect Autofix suggestions and generated commits from code scanning surfaces.
  - Gap: no standalone mutate/rollback surface.

## Composed Through Orchestration

- `ai_assistant_deployment`
  - This is not a single GitHub REST resource.
  - It is modeled as workflow orchestration across `deployment`, `environments`, `actions`, `actions_secrets_variables`, `webhooks`, and `pages`.

## Notes

- Coverage here follows GitHub public APIs. If a GitHub UI pane has no public write surface, gitdex does not fake CRUD support for it.
- Workflow selection now materializes a multi-step platform orchestration flow and persists it for later LLM reasoning, execution tracking, and automation recovery.
