---
stepsCompleted: [1, 2, 3, 4, 5, 6]
inputDocuments:
  - ../../brainstorming/brainstorming-session-20260318-152000.md
  - ./domain-repository-autonomous-operations-research-2026-03-18.md
workflowType: "research"
lastStep: 6
research_type: "technical"
research_topic: "Gitdex architecture directions for repository autonomous operations"
research_goals: "确定 Git 执行层、GitHub API/GraphQL 层、事件驱动编排、作业调度、回滚、安全隔离、审计与 operator surface 的高质量架构方向，为 Product Brief、PRD 与 Architecture 提供技术基础"
user_name: "Chika Komari"
date: "2026-03-18"
web_research_enabled: true
source_verification: true
---

# Technical Research Report: Gitdex Architecture Directions

**Date:** 2026-03-18  
**Author:** Chika Komari  
**Research Type:** technical

---

## Research Overview

这份报告的目标不是给 Gitdex 选一个具体技术栈，而是先把架构方向定准。对 Gitdex 这种要长期运行、可修改本地文件、可操作 Git 仓库、可调用 GitHub API、可推进 PR / issue / comment / action / deployment 的自治系统来说，真正决定成败的不是“用哪一个 SDK”，而是以下几个设计是否正确：

- 是否把 `control plane`、`execution plane`、`trust plane`、`operator plane` 分开。
- 是否把 Git 变更、GitHub 变更、部署变更都建模为可审计、可补偿、可回放的操作。
- 是否默认采用 `GitHub App + short-lived credentials + webhook-first + async orchestration`。
- 是否把 AI / 自动化层放在“计划与建议”侧，而不是直接越过治理边界进入高权限执行。

基于前序 brainstorming 和 domain research，这里给出的结论是：Gitdex 不应该被实现成一个“全能终端 bot”，而应该被实现成一个**受治理的仓库控制平面**，终端只是它的 operator surface。

---

## Technical Research Scope Confirmation

**Research Topic:** Gitdex architecture directions for repository autonomous operations  
**Research Goals:** 确定 Git 执行层、GitHub API/GraphQL 层、事件驱动编排、作业调度、回滚、安全隔离、审计与 operator surface 的高质量架构方向。  
**Technical Research Scope:**

- Git execution layer: 多工作区、事务边界、工作副本隔离、冲突处理、回滚策略
- GitHub integration layer: GitHub App 身份、REST / GraphQL / webhook 分工、权限矩阵、速率限制
- Event-driven orchestration: 事件入口、状态机、幂等、重试、补偿、reconciliation
- Job scheduling: 并发控制、资源配额、租户隔离、优先级、长期任务管理
- Security isolation: 凭证、工作区、runner / container、网络、策略执行、人工审批
- Audit and observability: 审计流水、trace correlation、决策可追溯、handoff pack
- Operator surface: CLI / TUI / daemon 的职责分离
- Local reference project analysis: 从 `reference_project` 中提取可复用的模式，而不是直接复制实现

---

## Executive Summary

Gitdex 的正确技术方向，不是把 CLI、Git 命令、GitHub API 和 AI workflow 硬拼在一起，而是先定义一个有明确边界的自治控制系统。该系统必须把“做决定”和“执行副作用”拆开，把“本地 Git 事务”与“远程 GitHub / deployment 事务”拆开，把“高权限凭证”与“低权限观察流量”拆开。

从 GitHub 官方文档和 Git 官方文档看，几个方向已经很清楚。第一，长期自动化优先使用 `GitHub App`，因为它有细粒度权限、短时令牌、可跨仓库安装、内建 webhooks，并且适合跨仓库/跨组织的持续服务。第二，GitHub 官方同时明确建议 `webhook-first`、异步队列、最低权限、避免并发突发写请求、对 mutative requests 至少做 1 秒节流。第三，Git 侧最适合采用 `git worktree` 做隔离工作区，而不是在单 checkout 上复用和切分上下文。

因此，Gitdex 应采用以下总架构：`event ingress -> durable event log -> orchestrator -> typed job queue -> isolated executors -> audit ledger + trace store -> terminal operator surface`。在这个架构里，AI 或策略引擎只能生成**typed execution plan**，不能直接调用高权限执行器；所有真正会产生副作用的动作都要经过 policy engine、capability gate、rate-limit budget 和 approval gate。

**Key Technical Findings:**

- Gitdex 必须是 daemon-first 的控制平面，CLI / TUI 只是入口和观察面。
- Git 执行层应以 `git worktree` 为基本隔离单元，以“每个 job / branch / change-set 独立工作区”为默认模型。
- GitHub 集成层必须采用 `GitHub App` 为主身份，`REST + GraphQL` 组合使用，`webhook-first, poll-for-reconciliation-only`。
- 编排层必须是 durable orchestration，不应依赖单进程内存状态作为唯一事实来源。
- 回滚不能被理解成单一的 `git reset`；对远程副作用必须建模为 compensation。
- 质量优先时，deployment 不应先做“直接替你发版”，而应先做“受治理的 deployment orchestration”。

**Technical Recommendations:**

- 先建设 `trust plane`: policy-as-code、approval gates、audit ledger、handoff pack、kill switch。
- 把执行器拆成 `git executor`、`github write executor`、`deployment executor`，彼此不共享最高权限上下文。
- 采用 `single-writer-per-repo-ref` 与安装级速率预算，避免分布式条件竞争。
- 默认只让 Gitdex 通过现有 CI/CD 面发起 deployment，而不是直接持有长期云端 root 级凭证。
- 首个长期运行版本先做 single-tenant 或 installation-scoped 多租户，不要一开始做共享型高权限 SaaS 内核。

---

## Table of Contents

1. Technical Research Introduction and Methodology
2. System Constraints and Architecture Principles
3. Core Architecture Direction
4. Git Execution Layer
5. GitHub Integration Layer
6. Event-Driven Orchestration and Job Scheduling
7. Rollback, Reconciliation, and Recovery
8. Security Isolation and Trust Plane
9. Audit, Observability, and Operator Surface
10. Reference Project Pattern Extraction
11. Recommended Target Architecture for Gitdex
12. Phased Implementation Direction and Risk Assessment
13. Research Methodology, Confidence, and Limitations
14. Source Documentation
15. Research Conclusion

---

## 1. Technical Research Introduction and Methodology

### 1.1 Why This Research Matters

Gitdex 的难点不在于“能不能自动发一个 PR”，而在于“当系统 7x24 持续持有仓库与 GitHub 控制权时，怎样让它长期可靠、可控、可审计”。GitHub 官方关于 GitHub App、webhooks、REST / GraphQL rate limits、Actions token、OIDC、environments、audit log 的文档，实际上已经把这类系统的基础约束勾勒得很明确：最小权限、短时令牌、异步处理、低并发突发、事件驱动、审批保护、审计与流式观测。

对 Gitdex 来说，这意味着架构必须先围绕治理与恢复设计，再围绕自动化能力设计。

### 1.2 Research Methodology

- 使用 GitHub Docs、Git 官方文档、GitHub Blog Changelog 等一手来源确认当前平台约束。
- 使用本地 `reference_project` 中的优质开源项目提取架构模式，作为实现思路参考，不作为平台事实来源。
- 把 brainstorming 中的产品边界与 domain research 中的操作/权限/审计要求映射到技术设计。
- 对所有结论区分“平台明示事实”与“基于事实的架构推断”。

### 1.3 Goals Achieved

- 明确了 Gitdex 必须采用的高层系统分层。
- 明确了 Git 与 GitHub 两类副作用的边界和不同恢复模型。
- 明确了 GitHub App、webhook、REST / GraphQL、OIDC、environments 在 Gitdex 中的职责位置。
- 明确了终端产品形态下，CLI / TUI 不足以承担 7x24 托管职责，必须配套 daemon / service runtime。

---

## 2. System Constraints and Architecture Principles

### 2.1 External Platform Constraints

来自官方文档的几个关键约束直接决定了架构方向：

- GitHub App 比 OAuth app / PAT 更适合长期集成，因为它有细粒度权限、短时令牌、内建 webhooks、可独立于具体用户运行，并适合跨仓库 / 跨组织自动化。
- GitHub 明确建议优先使用 webhook，而不是持续轮询 API。
- GitHub REST 和 GraphQL 都有限流与二级限流；GraphQL 对 query cost、node 数、并发数都有约束，REST 明确建议避免并发写请求，并对 mutative requests 做节流。
- `GITHUB_TOKEN` 是作业级安装令牌，作业开始前生成，作业结束或达到最大生命周期后过期，因此不适合作为系统级长期身份。
- OIDC 的定位是让 workflow 按 job 获取短时云凭证，而不是把长期云密钥复制到 GitHub secrets。
- Environments 的定位是 deployment protection rule 和 required reviewers，这意味着 deployment 相关的高风险动作应挂在 protection gate 后面，而不是直接执行。

### 2.2 Core Architecture Principles

基于这些约束，Gitdex 应遵守以下原则：

- **Control plane first**: 先定义资源、状态、策略、审计，再定义自动化动作。
- **Typed effects**: 所有外部副作用必须变成显式类型化操作，而不是让 agent 直接“跑命令”。
- **Webhook-first**: webhook 负责实时触发，polling 只负责 reconciliation。
- **Single source of truth**: 调度状态不能散落在 worker 与终端会话里。
- **Compensation over fantasy rollback**: 远程副作用只能补偿，不能假设总能真正回滚。
- **Tenant and installation isolation**: 租户边界至少要以 GitHub App installation 为单位。

---

## 3. Core Architecture Direction

### 3.1 Gitdex Is Not Just a Terminal Tool

虽然 Gitdex 以终端为产品入口，但如果目标是“7x24 托管无需人工干预”，它本质上必须是一个长期运行的控制平面。也就是说：

- CLI 负责一次性命令、配置、手动触发、检查状态。
- TUI 负责 operator console、审批、故障检查、追踪与重放。
- Daemon / service runtime 负责 webhook 消费、调度、执行、reconciliation、告警、审计。

结论：**产品形态可以是 terminal-first，但系统形态必须是 service-first。**

### 3.2 Recommended Planes

推荐从一开始就按四个平面拆：

- **Control plane**: 租户、仓库、安装、策略、状态机、速率预算、审批规则、审计索引。
- **Execution plane**: Git executor、GitHub write executor、deployment executor、artifact collector。
- **Trust plane**: 身份、凭证、策略执行、审批、隔离、kill switch。
- **Operator plane**: CLI / TUI、状态查询、diff 审阅、审批、回放、诊断。

### 3.3 Recommended Domain Model

建议核心实体至少包括：

- `Tenant`
- `Installation`
- `RepositoryTarget`
- `EventEnvelope`
- `Intent`
- `ExecutionPlan`
- `Job`
- `Operation`
- `ApprovalCheckpoint`
- `RollbackCheckpoint`
- `AuditRecord`
- `HandoffPack`

其中 `ExecutionPlan` 必须是显式结构化对象，而不是自由文本。AI 可以提出 plan，但不能直接持有最终执行权。

---

## 4. Git Execution Layer

### 4.1 Worktree-Centric Isolation

Git 官方文档明确指出，一个仓库可以附着多个 working trees，从而同时 checkout 多个分支。对 Gitdex 来说，`git worktree` 是最合适的执行隔离基础，而不是在单工作目录里频繁切分上下文。

推荐方向：

- 每个 repo/ref/change-set/job 使用独立 worktree。
- 对短生命周期实验或验证任务，可使用 detached HEAD worktree。
- 对可提交型任务，使用独立 topic branch worktree。
- 任何 worktree 都必须能被确定性重建，不能依赖人工历史状态。

### 4.2 Git Operation Classes

Git 执行层应把操作分成三类，而不是统一包装成 shell：

- **Read-only operations**: status, diff, blame, log, grep, merge-base
- **Workspace mutations**: file edits, add, restore, checkout/switch, stash-like snapshots
- **History mutations**: commit, cherry-pick, rebase, merge, revert, ref updates

每类操作需要不同的 policy、审批和恢复语义。特别是 history mutation 不应与普通文件改动走同一条 auto-approve 路径。

### 4.3 Transaction Envelope for Local Git

Git 本身不是数据库事务系统，因此 Gitdex 需要自己定义 `transaction envelope`：

1. 创建或复用隔离 worktree
2. 记录 `pre-op` 快照
3. 执行类型化操作序列
4. 记录 `post-op` 快照与 diff stats
5. 运行 policy checks / tests
6. 进入提交、PR、或丢弃流程

`pre-op` 快照建议至少记录：

- HEAD commit SHA
- 当前 branch / detached state
- index 状态
- untracked file inventory
- worktree path
- repo config fingerprint

### 4.4 Rollback Strategy for Local Git

Git 官方文档表明，`reflog` 记录了本地引用如何变化，因此适合做本地 ref 恢复锚点；但这并不意味着 Gitdex 可以把所有回滚都简化成 `reset --hard`。

推荐三层恢复：

- **Layer 1: disposable workspace rollback**
  - 对未发布的变更，直接销毁 worktree 并重建。
- **Layer 2: ref rollback**
  - 对本地分支引用变化，利用 checkpoint ref 或 reflog 回到已知安全点。
- **Layer 3: semantic rollback**
  - 对已经发布为 PR、merge 或 deployment 的动作，必须用补偿操作而不是假装“撤销所有历史”。

质量优先时，Gitdex 不应默认在共享工作副本上执行破坏性重置。

### 4.5 Concurrency Model for Git

同一 `repository + ref` 上必须实行 `single writer`。否则会出现：

- 多 job 争抢同一 branch
- 自动 rebase / merge 交叉覆盖
- 同一工作副本被不同 agent 污染

因此调度层至少要有两个锁域：

- repo-level read/write lock
- repo-ref single-writer lease

---

## 5. GitHub Integration Layer

### 5.1 Identity: GitHub App First

GitHub 官方明确建议：如果你要访问组织资源、构建长期集成、跨仓库运行、需要高于单个 workflow 的执行时间或权限，应该优先构建 GitHub App。GitHub App 还具备更细粒度权限、短时令牌、内建集中 webhooks，以及独立于具体用户持续运行的能力。

因此 Gitdex 的主身份应是：

- **Primary identity**: GitHub App installation token
- **Secondary identity**: GitHub App user token，仅用于“代表某个用户执行”的场景
- **Avoid by default**: long-lived PAT

### 5.2 REST and GraphQL Split

GitHub 官方文档明确指出，你不需要只用一个 API；GraphQL 适合聚合读取，REST 适合很多标准 HTTP 风格操作，而且有些功能只在某一侧提供。

推荐分工：

- **GraphQL**
  - issue / PR / discussion / project 视图聚合
  - 批量读取状态
  - 关联对象解析
  - 低请求数读取 dashboard / TUI 所需的密集上下文
- **REST**
  - 大多数 mutative operations
  - workflow runs / deployments / checks / comments / labels 等明确端点
  - endpoint-specific capability control
  - 权限头与错误处理更直接

结论：Gitdex 应构建一个**GitHub capability layer**，对上提供统一领域动作，对下根据动作选择 REST 或 GraphQL。

### 5.3 Webhook Ingress Model

GitHub 官方 webhooks 最佳实践非常明确：

- 只订阅最少事件
- 使用 webhook secret
- 使用 HTTPS
- 10 秒内返回 2XX
- 后台异步处理 payload
- 利用 `X-GitHub-Delivery` 做唯一性与防重放
- 服务恢复后 redeliver missed deliveries

这意味着 Gitdex 的 webhook 入口必须设计成：

- 快速验签
- 写入 durable event log
- 立刻 ack
- 后台异步消费
- delivery-id 去重

任何在 webhook handler 里直接跑长任务的实现，质量上都不够。

### 5.4 Permission Matrix as Code

GitHub 官方的 GitHub App 权限文档已经把 repo、issues、pull requests、deployments、environments、workflows、webhooks、custom properties 等权限域列得很细。Gitdex 不应该把这些权限隐含在代码里，而应把权限矩阵显式化：

- capability -> required GitHub permission
- capability -> required installation scope
- capability -> approval policy
- capability -> logging level

这实际上是 Gitdex `trust plane` 的一部分。

### 5.5 Rate Limit and Budgeting

GitHub GraphQL 文档指出：

- 安装令牌按 installation 计点数预算
- GraphQL 存在 query cost、node limit、secondary limit
- 不应并发猛打 API
- mutative requests 应避免高频突发

GitHub REST 最佳实践也明确要求：

- 尽量不要 polling
- 避免 concurrent requests
- 对 `POST/PATCH/PUT/DELETE` 至少间隔 1 秒
- 依据 `Retry-After` 与 `x-ratelimit-reset` 退避重试

因此 Gitdex 需要：

- installation-scoped rate budget
- endpoint family budget
- mutation queue
- backoff scheduler
- conditional GET / caching for read-heavy operator surfaces

---

## 6. Event-Driven Orchestration and Job Scheduling

### 6.1 Durable Orchestration, Not In-Memory Automation

`reference_project/symphony/SPEC.md` 的价值在于，它清楚地展示了一个高质量自治执行器应具备的几个模式：单一调度权威、bounded concurrency、per-issue workspace、retry queue、reconciliation、operator-visible observability。Gitdex 应吸收这些模式，但做得更完整。

推荐编排模型：

- webhook / schedule / manual trigger 统一转成 `EventEnvelope`
- 先入 durable log，再做 classification
- 产出 typed `Intent`
- plan compiler 把 intent 转成 `ExecutionPlan`
- orchestrator 把 plan 拆成可调度 `Job`
- job dispatcher 把 job 送入 effect-specific queues
- reconciler 周期性对账并修复状态漂移

### 6.2 Recommended Job State Machine

建议最小状态机：

- `received`
- `deduplicated`
- `classified`
- `planned`
- `awaiting_policy`
- `awaiting_approval`
- `runnable`
- `executing`
- `reconciling`
- `succeeded`
- `compensating`
- `failed`
- `quarantined`
- `cancelled`

这样做的价值是：失败、审批、补偿、人工接管都能有明确状态，不会被“任务失败”一个状态吞掉。

### 6.3 Scheduling Strategy

调度器建议至少支持以下维度：

- tenant / installation 级配额
- repo 级并发上限
- repo-ref single writer
- effect domain queue
  - read-only
  - repo-write
  - github-write
  - deployment-write
- priority lane
  - security / incident
  - governance hygiene
  - maintenance
  - batch / background

### 6.4 Reconciliation and Sweepers

质量优先时，系统不能假设“worker 说成功就真的成功”。必须有独立的 reconciler：

- 检查 job 记录与 GitHub 实际状态是否一致
- 检查 branch / PR / issue / deployment 是否处于期望状态
- 检查 rate-limited / network-partition / partial write 后的中间态
- 触发补偿、重试或 quarantine

这也是 `poll-for-reconciliation-only` 的正确位置。

### 6.5 Idempotency Model

每个 job 都必须带幂等键。推荐：

- webhook-triggered: `installation_id + delivery_id`
- schedule-triggered: `schedule_name + period_start + target`
- manual-triggered: `user + command + target + nonce`
- execution-step: `plan_id + operation_seq`

没有幂等键，redelivery 和 retry 会变成重复写入灾难。

---

## 7. Rollback, Reconciliation, and Recovery

### 7.1 Three Different Failure Domains

Gitdex 至少要区分三种失败域：

- **Local Git failure**
  - merge conflict
  - dirty worktree
  - hook / test failure
  - rebase failure
- **GitHub side-effect failure**
  - PR created but labels/comment failed
  - issue updated but workflow dispatch failed
  - secondary rate limit after partial sequence
- **Deployment / external system failure**
  - deployment requested but environment approval pending
  - workflow run started but downstream cloud access denied
  - rollout partial success

三种失败域恢复方式完全不同，不能共用一个“rollback”按钮。

### 7.2 Compensation Model for Remote Side Effects

远程动作推荐按补偿建模：

- PR creation -> close PR / add diagnostic comment / supersede with replacement PR
- label/comment/write failure -> retry or mark partial completion
- deployment request -> cancel run / mark rollback requested / open incident issue
- merge / release -> create revert PR, not force-reset shared remote history

Gitdex 应存储 `compensation plan`，并在高风险动作执行前就算好。

### 7.3 Handoff Pack

一旦进入 `awaiting_approval`、`failed` 或 `quarantined`，系统必须生成 `handoff pack`，至少包含：

- triggering event
- intended plan
- executed steps
- diffs / artifacts
- current repo / PR / deployment state
- error taxonomy
- recommended next actions

这既是人工接管机制，也是审计材料。

---

## 8. Security Isolation and Trust Plane

### 8.1 Credentials and Identity Boundaries

GitHub 官方关于 GitHub App、`GITHUB_TOKEN`、OIDC 的文档实际上给出了很清晰的身份层次：

- `GITHUB_TOKEN` 是 job 级、仓库级、短生命周期令牌
- GitHub App installation token 适合长期自动化
- OIDC 适合按 job 获取短时云端访问令牌

Gitdex 应据此设计：

- 平台主身份使用 GitHub App installation token
- 云端访问通过 OIDC 或等价短时 broker 获取
- 不把长期云密钥复制进 repo secrets 作为默认路径
- 用户代表性动作只用 user token，不升级为安装令牌替用户做越权动作

### 8.2 Execution Isolation

本地或自托管执行器必须至少隔离以下内容：

- filesystem workspace
- process namespace / container
- environment variables
- network egress
- temp directories
- tool allowlist

对 Gitdex 这种会修改仓库和调用外部系统的产品，推荐默认模型是：

- 每个 job 在独立 worktree 中运行
- 每个高权限 job 在独立容器或轻量 VM 中运行
- 凭证通过 sidecar / broker 注入，不落盘或最小化落盘
- 任务结束立即清理令牌和临时介质

### 8.3 Policy-as-Code Before Actuation

Gitdex 不能把安全策略写死在 prompt 或零散代码里。推荐从一开始就有 policy engine，输入为：

- capability
- target repo metadata
- branch / environment
- actor origin
- autonomy level
- diff characteristics
- requested permissions

输出为：

- allow
- deny
- require approval
- require environment gate
- require human ownership

### 8.4 Deployment Isolation

deployment 是最高风险 effect domain。质量优先时，Gitdex 不应先做“自己拿着云权限直接发布”。更好的方向是：

- Gitdex 负责创建/推进 deployment intent
- 具体发布由现有 CI/CD 或 GitHub Actions 执行
- GitHub environment protection rules 与 required reviewers 拦住高风险环境
- 云凭证通过 OIDC 按 job 获取

这会让 Gitdex 成为 deployment governance plane，而不是一开始就变成另一个自建 CD 引擎。

---

## 9. Audit, Observability, and Operator Surface

### 9.1 Audit Requirements for Gitdex

GitHub enterprise audit log 文档表明，企业会关注 actor、resource、action、time、IP、token-related attribution，以及事件流式导出。Gitdex 自己也必须达到类似可追溯度。

推荐审计字段：

- correlation_id
- installation_id / tenant_id
- repository / branch / environment
- triggering event identity
- effective credential type
- plan hash
- operation sequence
- before / after refs
- GitHub request ids / delivery ids
- approval record
- compensation record

### 9.2 Observability Model

参考 `openai-agents-python` 的 sessions / tracing / guardrails 语义，以及 `symphony` 的结构化运行态设计，Gitdex 需要：

- structured logs
- trace spans across event -> plan -> execution -> reconciliation
- per-job timeline
- token / rate-limit budget telemetry
- queue depth and lag
- policy deny / approval wait metrics
- operator-visible status snapshots

### 9.3 Operator Surface Design Direction

`gh-dash`、`octo.nvim`、`lazygit` 说明了三个重要事实：

- 终端里的高密度状态面板是可行的。
- 用户需要围绕 PR / issue / workflow / branch 做聚合操作，而不是一条条命令记忆。
- 高风险 Git 操作的价值不只在“能执行”，还在“能看懂当前状态并安全撤回”。

因此 Gitdex 的 TUI 不应只是日志窗，而应包含：

- queue / job dashboard
- repo / PR / issue / deployment targets
- diff and plan preview
- approval inbox
- retry / quarantine / replay controls
- rate budget and policy denial diagnostics

同时要吸收 `symphony` 的一个正确原则：**operator surface 不能成为系统正确性的前提**。也就是 UI 坏了，调度和审计仍然正确。

---

## 10. Reference Project Pattern Extraction

### 10.1 Symphony

从 `reference_project/symphony/SPEC.md` 可提取的高价值模式：

- 单一调度权威
- bounded concurrency
- per-unit isolated workspace
- retry queue + reconciliation
- structured observability
- dashboard 只是观察面，不是正确性来源

对 Gitdex 的启发：编排层应该有自己的 durable state 和明确状态机。

### 10.2 Lazygit

从 `reference_project/lazygit/README.md` 可提取的模式：

- worktree 是一等操作对象
- reflog-based undo 是重要安全网
- rebase / bisect / graph 需要高可见性与可撤销感知

对 Gitdex 的启发：Git 层不能只暴露“执行命令”，还必须暴露“可理解的事务与撤回面”。

### 10.3 gh-dash and octo.nvim

从 `reference_project/gh-dash/README.md` 和 `reference_project/octo.nvim/README.md` 可提取的模式：

- PR / issue / notification / workflow run 是 operator 视角的一级对象
- 用户需要 per-repo 自定义视图和 action
- GitHub surface 的高频操作应该被聚合为 terminal workflow

对 Gitdex 的启发：operator plane 应围绕 GitHub 工作对象组织，而不是围绕底层 API endpoint 组织。

### 10.4 openai-agents-python

从 `reference_project/openai-agents-python/README.md` 可提取的模式：

- guardrails
- handoffs
- sessions
- tracing

对 Gitdex 的启发：AI 层应是“有 guardrails 的计划系统”，不是直接接入高权限 actuator 的自治黑箱。

### 10.5 How to Use These References Correctly

这些参考项目的正确用法是：

- 学模式，不抄边界
- 学职责分离，不照搬交互
- 学失败处理，不只学 happy path

---

## 11. Recommended Target Architecture for Gitdex

### 11.1 Textual Architecture

```text
GitHub Webhooks / Schedules / Manual CLI
    -> Event Ingress
    -> Signature Verification + Deduplication
    -> Durable Event Log
    -> Intent Classifier / Plan Compiler
    -> Policy Engine + Approval Gate
    -> Orchestrator
    -> Typed Job Queues
       -> Git Executor
       -> GitHub Write Executor
       -> Deployment Executor
       -> Artifact / Evidence Collector
    -> Reconciler + Compensation Engine
    -> Audit Ledger + Trace Store + Metrics
    -> CLI / TUI Operator Surface
```

### 11.2 Key Components

- **Ingress Service**
  - webhook 验签、delivery 去重、快速 ack
- **Event Log**
  - durable append-only event storage
- **Plan Compiler**
  - 将 intent 转成 typed execution plan
- **Policy Engine**
  - capability, repo metadata, environment, autonomy level 检查
- **Approval Service**
  - 人工审批、环境 gate、暂停 / 恢复
- **Orchestrator**
  - 状态机、调度、budget、lease、retry
- **Executors**
  - Git executor
  - GitHub write executor
  - deployment executor
- **Reconciler**
  - 对账、修复、补偿、quarantine
- **Audit / Trace Layer**
  - 全链路审计与观察
- **Operator Surface**
  - CLI / TUI / API

### 11.3 Architecture Style

推荐采用：

- **Ports and Adapters / Hexagonal Architecture**
  - 上层领域模型不直接依赖 GitHub 或 Git CLI
- **Event-sourced tendencies, not mandatory full event sourcing**
  - 至少事件要 append-only，可重放
- **Capability-based execution**
  - 执行器只暴露受限动作，不暴露任意 shell

### 11.4 Multi-Tenant Direction

质量优先时建议的租户演进顺序：

1. single-tenant self-hosted
2. installation-scoped multi-tenant
3. enterprise-scoped managed control plane

不建议一开始做共享式高权限 SaaS 内核，因为审计、隔离、secret boundary 和 noisy neighbor 风险会显著放大。

---

## 12. Phased Implementation Direction and Risk Assessment

### 12.1 Recommended Build Order

**Phase 0: Trust Plane Foundation**

- policy-as-code
- audit ledger
- rate budget
- approval gate
- kill switch

**Phase 1: Git Read / Safe Write Plane**

- worktree isolation
- typed Git operations
- disposable workspace rollback
- diff / evidence collection

**Phase 2: GitHub Governance Plane**

- GitHub App identity
- webhook ingress
- issue / PR / comment orchestration
- mutation queue + reconciliation

**Phase 3: Repo Maintenance Automation**

- hygiene campaigns
- branch / rules / metadata orchestration
- bulk safe operations with tenant budgets

**Phase 4: Deployment Governance**

- environment-aware deployment intent
- approval and OIDC integration
- deployment status tracking and compensation

### 12.2 Primary Technical Risks

- 把 AI plan 与高权限 actuation 混在一起
- 低估 GitHub secondary rate limits 与 mutation sequencing
- 在共享工作副本里做高风险 Git 事务
- 认为 deployment rollback 总能自动化完成
- 让 TUI / UI 变成系统正确性的依赖
- 一开始就做复杂多租户而忽视 installation isolation

### 12.3 Recommended Explicit Non-Goals for Early Versions

- 任意 shell 自由执行
- 默认无人审批的 production deployment
- 在共享远程分支上自动 force-push 历史改写
- 依赖 PAT 作为主身份模型
- 无 durable audit 的全自治 agent 模式

---

## 13. Research Methodology, Confidence, and Limitations

### Confidence Assessment

- **High confidence**
  - GitHub App first
  - webhook-first, queue-backed ingestion
  - worktree-centric Git isolation
  - durable orchestration + reconciliation
  - OIDC and environment gates for deployment governance

- **Medium confidence**
  - 最优的 durable state 存储形态
  - first production topology 是单进程 modular monolith 还是 service split
  - multi-tenant 演进的具体时间点

- **Lower confidence**
  - 企业客户对“全自治 deployment”开放程度
  - 哪些 effect domains 能在早期版本安全默认自动批准

### Research Limitations

- 这份报告刻意不做具体库/框架选择，因此不会回答“用 Temporal 还是自己做队列”这类实现题。
- 部分编排与恢复策略属于基于官方约束和参考项目模式的架构推断，而非 GitHub 官方直接规定。
- `reference_project` 提供的是优秀模式样本，不是 Gitdex 的正式依赖或标准答案。

### Research Quality Assurance

- 关键平台事实均来自 GitHub Docs、Git 官方文档、GitHub Blog 等一手来源。
- 参考项目只用于模式抽取，不用于平台事实证明。
- 结论已与前序 brainstorming 和 domain research 的产品边界、风险模型、权限模型保持一致。

---

## 14. Source Documentation

### Primary Sources

1. GitHub Docs, "Deciding when to build a GitHub App"  
   https://docs.github.com/en/apps/creating-github-apps/about-creating-github-apps/deciding-when-to-build-a-github-app

2. GitHub Docs, "Best practices for creating a GitHub App"  
   https://docs.github.com/en/apps/creating-github-apps/about-creating-github-apps/best-practices-for-creating-a-github-app

3. GitHub Docs, "Permissions required for GitHub Apps"  
   https://docs.github.com/en/rest/authentication/permissions-required-for-github-apps

4. GitHub Docs, "Comparing GitHub's REST API and GraphQL API"  
   https://docs.github.com/en/rest/about-the-rest-api/comparing-githubs-rest-api-and-graphql-api

5. GitHub Docs, "Rate limits and query limits for the GraphQL API"  
   https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api

6. GitHub Docs, "Best practices for using the REST API"  
   https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api

7. GitHub Docs, "Best practices for using webhooks"  
   https://docs.github.com/en/webhooks/using-webhooks/best-practices-for-using-webhooks

8. GitHub Docs, "GITHUB_TOKEN"  
   https://docs.github.com/en/actions/concepts/security/github_token

9. GitHub Docs, "OpenID Connect"  
   https://docs.github.com/en/actions/concepts/security/openid-connect

10. GitHub Docs, "Managing environments for deployment"  
    https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments

11. GitHub Docs, "Using the audit log for your enterprise"  
    https://docs.github.com/en/enterprise-cloud@latest/enterprise-onboarding/govern-people-and-repositories/using-the-audit-log-for-your-enterprise

12. GitHub Docs, "Creating custom properties for repositories in your enterprise"  
    https://docs.github.com/en/enterprise-cloud@latest/enterprise-onboarding/govern-people-and-repositories/create-custom-properties

13. GitHub Docs, "Secure use reference"  
    https://docs.github.com/actions/security-guides/security-hardening-for-github-actions

14. GitHub Docs, "Self-hosted runners"  
    https://docs.github.com/en/actions/concepts/runners/self-hosted-runners

15. GitHub Docs, "Runner scale sets"  
    https://docs.github.com/en/actions/concepts/runners/runner-scale-sets

16. GitHub Docs, "Actions Runner Controller"  
    https://docs.github.com/en/actions/concepts/runners/actions-runner-controller

17. Git official documentation, "git-worktree"  
    https://git-scm.com/docs/git-worktree

18. Git official documentation, "git-reflog"  
    https://git-scm.com/docs/git-reflog

19. Git official documentation, "git-reset"  
    https://git-scm.com/docs/git-reset

20. Git official documentation, "git-restore"  
    https://git-scm.com/docs/git-restore

21. GitHub Blog Changelog, "Enterprise-owned GitHub Apps are now generally available"  
    https://github.blog/changelog/2025-03-10-enterprise-owned-github-apps-are-now-generally-available/

22. GitHub Resources, "What is platform engineering?"  
    https://github.com/resources/articles/what-is-platform-engineering

### Local Reference Projects

1. `reference_project/symphony/SPEC.md`
2. `reference_project/lazygit/README.md`
3. `reference_project/gh-dash/README.md`
4. `reference_project/octo.nvim/README.md`
5. `reference_project/openai-agents-python/README.md`

### Search Queries Used

- `site:docs.github.com GitHub App best practices`
- `site:docs.github.com permissions required for GitHub Apps`
- `site:docs.github.com comparing GitHub REST API and GraphQL API`
- `site:docs.github.com GraphQL rate limits and query limits`
- `site:docs.github.com best practices for using webhooks`
- `site:docs.github.com best practices for using the REST API`
- `site:docs.github.com GITHUB_TOKEN GitHub Actions`
- `site:docs.github.com OpenID Connect GitHub Actions`
- `site:docs.github.com manage environments GitHub Actions required reviewers`
- `site:git-scm.com git worktree documentation`
- `site:git-scm.com git reflog documentation`

---

## 15. Research Conclusion

Gitdex 的高质量技术方向已经足够清楚：它应该被实现成一个**terminal-first operator experience + daemon-first governed control plane**。Git 侧以 `worktree` 为隔离基础，GitHub 侧以 `GitHub App` 为主身份，系统编排以 `webhook-first + durable state + reconciliation` 为基本模式，deployment 侧以 `OIDC + environment protection + approval` 为治理基础。

最关键的架构判断有两条。第一，Gitdex 不能让 AI 或脚本直接越过治理边界进入高权限执行器；所有副作用都必须经过 typed plan、policy gate、approval gate、audit ledger。第二，Gitdex 不能把“回滚”理解成某个 Git 命令，而必须区分本地 Git 回退、GitHub 远程补偿、deployment 恢复与人工接管。

如果下一步进入正式产物，最合适的是基于这份技术研究直接推进 `bmad-create-product-brief` 或 `bmad-create-prd`，并把以下内容写成强制性设计约束：`GitHub App first`、`webhook-first async orchestration`、`single-writer-per-repo-ref`、`policy-as-code`、`handoff pack`、`deployment via governed pipeline not direct imperative cloud control`。

**Research Completion Date:** 2026-03-18  
**Research Period:** current-source technical architecture analysis  
**Source Verification:** official technical sources plus local reference project pattern extraction  
**Confidence Level:** high for architecture direction and platform constraints; medium for concrete implementation topology choices
