# Story 1.1: Set Up Initial Project from Starter Template

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

作为平台工程师，
我希望从已批准的 starter foundation 初始化 Gitdex，
以便后续所有 story 都建立在一致的 runtime、command tree 和 repository structure 上。

## Acceptance Criteria

1. **Given** 一个空的 Gitdex 代码库  
   **When** 执行 starter setup  
   **Then** 仓库中包含约定好的 Go workspace baseline，且具备 `gitdex` 与 `gitdexd` 双入口、核心目录、配置脚手架、schema 目录和 migration 占位文件。
2. **Given** 已生成 starter baseline  
   **When** 在受支持的开发机上执行基础验证  
   **Then** baseline 的 build、test 与 local run 命令可以成功执行。
3. **Given** starter baseline 已完成  
   **When** 后续 story 需要扩展 CLI 与配置体系  
   **Then** shell completion 与配置加载钩子已经接好，可直接复用。

## Tasks / Subtasks

- [x] 初始化 Go workspace 与根目录工具链骨架 (AC: 1)
  - [x] 在仓库根目录创建 `go.mod`，采用架构要求的 Go 版本线，并为正式模块路径保留可替换空间。
  - [x] 补齐根目录基础文件：`README.md`、`.gitignore`、`.env.example`、`Taskfile.yml`、`Makefile`、`.golangci.yml`、`.goreleaser.yml`。
  - [x] 新增 `configs/gitdex.example.yaml`，并建立 `configs/policies/default/` 默认策略文件骨架。

- [x] 建立双二进制入口与最小可编译命令树 (AC: 1, 2, 3)
  - [x] 创建 `cmd/gitdex/main.go` 与 `cmd/gitdexd/main.go`。
  - [x] 用 `Cobra` 建立最小根命令及占位子命令分组，至少覆盖 `completion`、`version`、`daemon` 入口语义。
  - [x] 为 `gitdexd` 提供可成功启动的 `run` 路径，允许先用轻量 stub 行为占位，但必须能本地运行并明确输出当前是 starter baseline。

- [x] 接入配置加载与 shell completion 钩子 (AC: 2, 3)
  - [x] 通过 `Viper` 接好 `flag + env + config file` 的配置加载入口。
  - [x] 在 `internal/cli/completion` 中封装 completion 输出逻辑，支持至少 `bash`、`zsh`、`fish`、`powershell`。
  - [x] 在 `internal/platform/config` 中建立可复用的配置装载与示例配置路径约定。

- [x] 创建架构规定的目录骨架与占位文件 (AC: 1)
  - [x] 建立最小必需目录：`internal/`、`pkg/contracts/`、`schema/`、`migrations/`、`scripts/`、`test/`。
  - [x] 创建 `schema/json/` 与 `schema/openapi/control_plane.yaml` 占位文件。
  - [x] 创建 migration 占位文件：`000001_init.sql`、`000002_task_events.sql`、`000003_repo_projections.sql`、`000004_audit_records.sql`。
  - [x] 为后续 story 预留空目录时，使用 `.gitkeep` 或最小 README 占位，避免目录在版本控制中丢失。

- [x] 让 starter baseline 可验证、可运行、可交接 (AC: 2, 3)
  - [x] 确保 `go test ./...` 成功。
  - [x] 确保 `go run ./cmd/gitdex --help` 成功。
  - [x] 确保 `go run ./cmd/gitdex completion powershell` 成功。
  - [x] 确保 `go run ./cmd/gitdexd run` 成功，并有明确的启动或占位日志。
  - [x] 在 README 或开发说明中记录本地启动与验证命令，方便后续 story 直接复用。

- [x] 控制范围，避免提前实现后续能力 (AC: 1, 2, 3)
  - [x] 本 story 只交付 starter skeleton，不实现真实 GitHub App auth、PostgreSQL 持久层、repo scan、policy engine、structured plan compiler、worktree execution 或 rich TUI。
  - [x] 仅保留这些能力的目录、契约和占位点，避免开发者误把架构终态一次性塞进 starter story。

## Dev Notes

- 当前仓库根目录还没有正式的 Go 应用代码；现有内容以 `.agents/`、`_bmad/`、`_bmad-output/`、`docs/`、`design-artifacts/`、`reference_project/` 为主。本 story 需要在同一根目录下建立产品代码骨架，但不得破坏这些已有文档与参考材料。
- `reference_project/` 里的项目只可作为工程形态参考，不是 Gitdex 的直接 starter template 来源。已批准的 starter foundation 来自架构文档中选定的 `cobra-cli` + Go workspace 方案，而不是对任何参考仓库做二次包装。
- 该 story 是整个实现序列的地基。优先目标是“结构正确、命令可编译、目录稳定、后续 story 有挂点”，而不是“能力尽量多”。

### Technical Requirements

- 强制采用 `Go 1.26.1` 作为 starter 语言底座。
- CLI foundation 使用 `Cobra v1.10.2`；starter 应基于 `cobra-cli init --viper` 的能力模型来组织命令树和配置接线。
- `Bubble Tea v2` 是架构选定的 TUI 方向，但本 story 不应交付 rich TUI 实现；最多只预留 `internal/tui/` 相关目录或接口位。
- `PostgreSQL` 是系统记录源，但本 story 仅创建 `migrations/` 与相关占位结构，不要求真实数据库接入。
- 配置层必须为后续 `global + repo + session + env` 四层叠加留出扩展位，当前先把入口和示例配置路径接好。

### Architecture Compliance

- `cmd/` 只放二进制入口；核心实现进入 `internal/`。
- `pkg/contracts/` 只放外部共享的 schema 或 struct，不放业务逻辑。
- `schema/` 负责 JSON Schema 与 OpenAPI 占位物。
- `test/` 负责跨模块集成、合约、端到端和 conformance 测试挂点。
- 本 story 不得绕开架构里已经明确的双二进制模型；必须同时建立 `gitdex` 与 `gitdexd`。
- 未来所有 Git 写操作都必须在 `git worktree` 内运行，并受 `single-writer-per-repo-ref` 约束；本 story 只需要预留相关目录，不要伪造执行能力。

### Library / Framework Requirements

- 依赖选型以架构文档为准，禁止自行把 starter 改成其他 CLI 框架或多语言脚手架。
- `Cobra` 负责命令树、help、completion 和命令发现；`Viper` 负责配置加载入口。
- 如果为了 starter compile 需要少量辅助依赖，应保持最小集合，不要提前引入 GitHub SDK、数据库 ORM、消息队列或大型 TUI 生态。
- `gitdexd` 的 starter 行为应尽量轻量，避免为了“能跑起来”引入尚未需要的后台框架。

### File Structure Requirements

- 本 story 至少应建立下列稳定路径：
  - `cmd/gitdex/main.go`
  - `cmd/gitdexd/main.go`
  - `configs/gitdex.example.yaml`
  - `configs/policies/default/global.yaml`
  - `configs/policies/default/repo_class_public.yaml`
  - `configs/policies/default/repo_class_sensitive.yaml`
  - `configs/policies/default/repo_class_release_critical.yaml`
  - `internal/app/bootstrap/`
  - `internal/app/version/`
  - `internal/cli/command/`
  - `internal/cli/completion/`
  - `internal/cli/output/`
  - `internal/platform/config/`
  - `internal/platform/ids/`
  - `internal/platform/logging/`
  - `pkg/contracts/plan/`
  - `pkg/contracts/task/`
  - `pkg/contracts/audit/`
  - `pkg/contracts/campaign/`
  - `pkg/contracts/handoff/`
  - `pkg/contracts/api/`
  - `schema/json/plan.schema.json`
  - `schema/json/task.schema.json`
  - `schema/json/campaign.schema.json`
  - `schema/json/audit_event.schema.json`
  - `schema/json/handoff_pack.schema.json`
  - `schema/json/api_error.schema.json`
  - `schema/openapi/control_plane.yaml`
  - `migrations/000001_init.sql`
  - `migrations/000002_task_events.sql`
  - `migrations/000003_repo_projections.sql`
  - `migrations/000004_audit_records.sql`
  - `scripts/dev/`
  - `scripts/ci/`
  - `scripts/fixtures/`
  - `test/integration/`
  - `test/e2e/`
  - `test/contracts/`
  - `test/conformance/`
  - `test/fixtures/repos/`
  - `test/fixtures/policies/`
  - `test/fixtures/webhooks/`
  - `test/fixtures/campaigns/`

### Testing Requirements

- 最低验收命令集：
  - `go test ./...`
  - `go run ./cmd/gitdex --help`
  - `go run ./cmd/gitdex completion powershell`
  - `go run ./cmd/gitdexd run`
- 如果 `gitdexd run` 需要阻塞运行，应提供可预测、可退出的本地开发行为，例如输出 starter banner、启动最小 health stub，或在显式 flag 下运行一次性 smoke 模式。
- 从本 story 开始就要为后续 `test/conformance` 留出结构，尤其是跨平台终端行为与 text-only 输出的测试挂点。

### Latest Technical Validation

- 2026-03-18 核验结果显示，Go 官方 release history 已列出 `go1.26.0` 于 2026-02-10 发布，`go1.26.1` 于 2026-03-05 发布；架构中锁定 `Go 1.26.1` 是有效且新鲜的。
- `spf13/cobra` 官方发布页将 `v1.10.2` 标记为 Latest；因此 starter 直接采用这一版本线与架构一致。
- `Bubble Tea v2` 官方发布页显示 `v2.0.0-beta.5` 仍是 pre-release；因此本 story 只保留 TUI 方向与目录挂点，不应把 rich TUI 当作 starter 必交项。
- PostgreSQL 官方 versioning 页面显示 2026-02-26 发布了 `PostgreSQL 18.3` 和 `17.9`；因此架构提出的“以 17.x 为参考生产基线并兼容 18.x”仍然成立，starter 里的 migration 占位应避免依赖仅 18.x 才有的特性。

### Project Structure Notes

- 现阶段最重要的是把“物理边界”搭对，而不是把所有目录填满。空目录可以用最小占位文件保留。
- `cmd/`、`internal/`、`pkg/contracts/`、`schema/`、`migrations/`、`test/` 的职责边界必须从第一天开始保持清晰，否则后续多 agent 协作会快速发散。
- 由于当前仓库还承载 BMAD 规划与参考材料，新增产品代码时要保持根目录整洁，不要把实现文件散落到 `_bmad-output/`、`docs/` 或 `reference_project/` 中。

### Non-Goals / Scope Guardrails

- 不实现真实 GitHub App 安装、授权或 webhook 处理。
- 不实现真实 PostgreSQL 连接、仓储层或审计流水。
- 不实现 repo summary、structured plan、policy verdict、takeover/handoff UI。
- 不实现真实 `git worktree` 生命周期管理。
- 不实现 rich TUI 屏幕，只保留未来扩展位置。
- 不把参考项目代码直接复制为 Gitdex 正式实现。

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Epic-1-Terminal-Onboarding-Identity-and-Repository-Visibility]
- [Source: _bmad-output/planning-artifacts/epics.md#Story-11-Set-Up-Initial-Project-from-Starter-Template-Architecture-Starter-Requirement]
- [Source: _bmad-output/planning-artifacts/prd.md#Configuration-Onboarding--Operator-Enablement]
- [Source: _bmad-output/planning-artifacts/architecture.md#Selected-Starter-Cobra-Based-Go-Workspace-Foundation]
- [Source: _bmad-output/planning-artifacts/architecture.md#Core-Architectural-Decisions]
- [Source: _bmad-output/planning-artifacts/architecture.md#Complete-Project-Directory-Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#Structure-Patterns]
- [Source: _bmad-output/planning-artifacts/architecture.md#Development-Workflow-Integration]
- [Source: _bmad-output/planning-artifacts/architecture.md#Implementation-Handoff]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Journey-1-新用户从首次-setup-到第一次值得保留的成功体验]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Component-Implementation-Strategy]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Testing-Strategy]
- [External: https://go.dev/doc/devel/release]
- [External: https://github.com/spf13/cobra/releases/tag/v1.10.2]
- [External: https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0-beta.5]
- [External: https://www.postgresql.org/support/versioning/]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Sprint auto-discovery selected `1-1-set-up-initial-project-from-starter-template` as the first backlog story.
- No previous story file exists for Epic 1, so there are no earlier implementation learnings to inherit.
- No product code exists at repository root yet; this story is the first formal implementation handoff.
- Updated sprint tracking from `ready-for-dev` to `in-progress` before implementation.
- Added starter conformance and package tests first, confirmed they failed against the empty baseline, then implemented the skeleton to make them pass.
- Installed and ran `golangci-lint` locally after the Go baseline was in place so the configured lint gate could be exercised instead of skipped.
- Resumed Story 1.1 from `review` after code review surfaced four issues spanning config precedence, version injection, Go baseline pinning, and nested-directory validation coverage.
- Added regression tests around config layering, explicit flag override detection, injected version reporting, and real nested-directory bootstrap execution before re-running the full validation suite.
- Resumed Story 1.1 again after follow-up review found the nested-directory acceptance test was leaking a `testdata` tree into the working directory on repeated runs.
- Reworked the nested-directory test to use a unique repo-local temporary directory, restore the original working directory before cleanup, and assert cleanup success while removing the leaked `internal/cli/command/testdata` residue from the workspace.

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Implemented a Go 1.26.1 starter workspace with `gitdex` and `gitdexd` entrypoints, root tooling files, starter config, and policy placeholders.
- Implemented Cobra-based command trees with `version`, `completion`, and daemon run paths, plus a Viper-backed config loader and daemon stub service.
- Added schema placeholders, migration placeholders, reserved directory markers, unit tests, conformance tests, and integration tests for starter command execution.
- Validation passed with `go test ./...`, `go run ./cmd/gitdex --help`, `go run ./cmd/gitdex completion powershell`, `go run ./cmd/gitdexd run`, and `golangci-lint run`.
- Story completion moved the story artifact to `review` and the sprint tracking entry to `review`.
- Fixed CLI/config precedence so only explicitly passed `--output`, `--log-level`, and `--profile` flags override Viper config or environment values.
- Updated the starter baseline to pin `go 1.26.1`, made `internal/app/version.Version` build-injectable, and wired GoReleaser to stamp release binaries with the real version.
- Corrected nested-directory bootstrap coverage by executing the daemon command from a real subdirectory and added conformance coverage for version ldflags injection.
- Re-validated the story with `go test ./...`, `golangci-lint run`, `go run ./cmd/gitdex --help`, `go run ./cmd/gitdex completion powershell`, `go run ./cmd/gitdexd run`, and `go run -ldflags "-X github.com/your-org/gitdex/internal/app/version.Version=1.2.3-test" ./cmd/gitdex version`.
- Hardened `TestDaemonRunStillBootstrapsFromNestedDirectory` so it no longer uses `testdata`, now cleans up a unique `.gitdex-nested-*` temp root only after restoring the working directory, and fails visibly if cleanup does not complete.
- Re-ran the nested-directory test multiple times, removed the previously leaked `internal/cli/command/testdata` directory, confirmed no residue remained under `internal/cli/command`, and then re-ran the full validation suite.

### File List

- .env.example
- .gitignore
- .golangci.yml
- .goreleaser.yml
- Makefile
- README.md
- Taskfile.yml
- cmd/gitdex/main.go
- cmd/gitdexd/main.go
- configs/gitdex.example.yaml
- configs/policies/default/global.yaml
- configs/policies/default/repo_class_public.yaml
- configs/policies/default/repo_class_release_critical.yaml
- configs/policies/default/repo_class_sensitive.yaml
- go.mod
- go.sum
- internal/app/bootstrap/bootstrap.go
- internal/app/version/version.go
- internal/cli/command/root.go
- internal/cli/command/root_internal_test.go
- internal/cli/command/root_test.go
- internal/cli/completion/completion.go
- internal/cli/output/format.go
- internal/daemon/service/run.go
- internal/daemon/service/run_test.go
- internal/platform/config/config.go
- internal/platform/config/config_test.go
- internal/platform/ids/.gitkeep
- internal/platform/logging/.gitkeep
- migrations/000001_init.sql
- migrations/000002_task_events.sql
- migrations/000003_repo_projections.sql
- migrations/000004_audit_records.sql
- pkg/contracts/api/.gitkeep
- pkg/contracts/audit/.gitkeep
- pkg/contracts/campaign/.gitkeep
- pkg/contracts/handoff/.gitkeep
- pkg/contracts/plan/.gitkeep
- pkg/contracts/task/.gitkeep
- schema/json/api_error.schema.json
- schema/json/audit_event.schema.json
- schema/json/campaign.schema.json
- schema/json/handoff_pack.schema.json
- schema/json/plan.schema.json
- schema/json/task.schema.json
- schema/openapi/control_plane.yaml
- scripts/ci/.gitkeep
- scripts/dev/.gitkeep
- scripts/fixtures/.gitkeep
- test/conformance/starter_skeleton_test.go
- test/contracts/.gitkeep
- test/e2e/.gitkeep
- test/fixtures/campaigns/.gitkeep
- test/fixtures/policies/.gitkeep
- test/fixtures/repos/.gitkeep
- test/fixtures/webhooks/.gitkeep
- test/integration/.gitkeep
- test/integration/starter_commands_test.go
- _bmad-output/implementation-artifacts/1-1-set-up-initial-project-from-starter-template.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

### Change Log

- 2026-03-18: Implemented Story 1.1 starter baseline, added command/config/test scaffolding, and validated the workspace through test, smoke, completion, daemon, and lint gates.
- 2026-03-18: Addressed code review findings for Story 1.1 by fixing config precedence, enabling release version injection, pinning the Go baseline to 1.26.1, and strengthening nested-directory/conformance coverage.
- 2026-03-18: Fixed nested-directory test cleanup so repeated validation runs no longer leak `testdata` or temporary directories into the repository workspace.
