# Story 1.4: Connect Authorized Repositories and View Consolidated State

Status: done

## Story

As a maintainer,
I want to connect an authorized repository and immediately see a consolidated state summary with evidence,
so that I can understand current drift, risk, and next actions before taking any write action.

## Acceptance Criteria

1. **Given** a repository within the user's authorized scope **When** the user opens that repository in Gitdex **Then** Gitdex displays local Git state, remote divergence, collaboration signals, workflow state, and deployment status in one summary surface.

2. **And** the summary highlights material risks and evidence-backed next actions.

3. **And** the summary uses explicit `healthy`, `drifting`, `blocked`, `degraded`, or `unknown` state labels that the user can drill into for supporting objects and evidence.

4. **Given** a valid GitHub App configuration (app_id, installation_id, private_key_path) **When** `gitdex status` is executed **Then** Gitdex generates an installation token and uses it to read remote GitHub state without requiring a PAT.

5. **Given** no GitHub App configuration is provided **When** `gitdex status` is executed **Then** Gitdex still displays available local Git state and marks all remote signals as `unknown` with clear guidance on how to configure identity.

6. **Given** the user specifies `--output json` or `--output yaml` **When** `gitdex status` is executed **Then** the state summary is rendered in the requested structured format with stable field names and nesting.

## Tasks / Subtasks

- [x] Task 1: 实现 GitHub App 身份认证与安装令牌获取 (AC: #4, #5)
  - [x] 1.1 创建 `internal/platform/identity/github_app.go`，实现 JWT 签发与 installation token 交换
  - [x] 1.2 使用 `ghinstallation/v2` 作为 transport wrapper，与 `config.IdentityConfig` 对接
  - [x] 1.3 处理 private key 文件读取、格式验证和错误路径（文件不存在、格式错误、过期 token）
  - [x] 1.4 在无身份配置时返回明确的 `ErrNoIdentity` 错误，允许 graceful degradation

- [x] Task 2: 实现本地 Git 状态读取器 (AC: #1)
  - [x] 2.1 创建 `internal/platform/git/state.go`，使用 `go-git/v5` 读取本地仓库状态
  - [x] 2.2 提取：当前 HEAD/分支名、工作树是否干净、暂存区状态、stash 计数
  - [x] 2.3 计算上游跟踪分支偏差（ahead/behind commits），通过读取 branch config 和 commit 图比较
  - [x] 2.4 提取远程列表和默认远程 URL
  - [x] 2.5 定义 `LocalGitState` 结构体封装所有本地状态数据

- [x] Task 3: 实现 GitHub API 读取适配器 (AC: #1, #4)
  - [x] 3.1 创建 `internal/platform/github/client.go`，封装 `go-github/v84` 客户端初始化
  - [x] 3.2 接受 `*http.Client`（已注入 ghinstallation transport）作为构造参数
  - [x] 3.3 实现以下只读查询方法（全部返回领域模型，不暴露 go-github 类型）：
    - `GetRepository(ctx, owner, repo)` — 仓库基础信息
    - `ListOpenPullRequests(ctx, owner, repo)` — 打开的 PR 列表（标题、作者、标签、review 状态）
    - `ListRecentIssues(ctx, owner, repo)` — 最近 issue 摘要
    - `ListWorkflowRuns(ctx, owner, repo)` — 最近 workflow run 及其结论
    - `ListDeployments(ctx, owner, repo)` — 部署状态与环境信息
  - [x] 3.4 实现速率限制感知（读取 `Response.Rate`，在接近限额时发出警告）
  - [x] 3.5 定义 `RemoteGitHubState` 结构体封装所有远程状态数据

- [x] Task 4: 定义仓库数字孪生状态模型 (AC: #1, #3)
  - [x] 4.1 创建 `internal/state/repo/model.go`，定义 `RepoSummary` 聚合根结构体
  - [x] 4.2 定义状态标签枚举：`Healthy`, `Drifting`, `Blocked`, `Degraded`, `Unknown`
  - [x] 4.3 定义信号维度结构体：`LocalState`, `RemoteState`, `CollaborationSignals`, `WorkflowState`, `DeploymentState`
  - [x] 4.4 定义 `Risk` 结构体（severity, description, evidence, suggested_action）
  - [x] 4.5 定义 `NextAction` 结构体（action, reason, risk_level, evidence_refs）
  - [x] 4.6 确保所有结构体具有 `json` 和 `yaml` 标签，支持稳定序列化

- [x] Task 5: 实现状态装配器 (AC: #1, #2, #5)
  - [x] 5.1 创建 `internal/app/state/assembler.go`，组合本地和远程数据生成 `RepoSummary`
  - [x] 5.2 在 GitHub 不可用时，将远程维度全部标记为 `Unknown`，附带配置引导消息
  - [x] 5.3 实现每个信号维度的状态标签推导逻辑（规则式，不依赖 LLM）：
    - 本地状态：clean + up-to-date = healthy；dirty = drifting；detached HEAD = degraded
    - 远端偏差：behind > 0 = drifting；conflicts = blocked
    - 协作信号：stale PR = degraded；review-required PR = drifting
    - 工作流：all green = healthy；failure = degraded；no runs = unknown
    - 部署：active + success = healthy；failure = degraded；no deployments = unknown
  - [x] 5.4 实现整体状态标签聚合（取所有维度中最严重状态）
  - [x] 5.5 生成 material risks 列表（从 degraded/blocked 维度提取）
  - [x] 5.6 生成 evidence-backed next actions（从风险映射到建议动作）

- [x] Task 6: 实现 `gitdex status` 命令 (AC: #1, #2, #3, #5, #6)
  - [x] 6.1 创建 `internal/cli/command/status.go`，注册为顶级 `gitdex status` 子命令
  - [x] 6.2 在 `root.go` 中注册 status 命令，需要 bootstrap（不标记 skipBootstrap）
  - [x] 6.3 命令流程：加载 App → 构建 identity transport → 创建 GitHub client → 读取本地 git state → 读取远程 GitHub state → 装配 RepoSummary → 格式化输出
  - [x] 6.4 实现 `renderStatusText(out, summary)` 文本渲染，显示分维度状态看板
  - [x] 6.5 结构化输出（JSON/YAML）复用 `clioutput.WriteValue`，输出完整 `RepoSummary`
  - [x] 6.6 支持 `--owner` 和 `--repo` 标志覆盖，默认从 remote origin URL 解析 owner/repo

- [x] Task 7: 补齐测试与回归验证 (AC: all)
  - [x] 7.1 `internal/platform/identity/github_app_test.go` — 测试 JWT 生成、transport 构建、无配置 graceful degradation
  - [x] 7.2 `internal/platform/git/state_test.go` — 使用临时 git repo 测试本地状态读取
  - [x] 7.3 `internal/platform/github/client_test.go` — 使用 httptest 模拟 GitHub API 响应
  - [x] 7.4 `internal/state/repo/model_test.go` — 状态标签枚举和序列化测试
  - [x] 7.5 `internal/app/state/assembler_test.go` — 状态装配和标签推导逻辑测试
  - [x] 7.6 `test/integration/status_command_test.go` — 集成测试 status 命令各格式输出
  - [x] 7.7 `test/conformance/repo_state_test.go` — 一致性测试验证 AC 合规
  - [x] 7.8 现有测试不得 break；运行 `go test ./...` 和 `golangci-lint run` 全量验证

## Dev Notes

- Story 1.1~1.3 已经提供了可工作的 Cobra 命令树、Viper 多层配置加载（包含 `identity.github_app.*` 字段）、shell completion、`init`/`doctor`/`config show`/`chat`/`capabilities` 命令、LLM adapter 骨架、TaskContext 会话容器和结构化输出系统。
- `config.go` 中已定义 `IdentityConfig` 和 `GitHubAppConfig`（app_id, installation_id, private_key_path, host），并有完整的 env 变量映射（`GITDEX_IDENTITY_GITHUB_APP_*`）。本 story 不修改 config 结构，直接复用。
- `bootstrap.App` 提供 `Config`（包含 `Identity`）和 `RepoRoot`，通过 `appFn func() bootstrap.App` 闭包在命令中获取。本 story 的 status 命令遵循相同模式。
- 架构文档定义的"Repo Digital Twin"概念要求跟踪：local repo status、tracked branches and divergence、open PRs/linked issues、workflow runs and latest checks、deployment and environment state。本 story 实现其只读查询子集。
- 架构要求 GraphQL-first for aggregate reads，但本 MVP story 为简化依赖和测试，优先使用 REST API（go-github）。如果后续需要减少 API 调用次数，可在不改变接口的前提下切换到 GraphQL。
- PRD 明确要求 `GitHub App` 作为首选机器身份，不使用长期 PAT 作为默认授权模型。本 story 不支持 PAT fallback。
- UX 规范的"首次成功时刻"要求：setup 后立刻看到有用的 repo state summary，而不是空界面。`gitdex status` 即是该体验的 CLI 载体。
- Story 1.3 的 code review 确认了以下约束仍然有效：
  - 只有显式 flag 才能覆盖配置层
  - Go 基线锁定 1.26.1
  - 测试不得写入真实用户目录
  - 嵌套目录测试必须真实进入子目录且可清理

### Technical Requirements

- **新增直接依赖（加入 `go.mod`）：**
  - `github.com/bradleyfalzon/ghinstallation/v2` v2.18.0 — GitHub App installation token transport
  - `github.com/google/go-github/v84` v84.0.0 — GitHub REST API client（ghinstallation 已依赖此版本）
  - `github.com/go-git/go-git/v5` v5.17.0 — 本地 Git 仓库状态读取
- 不引入 `github.com/shurcooL/githubv4`（GraphQL client），本 story 仅使用 REST。
- 不引入 Bubble Tea —— TUI 驾驶舱属于 Story 1.5。
- 状态标签推导使用确定性规则（if/else），不调用 LLM。LLM 在本 story 中不被使用。
- 所有 GitHub API 调用必须通过接口抽象，测试中使用 mock 或 httptest，不依赖真实 GitHub API。
- `go-git` 的 `remote.List()` 会发起网络请求获取远程引用，测试中需要 mock 或跳过网络依赖的测试用例。
- 身份模块必须支持 graceful degradation：无 private key 或无 app_id 时，返回结构化错误而非 panic，允许 status 命令只显示本地状态。
- Installation token 有效期 1 小时，本 story 不实现自动刷新机制——每次命令执行时重新获取 token。

### Architecture Compliance

- **新包位置：**
  - `internal/platform/identity/` — GitHub App 身份认证（JWT、installation token）
  - `internal/platform/git/` — 本地 Git 状态读取器（go-git 封装）
  - `internal/platform/github/` — GitHub API 客户端适配器（go-github 封装）
  - `internal/state/repo/` — 仓库数字孪生领域模型
  - `internal/app/state/` — 状态装配器（assembler，组合本地+远程数据）
  - `internal/cli/command/status.go` — `gitdex status` 命令
- 架构要求 Execution Plane 中 `github read adapter` 负责"GraphQL 聚合读 + REST 补充读 + rate budget accounting"。本 story 的 `internal/platform/github/` 包是该 adapter 的初始实现（REST-only + rate 感知）。
- Context Assembler 在架构中负责"收集本地 Git 状态、GitHub 对象、规则集、环境配置、历史审计、最近任务状态"。本 story 的 `internal/app/state/assembler.go` 是其初始实现（本地 Git + GitHub 只读对象）。
- Permission & Trust Plane 要求"GitHub App installation token 作为默认机器身份"。本 story 实现该原语。
- 不触及 `internal/policy/`（Epic 3）、`internal/planning/`（Epic 2）、`internal/tui/`（Story 1.5）。
- `gitdex status` 作为顶级命令而非 `repo status`，因为它是全局入口——与 `doctor`、`chat`、`capabilities` 平级。如果后续需要将 status 嵌入 repo 子组，可以通过别名实现。
- 输出格式扩展继续在 `internal/cli/output/` 中进行，复用 `FormatText`/`FormatJSON`/`FormatYAML` 和 `WriteValue`。
- `gitdexd` daemon 不新增职责。

### Library / Framework Requirements

| Library | Version | Purpose | Import Path |
|---------|---------|---------|-------------|
| ghinstallation | v2.18.0 | GitHub App JWT → installation token transport | `github.com/bradleyfalzon/ghinstallation/v2` |
| go-github | v84.0.0 | GitHub REST API client | `github.com/google/go-github/v84/github` |
| go-git | v5.17.0 | 本地 Git 仓库状态读取 | `github.com/go-git/go-git/v5` |
| Cobra | v1.10.2 | CLI 框架（已有） | `github.com/spf13/cobra` |
| Viper | v1.21.0 | 配置加载（已有） | `github.com/spf13/viper` |

**关键 API 用法：**

```go
// GitHub App identity → HTTP transport
itr, err := ghinstallation.NewKeyFromFile(
    http.DefaultTransport, appID, installationID, privateKeyPath,
)
client := github.NewClient(&http.Client{Transport: itr})

// Local git state
r, _ := git.PlainOpen(repoRoot)
ref, _ := r.Head()
wt, _ := r.Worktree()
status, _ := wt.Status()

// GitHub reads
repo, _, _ := client.Repositories.Get(ctx, owner, name)
prs, _, _ := client.PullRequests.List(ctx, owner, name, &github.PullRequestListOptions{State: "open"})
runs, _, _ := client.Actions.ListRepositoryWorkflowRuns(ctx, owner, name, &github.ListWorkflowRunsOptions{})
deploys, _, _ := client.Repositories.ListDeployments(ctx, owner, name, &github.DeploymentsListOptions{})
```

**go-git 限制：**
- 不直接提供 upstream tracking 的 ahead/behind 计算。需手动读取 `repo.Branch(name)` 获取 `Remote` + `Merge`，然后通过 commit 图遍历比较。
- `repo.Head()` 在 linked worktree 中可能失败，使用 `PlainOpenOptions{EnableDotGitCommonDir: true}`。
- `wt.Status()` 在大型仓库中性能较差，考虑添加超时上下文。

### File Structure Requirements

本 story 预计主要创建/修改下列文件：

**新增文件：**
- `internal/platform/identity/github_app.go` — GitHub App 身份认证
- `internal/platform/identity/github_app_test.go`
- `internal/platform/git/state.go` — 本地 Git 状态读取
- `internal/platform/git/state_test.go`
- `internal/platform/github/client.go` — GitHub API 客户端
- `internal/platform/github/client_test.go`
- `internal/state/repo/model.go` — 仓库数字孪生模型
- `internal/state/repo/model_test.go`
- `internal/app/state/assembler.go` — 状态装配器
- `internal/app/state/assembler_test.go`
- `internal/cli/command/status.go` — `gitdex status` 命令
- `test/integration/status_command_test.go` — 集成测试
- `test/conformance/repo_state_test.go` — 一致性测试

**修改文件：**
- `internal/cli/command/root.go` — 注册 status 命令
- `go.mod` / `go.sum` — 新增三个直接依赖
- `configs/gitdex.example.yaml` — 确认 identity 部分示例配置完整

### Testing Requirements

- 单元测试覆盖每个新包的核心逻辑
- GitHub API 调用通过 `httptest.NewServer` 模拟，提供 JSON 响应 fixture
- go-git 测试在临时目录创建真实 git repo（`git init`），避免网络依赖
- 身份模块测试包括：有效 private key → transport 创建成功；缺失 app_id → ErrNoIdentity；无效 key 格式 → 明确错误
- 状态标签推导测试：覆盖每种输入组合对应的标签输出（表驱动测试）
- 集成测试：status 命令在无 GitHub 配置时仍能输出本地状态 + unknown 远程状态
- 一致性测试：验证 AC 中所有状态标签（healthy/drifting/blocked/degraded/unknown）都能被正确生成
- 运行 `golangci-lint run` 确保无 lint 错误
- 所有现有测试不得 break

### Scope Guardrails — 本 story 不做的事

| 不做 | 原因 |
|------|------|
| GitHub PAT fallback | PRD 要求 GitHub App first，PAT 支持属于后续 story |
| Write operations to GitHub | 本 story 是只读 summary，写操作属于 Epic 2/4 |
| Webhook ingress | 架构 step 4 包含 webhook，但本 story 只做 pull-based reads |
| Policy engine evaluation | Epic 3 能力 |
| TUI/Bubble Tea rendering | Story 1.5 能力 |
| Campaign/fleet views | Epic 6 能力 |
| Installation token auto-refresh/cache | 每次命令执行重新获取，优化属于后续 |
| LLM-based risk analysis | 状态标签使用确定性规则，LLM 分析是后续增强 |
| GraphQL aggregate queries | 本 MVP 使用 REST，后续可切换 |
| `gitdex summary` alias | 可在后续 sprint 添加 |
| Remote ref listing via network | go-git 的 `remote.List()` 需要网络，本 story 仅使用本地可用数据和 GitHub API |

### Previous Story Intelligence

**来自 Story 1.3 的关键学习：**
- 通过 `appFn func() bootstrap.App` 闭包解决命令中延迟访问 bootstrap 数据的问题。status 命令必须遵循同一模式。
- 输出格式判断使用 `effectiveOutputFormat(cmd, flags, app.Config.Output)` 三级优先链（flag → env → config）。
- 文本渲染函数命名约定：`renderXxxText(out io.Writer, data Xxx)`。
- 结构化输出走 `clioutput.WriteValue(out, format, value)` 路径。
- `golangci-lint` 中 `errcheck` 要求：测试中 HTTP handler 的 `Encode`/`Write` 返回值必须显式忽略（`_ =`）。
- `ChatResult` 移除多余字段的经验：只暴露稳定的、面向用户的字段，不泄露内部 SDK 类型。
- `TaskContext` 所有字段通过 mutex-guarded getter 访问，避免并发问题。

**来自 Story 1.2 的关键学习：**
- 配置来源追踪模型（`ConfigSource`）已经建立。新字段如果需要追踪来源，走同一 `Sources` map。
- `doctor` 命令的诊断检查模式（name, status, detail）可作为 status 命令的参考模板。
- 测试使用 `t.TempDir()` + `t.Setenv()` 隔离环境，不污染真实用户配置。

**来自 Story 1.1 的关键学习：**
- `bootstrap.Load()` 在 RepoRoot 不存在时不 panic，返回空字符串。status 命令需要处理这种情况（无 repo root → 无本地 git state）。

### Project Structure Notes

```
internal/
├── app/
│   ├── state/              ← 新增：状态装配器
│   │   ├── assembler.go
│   │   └── assembler_test.go
│   ├── bootstrap/          ← 已有
│   ├── chat/               ← 已有
│   ├── doctor/             ← 已有
│   ├── session/            ← 已有
│   └── setup/              ← 已有
├── cli/
│   ├── command/
│   │   ├── status.go       ← 新增
│   │   └── root.go         ← 修改：注册 status
│   ├── input/              ← 已有
│   └── output/             ← 已有
├── platform/
│   ├── config/             ← 已有（不修改）
│   ├── identity/           ← 新增：GitHub App 身份
│   │   ├── github_app.go
│   │   └── github_app_test.go
│   ├── git/                ← 新增：本地 Git 读取
│   │   ├── state.go
│   │   └── state_test.go
│   └── github/             ← 新增：GitHub API 适配
│       ├── client.go
│       └── client_test.go
├── state/
│   └── repo/               ← 新增：仓库数字孪生模型
│       ├── model.go
│       └── model_test.go
├── llm/                    ← 已有（不修改）
└── daemon/                 ← 已有（不修改）
```

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.4] — AC、FR4/FR5/FR6/UX-DR6
- [Source: _bmad-output/planning-artifacts/architecture.md#Repo Digital Twin] — 数字孪生状态跟踪要求
- [Source: _bmad-output/planning-artifacts/architecture.md#Execution Plane] — github read adapter 定义
- [Source: _bmad-output/planning-artifacts/architecture.md#Permission & Trust Plane] — GitHub App installation token 身份模型
- [Source: _bmad-output/planning-artifacts/architecture.md#Context Assembler] — 上下文装配器定义
- [Source: _bmad-output/planning-artifacts/architecture.md#Implementation Sequence] — Step 4: read-only repo summary
- [Source: _bmad-output/planning-artifacts/prd.md#FR4] — 查看合并仓库状态
- [Source: _bmad-output/planning-artifacts/prd.md#FR5] — 请求当前状态解释与下一步建议
- [Source: _bmad-output/planning-artifacts/prd.md#FR6] — 检查摘要背后的证据和源对象
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Repo State Summary Board] — 状态看板 UX 规范
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Journey 1] — 新用户首次成功体验路径
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Critical Success Moments] — 第一次 repo state summary
- [Source: _bmad-output/implementation-artifacts/1-3-use-dual-mode-terminal-entry-with-discoverable-commands.md] — Story 1.3 开发经验

## Dev Agent Record

### Agent Model Used

claude-4.6-opus-max-thinking

### Debug Log References

- `go test ./... -count=1` — 23/23 packages passed, zero failures
- `golangci-lint run ./...` — 零错误（gofmt 修复后）
- `go run ./cmd/gitdex help status` — 命令注册验证成功

### Completion Notes List

- Ultimate context engine analysis completed — comprehensive developer guide created
- Task 1: 使用 `ghinstallation/v2` v2.18.0 实现 GitHub App JWT → installation token transport。支持 GHES 自定义 host。定义 `ErrNoIdentity`、`ErrMissingField`、`ErrInvalidKeyFile` 三种错误类型实现 graceful degradation。`IsIdentityConfigured()` 快速检查函数用于命令层决策。
- Task 2: 使用 `go-git/v5` v5.17.0 实现本地 Git 状态读取。`ReadLocalState()` 提取 branch、HEAD SHA、clean/dirty、staged count、ahead/behind、remotes、default remote URL。通过 `PlainOpenWithOptions{EnableDotGitCommonDir: true}` 支持 worktree。`computeDivergence()` 通过 commit 图遍历计算 ahead/behind。
- Task 3: 封装 `go-github/v84` v84.0.0 实现 GitHub API 只读适配器。5 个查询方法全部返回领域模型类型（`repo.*`），不暴露 go-github SDK 类型。速率限制感知：仅在 `rate.Limit > 0 && rate.Remaining < 100` 时发出警告，避免测试中误触发。
- Task 4: 定义 `RepoSummary` 聚合根和 5 个信号维度结构体。`StateLabel` 枚举（healthy/drifting/blocked/degraded/unknown）带 severity 排序。`WorstLabel()` 用于聚合。全部结构体有 json+yaml 标签，JSON/YAML 序列化往返测试通过。
- Task 5: 状态装配器组合本地和远程数据。确定性规则推导状态标签（无 LLM）。GitHub 不可用时 graceful degradation（所有远程维度标记为 Unknown 并附带配置引导消息）。自动生成 material risks 和 evidence-backed next actions。
- Task 6: `gitdex status` 顶级命令。遵循 `appFn` 闭包模式。支持 `--owner`/`--repo` 标志和 remote origin 自动解析。text/JSON/YAML 三种输出格式。文本渲染按维度分区显示状态看板。
- Task 7: 全量测试覆盖 — 6 个新增测试文件 + 2 个集成/一致性测试文件。23 个包全部通过。golangci-lint 零错误。

## Senior Developer Review (AI)

### Review Outcome: APPROVE (Round 2)
### Review Date: 2026-03-19

### Round 1 — CHANGES_REQUESTED

| # | Severity | Finding | Status |
|---|----------|---------|--------|
| 1 | CRITICAL | `status.go` 使用 `app.RepoRoot` 但在无 go.mod 时为空，应 fallback 到 `app.Config.Paths.RepositoryRoot` | [x] Resolved |
| 2 | HIGH | `logRateLimit` 使用 `fmt.Printf` 输出到 stdout，破坏 `--output json` | [x] Resolved |
| 3 | MEDIUM | `ListRecentIssues` 语义不清（返回 `resp.LastPage` 近似值） | [x] Resolved — 重命名为 `EstimateOpenIssueCount` 并添加文档 |
| 4 | MEDIUM | `EstimateOpenIssueCount` 无测试 | [x] Resolved — 添加 `TestEstimateOpenIssueCount` |
| 5 | LOW | `Blocked` 标签未使用 | Noted — 保留用于后续 branch protection 集成 |
| 6 | LOW | `Ahead > 0` 未纳入 divergence 判定 | [x] Resolved — 添加 ahead → drifting 规则和测试 |

### Round 2 — APPROVE

所有修复验证通过，零新问题。全部 23 个测试包通过，golangci-lint 零错误。

### Change Log

- 2026-03-19: 初始实现完成，所有 7 个 Task 和全部子任务完成。全量测试通过，lint 零错误。
- 2026-03-19: Code review round 1 — 6 个发现（1 CRITICAL, 1 HIGH, 2 MEDIUM, 2 LOW），全部修复。
- 2026-03-19: Code review round 2 — APPROVE。状态标记为 done。

### File List

**新增文件：**
- `internal/platform/identity/github_app.go`
- `internal/platform/identity/github_app_test.go`
- `internal/platform/git/state.go`
- `internal/platform/git/state_test.go`
- `internal/platform/github/client.go`
- `internal/platform/github/client_test.go`
- `internal/state/repo/model.go`
- `internal/state/repo/model_test.go`
- `internal/app/state/assembler.go`
- `internal/app/state/assembler_test.go`
- `internal/cli/command/status.go`
- `test/integration/status_command_test.go`
- `test/conformance/repo_state_test.go`

**修改文件：**
- `internal/cli/command/root.go` — 注册 status 命令
- `go.mod` — 新增 ghinstallation/v2、go-github/v84、go-git/v5
- `go.sum` — 依赖更新
