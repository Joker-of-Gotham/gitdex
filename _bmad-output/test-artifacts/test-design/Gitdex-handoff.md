---
title: 'TEA Test Design -> BMAD Handoff Document'
version: '1.0'
workflowType: 'testarch-test-design-handoff'
inputDocuments:
  - _bmad-output/test-artifacts/test-design-architecture.md
  - _bmad-output/test-artifacts/test-design-qa.md
  - _bmad-output/planning-artifacts/epics.md
sourceWorkflow: 'testarch-test-design'
generatedBy: 'TEA Master Test Architect'
generatedAt: '2026-03-18'
projectName: 'Gitdex'
---

# TEA -> BMAD Integration Handoff

## Purpose

This handoff bridges the completed TEA system-level test design with BMAD implementation planning so that risk controls, testability prerequisites, and release gates remain attached to the right epics and stories as execution begins.

## TEA Artifacts Inventory

| Artifact | Path | BMAD Integration Point |
| --- | --- | --- |
| Architecture Test Design | `_bmad-output/test-artifacts/test-design-architecture.md` | Pre-implementation blockers, ASRs, high-risk mitigation ownership |
| QA Test Design | `_bmad-output/test-artifacts/test-design-qa.md` | Story acceptance coverage, execution sequencing, QA dependencies |
| Workflow Progress Log | `_bmad-output/test-artifacts/test-design-progress.md` | Traceability of assumptions and validation decisions |

## Epic-Level Integration Guidance

### Risk References

- **Epic 2 - Governed Planning and Safe Single-Repository Action**
  - Must absorb `R-001`, `R-003`
  - No mutative story in Epic 2 should close without governed-write parity and worktree isolation evidence
- **Epic 3 - Governance, Policy, Audit, and Emergency Control**
  - Must absorb `R-001`, `R-004`, `R-006`
  - This epic carries the contract for policy bundles, audit lineage, shared IDs, and emergency controls
- **Epic 5 - Background Autonomy, Recovery, and Human Takeover**
  - Must absorb `R-002`, `R-004`, `R-005`, `R-009`
  - Autonomy should not proceed without replay-safe ingestion and deterministic recovery harnesses
- **Epic 6 - Multi-Repository Campaign Operations**
  - Must absorb `R-008`
  - Campaign semantics are only acceptable with per-repo intervention and partial-success invariants
- **Epic 7 - Platform Integrations and Structured Exchange**
  - Must absorb `R-006`
  - Machine-facing APIs and exported artifacts need versioned schema and round-trip compatibility from the first public contract

### Quality Gates

- **Gate A - Before first mutative story implementation**
  - `B-001`, `B-004`, and `B-005` resolved
  - Shared write facade, audit primitives, and repo/ref lease model available
- **Gate B - Before webhook or schedule-driven autonomy**
  - `B-002` and `B-003` resolved
  - Deterministic simulator and replay/reconciliation contract available
- **Gate C - Before campaign beta**
  - `R-008` mitigation implemented
  - Per-repo plan/state/approval/intervention semantics validated
- **Gate D - Before machine API beta**
  - `R-006` mitigation implemented
  - Versioned schema, compatibility fixtures, and round-trip tests in place

## Story-Level Integration Guidance

### P0/P1 Test Scenarios -> Story Acceptance Criteria

- **Story 2.1 / 2.2**
  - Acceptance criteria should explicitly require plan compilation, risk explanation, policy verdict visibility, and no hidden writes before approval.
- **Story 2.6**
  - Acceptance criteria should require isolated worktree execution and lease protection for concurrent repo/ref access.
- **Story 3.2**
  - Acceptance criteria should require policy parity across CLI, chat, API, and autonomous entry points.
- **Story 3.3**
  - Acceptance criteria should require task -> plan -> policy -> approvals -> outcome -> evidence lineage traversal.
- **Story 5.3 / 5.5**
  - Acceptance criteria should require replay-safe webhook ingestion, explainable degraded states, and reconciliation recovery.
- **Story 5.6**
  - Acceptance criteria should require handoff pack completeness and exportability.
- **Story 6.2 / 6.3**
  - Acceptance criteria should require per-repo intervention without collapsing remaining campaign work.
- **Story 7.1 / 7.2 / 7.3 / 7.4**
  - Acceptance criteria should require versioned schemas, backward-compatible fields, and artifact round-trip behavior.

### Stable Selector and ID Requirements

- Use stable `task_id`, `plan_id`, `campaign_id`, `audit_id`, and `correlation_id` in all machine-readable outputs.
- Rich TUI and text-only views should expose stable object labels so parity tests can assert semantics without rendering coupling.
- Exported plans, reports, and handoff artifacts must include schema version and required field sets explicitly.

## Risk-to-Story Mapping

| Risk ID | Category | P×I | Recommended Story/Epic | Test Level |
| --- | --- | --- | --- | --- |
| R-001 | SEC | 3x3 | Stories `2.2`, `3.2`; Epics `2`, `3` | E2E / API |
| R-002 | OPS | 3x3 | Stories `5.3`, `5.5`; Epic `5` | E2E |
| R-003 | DATA | 2x3 | Story `2.6`; Epic `2` | API |
| R-004 | DATA | 2x3 | Stories `3.3`, `5.6`; Epics `3`, `5` | API |
| R-005 | TECH | 3x2 | Foundation + Stories `5.3`, `5.5`; Epic `5` | API / E2E |
| R-006 | TECH | 2x3 | Stories `7.1-7.4`; Epic `7` | API |
| R-007 | BUS | 2x2 | Story `1.5`; Epic `1` | E2E |
| R-008 | OPS | 2x3 | Stories `6.2`, `6.3`; Epic `6` | E2E |
| R-009 | OPS | 2x2 | Stories `4.5`, `5.3`; Epics `4`, `5` | API |
| R-010 | BUS | 1x2 | Stories `1.4`, `2.1`; Epics `1`, `2` | Component |

## Recommended BMAD -> TEA Workflow Sequence

1. **TEA Test Design** -> completed system-level risk and coverage design
2. **BMAD Create Story / Sprint Planning** -> embed the risk gates and blockers into story sequencing
3. **TEA ATDD** -> generate failing acceptance tests for each story carrying P0/P1 obligations
4. **BMAD Implementation** -> implement against shared contracts, simulator hooks, and quality gates
5. **TEA Automate / CI / Trace** -> expand automation, pipeline gates, and requirement coverage checks

## Phase Transition Quality Gates

| From Phase | To Phase | Gate Criteria |
| --- | --- | --- |
| Test Design | Story Execution | All blocker items `B-001` to `B-005` assigned and sequenced |
| Story Execution | Mutative Implementation | Shared write facade, audit primitives, and repo/ref lease model available |
| Foundation | Event-Driven Autonomy | Deterministic simulator and replay/reconciliation fixtures available |
| Campaign Implementation | Campaign Beta | Partial-success and per-repo intervention acceptance suite passing |
| API / Artifact Beta | External Integration | Versioned schemas and round-trip compatibility suite passing |
