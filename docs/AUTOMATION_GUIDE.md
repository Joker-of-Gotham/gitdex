# Automation Guide

## Routing

- GitHub executor routing is modeled as `API -> gh -> browser -> explicit boundary failure`.
- Adapter-backed executions are written into the mutation ledger and labeled in the UI/report as `api-backed`, `gh-backed`, or `browser-backed`.
- Route failures are reported explicitly with the attempted chain so operators can see where `API`, `gh`, or `browser` resolution stopped.

## Unattended Safety

- Unattended execution only auto-runs safe inspect/validate flows unless trust policy allows more.
- Approval-required, partial/composed, adapter-backed, or non-reversible surfaces are surfaced through capability boundary metadata before execution.
- Diagnostics run before platform execution and may auto-repair one placeholder pass, warn, or block.

## Escalation

- Repeated executor failures can degrade automation into observe-only mode.
- Dead-letter accumulation can pause the workflow flow and trigger operator-facing escalation.
- Checkpoints persist workflow flow, schedule state, locks, failures, ledger, and observe-only state for later resume.

## Operator Recovery

- `H` clears observe-only state, resets automation failure counters, records a recovery timestamp, and keeps the escalation timestamp for audit history.
- `Y` approves the selected approval-required workflow step so unattended execution can continue without broadening trust policy for the whole capability.
- Selected-step pause, resume, retry, acknowledge, skip, and compensate all preserve ledger chain history.
- Recovery remains explicit: after `H`, operators can resume the paused step, retry a dead-lettered step, or run compensation while retaining the existing ledger chain.
- Browser-backed paths create explicit manual recovery records instead of faking success.
