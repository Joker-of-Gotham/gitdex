# Story 4.3: Triage, Prioritize, and Summarize Inbound Activity (FR20)

Status: done

## Story

As a maintainer,
I want to triage and summarize incoming collaboration activity with priorities and actionable next steps,
so that I can focus on the most important items and scope work by repository or campaign boundary.

## Acceptance Criteria

1. **Given** a repository or campaign scope with incoming collaboration activity **When** the maintainer asks Gitdex for triage or summary support **Then** Gitdex returns a prioritized summary with explicit reasons, grouping, and actionable next steps.
2. **Given** a scope boundary **When** the maintainer requests a summary **Then** the summary can be scoped to a repository or campaign boundary.
3. **Given** a triage result **When** the operator inspects it **Then** the operator can inspect the underlying objects behind any triage recommendation.

## Tasks / Subtasks

- [x] Task 1: Define triage domain model (AC: #1)
  - [x] 1.1 Define `TriagePriority` (critical, high, medium, low, informational)
  - [x] 1.2 Define `TriageResult` with ObjectRef, Priority, Reason, SuggestedAction, Tags
  - [x] 1.3 Define `ActivitySummary` with Period, TotalObjects, ByType, ByPriority, TopItems, GeneratedAt
  - [x] 1.4 Define `TriageEngine` interface (Triage, Summarize)
- [x] Task 2: Implement RuleBasedTriageEngine (AC: #1)
  - [x] 2.1 Label rules: security→critical, bug→high, stale/wontfix→low, docs/question→informational
  - [x] 2.2 Closed state→low with "no action"
  - [x] 2.3 Default medium with "review"
  - [x] 2.4 ObjectRef() for issues vs PRs (owner/repo#N vs owner/repo#pr/N)
- [x] Task 3: Implement Summarize with period and top items (AC: #1, #2)
  - [x] 3.1 ByType, ByPriority aggregation from triage results
  - [x] 3.2 Sort by priority order, take top 10
  - [x] 3.3 Period passed as string (e.g. 7d, 24h)
- [x] Task 4: Implement `collab triage` and `collab summary` commands (AC: #1–#3)
  - [x] 4.1 triage: --repo required, filter open objects, run Triage per object
  - [x] 4.2 summary: --repo, --period (default 7d)
  - [x] 4.3 renderTriageText: ObjectRef, Priority, Reason, SuggestedAction, Tags
  - [x] 4.4 renderSummaryText: ByType, ByPriority, TopItems
  - [x] 4.5 Structured output for triage_results and summary JSON
- [x] Task 5: Add tests (AC: #1–#3)
  - [x] 5.1 triage_test.go: SecurityLabel, BugLabel, StaleLabel, DefaultMedium, Summarize, ObjectRef
  - [x] 5.2 collab_triage_command_test.go: RequiresRepo, WithRepo, JSONOutput, SummaryCommand
  - [x] 5.3 collab_triage_contract_test.go: TriageResult, ActivitySummary JSON contract

## Dev Notes

### 关键实现细节
- `RuleBasedTriageEngine` 使用标签优先级：security > bug > stale/wontfix > docs/question > 默认 medium。
- `ObjectRef(obj)` 对 PR 返回 `owner/repo#pr/N`，对 issue 返回 `owner/repo#N`。
- `Summarize` 的 `period` 为元数据字段，不参与过滤；实际过滤依赖 ListObjects 的 filter（如 state）。
- TopItems 按 priorityOrder 排序后取前 10 个。

### 文件结构
```
internal/collaboration/triage.go          # TriagePriority, TriageResult, ActivitySummary, TriageEngine, RuleBasedTriageEngine
internal/cli/command/collab.go            # newCollabTriageCommand, newCollabSummaryCommand, renderTriageText, renderSummaryText
internal/collaboration/triage_test.go
test/integration/collab_triage_command_test.go
test/conformance/collab_triage_contract_test.go
```

### 设计决策
- 规则引擎为简单 label-based，未接入 ML 或自定义规则配置。
- 操作者可通过 `collab show <ObjectRef>` 查看底层对象，满足 AC#3。
- period 未解析为时间窗过滤，由 caller 控制 ListObjects 的范围。

### References
- FR20: Triage, prioritize, summarize inbound activity
- Story 4.1: ObjectStore, ObjectFilter, CollaborationObject
