# Review Prompt: Blind Hunter

Use `$bmad-review-adversarial-general` in a separate session and follow these rules exactly:

- Output language must be Chinese.
- You have no project context, no spec, no repository access, and no conversation history.
- Review only the diff below.
- Focus on real bugs, regressions, design flaws, maintainability traps, and misleading tests or validation.
- Do not include praise or summary text.
- If there are no actionable issues, output exactly: `未发现可操作问题`.
- Output format: a Markdown list. Each finding must include: title, severity, evidence, impact, and one-line fix direction.

## Review Scope
- Story: `1-1-set-up-initial-project-from-starter-template`
- Diff source: `synthetic diff from current Story 1.1 implementation scope (repository has no .git)`
- Files: `66`
- Added lines: `1834`
- Removed lines: `0`
- Review mode: `full`
- Spec file: `_bmad-output/implementation-artifacts/1-1-set-up-initial-project-from-starter-template.md`
- Additional context docs: `none`
- Reviewer layer: `blind-hunter`
- Context access: `diff only`

## Diff
```diff
--- /dev/null
+++ .env.example
@@ -0,0 +1,5 @@
+GITDEX_CONFIG=
+GITDEX_OUTPUT=text
+GITDEX_LOG_LEVEL=info
+GITDEX_PROFILE=local
+GITDEX_DAEMON_HEALTH_ADDRESS=127.0.0.1:7777
--- /dev/null
+++ .gitignore
@@ -0,0 +1,8 @@
+.DS_Store
+.env
+.idea/
+.vscode/
+bin/
+dist/
+coverage.out
+*.test
--- /dev/null
+++ .golangci.yml
@@ -0,0 +1,8 @@
+run:
+  timeout: 5m
+
+linters:
+  enable:
+    - errcheck
+    - gofmt
+    - govet
--- /dev/null
+++ .goreleaser.yml
@@ -0,0 +1,27 @@
+project_name: gitdex
+
+builds:
+  - id: gitdex
+    main: ./cmd/gitdex
+    binary: gitdex
+    ldflags:
+      - -s -w -X github.com/your-org/gitdex/internal/app/version.Version={{.Version}}
+    goos:
+      - windows
+      - linux
+      - darwin
+    goarch:
+      - amd64
+      - arm64
+  - id: gitdexd
+    main: ./cmd/gitdexd
+    binary: gitdexd
+    ldflags:
+      - -s -w -X github.com/your-org/gitdex/internal/app/version.Version={{.Version}}
+    goos:
+      - windows
+      - linux
+      - darwin
+    goarch:
+      - amd64
+      - arm64
--- /dev/null
+++ Makefile
@@ -0,0 +1,18 @@
+GO ?= go
+
+.PHONY: test run daemon completion-powershell fmt
+
+test:
+	$(GO) test ./...
+
+run:
+	$(GO) run ./cmd/gitdex --help
+
+daemon:
+	$(GO) run ./cmd/gitdexd run
+
+completion-powershell:
+	$(GO) run ./cmd/gitdex completion powershell
+
+fmt:
+	gofmt -w ./cmd ./internal ./test
--- /dev/null
+++ README.md
@@ -0,0 +1,33 @@
+# Gitdex
+
+Gitdex is a terminal-first, daemon-backed governed control plane for repository operations.
+
+This repository currently contains the Story 1.1 starter baseline:
+
+- Go workspace foundation
+- `gitdex` and `gitdexd` entrypoints
+- Cobra command tree with config and shell completion hooks
+- Placeholder schema, migration, policy, and test structure for later stories
+
+## Starter Validation
+
+```powershell
+go test ./...
+go run ./cmd/gitdex --help
+go run ./cmd/gitdex completion powershell
+go run ./cmd/gitdexd run
+```
+
+## Starter Commands
+
+```powershell
+go run ./cmd/gitdex version
+go run ./cmd/gitdex daemon run
+go run ./cmd/gitdexd run
+```
+
+## Configuration
+
+- Example config: `configs/gitdex.example.yaml`
+- Environment prefix: `GITDEX_`
+- Supported starter flags: `--config`, `--output`, `--log-level`, `--profile`
--- /dev/null
+++ Taskfile.yml
@@ -0,0 +1,22 @@
+version: "3"
+
+tasks:
+  test:
+    cmds:
+      - go test ./...
+
+  run:
+    cmds:
+      - go run ./cmd/gitdex --help
+
+  daemon:
+    cmds:
+      - go run ./cmd/gitdexd run
+
+  completion:powershell:
+    cmds:
+      - go run ./cmd/gitdex completion powershell
+
+  fmt:
+    cmds:
+      - gofmt -w ./cmd ./internal ./test
--- /dev/null
+++ _bmad-output/implementation-artifacts/1-1-set-up-initial-project-from-starter-template.md
@@ -0,0 +1,283 @@
+# Story 1.1: Set Up Initial Project from Starter Template
+
+Status: review
+
+<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
+
+## Story
+
+作为平台工程师，
+我希望从已批准的 starter foundation 初始化 Gitdex，
+以便后续所有 story 都建立在一致的 runtime、command tree 和 repository structure 上。
+
+## Acceptance Criteria
+
+1. **Given** 一个空的 Gitdex 代码库  
+   **When** 执行 starter setup  
+   **Then** 仓库中包含约定好的 Go workspace baseline，且具备 `gitdex` 与 `gitdexd` 双入口、核心目录、配置脚手架、schema 目录和 migration 占位文件。
+2. **Given** 已生成 starter baseline  
+   **When** 在受支持的开发机上执行基础验证  
+   **Then** baseline 的 build、test 与 local run 命令可以成功执行。
+3. **Given** starter baseline 已完成  
+   **When** 后续 story 需要扩展 CLI 与配置体系  
+   **Then** shell completion 与配置加载钩子已经接好，可直接复用。
+
+## Tasks / Subtasks
+
+- [x] 初始化 Go workspace 与根目录工具链骨架 (AC: 1)
+  - [x] 在仓库根目录创建 `go.mod`，采用架构要求的 Go 版本线，并为正式模块路径保留可替换空间。
+  - [x] 补齐根目录基础文件：`README.md`、`.gitignore`、`.env.example`、`Taskfile.yml`、`Makefile`、`.golangci.yml`、`.goreleaser.yml`。
+  - [x] 新增 `configs/gitdex.example.yaml`，并建立 `configs/policies/default/` 默认策略文件骨架。
+
+- [x] 建立双二进制入口与最小可编译命令树 (AC: 1, 2, 3)
+  - [x] 创建 `cmd/gitdex/main.go` 与 `cmd/gitdexd/main.go`。
+  - [x] 用 `Cobra` 建立最小根命令及占位子命令分组，至少覆盖 `completion`、`version`、`daemon` 入口语义。
+  - [x] 为 `gitdexd` 提供可成功启动的 `run` 路径，允许先用轻量 stub 行为占位，但必须能本地运行并明确输出当前是 starter baseline。
+
+- [x] 接入配置加载与 shell completion 钩子 (AC: 2, 3)
+  - [x] 通过 `Viper` 接好 `flag + env + config file` 的配置加载入口。
+  - [x] 在 `internal/cli/completion` 中封装 completion 输出逻辑，支持至少 `bash`、`zsh`、`fish`、`powershell`。
+  - [x] 在 `internal/platform/config` 中建立可复用的配置装载与示例配置路径约定。
+
+- [x] 创建架构规定的目录骨架与占位文件 (AC: 1)
+  - [x] 建立最小必需目录：`internal/`、`pkg/contracts/`、`schema/`、`migrations/`、`scripts/`、`test/`。
+  - [x] 创建 `schema/json/` 与 `schema/openapi/control_plane.yaml` 占位文件。
+  - [x] 创建 migration 占位文件：`000001_init.sql`、`000002_task_events.sql`、`000003_repo_projections.sql`、`000004_audit_records.sql`。
+  - [x] 为后续 story 预留空目录时，使用 `.gitkeep` 或最小 README 占位，避免目录在版本控制中丢失。
+
+- [x] 让 starter baseline 可验证、可运行、可交接 (AC: 2, 3)
+  - [x] 确保 `go test ./...` 成功。
+  - [x] 确保 `go run ./cmd/gitdex --help` 成功。
+  - [x] 确保 `go run ./cmd/gitdex completion powershell` 成功。
+  - [x] 确保 `go run ./cmd/gitdexd run` 成功，并有明确的启动或占位日志。
+  - [x] 在 README 或开发说明中记录本地启动与验证命令，方便后续 story 直接复用。
+
+- [x] 控制范围，避免提前实现后续能力 (AC: 1, 2, 3)
+  - [x] 本 story 只交付 starter skeleton，不实现真实 GitHub App auth、PostgreSQL 持久层、repo scan、policy engine、structured plan compiler、worktree execution 或 rich TUI。
+  - [x] 仅保留这些能力的目录、契约和占位点，避免开发者误把架构终态一次性塞进 starter story。
+
+## Dev Notes
+
+- 当前仓库根目录还没有正式的 Go 应用代码；现有内容以 `.agents/`、`_bmad/`、`_bmad-output/`、`docs/`、`design-artifacts/`、`reference_project/` 为主。本 story 需要在同一根目录下建立产品代码骨架，但不得破坏这些已有文档与参考材料。
+- `reference_project/` 里的项目只可作为工程形态参考，不是 Gitdex 的直接 starter template 来源。已批准的 starter foundation 来自架构文档中选定的 `cobra-cli` + Go workspace 方案，而不是对任何参考仓库做二次包装。
+- 该 story 是整个实现序列的地基。优先目标是“结构正确、命令可编译、目录稳定、后续 story 有挂点”，而不是“能力尽量多”。
+
+### Technical Requirements
+
+- 强制采用 `Go 1.26.1` 作为 starter 语言底座。
+- CLI foundation 使用 `Cobra v1.10.2`；starter 应基于 `cobra-cli init --viper` 的能力模型来组织命令树和配置接线。
+- `Bubble Tea v2` 是架构选定的 TUI 方向，但本 story 不应交付 rich TUI 实现；最多只预留 `internal/tui/` 相关目录或接口位。
+- `PostgreSQL` 是系统记录源，但本 story 仅创建 `migrations/` 与相关占位结构，不要求真实数据库接入。
+- 配置层必须为后续 `global + repo + session + env` 四层叠加留出扩展位，当前先把入口和示例配置路径接好。
+
+### Architecture Compliance
+
+- `cmd/` 只放二进制入口；核心实现进入 `internal/`。
+- `pkg/contracts/` 只放外部共享的 schema 或 struct，不放业务逻辑。
+- `schema/` 负责 JSON Schema 与 OpenAPI 占位物。
+- `test/` 负责跨模块集成、合约、端到端和 conformance 测试挂点。
+- 本 story 不得绕开架构里已经明确的双二进制模型；必须同时建立 `gitdex` 与 `gitdexd`。
+- 未来所有 Git 写操作都必须在 `git worktree` 内运行，并受 `single-writer-per-repo-ref` 约束；本 story 只需要预留相关目录，不要伪造执行能力。
+
+### Library / Framework Requirements
+
+- 依赖选型以架构文档为准，禁止自行把 starter 改成其他 CLI 框架或多语言脚手架。
+- `Cobra` 负责命令树、help、completion 和命令发现；`Viper` 负责配置加载入口。
+- 如果为了 starter compile 需要少量辅助依赖，应保持最小集合，不要提前引入 GitHub SDK、数据库 ORM、消息队列或大型 TUI 生态。
+- `gitdexd` 的 starter 行为应尽量轻量，避免为了“能跑起来”引入尚未需要的后台框架。
+
+### File Structure Requirements
+
+- 本 story 至少应建立下列稳定路径：
+  - `cmd/gitdex/main.go`
+  - `cmd/gitdexd/main.go`
+  - `configs/gitdex.example.yaml`
+  - `configs/policies/default/global.yaml`
+  - `configs/policies/default/repo_class_public.yaml`
+  - `configs/policies/default/repo_class_sensitive.yaml`
+  - `configs/policies/default/repo_class_release_critical.yaml`
+  - `internal/app/bootstrap/`
+  - `internal/app/version/`
+  - `internal/cli/command/`
+  - `internal/cli/completion/`
+  - `internal/cli/output/`
+  - `internal/platform/config/`
+  - `internal/platform/ids/`
+  - `internal/platform/logging/`
+  - `pkg/contracts/plan/`
+  - `pkg/contracts/task/`
+  - `pkg/contracts/audit/`
+  - `pkg/contracts/campaign/`
+  - `pkg/contracts/handoff/`
+  - `pkg/contracts/api/`
+  - `schema/json/plan.schema.json`
+  - `schema/json/task.schema.json`
+  - `schema/json/campaign.schema.json`
+  - `schema/json/audit_event.schema.json`
+  - `schema/json/handoff_pack.schema.json`
+  - `schema/json/api_error.schema.json`
+  - `schema/openapi/control_plane.yaml`
+  - `migrations/000001_init.sql`
+  - `migrations/000002_task_events.sql`
+  - `migrations/000003_repo_projections.sql`
+  - `migrations/000004_audit_records.sql`
+  - `scripts/dev/`
+  - `scripts/ci/`
+  - `scripts/fixtures/`
+  - `test/integration/`
+  - `test/e2e/`
+  - `test/contracts/`
+  - `test/conformance/`
+  - `test/fixtures/repos/`
+  - `test/fixtures/policies/`
+  - `test/fixtures/webhooks/`
+  - `test/fixtures/campaigns/`
+
+### Testing Requirements
+
+- 最低验收命令集：
+  - `go test ./...`
+  - `go run ./cmd/gitdex --help`
+  - `go run ./cmd/gitdex completion powershell`
+  - `go run ./cmd/gitdexd run`
+- 如果 `gitdexd run` 需要阻塞运行，应提供可预测、可退出的本地开发行为，例如输出 starter banner、启动最小 health stub，或在显式 flag 下运行一次性 smoke 模式。
+- 从本 story 开始就要为后续 `test/conformance` 留出结构，尤其是跨平台终端行为与 text-only 输出的测试挂点。
+
+### Latest Technical Validation
+
+- 2026-03-18 核验结果显示，Go 官方 release history 已列出 `go1.26.0` 于 2026-02-10 发布，`go1.26.1` 于 2026-03-05 发布；架构中锁定 `Go 1.26.1` 是有效且新鲜的。
+- `spf13/cobra` 官方发布页将 `v1.10.2` 标记为 Latest；因此 starter 直接采用这一版本线与架构一致。
+- `Bubble Tea v2` 官方发布页显示 `v2.0.0-beta.5` 仍是 pre-release；因此本 story 只保留 TUI 方向与目录挂点，不应把 rich TUI 当作 starter 必交项。
+- PostgreSQL 官方 versioning 页面显示 2026-02-26 发布了 `PostgreSQL 18.3` 和 `17.9`；因此架构提出的“以 17.x 为参考生产基线并兼容 18.x”仍然成立，starter 里的 migration 占位应避免依赖仅 18.x 才有的特性。
+
+### Project Structure Notes
+
+- 现阶段最重要的是把“物理边界”搭对，而不是把所有目录填满。空目录可以用最小占位文件保留。
+- `cmd/`、`internal/`、`pkg/contracts/`、`schema/`、`migrations/`、`test/` 的职责边界必须从第一天开始保持清晰，否则后续多 agent 协作会快速发散。
+- 由于当前仓库还承载 BMAD 规划与参考材料，新增产品代码时要保持根目录整洁，不要把实现文件散落到 `_bmad-output/`、`docs/` 或 `reference_project/` 中。
+
+### Non-Goals / Scope Guardrails
+
+- 不实现真实 GitHub App 安装、授权或 webhook 处理。
+- 不实现真实 PostgreSQL 连接、仓储层或审计流水。
+- 不实现 repo summary、structured plan、policy verdict、takeover/handoff UI。
+- 不实现真实 `git worktree` 生命周期管理。
+- 不实现 rich TUI 屏幕，只保留未来扩展位置。
+- 不把参考项目代码直接复制为 Gitdex 正式实现。
+
+### References
+
+- [Source: _bmad-output/planning-artifacts/epics.md#Epic-1-Terminal-Onboarding-Identity-and-Repository-Visibility]
+- [Source: _bmad-output/planning-artifacts/epics.md#Story-11-Set-Up-Initial-Project-from-Starter-Template-Architecture-Starter-Requirement]
+- [Source: _bmad-output/planning-artifacts/prd.md#Configuration-Onboarding--Operator-Enablement]
+- [Source: _bmad-output/planning-artifacts/architecture.md#Selected-Starter-Cobra-Based-Go-Workspace-Foundation]
+- [Source: _bmad-output/planning-artifacts/architecture.md#Core-Architectural-Decisions]
+- [Source: _bmad-output/planning-artifacts/architecture.md#Complete-Project-Directory-Structure]
+- [Source: _bmad-output/planning-artifacts/architecture.md#Structure-Patterns]
+- [Source: _bmad-output/planning-artifacts/architecture.md#Development-Workflow-Integration]
+- [Source: _bmad-output/planning-artifacts/architecture.md#Implementation-Handoff]
+- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Journey-1-新用户从首次-setup-到第一次值得保留的成功体验]
+- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Component-Implementation-Strategy]
+- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Testing-Strategy]
+- [External: https://go.dev/doc/devel/release]
+- [External: https://github.com/spf13/cobra/releases/tag/v1.10.2]
+- [External: https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0-beta.5]
+- [External: https://www.postgresql.org/support/versioning/]
+
+## Dev Agent Record
+
+### Agent Model Used
+
+GPT-5 Codex
+
+### Debug Log References
+
+- Sprint auto-discovery selected `1-1-set-up-initial-project-from-starter-template` as the first backlog story.
+- No previous story file exists for Epic 1, so there are no earlier implementation learnings to inherit.
+- No product code exists at repository root yet; this story is the first formal implementation handoff.
+- Updated sprint tracking from `ready-for-dev` to `in-progress` before implementation.
+- Added starter conformance and package tests first, confirmed they failed against the empty baseline, then implemented the skeleton to make them pass.
+- Installed and ran `golangci-lint` locally after the Go baseline was in place so the configured lint gate could be exercised instead of skipped.
+- Resumed Story 1.1 from `review` after code review surfaced four issues spanning config precedence, version injection, Go baseline pinning, and nested-directory validation coverage.
+- Added regression tests around config layering, explicit flag override detection, injected version reporting, and real nested-directory bootstrap execution before re-running the full validation suite.
+
+### Completion Notes List
+
+- Ultimate context engine analysis completed - comprehensive developer guide created.
+- Implemented a Go 1.26.1 starter workspace with `gitdex` and `gitdexd` entrypoints, root tooling files, starter config, and policy placeholders.
+- Implemented Cobra-based command trees with `version`, `completion`, and daemon run paths, plus a Viper-backed config loader and daemon stub service.
+- Added schema placeholders, migration placeholders, reserved directory markers, unit tests, conformance tests, and integration tests for starter command execution.
+- Validation passed with `go test ./...`, `go run ./cmd/gitdex --help`, `go run ./cmd/gitdex completion powershell`, `go run ./cmd/gitdexd run`, and `golangci-lint run`.
+- Story completion moved the story artifact to `review` and the sprint tracking entry to `review`.
+- Fixed CLI/config precedence so only explicitly passed `--output`, `--log-level`, and `--profile` flags override Viper config or environment values.
+- Updated the starter baseline to pin `go 1.26.1`, made `internal/app/version.Version` build-injectable, and wired GoReleaser to stamp release binaries with the real version.
+- Corrected nested-directory bootstrap coverage by executing the daemon command from a real subdirectory and added conformance coverage for version ldflags injection.
+- Re-validated the story with `go test ./...`, `golangci-lint run`, `go run ./cmd/gitdex --help`, `go run ./cmd/gitdex completion powershell`, `go run ./cmd/gitdexd run`, and `go run -ldflags "-X github.com/your-org/gitdex/internal/app/version.Version=1.2.3-test" ./cmd/gitdex version`.
+
+### File List
+
+- .env.example
+- .gitignore
+- .golangci.yml
+- .goreleaser.yml
+- Makefile
+- README.md
+- Taskfile.yml
+- cmd/gitdex/main.go
+- cmd/gitdexd/main.go
+- configs/gitdex.example.yaml
+- configs/policies/default/global.yaml
+- configs/policies/default/repo_class_public.yaml
+- configs/policies/default/repo_class_release_critical.yaml
+- configs/policies/default/repo_class_sensitive.yaml
+- go.mod
+- go.sum
+- internal/app/bootstrap/bootstrap.go
+- internal/app/version/version.go
+- internal/cli/command/root.go
+- internal/cli/command/root_internal_test.go
+- internal/cli/command/root_test.go
+- internal/cli/completion/completion.go
+- internal/cli/output/format.go
+- internal/daemon/service/run.go
+- internal/daemon/service/run_test.go
+- internal/platform/config/config.go
+- internal/platform/config/config_test.go
+- internal/platform/ids/.gitkeep
+- internal/platform/logging/.gitkeep
+- migrations/000001_init.sql
+- migrations/000002_task_events.sql
+- migrations/000003_repo_projections.sql
+- migrations/000004_audit_records.sql
+- pkg/contracts/api/.gitkeep
+- pkg/contracts/audit/.gitkeep
+- pkg/contracts/campaign/.gitkeep
+- pkg/contracts/handoff/.gitkeep
+- pkg/contracts/plan/.gitkeep
+- pkg/contracts/task/.gitkeep
+- schema/json/api_error.schema.json
+- schema/json/audit_event.schema.json
+- schema/json/campaign.schema.json
+- schema/json/handoff_pack.schema.json
+- schema/json/plan.schema.json
+- schema/json/task.schema.json
+- schema/openapi/control_plane.yaml
+- scripts/ci/.gitkeep
+- scripts/dev/.gitkeep
+- scripts/fixtures/.gitkeep
+- test/conformance/starter_skeleton_test.go
+- test/contracts/.gitkeep
+- test/e2e/.gitkeep
+- test/fixtures/campaigns/.gitkeep
+- test/fixtures/policies/.gitkeep
+- test/fixtures/repos/.gitkeep
+- test/fixtures/webhooks/.gitkeep
+- test/integration/.gitkeep
+- test/integration/starter_commands_test.go
+- _bmad-output/implementation-artifacts/1-1-set-up-initial-project-from-starter-template.md
+- _bmad-output/implementation-artifacts/sprint-status.yaml
+
+### Change Log
+
+- 2026-03-18: Implemented Story 1.1 starter baseline, added command/config/test scaffolding, and validated the workspace through test, smoke, completion, daemon, and lint gates.
+- 2026-03-18: Addressed code review findings for Story 1.1 by fixing config precedence, enabling release version injection, pinning the Go baseline to 1.26.1, and strengthening nested-directory/conformance coverage.
--- /dev/null
+++ _bmad-output/implementation-artifacts/sprint-status.yaml
@@ -0,0 +1,97 @@
+# generated: 2026-03-18T21:28:33.8157165+08:00
+# last_updated: 2026-03-18T22:01:55.0000000+08:00
+# project: Gitdex
+# project_key: NOKEY
+# tracking_system: file-system
+# story_location: E:/Work/Engineering-Development/Gitdex/_bmad-output/implementation-artifacts
+
+# STATUS DEFINITIONS:
+# ==================
+# Epic Status:
+#   - backlog: Epic not yet started
+#   - in-progress: Epic actively being worked on
+#   - done: All stories in epic completed
+#
+# Epic Status Transitions:
+#   - backlog -> in-progress: Automatically when first story is created (via create-story)
+#   - in-progress -> done: Manually when all stories reach 'done' status
+#
+# Story Status:
+#   - backlog: Story only exists in epic file
+#   - ready-for-dev: Story file created in stories folder
+#   - in-progress: Developer actively working on implementation
+#   - review: Ready for code review (via Dev's code-review workflow)
+#   - done: Story completed
+#
+# Retrospective Status:
+#   - optional: Can be completed but not required
+#   - done: Retrospective has been completed
+#
+# WORKFLOW NOTES:
+# ===============
+# - Epic transitions to 'in-progress' automatically when first story is created
+# - Stories can be worked in parallel if team capacity allows
+# - SM typically creates next story after previous one is 'done' to incorporate learnings
+# - Dev moves story to 'review', then runs code-review (fresh context, different LLM recommended)
+
+generated: 2026-03-18T21:28:33.8157165+08:00
+last_updated: 2026-03-18T22:35:04.2830507+08:00
+project: Gitdex
+project_key: NOKEY
+tracking_system: file-system
+story_location: "E:/Work/Engineering-Development/Gitdex/_bmad-output/implementation-artifacts"
+
+development_status:
+  epic-1: in-progress
+  1-1-set-up-initial-project-from-starter-template: review
+  1-2-run-terminal-first-setup-and-environment-diagnostics: backlog
+  1-3-use-dual-mode-terminal-entry-with-discoverable-commands: backlog
+  1-4-connect-authorized-repositories-and-view-consolidated-state: backlog
+  1-5-operate-the-cockpit-in-rich-tui-and-text-only-modes: backlog
+  epic-1-retrospective: optional
+
+  epic-2: backlog
+  2-1-compile-structured-plans-from-commands-and-chat: backlog
+  2-2-review-approve-reject-edit-or-defer-a-plan: backlog
+  2-3-execute-approved-tasks-with-explicit-lifecycle-tracking: backlog
+  2-4-inspect-local-git-state-and-perform-controlled-upstream-sync: backlog
+  2-5-run-low-risk-repository-hygiene-tasks: backlog
+  2-6-apply-controlled-local-file-modifications-in-isolated-worktrees: backlog
+  epic-2-retrospective: optional
+
+  epic-3: backlog
+  3-1-authorize-gitdex-identity-and-scope-through-github-app: backlog
+  3-2-configure-policy-bundles-risk-tiers-and-execution-boundaries: backlog
+  3-3-inspect-audit-history-evidence-and-task-lineage: backlog
+  3-4-trigger-emergency-controls-and-containment: backlog
+  epic-3-retrospective: optional
+
+  epic-4: backlog
+  4-1-view-github-collaboration-objects-in-the-terminal: backlog
+  4-2-create-update-and-respond-to-collaboration-objects: backlog
+  4-3-triage-prioritize-and-summarize-inbound-activity: backlog
+  4-4-coordinate-cross-object-task-context: backlog
+  4-5-prepare-release-and-deployment-decisions-with-approval-aware-summaries: backlog
+  epic-4-retrospective: optional
+
+  epic-5: backlog
+  5-1-define-autonomy-levels-for-supported-capabilities: backlog
+  5-2-monitor-authorized-repositories-continuously-or-on-schedule: backlog
+  5-3-start-governed-tasks-from-events-schedules-apis-or-operators: backlog
+  5-4-pause-resume-cancel-and-take-over-autonomous-tasks: backlog
+  5-5-recover-from-blocked-failed-or-drifted-tasks: backlog
+  5-6-generate-handoff-packages-and-persist-long-running-task-state: backlog
+  epic-5-retrospective: optional
+
+  epic-6: backlog
+  6-1-define-a-governed-multi-repository-campaign: backlog
+  6-2-review-per-repository-plans-and-status-in-a-campaign-matrix: backlog
+  6-3-approve-exclude-and-intervene-per-repository-within-a-campaign: backlog
+  epic-6-retrospective: optional
+
+  epic-7: backlog
+  7-1-submit-structured-intents-plans-and-tasks-through-a-machine-api: backlog
+  7-2-query-task-campaign-and-audit-friendly-state-through-the-api: backlog
+  7-3-exchange-versioned-plans-results-and-status-with-external-tooling: backlog
+  7-4-export-plans-reports-and-handoff-artifacts-for-external-reuse: backlog
+  epic-7-retrospective: optional
--- /dev/null
+++ cmd/gitdex/main.go
@@ -0,0 +1,15 @@
+package main
+
+import (
+	"fmt"
+	"os"
+
+	"github.com/your-org/gitdex/internal/cli/command"
+)
+
+func main() {
+	if err := command.NewRootCommand().Execute(); err != nil {
+		_, _ = fmt.Fprintln(os.Stderr, err)
+		os.Exit(1)
+	}
+}
--- /dev/null
+++ cmd/gitdex/main_test.go
@@ -0,0 +1,25 @@
+package main
+
+import (
+	"bytes"
+	"testing"
+
+	"github.com/your-org/gitdex/internal/cli/command"
+)
+
+func TestGitdexEntrypointCommandExecutesVersion(t *testing.T) {
+	root := command.NewRootCommand()
+	var out bytes.Buffer
+
+	root.SetOut(&out)
+	root.SetErr(&out)
+	root.SetArgs([]string{"version"})
+
+	if err := root.Execute(); err != nil {
+		t.Fatalf("Execute returned error: %v", err)
+	}
+
+	if out.Len() == 0 {
+		t.Fatal("expected version output")
+	}
+}
--- /dev/null
+++ cmd/gitdexd/main.go
@@ -0,0 +1,15 @@
+package main
+
+import (
+	"fmt"
+	"os"
+
+	"github.com/your-org/gitdex/internal/cli/command"
+)
+
+func main() {
+	if err := command.NewDaemonBinaryRootCommand().Execute(); err != nil {
+		_, _ = fmt.Fprintln(os.Stderr, err)
+		os.Exit(1)
+	}
+}
--- /dev/null
+++ cmd/gitdexd/main_test.go
@@ -0,0 +1,25 @@
+package main
+
+import (
+	"bytes"
+	"testing"
+
+	"github.com/your-org/gitdex/internal/cli/command"
+)
+
+func TestGitdexdEntrypointCommandExecutesVersion(t *testing.T) {
+	root := command.NewDaemonBinaryRootCommand()
+	var out bytes.Buffer
+
+	root.SetOut(&out)
+	root.SetErr(&out)
+	root.SetArgs([]string{"version"})
+
+	if err := root.Execute(); err != nil {
+		t.Fatalf("Execute returned error: %v", err)
+	}
+
+	if out.Len() == 0 {
+		t.Fatal("expected version output")
+	}
+}
--- /dev/null
+++ configs/gitdex.example.yaml
@@ -0,0 +1,6 @@
+output: text
+log_level: info
+profile: local
+
+daemon:
+  health_address: 127.0.0.1:7777
--- /dev/null
+++ configs/policies/default/global.yaml
@@ -0,0 +1,2 @@
+bundle: default-global
+description: Starter placeholder for global Gitdex policy defaults.
--- /dev/null
+++ configs/policies/default/repo_class_public.yaml
@@ -0,0 +1,2 @@
+bundle: default-public
+description: Starter placeholder policy bundle for public repositories.
--- /dev/null
+++ configs/policies/default/repo_class_release_critical.yaml
@@ -0,0 +1,2 @@
+bundle: default-release-critical
+description: Starter placeholder policy bundle for release-critical repositories.
--- /dev/null
+++ configs/policies/default/repo_class_sensitive.yaml
@@ -0,0 +1,2 @@
+bundle: default-sensitive
+description: Starter placeholder policy bundle for sensitive repositories.
--- /dev/null
+++ go.mod
@@ -0,0 +1,24 @@
+module github.com/your-org/gitdex
+
+go 1.26.1
+
+require (
+	github.com/spf13/cobra v1.10.2
+	github.com/spf13/viper v1.21.0
+)
+
+require (
+	github.com/fsnotify/fsnotify v1.9.0 // indirect
+	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
+	github.com/inconshreveable/mousetrap v1.1.0 // indirect
+	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
+	github.com/sagikazarmark/locafero v0.11.0 // indirect
+	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
+	github.com/spf13/afero v1.15.0 // indirect
+	github.com/spf13/cast v1.10.0 // indirect
+	github.com/spf13/pflag v1.0.10 // indirect
+	github.com/subosito/gotenv v1.6.0 // indirect
+	go.yaml.in/yaml/v3 v3.0.4 // indirect
+	golang.org/x/sys v0.29.0 // indirect
+	golang.org/x/text v0.28.0 // indirect
+)
--- /dev/null
+++ go.sum
@@ -0,0 +1,54 @@
+github.com/cpuguy83/go-md2man/v2 v2.0.6/go.mod h1:oOW0eioCTA6cOiMLiUPZOpcVxMig6NIQQ7OS05n1F4g=
+github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
+github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
+github.com/frankban/quicktest v1.14.6 h1:7Xjx+VpznH+oBnejlPUj8oUpdxnVs4f8XU8WnHkI4W8=
+github.com/frankban/quicktest v1.14.6/go.mod h1:4ptaffx2x8+WTWXmUCuVU6aPUX1/Mz7zb5vbUoiM6w0=
+github.com/fsnotify/fsnotify v1.9.0 h1:2Ml+OJNzbYCTzsxtv8vKSFD9PbJjmhYF14k/jKC7S9k=
+github.com/fsnotify/fsnotify v1.9.0/go.mod h1:8jBTzvmWwFyi3Pb8djgCCO5IBqzKJ/Jwo8TRcHyHii0=
+github.com/go-viper/mapstructure/v2 v2.4.0 h1:EBsztssimR/CONLSZZ04E8qAkxNYq4Qp9LvH92wZUgs=
+github.com/go-viper/mapstructure/v2 v2.4.0/go.mod h1:oJDH3BJKyqBA2TXFhDsKDGDTlndYOZ6rGS0BRZIxGhM=
+github.com/google/go-cmp v0.6.0 h1:ofyhxvXcZhMsU5ulbFiLKl/XBFqE1GSq7atu8tAmTRI=
+github.com/google/go-cmp v0.6.0/go.mod h1:17dUlkBOakJ0+DkrSSNjCkIjxS6bF9zb3elmeNGIjoY=
+github.com/inconshreveable/mousetrap v1.1.0 h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=
+github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLfsEA9PFc4w1p2J65bw=
+github.com/kr/pretty v0.3.1 h1:flRD4NNwYAUpkphVc1HcthR4KEIFJ65n8Mw5qdRn3LE=
+github.com/kr/pretty v0.3.1/go.mod h1:hoEshYVHaxMs3cyo3Yncou5ZscifuDolrwPKZanG3xk=
+github.com/kr/text v0.2.0 h1:5Nx0Ya0ZqY2ygV366QzturHI13Jq95ApcVaJBhpS+AY=
+github.com/kr/text v0.2.0/go.mod h1:eLer722TekiGuMkidMxC/pM04lWEeraHUUmBw8l2grE=
+github.com/pelletier/go-toml/v2 v2.2.4 h1:mye9XuhQ6gvn5h28+VilKrrPoQVanw5PMw/TB0t5Ec4=
+github.com/pelletier/go-toml/v2 v2.2.4/go.mod h1:2gIqNv+qfxSVS7cM2xJQKtLSTLUE9V8t9Stt+h56mCY=
+github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
+github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
+github.com/rogpeppe/go-internal v1.9.0 h1:73kH8U+JUqXU8lRuOHeVHaa/SZPifC7BkcraZVejAe8=
+github.com/rogpeppe/go-internal v1.9.0/go.mod h1:WtVeX8xhTBvf0smdhujwtBcq4Qrzq/fJaraNFVN+nFs=
+github.com/russross/blackfriday/v2 v2.1.0/go.mod h1:+Rmxgy9KzJVeS9/2gXHxylqXiyQDYRxCVz55jmeOWTM=
+github.com/sagikazarmark/locafero v0.11.0 h1:1iurJgmM9G3PA/I+wWYIOw/5SyBtxapeHDcg+AAIFXc=
+github.com/sagikazarmark/locafero v0.11.0/go.mod h1:nVIGvgyzw595SUSUE6tvCp3YYTeHs15MvlmU87WwIik=
+github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 h1:+jumHNA0Wrelhe64i8F6HNlS8pkoyMv5sreGx2Ry5Rw=
+github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8/go.mod h1:3n1Cwaq1E1/1lhQhtRK2ts/ZwZEhjcQeJQ1RuC6Q/8U=
+github.com/spf13/afero v1.15.0 h1:b/YBCLWAJdFWJTN9cLhiXXcD7mzKn9Dm86dNnfyQw1I=
+github.com/spf13/afero v1.15.0/go.mod h1:NC2ByUVxtQs4b3sIUphxK0NioZnmxgyCrfzeuq8lxMg=
+github.com/spf13/cast v1.10.0 h1:h2x0u2shc1QuLHfxi+cTJvs30+ZAHOGRic8uyGTDWxY=
+github.com/spf13/cast v1.10.0/go.mod h1:jNfB8QC9IA6ZuY2ZjDp0KtFO2LZZlg4S/7bzP6qqeHo=
+github.com/spf13/cobra v1.10.2 h1:DMTTonx5m65Ic0GOoRY2c16WCbHxOOw6xxezuLaBpcU=
+github.com/spf13/cobra v1.10.2/go.mod h1:7C1pvHqHw5A4vrJfjNwvOdzYu0Gml16OCs2GRiTUUS4=
+github.com/spf13/pflag v1.0.9/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
+github.com/spf13/pflag v1.0.10 h1:4EBh2KAYBwaONj6b2Ye1GiHfwjqyROoF4RwYO+vPwFk=
+github.com/spf13/pflag v1.0.10/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
+github.com/spf13/viper v1.21.0 h1:x5S+0EU27Lbphp4UKm1C+1oQO+rKx36vfCoaVebLFSU=
+github.com/spf13/viper v1.21.0/go.mod h1:P0lhsswPGWD/1lZJ9ny3fYnVqxiegrlNrEmgLjbTCAY=
+github.com/stretchr/testify v1.11.1 h1:7s2iGBzp5EwR7/aIZr8ao5+dra3wiQyKjjFuvgVKu7U=
+github.com/stretchr/testify v1.11.1/go.mod h1:wZwfW3scLgRK+23gO65QZefKpKQRnfz6sD981Nm4B6U=
+github.com/subosito/gotenv v1.6.0 h1:9NlTDc1FTs4qu0DDq7AEtTPNw6SVm7uBMsUCUjABIf8=
+github.com/subosito/gotenv v1.6.0/go.mod h1:Dk4QP5c2W3ibzajGcXpNraDfq2IrhjMIvMSWPKKo0FU=
+go.yaml.in/yaml/v3 v3.0.4 h1:tfq32ie2Jv2UxXFdLJdh3jXuOzWiL1fo0bu/FbuKpbc=
+go.yaml.in/yaml/v3 v3.0.4/go.mod h1:DhzuOOF2ATzADvBadXxruRBLzYTpT36CKvDb3+aBEFg=
+golang.org/x/sys v0.29.0 h1:TPYlXGxvx1MGTn2GiZDhnjPA9wZzZeGKHHmKhHYvgaU=
+golang.org/x/sys v0.29.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
+golang.org/x/text v0.28.0 h1:rhazDwis8INMIwQ4tpjLDzUhx6RlXqZNPEM0huQojng=
+golang.org/x/text v0.28.0/go.mod h1:U8nCwOR8jO/marOQ0QbDiOngZVEBB7MAiitBuMjXiNU=
+gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
+gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 h1:YR8cESwS4TdDjEe65xsg0ogRM/Nc3DYOhEAlW+xobZo=
+gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
+gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
+gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
--- /dev/null
+++ internal/app/bootstrap/bootstrap.go
@@ -0,0 +1,48 @@
+package bootstrap
+
+import "github.com/your-org/gitdex/internal/platform/config"
+
+type App struct {
+	Config   config.Config
+	RepoRoot string
+	Version  string
+}
+
+type Options struct {
+	RepoRoot    string
+	ConfigFile  string
+	Output      string
+	OutputSet   bool
+	LogLevel    string
+	LogLevelSet bool
+	Profile     string
+	ProfileSet  bool
+	Version     string
+}
+
+func Load(opts Options) (App, error) {
+	repoRoot, err := config.ResolveRepoRoot(opts.RepoRoot)
+	if err != nil {
+		return App{}, err
+	}
+
+	cfg, err := config.Load(config.Options{
+		RepoRoot:    repoRoot,
+		ConfigFile:  opts.ConfigFile,
+		Output:      opts.Output,
+		OutputSet:   opts.OutputSet,
+		LogLevel:    opts.LogLevel,
+		LogLevelSet: opts.LogLevelSet,
+		Profile:     opts.Profile,
+		ProfileSet:  opts.ProfileSet,
+	})
+	if err != nil {
+		return App{}, err
+	}
+
+	return App{
+		Config:   cfg,
+		RepoRoot: repoRoot,
+		Version:  opts.Version,
+	}, nil
+}
--- /dev/null
+++ internal/app/bootstrap/bootstrap_test.go
@@ -0,0 +1,31 @@
+package bootstrap_test
+
+import (
+	"path/filepath"
+	"testing"
+
+	"github.com/your-org/gitdex/internal/app/bootstrap"
+)
+
+func TestLoadReturnsRepoRootAndVersion(t *testing.T) {
+	repoRoot, err := filepath.Abs("../../..")
+	if err != nil {
+		t.Fatalf("filepath.Abs failed: %v", err)
+	}
+
+	app, err := bootstrap.Load(bootstrap.Options{
+		RepoRoot: repoRoot,
+		Version:  "test-version",
+	})
+	if err != nil {
+		t.Fatalf("Load returned error: %v", err)
+	}
+
+	if app.RepoRoot != repoRoot {
+		t.Fatalf("RepoRoot = %q, want %q", app.RepoRoot, repoRoot)
+	}
+
+	if app.Version != "test-version" {
+		t.Fatalf("Version = %q, want %q", app.Version, "test-version")
+	}
+}
--- /dev/null
+++ internal/app/version/version.go
@@ -0,0 +1,8 @@
+package version
+
+const (
+	CLIName    = "gitdex"
+	DaemonName = "gitdexd"
+)
+
+var Version = "dev"
--- /dev/null
+++ internal/app/version/version_test.go
@@ -0,0 +1,19 @@
+package version_test
+
+import (
+	"testing"
+
+	"github.com/your-org/gitdex/internal/app/version"
+)
+
+func TestStarterVersionConstantsAreDefined(t *testing.T) {
+	if version.CLIName == "" {
+		t.Fatal("CLIName should not be empty")
+	}
+	if version.DaemonName == "" {
+		t.Fatal("DaemonName should not be empty")
+	}
+	if version.Version == "" {
+		t.Fatal("Version should not be empty")
+	}
+}
--- /dev/null
+++ internal/cli/command/root.go
@@ -0,0 +1,197 @@
+package command
+
+import (
+	"context"
+	"fmt"
+	"io"
+	"os"
+
+	"github.com/spf13/cobra"
+
+	"github.com/your-org/gitdex/internal/app/bootstrap"
+	"github.com/your-org/gitdex/internal/app/version"
+	cliCompletion "github.com/your-org/gitdex/internal/cli/completion"
+	"github.com/your-org/gitdex/internal/daemon/service"
+)
+
+type commandOptions struct {
+	in      io.Reader
+	out     io.Writer
+	errOut  io.Writer
+	use     string
+	version string
+}
+
+type runtimeOptions struct {
+	configFile string
+	output     string
+	logLevel   string
+	profile    string
+}
+
+func NewRootCommand() *cobra.Command {
+	return newCommandTree(commandOptions{
+		in:      os.Stdin,
+		out:     os.Stdout,
+		errOut:  os.Stderr,
+		use:     version.CLIName,
+		version: version.Version,
+	}, true)
+}
+
+func NewDaemonBinaryRootCommand() *cobra.Command {
+	return newCommandTree(commandOptions{
+		in:      os.Stdin,
+		out:     os.Stdout,
+		errOut:  os.Stderr,
+		use:     version.DaemonName,
+		version: version.Version,
+	}, false)
+}
+
+func newCommandTree(opts commandOptions, includeDaemonGroup bool) *cobra.Command {
+	if opts.in == nil {
+		opts.in = os.Stdin
+	}
+	if opts.out == nil {
+		opts.out = os.Stdout
+	}
+	if opts.errOut == nil {
+		opts.errOut = os.Stderr
+	}
+	if opts.version == "" {
+		opts.version = version.Version
+	}
+
+	flags := runtimeOptions{}
+	var app bootstrap.App
+
+	root := &cobra.Command{
+		Use:           opts.use,
+		Short:         "Gitdex starter baseline",
+		SilenceErrors: true,
+		SilenceUsage:  true,
+		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
+			if shouldSkipBootstrap(cmd) {
+				return nil
+			}
+
+			loaded, err := bootstrap.Load(buildBootstrapOptions(flags, opts.version, func(name string) bool {
+				return commandFlagChanged(cmd, name)
+			}))
+			if err != nil {
+				return err
+			}
+
+			app = loaded
+			return nil
+		},
+		RunE: func(cmd *cobra.Command, args []string) error {
+			return cmd.Help()
+		},
+	}
+
+	root.SetIn(opts.in)
+	root.SetOut(opts.out)
+	root.SetErr(opts.errOut)
+
+	root.PersistentFlags().StringVar(&flags.configFile, "config", "", "Path to a Gitdex config file")
+	root.PersistentFlags().StringVar(&flags.output, "output", "text", "Output format")
+	root.PersistentFlags().StringVar(&flags.logLevel, "log-level", "info", "Log level")
+	root.PersistentFlags().StringVar(&flags.profile, "profile", "local", "Runtime profile")
+
+	root.AddCommand(newVersionCommand(opts.version))
+	root.AddCommand(cliCompletion.NewCommand(root))
+
+	if includeDaemonGroup {
+		root.AddCommand(newDaemonGroupCommand(func() bootstrap.App { return app }, opts.version))
+	} else {
+		root.AddCommand(newDaemonRunCommand(func() bootstrap.App { return app }, opts.version, version.DaemonName))
+	}
+
+	return root
+}
+
+func buildBootstrapOptions(flags runtimeOptions, currentVersion string, flagChanged func(string) bool) bootstrap.Options {
+	if flagChanged == nil {
+		flagChanged = func(string) bool { return false }
+	}
+
+	return bootstrap.Options{
+		ConfigFile:  flags.configFile,
+		Output:      flags.output,
+		OutputSet:   flagChanged("output"),
+		LogLevel:    flags.logLevel,
+		LogLevelSet: flagChanged("log-level"),
+		Profile:     flags.profile,
+		ProfileSet:  flagChanged("profile"),
+		Version:     currentVersion,
+	}
+}
+
+func commandFlagChanged(cmd *cobra.Command, name string) bool {
+	if cmd == nil {
+		return false
+	}
+
+	flag := cmd.Flags().Lookup(name)
+	return flag != nil && flag.Changed
+}
+
+func shouldSkipBootstrap(cmd *cobra.Command) bool {
+	if cmd == nil {
+		return false
+	}
+
+	switch cmd.Name() {
+	case "completion", "help", "version":
+		return true
+	default:
+		return false
+	}
+}
+
+func newVersionCommand(currentVersion string) *cobra.Command {
+	return &cobra.Command{
+		Use:   "version",
+		Short: "Print the current starter version",
+		RunE: func(cmd *cobra.Command, args []string) error {
+			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", currentVersion)
+			return err
+		},
+	}
+}
+
+func newDaemonGroupCommand(appFn func() bootstrap.App, currentVersion string) *cobra.Command {
+	cmd := &cobra.Command{
+		Use:   "daemon",
+		Short: "Run local daemon-oriented starter commands",
+	}
+	cmd.AddCommand(newDaemonRunCommand(appFn, currentVersion, version.CLIName))
+	return cmd
+}
+
+func newDaemonRunCommand(appFn func() bootstrap.App, currentVersion, binaryName string) *cobra.Command {
+	return &cobra.Command{
+		Use:   "run",
+		Short: "Start the starter daemon stub",
+		RunE: func(cmd *cobra.Command, args []string) error {
+			app := appFn()
+			healthAddress := app.Config.Daemon.HealthAddress
+			if healthAddress == "" {
+				healthAddress = "127.0.0.1:7777"
+			}
+
+			name := binaryName
+			if binaryName == version.CLIName {
+				name = "gitdex daemon"
+			}
+
+			return service.Run(context.Background(), cmd.OutOrStdout(), service.Options{
+				Version:       currentVersion,
+				BinaryName:    name,
+				HealthAddress: healthAddress,
+			})
+		},
+	}
+}
--- /dev/null
+++ internal/cli/command/root_internal_test.go
@@ -0,0 +1,63 @@
+package command
+
+import "testing"
+
+func TestBuildBootstrapOptionsOnlyMarksExplicitFlagsAsOverrides(t *testing.T) {
+	got := buildBootstrapOptions(runtimeOptions{
+		configFile: "configs/gitdex.example.yaml",
+		output:     "text",
+		logLevel:   "info",
+		profile:    "local",
+	}, "test-version", func(string) bool { return false })
+
+	if got.ConfigFile != "configs/gitdex.example.yaml" {
+		t.Fatalf("ConfigFile = %q, want %q", got.ConfigFile, "configs/gitdex.example.yaml")
+	}
+
+	if got.OutputSet {
+		t.Fatal("OutputSet should be false when --output is not explicitly passed")
+	}
+
+	if got.LogLevelSet {
+		t.Fatal("LogLevelSet should be false when --log-level is not explicitly passed")
+	}
+
+	if got.ProfileSet {
+		t.Fatal("ProfileSet should be false when --profile is not explicitly passed")
+	}
+}
+
+func TestBuildBootstrapOptionsMarksChangedFlags(t *testing.T) {
+	changed := map[string]bool{
+		"output":  true,
+		"profile": true,
+	}
+
+	got := buildBootstrapOptions(runtimeOptions{
+		output:   "json",
+		logLevel: "info",
+		profile:  "prod",
+	}, "test-version", func(name string) bool {
+		return changed[name]
+	})
+
+	if !got.OutputSet {
+		t.Fatal("OutputSet should be true when --output is explicitly passed")
+	}
+
+	if got.LogLevelSet {
+		t.Fatal("LogLevelSet should be false when --log-level is not explicitly passed")
+	}
+
+	if !got.ProfileSet {
+		t.Fatal("ProfileSet should be true when --profile is explicitly passed")
+	}
+
+	if got.Output != "json" {
+		t.Fatalf("Output = %q, want %q", got.Output, "json")
+	}
+
+	if got.Profile != "prod" {
+		t.Fatalf("Profile = %q, want %q", got.Profile, "prod")
+	}
+}
--- /dev/null
+++ internal/cli/command/root_test.go
@@ -0,0 +1,125 @@
+package command_test
+
+import (
+	"bytes"
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+
+	"github.com/your-org/gitdex/internal/app/version"
+	"github.com/your-org/gitdex/internal/cli/command"
+)
+
+func TestNewRootCommandExposesStarterSubcommands(t *testing.T) {
+	root := command.NewRootCommand()
+
+	if got, want := root.Use, "gitdex"; got != want {
+		t.Fatalf("root.Use = %q, want %q", got, want)
+	}
+
+	required := map[string]bool{
+		"completion": false,
+		"daemon":     false,
+		"version":    false,
+	}
+
+	for _, sub := range root.Commands() {
+		if _, ok := required[sub.Name()]; ok {
+			required[sub.Name()] = true
+		}
+	}
+
+	for name, found := range required {
+		if !found {
+			t.Fatalf("expected subcommand %q to be registered", name)
+		}
+	}
+}
+
+func TestVersionCommandDoesNotRequireRepoBootstrap(t *testing.T) {
+	originalWD, err := os.Getwd()
+	if err != nil {
+		t.Fatalf("os.Getwd failed: %v", err)
+	}
+	defer func() {
+		_ = os.Chdir(originalWD)
+	}()
+
+	tempDir := t.TempDir()
+	if err := os.Chdir(tempDir); err != nil {
+		t.Fatalf("os.Chdir failed: %v", err)
+	}
+
+	root := command.NewRootCommand()
+	var out bytes.Buffer
+	root.SetOut(&out)
+	root.SetErr(&out)
+	root.SetArgs([]string{"version"})
+
+	if err := root.Execute(); err != nil {
+		t.Fatalf("version command should not require repo root: %v", err)
+	}
+
+	if strings.TrimSpace(out.String()) == "" {
+		t.Fatal("expected version output")
+	}
+}
+
+func TestDaemonRunStillBootstrapsFromNestedDirectory(t *testing.T) {
+	originalWD, err := os.Getwd()
+	if err != nil {
+		t.Fatalf("os.Getwd failed: %v", err)
+	}
+	defer func() {
+		_ = os.Chdir(originalWD)
+	}()
+
+	nestedDir := filepath.Join(originalWD, "testdata", "nested", "deeper")
+	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
+		t.Fatalf("os.MkdirAll failed: %v", err)
+	}
+	defer func() {
+		_ = os.RemoveAll(filepath.Join(originalWD, "testdata"))
+	}()
+
+	if err := os.Chdir(nestedDir); err != nil {
+		t.Fatalf("os.Chdir failed: %v", err)
+	}
+
+	root := command.NewRootCommand()
+	var out bytes.Buffer
+	root.SetOut(&out)
+	root.SetErr(&out)
+	root.SetArgs([]string{"daemon", "run"})
+
+	if err := root.Execute(); err != nil {
+		t.Fatalf("daemon run should bootstrap from nested repo directory: %v", err)
+	}
+
+	if !strings.Contains(out.String(), "starter baseline") {
+		t.Fatalf("expected daemon output, got %q", out.String())
+	}
+}
+
+func TestVersionCommandUsesInjectedVersionValue(t *testing.T) {
+	originalVersion := version.Version
+	version.Version = "1.2.3-test"
+	defer func() {
+		version.Version = originalVersion
+	}()
+
+	root := command.NewRootCommand()
+	var out bytes.Buffer
+	root.SetOut(&out)
+	root.SetErr(&out)
+	root.SetArgs([]string{"version"})
+
+	if err := root.Execute(); err != nil {
+		t.Fatalf("Execute returned error: %v", err)
+	}
+
+	if got, want := strings.TrimSpace(out.String()), "1.2.3-test"; got != want {
+		t.Fatalf("version output = %q, want %q", got, want)
+	}
+}
--- /dev/null
+++ internal/cli/completion/completion.go
@@ -0,0 +1,30 @@
+package completion
+
+import (
+	"fmt"
+
+	"github.com/spf13/cobra"
+)
+
+func NewCommand(root *cobra.Command) *cobra.Command {
+	return &cobra.Command{
+		Use:       "completion [bash|zsh|fish|powershell]",
+		Short:     "Generate shell completion scripts",
+		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
+		Args:      cobra.ExactArgs(1),
+		RunE: func(cmd *cobra.Command, args []string) error {
+			switch args[0] {
+			case "bash":
+				return root.GenBashCompletionV2(cmd.OutOrStdout(), true)
+			case "zsh":
+				return root.GenZshCompletion(cmd.OutOrStdout())
+			case "fish":
+				return root.GenFishCompletion(cmd.OutOrStdout(), true)
+			case "powershell":
+				return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
+			default:
+				return fmt.Errorf("unsupported shell %q", args[0])
+			}
+		},
+	}
+}
--- /dev/null
+++ internal/cli/completion/completion_test.go
@@ -0,0 +1,48 @@
+package completion_test
+
+import (
+	"bytes"
+	"strings"
+	"testing"
+
+	"github.com/spf13/cobra"
+
+	"github.com/your-org/gitdex/internal/cli/completion"
+)
+
+func TestCompletionCommandRejectsUnsupportedShell(t *testing.T) {
+	root := &cobra.Command{Use: "gitdex"}
+	cmd := completion.NewCommand(root)
+	var out bytes.Buffer
+
+	cmd.SetOut(&out)
+	cmd.SetErr(&out)
+	cmd.SetArgs([]string{"invalid-shell"})
+
+	err := cmd.Execute()
+	if err == nil {
+		t.Fatal("expected error for unsupported shell")
+	}
+
+	if !strings.Contains(err.Error(), "unsupported shell") && !strings.Contains(err.Error(), "accepts 1 arg") {
+		t.Fatalf("unexpected error: %v", err)
+	}
+}
+
+func TestCompletionCommandGeneratesPowerShellScript(t *testing.T) {
+	root := &cobra.Command{Use: "gitdex"}
+	cmd := completion.NewCommand(root)
+	var out bytes.Buffer
+
+	cmd.SetOut(&out)
+	cmd.SetErr(&out)
+	cmd.SetArgs([]string{"powershell"})
+
+	if err := cmd.Execute(); err != nil {
+		t.Fatalf("Execute returned error: %v", err)
+	}
+
+	if !strings.Contains(out.String(), "powershell completion") {
+		t.Fatalf("expected powershell completion script, got %q", out.String())
+	}
+}
--- /dev/null
+++ internal/cli/output/format.go
@@ -0,0 +1,7 @@
+package output
+
+const (
+	FormatText = "text"
+	FormatJSON = "json"
+	FormatYAML = "yaml"
+)
--- /dev/null
+++ internal/cli/output/format_test.go
@@ -0,0 +1,25 @@
+package output_test
+
+import (
+	"testing"
+
+	"github.com/your-org/gitdex/internal/cli/output"
+)
+
+func TestFormatsAreStableAndDistinct(t *testing.T) {
+	formats := map[string]bool{
+		output.FormatText: false,
+		output.FormatJSON: false,
+		output.FormatYAML: false,
+	}
+
+	if len(formats) != 3 {
+		t.Fatalf("expected 3 distinct formats, got %d", len(formats))
+	}
+
+	for format := range formats {
+		if format == "" {
+			t.Fatal("format should not be empty")
+		}
+	}
+}
--- /dev/null
+++ internal/daemon/service/run.go
@@ -0,0 +1,42 @@
+package service
+
+import (
+	"context"
+	"fmt"
+	"io"
+
+	"github.com/your-org/gitdex/internal/app/version"
+)
+
+type Options struct {
+	Version       string
+	BinaryName    string
+	HealthAddress string
+}
+
+func Run(ctx context.Context, out io.Writer, opts Options) error {
+	select {
+	case <-ctx.Done():
+		return ctx.Err()
+	default:
+	}
+
+	if opts.Version == "" {
+		opts.Version = version.Version
+	}
+	if opts.BinaryName == "" {
+		opts.BinaryName = version.DaemonName
+	}
+	if opts.HealthAddress == "" {
+		opts.HealthAddress = "127.0.0.1:7777"
+	}
+
+	_, err := fmt.Fprintf(
+		out,
+		"%s starter baseline active (version %s)\nhealth stub configured for %s\n",
+		opts.BinaryName,
+		opts.Version,
+		opts.HealthAddress,
+	)
+	return err
+}
--- /dev/null
+++ internal/daemon/service/run_test.go
@@ -0,0 +1,30 @@
+package service_test
+
+import (
+	"bytes"
+	"context"
+	"strings"
+	"testing"
+
+	"github.com/your-org/gitdex/internal/daemon/service"
+)
+
+func TestRunWritesStarterBaselineMessage(t *testing.T) {
+	var out bytes.Buffer
+
+	err := service.Run(context.Background(), &out, service.Options{
+		Version: "dev",
+	})
+	if err != nil {
+		t.Fatalf("Run returned error: %v", err)
+	}
+
+	text := out.String()
+	if !strings.Contains(text, "starter baseline") {
+		t.Fatalf("expected starter baseline message, got %q", text)
+	}
+
+	if !strings.Contains(text, "gitdexd") {
+		t.Fatalf("expected daemon binary name in output, got %q", text)
+	}
+}
--- /dev/null
+++ internal/platform/config/config.go
@@ -0,0 +1,112 @@
+package config
+
+import (
+	"fmt"
+	"os"
+	"path/filepath"
+	"strings"
+
+	"github.com/spf13/viper"
+)
+
+const envPrefix = "GITDEX"
+
+type Config struct {
+	ExampleConfigPath string
+	ConfigFile        string
+	Output            string
+	LogLevel          string
+	Profile           string
+	Daemon            DaemonConfig
+}
+
+type DaemonConfig struct {
+	HealthAddress string
+}
+
+type Options struct {
+	RepoRoot    string
+	ConfigFile  string
+	Output      string
+	OutputSet   bool
+	LogLevel    string
+	LogLevelSet bool
+	Profile     string
+	ProfileSet  bool
+}
+
+func ResolveRepoRoot(start string) (string, error) {
+	if start == "" {
+		wd, err := os.Getwd()
+		if err != nil {
+			return "", fmt.Errorf("resolve working directory: %w", err)
+		}
+		start = wd
+	}
+
+	current, err := filepath.Abs(start)
+	if err != nil {
+		return "", fmt.Errorf("resolve repo root from %q: %w", start, err)
+	}
+
+	for {
+		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
+			return current, nil
+		}
+
+		parent := filepath.Dir(current)
+		if parent == current {
+			return "", fmt.Errorf("go.mod not found from %q upward", start)
+		}
+		current = parent
+	}
+}
+
+func Load(opts Options) (Config, error) {
+	repoRoot, err := ResolveRepoRoot(opts.RepoRoot)
+	if err != nil {
+		return Config{}, err
+	}
+
+	v := viper.New()
+	v.SetEnvPrefix(envPrefix)
+	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
+	v.AutomaticEnv()
+
+	v.SetDefault("output", "text")
+	v.SetDefault("log_level", "info")
+	v.SetDefault("profile", "local")
+	v.SetDefault("daemon.health_address", "127.0.0.1:7777")
+
+	configFile := opts.ConfigFile
+	if configFile == "" {
+		configFile = os.Getenv(envPrefix + "_CONFIG")
+	}
+	if configFile != "" {
+		v.SetConfigFile(configFile)
+		if err := v.ReadInConfig(); err != nil {
+			return Config{}, fmt.Errorf("read config file: %w", err)
+		}
+	}
+
+	if opts.OutputSet {
+		v.Set("output", opts.Output)
+	}
+	if opts.LogLevelSet {
+		v.Set("log_level", opts.LogLevel)
+	}
+	if opts.ProfileSet {
+		v.Set("profile", opts.Profile)
+	}
+
+	return Config{
+		ExampleConfigPath: filepath.Join(repoRoot, "configs", "gitdex.example.yaml"),
+		ConfigFile:        v.ConfigFileUsed(),
+		Output:            v.GetString("output"),
+		LogLevel:          v.GetString("log_level"),
+		Profile:           v.GetString("profile"),
+		Daemon: DaemonConfig{
+			HealthAddress: v.GetString("daemon.health_address"),
+		},
+	}, nil
+}
--- /dev/null
+++ internal/platform/config/config_test.go
@@ -0,0 +1,168 @@
+package config_test
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+
+	configpkg "github.com/your-org/gitdex/internal/platform/config"
+)
+
+func TestLoadDefaultsUsesExampleConfigPath(t *testing.T) {
+	repoRoot := repoRoot(t)
+
+	cfg, err := configpkg.Load(configpkg.Options{
+		RepoRoot: repoRoot,
+	})
+	if err != nil {
+		t.Fatalf("Load returned error: %v", err)
+	}
+
+	expectedConfigPath := filepath.Join(repoRoot, "configs", "gitdex.example.yaml")
+	if cfg.ExampleConfigPath != expectedConfigPath {
+		t.Fatalf("ExampleConfigPath = %q, want %q", cfg.ExampleConfigPath, expectedConfigPath)
+	}
+
+	if cfg.Output != "text" {
+		t.Fatalf("Output = %q, want %q", cfg.Output, "text")
+	}
+
+	if cfg.LogLevel != "info" {
+		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "info")
+	}
+
+	if cfg.Profile != "local" {
+		t.Fatalf("Profile = %q, want %q", cfg.Profile, "local")
+	}
+}
+
+func TestLoadUsesConfigFileValuesWhenOverridesAreUnset(t *testing.T) {
+	repoRoot := repoRoot(t)
+	configFile := writeConfigFile(t, `output: json
+log_level: debug
+profile: prod
+daemon:
+  health_address: 127.0.0.1:9999
+`)
+
+	cfg, err := configpkg.Load(configpkg.Options{
+		RepoRoot:   repoRoot,
+		ConfigFile: configFile,
+		Output:     "text",
+		LogLevel:   "info",
+		Profile:    "local",
+	})
+	if err != nil {
+		t.Fatalf("Load returned error: %v", err)
+	}
+
+	if cfg.Output != "json" {
+		t.Fatalf("Output = %q, want %q", cfg.Output, "json")
+	}
+
+	if cfg.LogLevel != "debug" {
+		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
+	}
+
+	if cfg.Profile != "prod" {
+		t.Fatalf("Profile = %q, want %q", cfg.Profile, "prod")
+	}
+
+	if cfg.Daemon.HealthAddress != "127.0.0.1:9999" {
+		t.Fatalf("Daemon.HealthAddress = %q, want %q", cfg.Daemon.HealthAddress, "127.0.0.1:9999")
+	}
+}
+
+func TestLoadUsesEnvironmentValuesWhenOverridesAreUnset(t *testing.T) {
+	repoRoot := repoRoot(t)
+	t.Setenv("GITDEX_OUTPUT", "yaml")
+	t.Setenv("GITDEX_LOG_LEVEL", "warn")
+	t.Setenv("GITDEX_PROFILE", "ci")
+
+	cfg, err := configpkg.Load(configpkg.Options{
+		RepoRoot: repoRoot,
+		Output:   "text",
+		LogLevel: "info",
+		Profile:  "local",
+	})
+	if err != nil {
+		t.Fatalf("Load returned error: %v", err)
+	}
+
+	if cfg.Output != "yaml" {
+		t.Fatalf("Output = %q, want %q", cfg.Output, "yaml")
+	}
+
+	if cfg.LogLevel != "warn" {
+		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
+	}
+
+	if cfg.Profile != "ci" {
+		t.Fatalf("Profile = %q, want %q", cfg.Profile, "ci")
+	}
+}
+
+func TestLoadUsesExplicitOverridesWhenSet(t *testing.T) {
+	repoRoot := repoRoot(t)
+	configFile := writeConfigFile(t, `output: json
+log_level: debug
+profile: prod
+daemon:
+  health_address: 127.0.0.1:9999
+`)
+	t.Setenv("GITDEX_OUTPUT", "yaml")
+	t.Setenv("GITDEX_LOG_LEVEL", "warn")
+	t.Setenv("GITDEX_PROFILE", "ci")
+
+	cfg, err := configpkg.Load(configpkg.Options{
+		RepoRoot:    repoRoot,
+		ConfigFile:  configFile,
+		Output:      "text",
+		OutputSet:   true,
+		LogLevel:    "info",
+		LogLevelSet: true,
+		Profile:     "local",
+		ProfileSet:  true,
+	})
+	if err != nil {
+		t.Fatalf("Load returned error: %v", err)
+	}
+
+	if cfg.Output != "text" {
+		t.Fatalf("Output = %q, want %q", cfg.Output, "text")
+	}
+
+	if cfg.LogLevel != "info" {
+		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "info")
+	}
+
+	if cfg.Profile != "local" {
+		t.Fatalf("Profile = %q, want %q", cfg.Profile, "local")
+	}
+
+	if cfg.Daemon.HealthAddress != "127.0.0.1:9999" {
+		t.Fatalf("Daemon.HealthAddress = %q, want %q", cfg.Daemon.HealthAddress, "127.0.0.1:9999")
+	}
+}
+
+func repoRoot(t *testing.T) string {
+	t.Helper()
+
+	repoRoot, err := filepath.Abs("../../../")
+	if err != nil {
+		t.Fatalf("filepath.Abs failed: %v", err)
+	}
+
+	return repoRoot
+}
+
+func writeConfigFile(t *testing.T, content string) string {
+	t.Helper()
+
+	configFile := filepath.Join(t.TempDir(), "gitdex.yaml")
+	if err := os.WriteFile(configFile, []byte(content), 0o600); err != nil {
+		t.Fatalf("os.WriteFile failed: %v", err)
+	}
+
+	return configFile
+}
--- /dev/null
+++ internal/platform/ids/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ internal/platform/logging/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ migrations/000001_init.sql
@@ -0,0 +1 @@
+-- Starter placeholder migration: initialize core tables.
--- /dev/null
+++ migrations/000002_task_events.sql
@@ -0,0 +1 @@
+-- Starter placeholder migration: task event log tables.
--- /dev/null
+++ migrations/000003_repo_projections.sql
@@ -0,0 +1 @@
+-- Starter placeholder migration: repository projections.
--- /dev/null
+++ migrations/000004_audit_records.sql
@@ -0,0 +1 @@
+-- Starter placeholder migration: audit records.
--- /dev/null
+++ pkg/contracts/api/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ pkg/contracts/audit/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ pkg/contracts/campaign/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ pkg/contracts/handoff/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ pkg/contracts/plan/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ pkg/contracts/task/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ schema/json/api_error.schema.json
@@ -0,0 +1,6 @@
+{
+  "$schema": "https://json-schema.org/draft/2020-12/schema",
+  "title": "Gitdex API Error",
+  "type": "object",
+  "description": "Starter placeholder schema for API errors."
+}
--- /dev/null
+++ schema/json/audit_event.schema.json
@@ -0,0 +1,6 @@
+{
+  "$schema": "https://json-schema.org/draft/2020-12/schema",
+  "title": "Gitdex Audit Event",
+  "type": "object",
+  "description": "Starter placeholder schema for audit events."
+}
--- /dev/null
+++ schema/json/campaign.schema.json
@@ -0,0 +1,6 @@
+{
+  "$schema": "https://json-schema.org/draft/2020-12/schema",
+  "title": "Gitdex Campaign",
+  "type": "object",
+  "description": "Starter placeholder schema for campaign state."
+}
--- /dev/null
+++ schema/json/handoff_pack.schema.json
@@ -0,0 +1,6 @@
+{
+  "$schema": "https://json-schema.org/draft/2020-12/schema",
+  "title": "Gitdex Handoff Pack",
+  "type": "object",
+  "description": "Starter placeholder schema for handoff packages."
+}
--- /dev/null
+++ schema/json/plan.schema.json
@@ -0,0 +1,6 @@
+{
+  "$schema": "https://json-schema.org/draft/2020-12/schema",
+  "title": "Gitdex Plan",
+  "type": "object",
+  "description": "Starter placeholder schema for structured plans."
+}
--- /dev/null
+++ schema/json/task.schema.json
@@ -0,0 +1,6 @@
+{
+  "$schema": "https://json-schema.org/draft/2020-12/schema",
+  "title": "Gitdex Task",
+  "type": "object",
+  "description": "Starter placeholder schema for task state."
+}
--- /dev/null
+++ schema/openapi/control_plane.yaml
@@ -0,0 +1,5 @@
+openapi: 3.1.0
+info:
+  title: Gitdex Control Plane API
+  version: 0.0.0-starter
+paths: {}
--- /dev/null
+++ scripts/ci/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ scripts/dev/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ scripts/fixtures/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ test/conformance/starter_skeleton_test.go
@@ -0,0 +1,91 @@
+package conformance_test
+
+import (
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+)
+
+func TestStarterSkeletonContainsRequiredPaths(t *testing.T) {
+	repoRoot, err := filepath.Abs("../..")
+	if err != nil {
+		t.Fatalf("filepath.Abs failed: %v", err)
+	}
+
+	requiredPaths := []string{
+		"README.md",
+		".gitignore",
+		".env.example",
+		"Taskfile.yml",
+		"Makefile",
+		".golangci.yml",
+		".goreleaser.yml",
+		"configs/gitdex.example.yaml",
+		"configs/policies/default/global.yaml",
+		"configs/policies/default/repo_class_public.yaml",
+		"configs/policies/default/repo_class_sensitive.yaml",
+		"configs/policies/default/repo_class_release_critical.yaml",
+		"cmd/gitdex/main.go",
+		"cmd/gitdexd/main.go",
+		"internal/app/bootstrap",
+		"internal/app/version",
+		"internal/cli/command/root.go",
+		"internal/cli/completion/completion.go",
+		"internal/cli/output",
+		"internal/platform/config/config.go",
+		"internal/platform/ids",
+		"internal/platform/logging",
+		"pkg/contracts/plan",
+		"pkg/contracts/task",
+		"pkg/contracts/audit",
+		"pkg/contracts/campaign",
+		"pkg/contracts/handoff",
+		"pkg/contracts/api",
+		"schema/json/plan.schema.json",
+		"schema/json/task.schema.json",
+		"schema/json/campaign.schema.json",
+		"schema/json/audit_event.schema.json",
+		"schema/json/handoff_pack.schema.json",
+		"schema/json/api_error.schema.json",
+		"schema/openapi/control_plane.yaml",
+		"migrations/000001_init.sql",
+		"migrations/000002_task_events.sql",
+		"migrations/000003_repo_projections.sql",
+		"migrations/000004_audit_records.sql",
+		"scripts/dev",
+		"scripts/ci",
+		"scripts/fixtures",
+		"test/integration",
+		"test/e2e",
+		"test/contracts",
+		"test/conformance",
+		"test/fixtures/repos",
+		"test/fixtures/policies",
+		"test/fixtures/webhooks",
+		"test/fixtures/campaigns",
+	}
+
+	for _, rel := range requiredPaths {
+		if _, err := os.Stat(filepath.Join(repoRoot, rel)); err != nil {
+			t.Fatalf("required starter path %q missing: %v", rel, err)
+		}
+	}
+}
+
+func TestGoreleaserInjectsBuildVersion(t *testing.T) {
+	repoRoot, err := filepath.Abs("../..")
+	if err != nil {
+		t.Fatalf("filepath.Abs failed: %v", err)
+	}
+
+	content, err := os.ReadFile(filepath.Join(repoRoot, ".goreleaser.yml"))
+	if err != nil {
+		t.Fatalf("os.ReadFile failed: %v", err)
+	}
+
+	const want = "github.com/your-org/gitdex/internal/app/version.Version={{.Version}}"
+	if !strings.Contains(string(content), want) {
+		t.Fatalf(".goreleaser.yml does not inject version with %q", want)
+	}
+}
--- /dev/null
+++ test/contracts/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ test/e2e/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ test/fixtures/campaigns/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ test/fixtures/policies/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ test/fixtures/repos/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ test/fixtures/webhooks/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ test/integration/.gitkeep
@@ -0,0 +1 @@
+
--- /dev/null
+++ test/integration/starter_commands_test.go
@@ -0,0 +1,54 @@
+package integration_test
+
+import (
+	"bytes"
+	"strings"
+	"testing"
+
+	"github.com/spf13/cobra"
+
+	"github.com/your-org/gitdex/internal/cli/command"
+)
+
+func TestStarterCommandsExecuteDaemonRunPaths(t *testing.T) {
+	tests := []struct {
+		name string
+		cmd  func() *cobra.Command
+		args []string
+	}{
+		{
+			name: "gitdex daemon run",
+			cmd: func() *cobra.Command {
+				return command.NewRootCommand()
+			},
+			args: []string{"daemon", "run"},
+		},
+		{
+			name: "gitdexd run",
+			cmd: func() *cobra.Command {
+				return command.NewDaemonBinaryRootCommand()
+			},
+			args: []string{"run"},
+		},
+	}
+
+	for _, tt := range tests {
+		t.Run(tt.name, func(t *testing.T) {
+			root := tt.cmd()
+			var out bytes.Buffer
+
+			root.SetOut(&out)
+			root.SetErr(&out)
+			root.SetArgs(tt.args)
+
+			if err := root.Execute(); err != nil {
+				t.Fatalf("Execute returned error: %v", err)
+			}
+
+			output := out.String()
+			if !strings.Contains(output, "starter baseline") {
+				t.Fatalf("expected starter baseline output, got %q", output)
+			}
+		})
+	}
+}
```
