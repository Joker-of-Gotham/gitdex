# Operator Guide

## Capability Interpretation

- `full`: expected API-backed lifecycle with reversible or compensating semantics defined in execution policy.
- `partial_mutate`: public mutation exists, but rollback or adjacent UI parity is incomplete.
- `inspect_only`: safe read surface only.
- `composed`: orchestration surface spanning multiple capabilities.

## Report Surfaces

- `flow-report`: workflow health, approval state, locks, dead-letter queue, and next retry.
- `platform-mutation-ledger`: adapter, adapter detail (`gh` binary or browser driver when applicable), coverage, rollback kind, diagnostics, and summary for each mutation.
- `memory-snapshot`: episodic memory, semantic facts, and task state.
- `failure-taxonomy`: grouped failure causes.
- `operator-report`: compact operator-facing summary over the same data sources.

## Recovery Semantics

- `reversible`: rollback is expected to restore prior state.
- `compensating`: rollback may create a compensating action rather than true restoration.
- `manual restore required`: operator intervention is mandatory and remains in the audit chain.

## Adapter Conditions

- `gh-backed`: GitHub CLI was required because REST coverage was insufficient or unavailable.
- `browser-backed`: execution is stubbed for operator completion and explicitly marked as such.

## Explicit Recovery Path

- Observe-only automation is an escalated safety mode, not a terminal state.
- Use `H` to clear observe-only mode and reset failure counters while keeping escalation history in the audit export.
- Use `Y` on the selected step when approval policy is blocking unattended execution for that step but you do not want to broaden the global trust policy.
- Resume the selected step with `R`, retry a dead-lettered step with `X`, or run compensating rollback with `C` as the follow-up recovery action.
