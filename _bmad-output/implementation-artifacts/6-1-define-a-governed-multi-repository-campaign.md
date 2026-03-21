# Story 6.1: Define Governed Multi-Repository Campaign (FR37)

Status: done

## Story

As an operator with an authorized repository set,
I want to create a campaign in Gitdex that stores the campaign intent, target repository set, and governing scope as a first-class object,
so that I can review the campaign definition before any per-repository work begins and campaign creation respects repository authorization boundaries.

## Acceptance Criteria

1. **Given** an operator with an authorized repository set **When** they create a campaign in Gitdex **Then** Gitdex stores the campaign intent, target repository set, and governing scope as a first-class object **And** campaign creation respects repository authorization boundaries **And** the operator can review the campaign definition before any per-repository work begins

## Tasks / Subtasks

- [x] Task 1: 定义 Campaign 与 RepoTarget 领域模型
  - [x] 1.1 在 `internal/campaign/campaign.go` 中定义 `CampaignStatus`（draft, planning, executing, paused, completed, cancelled）
  - [x] 1.2 定义 `InclusionStatus`（included, excluded, pending）
  - [x] 1.3 定义 `RepoTarget`（Owner, Repo, InclusionStatus, PerRepoOverrides）
  - [x] 1.4 定义 `Campaign`（CampaignID, Name, Description, Status, TargetRepos, PlanTemplate, PolicyBundleID, CreatedBy, CreatedAt, UpdatedAt）

- [x] Task 2: 实现 Campaign 存储层
  - [x] 2.1 定义 `CampaignStore` 接口：SaveCampaign、GetCampaign、ListCampaigns、UpdateCampaign
  - [x] 2.2 实现 `MemoryCampaignStore`，含 duplicate repo 校验
  - [x] 2.3 SaveCampaign 时自动生成 `camp_` + UUID 前缀，copyCampaign 深拷贝防外部修改

- [x] Task 3: 注册 CLI campaign 命令
  - [x] 3.1 `campaign create --name <name> [--description <desc>]`：创建 campaign
  - [x] 3.2 `campaign show <campaign_id>`：展示 campaign 详情
  - [x] 3.3 `campaign list`：列出全部 campaign
  - [x] 3.4 `campaign add-repo <campaign_id> --repo owner/repo`：添加 target repo
  - [x] 3.5 `campaign remove-repo <campaign_id> --repo owner/repo`：移除 target repo
  - [x] 3.6 支持 JSON/YAML 与文本输出

- [x] Task 4: 编写单元、契约与集成测试
  - [x] 4.1 `internal/campaign/campaign_test.go`：SaveAndGet、ListCampaigns、UpdateCampaign、GetNotFound
  - [x] 4.2 `test/conformance/campaign_contract_test.go`：Campaign/RepoTarget JSON 契约、CampaignStatus 枚举校验
  - [x] 4.3 `test/integration/campaign_command_test.go`：命令注册、create 必填 name、show 必填 id、list/add-repo 参数校验

## Dev Notes

### 关键实现细节

- **Campaign**：包含 TargetRepos 切片，每个 RepoTarget 有 Owner/Repo 与 InclusionStatus
- **MemoryCampaignStore**：SaveCampaign 时校验 TargetRepos 无重复 owner/repo；空 CampaignID 时生成 `camp_` + uuid 前 8 位
- **add-repo**：使用 `parseOwnerRepo` 解析 owner/repo，InclusionStatus 默认为 pending；已存在则静默返回
- **remove-repo**：过滤掉指定 repo，无匹配时静默返回

### 文件结构

```
internal/campaign/campaign.go
internal/cli/command/campaign.go   # create/show/list/add-repo/remove-repo (+ matrix/status/approve/exclude/retry/intervene 在 6.2/6.3)
internal/campaign/campaign_test.go
test/conformance/campaign_contract_test.go
test/integration/campaign_command_test.go
```

### 设计决策

- **Campaign 作为一等对象**：CampaignID、Name、Description、Status、TargetRepos、PlanTemplate、PolicyBundleID 均持久化
- **RepoTarget**：支持 PerRepoOverrides 便于后续 override_plan 等干预操作
- **Repository authorization**：当前实现为内存存储，授权边界待与 GitHub App 集成时强化

### References

- Epic 6: Campaign Orchestration
- FR37: Governed multi-repository campaign definition

## Dev Agent Record

### Completion Notes List

- Task 1：定义 CampaignStatus、InclusionStatus、RepoTarget、Campaign
- Task 2：实现 CampaignStore 与 MemoryCampaignStore，含 copyCampaign 深拷贝
- Task 3：实现 campaign create/show/list/add-repo/remove-repo 子命令
- Task 4：编写 campaign_test、campaign_contract_test、campaign_command_test

### File List

**New files:**
- `internal/campaign/campaign.go`
- `internal/campaign/campaign_test.go`
- `internal/cli/command/campaign.go`
- `test/conformance/campaign_contract_test.go`
- `test/integration/campaign_command_test.go`
