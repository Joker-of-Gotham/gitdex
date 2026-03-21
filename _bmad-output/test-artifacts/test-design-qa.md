---
stepsCompleted: ['step-05-generate-output']
lastStep: 'step-05-generate-output'
lastSaved: '2026-03-18'
workflowType: 'testarch-test-design'
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/planning-artifacts/epics.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
  - _bmad-output/test-artifacts/test-design-architecture.md
---

# Test Design for QA: Gitdex Phase 1 Governed Repository Operations

**Purpose:** Gitdex Phase 1 的 QA 执行配方，定义要测什么、如何测、先卡住哪些前置条件，以及哪些验证要进 PR / Nightly / Weekly。

**Date:** 2026-03-18  
**Author:** Chika Komari  
**Status:** Draft  
**Project:** Gitdex

**Related:** 架构可测性缺口与高风险缓解见 `test-design-architecture.md`。

---

## Executive Summary

**Scope:** 覆盖 Gitdex Phase 1 的受治理写路径、repo state summary、policy/approval、task lifecycle、handoff、multi-repo campaign、machine-facing contracts、terminal parity，以及核心恢复路径。

**Risk Summary:**

- Total Risks: `10` (`7` high-priority score `>= 6`, `2` medium, `1` low)
- Critical Categories: `SEC`, `OPS`, `DATA`

**Coverage Summary:**

- P0 tests: `~2`
- P1 tests: `~7`
- P2 tests: `~8`
- P3 tests: `~4`
- **Total:** `~21` scenarios (`~2.5-4 weeks` with 1 QA owner)

---

## Not in Scope

| Item | Reasoning | Mitigation |
| --- | --- | --- |
| **Direct cloud-imperative deployment execution** | Phase 1 明确定义为 deployment governance，不是新的 CD engine | 通过 adapter contract 和 upstream pipeline validation 验证 |
| **Full GHES / enterprise SSO matrix** | 当前目标是 local workstation 与 single-tenant team baseline | 后续以 dedicated enterprise compatibility plan 覆盖 |
| **General browser UI testing** | Gitdex 不是 browser-first 产品，核心 operator flow 在终端完成 | 仅验证 exported HTML/report artifacts 的结构与可读性 |
| **Third-party non-GitHub integrations beyond versioned machine contracts** | 当前先验证 Gitdex 的控制面契约，不追求每个外部生态适配器 | 通过 API/artifact schema compatibility 先锁定边界 |

**Note:** 以上排除项默认需由 PM / Architect / QA 在进入实现前确认接受。

---

## Dependencies & Test Blockers

### Backend/Architecture Dependencies (Pre-Implementation)

1. **Shared governed write facade** - Architecture / Security - before Stories `2.2`, `3.2`
   - QA 需要所有 mutative entry points 共享同一 façade，才能做 cross-entry parity assertions。
   - 若缺失，该类测试只能退化为入口级冒烟，无法证明系统治理一致性。

2. **Deterministic GitHub + webhook simulator** - Platform Tooling - before Stories `5.3`, `5.5`
   - QA 需要 provider states、webhook replay fixtures、fake clock 和 failure injection knobs。
   - 若缺失，`R-002`, `R-005`, `R-009` 将被迫依赖慢环境和人工复现。

3. **Repo fixture factory + repo/ref lease controls** - Git Execution - before Story `2.6`
   - QA 需要可重复生成 fixture repos、protected refs、concurrent mutation scenarios。
   - 若缺失，worktree / lease isolation 测试将高度脆弱。

4. **Shared schema registry for task/plan/audit/handoff/campaign** - API / Platform - before Stories `7.1-7.4`
   - QA 需要明确 schema version、required fields、compatibility fixtures。
   - 若缺失，machine contract drift 只能在集成方环境暴露。

### QA Infrastructure Setup (Pre-Implementation)

1. **Fixture and Seeding Layer** - QA + Platform
   - repo fixture factory with divergent branches, protected refs, staged diffs
   - webhook fixture catalog for duplicate / reordered / missing deliveries
   - policy/approval fixtures for allow / block / escalate cases
   - failure-path fixtures for blocked, quarantined, and handoff states

2. **Test Environments** - QA
   - Local: `gitdexd + PostgreSQL + fixture repos + fake GitHub adapter`
   - CI PR lane: Linux runner with contract tests, API smoke, audit/handoff conformance
   - Scheduled lane: Windows / macOS / Linux parity matrix and long-running recovery drills

**Example API factory pattern using Playwright Utils:**

```typescript
import { test } from '@seontechnologies/playwright-utils/api-request/fixtures';
import { expect } from '@playwright/test';

test('@P1 @API compile governed plan from intent', async ({ apiRequest }) => {
  const { status, body } = await apiRequest({
    method: 'POST',
    path: '/v1/intents/compile',
    body: {
      intent: 'sync upstream and prepare a low-risk maintenance plan',
      scope: { repository: 'acme/gitdex-fixture' },
      mode: 'dry_run',
    },
  });

  expect(status).toBe(202);
  expect(body.data.plan.state).toBe('plan_ready');
  expect(body.data.plan.risk.level).toBeDefined();
});
```

---

## Risk Assessment

**Note:** 完整风险详情见架构文档；此处只保留与 QA 覆盖直接相关的映射。

### High-Priority Risks (Score >= 6)

| Risk ID | Category | Description | Score | QA Test Coverage |
| --- | --- | --- | --- | --- |
| **R-001** | SEC | 多入口 policy/approval bypass | **9** | Cross-entry governed write parity suite covering CLI/chat/API/webhook |
| **R-002** | OPS | replay / dedup / reconciliation drift | **9** | webhook replay + duplicate side-effect prevention + terminal state convergence |
| **R-003** | DATA | worktree / repo-ref lease isolation failure | **6** | concurrent mutation integration suite with protected target fixtures |
| **R-004** | DATA | audit / handoff lineage incomplete | **6** | lineage completeness checks from intent to audit close and handoff export |
| **R-005** | TECH | missing deterministic simulator blocks fast feedback | **6** | provider-state-driven API/E2E suites in PR with no external dependency |
| **R-006** | TECH | CLI/API/artifact contract drift | **6** | schema round-trip, compatibility fixtures, versioned contract checks |
| **R-008** | OPS | campaign blast radius without per-repo isolation | **6** | multi-repo partial-success acceptance suite with intervention paths |

### Medium/Low-Priority Risks

| Risk ID | Category | Description | Score | QA Test Coverage |
| --- | --- | --- | --- | --- |
| R-007 | BUS | terminal parity drift across shells and modes | 4 | text-only and cross-shell conformance matrix |
| R-009 | OPS | GitHub App auth/rate degradation not surfaced clearly | 4 | synthetic token expiry and rate-limit scenarios |
| R-010 | BUS | explanation quality drifts from evidence/policy basis | 2 | component-level explanation review set and regression snapshots |

---

## Entry Criteria

**QA testing cannot begin until ALL of the following are met:**

- [ ] Shared governed write facade available for all mutative entry points
- [ ] Deterministic simulator and webhook replay fixtures available
- [ ] Repo fixture factory and disposable worktree root ready
- [ ] Versioned schemas checked into the repo for task/plan/audit/handoff/campaign
- [ ] Local and CI Postgres-backed environment accessible
- [ ] GitHub App sandbox installation and webhook secrets provisioned
- [ ] Test data factories, approval fixtures, and failure-path fixtures prepared

## Exit Criteria

**Testing phase is complete when ALL of the following are met:**

- [ ] All `P0` tests passing
- [ ] `P1` pass rate `>= 95%`
- [ ] No open blocker risks with score `9`
- [ ] All score `6-8` mitigations implemented or explicitly accepted with owner and timeline
- [ ] CLI/API/artifact round-trip conformance `100%`
- [ ] Audit lineage and handoff minimum required fields `100%`
- [ ] Text-only core-flow parity passes on supported OS matrix
- [ ] Baseline NFR rehearsals for query latency, webhook ack path, handoff latency, pause/kill-switch latency completed

---

## Test Coverage Plan

**IMPORTANT:** `P0/P1/P2/P3` 表示优先级与风险级别，不表示执行时机。执行时机见后面的 `Execution Strategy`。

### P0 (Critical)

**Criteria:** Blocks core functionality + High risk (`>= 6`) + No workaround + Directly challenges the product trust model

| Test ID | Requirement | Test Level | Risk Link | Notes |
| --- | --- | --- | --- | --- |
| **P0-001** | All mutative entry points must require plan -> policy -> approval before side effects | E2E | R-001 | Matrix includes CLI, chat, API, webhook-triggered task |
| **P0-002** | Webhook replay, dedup, and reconciliation must not create duplicate writes and must converge to explainable state | E2E | R-002 | Covers duplicate, reordered, and missing delivery patterns |

**Total P0:** `~2` tests

---

### P1 (High)

**Criteria:** High-value workflows + high risk (`>= 6`) + key trust, governance, or integration paths

| Test ID | Requirement | Test Level | Risk Link | Notes |
| --- | --- | --- | --- | --- |
| **P1-001** | Compile structured plan from command/chat with dry-run risk explanation | API | R-005 | Validates plan drafting, risk shaping, evidence basis |
| **P1-002** | Approval routing and policy verdict explanation for escalated actions | API | R-001 | Covers allow, block, escalate, and downgrade semantics |
| **P1-003** | Controlled local file modification executes inside isolated worktree with repo/ref lease enforcement | API | R-003 | Concurrent mutation and protected target cases |
| **P1-004** | Failed or blocked task emits complete handoff pack within governed terminal state | API | R-004 | Validates required fields and recommended next actions |
| **P1-005** | CLI JSON/YAML, HTTP API payloads, and exported artifacts remain schema-compatible and round-trippable | API | R-006 | Contract fixtures across plan, report, campaign, handoff |
| **P1-006** | Campaign supports per-repo review, exclusion, retry, and intervention without collapsing whole wave | E2E | R-008 | Partial success and exception surfacing are mandatory |
| **P1-007** | Audit query can navigate intent -> plan -> policy -> approvals -> outcome -> evidence | API | R-004 | Append-only lineage and related object traversal |

**Total P1:** `~7` tests

---

### P2 (Medium)

**Criteria:** Important secondary flows + medium risk (`3-5`) + regression prevention for operator trust

| Test ID | Requirement | Test Level | Risk Link | Notes |
| --- | --- | --- | --- | --- |
| **P2-001** | Repo digital twin summary merges local Git, PR, issue, workflow, and deployment signals accurately | API | R-005 | Read-model fidelity and evidence linkage |
| **P2-002** | Event, schedule, API, and operator triggers preserve trigger source and lifecycle history | API | R-002 | Focus on task creation semantics rather than write effects |
| **P2-003** | Pause, resume, cancel, and kill-switch commands produce explicit governed states | E2E | R-004 | Includes queued and running task cases |
| **P2-004** | Text-only mode supports state view, plan review, approval/rejection, handoff view, and audit query | E2E | R-007 | Core Phase 1 parity requirement |
| **P2-005** | PowerShell, bash, and zsh preserve command discovery, exit semantics, and machine-readable outputs | E2E | R-007 | Cross-shell conformance, not UI styling |
| **P2-006** | Deployment governance respects environment rules and records approval lineage without direct cloud imperative control | API | R-009 | Governance adapter path only |
| **P2-007** | GitHub App token expiry and rate-limit pressure degrade into explicit retry/blocked states | API | R-009 | No silent retries without surfaced state |
| **P2-008** | Recommendation mode explanations cite policy basis and evidence and remain side-effect-free | Component | R-010 | LLM output must not imply hidden execution |

**Total P2:** `~8` tests

---

### P3 (Low)

**Criteria:** Exploratory, benchmark, and presentation-layer confidence checks

| Test ID | Requirement | Test Level | Risk Link | Notes |
| --- | --- | --- | --- | --- |
| **P3-001** | Exported HTML/report artifacts preserve semantic structure and accessibility basics | Component | R-010 | Focus on headings, tables, and readable status semantics |
| **P3-002** | Terminal layout regressions across `80/100/140` columns remain navigable | E2E | R-007 | Exploratory plus snapshot-based smoke checks |
| **P3-003** | `20`-repo campaign soak maintains per-repo visibility and exception reporting | E2E | R-008 | Long-running scheduled confidence check |
| **P3-004** | Baseline read/query and plan-generation latency benchmarks stay within early targets | API | R-005 | Supports NFR rehearsal rather than release blocking |

**Total P3:** `~4` tests

---

## Execution Strategy

**Philosophy:** 能在 PR 里跑完的功能验证尽量都放进 PR。只有昂贵、长耗时或强依赖多平台/长时运行的验证才延后到 Nightly 或 Weekly。

### Every PR: Functional and Contract Validation (~12-15 min target)

- `go test` unit/integration suites for planning, policy, state machine, audit primitives, and Git executor seams
- Playwright API smoke for governed write path, plan compilation, and key failure/handoff flows
- Schema and round-trip conformance for CLI JSON/YAML, HTTP payloads, plans, reports, campaigns, and handoff artifacts
- Selected text-only CLI parity checks and exit-code conformance

### Nightly: Baseline NFR and Cross-Platform Confidence (~30-60 min)

- Baseline latency and status-refresh rehearsals for `NFR1-NFR5`
- webhook replay / reconciliation drills for `NFR8-NFR10`, `NFR24-NFR25`
- GitHub App auth/rate-limit synthetic faults
- Windows / macOS / Linux text-only parity subset

### Weekly: Long-Running and Chaos Validation (~2-4 hours)

- `20`-repo campaign soak and `100` active-task concurrency rehearsal
- failure injection across queue, Postgres, webhook ordering, and external API degradation
- exploratory terminal layout and keyboard-only navigation regression
- exported report / handoff artifact review samples

---

## QA Effort Estimate

**QA effort only** (不包含 DevOps、后端、平台基础设施开发工时):

| Priority | Count | Effort Range | Notes |
| --- | --- | --- | --- |
| P0 | `~2` | `~4-7 days` | 需要 simulator、cross-entry matrix、replay fixtures |
| P1 | `~7` | `~1.5-2.5 weeks` | 核心治理、schema、handoff、campaign 验证 |
| P2 | `~8` | `~1-1.5 weeks` | parity、degraded-state、read-model 与 recommendation 验证 |
| P3 | `~4` | `~3-5 days` | soak、layout、accessibility、baseline benchmarks |
| **Total** | `~21` | **`~2.5-4 weeks`** | **1 QA owner; with shared platform support the elapsed time can drop to `~1.5-2.5 weeks`** |

**Assumptions:**

- Includes test design refinement, implementation planning, fixture shaping, CI integration, and debugging buffer
- Excludes post-release maintenance and flaky-test burn-down
- Assumes pre-implementation blockers in the Dependencies section are resolved

---

## Implementation Planning Handoff

| Work Item | Owner | Target Milestone (Optional) | Dependencies/Notes |
| --- | --- | --- | --- |
| Build deterministic GitHub/webhook simulator and replay driver | Platform Tooling | Foundation sprint | Required for P0-002 and all replay/recovery tests |
| Add shared schema snapshots for task/plan/audit/handoff/campaign | API / Platform | Foundation sprint | Required for P1-005 and Story 7.x |
| Deliver repo fixture catalog and repo/ref lease harness | Git Execution | Before Story 2.6 | Required for P1-003 |
| Expose fake clock and failure-injection seams in orchestrator | Control Plane | Before Story 5.3 | Required for replay, retry, quarantine, handoff drills |
| Set up multi-OS CLI parity lane in CI | QA + Platform | Before Story 1.5 hardening | Required for P2-004 and P2-005 |

---

## Tooling & Access

| Tool or Service | Purpose | Access Required | Status |
| --- | --- | --- | --- |
| Local PostgreSQL + migration runner | durable state and projection validation | local/CI service access | Ready |
| Fixture repo catalog + disposable worktree root | Git mutation and drift scenarios | writable temp filesystem in CI | Pending |
| GitHub App sandbox org/repos + webhook secret | auth, webhook, installation-scope tests | GitHub sandbox credentials | Pending |
| Pact Broker or equivalent schema registry | contract compatibility and version tracking | service credentials | Pending |
| Multi-OS CI runners | Windows/Linux/macOS parity | CI platform provisioning | Pending |
| k6 or equivalent load tool | baseline latency and capacity rehearsal | CI/staging execution slot | Pending |

---

## Interworking & Regression

| Service/Component | Impact | Regression Scope | Validation Steps |
| --- | --- | --- | --- |
| **Planning + Policy Engine** | Governs all write-capable flows | plan compilation, risk shaping, policy parity | P0-001, P1-001, P1-002 |
| **Git Execution + Worktree Manager** | Repository mutation safety | worktree isolation, protected refs, concurrent tasks | P1-003 |
| **GitHub Ingress + Reconciliation** | Event-driven automation correctness | replay, dedup, degraded auth/rate conditions | P0-002, P2-007 |
| **Orchestrator + Task State Machine** | Long-running task reliability | state convergence, pause/resume/cancel, handoff timing | P1-004, P2-003 |
| **Audit / Handoff / Artifact Exports** | Explainability and reuse | lineage completeness, export round-trip, external reuse | P1-005, P1-007 |
| **Terminal Surfaces** | Operator trust and usability | text-only parity, shell semantics, layout resilience | P2-004, P2-005, P3-002 |
| **Campaign Controller** | Multi-repo blast radius management | per-repo visibility, intervention, partial success | P1-006, P3-003 |

---

## Appendix A: Code Examples & Tagging

**Suggested tags:**

- `@P0`, `@P1`, `@P2`, `@P3`
- `@Governance`, `@Replay`, `@Audit`, `@Campaign`, `@Parity`, `@Contract`

```typescript
import { test } from '@seontechnologies/playwright-utils/api-request/fixtures';
import { expect } from '@playwright/test';

test('@P0 @Governance blocked write without approval stays blocked', async ({ apiRequest }) => {
  const { status, body } = await apiRequest({
    method: 'POST',
    path: '/v1/tasks',
    body: {
      intent: 'apply a high-risk release action',
      scope: { repository: 'acme/gitdex-fixture' },
      mode: 'execute',
    },
  });

  expect(status).toBe(202);
  expect(body.data.policy_result.verdict).toBe('escalate');
  expect(body.data.task.state).toBe('awaiting_approval');
});
```

```bash
# Run critical governance checks
npx playwright test --grep "@P0|@Governance"

# Run nightly replay and audit coverage
npx playwright test --grep "@Replay|@Audit"
```

---

## Appendix B: Knowledge Base References

- `risk-governance.md` - 风险登记与高风险缓解分类
- `probability-impact.md` - `P x I` scoring and gate thresholds
- `test-levels-framework.md` - E2E/API/Component/Unit 分层原则
- `test-priorities-matrix.md` - P0-P3 优先级划分
- `test-quality.md` - 快反馈、可维护、非重复覆盖原则
- `api-request.md` / `auth-session.md` - Playwright API test patterns
- `pactjs-utils-overview.md` / `pact-mcp.md` - machine contract and broker-aware design support

---

**Generated by:** BMad TEA Agent  
**Workflow:** `_bmad/tea/testarch/bmad-testarch-test-design`  
**Version:** 5.0
