# Story 6.3: Approve, Exclude, and Intervene Per Repository Within a Campaign

Status: done

## Story

As a campaign operator,
I want to approve, exclude, retry, or take over individual repositories within a campaign,
So that one problematic repository does not force me to stop or rerun the entire fleet operation.

## Acceptance Criteria

1. **Given** a running or reviewable campaign **When** the operator acts on an individual repository inside that campaign **Then** Gitdex supports per-repository approval, exclusion, or intervention without invalidating the rest of the campaign.

2. **And** the campaign summary reflects partial completion and exceptions explicitly.

3. **And** repository-level interventions remain linked to the same campaign audit trail.

## Tasks / Subtasks

- [x] Task 1: 干预数据模型 (AC: #1, #3)
  - [x] 1.1 定义 `InterventionType` 枚举：`approve_repo`, `exclude_repo`, `retry_repo`, `override_plan`, `pause_repo`, `resume_repo`
  - [x] 1.2 定义 `InterventionRequest` 结构体：intervention_type, campaign_id, owner, repo, reason, actor, overrides
  - [x] 1.3 定义 `InterventionResult` 结构体：request, success, previous_status, new_status, message
  - [x] 1.4 所有结构体添加 `json` + `yaml` 标签，`snake_case` 格式

- [x] Task 2: 干预引擎 (AC: #1, #2)
  - [x] 2.1 定义 `InterventionEngine` 接口：`Execute(ctx, req) (*InterventionResult, error)`
  - [x] 2.2 实现 `DefaultInterventionEngine`，依赖 `CampaignStore`
  - [x] 2.3 实现 6 种干预类型：approve（设为 included）、exclude（设为 excluded）、retry（设为 pending）、override_plan（合并 overrides 到 PerRepoOverrides）、pause（暂停 campaign）、resume（恢复 campaign）
  - [x] 2.4 添加 empty CampaignID 验证
  - [x] 2.5 仓库未找到时返回失败结果
  - [x] 2.6 未知干预类型返回失败结果

- [x] Task 3: CLI 干预子命令 (AC: #1, #2)
  - [x] 3.1 在 `campaign.go` 中添加 `gitdex campaign approve <campaign_id> --repo owner/repo` 命令
  - [x] 3.2 添加 `gitdex campaign exclude <campaign_id> --repo owner/repo --reason "reason"` 命令
  - [x] 3.3 添加 `gitdex campaign retry <campaign_id> --repo owner/repo` 命令
  - [x] 3.4 添加 `gitdex campaign intervene <campaign_id> --repo owner/repo --action <action> [--overrides k=v]` 命令
  - [x] 3.5 支持 text/JSON/YAML 输出格式

- [x] Task 4: 测试 (AC: #1-#3)
  - [x] 4.1 `internal/campaign/intervention_test.go` — ApproveRepo、ExcludeRepo、RepoNotFound 单元测试
  - [x] 4.2 `test/integration/campaign_intervention_command_test.go` — approve/exclude/retry/intervene CLI 集成测试

## Dev Notes

### 关键实现细节

- `DefaultInterventionEngine` 接受 `CampaignStore` 依赖，通过 `GetCampaign` 获取战役，修改目标仓库状态后通过 `UpdateCampaign` 持久化
- `InterventionApproveRepo` 将仓库 `InclusionStatus` 改为 `included`
- `InterventionExcludeRepo` 改为 `excluded`，并附加 reason 到 message
- `InterventionRetryRepo` 重置为 `pending`
- `InterventionOverridePlan` 合并 overrides map 到 `PerRepoOverrides`
- `InterventionPauseRepo` / `InterventionResumeRepo` 修改整个 campaign 的 Status
- 空 CampaignID 直接返回失败结果而非 error

### 文件结构

```
internal/campaign/
├── intervention.go          # InterventionType, InterventionRequest, InterventionResult, InterventionEngine, DefaultInterventionEngine
├── intervention_test.go     # 单元测试

internal/cli/command/
├── campaign.go              # approve/exclude/retry/intervene 子命令

test/integration/
├── campaign_intervention_command_test.go  # CLI 集成测试
```

### 设计决策

1. **干预即状态修改**：每种干预类型映射到 `RepoTarget.InclusionStatus` 或 `Campaign.Status` 的特定状态转换
2. **Overrides 合并语义**：`override_plan` 不替换整个 overrides map，而是逐键合并，保留已有 overrides
3. **按值传递 Request**：`InterventionRequest` 使用值类型而非指针，避免调用者修改影响引擎内部状态
4. **错误 vs 失败**：仓库未找到或 campaign 未找到返回 `Success=false` 的 result 而非 Go error，保持 CLI 友好

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Story 6.3 验收标准]
- [Source: _bmad-output/planning-artifacts/architecture.md — Campaign 子系统]
- [Source: _bmad-output/planning-artifacts/prd.md — FR39]

## Dev Agent Record

### Completion Notes List

- **Task 1:** 干预数据模型 — 6 种 InterventionType 枚举、InterventionRequest（含 overrides map）、InterventionResult（含 previous/new status）。全部 JSON/YAML snake_case 标签。
- **Task 2:** DefaultInterventionEngine — 通过 CampaignStore 实现读-改-写模式。6 种干预操作：approve（included）、exclude（excluded+reason）、retry（pending）、override（合并 map）、pause（StatusPaused）、resume（StatusExecuting）。空 CampaignID 验证、未知仓库和未知类型均返回失败。
- **Task 3:** CLI 子命令 — approve/exclude/retry/intervene 均支持 --repo 标志解析 owner/repo。intervene 支持 --action 和 --overrides 标志。text/JSON/YAML 输出。
- **Task 4:** 3 个单元测试（ApproveRepo/ExcludeRepo/RepoNotFound）+ CLI 集成测试覆盖所有 4 个子命令。

### File List

**New files:**
- `internal/campaign/intervention.go`
- `internal/campaign/intervention_test.go`
- `test/integration/campaign_intervention_command_test.go`

**Modified files:**
- `internal/cli/command/campaign.go` — 添加 approve/exclude/retry/intervene 子命令
