---
stepsCompleted: ['step-01-detect-mode', 'step-02-load-context', 'step-03-risk-and-testability', 'step-04-coverage-plan', 'step-05-generate-output']
lastStep: 'step-05-generate-output'
lastSaved: '2026-03-18'
workflowType: 'testarch-test-design'
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/planning-artifacts/epics.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
  - _bmad/bmm/config.yaml
  - _bmad/tea/config.yaml
  - _bmad/tea/testarch/knowledge/adr-quality-readiness-checklist.md
  - _bmad/tea/testarch/knowledge/test-levels-framework.md
  - _bmad/tea/testarch/knowledge/risk-governance.md
  - _bmad/tea/testarch/knowledge/test-quality.md
  - _bmad/tea/testarch/knowledge/probability-impact.md
  - _bmad/tea/testarch/knowledge/test-priorities-matrix.md
  - _bmad/tea/testarch/knowledge/overview.md
  - _bmad/tea/testarch/knowledge/api-request.md
  - _bmad/tea/testarch/knowledge/auth-session.md
  - _bmad/tea/testarch/knowledge/recurse.md
  - _bmad/tea/testarch/knowledge/pactjs-utils-overview.md
  - _bmad/tea/testarch/knowledge/pactjs-utils-consumer-helpers.md
  - _bmad/tea/testarch/knowledge/pactjs-utils-provider-verifier.md
  - _bmad/tea/testarch/knowledge/pactjs-utils-request-filter.md
  - _bmad/tea/testarch/knowledge/pact-mcp.md
---

# Test Design Progress

## Step 1 - Detect Mode and Prerequisites

- Selected mode: `system-level`
- Reason: current planning set includes `PRD + Architecture + Epics/Stories`; this workflow prefers system-level test design first in that situation.
- Required inputs found:
  - PRD: `[_bmad-output/planning-artifacts/prd.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/planning-artifacts/prd.md)`
  - Architecture: `[_bmad-output/planning-artifacts/architecture.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/planning-artifacts/architecture.md)`
  - Supporting backlog: `[_bmad-output/planning-artifacts/epics.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/planning-artifacts/epics.md)`
  - Supporting UX: `[_bmad-output/planning-artifacts/ux-design-specification.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/planning-artifacts/ux-design-specification.md)`
- Prerequisite result: pass
- Note: no `sprint-status.yaml` was found, so there is no file-based reason to force epic-level mode.

## Step 2 - Load Context and Knowledge Base

- Loaded TEA configuration from `_bmad/tea/config.yaml`
  - `tea_use_playwright_utils: true`
  - `tea_use_pactjs_utils: true`
  - `tea_pact_mcp: mcp`
  - `tea_browser_automation: auto`
  - `test_stack_type: auto`
  - `test_artifacts: _bmad-output/test-artifacts`
- Loaded BMM configuration from `_bmad/bmm/config.yaml`
  - `project_name: Gitdex`
  - `user_name: Chika Komari`
- Detected stack from repository files: `unknown`
  - Reason: this repository is still in planning mode and does not yet contain implementation markers such as `go.mod` or `package.json`.
  - Working assumption for test design: use the planned stack from architecture, namely `Go + Cobra + Bubble Tea + PostgreSQL + GitHub App + git worktree`.
- Loaded system-level artifacts:
  - `[_bmad-output/planning-artifacts/prd.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/planning-artifacts/prd.md)`
  - `[_bmad-output/planning-artifacts/architecture.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/planning-artifacts/architecture.md)`
  - `[_bmad-output/planning-artifacts/epics.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/planning-artifacts/epics.md)`
  - `[_bmad-output/planning-artifacts/ux-design-specification.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/planning-artifacts/ux-design-specification.md)`
- Loaded core TEA knowledge fragments:
  - `adr-quality-readiness-checklist.md`
  - `test-levels-framework.md`
  - `risk-governance.md`
  - `test-quality.md`
  - `probability-impact.md`
  - `test-priorities-matrix.md`
- Loaded extended fragments required by current config and architecture shape:
  - Playwright Utils API profile: `overview.md`, `api-request.md`, `auth-session.md`, `recurse.md`
  - Pact.js / contract design: `pactjs-utils-overview.md`, `pactjs-utils-consumer-helpers.md`, `pactjs-utils-provider-verifier.md`, `pactjs-utils-request-filter.md`
  - Pact MCP reference: `pact-mcp.md`
- Context note:
  - Browser exploration was skipped because there is no runnable UI or target URL yet.
  - Live Pact broker querying was not attempted because no broker credentials or tenant configuration are present in the repo.

## Step 3 - Testability and Risk Assessment

### Testability Concerns

1. `ACTIONABLE` - There is not yet a defined deterministic simulator for GitHub App auth, webhook replay, queue timing, and external failure injection.
2. `ACTIONABLE` - The architecture requires one governed write pipeline for CLI, chat, API, schedule, and webhook entry points, but test hooks for proving that invariant are not yet specified.
3. `ACTIONABLE` - Audit lineage, handoff packs, and artifact round-trips depend on shared `task_id` and `correlation_id` primitives.
4. `ACTIONABLE` - `single-writer-per-repo-ref` and isolated `git worktree` execution are architectural requirements, but no lock/lease test harness is described yet.
5. `ACTIONABLE` - Versioned contract testing for CLI JSON/YAML, HTTP API payloads, plans, reports, campaigns, and handoff artifacts must be first-class.

### Testability Assessment Summary

- Strong points already present in the planning set:
  - Explicit task lifecycle and campaign state models
  - Clear separation between control plane, execution plane, policy plane, and audit plane
  - Handoff pack, audit record, and contract field minimums already specified
  - Rich TUI and text-only parity defined as a product requirement rather than a nice-to-have
- ASRs identified:
  - `ACTIONABLE`: one governed write gate across all entry points
  - `ACTIONABLE`: replay-safe and idempotent ingress event handling
  - `ACTIONABLE`: isolated worktree execution with repo/ref lease control
  - `ACTIONABLE`: append-only audit ledger with shared IDs
  - `ACTIONABLE`: versioned machine-readable contracts and round-trip guarantees
  - `FYI`: GraphQL-first reads / REST-first writes split
  - `FYI`: deployment governance boundary instead of direct cloud imperative control
  - `FYI`: rich TUI and text-only modes share semantics even if rendering differs

### Risk Register Summary

| Risk ID | Category | Description | P | I | Score | Action |
| --- | --- | --- | --- | --- | --- | --- |
| R-001 | SEC | Policy/approval bypass across entry points enables unauthorized writes | 3 | 3 | 9 | Block |
| R-002 | OPS | Webhook replay/dedup/reconciliation defects create duplicate side effects or drift | 3 | 3 | 9 | Block |
| R-003 | DATA | Worktree isolation or repo/ref lease failure corrupts repository state | 2 | 3 | 6 | Mitigate |
| R-004 | DATA | Audit/correlation gaps make handoff and traceability incomplete | 2 | 3 | 6 | Mitigate |
| R-005 | TECH | No deterministic simulator/fault injection harness for core control-plane behavior | 3 | 2 | 6 | Mitigate |
| R-006 | TECH | CLI/API/artifact contract drift breaks integrations and external reuse | 2 | 3 | 6 | Mitigate |
| R-007 | BUS | Rich TUI, text-only, and cross-shell semantics diverge across platforms | 2 | 2 | 4 | Monitor |
| R-008 | OPS | Campaign execution lacks per-repo isolation and partial-success invariants | 2 | 3 | 6 | Mitigate |
| R-009 | OPS | GitHub App token/rate-budget failures degrade autonomy silently | 2 | 2 | 4 | Monitor |
| R-010 | BUS | LLM explanations drift from evidence or policy basis and reduce trust | 1 | 2 | 2 | Document |

## Step 4 - Coverage Plan and Execution Strategy

### Coverage Design Summary

- Priority distribution:
  - `P0`: ~2 scenarios
  - `P1`: ~7 scenarios
  - `P2`: ~8 scenarios
  - `P3`: ~4 scenarios
- Test level allocation:
  - `E2E`: cross-entry governed write path, webhook/reconciliation, campaigns, text-only parity
  - `API`: plan compilation, policy routing, machine-facing contracts, performance baselines
  - `Component`: explanation/evidence assembly and focused failure-path behavior
  - `Unit`: reserved for implementation-time domain logic and edge conditions

### Execution Strategy

- `PR`: run all functional Go unit/integration suites, Playwright API smoke checks, contract/schema conformance, and selected CLI parity checks; target `< 15 min`
- `Nightly`: run baseline performance, replay/reconciliation drills, GitHub auth/rate-limit synthetic faults, and cross-platform parity subsets
- `Weekly`: run long campaign soak, concurrency stress, kill-switch/fault-injection rehearsals, and exploratory terminal layout/accessibility checks

### Resource Estimates

- `P0`: `~4-7 days`
- `P1`: `~1.5-2.5 weeks`
- `P2`: `~1-1.5 weeks`
- `P3`: `~3-5 days`
- `Total`: `~2.5-4 weeks` for one QA owner, or `~1.5-2.5 weeks` with one QA plus one dev-in-test/platform partner

### Quality Gates

- `P0` pass rate: `100%`
- `P1` pass rate: `>= 95%`
- High-risk mitigations (`R-001` to `R-008` where score `>= 6`) complete or explicitly waived before release
- Versioned contract round-trip conformance: `100%`
- Audit lineage and handoff minimum-field conformance: `100%`
- Text-only Phase 1 core-flow parity on supported OS matrix: `100%`

## Step 5 - Generate Outputs and Validate

- Resolved execution mode: `sequential`
  - Reason: no supported subagent or agent-team runtime was available in this session.
- Output documents generated:
  - `[_bmad-output/test-artifacts/test-design-architecture.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/test-artifacts/test-design-architecture.md)`
  - `[_bmad-output/test-artifacts/test-design-qa.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/test-artifacts/test-design-qa.md)`
  - `[_bmad-output/test-artifacts/test-design/Gitdex-handoff.md](E:/Work/Engineering-Development/Gitdex/_bmad-output/test-artifacts/test-design/Gitdex-handoff.md)`
- Validation result:
  - System-level mode output pair created
  - Risk IDs and priorities are consistent across both documents
  - PR / Nightly / Weekly execution strategy retained
  - Resource estimates use ranges only
  - Architecture doc stays concerns-first and avoids QA recipe bloat
  - QA doc contains the execution recipe, dependencies, coverage matrix, and Playwright API example
- Open assumptions:
  - Implementation stack remains aligned with the current architecture document
  - GitHub sandbox, Pact broker or schema registry, and cross-platform CI runners will be provisioned before automation work begins
