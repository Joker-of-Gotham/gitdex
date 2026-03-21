---
stepsCompleted:
  - step-01-init
  - step-02-discovery
  - step-02b-vision
  - step-02c-executive-summary
  - step-03-success
  - step-04-journeys
  - step-05-domain
  - step-06-innovation
  - step-07-project-type
  - step-08-scoping
  - step-09-functional
  - step-10-nonfunctional
  - step-11-polish
  - step-e-01-discovery
  - step-e-02-review
  - step-e-03-edit
inputDocuments:
  - ./product-brief-Gitdex-2026-03-18.md
  - ./research/domain-repository-autonomous-operations-research-2026-03-18.md
  - ./research/technical-gitdex-architecture-directions-research-2026-03-18.md
  - ./research/market-gitdex-competitive-boundaries-and-trust-models-research-2026-03-18.md
  - ../brainstorming/brainstorming-session-20260318-152000.md
documentCounts:
  productBriefs: 1
  research: 3
  brainstorming: 1
  projectDocs: 0
workflowType: 'prd'
workflow: 'edit'
projectName: 'Gitdex'
author: 'Chika Komari'
date: '2026-03-18'
lastEdited: '2026-03-18'
classification:
  projectType: 'cli_tool'
  domain: 'developer infrastructure / repository autonomous operations'
  complexity: 'high'
  projectContext: 'greenfield'
  productShapeNote: 'terminal-first dual-mode operator interface with explicit commands and natural-language chat, backed by a daemon-backed governed control plane'
  systemRealityNote: 'The CLI/chat interface is the operator entry surface; the long-running governed control plane is the system core.'
  primaryJobNote: 'Gitdex''s primary job is governed repository operations and workload reduction around repo-centric engineering workflows, not general-purpose project management or generic AI coding assistance.'
  aiDifferentiationNote: 'LLM-assisted intent capture, planning, coordination, triage, context compression, and repository-adjacent project-management workload reduction'
  interactionNote: 'Commands and chat are co-equal operator surfaces, but not co-equal execution authorities: chat optimizes for intent, exploration, explanation, and planning; commands optimize for precision, repeatability, automation, and deterministic invocation.'
  executionContractNote: 'Any natural-language request that can lead to a governed write action must compile into an explicit, reviewable, structured execution plan before execution.'
  adoptionNote: 'Gitdex must support low-friction entry through terminal chat and explicit commands, while allowing users to progressively authorize deeper automation as trust is established.'
  boundaryStatement: 'Gitdex is a governed repository operations system, not a generic AI coding assistant or general project-management platform; LLMs provide cognitive leverage, while governed execution, auditability, and policy-bound control remain the product core.'
editHistory:
  - date: '2026-03-18'
    changes: 'Added explicit experience principles, onboarding and adoption journeys, clarified FR37, and converted key NFRs into measurable contracts.'
  - date: '2026-03-18'
    changes: 'Contractized remaining NFRs, added a phase-to-capability matrix, tightened selected FRs, and compressed repeated positioning across domain/innovation/CLI sections.'
---

# Product Requirements Document - Gitdex

**Author:** Chika Komari
**Date:** 2026-03-18

## Executive Summary

Gitdex 是一个面向终端环境的受治理仓库操作系统，用双模交互入口承载仓库自治运维：用户既可以通过明确命令进行精确、可脚本化、可复现的操作，也可以通过自然语言聊天表达目标、请求建议、理解状态并委派任务。它面向独立维护者、开源维护者、初创团队工程负责人和平台工程师，目标不是替代开发判断，而是把仓库维护、协作编排、上下文整理和重复性运维从高摩擦的人工作业，升级为可托管、可解释、可接管、可持续巡航的系统能力。

Gitdex 要解决的真实问题不是“缺少一个更聪明的命令行工具”或“缺少一个会聊天的 AI 助手”，而是现代仓库协作和维护的认知负担已经过高。维护者需要不断在本地文件修正、Git 事务、上游同步、PR 与 issue 处理、评论沟通、工作流触发、发布协调和仓库治理之间切换，同时还要自己判断该在什么时机、用什么工具、查看哪些上下文、执行哪些命令。随着节奏加快，这种以人工上下文拼接和单次指令执行为核心的工作方式正在直接消耗精力、影响情绪、放大协作摩擦，并降低仓库长期可维护性。

Gitdex 的愿景是把用户从“怎么做”中解放出来，让用户更专注于“做什么”。当 Gitdex 成功时，用户不再因为上游更新未同步、分支管理混乱、PR/issue 流程低效、多人协作失序或维护动作过于琐碎而消耗注意力；他们可以将更多精力投入到设计、优化、审阅、沟通和决策本身，而把仓库相关的重复性维护、状态跟踪、建议生成、任务编排和受控执行交给一个 7x24 持续运行、可治理、可审计的系统。

### What Makes This Special
**Experience Principles**

- Gitdex 默认像一个 `7x24` 的工程值班替身与长期协作系统，而不是一次性命令执行器或只会回答问题的聊天窗口。
- Gitdex 默认坚持 `all in terminal`，尽量减少网页、脚本、零散 dashboard 与终端之间的反复切换。
- Gitdex 默认先解释、再行动、可接管；任何高影响动作都必须先暴露计划、证据、风险与接管入口。
- Gitdex 默认把“减少认知切换、减少盯盘、减少上下文重建”视作一等产品结果，而不是附带收益。

Gitdex 的差异化不在于“自动化更多”或“聊天更自然”，而在于把自动化、可解释性、可溯源性、可接管性和持续巡航能力整合到同一个受治理产品中。它既能看、能读、能理解仓库及其协作状态，也能在 LLM 协助下做意图解析、计划生成、上下文压缩、triage、协调和建议输出；同时，它又不是一个只会“听令执行”的被动代理，而是一个能在受控边界内主动提出建议、持续观察、组织任务、暴露风险、保留证据并在异常时交还控制权的仓库操作系统。

Gitdex 的核心洞察是：当前仓库运维真正稀缺的不是更多 API 覆盖面，而是一个把“看见状态、理解上下文、组织行动、解释原因、执行动作、保留证据、允许接管”串成闭环的统一控制平面。GitHub 原生能力、专用 bot、单点自动化工具和 Claude Code / Copilot 类产品各自解决了一部分问题，但它们通常停留在单一对象、单一步骤、单一权限边界或单次任务代理上。Gitdex 要提供的是跨仓库对象、跨维护流程、跨时间维度的连续协作能力，让用户不必在多个工具和多个心智模型之间来回切换。

Gitdex 同时将命令与聊天定义为并列的操作入口，但不把它们视为并列的执行权限。聊天负责降低表达门槛、支持发现式探索、状态解释、任务分解和项目管理减负；命令负责提供精确调用、自动化、可复现性和工程纪律。任何可能导致受治理写操作的自然语言输入，都必须先收敛为显式、可审查、结构化的执行计划，再进入策略检查、审批、执行和审计链路。这样，Gitdex 才能在发挥 LLM 价值的同时，维持高权限仓库操作所要求的可靠性与信任边界。

## Project Classification

- **Project Type:** `cli_tool`
- **Domain:** `developer infrastructure / repository autonomous operations`
- **Complexity:** `high`
- **Project Context:** `greenfield`
- **Product Shape:** `terminal-first dual-mode operator interface with explicit commands and natural-language chat, backed by a daemon-backed governed control plane`
- **System Reality:** CLI 与聊天界面是 operator entry surface，长期运行的受治理控制平面才是系统核心。
- **Primary Job:** 受治理的仓库操作与围绕 repo-centric engineering workflows 的工作负担减轻，而不是通用项目管理平台或通用 AI 编码助手。
- **AI Differentiation:** LLM 用于意图捕获、计划生成、协调、triage、上下文压缩和与仓库相邻的项目管理减负。
- **Interaction Contract:** 命令与聊天是并列入口，但不是并列执行权限；命令偏向精确与自动化，聊天偏向解释、探索与规划。
- **Execution Contract:** 任何可能触发受治理写操作的自然语言请求，都必须先编译为显式、可审查、结构化的执行计划。
- **Adoption Model:** 通过终端聊天和显式命令提供低摩擦入口，并允许用户在信任建立后逐步授权更深层自动化。
- **Boundary:** Gitdex 是受治理的仓库操作系统，不是通用 AI 编码助手，也不是通用项目管理平台；LLM 提供认知杠杆，治理执行、审计可追溯与策略约束才是产品核心。

## Success Criteria

### User Success
其中，“首次成功体验”必须是一个完整闭环：用户完成首次 setup、看到可信状态摘要、审阅至少一个结构化计划，并成功完成一次值得保留的低风险维护或协作动作，而不是只完成一轮对话演示。

Gitdex 的用户成功不应只被定义为“更快”，而应被定义为“更少的认知切换、更少的盯盘、更少的工具跳转、更明确的下一步”。对核心目标用户而言，成功意味着仓库维护工作从高频人工跟进转变为受控托管，用户将注意力重新投入设计、审阅、沟通和决策本身。

首发阶段的用户成功标准定义为：

- 在目标维护场景中，用户主动盯盘和手动检查状态的时间较基线下降 `60%` 以上。
- 针对 Gitdex 首发覆盖的仓库维护场景，至少 `80%` 可在终端内完成，无需频繁切换到网页或额外工具。
- 对于自然语言提出的目标型请求，Gitdex 在 `60 秒` 内为至少 `90%` 的请求给出：
  - 当前状态摘要
  - 明确的下一步建议
  - 可审查的结构化执行计划或行动草案
- 新用户在首次安装后的 `24 小时` 内完成至少一次“值得保留”的成功体验；该体验必须是一个真实的仓库维护闭环，而不是单纯对话演示。
- 已激活仓库中，至少 `70%` 在 `30 天` 内持续启用两项以上低风险自治能力，并保留使用。
- 在定性层面，用户应明确表达一种核心价值感受：`Gitdex 让我不必再一直想“在哪里、什么时候、用什么工具、执行什么命令”，而可以直接聚焦要完成什么。`

### Business Success

Gitdex 的业务成功不应先用泛用户规模衡量，而应先验证它是否能成为高强度开发者和维护者愿意持续依赖的默认仓库操作入口。短期成功是“dogfood 可用性”与“真实维护留存”，中期成功才是“可扩展 adoption”。

#### 3 个月成功标准

- 创始人/主维护者在至少 `10` 个真实仓库上连续 `4` 周将 Gitdex 作为默认终端运维入口，用于日常维护、issue/PR 处理、状态理解、仓库整理或上游同步。
- 至少 `70%` 的试点仓库每周发生一次以上由 Gitdex 主导或辅助完成的真实管理活动。
- 至少 `3` 个核心能力包被稳定启用并持续使用：
  - 双模终端交互
  - 结构化执行计划与 dry-run
  - 基础 issue/PR/comment/repo maintenance orchestration
- 至少 `30%` 的试点仓库从纯观察/建议模式升级到可逆的低风险执行模式。
- 创始人和首批试点用户给出明确结论：Gitdex 已从“概念演示”跨过到“真实可用”。

#### 12 个月成功标准

- 至少 `50` 个活跃仓库持续由 Gitdex 管理或辅助管理。
- 已激活仓库的 `8 周留存率` 达到 `60%` 以上。
- 至少 `40%` 的活跃仓库将 Gitdex 用作默认仓库运维入口，而非偶发性实验工具。
- 至少 `50%` 的活跃仓库启用了 `L3` 或以上的自治能力。
- 至少 `10` 个外部维护者或团队完成持续试点，其中至少 `3` 个明确表达付费意愿、预算申请意向或正式采用信号。
- Gitdex 在目标人群中的认知从“终端里的 AI 助手”转变为“可授权的仓库操作控制平面”。

与外部采用相关的成功信号必须来自真实评估链路，而不是泛泛意向收集；至少一部分采用信号应来自 buyer、approver 或 security reviewer 等正式评估角色。

### Technical Success

Gitdex 的技术成功标准必须围绕三件事展开：受治理执行、可恢复、可审计。成功不是“永远不失败”，而是“在支持范围内高概率成功，在失败时安全收敛，在全链路上保留证据”。

- 在明确支持的低风险与中风险操作集合中，任务最终成功收敛率达到 `95%` 以上。
- 所有任务中，最终状态必须 `100%` 收敛为以下之一：
  - `succeeded`
  - `cancelled`
  - `failed with handoff complete`
  - `blocked by policy/approval`
  不允许出现无解释悬挂状态。
- 所有受治理写操作的审计链完整率必须达到 `100%`。
  审计链至少包含：
  - 触发来源
  - 结构化执行计划
  - 权限/策略判断
  - 执行结果
  - 关联证据
- 严重误操作事故目标为 `0`。
  严重误操作包括：
  - 未授权高风险写操作
  - 错误修改受保护目标
  - 未经控制推进高风险发布/合并/规则变更
- 所有可能导致受治理写操作的自然语言请求，结构化执行计划的可解释率和可审查率必须达到 `100%`。
- 对高风险动作，不要求固定 `100%` 人工审批覆盖率，因为策略可配置；但要求 `100%` 满足以下之一：
  - 命中审批策略并留下审批记录
  - 命中允许自动执行的明确策略并留下策略命中记录
  - 被策略阻止并留下拒绝记录
- 进入异常或阻塞状态后，`handoff pack` 生成时间应在 `60 秒` 内完成，并允许系统无限期保持可接管的安全暂停状态。
- Webhook 驱动链路必须符合 GitHub 官方最佳实践：
  - 入站处理在 `10 秒` 内完成确认
  - 后台异步处理
  - 基于 delivery ID 去重
  - 支持缺失事件补送与对账
- 所有权限扩展、关键配置变更、对象读写、认证授权事件都应进入安全日志，满足 GitHub App 官方最佳实践要求。

### Measurable Outcomes

Gitdex 的核心衡量体系采用平衡计分，而不是单一效率指标。最终仪表板至少应覆盖：

- `User Task Success`
  - 终端内完成率
  - 首次价值时间
  - 自然语言请求到结构化计划的成功率
  - 盯盘时间下降幅度
- `Adoption & Retention`
  - 激活仓库数
  - 周活跃仓库数
  - 30 天持续启用率
  - 8 周留存率
  - 自治层级升级率
- `Operational Trust`
  - 审计链完整率
  - 高风险动作策略命中/审批/阻止覆盖率
  - 严重误操作事故数
  - handoff pack 完成率
- `System Effectiveness`
  - 任务最终成功收敛率
  - safe pause / handoff / reconciliation 成功率
  - webhook 漏处理/重放问题发生率
  - 计划生成和状态摘要延迟
- `User Sentiment`
  - 开发者满意度调查
  - “是否愿意继续把这类任务交给 Gitdex” 的正向回答比例
  - “是否减少了迷茫、切换和维护摩擦” 的正向回答比例

## Product Scope

### MVP - Minimum Viable Product

Gitdex 的 MVP 不是“玩具演示版”。首发必须达到“真实可用、可持续使用、可被信任”的产品可用态。MVP 必须包含以下最小但完整的能力闭环：

- 双模终端交互：
  - 明确命令
  - 自然语言聊天
  - 两者共享同一受治理执行链路
- 仓库状态理解与上下文汇总：
  - 读取本地与远程仓库状态
  - 汇总 issue / PR / branch / workflow / upstream context
  - 生成状态摘要和下一步建议
- 结构化执行计划：
  - 所有潜在写操作先生成可审查计划
  - 默认支持 dry-run / preview / risk explanation
- 基础仓库操作与协作编排：
  - issue / PR / comment 基础操作
  - 上游同步与基础分支管理建议
  - 基础 repo hygiene 与低风险维护任务
- 受治理执行基础设施：
  - 审计链
  - autonomy levels L0-L3
  - safe pause
  - handoff pack
  - kill switch
- daemon-backed 长时运行能力：
  - webhook / queue / schedule / reconciliation 基础能力
- 多操作系统可用性：
  - 至少支持目标用户的主流终端环境并保持一致的核心体验

### Growth Features (Post-MVP)

Growth 阶段的目标不是补齐“缺失功能”，而是让 Gitdex 从单仓库可用工具升级为更强的仓库自治运维平台。

- all in terminal 的更完整落地：
  - 更强 TUI
  - 批量操作
  - 更完整的终端内审查、审批、回放和诊断
- 更高自治等级：
  - bounded L4-L5 任务窗口
  - 更成熟的 7x24 巡航式维护
- 多仓库治理：
  - maintenance campaigns
  - fleet-level policy orchestration
  - cross-repo status boards
- 更强 GitHub 编排：
  - workflow orchestration
  - deployment governance
  - richer release coordination
- 更强 LLM 协作：
  - 更优任务分解
  - 更优状态解释
  - 更优项目管理减负与协同建议

### Vision (Future)

Vision 阶段是 Gitdex 从受治理仓库操作系统进一步演化为 7x24 持续开发与发布控制平面的阶段，但不应进入首发承诺。

- 与 Cursor、Claude Code、OpenHands / OpenDevin 类系统形成明确分工和协同接口
- 贯通从需求、实现、仓库维护、代码审查、发布治理到部署交接的可解释全流程
- 成为高质量、非混乱、强可追溯的 agentic software development control plane
- 在保持治理、审计和可接管的前提下，成为 7x24 无间断编程开发、仓库维护与产品发布的重要基础设施

## User Journeys

### Journey 1: 独立维护者把 Gitdex 变成默认终端入口

**Persona:** 林川，独立开发者 / 小型仓库主理人，维护多个开源和个人项目，经常在夜间或碎片时间处理仓库维护、上游同步、PR 检查和 issue 回应。

**Opening Scene**  
我们遇到林川时，他已经习惯了在多个终端、多个网页标签和多个仓库之间来回切换。每次打开项目，他都要先确认上游有没有更新、哪些分支已经漂移、哪些 PR 卡住、哪些 issue 需要回应。真正让他疲惫的不是单个操作本身，而是每次都要重新建立上下文并决定“先看什么、先做什么、用什么做”。

**Rising Action**  
林川第一次使用 Gitdex，不是为了自动完成所有事，而是为了把每天重复的仓库维护工作收拢到一个终端入口中。他先用自然语言问：“现在这个仓库最需要处理什么？” Gitdex 汇总本地仓库状态、上游差异、未处理 PR、活跃 issue 和 workflow 状态，给出清晰摘要与建议。接着，林川通过命令确认某个建议动作的 dry-run 计划，比如同步上游、整理分支、补充 PR 描述或清理 stale issue。

**Climax**  
决定性的时刻发生在一个平常但高摩擦的夜晚。林川打开项目时，不再需要先手动检查十几个地方；Gitdex 已经把仓库当前最重要的变化和待处理动作整理好，并给出结构化执行计划。他只需要确认重点和边界，而不是自己重新拼出全局上下文。那一刻，林川第一次感觉到“Gitdex 不是又一个工具，而是一个真正会陪着我维护仓库的系统”。

**Resolution**  
随着使用加深，林川开始把更多低风险操作交给 Gitdex 巡航完成，例如仓库卫生整理、issue/PR 基础分流、状态摘要和部分可逆维护动作。Gitdex 成为他的默认终端入口；他保留决策权和关键审批，但不再被日常重复劳动持续消耗。

**Journey Requirements Revealed**
- 双模终端交互：自然语言提问 + 明确命令执行
- 仓库状态聚合、上游比较、PR/issue/workflow 汇总
- dry-run、风险说明、结构化执行计划
- 低风险自治任务与可逆执行能力
- 终端内持续上下文，而不是一次性问答

### Journey 2: 开源维护者在混乱边缘重新找回秩序

**Persona:** 阿青，开源维护者，管理一个活跃社区仓库。issue、PR、评论和上游变化持续涌入，最害怕的是社区体验变差和维护节奏失控。

**Opening Scene**  
我们遇到阿青时，仓库已经开始出现熟悉的失控迹象：重复 issue 增多、PR 描述质量参差不齐、review 路由混乱、comment 中混杂着问题、建议和情绪。她不是不会处理，而是每次都要先花大量精力给信息“去噪”。

**Rising Action**  
阿青引入 Gitdex 后，先把它作为“终端里的维护协作者”。她会直接提问：“今天有哪些 issue/PR 需要优先回应？” 或要求 Gitdex 对仓库当前的 backlog 做聚类和风险解释。Gitdex 并不直接接管社区，而是先帮助她把仓库里的噪音变成结构化对象：哪些是重复问题、哪些是上游已解决、哪些需要补上下文、哪些需要人工判断。

**Climax**  
关键时刻出现在一次上游变化和社区讨论同时爆发的窗口期。阿青不再被动地被 comment 和通知驱着走，而是让 Gitdex 先总结当前局势，列出可处理项、建议项和必须人工接管项，并在终端内提供 issue/PR/comment 的整理与草案。她不再只是“救火”，而是重新掌握了秩序。

**Resolution**  
Gitdex 没有把阿青从维护链路中拿掉，而是让她从噪音管理者变成判断者和协调者。仓库协作的可读性提高，社区沟通更加清晰，重复劳动减少，情绪成本下降。

**Journey Requirements Revealed**
- issue / PR / comment 统一视图与优先级建议
- 重复/陈旧/缺失上下文对象识别
- 终端内 comment / issue / PR 草案生成与建议回复
- 社区协作不透明场景下的解释与风险提示
- 人工判断与系统建议之间的清晰边界

### Journey 3: 平台工程师为团队建立受治理的授权边界

**Persona:** 周祁，初创团队工程负责人 / 平台工程师，负责多个仓库的维护秩序、工作流治理和自动化边界。他愿意授权自动化，但不接受不可控的黑盒。

**Opening Scene**  
我们遇到周祁时，团队已经有不少脚本、GitHub Actions 和散落的维护约定，但它们缺乏统一控制面。每当团队讨论“能不能自动做更多”时，真正阻碍推进的不是技术，而是缺乏一个能被解释、被授权、被审计的治理层。

**Rising Action**  
周祁第一次接触 Gitdex 时，关注的不是聊天能力，而是它能否在终端里同时提供配置、策略、审批和观察入口。他以 GitHub App 安装和仓库级授权为起点，为不同仓库设置自治级别、审批边界和可用能力包，并验证 Gitdex 是否能在不突破现有治理规则的情况下运作。

**Climax**  
真正建立信任的时刻，不是 Gitdex 成功做对了一件事，而是它在高风险边界前停了下来，明确告诉周祁：这一步命中了策略限制，需要审批或人工确认。Gitdex 证明了它不仅会执行，更会克制。

**Resolution**  
随着信任建立，周祁开始在部分仓库放开更深的低风险自治能力，把原本散落在脚本、Action 和人工值守中的工作，逐步迁移到一个可观察、可追溯、可干预的终端控制平面中。

**Journey Requirements Revealed**
- GitHub App 安装、repo-scoped 授权与 capability 管理
- 自治等级配置、审批策略、风险边界设置
- 审计链、策略命中、拒绝记录、handoff pack
- 多仓库可视化治理与逐步放权机制
- “拒绝执行”与“需要审批”同样是一等体验

### Journey 4: 值班负责人在失败与阻塞状态下快速接管

**Persona:** 许岚，发布经理 / 值班负责人，在关键窗口内负责判断是否继续、暂停、回滚或交接。她不要求 Gitdex 永远不失败，但要求失败时不混乱。

**Opening Scene**  
我们遇到许岚时，某个仓库任务已经进入异常状态：可能是分支冲突、审批未通过、远端状态不一致，或执行了一部分后被策略阻挡。最糟糕的情况不是失败，而是没有人能快速回答“现在到底做到哪一步了”。

**Rising Action**  
许岚通过 Gitdex 终端查看阻塞任务，系统已经生成 handoff pack：触发来源、当前状态、已执行步骤、未执行步骤、风险点、相关对象和下一步建议一应俱全。她不需要先翻日志、找 comment、对照网页状态，而是直接在终端里完成理解、判断和接管。

**Climax**  
关键时刻是当她发现自己不需要亲自“重新构建事实”，而是直接在一个结构化视图上作出判断：继续等待审批、改为取消、重新执行某个阶段，或将任务切换为人工处理。Gitdex 把故障场景从混乱信息堆变成了可决策对象。

**Resolution**  
对于许岚来说，Gitdex 的价值不只在“平时省力”，更在“出事时可接管、可解释、可安全暂停”。这使得她愿意把更多常规任务留给系统，因为系统证明了自己在失败时依然守纪律。

**Journey Requirements Revealed**
- handoff pack 自动生成
- 任务时间线、状态机和执行证据可视化
- safe pause / cancel / retry / quarantine / takeover 入口
- 异常状态的终端内解释与下一步建议
- 失败恢复流程必须与正常流程同样顺滑

### Journey 5: 集成用户与 campaign operator 让多仓库治理变成可编排能力

**Persona:** 陈放，平台工具开发者兼多仓库治理操作者。他一方面希望通过 API / integration 把 Gitdex 接入现有内部工具或工作流，另一方面需要跨多个仓库发起 maintenance campaigns、策略整治和批量修复。

**Opening Scene**  
我们遇到陈放时，团队已经不满足于单仓库管理。他需要统一推动多个仓库做规则收敛、依赖升级、仓库卫生治理和策略执行，但现状是这些动作分散在脚本、表格、issue 模板和人工广播里，既难协调，也难审计。

**Rising Action**  
陈放先把 Gitdex 作为可编排控制平面来使用：他通过 API、配置或集成入口向 Gitdex 提交一个 campaign intent，例如“检查所有目标仓库的上游同步状态并生成维护计划”或“对一组仓库执行低风险规则整治”。Gitdex 不直接盲目执行，而是先产出 fleet-level 计划、影响分析和逐仓库执行建议。

**Climax**  
决定性时刻不是 campaign 被发起，而是它能在终端与集成入口中被同时观察、审查和干预：陈放可以在终端内逐批审查计划、批准部分仓库执行、阻止高风险对象、查看失败项，并将结果反馈回内部工具或下游系统。Gitdex 在这里不再只是“终端工具”，而是变成了多仓库操作的可治理协调器。

**Resolution**  
陈放不再需要靠散乱脚本和人工广播去驱动多仓库维护。Gitdex 让 API / integration 与 operator terminal 融合在同一个状态机和审计链之下，使多仓库 campaign 从一次性项目变成持续、可回放、可追踪的治理能力。

**Journey Requirements Revealed**
- 可编排 API / integration surface
- campaign intent -> structured fleet plan 的转换能力
- 多仓库影响分析、分批执行、逐仓库状态跟踪
- 跨仓库审批、失败对账和结果回传能力
- integration surface 与 terminal operator surface 共享同一治理与审计模型

### Journey 6: 新用户在 24 小时内完成第一次“值得保留”的成功体验

**Persona:** 林舟，第一次接触 Gitdex 的独立维护者或团队成员。他对仓库运维痛点非常熟悉，但不愿意再为一个“看起来很强”的新工具投入高成本学习曲线。

**Opening Scene**  
林舟第一次安装 Gitdex 时，并不期待它立刻接管仓库；他更关心的是能否在第一次使用中快速完成身份配置、权限确认、仓库接入和可信状态查看，而不是卡在复杂 setup、模糊授权或空白界面上。

**Rising Action**  
Gitdex 用交互式 setup 帮他完成最小可用配置，然后直接引导到一个真实仓库：汇总当前状态、指出最值得处理的事项，并给出一个低风险、可审查的结构化计划，例如上游同步预演、issue 分流草案或分支清理建议。林舟不需要自己猜“下一步试什么”，而是被带入一个可验证的闭环。

**Climax**  
关键时刻不是他看懂了功能列表，而是他在第一次使用里真的完成了一次有价值的动作闭环：看见状态、理解风险、审阅计划、完成执行或确认建议，并得到清晰结果与证据。这个时刻决定 Gitdex 是“试过一次的工具”还是“值得保留的默认入口”。

**Resolution**  
完成首次闭环后，林舟愿意在后续 24 小时内再次进入 Gitdex，而不是回到原来的网页+脚本+命令拼接流程。Gitdex 对他证明的不是“功能多”，而是“第一次就能减少混乱并创造真实价值”。

**Journey Requirements Revealed**
- 交互式 setup、身份配置、权限说明与默认配置引导
- 首次接入后的可信状态摘要与下一步建议
- 低风险、可审查、可 dry-run 的首次任务闭环
- 首次成功体验的结果记录、证据展示与后续引导
- 从首次体验到持续使用的 trust ramp 设计

### Journey 7: buyer / approver / security reviewer 确认 Gitdex 值得被授权与采用

**Persona:** 王宁，工程负责人、平台 Owner、审批者或安全评审参与者。他未必每天直接使用 Gitdex，但会决定 Gitdex 能否进入正式试点、获得更深权限，或在团队内被持续采用。

**Opening Scene**  
王宁第一次评估 Gitdex 时，关心的不是“它会不会更酷”，而是“它凭什么值得被授权”。他需要看到的不是一个会聊天的终端工具，而是一个权限边界清楚、策略可解释、失败可接管、审计可导出的受治理系统。

**Rising Action**  
Gitdex 向他展示的重点不是操作炫技，而是治理证据：GitHub App 边界、能力授权模型、structured plan 合约、审批路径、审计链、kill switch、handoff pack，以及在 text-only 终端和集成入口中保持一致的治理语义。王宁需要确认的是，这个系统能否在团队、部门或企业环境中被负责任地引入。

**Climax**  
决定是否放行的关键时刻，是当王宁看到 Gitdex 能在高风险动作前主动停下、解释原因、保留证据、等待审批，且在异常时把控制权优雅地还给人。Gitdex 在这里证明的不是“我可以做”，而是“我知道什么时候不该做，以及怎样把责任边界说清楚”。

**Resolution**  
一旦评估通过，王宁会允许 Gitdex 从小范围试点进入正式采用，并逐步扩大授权深度与仓库范围。此时 Gitdex 的价值不再只是单个 operator 的效率工具，而是一个可被组织接受的仓库治理基础设施。

**Journey Requirements Revealed**
- 面向 buyer / approver / security reviewer 的授权与治理解释面
- GitHub App、capability grants、approval policy、audit trail 的可检查证据
- 正式试点、权限升级与组织采用的评估路径
- 可导出的审计、报告、handoff 与风险说明材料
- 组织采用前的信任建立、责任边界和例外处理机制

### Journey Requirements Summary

这些旅程共同揭示出 Gitdex 不能只是“会聊天的 CLI”，也不能只是“会跑命令的 bot”。产品至少需要以下能力层共同成立：

- **Operator Surface**
  - 明确命令
  - 自然语言聊天
  - TUI / 终端内状态面板
  - 审批、回放、接管入口
- **Context Assembly**
  - 本地仓库状态、远程 GitHub 状态、上游差异、issue / PR / comment / workflow / deployment 汇总
  - 面向人可读的摘要
  - 面向系统可执行的结构化上下文
- **Governed Planning**
  - 自然语言和命令都能收敛到结构化执行计划
  - dry-run、风险解释、审批需求、可逆性说明
- **Execution & Recovery**
  - 低风险自治执行
  - 高风险边界暂停
  - safe pause、handoff、retry、cancel、quarantine、takeover
- **Governance & Trust**
  - GitHub App 授权
  - capability grants
  - autonomy levels
  - policy checks
  - 审计链与证据保留
- **Fleet & Integration**
  - API / integration surface
  - 多仓库 campaign orchestration
  - fleet-level observability
  - integration 与 terminal 共享同一状态机与治理模型

这些旅程也进一步确认了一个产品事实：Gitdex 的核心不是“替用户写代码”，而是“替用户持续组织和治理仓库操作，让用户把精力放回真正重要的工程判断与创造性工作上”。

这些新增旅程进一步补足了此前隐含但未显式化的两层产品能力：

- **Activation & First Value**
  - 首次 setup
  - 首次仓库接入
  - 首次可信状态摘要
  - 首次低风险闭环任务
- **Adoption & Authorization**
  - buyer / approver / security reviewer 评估路径
  - 正式试点与权限升级证据
  - 组织采用所需的报告、审计与责任边界说明

## Domain-Specific Requirements

### Compliance & Governance

Gitdex 所处的领域虽然不属于医疗、金融这类强监管垂直行业，但它天然落在高权限开发者基础设施与软件供应链治理的交叉带，因此其“合规”重点不在业务牌照，而在授权、审计、最小权限和可追溯性。

- 必须以 `GitHub App` 作为首选机器身份，而不是长期 `PAT` 作为默认授权模型。
- 必须支持安装级、仓库级和能力级边界，避免默认获得组织范围的隐式高权限。
- 对受治理写操作，必须保留完整审计链，满足内部审计、故障复盘和安全调查要求。
- 必须支持策略命中、审批命中、拒绝命中和降级命中的记录保存，而不是只记录“执行成功”。
- 必须允许组织将 Gitdex 作为受治理自动化系统来授权，而不是作为不可解释的黑盒代理来授权。
- 对企业场景，应预留与 `SAML/SSO`、企业审计导出、私有部署或受限网络环境对接的能力路径，即便这些不全部进入首发 MVP。

### Technical Constraints

Gitdex 的技术约束来自三个事实：
1. 它处理的是仓库与协作对象，而不是单纯文本会话。
2. 它会触发真实副作用，而不是只生成建议。
3. 它必须在失败时保持可接管、可恢复，而不是只追求 happy path 自动化。

因此必须满足以下技术约束：

- 所有自然语言输入若可能导致写操作，必须先收敛为结构化执行计划。
- 命令入口与聊天入口必须共享同一治理链路，不能形成两个不同的执行真相。
- 所有高风险动作都必须经过策略评估，必要时进入审批、阻止或降级执行。
- Git 执行必须基于隔离工作区和可逆事务边界，避免共享工作目录污染。
- 远端副作用必须采用补偿与对账模型，不能假设所有动作都能简单“回滚”。
- webhook 驱动链路必须异步化、可去重、可补送、可重放、可对账。
- 审计、追踪、handoff pack、safe pause、kill switch 必须是产品核心，不是后补功能。
- 系统必须支持长期运行，但不能把稳定建立在“人工一直盯着看”之上。

### Security & Privacy Constraints

Gitdex 的安全重点是“高权限仓库操作安全”而不是传统终端工具安全，因此要特别强调以下几点：

- 默认最小权限：只申请当前 capability 所需 GitHub App 权限。
- 默认短时凭证：优先 installation token、job-scoped token、OIDC，而非长期静态密钥。
- 默认受限执行：高风险执行需要隔离工作区、受控环境变量、受控网络和工具白名单。
- 默认可解释：所有拟执行操作都必须向用户解释“为什么做、影响什么、依据什么策略”。
- 默认可阻断：用户或策略应能在任务级、仓库级、租户级触发暂停或熔断。
- 对 LLM 处理的上下文、日志、缓存和第三方模型调用必须明确数据边界、保留策略和训练策略，避免企业授权阻力。
- 对恶意仓库内容、prompt injection、comment injection 和不可信脚本内容必须有明确防御策略。

### Integration Requirements

Gitdex 不是孤立终端工具，它必须作为仓库控制平面嵌入现有 GitHub 与工程环境：

- GitHub 核心集成：
  - repository
  - branch / ref
  - issue
  - pull request
  - comment
  - workflow / actions
  - deployment / environment
  - webhook
  - audit/event surfaces
- 本地集成：
  - 本地 Git 仓库
  - 多工作区 / worktree
  - 本地终端与 shell 环境
- 平台集成：
  - CI/CD
  - 身份与审批系统
  - 日志 / 审计导出系统
  - 未来的 IDE / agent / internal tooling 接口
- fleet-level 集成：
  - 多仓库 campaign orchestration
  - repo classification / policy bundles
  - integration/API 入口与 terminal operator 入口共用状态模型

### Domain Patterns & Anti-Patterns

这个领域里有一些模式应当直接采纳，也有一些反模式必须明确禁止。

**Recommended patterns**
- `GitHub App first`
- `webhook-first + async queue + reconciliation`
- `simulation before mutation`
- `policy-as-code`
- `single source of truth for job state`
- `handoff pack and safe pause by default`
- `progressive autonomy by capability and risk tier`

**Anti-patterns**
- 用长期 `PAT` 驱动默认高权限自动化
- 自然语言直接跳过结构化计划进入执行
- 用单次会话上下文代替持久状态机
- 把聊天体验当作治理能力的替代品
- 没有审计链就做跨仓库或高风险写操作
- 默认把 deployment autonomy 和 repo maintenance autonomy 放在同一信任等级
- 把失败恢复寄托在人工翻日志和手工重构事实之上

### Domain Risks & Mitigations

这个领域最容易被低估的风险不是“系统会不会报错”，而是“系统在高信任状态下做了错误但貌似合理的事”。

- **Risk: 授权过宽**
  - Mitigation: GitHub App + capability grants + installation-scoped boundaries
- **Risk: 聊天入口造成模糊执行**
  - Mitigation: natural language -> structured plan -> policy gate -> execution
- **Risk: 跨仓库 campaign 放大 blast radius**
  - Mitigation: 分批执行、逐仓库审批、逐仓库回执、fleet-level rollback/handoff strategy
- **Risk: webhook / API 不一致导致状态漂移**
  - Mitigation: durable event log + reconciler + delivery dedupe + replay support
- **Risk: 恶意仓库内容或 comment 注入**
  - Mitigation: content trust boundaries + validator + execution allowlists + policy filters
- **Risk: 用户把 Gitdex 误解为“通用 AI 编码代理”**
  - Mitigation: 明确产品边界、能力边界和默认自治等级
- **Risk: 失败时没有人能快速接管**
  - Mitigation: handoff pack、safe pause、terminal diagnostics、可追溯状态机

## Innovation & Novel Patterns

### Detected Innovation Areas

Gitdex 的创新不在于单独引入某一项技术，而在于把 repo-centric 的认知辅助、受治理执行和长期自治收敛进同一产品边界。它明确挑战两个主流假设：

1. 仓库维护必须依赖人持续盯盘、持续切换工具、持续手动建立上下文。  
2. 自动化一旦变强，就必然难以解释、难以授权、难以接管。

围绕这两个假设，Gitdex 的首要创新集中在两个方面：

- **repo-centric 的 LLM 协作与项目管理减负**  
  Gitdex 把 LLM 放进仓库操作上下文，而不是把仓库操作降格为泛聊天任务。它的重点是状态理解、上下文压缩、风险解释、下一步建议、triage 和任务组织，从而减少维护者在“先看什么、先做什么、在哪里做”的认知负担。

- **7x24 可解释、可接管的自治运维**  
  Gitdex 试图证明仓库自动化不必在“强自动化”和“强治理”之间二选一。它把长期自治、结构化计划、审批边界、审计链和 handoff 机制整合进一个长期运行系统中，使自治不是黑盒替代人，而是高可解释、高可接管的协作形态。

这两个创新点共同定义了 Gitdex 的产品位置：一个终端优先、以仓库为中心、可被授权的认知与执行控制平面。

### Market Context & Competitive Landscape

当前市场已经分别存在：
- GitHub 原生能力与专用仓库治理能力
- Claude Code / Copilot 类 AI 编码与审查助手
- 各类单点自动化 bot、脚本和工作流
- 平台工程和多仓库治理工具

但这些方案通常分散在几个边界中：
- 只擅长单次任务，而不擅长持续协作
- 只擅长建议或生成，而不擅长受治理执行
- 只擅长自动化，而不擅长在失败时把控制权优雅交还给人
- 只覆盖单仓库或单对象，而不覆盖 repo-centric 的长期维护负担

Gitdex 的市场创新点因此不是“首次发明某项底层技术”，而是首次将以下组合收敛进一个终端优先产品边界内：
- all in terminal
- repo-centric LLM 协作
- 结构化计划先于执行
- 可解释、可审计、可接管的 7x24 自治
- 从单仓库维护到多仓库治理的渐进路径

它的竞争优势不是“功能最多”，而是“把原本分散的能力组合成可持续、可授权、可依赖的统一使用方式”。

### Validation Approach

Gitdex 的创新不能通过“概念认同”来验证，必须通过行为和授权意愿来验证。核心验证方法如下：

- **默认入口验证**
  - 用户是否持续把 Gitdex 当作默认仓库运维入口，而不是偶尔体验的附加工具。
- **all in terminal 验证**
  - 目标维护场景中，终端内完成率是否显著高于现有工具组合，并减少网页与脚本之间的切换。
- **认知减负验证**
  - 用户是否明显减少“要先看什么、先做什么、在哪里做”的迷茫状态。
- **授权升级验证**
  - 用户是否愿意在建立信任后，把更多低风险或中风险动作交给 Gitdex 托管。
- **多仓库治理验证**
  - 与脚本、网页和分散工作流相比，Gitdex 是否让多仓库 campaign 更可观察、更可干预、更可追溯。
- **异常可接管验证**
  - 当自治任务失败或阻塞时，Gitdex 是否仍然被视作可依赖系统，而不是额外制造混乱的黑盒。

如果这些验证信号不能成立，Gitdex 的“创新”就只是概念包装；只有当用户把它作为默认入口并逐步授权更深层自治时，这些创新才算被证实。

### Risk Mitigation

Gitdex 的创新风险不在于“太前沿”，而在于以下几类现实失败模式：

- **Risk: 用户接受终端统一入口，但不接受 7x24 自治**
  - Mitigation: 先把 Gitdex 做成最强的终端仓库控制台，让双模终端、状态理解、结构化计划和受治理执行独立成立，再逐步扩展自治深度。
- **Risk: LLM 协作提高了便利性，但没有形成真实减负**
  - Mitigation: 所有 LLM 能力必须围绕 repo-centric workflow 设计，优先验证摘要、建议、triage、计划和协调，而不是泛聊天能力。
- **Risk: 用户喜欢聊天入口，但不愿让聊天触发高权限动作**
  - Mitigation: 明确保持“自然语言 -> 结构化计划 -> 审批/策略 -> 执行”的执行合同。
- **Risk: 7x24 自治被理解为不可控黑盒**
  - Mitigation: 把可解释性、审计链、handoff pack、safe pause 和 kill switch 产品化，并将其作为创新本身的一部分，而不是附属特性。
- **Risk: 市场把 Gitdex 误认成 AI coding assistant，而不是仓库操作控制平面**
  - Mitigation: 所有定位、交互和指标都围绕 repo-centric operations、all in terminal 和 governed autonomy 展开，而不是围绕“生成代码”展开。

### Fallback Strategy

如果 Gitdex 的完整 7x24 自治形态在早期阶段没有完全成立，产品仍然必须以一个强而独立的 fallback 形态存在：

- Gitdex 至少应成为最强的终端仓库控制台：
  - 统一状态入口
  - 双模交互
  - 结构化执行计划
  - 终端内 issue / PR / branch / workflow / upstream 管理
  - 可审计、可预演、可接管的仓库操作体验

这意味着 Gitdex 的创新路线允许分阶段落地：  
先证明“all in terminal + repo-centric LLM 协作 + governed execution”可以独立成立，再证明“7x24 自治巡航”可以在此基础上被用户接受并逐步授权。

## CLI Tool Specific Requirements

### Project-Type Overview

Gitdex 虽然在项目分类上属于 `cli_tool`，但它不是传统的一次性命令集合，而是一个终端优先的双模 operator product。项目类型约束必须同时覆盖两套能力：

- `interactive-first for intent`
  - 处理“要做什么、为什么做、现在发生了什么、下一步建议是什么”
- `scriptable-first for execution`
  - 处理“怎么做、如何重复执行、如何集成到 automation / CI / agent workflow 中”

Gitdex 的 CLI 不是聊天前端附庸，也不是 daemon 的调试壳；它本身就是正式的 operator surface。“all in terminal” 也不等于“所有东西都塞进单一聊天框”，而是要求命令、聊天、状态浏览、审批、接管、审计和 fleet/campaign 操作在终端内形成结构化、分层的统一体验。

### Technical Architecture Considerations

作为 `cli_tool`，Gitdex 需要被设计为 `thin operator client + governed local/remote control plane`。这意味着终端负责 operator interaction，后台负责长期任务、策略和审计，而不是把所有状态、权限和编排都塞进一次性 CLI 进程里。项目类型决定了以下技术方向：

- CLI/TUI 层负责：
  - 命令解析
  - 会话交互
  - 聊天式意图输入
  - 状态呈现
  - 计划预览
  - 审批与接管
- 后台控制平面负责：
  - 任务状态机
  - webhook / schedule / queue 驱动
  - policy evaluation
  - structured plan execution
  - audit ledger
  - reconciliation / retry / handoff

这一定义的是职责边界，而不是实现细节：终端是默认入口，后台是长期自治与治理核心，但两者必须共享同一状态、计划和审计契约。

### Command Structure

Gitdex 的命令结构必须同时支持显式操作和自然语言协作，两者需要共享同一治理链路，但承担不同职责。

建议命令结构按能力域组织，而不是按底层 API 或对象碎片组织：

- `gitdex status` / `gitdex summary`
  - 查看 repo / fleet 当前状态、风险、待办和建议动作
- `gitdex chat`
  - 进入自然语言协作模式，用于提问、探索、任务分解、状态解释、计划草拟
- `gitdex plan`
  - 将目标、命令或自然语言请求编译为结构化执行计划
- `gitdex run` / `gitdex apply`
  - 执行已批准或低风险可自动执行的计划
- `gitdex approve` / `gitdex deny`
  - 对计划、任务、campaign 或高风险动作进行审批/拒绝
- `gitdex takeover` / `gitdex pause` / `gitdex resume` / `gitdex cancel`
  - 接管、暂停、恢复或取消自治任务
- `gitdex repo` / `gitdex pr` / `gitdex issue` / `gitdex workflow` / `gitdex deployment`
  - 面向对象的精确命令入口
- `gitdex campaign`
  - 多仓库治理、批量策略执行、fleet 级维护与观察
- `gitdex audit` / `gitdex report` / `gitdex handoff`
  - 导出审计记录、交接包、执行报告和结构化上下文
- `gitdex config` / `gitdex policy` / `gitdex auth`
  - 配置、权限、安装和治理策略管理
- `gitdex doctor`
  - 诊断环境、权限、连接、hook、队列和状态机问题

命令与聊天的关系必须明确：

- 聊天更适合表达目标、询问原因、理解状态、请求建议和形成计划
- 命令更适合稳定脚本化、明确对象操作、自动化集成和精确重放
- 任意自然语言如果可能触发受治理写操作，必须先落为 `structured execution plan`

### Output Formats

Gitdex 必须把“输出格式”视为产品能力，而不是序列化细节。不同输出格式服务不同消费方，但应共享同一事实来源。

必须支持至少四类输出：

- `human-readable terminal output`
  - 面向操作者的清晰终端文本、表格、diff 预览、状态摘要、风险解释、下一步建议
- `structured machine output`
  - `JSON` / `YAML`，用于脚本、CI、agent、IDE、集成系统消费
- `audit and event output`
  - 稳定 schema 的事件流、任务日志、安全日志、审批记录、策略命中记录
- `handoff/report artifacts`
  - handoff pack、incident pack、campaign report、structured plan、execution result bundle

输出契约还应满足：

- 同一任务在人类可读和机器可读层面应可相互映射
- 关键对象必须可被统一抽象和稳定表示：
  - 本地文件与目录
  - Git 工作区、分支、提交、差异、同步状态
  - GitHub 的 issue、PR、comment、workflow、deployment、environment、policy 结果
- 任意计划、执行、失败、接管都必须可导出为可复制给其他 agent / IDE / CI 的结构化工件

### Config Schema

Gitdex 必须采用多层配置模型，以匹配 CLI 工具、仓库治理系统和长期自治平台的复合属性。单一配置入口不足以支持真实使用场景。

必须支持以下配置来源：

- 全局配置文件
  - 用户级默认行为、终端偏好、身份与输出偏好
- 仓库级配置文件
  - repo policy、risk tier、autonomy level、默认集成、对象规则
- 命令即时配置
  - 本次调用的 flags、session overrides、临时策略调整
- 环境变量
  - 适合集成、CI、临时 secret 注入和脚本化调用
- 交互式 setup / onboarding
  - 首次使用时完成身份安装、能力授权、默认行为、终端模式和安全边界初始化

推荐优先级为：

`session flags > environment variables > repo config > global config > built-in defaults`

其中交互式 setup 不是“锦上添花”，而是关键 adoption 机制。对于 Gitdex 这种高上下文、高能力产品，首次使用必须允许用户通过 guided setup 在终端中完成：

- GitHub App / auth 安装
- 默认输出风格选择
- 命令模式与聊天模式偏好
- 风险级别与审批默认值
- 默认 repo / fleet scope
- 日志与审计保留策略

### Scripting Support

虽然 Gitdex 强调交互和聊天，但它若不能被脚本化，就无法成为真实的 developer infrastructure 工具。脚本化支持必须是一等能力。

Gitdex 必须支持：

- 非交互模式调用
- 稳定 exit codes
- `--json` / `--yaml` 等结构化输出开关
- `--dry-run` / `--preview` 模式
- `--plan-out` / `--report-out` / `--handoff-out` 等工件导出
- 在 CI、agent runtime、scheduler、cron、pipeline 中可预测执行
- 可将聊天/目标输入收敛成结构化计划，再交给脚本执行

脚本化支持的设计原则：

- 交互用于形成意图与理解上下文
- 脚本用于稳定执行、批量化处理和系统间编排
- 同一底层计划与状态机既能被交互层触发，也能被自动化层复用

### Terminal UX Requirements

Gitdex 的终端体验不应停留在传统“命令成功/失败 + 一段输出”的层面。作为 terminal-first 产品，它必须在 `Windows`、`Linux`、`macOS` 三端都提供高质量 operator UX，并从 `gh-dash`、`Claude Code`、`Codex`、`iFlow` 一类产品中吸收成熟模式。

终端体验要求包括：

- 键盘优先
  - 高频操作不依赖鼠标
- 命令与聊天并存
  - 显式命令和自然语言聊天都是一等入口
- TUI / rich terminal view
  - 至少对高价值场景提供状态面板、任务列表、diff 预览、审批视图、campaign 视图
- 持续状态感知
  - watch mode、live status、后台任务观察、事件流更新
- 结构化信息布局
  - 参考 `gh-dash` 式对象聚焦浏览
  - 参考 `Claude Code` / `Codex` 式对话-计划-执行切换
  - 参考 `iFlow` 式配置、运行与诊断的一致终端工作流
- 可复制性
  - 计划、命令建议、handoff 包、报告都能方便复制/导出/重放
- 渐进披露
  - 默认输出不过载，但允许钻取细节、证据、策略命中和日志
- shell completion
  - PowerShell、bash、zsh 至少要有路径
- 失败与接管体验
  - 异常时能一眼看清“卡在哪里、为什么、接下来谁接管”

### Implementation Considerations

作为 `cli_tool`，Gitdex 的实现边界还需要固定以下约束：

- 必须一开始就把 `Windows + Linux + macOS` 当作正式支持目标，而不是事后兼容
- 必须提供 text-only fallback，防止 TUI 依赖导致不可用
- 必须让聊天、命令、后台任务、审计记录共享同一 correlation ID / task ID 体系
- 必须把 `structured plan` 作为 CLI、TUI、API、IDE、CI 之间的共同交换格式
- 必须允许 operator 在终端内完成：
  - 查看
  - 理解
  - 审批
  - 执行
  - 接管
  - 审计
  - 报告导出
- 必须避免“all in terminal” 退化成“所有东西都塞进单一聊天框”的反模式

项目类型分析的最终结论是：Gitdex 的 CLI 属性是真实的，但它的设计标准应该对齐“终端内的受治理仓库控制台”，而不是传统的命令集合工具。这一结论将直接影响后续功能需求、系统架构、测试策略和 MVP 切分方式。

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** `Trust-first problem-solving MVP with experience-led terminal entry`

Gitdex 的首发版本不应被定义为“功能最少”，而应被定义为“最早达到可托管、可解释、可接管、可持续使用门槛的版本”。因此，Phase 1 的核心不是把所有能力一次性做完，而是先证明三个事实：

- 用户愿意把 Gitdex 作为默认终端入口，而不是偶尔试用的附加工具。
- 用户可以在不离开终端的情况下，完成一组真实高频、低到中风险、具备持续价值的仓库维护闭环。
- 用户愿意在看到计划、审计、审批、暂停与接管机制都可靠的前提下，逐步授权更深层自动化。

这使 Gitdex 的 MVP 更接近 `trust-first + experience-led + platform-backed` 的组合型 MVP，而不是单纯的功能验证版或平台底座版。质量优先意味着首发范围必须足够完整到能被真实使用，但也必须足够克制到能被真正做深、做稳、做可信。

**Resource Requirements:**  
如果只是内部 dogfood 和概念收敛，`1-2` 人可以开始；但如果要做出可持续试点的质量型 MVP，至少需要覆盖四类能力：产品与运维场景 owner、终端体验与交互设计、后台控制平面与 GitHub 集成、测试与治理/安全。资源受限时，优先保留治理链路与终端闭环，后推 deployment autonomy、企业集成和大规模 fleet。

### MVP Feature Set (Phase 1)

**Core User Journeys Supported:**

- 独立维护者把 Gitdex 作为每天打开项目后的默认终端入口。
- 开源维护者在终端内完成 repo 状态理解、PR/issue/comment 处理、上游同步建议与低风险维护执行。
- 平台/团队负责人为单仓库或小规模仓库集建立可审计的授权边界。
- 值班负责人在任务失败或阻塞时能在终端内接管、暂停、恢复和导出 handoff。
- 集成用户可以用受限 API / structured plan 方式触发和观察小规模 campaign，但仍以 operator-in-the-loop 为主。

**Must-Have Capabilities:**

- 双模终端入口必须首发成立：明确命令和自然语言聊天都可用，且共享同一治理链路。
- Repo-centric 状态装配必须首发成立：本地 Git 状态、远端仓库状态、issue/PR/comment/workflow/upstream 信息能在终端内汇总为清晰摘要。
- 所有可能产生写操作的请求都必须先落成 `structured execution plan`，并支持 `dry-run`、风险解释和审批/拒绝。
- 首发必须支持一组高频真实操作闭环：issue/PR/comment 基础读写、上游同步建议与执行、基础分支治理、基础 repo hygiene、受控的本地文件修改。
- 首发必须具备最小治理骨架：autonomy levels、policy checks、approval gates、kill switch、safe pause、handoff pack、完整审计链。
- 首发必须具备最小 daemon/backplane：schedule、queue、webhook intake、task state machine、reconciliation。
- 首发必须支持结构化输出与报告导出：终端文本、JSON/YAML、审计事件、handoff/report artifacts。
- 首发必须支持 `Windows / Linux / macOS` 三端可用，但允许 TUI 丰富度分层，不能允许核心流程仅在单一平台可用。

**MVP Boundaries:**

- Phase 1 的默认自治边界应锁在 `L0-L3`，重点是建议、计划、低风险执行和可接管自治，不默认开放高风险全自动写操作。
- Phase 1 的 deployment 能力应以“观察、解释、编排、审批衔接”为主，而不是“默认代你发布”。
- Phase 1 的 multi-repo / campaign 应限定为小规模、低风险、可分批、可人工介入的治理场景，不做大规模 fleet 平台。
- Phase 1 的 API / integration 应限定为 structured plan 提交、任务状态查询、报告导出和受限触发，不做全量公开平台 API。
- Phase 1 的 LLM 协作应围绕 repo-centric maintenance、triage、planning、context compression、project-management workload reduction，不走泛智能聊天。

**Phase-to-Capability Matrix:**

| Phase | Capability clusters | Primary FR range | Default autonomy boundary |
| --- | --- | --- | --- |
| Phase 1 | 双模终端入口、repo state summary、structured execution plan、低风险 repo operations、governed execution、small-scale campaign、audit/export | `FR1-FR22`, `FR23-FR36`, `FR37-FR50` 的首发受限子集 | `L0-L3` |
| Phase 2 | 更强 TUI、增强 triage/drafting、扩大 multi-repo governance、稳定 integration surface、共享 policy bundles | `FR18-FR22`, `FR24-FR43` 的扩展深度 | `L0-L4`，按 capability 和风险分层开放 |
| Phase 3 | fleet-level governance、enterprise controls、deep platform integrations、governed 7x24 cruising | `FR23-FR43` 的扩大范围与企业化能力 | 仅在策略、审计和信任成熟后逐步开放更深自治 |

### Post-MVP Features

**Phase 2 (Post-MVP):**

- 扩展从单仓库到多仓库日常治理，把 campaign 从“小规模 operator-in-the-loop”推进到“有策略边界的批量执行”。
- 扩展 LLM 协作到更强的 triage、comment drafting、review preparation、cross-repo context stitching 和 maintenance planning。
- 扩展终端体验，形成更成熟的 TUI：多面板、watch mode、fleet dashboard、审批队列、diff/replay 浏览。
- 扩展 integration surface：更稳定的 API、CI/agent/IDE 交换格式、structured plan 复用。
- 扩展治理配置：repo policy bundles、capability grants、risk presets、org/team defaults。
- 扩展更多 GitHub 对象与流水线编排能力，但仍坚持 plan-first 和 audit-first。

**Phase 3 (Expansion):**

- 深化 7x24 自治巡航，使 Gitdex 从“终端控制台 + 受控自治”成长为“可授权的 repository operations platform”。
- 推进更成熟的 fleet-level governance，包括跨仓库 campaign、策略 rollout、对账、分批回滚和集中观察。
- 推进企业能力：SSO/SAML、GHES、私有部署、合规导出、细粒度组织治理。
- 推进 deployment orchestration，但以前提条件为主：OIDC、environments、approval policies、evidence-backed release governance。
- 推进与 Claude Code、Codex、IDE、CI、内部 agent runtime 的深度衔接，使 Gitdex 成为开发全流程里的 repo operations backbone，而不是孤立工具。

### Risk Mitigation Strategy

**Technical Risks:**  
最大技术风险不是某个 API 能不能接，而是能否把“聊天、命令、状态机、策略、执行、审计、接管”压成同一条真实链路。MVP 的降险方式应是：单一状态机、单一 structured plan 合约、单一审计模型，避免每个入口一套逻辑。另一个高风险点是同时做本地 Git 事务、GitHub 远端副作用和 daemon 巡航，因此首发应先把低风险集合做稳，而不是过早扩展高风险动作面。

**Market Risks:**  
最大市场风险不是“大家觉得酷不酷”，而是“大家愿不愿意授权它做事”。MVP 必须先证明即使用户不授权深层自治，Gitdex 也已经是最强的终端仓库控制台。只有当用户持续把 Gitdex 当默认入口，并逐步从观察/建议模式升级到低风险执行模式，市场假设才算成立。

**Resource Risks:**  
最大资源风险是过早并行推进太多前线：终端 UX、daemon、GitHub、policy、安全、fleet、deployment。如果资源受限，最先缩掉的应是生产发布自治、企业集成、大规模 campaign 和广义 project management 平台化；最不能缩掉的是双模终端入口、repo state summary、structured plan、governed execution、audit chain 与 handoff/takeover。

## Functional Requirements

### Operator Interaction & Context Assembly

- FR1: Operators can use explicit terminal commands to access Gitdex capabilities.
- FR2: Operators can use natural-language chat in the terminal to express goals, ask questions, and request assistance.
- FR3: Operators can move between command-driven and chat-driven workflows within the same task context.
- FR4: Operators can view consolidated repository state spanning local Git, remote repository, collaboration activity, and automation status.
- FR5: Operators can request explanations of current state, material risks, and evidence-backed next actions for a selected repository, task, or campaign scope.
- FR6: Operators can inspect the evidence and source objects behind Gitdex summaries, recommendations, and decisions.

### Planning & Governed Execution

- FR7: Operators can turn commands or natural-language goals into structured execution plans before governed write actions occur.
- FR8: Operators can preview intended actions, affected objects, and risk level for a plan before execution.
- FR9: Operators can approve, reject, edit, or defer a plan when review is required.
- FR10: Operators can run supported tasks in observation, recommendation, dry-run, or execution mode.
- FR11: Gitdex can execute approved plans as tracked tasks with explicit lifecycle states.
- FR12: Gitdex can explain why a requested action is allowed, blocked, escalated, or downgraded.
- FR13: Gitdex can preserve traceability between user intent, generated plan, policy decision, execution results, and evidence.

### Repository & Collaboration Operations

- FR14: Operators can inspect and manage local repository working state, branches, diffs, and synchronization status.
- FR15: Operators can request upstream comparison, sync recommendations, and controlled synchronization actions.
- FR16: Operators can perform governed low-risk repository hygiene and maintenance tasks.
- FR17: Operators can request controlled local file modifications within an authorized repository scope.
- FR18: Operators can view issues, pull requests, comments, reviews, workflows, and deployment status from the terminal.
- FR19: Operators can create, update, and respond to supported GitHub collaboration objects from within Gitdex.
- FR20: Operators can ask Gitdex to triage, prioritize, and summarize incoming issues, pull requests, and comment activity within a defined repository or campaign scope.
- FR21: Operators can coordinate branch, PR, issue, comment, workflow, and deployment context as part of a single tracked task or structured plan.
- FR22: Operators can prepare release or deployment-related decisions through governed summaries, checks, and approval-aware workflows.

### Autonomous Operations & Task Lifecycle

- FR23: Repository owners can define autonomy levels for supported capabilities and scopes.
- FR24: Gitdex can monitor authorized repositories continuously or on schedules for explicitly supported maintenance and governance scenarios.
- FR25: Gitdex can start governed tasks from repository events, schedules, API requests, or operator requests.
- FR26: Operators can pause, resume, cancel, or take over autonomous tasks without losing task context.
- FR27: Gitdex can recover from blocked, failed, or incomplete tasks through supported retry, reconciliation, quarantine, or safe handoff paths that preserve task state and evidence.
- FR28: Gitdex can generate handoff packages for tasks that require human continuation.
- FR29: Gitdex can maintain long-running task state across terminal sessions and background processing windows until the task reaches a terminal or handoff state.

### Governance, Security & Audit

- FR30: Administrators can authorize Gitdex at repository, installation, organization, or fleet scope with bounded capabilities.
- FR31: Administrators can define policies for approvals, risk tiers, protected targets, and execution boundaries.
- FR32: Gitdex can enforce policy decisions consistently across command, chat, API, integration, and autonomous entry points.
- FR33: Gitdex can record complete audit trails for governed actions, approvals, policy evaluations, security-relevant events, and task outcomes.
- FR34: Operators and administrators can inspect audit history, evidence, and task lineage for any governed action.
- FR35: Authorized users can trigger emergency controls such as pause, capability suspension, or kill switch actions.
- FR36: Administrators can define data-handling rules for logs, caches, model use, and external integrations by scope, retention policy, and sensitivity class.

### Multi-Repository Governance & Integrations

- FR37: Operators can define and run governed campaigns across two or more repositories within an authorized repository set.
- FR38: Operators can review per-repository plans, statuses, and outcomes within a campaign.
- FR39: Operators can approve, exclude, or intervene on individual repositories within a campaign.
- FR40: Integrators can submit structured intents, plans, or tasks to Gitdex through machine-facing interfaces.
- FR41: Integrators can query task state, campaign state, reports, and audit-friendly outputs from Gitdex.
- FR42: Gitdex can exchange structured plans, results, and status with CI systems, IDEs, agent runtimes, and internal tooling.
- FR43: Administrators can apply shared policy bundles, defaults, and governance settings across defined groups of repositories within an authorized administrative scope.

### Configuration, Onboarding & Operator Enablement

- FR44: Users can complete terminal-based initial setup for identity, permissions, defaults, and operating preferences.
- FR45: Users can configure Gitdex through global, repository, session, and environment-specific settings.
- FR46: Users can select human-readable or structured output formats for supported commands, plans, reports, and task results.
- FR47: Users can discover available capabilities, command patterns, and object actions from within Gitdex.
- FR48: Users can diagnose environment, authorization, configuration, and connectivity issues from within the product.
- FR49: Users can export plans, reports, handoff packages, and other structured artifacts for reuse in external workflows.
- FR50: Users can apply Gitdex consistently across Windows, Linux, and macOS environments while preserving the same core operating model.

## Non-Functional Requirements

### Performance

- NFR1: 单仓库范围内的状态读取、摘要查看和对象查询类请求，在已具备认证与基础上下文的前提下，按正常负载下滚动 `7` 天应用遥测统计，`P95 <= 5 秒`。
- NFR2: 单仓库范围内由自然语言或命令触发的结构化计划生成，在正式支持范围内按滚动 `7` 天应用遥测与每日抽样回放统计，`90% <= 60 秒` 完成，`P95 <= 90 秒`，并以任务进入 `plan generated` 或 `plan review required` 状态作为完成判定。
- NFR3: 对已生成计划的策略评估、风险解释和 dry-run 预览，在上下文已收集完毕后按滚动 `7` 天应用遥测统计，`P95 <= 10 秒`。
- NFR4: 对正式支持的 GitHub webhook 事件类型，入站确认必须按滚动 `30` 天 delivery telemetry 统计达到 `99.9% <= 10 秒`，并通过 webhook replay/integration suite 证明最慢路径仍采用异步后处理模式。
- NFR5: 活跃任务状态刷新、阻塞原因查询和最近一次状态转换查询，在正常负载下按滚动 `7` 天应用遥测统计 `P95 <= 5 秒`。

### Reliability & Recoverability

- NFR6: 在正式支持的低风险与中风险操作集合中，按滚动 `30` 天统计窗口计算，分母为已进入执行阶段的受支持任务且排除用户主动取消与策略阻止项，任务最终成功收敛率必须 `>= 95%`。
- NFR7: `100%` 的任务必须最终收敛到以下状态之一：`succeeded`、`cancelled`、`failed with handoff complete`、`blocked by policy/approval`；该要求必须通过滚动 `30` 天任务状态对账作业与日度审计扫描证明。
- NFR8: 对于进入失败、阻塞或需人工接管状态的任务，`handoff pack` 必须在 `60 秒` 内可用；该目标必须通过失败路径集成测试和滚动 `30` 天异常任务遥测统计验证。
- NFR9: 对于排队中或可中断阶段的任务，`safe pause`、`cancel`、`kill switch` 指令必须在 `30 秒` 内生效；若外部副作用已提交，系统必须显式转入 containment/handoff 状态。该要求必须通过控制面时延遥测与中断演练验证。
- NFR10: 对 webhook 丢失、重复、乱序或部分处理失败引起的状态漂移，系统必须在 `15 分钟` 内通过 reconciliation 检出并恢复可解释状态；该要求必须通过故障注入、重放演练和滚动 `30` 天事件异常遥测验证。
- NFR11: 活跃后台任务在执行期间不得出现超过 `5 分钟` 的无状态更新静默窗口，除非任务被明确标记为等待外部系统；该要求必须通过滚动 `30` 天 heartbeat telemetry 验证。

### Security & Governance

- NFR12: `100%` 的受治理写操作必须在执行前经过结构化计划生成与策略评估；该要求必须通过写路径合约测试与滚动 `30` 天审计抽样证明。
- NFR13: `100%` 的高风险动作必须满足以下之一：人工审批通过、明确策略允许自动执行、被策略阻止并留下拒绝记录；该要求必须通过滚动 `30` 天 approval/policy audit 验证。
- NFR14: 默认生产授权模式不得依赖长期 `PAT` 作为必需前提；默认机器身份必须支持短时凭证和最小权限边界。该要求必须通过基线部署检查表和发布前安全评审证明 `100%` 正式部署形态均无需长期 `PAT` 作为默认执行前提。
- NFR15: `100%` 的密钥、令牌、敏感配置和持久化凭证必须在传输中加密，在存储时受保护；该要求必须通过密钥管理检查表、传输安全测试和静态配置审计验证。
- NFR16: 严重误操作事故目标为 `0`，其中包括未授权高风险写操作、错误修改受保护目标、未受控推进高风险发布或规则变更；该目标按滚动 `90` 天事故分级台账和事后复盘记录统计。
- NFR17: `100%` 的权限提升尝试、策略绕过尝试、认证失败和紧急控制动作必须进入安全日志；该要求必须通过安全日志完整性抽样和故障注入测试验证。
- NFR18: 默认模型上下文与日志中不得暴露明文 secrets、tokens 或被显式标记为敏感的受保护内容；相关拦截覆盖率必须达到 `100%`，并通过预发布 redaction suite、样本日志扫描和敏感数据路径检查证明。

### Scalability & Capacity

- NFR19: Phase 1 必须在单租户基线部署形态下，通过持续 `30` 分钟的容量与负载测试支持至少 `50` 个活跃受管仓库，同时保持 `NFR1` 与 `NFR5` 的时延目标不失效。
- NFR20: Phase 1 必须支持单次 campaign 至少覆盖 `20` 个仓库，并保持逐仓库计划、状态、审批与结果可见性；该要求必须通过预发布 campaign acceptance run 和容量演练验证。
- NFR21: Phase 1 必须支持单租户至少 `100` 个并发追踪任务，而不丢失任务状态、审计链或相关性标识；该要求必须通过持续 `30` 分钟并发压力测试和任务完整性校验验证。
- NFR22: 当受管仓库规模从基线增长到 `10x` 时，系统不得要求改变外部命令、聊天或 structured plan 合约；增长应表现为容量扩展问题，而不是产品行为重定义问题。该要求必须通过基线与扩容场景的接口/行为回归测试验证。

### Integration & Interoperability

- NFR23: 机器可读输出、计划工件和集成接口负载必须采用版本化 schema，并通过针对前一兼容次版本消费者与生产者的合约测试证明在同一兼容版本线内保持向后兼容。
- NFR24: `100%` 的 webhook 事件必须可通过稳定标识去重，并支持安全重放而不产生重复写副作用；该要求必须通过 replay/deduplication conformance suite 与故障注入测试证明。
- NFR25: 外部集成故障不得导致任务进入不可解释状态；在故障注入测试中，`100%` 的相关失败都必须在 `30 秒` 内显式呈现为 retry、blocked、failed with handoff complete 或 equivalent governed state，并附带最近成功步骤、失败原因和下一恢复路径。
- NFR26: 对正式支持的脚本化命令集，`100%` 的核心输出必须同时提供 `human-readable text` 与 `JSON/YAML` 两种表达形式；该要求必须通过 CLI 输出合约测试和文档化命令样本验证。
- NFR27: Gitdex 输出的 structured plans、reports 和 handoff artifacts 必须可在 CLI、CI、IDE、agent runtime 和内部工具之间无损交换；对正式支持的 artifact 类型，round-trip 合约测试必须证明 `100%` 保留必填字段、标识符、状态语义与策略判定结果。

### Auditability & Observability

- NFR28: `100%` 的受治理任务、计划、审批、策略判断、执行结果和安全相关事件必须带有可追踪的 correlation ID / task ID；该要求必须通过审计抽样、事件流完整性扫描和预发布 conformance suite 验证。
- NFR29: `100%` 的受治理写操作必须保留完整审计链，至少包括触发来源、执行计划、策略判断、结果状态和关联证据；该要求必须通过滚动 `30` 天审计完整性扫描和写路径抽样检查验证。
- NFR30: 操作者必须能在正常负载下按滚动 `7` 天遥测统计于 `P95 <= 5 秒` 内查到任一活跃任务的当前状态、最近一次状态转换和阻塞原因。
- NFR31: 默认部署形态下，操作日志与审计记录的保留策略必须可配置，且默认保留期不得低于 `90 天`；该要求必须通过部署配置检查和保留策略验收测试验证。
- NFR32: 对于每一次自治任务失败，系统必须输出最小证据包，且在失败演练样本中 `100%` 至少包含触发来源、作用范围、最近成功步骤、失败/阻塞步骤、相关对象标识、策略判断、关联证据和建议下一动作，以支持人工复盘、接管和责任界定。

### Portability & Terminal Compatibility

- NFR33: Phase 1 的核心闭环能力必须在 `Windows`、`Linux`、`macOS` 三端均可用，包括状态查看、计划生成、审批/拒绝、执行、接管与审计查询；该要求必须通过正式支持平台矩阵的端到端 conformance suite 验证。
- NFR34: 若 rich TUI 不可用，text-only 终端模式仍必须完整支持 `100%` 的 Phase 1 核心 operator 流程，包括状态查看、计划审查、审批/拒绝、执行启动/暂停/取消、handoff 查看与审计查询，而不是退化为只读工具；该要求必须通过 text-only regression suite 验证。
- NFR35: Gitdex 不得要求浏览器作为 Phase 1 核心维护闭环的唯一入口；所有核心运维动作必须可在终端内完成。该要求必须通过核心运维场景验收清单验证。
- NFR36: 同一命令、聊天请求和结构化输出在支持的 shell 环境中必须保持一致的核心语义；对正式支持的跨 shell 基准用例集，合约测试必须证明 `100%` 保持相同的治理结果、计划结构、退出语义和审批需求，不因 PowerShell、bash、zsh 差异而改变。
- NFR37: Phase 1 至少应为每个正式支持的操作系统提供一种原生 shell completion 或等价命令发现机制；该要求必须通过每个平台默认支持 shell 的安装与发现性验收清单验证。
