# Story 1.3: Use Dual-Mode Terminal Entry with Discoverable Commands

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

作为仓库操作者，
我希望在同一个终端会话中同时使用显式命令和自然语言聊天，
以便在精确控制和自由探索之间随意切换，不必更换工具，也不丢失当前任务上下文。

## Acceptance Criteria

1. **Given** 一个处于活动状态的 Gitdex 终端会话
   **When** 操作者发出显式命令或自然语言请求
   **Then** 两种模式在同一任务上下文中运行，可以无损地互相切换，不丢失作用域

2. **Given** 操作者在终端内需要发现可用能力
   **When** 操作者使用帮助或发现机制
   **Then** Gitdex 暴露可发现的命令帮助、能力列表和对象可用操作

3. **Given** 命令或聊天产生了结果输出
   **When** 操作者选择输出格式
   **Then** 受支持的输出可以以 human-readable text 和机器可读 JSON/YAML 两种形式发出

## Tasks / Subtasks

- [x] 实现双模输入解析与路由层 (AC: 1)
  - [x] 在 `internal/cli/input/` 创建输入解析器，能区分显式命令（以 Cobra 命令树匹配）和自然语言自由文本
  - [x] 定义输入分类策略：先尝试 Cobra 命令解析，解析失败的输入归入自然语言通道；支持显式 `chat` 子命令直接进入聊天模式
  - [x] 实现共享的 `TaskContext` 结构体，在命令和聊天之间传递当前 repo scope、活跃对象引用和会话状态

- [x] 实现自然语言聊天入口与 LLM adapter 骨架 (AC: 1)
  - [x] 新增 `gitdex chat` 子命令，支持单次消息模式（`gitdex chat "当前仓库有什么需要处理的？"`）和交互式 REPL 模式（`gitdex chat --interactive`）
  - [x] 在 `internal/llm/adapter/` 创建 LLM provider adapter 接口，定义 `ChatCompletion`、`StreamChatCompletion` 契约，支持可注入的 provider 实现
  - [x] 实现至少一个 LLM adapter stub（如 OpenAI 兼容接口），用于开发期验证；adapter 必须支持注入 mock 以保持测试稳定
  - [x] 聊天请求的系统提示词必须限定 Gitdex 的职责边界——状态理解、意图解析、计划草拟、风险解释——不允许 LLM 直接调用执行器

- [x] 实现命令与聊天的上下文共享机制 (AC: 1)
  - [x] `TaskContext` 至少包含：当前 repo path/scope、活跃配置 profile、会话开始时间、已执行命令历史摘要、当前对话历史（截断到合理 token 窗口）
  - [x] 命令执行结果可以被注入聊天上下文（如 `gitdex doctor` 结果可供后续 `chat` 引用）
  - [x] 聊天中识别出的结构化意图可以路由回 Cobra 命令树执行（如聊天中说"同步上游"可转为 `gitdex repo sync` 命令路径的占位逻辑）
  - [x] 上下文在同一进程会话内保持；跨会话持久化留作后续 story

- [x] 实现命令发现与可发现帮助系统 (AC: 2)
  - [x] 确保所有已注册 Cobra 命令都有完整的 `Short`、`Long`、`Example` 描述，且描述语言与 config 中 `communication_language` 一致或至少提供英文基线
  - [x] 新增 `gitdex help <object>` 模式，除了标准 Cobra help 外，支持按对象域查看可用操作（如 `gitdex help repo` 列出 repo 相关所有子命令和能力）
  - [x] 新增 `gitdex capabilities` 命令，输出当前环境下所有可用能力的结构化列表，包括命令路径、简述和可用性状态
  - [x] 确保 shell completion 在 Story 1.1 基础上继续覆盖新增的 `chat`、`capabilities` 等子命令

- [x] 扩展输出格式系统以覆盖双模输出 (AC: 3)
  - [x] 在 Story 1.2 建立的 `internal/cli/output/` 基础上，扩展 format 系统使其能处理聊天响应、命令结果、能力列表和帮助输出
  - [x] 所有新增输出都必须支持 `--output text`（默认 human-readable）和 `--output json` 两种形式
  - [x] 聊天响应在 text 模式下以对话格式呈现；在 json 模式下以 `{"role": "assistant", "content": "...", "context": {...}}` 结构输出
  - [x] capabilities 列表在 json 模式下以 `{"capabilities": [{"command": "...", "description": "...", "available": true}]}` 结构输出

- [x] 建立稳定的诊断结果模型供聊天上下文复用 (AC: 1, 2)
  - [x] 将 Story 1.2 `doctor` 和 `config show` 的输出接入 `TaskContext`，使聊天可以引用当前环境诊断结果
  - [x] 对象域发现机制可以读取命令树和当前授权状态，动态生成"当前可用能力"回答

- [x] 补齐测试与回归验证 (AC: 1, 2, 3)
  - [x] 为输入解析器补单元测试：命令匹配、自由文本分类、边界情况（空输入、仅 flag、未知子命令）
  - [x] 为 `gitdex chat` 补集成测试：使用 mock LLM adapter 验证单次消息模式和交互模式的基本流转
  - [x] 为 `gitdex capabilities` 补集成测试：验证 text 和 json 输出格式的稳定性
  - [x] 为上下文共享补测试：命令结果注入聊天上下文后可被正确引用
  - [x] 所有新增命令的 shell completion 回归验证
  - [x] 本 story 完成时至少跑通：
    - `go test ./...`
    - `go run ./cmd/gitdex chat "hello"`
    - `go run ./cmd/gitdex chat --help`
    - `go run ./cmd/gitdex capabilities`
    - `go run ./cmd/gitdex capabilities --output json`
    - `go run ./cmd/gitdex help repo`
    - `golangci-lint run`

- [x] 控制范围，避免提前实现后续能力 (AC: 1, 2, 3)
  - [x] 不实现真实的 LLM provider 密钥管理或生产级认证——使用环境变量或 config 中的 API key 字段即可
  - [x] 不实现 structured plan compiler——聊天中识别出写意图时只记录到 `TaskContext` 并输出提示，不自动生成执行计划
  - [x] 不实现 rich TUI 交互式聊天面板——当前只做 text-first CLI 层面的聊天体验
  - [x] 不实现跨会话对话持久化——对话历史仅在当前进程生命周期内保持
  - [x] 不实现 repo state summary、GitHub API 调用或 webhook intake——这些属于 Story 1.4 和后续 epic
  - [x] 不把 LLM adapter 绑死到特定 provider——必须通过接口抽象，支持后续替换

## Dev Notes

- Story 1.1 和 1.2 已经提供了可工作的 Cobra 命令树、Viper 配置加载、shell completion、`init`/`doctor`/`config show` 命令、多层配置模型和诊断结果模型。本 story 在此基础上增加自然语言入口和命令发现能力。
- Story 1.2 的 review 确认了以下约束仍然有效，本 story 必须继承：
  - 只有显式传入的 flag 才能覆盖配置层（`--output`、`--log-level`、`--profile`）
  - Go baseline 锁定 `1.26.1`
  - 测试不得写入真实用户目录；全局配置路径重定向到临时目录
  - 嵌套目录测试必须真实进入子目录且可清理
- 当前仓库不是 Git working tree，因此没有 commit history 可用于额外模式提炼。
- PRD 和架构明确要求：聊天只能提出意图，不能绕过结构化计划直接执行写操作。本 story 中 LLM 产出的任何写意图只做记录和提示，不触发执行。
- UX 规范中的 Intent Composer（UX-DR5）描述了统一命令与聊天输入的组件，但完整的 Intent Composer 是 TUI 层能力。本 story 只实现 CLI text-first 的基础版本：命令行参数/子命令 + `chat` 子命令交互模式。

### Technical Requirements

- 继续使用 Go 1.26.1 / Cobra v1.10.2 / Viper v1.21.0 基线，不替换 CLI 框架。
- LLM adapter 接口设计为 `internal/llm/adapter/`，定义核心接口：

```go
type ChatMessage struct {
    Role    string // "system", "user", "assistant"
    Content string
}

type ChatRequest struct {
    Messages    []ChatMessage
    MaxTokens   int
    Temperature float64
    Stream      bool
}

type ChatResponse struct {
    Content      string
    FinishReason string
    Usage        TokenUsage
}

type Provider interface {
    ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    StreamChatCompletion(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error)
}
```

- 输入解析策略：`internal/cli/input/parser.go` 负责判断输入是命令还是自然语言。核心逻辑为先尝试 Cobra 命令匹配（通过 `rootCmd.Find()` 或等价方式），匹配失败则归类为自然语言输入。
- `TaskContext` 定义在 `internal/cli/context/` 或 `internal/app/session/` 中，作为进程内共享状态容器。
- 聊天系统提示词必须包含 Gitdex 身份说明和职责边界，避免 LLM 越权回答代码生成、直接执行或策略绕过类请求。
- 配置中需要新增 LLM 相关字段（`llm.provider`、`llm.model`、`llm.api_key`、`llm.endpoint`），这些字段走 Viper 配置加载，优先级与 Story 1.2 一致。

### Architecture Compliance

- 命令 wiring 留在 `internal/cli/command/`；输入解析逻辑放 `internal/cli/input/`；LLM adapter 放 `internal/llm/adapter/`。
- 聊天业务逻辑应放在 `internal/app/chat/` 服务层，不堆进 Cobra handler。
- `TaskContext` 放 `internal/app/session/` 或等价位置，与 CLI command handler 和 chat service 双向可访问。
- 架构要求 `planning` 不能直接调用 `execution`。本 story 中聊天识别出写意图时，只做记录和提示，不绕过 plan compiler 直接执行——plan compiler 是 Epic 2 的能力。
- `internal/llm/adapter/` 只做 LLM 通信，不做策略评估。策略评估属于 `internal/policy/`（后续 epic）。
- 输出格式扩展继续在 `internal/cli/output/` 中进行，保持与 Story 1.2 的格式系统一致。
- `gitdexd` daemon 在本 story 中不新增职责；双模交互只通过 `gitdex` CLI 入口。

### Library / Framework Requirements

- 继续使用 `github.com/spf13/cobra v1.10.2` 扩展命令树。
- 继续使用 `github.com/spf13/viper v1.21.0`，新增 LLM 配置字段走相同多层配置路径。
- LLM HTTP 通信使用 Go 标准库 `net/http`，不引入第三方 LLM SDK 作为硬依赖。adapter 接口设计为可替换，开发期可选用 `github.com/sashabaranov/go-openai` 或直接 HTTP 调用 OpenAI 兼容接口。如果引入第三方 SDK，必须通过接口隔离。
- 交互式 REPL 模式可使用 `github.com/reeflective/readline` 或等效纯 Go readline 库提供历史、补全和多行输入能力。如果引入，需在 `go.mod` 中显式管理。
- **不引入** Bubble Tea v2 作为本 story 依赖——Bubble Tea 用于 rich TUI（Story 1.5），本 story 只做 text-first CLI 层面的聊天。
- 测试中 LLM 调用必须可 mock，不依赖真实 API endpoint。

### File Structure Requirements

- 本 story 预计主要触及下列区域：
  - `internal/cli/command/chat.go` — `gitdex chat` 子命令
  - `internal/cli/command/capabilities.go` — `gitdex capabilities` 子命令
  - `internal/cli/command/root.go` — 扩展命令注册
  - `internal/cli/input/parser.go` — 输入分类器
  - `internal/cli/input/parser_test.go`
  - `internal/cli/output/format.go` — 扩展输出格式
  - `internal/app/chat/service.go` — 聊天业务逻辑
  - `internal/app/chat/service_test.go`
  - `internal/app/session/context.go` — TaskContext 定义
  - `internal/app/session/context_test.go`
  - `internal/llm/adapter/provider.go` — Provider 接口
  - `internal/llm/adapter/openai.go` — OpenAI 兼容 adapter
  - `internal/llm/adapter/mock.go` — 测试用 mock adapter
  - `internal/llm/adapter/provider_test.go`
  - `internal/llm/guardrails/system_prompt.go` — 系统提示词模板
  - `configs/gitdex.example.yaml` — 新增 LLM 配置示例
  - `test/integration/chat_commands_test.go`
  - `test/integration/capabilities_test.go`
  - `test/conformance/dual_mode_test.go`

- 如需新增共享服务，优先考虑：
  - `internal/app/chat/`
  - `internal/app/session/`
  - `internal/llm/adapter/`
  - `internal/llm/guardrails/`

- 不在 `cmd/gitdex/main.go` 中写业务逻辑；main 仍只负责调用 command tree。

### Testing Requirements

- 单元测试必须覆盖：
  - 输入分类器对命令 vs 自然语言的正确分类
  - TaskContext 的创建、更新和命令结果注入
  - 聊天服务的消息构建、系统提示注入和 token 截断逻辑
  - LLM adapter 的请求构建和响应解析
  - 输出格式器对聊天响应的 text/json 渲染
  - capabilities 列表的动态生成
- 集成测试必须覆盖：
  - `gitdex chat "hello"` 使用 mock adapter 成功返回
  - `gitdex chat --interactive` 使用 mock adapter 进入并退出 REPL
  - `gitdex capabilities` 的 text 和 json 输出稳定断言
  - `gitdex help repo` 输出正确的子命令列表
  - 命令结果注入聊天上下文后的引用验证
- 测试不得写入真实用户目录；所有配置路径重定向到临时目录。
- 测试不得依赖真实 LLM API endpoint；所有 LLM 调用通过 mock adapter 完成。
- 本 story 完成时至少跑通：
  - `go test ./...`
  - `go run ./cmd/gitdex chat "hello"`
  - `go run ./cmd/gitdex chat --help`
  - `go run ./cmd/gitdex capabilities`
  - `go run ./cmd/gitdex capabilities --output json`
  - `go run ./cmd/gitdex help repo`
  - `go run ./cmd/gitdexd run`
  - `golangci-lint run`

### Previous Story Intelligence

- Story 1.1 落地了 starter 骨架：双入口二进制、Cobra 命令树、Viper 配置加载、completion、daemon stub、配置与 conformance / integration 测试骨架。
- Story 1.1 review 修复了 4 个问题：flag 默认值不能压盖 env/config、版本号可注入、Go baseline 锁定 1.26.1、嵌套目录测试必须真实进入子目录且可清理。
- Story 1.2 落地了多层配置模型（global/repo/session/env）、`gitdex init`/`gitdex doctor`/`gitdex config show` 命令、诊断结果模型（每个 check 暴露 `id`/`status`/`summary`/`detail`/`fix`/`source`）、以及相关测试。
- Story 1.2 的关键设计约束：
  - 诊断逻辑抽到共享服务层，不直接写在 Cobra handler 中
  - 连接性测试通过可注入 checker 完成，不依赖真实 GitHub API
  - setup 后必须返回配置写入结果、诊断摘要、下一步命令建议
  - 区分"尚未配置""配置不完整""已配置但验证失败""验证通过"四类状态
- Story 1.2 的文件列表提供了本 story 需要直接扩展的入口点：
  - `internal/cli/command/root.go` — 注册新子命令
  - `internal/cli/output/format.go` — 扩展输出格式
  - `internal/platform/config/config.go` — 新增 LLM 配置字段
  - `configs/gitdex.example.yaml` — 新增 LLM 配置示例
  - `test/integration/onboarding_commands_test.go` — 参考集成测试模式

### Latest Technical Validation

- 截至 2026-03-18，Go 官方 release history 仍列出 `go1.26.1`（2026-03-05 发布）；继续锁定 `Go 1.26.1` 合理。Go 1.26 引入了 Green Tea GC（默认启用，减少 10-40% GC 开销）和重写的 `go fix`，对 CLI 和长期运行应用均有性能收益。
- 截至 2026-03-18，Cobra 官方仓库仍显示 `v1.10.2` 为最新版本；shell completion 继续覆盖 bash/zsh/fish/powershell。Cobra 支持动态命令注册（`AddCommand()`）和自定义 flag completion（`RegisterFlagCompletionFunc()`），可用于命令发现扩展。
- Bubble Tea 已从 `v2.0.0-beta.5` 正式发布为 `v2.0.2`（2026-03 稳定版），import path 变更为 `charm.land/bubbletea/v2`。但本 story 不引入 Bubble Tea——TUI 属于 Story 1.5。
- OpenAI Go SDK（`github.com/sashabaranov/go-openai`）当前版本 v1.41.2+，支持 chat completion、streaming、function calling 和 structured output。Anthropic Go SDK（`github.com/anthropics/anthropic-sdk-go`）v1.27.0+ 可用。本 story LLM adapter 设计为接口抽象，不硬绑定特定 SDK。
- 交互式 REPL 可选用 `github.com/reeflective/readline`（v1.1.4，纯 Go，支持 Emacs/Vim 模式、多行输入、语法高亮）或 `github.com/reeflective/console`（v0.1.25，直接集成 Cobra + readline）。
- Viper `v1.21.0` 仍是仓库锁定版本；单实例只支持一个配置文件，多层配置需要显式 merge——与 Story 1.2 一致。

### Project Context Reference

- 当前工作区未检测到 `project-context.md`。
- 本 story 的权威上下文来源为：`epics.md`、`prd.md`、`architecture.md`、`ux-design-specification.md`、Story 1.1 和 Story 1.2 实现记录。

### Project Structure Notes

- Story 1.3 是 Epic 1 从"基础设施搭建"跨入"产品入口体验"的转折点。它引入了 Gitdex 的第一个真正的产品差异化能力——双模终端交互。
- 但本 story 仍然是 Phase 1 的窄范围基础设施：不应越级实现 repo orchestration、policy engine、task state machine 或 structured plan compiler。
- LLM adapter 在本 story 中是骨架级实现，不需要生产级错误恢复、rate limiting 或 token 预算管理。这些能力随后续 epic 逐步加入。
- 由于架构要求 Windows / Linux / macOS 统一操作语义，readline/REPL 库选型必须跨平台可用，避免 Unix-only 依赖。
- 当前仓库承载 BMAD 规划与参考材料；新增产品代码不得散落到 `_bmad-output/`、`docs/` 或 `reference_project/` 中。

### Non-Goals / Scope Guardrails

- 不实现真实 GitHub App 认证、installation token 获取或 GitHub API 调用。
- 不实现 repo state summary、repo scan 或 GitHub 对象聚合——属于 Story 1.4。
- 不实现 structured plan compiler、policy engine 或 approval router——属于 Epic 2。
- 不实现 rich TUI 驾驶舱、Bubble Tea 交互面板或多面板布局——属于 Story 1.5。
- 不实现 PostgreSQL 持久层、任务状态机或审计账本。
- 不实现跨会话对话持久化或 daemon 侧聊天处理。
- 不实现 LLM 的 function calling / tool use 来驱动命令执行——聊天中的写意图仅做记录和提示。
- 不为了"先跑起来"而引入长期 PAT 作为默认身份模式。

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-13-Use-Dual-Mode-Terminal-Entry-with-Discoverable-Commands-FR1-FR2-FR3-FR46-FR47]
- [Source: _bmad-output/planning-artifacts/prd.md#Operator-Interaction--Context-Assembly]
- [Source: _bmad-output/planning-artifacts/prd.md#Command-Structure]
- [Source: _bmad-output/planning-artifacts/prd.md#Output-Formats]
- [Source: _bmad-output/planning-artifacts/prd.md#Config-Schema]
- [Source: _bmad-output/planning-artifacts/prd.md#Terminal-UX-Requirements]
- [Source: _bmad-output/planning-artifacts/prd.md#Scripting-Support]
- [Source: _bmad-output/planning-artifacts/architecture.md#Selected-Starter-Cobra-Based-Go-Workspace-Foundation]
- [Source: _bmad-output/planning-artifacts/architecture.md#Core-Architectural-Decisions]
- [Source: _bmad-output/planning-artifacts/architecture.md#LLM-and-Cognitive-Architecture]
- [Source: _bmad-output/planning-artifacts/architecture.md#Operator-Experience-Plane]
- [Source: _bmad-output/planning-artifacts/architecture.md#Naming-Patterns]
- [Source: _bmad-output/planning-artifacts/architecture.md#Structure-Patterns]
- [Source: _bmad-output/planning-artifacts/architecture.md#Project-Structure--Boundaries]
- [Source: _bmad-output/planning-artifacts/architecture.md#Implementation-Handoff]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Core-Experience-Foundation]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Custom-Components-Intent-Composer]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Journey-2-维护者从自然语言请求到计划审查再到受治理执行]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Feedback-Patterns]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Navigation-Patterns]
- [Source: _bmad-output/implementation-artifacts/1-1-set-up-initial-project-from-starter-template.md]
- [Source: _bmad-output/implementation-artifacts/1-2-run-terminal-first-setup-and-environment-diagnostics.md]
- [External: https://go.dev/doc/devel/release — Go 1.26.1 release notes]
- [External: https://github.com/spf13/cobra/releases/tag/v1.10.2]
- [External: https://github.com/spf13/viper/releases/tag/v1.21.0]
- [External: https://github.com/sashabaranov/go-openai — Go OpenAI SDK]
- [External: https://github.com/anthropics/anthropic-sdk-go — Anthropic Go SDK]
- [External: https://github.com/reeflective/readline — Pure Go readline library]

## Senior Developer Review (AI)

### Review Date: 2026-03-18

### Review Outcome: Approve (after 1 iteration)

### Round 1 Findings (7 items — all resolved):

- [x] **[CRITICAL]** LLM config from file was never used — `resolveProvider` only checked env vars. **Fix:** Wired `bootstrap.App.Config.LLM` into `resolveProvider` via `appFn` closure.
- [x] **[CRITICAL]** `TaskContext.Metadata` direct map access without lock in test. **Fix:** Added `GetMetadata()` thread-safe getter; test updated.
- [x] **[HIGH]** `TaskContext.RepoPath` always empty (never populated from bootstrap). **Fix:** `getOrCreateSession` now receives `app.RepoRoot`.
- [x] **[HIGH]** `RepoPath`/`Profile` read without lock in service and guardrails. **Fix:** Added `GetRepoPath()`/`GetProfile()` getters used everywhere.
- [x] **[HIGH]** Chat JSON output included extra `Response` field beyond spec. **Fix:** Removed `Response` field from `ChatResult`.
- [x] **[MEDIUM]** Chat command ignored output format from config. **Fix:** Passes `app.Config.Output` to `effectiveOutputFormat`.
- [x] **[MEDIUM]** `TestChatSingleMessage_OutputFormatFromEnv` assertion too weak. **Fix:** Validates JSON parsing + required fields.

### Round 2 Verdict: APPROVE — all fixes verified, no new issues.

## Dev Agent Record

### Agent Model Used

claude-4.6-opus-max-thinking

### Debug Log References

- User explicitly requested Story `1.3`.
- Sprint tracking showed `1-3-use-dual-mode-terminal-entry-with-discoverable-commands` as the next backlog item in Epic 1.
- Previous story files `1-1-set-up-initial-project-from-starter-template.md` and `1-2-run-terminal-first-setup-and-environment-diagnostics.md` were loaded and mined for implementation constraints, review fixes, test patterns, and file structure.
- No `.git` working tree was detected, so commit-history intelligence was skipped.
- No `project-context.md` was found; context came from epics, PRD, architecture, UX, Story 1.1, and Story 1.2.
- Latest technical validation was refreshed against official Go, Cobra, Viper, Bubble Tea, OpenAI Go SDK, and Anthropic Go SDK sources on 2026-03-18.

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- Task 1: Created `internal/cli/input/parser.go` with Cobra-first classification strategy (InputCommand/InputNaturalLanguage/InputEmpty).
- Task 1: Created `internal/app/session/context.go` with TaskContext supporting RepoPath, Profile, CommandHistory, ChatHistory, DiagnosticData, Metadata.
- Task 2: Created `internal/llm/adapter/provider.go` defining Provider interface with ChatCompletion/StreamChatCompletion contracts.
- Task 2: Created `internal/llm/adapter/mock.go` (injectable mock) and `openai.go` (OpenAI-compatible HTTP adapter).
- Task 2: Created `internal/llm/guardrails/system_prompt.go` with boundary-enforcing system prompt (MUST NOT execute, MUST NOT bypass plan).
- Task 2: Created `internal/app/chat/service.go` as business logic layer, builds messages with system prompt, truncates history at 50 messages.
- Task 2: Created `internal/cli/command/chat.go` with single-message and interactive REPL modes, command detection in REPL.
- Task 3: TaskContext.InjectCommandResult() bridges command output into chat context as system messages.
- Task 3: Interactive REPL detects Cobra commands via parser and suggests running them directly.
- Task 3: Chat history lives in TaskContext for the process lifetime; no cross-session persistence (by design).
- Task 4: Created `internal/cli/command/capabilities.go` with text/json/yaml output, walks command tree dynamically.
- Task 4: Created `internal/cli/command/repo.go` as placeholder command group for `gitdex help repo`.
- Task 5: Chat JSON output uses `{"role","content","context","response"}` schema. Capabilities uses `{"capabilities":[...]}` schema.
- Task 6: TaskContext.DiagnosticData and InjectCommandResult enable doctor/config results to flow into chat.
- Task 6: Capabilities discovery reads command tree dynamically — no hardcoded list.
- Config: Added LLMConfig (provider, model, api_key, endpoint) to FileConfig with full Viper/env layer support.
- All tests pass: 18 packages, golangci-lint clean.
- Scope guardrails verified: no Bubble Tea, no function calling, no plan compiler, no cross-session persistence, no GitHub API calls.
- Code review round 1: 2 CRITICAL + 3 HIGH + 2 MEDIUM findings identified and resolved.
- Code review round 2: APPROVE — all fixes verified, no new issues.

### Change Log

- 2026-03-18: Story 1.3 implementation complete — dual-mode terminal entry with discoverable commands.
- 2026-03-18: Code review round 1 — addressed 7 findings (2 CRITICAL, 3 HIGH, 2 MEDIUM).
- 2026-03-18: Code review round 2 — APPROVED, status → done.

### File List

- internal/app/session/context.go (new)
- internal/app/session/context_test.go (new)
- internal/app/chat/service.go (new)
- internal/app/chat/service_test.go (new)
- internal/cli/input/parser.go (new)
- internal/cli/input/parser_test.go (new)
- internal/cli/command/chat.go (new)
- internal/cli/command/capabilities.go (new)
- internal/cli/command/repo.go (new)
- internal/cli/command/root.go (modified — register chat, capabilities, repo commands)
- internal/llm/adapter/provider.go (new)
- internal/llm/adapter/mock.go (new)
- internal/llm/adapter/openai.go (new)
- internal/llm/adapter/provider_test.go (new)
- internal/llm/guardrails/system_prompt.go (new)
- internal/llm/guardrails/system_prompt_test.go (new)
- internal/platform/config/config.go (modified — added LLMConfig struct and layer support)
- configs/gitdex.example.yaml (modified — added llm section)
- test/integration/chat_commands_test.go (new)
- test/integration/capabilities_test.go (new)
- test/conformance/dual_mode_test.go (new)
- _bmad-output/implementation-artifacts/sprint-status.yaml (modified)
- _bmad-output/implementation-artifacts/1-3-use-dual-mode-terminal-entry-with-discoverable-commands.md
