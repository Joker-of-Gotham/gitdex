# Story 4.5: Prepare Release and Deployment Decisions (FR22)

Status: done

## Story

As an operator,
I want to prepare release and deployment decisions with readiness signals, checks, blockers, and approval requirements,
so that I can make informed decisions without bypassing deployment governance.

## Acceptance Criteria

1. **Given** a repository with release or deployment-relevant changes **When** the operator asks Gitdex to prepare a release or deployment decision **Then** Gitdex summarizes readiness signals, relevant checks, current blockers, and approval requirements without directly bypassing deployment governance.
2. **Given** a release assessment **When** the assessment completes **Then** the resulting decision package links back to the underlying repository and workflow evidence.
3. **Given** blocked or escalated cases **When** the assessment runs **Then** blocked or escalated cases clearly identify which reviewer or control gate must be satisfied next.

## Tasks / Subtasks

- [x] Task 1: Define release domain model (AC: #1, #2, #3)
  - [x] 1.1 Define `ReleaseStatus` (ready, blocked, pending)
  - [x] 1.2 Define `CheckStatus` (passed, failed, pending)
  - [x] 1.3 Define `CheckResult` with Name, Status, Details
  - [x] 1.4 Define `ReleaseReadiness` with RepoOwner, RepoName, Tag, Status, Blockers, IncludedPRs, CheckResults, ApprovalStatus, Notes, AssessedAt
  - [x] 1.5 Define `ReleaseInfo` for listing (Tag, Status, PublishedAt)
  - [x] 1.6 Define `ReleaseEngine` interface (Assess, ListReleases)
- [x] Task 2: Implement SimulatedReleaseEngine (AC: #1–#3)
  - [x] 2.1 Assess: validate owner/repo/tag; return mock ReleaseReadiness (ready, approved)
  - [x] 2.2 CheckResults: build, tests with CheckPassed
  - [x] 2.3 ListReleases: mock v1.0.0, v0.9.0 published
  - [x] 2.4 Blockers/ApprovalStatus for blocked case identification
- [x] Task 3: Implement `release assess` command (AC: #1, #2)
  - [x] 3.1 Required --repo owner/repo, --tag (e.g. v1.0.0)
  - [x] 3.2 renderReleaseReadiness: Status, Blockers, IncludedPRs, CheckResults, Approval, Notes
  - [x] 3.3 Structured output with repo_owner, repo_name, tag, status, assessed_at
- [x] Task 4: Implement `release list` command (AC: #2)
  - [x] 4.1 Required --repo owner/repo
  - [x] 4.2 renderReleaseList: Tag, Status, PublishedAt
  - [x] 4.3 Structured output with releases array
- [x] Task 5: Add tests (AC: #1–#3)
  - [x] 5.1 release_test.go: Assess, Assess InvalidInput, ListReleases, ListReleases InvalidInput
  - [x] 5.2 release_command_test.go: assess, list, JSON output, required flags
  - [x] 5.3 release_contract_test.go: ReleaseReadiness, CheckResult JSON contract

## Dev Notes

### 关键实现细节
- `SimulatedReleaseEngine` 不调用真实 CI/deployment API；返回固定 mock 数据，便于离线测试。
- `ReleaseReadiness.Blockers` 和 `ApprovalStatus` 用于 AC#3：blocked 时 Blockers 列出需满足的门禁。
- `IncludedPRs` 关联底层的 PR 证据；CheckResults 关联 workflow 检查。
- Notes 字段注明 "Simulated assessment - no real checks run"，明确不绕过治理。

### 文件结构
```
internal/collaboration/release.go         # ReleaseStatus, CheckStatus, CheckResult, ReleaseReadiness, ReleaseInfo, ReleaseEngine, SimulatedReleaseEngine
internal/cli/command/release.go          # newReleaseGroupCommand, newReleaseAssessCommand, newReleaseListCommand, renderRelease*
internal/collaboration/release_test.go
test/integration/release_command_test.go
test/conformance/release_contract_test.go
```

### 设计决策
- 模拟实现不执行真实部署，满足"不绕过 deployment governance" 的约束。
- ReleaseEngine 接口允许后续接入 GitHub Releases API、CI status API 等。
- Blockers 为字符串数组，可存储 "Waiting for approver: @alice" 等说明。

### References
- FR22: Prepare release and deployment decisions with approval-aware summaries
- Story 4.1: repo owner/name format
