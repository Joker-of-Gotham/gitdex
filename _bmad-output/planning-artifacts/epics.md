---
stepsCompleted: [1, 2, 3, 4]
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
---

# Gitdex - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown foundation for Gitdex, decomposing the requirements from the PRD, UX Design, and Architecture into implementable stories.

## Requirements Inventory

### Functional Requirements

FR1: Operators can use explicit terminal commands to access Gitdex capabilities.
FR2: Operators can use natural-language chat in the terminal to express goals, ask questions, and request assistance.
FR3: Operators can move between command-driven and chat-driven workflows within the same task context.
FR4: Operators can view consolidated repository state spanning local Git, remote repository, collaboration activity, and automation status.
FR5: Operators can request explanations of current state, material risks, and evidence-backed next actions for a selected repository, task, or campaign scope.
FR6: Operators can inspect the evidence and source objects behind Gitdex summaries, recommendations, and decisions.
FR7: Operators can turn commands or natural-language goals into structured execution plans before governed write actions occur.
FR8: Operators can preview intended actions, affected objects, and risk level for a plan before execution.
FR9: Operators can approve, reject, edit, or defer a plan when review is required.
FR10: Operators can run supported tasks in observation, recommendation, dry-run, or execution mode.
FR11: Gitdex can execute approved plans as tracked tasks with explicit lifecycle states.
FR12: Gitdex can explain why a requested action is allowed, blocked, escalated, or downgraded.
FR13: Gitdex can preserve traceability between user intent, generated plan, policy decision, execution results, and evidence.
FR14: Operators can inspect and manage local repository working state, branches, diffs, and synchronization status.
FR15: Operators can request upstream comparison, sync recommendations, and controlled synchronization actions.
FR16: Operators can perform governed low-risk repository hygiene and maintenance tasks.
FR17: Operators can request controlled local file modifications within an authorized repository scope.
FR18: Operators can view issues, pull requests, comments, reviews, workflows, and deployment status from the terminal.
FR19: Operators can create, update, and respond to supported GitHub collaboration objects from within Gitdex.
FR20: Operators can ask Gitdex to triage, prioritize, and summarize incoming issues, pull requests, and comment activity within a defined repository or campaign scope.
FR21: Operators can coordinate branch, PR, issue, comment, workflow, and deployment context as part of a single tracked task or structured plan.
FR22: Operators can prepare release or deployment-related decisions through governed summaries, checks, and approval-aware workflows.
FR23: Repository owners can define autonomy levels for supported capabilities and scopes.
FR24: Gitdex can monitor authorized repositories continuously or on schedules for explicitly supported maintenance and governance scenarios.
FR25: Gitdex can start governed tasks from repository events, schedules, API requests, or operator requests.
FR26: Operators can pause, resume, cancel, or take over autonomous tasks without losing task context.
FR27: Gitdex can recover from blocked, failed, or incomplete tasks through supported retry, reconciliation, quarantine, or safe handoff paths that preserve task state and evidence.
FR28: Gitdex can generate handoff packages for tasks that require human continuation.
FR29: Gitdex can maintain long-running task state across terminal sessions and background processing windows until the task reaches a terminal or handoff state.
FR30: Administrators can authorize Gitdex at repository, installation, organization, or fleet scope with bounded capabilities.
FR31: Administrators can define policies for approvals, risk tiers, protected targets, and execution boundaries.
FR32: Gitdex can enforce policy decisions consistently across command, chat, API, integration, and autonomous entry points.
FR33: Gitdex can record complete audit trails for governed actions, approvals, policy evaluations, security-relevant events, and task outcomes.
FR34: Operators and administrators can inspect audit history, evidence, and task lineage for any governed action.
FR35: Authorized users can trigger emergency controls such as pause, capability suspension, or kill switch actions.
FR36: Administrators can define data-handling rules for logs, caches, model use, and external integrations by scope, retention policy, and sensitivity class.
FR37: Operators can define and run governed campaigns across two or more repositories within an authorized repository set.
FR38: Operators can review per-repository plans, statuses, and outcomes within a campaign.
FR39: Operators can approve, exclude, or intervene on individual repositories within a campaign.
FR40: Integrators can submit structured intents, plans, or tasks to Gitdex through machine-facing interfaces.
FR41: Integrators can query task state, campaign state, reports, and audit-friendly outputs from Gitdex.
FR42: Gitdex can exchange structured plans, results, and status with CI systems, IDEs, agent runtimes, and internal tooling.
FR43: Administrators can apply shared policy bundles, defaults, and governance settings across defined groups of repositories within an authorized administrative scope.
FR44: Users can complete terminal-based initial setup for identity, permissions, defaults, and operating preferences.
FR45: Users can configure Gitdex through global, repository, session, and environment-specific settings.
FR46: Users can select human-readable or structured output formats for supported commands, plans, reports, and task results.
FR47: Users can discover available capabilities, command patterns, and object actions from within Gitdex.
FR48: Users can diagnose environment, authorization, configuration, and connectivity issues from within the product.
FR49: Users can export plans, reports, handoff packages, and other structured artifacts for reuse in external workflows.
FR50: Users can apply Gitdex consistently across Windows, Linux, and macOS environments while preserving the same core operating model.

### NonFunctional Requirements

NFR1: 单仓库范围内的状态读取、摘要查看和对象查询类请求，在已具备认证与基础上下文的前提下，按正常负载下滚动 `7` 天应用遥测统计，`P95 <= 5 秒`。
NFR2: 单仓库范围内由自然语言或命令触发的结构化计划生成，在正式支持范围内按滚动 `7` 天应用遥测与每日抽样回放统计，`90% <= 60 秒` 完成，`P95 <= 90 秒`，并以任务进入 `plan generated` 或 `plan review required` 状态作为完成判定。
NFR3: 对已生成计划的策略评估、风险解释和 dry-run 预览，在上下文已收集完毕后按滚动 `7` 天应用遥测统计，`P95 <= 10 秒`。
NFR4: 对正式支持的 GitHub webhook 事件类型，入站确认必须按滚动 `30` 天 delivery telemetry 统计达到 `99.9% <= 10 秒`，并通过 webhook replay/integration suite 证明最慢路径仍采用异步后处理模式。
NFR5: 活跃任务状态刷新、阻塞原因查询和最近一次状态转换查询，在正常负载下按滚动 `7` 天应用遥测统计 `P95 <= 5 秒`。
NFR6: 在正式支持的低风险与中风险操作集合中，按滚动 `30` 天统计窗口计算，分母为已进入执行阶段的受支持任务且排除用户主动取消与策略阻止项，任务最终成功收敛率必须 `>= 95%`。
NFR7: `100%` 的任务必须最终收敛到以下状态之一：`succeeded`、`cancelled`、`failed with handoff complete`、`blocked by policy/approval`；该要求必须通过滚动 `30` 天任务状态对账作业与日度审计扫描证明。
NFR8: 对于进入失败、阻塞或需人工接管状态的任务，`handoff pack` 必须在 `60 秒` 内可用；该目标必须通过失败路径集成测试和滚动 `30` 天异常任务遥测统计验证。
NFR9: 对于排队中或可中断阶段的任务，`safe pause`、`cancel`、`kill switch` 指令必须在 `30 秒` 内生效；若外部副作用已提交，系统必须显式转入 containment/handoff 状态。该要求必须通过控制面时延遥测与中断演练验证。
NFR10: 对 webhook 丢失、重复、乱序或部分处理失败引起的状态漂移，系统必须在 `15 分钟` 内通过 reconciliation 检出并恢复可解释状态；该要求必须通过故障注入、重放演练和滚动 `30` 天事件异常遥测验证。
NFR11: 活跃后台任务在执行期间不得出现超过 `5 分钟` 的无状态更新静默窗口，除非任务被明确标记为等待外部系统；该要求必须通过滚动 `30` 天 heartbeat telemetry 验证。
NFR12: `100%` 的受治理写操作必须在执行前经过结构化计划生成与策略评估；该要求必须通过写路径合约测试与滚动 `30` 天审计抽样证明。
NFR13: `100%` 的高风险动作必须满足以下之一：人工审批通过、明确策略允许自动执行、被策略阻止并留下拒绝记录；该要求必须通过滚动 `30` 天 approval/policy audit 验证。
NFR14: 默认生产授权模式不得依赖长期 `PAT` 作为必需前提；默认机器身份必须支持短时凭证和最小权限边界。该要求必须通过基线部署检查表和发布前安全评审证明 `100%` 正式部署形态均无需长期 `PAT` 作为默认执行前提。
NFR15: `100%` 的密钥、令牌、敏感配置和持久化凭证必须在传输中加密，在存储时受保护；该要求必须通过密钥管理检查表、传输安全测试和静态配置审计验证。
NFR16: 严重误操作事故目标为 `0`，其中包括未授权高风险写操作、错误修改受保护目标、未受控推进高风险发布或规则变更；该目标按滚动 `90` 天事故分级台账和事后复盘记录统计。
NFR17: `100%` 的权限提升尝试、策略绕过尝试、认证失败和紧急控制动作必须进入安全日志；该要求必须通过安全日志完整性抽样和故障注入测试验证。
NFR18: 默认模型上下文与日志中不得暴露明文 secrets、tokens 或被显式标记为敏感的受保护内容；相关拦截覆盖率必须达到 `100%`，并通过预发布 redaction suite、样本日志扫描和敏感数据路径检查证明。
NFR19: Phase 1 必须在单租户基线部署形态下，通过持续 `30` 分钟的容量与负载测试支持至少 `50` 个活跃受管仓库，同时保持 `NFR1` 与 `NFR5` 的时延目标不失效。
NFR20: Phase 1 必须支持单次 campaign 至少覆盖 `20` 个仓库，并保持逐仓库计划、状态、审批与结果可见性；该要求必须通过预发布 campaign acceptance run 和容量演练验证。
NFR21: Phase 1 必须支持单租户至少 `100` 个并发追踪任务，而不丢失任务状态、审计链或相关性标识；该要求必须通过持续 `30` 分钟并发压力测试和任务完整性校验验证。
NFR22: 当受管仓库规模从基线增长到 `10x` 时，系统不得要求改变外部命令、聊天或 structured plan 合约；增长应表现为容量扩展问题，而不是产品行为重定义问题。该要求必须通过基线与扩容场景的接口/行为回归测试验证。
NFR23: 机器可读输出、计划工件和集成接口负载必须采用版本化 schema，并通过针对前一兼容次版本消费者与生产者的合约测试证明在同一兼容版本线内保持向后兼容。
NFR24: `100%` 的 webhook 事件必须可通过稳定标识去重，并支持安全重放而不产生重复写副作用；该要求必须通过 replay/deduplication conformance suite 与故障注入测试证明。
NFR25: 外部集成故障不得导致任务进入不可解释状态；在故障注入测试中，`100%` 的相关失败都必须在 `30 秒` 内显式呈现为 retry、blocked、failed with handoff complete 或 equivalent governed state，并附带最近成功步骤、失败原因和下一恢复路径。
NFR26: 对正式支持的脚本化命令集，`100%` 的核心输出必须同时提供 `human-readable text` 与 `JSON/YAML` 两种表达形式；该要求必须通过 CLI 输出合约测试和文档化命令样本验证。
NFR27: Gitdex 输出的 structured plans、reports 和 handoff artifacts 必须可在 CLI、CI、IDE、agent runtime 和内部工具之间无损交换；对正式支持的 artifact 类型，round-trip 合约测试必须证明 `100%` 保留必填字段、标识符、状态语义与策略判定结果。
NFR28: `100%` 的受治理任务、计划、审批、策略判断、执行结果和安全相关事件必须带有可追踪的 correlation ID / task ID；该要求必须通过审计抽样、事件流完整性扫描和预发布 conformance suite 验证。
NFR29: `100%` 的受治理写操作必须保留完整审计链，至少包括触发来源、执行计划、策略判断、结果状态和关联证据；该要求必须通过滚动 `30` 天审计完整性扫描和写路径抽样检查验证。
NFR30: 操作者必须能在正常负载下按滚动 `7` 天遥测统计于 `P95 <= 5 秒` 内查到任一活跃任务的当前状态、最近一次状态转换和阻塞原因。
NFR31: 默认部署形态下，操作日志与审计记录的保留策略必须可配置，且默认保留期不得低于 `90 天`；该要求必须通过部署配置检查和保留策略验收测试验证。
NFR32: 对于每一次自治任务失败，系统必须输出最小证据包，且在失败演练样本中 `100%` 至少包含触发来源、作用范围、最近成功步骤、失败/阻塞步骤、相关对象标识、策略判断、关联证据和建议下一动作，以支持人工复盘、接管和责任界定。
NFR33: Phase 1 的核心闭环能力必须在 `Windows`、`Linux`、`macOS` 三端均可用，包括状态查看、计划生成、审批/拒绝、执行、接管与审计查询；该要求必须通过正式支持平台矩阵的端到端 conformance suite 验证。
NFR34: 若 rich TUI 不可用，text-only 终端模式仍必须完整支持 `100%` 的 Phase 1 核心 operator 流程，包括状态查看、计划审查、审批/拒绝、执行启动/暂停/取消、handoff 查看与审计查询，而不是退化为只读工具；该要求必须通过 text-only regression suite 验证。
NFR35: Gitdex 不得要求浏览器作为 Phase 1 核心维护闭环的唯一入口；所有核心运维动作必须可在终端内完成。该要求必须通过核心运维场景验收清单验证。
NFR36: 同一命令、聊天请求和结构化输出在支持的 shell 环境中必须保持一致的核心语义；对正式支持的跨 shell 基准用例集，合约测试必须证明 `100%` 保持相同的治理结果、计划结构、退出语义和审批需求，不因 PowerShell、bash、zsh 差异而改变。
NFR37: Phase 1 至少应为每个正式支持的操作系统提供一种原生 shell completion 或等价命令发现机制；该要求必须通过每个平台默认支持 shell 的安装与发现性验收清单验证。

### Additional Requirements

- Starter foundation must use `Go 1.26.1 + Cobra v1.10.2` as the CLI and daemon baseline, with `Bubble Tea v2` reserved for rich TUI implementation.
- Gitdex must ship as two coordinated entrypoints, `gitdex` and `gitdexd`, rather than a single foreground-only CLI.
- Default production machine identity must be `GitHub App`, and the implementation must not assume long-lived PATs as the baseline operating model.
- The durable system record must be `PostgreSQL`, storing tasks, plans, approvals, events, repo projections, campaigns, and audit records.
- All Git-side mutative execution must run in isolated `git worktree` environments rather than shared working trees.
- All write-capable flows must follow the explicit sequence `intent -> context -> structured_plan -> policy -> approval -> queue -> execute -> reconcile -> audit_close`.
- The system must enforce `single-writer-per-repo-ref` for mutative operations.
- GitHub ingestion must be `webhook-first`, with asynchronous processing, deduplication, fast acknowledgment, and polling reserved for reconciliation only.
- The product must implement `policy bundles`, `capability grants`, `mission windows`, and layered kill switches as first-class architectural objects.
- The system must maintain a `repo digital twin` / projection model spanning Git, GitHub collaboration objects, workflow state, deployment state, and current open tasks.
- The audit system must be append-only and must pair an `audit ledger` with an evidence store and handoff package generation.
- GitHub aggregate reads should be designed `GraphQL-first`, while GitHub mutations should be designed `REST-first`.
- The machine-facing surface must provide versioned contracts for CLI artifacts, API payloads, plans, reports, campaigns, and handoff packages.
- Deployment work in Phase 1 must be implemented as `deployment governance`, integrating existing CI/CD and environment protections rather than direct imperative cloud control.
- Rich TUI and text-only terminal mode must share the same task, plan, approval, evidence, and audit semantics.
- The implementation must support local workstation mode, single-tenant team mode, and future enterprise self-hosted mode without redefining the core operator model.

### UX Design Requirements

UX-DR1: Implement a terminal-native semantic design token system covering neutral structure colors plus action, focus, success, warning, and danger states, with identical semantic mapping across rich TUI, text-only mode, and exported assets.
UX-DR2: Implement the typography system using a monospace stack for terminal/TUI surfaces and a complementary sans-serif stack for exported HTML/report assets, with defined density tiers for titles, panels,正文, dense tables, and annotations.
UX-DR3: Implement responsive terminal layouts for compact (`0-99` columns), standard (`100-139` columns), wide (`140+` columns), and short-height (`<32` rows) environments without removing core operator capabilities.
UX-DR4: Implement the chosen `Calm Ops Cockpit` information architecture: cockpit-first rather than chat-first, with the main workspace focused on current object/plan/task, persistent risk/evidence inspector, and bottom-anchored unified input.
UX-DR5: Implement the dual-mode `Intent Composer` component that unifies command input, natural-language chat, structured parameter completion, history recall, validation, and keyboard submission.
UX-DR6: Implement the `Repo State Summary Board` component to present branch, diff, upstream, PR, issue, workflow, and deployment signals in a scan-friendly summary with explicit healthy/drifting/blocked/degraded/unknown states.
UX-DR7: Implement the `Structured Plan Card` component as the primary review/sign-off surface, showing title, scope, steps, risk, policy hits, approval requirements, reversibility, expected outcomes, and lifecycle state.
UX-DR8: Implement the `Policy Verdict Bar` component to explain why an action is allowed, escalated, blocked, or degraded, instead of only showing a status flag.
UX-DR9: Implement the `Evidence Drawer` pattern for in-context inspection of diffs, logs, policy basis, and related objects without leaving the current task flow.
UX-DR10: Implement the `Handoff Pack Viewer` component for takeover scenarios, showing trigger source, current status, completed steps, pending steps, risks, and suggested next actions.
UX-DR11: Implement the `Campaign Matrix` component for multi-repository work, including per-repo plan/state/approval/intervention views plus sorting, filtering, exclusion, retry, and export.
UX-DR12: Implement a consistent action hierarchy in all views: at most one primary action, clearly separated secondary and utility actions, and visually distinct destructive actions with confirmation behavior.
UX-DR13: Implement feedback patterns for success, warning, error, running, and blocked states that always show current state, last completed step, current blocker, and the next actionable path.
UX-DR14: Implement progressive parameter collection and form behavior so missing inputs are resolved inline, while high-risk parameters require explicit second confirmation.
UX-DR15: Implement navigation patterns that preserve operator orientation across task, repo, plan, evidence, and audit contexts, including command palette, object jump, recent-history navigation, and stable back navigation.
UX-DR16: Implement empty-state and loading-state behavior that gives actionable next steps; long-running operations must show stage-aware progress after `3s` and whether the operator can safely leave after `10s`.
UX-DR17: Ensure rich TUI and text-only mode are both fully keyboard operable, with visible focus, non-color-only status signaling, and linearly readable approval/blocking/error information.
UX-DR18: Ensure exported HTML assets and documentation-oriented reports meet `WCAG 2.1 AA` contrast and semantic structure expectations.
UX-DR19: Implement text-only terminal parity for all Phase 1 core operator flows, including state viewing, plan review, approval/rejection, execution start/pause/cancel, handoff viewing, and audit query.
UX-DR20: Build a UX regression matrix covering `80/100/140`-column terminal layouts, Windows Terminal/PowerShell, bash/zsh, macOS Terminal/iTerm, rich TUI vs text-only parity, and failure/handoff/approval scenario drills.

### FR Coverage Map

FR1: Epic 1 - explicit terminal command entry for Gitdex capabilities.
FR2: Epic 1 - natural-language terminal chat entry.
FR3: Epic 1 - seamless switching between command and chat within one task context.
FR4: Epic 1 - consolidated repository state visibility.
FR5: Epic 1 - explainable current-state, risk, and next-action summaries.
FR6: Epic 1 - inspectable evidence behind summaries and recommendations.
FR7: Epic 2 - compile intents into structured execution plans before governed writes.
FR8: Epic 2 - preview scope, actions, and risk before execution.
FR9: Epic 2 - approve, reject, edit, or defer plans.
FR10: Epic 2 - run tasks in observe/recommend/dry-run/execute modes.
FR11: Epic 2 - execute approved plans through explicit lifecycle states.
FR12: Epic 2 - explain policy allow/block/escalate decisions.
FR13: Epic 2 - preserve traceability between intent, plan, policy, execution, and evidence.
FR14: Epic 2 - inspect local working state, branches, and diffs.
FR15: Epic 2 - compare upstream and perform controlled synchronization.
FR16: Epic 2 - low-risk governed repository hygiene and maintenance.
FR17: Epic 2 - controlled local file modification within authorized scope.
FR18: Epic 4 - terminal views for issues, PRs, comments, workflows, and deployments.
FR19: Epic 4 - create, update, and respond to GitHub collaboration objects.
FR20: Epic 4 - triage, prioritize, and summarize inbound collaboration activity.
FR21: Epic 4 - coordinate branch/PR/issue/comment/workflow/deployment context in one tracked task.
FR22: Epic 4 - governed release and deployment decision preparation.
FR23: Epic 5 - define autonomy levels by capability and scope.
FR24: Epic 5 - continuous or scheduled monitoring for supported maintenance/governance scenarios.
FR25: Epic 5 - start governed tasks from events, schedules, API requests, or operator requests.
FR26: Epic 5 - pause, resume, cancel, and take over autonomous tasks.
FR27: Epic 5 - retry, reconciliation, quarantine, and safe handoff recovery paths.
FR28: Epic 5 - generate handoff packages for human continuation.
FR29: Epic 5 - persist long-running task state across sessions and background windows.
FR30: Epic 3 - bounded authorization at repository, installation, organization, or fleet scope.
FR31: Epic 3 - define approvals, risk tiers, protected targets, and execution boundaries.
FR32: Epic 3 - enforce policy consistently across all entry points.
FR33: Epic 3 - record complete audit trails for governed actions and security-relevant events.
FR34: Epic 3 - inspect audit history, evidence, and task lineage.
FR35: Epic 3 - trigger emergency controls such as pause, suspension, and kill switch.
FR36: Epic 3 - define data-handling rules by scope, retention, and sensitivity.
FR37: Epic 6 - define and run governed multi-repository campaigns.
FR38: Epic 6 - review per-repository plans, status, and outcomes inside campaigns.
FR39: Epic 6 - approve, exclude, or intervene on repositories within a campaign.
FR40: Epic 7 - submit structured intents, plans, or tasks through machine-facing interfaces.
FR41: Epic 7 - query task state, campaign state, reports, and audit-friendly outputs.
FR42: Epic 7 - exchange structured plans, results, and status with CI, IDE, agent runtimes, and internal tooling.
FR43: Epic 3 - apply shared policy bundles and governance defaults across repo groups.
FR44: Epic 1 - terminal-based initial setup for identity, permissions, and defaults.
FR45: Epic 1 - configure Gitdex through global, repository, session, and environment settings.
FR46: Epic 1 - human-readable and structured output selection.
FR47: Epic 1 - discover available capabilities, commands, and object actions.
FR48: Epic 1 - diagnose environment, authorization, configuration, and connectivity issues.
FR49: Epic 7 - export plans, reports, handoff packages, and structured artifacts for external reuse.
FR50: Epic 1 - preserve the same core operating model across Windows, Linux, and macOS.

## Epic List

### Epic 1: Terminal Onboarding, Identity, and Repository Visibility
Users can install Gitdex from the approved starter foundation, connect authorized repositories, complete terminal-first setup, and make Gitdex their default all-in-terminal visibility surface for repository state, risks, and evidence.
**FRs covered:** FR1, FR2, FR3, FR4, FR5, FR6, FR44, FR45, FR46, FR47, FR48, FR50

### Epic 2: Governed Planning and Safe Single-Repository Action
Users can turn a goal into a structured plan, review risk and policy outcomes, and execute safe single-repository work such as sync, hygiene, and controlled local modifications with explicit lifecycle tracking.
**FRs covered:** FR7, FR8, FR9, FR10, FR11, FR12, FR13, FR14, FR15, FR16, FR17

### Epic 3: Governance, Policy, Audit, and Emergency Control
Administrators and operators can authorize Gitdex safely, define execution boundaries, inspect audit lineage, and stop or constrain risky automation without breaking the operator experience.
**FRs covered:** FR30, FR31, FR32, FR33, FR34, FR35, FR36, FR43

### Epic 4: Terminal Collaboration and Release Coordination
Maintainers can work with issues, pull requests, comments, workflows, and release/deployment context directly in the terminal, with unified summaries, triage, and coordination.
**FRs covered:** FR18, FR19, FR20, FR21, FR22

### Epic 5: Background Autonomy, Recovery, and Human Takeover
Users can let Gitdex monitor and run approved autonomous work in the background, then pause, recover, or take over when tasks block, drift, or fail.
**FRs covered:** FR23, FR24, FR25, FR26, FR27, FR28, FR29

### Epic 6: Multi-Repository Campaign Operations
Operators can define, review, and control governed campaigns across multiple repositories with per-repository visibility, approval, exclusion, and intervention.
**FRs covered:** FR37, FR38, FR39

### Epic 7: Platform Integrations and Structured Exchange
Integration users can treat Gitdex as a machine-consumable control plane by submitting intents, querying state, and exchanging versioned plans, reports, and handoff artifacts with other systems.
**FRs covered:** FR40, FR41, FR42, FR49

## Epic 1: Terminal Onboarding, Identity, and Repository Visibility

Users can install Gitdex from the approved starter foundation, connect authorized repositories, complete terminal-first setup, and make Gitdex their default all-in-terminal visibility surface for repository state, risks, and evidence.

### Story 1.1: Set Up Initial Project from Starter Template (Architecture Starter Requirement)

As a platform engineer,
I want to initialize Gitdex from the approved starter foundation,
So that all later stories build on the agreed runtime, command tree, and repository structure.

**Acceptance Criteria:**

**Given** an empty Gitdex codebase
**When** the starter setup is executed
**Then** the repository contains the agreed Go workspace baseline with `gitdex` and `gitdexd` entrypoints, core folders, config scaffolding, schema folders, and migration placeholders
**And** the baseline build, test, and local run commands succeed on a supported developer machine
**And** shell completion and configuration loading hooks are wired for future stories

### Story 1.2: Run Terminal-First Setup and Environment Diagnostics (`FR44`, `FR45`, `FR48`)

As a new Gitdex user,
I want a terminal-based setup flow for identity, permissions, defaults, and diagnostics,
So that I can get to a working state without leaving the terminal or guessing what is misconfigured.

**Acceptance Criteria:**

**Given** a user launching Gitdex for the first time
**When** they run the setup flow
**Then** Gitdex guides them through identity configuration, default operating preferences, and repository/global config creation
**And** Gitdex validates connectivity, authorization, and required local tooling, reporting actionable fixes for any failure
**And** the resulting configuration can be reloaded and inspected from the terminal in later sessions

### Story 1.3: Use Dual-Mode Terminal Entry with Discoverable Commands (`FR1`, `FR2`, `FR3`, `FR46`, `FR47`)

As a repository operator,
I want to use both explicit commands and natural-language chat inside the same terminal session,
So that I can move between precision and exploration without switching tools or losing context.

**Acceptance Criteria:**

**Given** an active Gitdex terminal session
**When** the operator issues commands or natural-language requests
**Then** both modes run inside the same task context and can hand off into one another without losing scope
**And** Gitdex exposes discoverable command help, capability listing, and object actions from within the terminal
**And** supported outputs can be emitted as human-readable text and machine-readable JSON/YAML

### Story 1.4: Connect Authorized Repositories and View Consolidated State (`FR4`, `FR5`, `FR6`, `UX-DR6`)

As a maintainer,
I want to connect an authorized repository and immediately see a consolidated state summary with evidence,
So that I can understand current drift, risk, and next actions before taking any write action.

**Acceptance Criteria:**

**Given** a repository within the user's authorized scope
**When** the user opens that repository in Gitdex
**Then** Gitdex displays local Git state, remote divergence, collaboration signals, workflow state, and deployment status in one summary surface
**And** the summary highlights material risks and evidence-backed next actions
**And** the summary uses explicit healthy, drifting, blocked, degraded, or unknown state labels that the user can drill into for supporting objects and evidence

### Story 1.5: Operate the Cockpit in Rich TUI and Text-Only Modes (`FR50`, `UX-DR1`, `UX-DR2`, `UX-DR3`, `UX-DR4`, `UX-DR12`, `UX-DR15`, `UX-DR17`, `UX-DR19`, `UX-DR20`)

As a terminal-first operator,
I want the Gitdex cockpit to work in rich TUI and text-only modes with the same core semantics,
So that I can operate Gitdex reliably across machines, shells, and terminal capabilities.

**Acceptance Criteria:**

**Given** supported terminal environments across Windows, Linux, and macOS
**When** the operator opens Gitdex in rich TUI or text-only mode
**Then** both modes provide the same core operator flows for navigation, repository selection, state viewing, and next-action discovery
**And** all primary interactions are keyboard operable with visible focus and non-color-only status signaling
**And** the cockpit adapts to compact, standard, and wide terminal widths without hiding critical task or risk information
**And** terminal typography and density remain readable for titles, panels, dense tables, and annotations across the supported `80/100/140`-column regression matrix

## Epic 2: Governed Planning and Safe Single-Repository Action

Users can turn a goal into a structured plan, review risk and policy outcomes, and execute safe single-repository work such as sync, hygiene, and controlled local modifications with explicit lifecycle tracking.

### Story 2.1: Compile Structured Plans from Commands and Chat (`FR7`, `FR8`, `FR12`, `FR13`, `UX-DR5`, `UX-DR7`, `UX-DR8`, `UX-DR14`)

As a maintainer,
I want Gitdex to compile my command or natural-language request into a structured plan before any governed write occurs,
So that I can review scope, risk, and policy outcomes before execution begins.

**Acceptance Criteria:**

**Given** a write-capable request expressed through command or chat
**When** Gitdex prepares the request for execution
**Then** it produces a structured plan showing target scope, intended steps, risk level, and policy verdict
**And** the plan is stored with a stable identifier and traceable back to the originating request
**And** blocked or escalated outcomes are explained in operator-readable language rather than hidden behind generic errors
**And** missing parameters are collected inline while high-risk parameter changes require an explicit second confirmation before the plan can advance

### Story 2.2: Review, Approve, Reject, Edit, or Defer a Plan (`FR9`, `FR10`, `FR12`, `FR13`, `UX-DR9`, `UX-DR12`, `UX-DR13`)

As a repository operator,
I want to review and control a plan before it runs,
So that governed actions only proceed when I understand and accept the proposed change.

**Acceptance Criteria:**

**Given** a structured plan awaiting operator review
**When** the operator inspects the plan
**Then** they can approve, reject, edit, or defer it from within the terminal
**And** the review surface includes linked evidence, current blockers, and the next actionable path
**And** the review result is recorded as part of the task's traceable lifecycle

### Story 2.3: Execute Approved Tasks with Explicit Lifecycle Tracking (`FR10`, `FR11`, `FR13`, `UX-DR16`)

As a maintainer,
I want approved single-repository tasks to run through explicit lifecycle states,
So that I can see what is happening, what already completed, and where intervention would occur if the task fails.

**Acceptance Criteria:**

**Given** an approved task for a supported single-repository action
**When** execution begins
**Then** Gitdex moves the task through explicit lifecycle states from queueing to execution to reconciliation to terminal outcome
**And** each state transition is tied to a stable task identifier and correlation identifier
**And** operators can inspect the latest completed step and current executing step from the terminal
**And** long-running execution shows stage-aware progress after `3s` and whether the operator can safely leave after `10s`

### Story 2.4: Inspect Local Git State and Perform Controlled Upstream Sync (`FR14`, `FR15`)

As a maintainer,
I want to inspect my repository state and run controlled upstream synchronization workflows,
So that I do not have to manually diagnose divergence or hand-compose the safest sync path.

**Acceptance Criteria:**

**Given** a repository with local and remote branch state
**When** the operator asks Gitdex to inspect or sync with upstream
**Then** Gitdex presents branch state, diffs, divergence, and the recommended sync action before any write occurs
**And** supported sync actions run in a governed mode with previewable impact
**And** conflict or blocked scenarios are surfaced with a clear explanation and safe next step

### Story 2.5: Run Low-Risk Repository Hygiene Tasks (`FR16`)

As a maintainer,
I want Gitdex to perform low-risk repository hygiene tasks under governance,
So that repetitive maintenance work stops draining attention from actual project work.

**Acceptance Criteria:**

**Given** a repository that qualifies for supported low-risk maintenance
**When** the operator selects a hygiene action
**Then** Gitdex presents a plan and executes the maintenance task under the same governed lifecycle as other write operations
**And** the resulting changes and affected objects are summarized on completion
**And** failed hygiene runs preserve enough context for retry or handoff rather than silently stopping

### Story 2.6: Apply Controlled Local File Modifications in Isolated Worktrees (`FR17`)

As a maintainer,
I want Gitdex to make controlled local file modifications inside isolated execution worktrees,
So that repository edits stay reviewable, reversible, and separated from my active working tree.

**Acceptance Criteria:**

**Given** a supported file modification request within an authorized repository scope
**When** the operator approves the associated plan
**Then** Gitdex performs the modification inside an isolated worktree rather than the shared live working tree
**And** the operator can review the resulting diff before accepting downstream branch or PR actions
**And** failed or cancelled modifications can be discarded without corrupting the operator's active workspace

## Epic 3: Governance, Policy, Audit, and Emergency Control

Administrators and operators can authorize Gitdex safely, define execution boundaries, inspect audit lineage, and stop or constrain risky automation without breaking the operator experience.

### Story 3.1: Authorize Gitdex Identity and Scope Through GitHub App (`FR30`)

As an administrator,
I want to authorize Gitdex at repository, installation, organization, or fleet scope through the supported machine identity model,
So that automation operates with explicit boundaries instead of implicit long-lived user power.

**Acceptance Criteria:**

**Given** an administrator connecting Gitdex to GitHub
**When** they authorize Gitdex for use
**Then** Gitdex uses the supported GitHub App-based identity model and records the granted scope and capability boundary
**And** the granted scope is visible and reviewable from within the product
**And** the default authorization path does not require a long-lived PAT to operate

### Story 3.2: Configure Policy Bundles, Risk Tiers, and Execution Boundaries (`FR31`, `FR32`, `FR36`, `FR43`)

As an administrator,
I want to define policy bundles and execution boundaries for repositories and groups,
So that Gitdex can apply consistent governance without hard-coded or ad hoc decisions.

**Acceptance Criteria:**

**Given** one or more authorized repositories or repository groups
**When** an administrator configures Gitdex governance
**Then** they can define approval rules, risk tiers, protected targets, data-handling rules, and shared policy defaults
**And** those policy decisions apply consistently across command, chat, API, integration, and autonomous entry points
**And** policy changes are versioned and traceable for later review

### Story 3.3: Inspect Audit History, Evidence, and Task Lineage (`FR33`, `FR34`)

As an operator or administrator,
I want to inspect the full audit trail and lineage for governed actions,
So that I can explain who triggered what, under which policy, with which evidence, and with what result.

**Acceptance Criteria:**

**Given** a governed task or action in Gitdex
**When** an operator opens the audit and evidence view
**Then** Gitdex shows trigger source, plan, policy result, approvals, lifecycle history, outcome state, and linked evidence
**And** the operator can navigate from a task to related plans, reports, and handoff artifacts without leaving the terminal
**And** the audit history remains append-only and queryable over time

### Story 3.4: Trigger Emergency Controls and Containment (`FR35`)

As an authorized operator,
I want to pause, suspend, or kill risky automation quickly,
So that a bad task or bad scope cannot continue causing damage while the team regains control.

**Acceptance Criteria:**

**Given** an active task, repository scope, or wider authorized scope
**When** an authorized operator triggers an emergency control
**Then** Gitdex applies the corresponding pause, suspension, or kill-switch action and records the event in the security and audit trail
**And** affected tasks visibly enter a contained or blocked state instead of continuing silently
**And** operators can see what was stopped and what manual follow-up is still required

## Epic 4: Terminal Collaboration and Release Coordination

Maintainers can work with issues, pull requests, comments, workflows, and release/deployment context directly in the terminal, with unified summaries, triage, and coordination.

### Story 4.1: View GitHub Collaboration Objects in the Terminal (`FR18`)

As a maintainer,
I want to view issues, pull requests, comments, reviews, workflows, and deployment state in the terminal,
So that I can stay inside Gitdex instead of constantly bouncing to web pages.

**Acceptance Criteria:**

**Given** an authorized repository with active GitHub collaboration objects
**When** the maintainer opens the collaboration view in Gitdex
**Then** the terminal surfaces issues, PRs, reviews, workflows, and deployment state in a unified, navigable interface
**And** the operator can move from summary rows into detailed object views without losing repository context
**And** rich TUI and text-only mode expose the same object information hierarchy

### Story 4.2: Create, Update, and Respond to Collaboration Objects (`FR19`)

As a maintainer,
I want to create, update, and respond to supported GitHub collaboration objects from Gitdex,
So that collaboration work happens in the same governed environment as repository operations.

**Acceptance Criteria:**

**Given** an authorized repository and a supported issue, pull request, comment, or review action
**When** the maintainer submits that action through Gitdex
**Then** Gitdex executes it through the supported GitHub integration path and records the resulting object linkage in the current task context
**And** any policy or scope restriction is explained before the write occurs
**And** the completion summary includes the affected GitHub object references

### Story 4.3: Triage, Prioritize, and Summarize Inbound Activity (`FR20`)

As a maintainer,
I want Gitdex to summarize and triage incoming issue, PR, and comment activity,
So that I can focus on what needs attention first instead of manually scanning every update.

**Acceptance Criteria:**

**Given** a repository or campaign scope with incoming collaboration activity
**When** the maintainer asks Gitdex for triage or summary support
**Then** Gitdex returns a prioritized summary with explicit reasons, grouping, and actionable next steps
**And** the summary can be scoped to a repository or campaign boundary
**And** the operator can inspect the underlying objects behind any triage recommendation

### Story 4.4: Coordinate Cross-Object Task Context (`FR21`)

As a maintainer,
I want a single tracked task to carry branch, PR, issue, comment, workflow, and deployment context together,
So that complex work is understandable as one coherent operation instead of a pile of disconnected objects.

**Acceptance Criteria:**

**Given** a task that touches multiple repository and GitHub objects
**When** the operator inspects the task in Gitdex
**Then** the task view shows the linked branch, PR, issue, comment, workflow, and deployment objects as one coordinated context
**And** evidence navigation preserves those cross-links rather than forcing manual lookups
**And** new linked objects created during the task are added to the same task lineage

### Story 4.5: Prepare Release and Deployment Decisions with Approval-Aware Summaries (`FR22`)

As a release-oriented maintainer,
I want Gitdex to prepare release and deployment decisions through governed summaries and checks,
So that I can understand readiness and approval requirements before pushing a high-risk change forward.

**Acceptance Criteria:**

**Given** a repository with release or deployment-relevant changes
**When** the operator asks Gitdex to prepare a release or deployment decision
**Then** Gitdex summarizes readiness signals, relevant checks, current blockers, and approval requirements without directly bypassing deployment governance
**And** the resulting decision package links back to the underlying repository and workflow evidence
**And** blocked or escalated cases clearly identify which reviewer or control gate must be satisfied next

## Epic 5: Background Autonomy, Recovery, and Human Takeover

Users can let Gitdex monitor and run approved autonomous work in the background, then pause, recover, or take over when tasks block, drift, or fail.

### Story 5.1: Define Autonomy Levels for Supported Capabilities (`FR23`)

As a repository owner,
I want to define autonomy levels for supported Gitdex capabilities,
So that I can decide which classes of work remain advisory and which can run with bounded autonomy.

**Acceptance Criteria:**

**Given** a repository owner managing Gitdex behavior for a repository scope
**When** they configure autonomy settings
**Then** Gitdex allows them to assign supported capabilities to explicit autonomy levels and scopes
**And** those settings are visible to operators before tasks are launched
**And** autonomous behavior cannot exceed the configured scope and level

### Story 5.2: Monitor Authorized Repositories Continuously or on Schedule (`FR24`)

As a repository owner,
I want Gitdex to monitor authorized repositories in the background for supported maintenance and governance scenarios,
So that repetitive checking work can happen without constant manual attention.

**Acceptance Criteria:**

**Given** a repository with supported background monitoring enabled
**When** Gitdex runs continuously or on an approved schedule
**Then** it evaluates only the explicitly supported maintenance or governance scenarios for that scope
**And** the resulting observations are surfaced as tasks, summaries, or recommended next actions rather than invisible background state
**And** disabled or unauthorized scenarios are not executed

### Story 5.3: Start Governed Tasks from Events, Schedules, APIs, or Operators (`FR25`)

As a platform operator,
I want Gitdex to start governed tasks from repository events, schedules, API requests, or terminal operators,
So that all supported entry points feed the same controlled task pipeline.

**Acceptance Criteria:**

**Given** a supported webhook, schedule, API request, or operator action
**When** it triggers Gitdex work
**Then** Gitdex normalizes that trigger into the same governed task envelope used by manual execution
**And** webhook ingestion is asynchronous and replay-safe rather than long-running in the request path
**And** the originating trigger source remains visible in task lineage and audit history

### Story 5.4: Pause, Resume, Cancel, and Take Over Autonomous Tasks (`FR26`)

As an operator,
I want to pause, resume, cancel, or take over autonomous tasks without losing context,
So that I can intervene safely when background work needs human judgment.

**Acceptance Criteria:**

**Given** an active autonomous task
**When** the operator issues a pause, resume, cancel, or take-over action
**Then** Gitdex applies the requested control while preserving current task state, current evidence, and the latest completed step
**And** the task view clearly shows whether control has returned to the human or remains in autonomous mode
**And** cancelled or paused work does not disappear into an unexplained terminal state

### Story 5.5: Recover from Blocked, Failed, or Drifted Tasks (`FR27`)

As an operator,
I want blocked, failed, or drifted tasks to enter explicit recovery paths,
So that the system remains governable even when automation cannot finish cleanly.

**Acceptance Criteria:**

**Given** a task that fails, blocks, or diverges from expected state
**When** Gitdex evaluates the outcome
**Then** it moves the task into a supported retry, reconciliation, quarantine, or equivalent governed recovery state
**And** the operator can see the failure reason, latest successful step, and next recovery path
**And** recovery behavior preserves traceability rather than starting a new unlinked task

### Story 5.6: Generate Handoff Packages and Persist Long-Running Task State (`FR28`, `FR29`, `UX-DR10`)

As an operator taking over autonomous work,
I want Gitdex to preserve long-running task state and generate handoff packages,
So that background work can survive session boundaries and still be understandable when a human must continue it.

**Acceptance Criteria:**

**Given** a long-running or failed task that spans terminal sessions or requires human continuation
**When** the operator reopens the task or requests a handoff package
**Then** Gitdex restores the latest task state and generates a handoff package containing trigger source, scope, completed steps, blocked step, evidence, and recommended next actions
**And** the restored task state remains consistent between daemon state and operator-visible views
**And** the handoff view presents current status, pending steps, risks, and suggested next actions without requiring manual reconstruction of task history
**And** handoff artifacts can be exported for use outside the immediate terminal session

## Epic 6: Multi-Repository Campaign Operations

Operators can define, review, and control governed campaigns across multiple repositories with per-repository visibility, approval, exclusion, and intervention.

### Story 6.1: Define a Governed Multi-Repository Campaign (`FR37`)

As a campaign operator,
I want to define a governed campaign across an authorized repository set,
So that repeated multi-repository work becomes an explicit, reviewable control-plane action rather than a loose script.

**Acceptance Criteria:**

**Given** an operator with an authorized repository set
**When** they create a campaign in Gitdex
**Then** Gitdex stores the campaign intent, target repository set, and governing scope as a first-class object
**And** campaign creation respects repository authorization boundaries rather than silently broadening scope
**And** the operator can review the campaign definition before any per-repository work begins

### Story 6.2: Review Per-Repository Plans and Status in a Campaign Matrix (`FR38`, `UX-DR11`)

As a campaign operator,
I want to review per-repository plans, statuses, and outcomes inside one campaign view,
So that I can understand progress and risk without opening each repository one by one.

**Acceptance Criteria:**

**Given** a campaign with multiple target repositories
**When** the operator opens the campaign view
**Then** Gitdex presents a per-repository matrix of plans, current states, and outcomes
**And** the operator can sort, filter, and drill into each repository row without losing campaign context
**And** completed, blocked, and pending repositories remain visible side by side rather than collapsing into one aggregate status
**And** the matrix exposes per-repository exclusion, retry, and export actions from the same campaign surface

### Story 6.3: Approve, Exclude, and Intervene Per Repository Within a Campaign (`FR39`)

As a campaign operator,
I want to approve, exclude, retry, or take over individual repositories within a campaign,
So that one problematic repository does not force me to stop or rerun the entire fleet operation.

**Acceptance Criteria:**

**Given** a running or reviewable campaign
**When** the operator acts on an individual repository inside that campaign
**Then** Gitdex supports per-repository approval, exclusion, or intervention without invalidating the rest of the campaign
**And** the campaign summary reflects partial completion and exceptions explicitly
**And** repository-level interventions remain linked to the same campaign audit trail

## Epic 7: Platform Integrations and Structured Exchange

Integration users can treat Gitdex as a machine-consumable control plane by submitting intents, querying state, and exchanging versioned plans, reports, and handoff artifacts with other systems.

### Story 7.1: Submit Structured Intents, Plans, and Tasks Through a Machine API (`FR40`)

As an integration developer,
I want to submit structured intents, plans, or tasks to Gitdex through a machine-facing interface,
So that external systems can request governed work without simulating a human terminal session.

**Acceptance Criteria:**

**Given** an authorized integration client
**When** it submits a supported structured intent, plan, or task payload
**Then** Gitdex validates the payload against a versioned contract and creates the corresponding governed task
**And** invalid or out-of-scope payloads are rejected with explicit, machine-readable error responses
**And** the created task remains visible to terminal operators using the same identifiers and task lineage

### Story 7.2: Query Task, Campaign, and Audit-Friendly State Through the API (`FR41`)

As an integration developer,
I want to query task state, campaign state, and audit-friendly outputs from Gitdex,
So that downstream systems can monitor work and consume trustworthy operational status.

**Acceptance Criteria:**

**Given** an authorized integration client querying Gitdex
**When** it requests task, campaign, or audit-friendly state
**Then** Gitdex returns structured state with stable identifiers, explicit status semantics, and the latest relevant evidence references
**And** active, blocked, and completed states are distinguishable without inferring hidden internal behavior
**And** query responses align with the same state model shown in the terminal

### Story 7.3: Exchange Versioned Plans, Results, and Status with External Tooling (`FR42`)

As an integration developer,
I want Gitdex to exchange structured plans, results, and status with CI systems, IDEs, agent runtimes, and internal tooling,
So that Gitdex becomes the governed control plane rather than an isolated terminal application.

**Acceptance Criteria:**

**Given** a supported external tool integrating with Gitdex
**When** plans, results, or status are exchanged
**Then** Gitdex uses versioned schemas that preserve required fields, identifiers, and governance semantics across boundaries
**And** the integration can round-trip those artifacts without losing core task meaning
**And** Gitdex clearly identifies which contract version each payload conforms to

### Story 7.4: Export Plans, Reports, and Handoff Artifacts for External Reuse (`FR49`, `UX-DR18`)

As an operator or integration user,
I want to export plans, reports, handoff packages, and other structured artifacts,
So that Gitdex outputs can be reused in documentation, CI pipelines, IDE workflows, and follow-on automation.

**Acceptance Criteria:**

**Given** a plan, report, handoff package, or supported task artifact in Gitdex
**When** the operator or integration client requests export
**Then** Gitdex produces a structured export in the supported machine-readable form and a human-usable form where applicable
**And** exported artifacts retain stable identifiers and linked evidence references
**And** export does not require the consumer to scrape terminal text to recover the governed meaning
**And** exported HTML or documentation-oriented reports preserve semantic structure and accessible contrast suitable for `WCAG 2.1 AA` review

---

## Epic 8: 全面功能升级 — LLM 集成、仓库发现、完整 Git/GitHub 操作与自主巡航

在 TUI 骨架、面板合并、焦点导航、配置系统完成的基础上，进入功能全面升级阶段。本 Epic 涵盖五大核心能力：LLM 实时对话、剪贴板修复、仓库自动发现、完整仓库操作、LLM 自主巡航。

**FRs covered:** FR2, FR3, FR4, FR5, FR14, FR15, FR16, FR17, FR18, FR19, FR20, FR23, FR24, FR25, FR26

### Story 8.1: LLM 实时对话集成 (`FR2`, `FR3`)

As a Gitdex 用户,
I want 在 Chat 视图中输入自然语言即可获得 LLM 的实时流式回复,
So that 我可以通过对话获取建议、执行操作、理解仓库状态。

**Acceptance Criteria:**

**Given** 用户已在 Settings 中配置 LLM Provider/Model/API Key
**When** 用户在 Composer 中输入自然语言并提交
**Then** Chat 视图显示 LLM 的流式逐字回复
**And** 多轮对话保持上下文窗口自动管理
**And** 用户可通过 Esc/Ctrl+C 中断流式回复

### Story 8.2: 剪贴板与右键粘贴修复

As a Gitdex 用户,
I want 在终端中使用右键粘贴、Ctrl+V 粘贴、以及选择文本右键复制,
So that 我可以在 TUI 中自由地复制粘贴内容。

**Acceptance Criteria:**

**Given** 用户在终端中右键粘贴或 Ctrl+V
**When** 终端发送 paste 事件
**Then** Composer 正确接收粘贴内容（含多行和特殊字符）
**And** Content 区域的文本选择不被 TUI 事件拦截
**And** 行为在所有支持的终端中一致

### Story 8.3: 仓库自动发现与选择 (`FR4`, `FR5`)

As a GitHub 用户,
I want Gitdex 在配置认证后自动发现我的所有仓库并展示为类 gh-dash 的列表,
So that 我可以快速选择一个仓库进入，开始工作。

**Acceptance Criteria:**

**Given** 用户已配置 GitHub PAT 或 GitHub App 认证
**When** 进入 Dashboard
**Then** 自动抓取用户 GitHub 仓库列表，每条显示名称、星数、语言、状态、PR/Issue 数
**And** 仅远端仓库可选择克隆到本地或以只读模式进入
**And** 仓库列表支持搜索过滤

### Story 8.4: 完整仓库操作系统 (`FR14`-`FR19`)

As a 进入仓库后的操作者,
I want 在 TUI 中完成所有查看、文件编辑、Git 操作和 GitHub 协作操作,
So that 我不需要离开终端就能完成全部仓库维护工作。

**Acceptance Criteria:**

**Given** 用户已进入一个仓库上下文
**When** 执行查看/编辑/Git/GitHub 操作
**Then** 支持完整的工作树查看、PR/Issue/Commit 详情、文件 CRUD、全部 git 命令、GitHub PR/Issue/Review/Actions 操作
**And** 只读模式下禁用所有写操作

### Story 8.5: LLM 自主巡航系统 (`FR20`, `FR23`-`FR26`)

As a 仓库管理者,
I want Gitdex 能基于 LLM 进行 7×24 无人干预自主巡航，自动发现问题并执行维护,
So that 仓库能保持健康状态而无需我持续手动监控。

**Acceptance Criteria:**

**Given** 用户启用自主巡航模式
**When** 巡航引擎运行
**Then** LLM 自动扫描仓库状态，低风险操作自动执行，中高风险提交审批队列
**And** 安全护栏拦截危险操作（force push、删除 protected branch 等）
**And** 每周期生成巡航报告
**And** 用户可通过自然语言发起多步操作计划
