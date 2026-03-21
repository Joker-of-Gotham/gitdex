---
stepsCompleted: [1, 2, 3, 4, 5, 6]
inputDocuments:
  - ../brainstorming/brainstorming-session-20260318-152000.md
  - ./research/domain-repository-autonomous-operations-research-2026-03-18.md
  - ./research/technical-gitdex-architecture-directions-research-2026-03-18.md
  - ./research/market-gitdex-competitive-boundaries-and-trust-models-research-2026-03-18.md
date: 2026-03-18
author: Chika Komari
---

# Product Brief: Gitdex

## Executive Summary

Gitdex 是一个面向终端环境的、受治理的仓库自治控制平面。它的目标不是成为另一个“会调 GitHub API 的 bot”，而是让维护者、平台团队和工程负责人能够把高频、重复、可标准化的仓库运维工作安全地托管给系统，同时保留明确的权限边界、审批机制、审计证据和人工接管能力。

Gitdex 要解决的核心问题，不是“仓库里缺自动化”，而是“现有自动化太分散、太脆弱、太难授权、太难审计”。今天的仓库运维通常散落在脚本、GitHub Actions、评论命令、平台工具、人工流程和值班经验里。能力看似很多，但真正跨本地文件修正、Git 事务、PR/issue/comment/action/deployment 编排的长期自治系统很少，因为一旦权限拉高，信任成本、审批成本和恢复成本会迅速上升。

Gitdex 的产品方向因此必须非常明确：它不是全自动万能 bot，而是一个把 `policy-as-code`、`dry-run`、`state machine`、`audit ledger`、`handoff pack` 和 `GitHub-native trust model` 产品化的自治平台。它先交付“可授权的自治”，再逐步扩展“更强的自治”。这会成为后续所有 PRD、架构和实现决策的上位约束。

---

## Core Vision

### Problem Statement

现代仓库管理已经不再只是代码托管问题，而是一个持续运行的控制问题。维护者和工程团队需要同时处理本地修复、分支与 PR 流程、issue triage、评论指令、工作流编排、部署门禁、规则漂移、安全修复和审计追溯，但现有工具链通常只覆盖其中某一个切面。结果是：

- 低风险高频工作长期靠人工，吞噬维护者时间。
- 高权限自动化缺乏统一的权限模型、审批链和恢复模型，组织不敢授权。
- Git、GitHub、CI/CD 和部署系统之间缺少统一状态视图，部分失败后难以对账和接管。
- 即便已有原生能力和脚本，团队仍然很难把这些能力组合成一个可 7x24 托管、可证明安全的自治系统。

### Problem Impact

这个问题的影响主要体现在四个层面：

- 工程效率层面：review 等待、仓库卫生、批量维护、夜间值守和重复治理工作不断累积。
- 治理层面：权限膨胀、规则漂移、环境门禁失配、审计材料分散，导致组织无法放心开放自动化权限。
- 可靠性层面：部分成功、重试、冲突、限流、回滚和人工接管缺乏统一模型，自动化一旦出错，恢复成本很高。
- 市场接受层面：功能再全的系统，只要默认边界不清、数据处理不透明、缺少审批与证据链，就很难通过安全和采购评审。

### Why Existing Solutions Fall Short

现有方案并非没有价值，而是边界各自成立、整体仍然断裂：

- GitHub 原生能力提供了大量治理原语，但缺少一个统一的跨表面控制平面来组织 repo、issue、PR、workflow、deployment 和审计证据。
- Mergify、Graphite 这类工具在 merge/review throughput 上很强，但并不试图成为完整的仓库自治控制层。
- Renovate、Dependabot 这类 bot 因边界足够窄而容易被接受，但能力范围天然有限。
- CodeRabbit、Copilot coding agent 等 AI 产品强调 PR 或单仓库边界内的受控辅助，而不是高权限、跨对象、长期自治。
- 自建脚本和 Actions 虽然灵活，但组织成本高、策略分散、可追溯性弱，一旦规模扩大就会变成不可维护的自动化拼贴。

市场已经证明“自动化”本身有需求，但也证明了另一个事实：越高权限、越跨表面、越长期运行的系统，就越需要控制平面式的治理设计。

### Proposed Solution

Gitdex 提供一个 terminal-first、daemon-first 的受治理自治平台，把仓库运维抽象为可计划、可模拟、可审批、可执行、可对账、可接管的工作流。它在产品上统一以下能力：

- 本地 Git 执行层：以隔离工作区和 Git 事务为基础进行文件修正、分支编排、补丁生成、回滚准备和证据采集。
- GitHub 编排层：统一 issue、PR、comment、workflow、deployment、rules 和 environment 相关动作。
- 治理层：以 `GitHub App first`、`capability grants`、`policy-as-code`、`approval gates`、`autonomy levels` 为核心的权限与策略体系。
- 运维与接管层：通过 `audit ledger`、`handoff pack`、重试/补偿、熔断和 operator TUI，让系统在长期运行时仍然可控。

Gitdex 的首要目标不是立即替用户自动做完所有事情，而是把“哪些事情可以被安全托管、在什么边界内托管、出了问题如何停下并交还给人”这件事做对。

### Key Differentiators

- `Authorizable autonomy` 而不是最大自治：Gitdex 的核心卖点是可授权、可治理、可追溯，而不是无边界自动化。
- `Control plane` 而不是单点 bot：它统一本地 Git、GitHub 表面能力、审批和审计，而不是只做一个表面。
- `Trust plane first`：产品从第一天就把策略、权限、审计、接管和恢复当作主能力，而不是事后补的企业功能。
- `Terminal-native operator experience`：既适合维护者和平台团队在终端中高密度操作，又具备后台持续运行能力。
- `GitHub-native trust model`：以 GitHub App、repo 安装边界、environment gate、rulesets、audit log 为基础，与组织现有治理体系对齐。
- `Simulation before mutation`：默认先 dry-run、再审批、后执行，把高风险操作从黑箱行为变成可解释行为。

## Target Users

### Primary Users

#### 1. 独立维护者 / 小型仓库主理人

**代表画像：** 林川，独立开发者或小型产品维护者，维护 3-10 个仓库，既写代码也处理 issue、PR、依赖升级和发布。

**工作环境：**

- 终端是主工作台。
- 很多流程靠记忆、别名脚本和 GitHub 原生功能拼接。
- 经常在非工作时段处理维护与发布问题。

**核心动机：**

- 减少机械性维护和夜间看护。
- 用最低认知成本保持仓库整洁、可发布、可协作。
- 在不牺牲控制感的前提下，把重复工作托管出去。

**当前痛点：**

- issue triage、标签治理、stale 清理、依赖升级和 release hygiene 非常耗时。
- 自动化一旦写坏，自己就是最后的接锅人，所以对高权限 bot 天然谨慎。
- 缺少一个能统一本地 Git 修正和 GitHub 运维动作的终端工具。

**成功标准：**

- 一周内明显减少重复维护工作。
- 需要人工介入时，系统能给出清晰的 diff、理由和接管入口。
- 能在不牺牲仓库安全感的情况下，让 Gitdex 长期托管低风险工作。

#### 2. 开源维护者 / 社区项目协作者

**代表画像：** 阿青，中型开源项目维护者，维护一个有活跃 issue 和 PR 流量的社区仓库，最在意贡献者体验和协作秩序。

**工作环境：**

- 日常在 GitHub issue、PR、评论和 release 流程中来回切换。
- 贡献者质量参差不齐，很多工作是分类、提醒、路由和重复解释。
- 希望自动化提高响应速度，但不能伤害社区信任。

**核心动机：**

- 让 issue/PR 流更快、更清晰、更少噪音。
- 维护一致的协作规范，而不是靠个人记忆管理社区。
- 在不显得“生硬”或“越权”的前提下让机器人承担例行工作。

**当前痛点：**

- 重复 issue、陈旧 PR、标签混乱和 review 路由占据大量精力。
- 现有 bot 通常只覆盖单一问题，组合后维护成本反而增加。
- 任何自动化越权或误操作都会直接影响社区感知和项目声誉。

**成功标准：**

- issue/PR 分类和流转明显更有秩序。
- 贡献者能从评论、PR 描述和自动化反馈中得到更清楚的下一步。
- Gitdex 被视作项目治理的一部分，而不是噪音制造者。

#### 3. 初创团队工程负责人 / 平台工程师

**代表画像：** 周衡，20-150 人团队的 CTO、Eng Lead 或平台工程师，管理多个服务仓库，希望减少 repo fleet 的治理与运维成本。

**工作环境：**

- 同时关心 PR throughput、分支策略、Actions、环境门禁、依赖修复、批量规则更新和夜间值守。
- 已经有一定 GitHub 原生能力和脚本，但系统分散，缺少统一控制。
- 需要向团队和管理层证明自动化是安全的、可审计的。

**核心动机：**

- 把低风险高频治理工作变成标准化、可批量执行的流程。
- 在不牺牲合规与恢复能力的情况下，把部分仓库运维托管给系统。
- 为未来的 deployment governance 和更强自治打基础。

**当前痛点：**

- 多仓库规则漂移、维护债务和 remediation campaign 很难统一推进。
- 现有脚本与 Actions 缺少审批、证据链和明确的 blast radius 管理。
- 团队需要的是可治理的 control plane，而不是更难解释的万能 bot。

**成功标准：**

- 可以按 repo、capability、风险等级逐步开放自动化。
- 批量维护、治理和安全修复可以标准化运行并保留审计证据。
- Gitdex 能成为团队现有 GitHub 治理能力的上层编排器，而不是另起炉灶。

### Secondary Users

#### 1. 仓库所有者 / 审批者

- 负责批准 GitHub App 安装、关键 capability 和高风险动作。
- 更关心权限边界、审批成本、误操作恢复和最终责任归属。

#### 2. 安全与合规负责人

- 关注数据处理、凭证模型、审计链、SAML/SSO、私网或自托管能力。
- 不是日常操作者，但往往决定 Gitdex 能否被组织正式授权。

#### 3. 发布经理 / 值班负责人

- 主要参与 deployment gate、冻结窗口、回滚与 incident handoff。
- 对系统的要求是“关键时刻不添乱，且能把上下文交完整”。

#### 4. 贡献者与普通开发者

- 不是主要配置者，但会直接体验 Gitdex 生成的 PR 描述、评论引导、任务流转和自动化反馈。
- 他们的体验决定 Gitdex 是否被视为高质量协作基础设施。

### User Journey

#### 主要旅程 A：独立维护者从“试用”到“托管低风险工作”

1. **发现**
   - 用户因 issue/PR 维护负担、依赖升级和仓库卫生问题开始寻找工具。
   - 被 Gitdex 的“终端友好 + 可治理自动化”定位吸引，而不是单点 bot 功能。
2. **首次上手**
   - 通过 CLI 连接 GitHub App，选择单个仓库安装。
   - 先运行只读扫描和 dry-run，查看仓库风险画像、建议动作和可启用能力。
3. **首次价值时刻**
   - Gitdex 成功自动完成 triage、stale 清理、标签整顿或一批低风险 maintenance PR。
   - 用户感受到“系统确实省时间，而且没有乱动仓库”。
4. **常规使用**
   - 用户逐步开放更多低风险 capability。
   - 通过 TUI 或评论命令查看计划、审批、证据和待接管任务。
5. **长期融入**
   - Gitdex 成为夜间运维和重复维护的托管层，用户只在高风险动作和异常场景下介入。

#### 主要旅程 B：工程负责人从“治理试点”到“多仓库编排”

1. **发现**
   - 团队在多仓库维护、规则漂移、批量修复和审计追踪上感到失控。
   - Gitdex 以“受治理的仓库控制平面”而不是“全自动 agent”进入评估清单。
2. **评估与授权**
   - 先由平台或安全团队评估 GitHub App 权限、日志能力、审批门和部署模型。
   - 以 installation-scoped 或 self-hosted 模式先做有限试点。
3. **首次价值时刻**
   - Gitdex 成功执行一次跨仓库 maintenance campaign 或规则整顿，并生成完整证据包。
   - 团队看到自动化不只是执行了动作，还降低了审批和对账成本。
4. **常规使用**
   - 通过 policy bundles 和 autonomy levels 按仓库逐步扩权。
   - TUI / API 成为平台团队观察队列、审批、重试和接管的日常入口。
5. **长期融入**
   - Gitdex 成为 repo governance 与 safe maintenance 的标准控制面。
   - 后续才逐步扩展到 deployment governance 和更高自治等级。

## Success Metrics

Gitdex 的成功不能只看“自动化次数”，必须同时满足三类结果：用户确实减少了维护负担，组织确实愿意授权更多能力，系统本身确实保持了可控、可审计和可恢复。

### 用户成功指标

- **低风险托管价值**
  - 激活后的 30 天内，至少 70% 的试点仓库成功启用并持续使用 2 项以上低风险自治能力。
- **维护效率改善**
  - 对试点用户，issue triage、标签治理、stale 清理、依赖维护等重复工作的手动处理时间降低 40% 以上。
- **首次价值时间**
  - 新用户从安装到完成第一次只读扫描与第一次安全自动化执行的中位时间不超过 1 天。
- **长期保留**
  - 已激活仓库在第 8 周仍有 60% 以上保持每周至少一次 Gitdex 管理活动。
- **接管体验**
  - 进入人工接管的任务中，80% 以上能在 handoff pack 基础上于 15 分钟内定位当前状态和下一步动作。

### 信任与控制成功指标

- **授权扩张**
  - 试点仓库在前 90 天内，从 L0-L2 逐步升级到至少一个 L3 能力的比例达到 50% 以上。
- **安全默认值有效**
  - 所有高风险动作在进入执行前均有明确的 capability check、policy evaluation 和审批/拒绝记录，覆盖率达到 100%。
- **可恢复性**
  - 发生失败的自治任务中，90% 以上可以通过自动补偿、重试或人工接管在既定流程内完成收束，不产生失控悬挂状态。
- **误操作控制**
  - 因 Gitdex 导致的高严重度生产事故目标为 0；因权限越界触发的事件目标为 0。
- **可审计性**
  - 所有写操作都能关联到 correlation id、执行计划、审批记录和结果证据，审计链完整率达到 100%。

### Business Objectives

#### 3 个月目标

- 验证 Gitdex 作为 `governed repository control plane` 的定位是否比“万能 bot”更容易被用户理解和接受。
- 在首发目标人群中完成一批高质量试点，证明低风险自治与终端 operator 体验的结合有真实价值。
- 把 trust plane 相关能力做成非可选基础设施，而不是后补特性。

#### 12 个月目标

- 在独立维护者、开源维护者和初创团队工程负责人中建立清晰的类别认知：Gitdex 是“可授权的仓库自治平台”。
- 形成以 repo governance、safe maintenance、PR/issue/action orchestration 为核心的稳定产品楔子。
- 为后续 PRD 和架构路线验证：installation-scoped 部署、policy bundles、operator TUI、handoff pack、deployment governance 等方向是否可扩张。

#### 商业与战略目标

- 建立一套足以通过中小团队与早期企业安全评审的默认信任模型。
- 证明 Gitdex 的商业价值来自“减少治理和维护成本、降低授权阻力、提高可恢复性”，而不是单纯堆叠自动化能力。
- 为后续 enterprise 版本预留清晰升级路径：SSO、审计导出、私网/自托管、更细 capability grants、更强审批链。

### Key Performance Indicators

#### 采用与激活

- **试点仓库激活率**：安装后 14 天内完成扫描、策略配置并执行至少一次低风险任务的仓库占比。
- **周活跃管理仓库数**：每周至少发生一次 Gitdex 管理活动的仓库数量。
- **能力启用深度**：每个已激活仓库平均启用的 capability 数量及其 autonomy level 分布。

#### 用户价值

- **每仓库每周节省的手动维护时间**：通过任务前后对比估算。
- **首次成功自治时间**：从安装到首次成功完成可逆低风险自动化的中位时长。
- **重复任务自动处理率**：目标场景中由 Gitdex 完成而非人工手动完成的任务占比。

#### 信任与风险

- **高风险动作审批覆盖率**：应为 100%。
- **策略拒绝与降级命中率**：反映 trust plane 是否真实发挥作用，而不是沦为空壳。
- **补偿 / 回滚成功率**：失败后成功收束的任务比例。
- **高严重度事故数**：目标为 0。
- **权限越界事件数**：目标为 0。

#### 系统运营

- **任务成功收束率**：任务最终进入 `succeeded`、`cancelled`、`failed with handoff complete` 之一的比例。
- **平均人工接管时长**：从系统暂停到操作人完成判断并接管的时间。
- **队列延迟与重试率**：用于识别系统扩容、调度或速率预算问题。
- **证据链完整率**：写操作关联到完整执行证据的比例，应为 100%。

## MVP Scope

### Core Features

Gitdex 的 MVP 必须验证两个问题：第一，受治理的仓库自治是否真的能减少维护负担；第二，这种自治是否能以用户愿意授权的方式落地。因此 MVP 只应包含能证明这两个问题的最小能力集合。

#### 1. 信任底座与治理主线

- GitHub App 身份模型
- capability grants 与基础 autonomy levels
- policy evaluation 与 approval gate
- audit ledger 与 correlation id
- handoff pack、暂停、降级和 kill switch

这是 MVP 的非可选部分。没有 trust plane，Gitdex 就只是一组高风险自动化脚本。

#### 2. 终端 Operator 入口

- CLI：安装、配置、扫描、dry-run、手动触发、状态检查
- TUI：任务队列、计划预览、审批入口、失败诊断、人工接管入口

MVP 不需要一个花哨的 UI，但必须提供足够清晰的 operator 体验来承载授权、审计和接管。

#### 3. 低风险仓库治理与维护自动化

- issue / PR triage 辅助
- 标签治理与 stale/duplicate 管理
- 夜间摘要和仓库卫生检查
- 仓库规则漂移与维护债提示

这些能力 blast radius 低、价值密度高、最适合建立早期信任。

#### 4. 安全可逆的 Git 与 PR 事务层

- 隔离 worktree 执行
- patch/diff 级变更生成
- dry-run 优先
- 受控分支创建与 PR 草案生成
- 基础回滚/补偿信息记录

MVP 不需要支持所有 Git 高阶手术，但必须证明 Gitdex 可以安全地产生、解释和交付变更。

#### 5. GitHub 编排基础能力

- webhook-first 事件接入
- issue、PR、comment、workflow dispatch 的基础编排
- 任务状态机、重试与 reconciliation

这部分是 Gitdex 从终端工具升级为长期控制平面的关键。

### Out of Scope for MVP

- 默认无人审批的 production deployment 自动化
- 任意 shell 自由执行或无边界脚本代理
- force-push、历史改写、跨仓库高风险大规模变更的默认自动执行
- 多 forge / 多 VCS 平台支持
- 共享式高权限多租户 SaaS 内核
- 自我修改策略引擎、权限模型或审批逻辑
- 完整替代 GitHub Actions、CI/CD 或企业 ITSM/Secrets 平台

这些能力不是永远不做，而是不应进入首发产品承诺。

### MVP Success Criteria

MVP 成功不看功能数量，而看是否跨过以下验证门：

- **问题验证**
  - 目标用户明确表示 Gitdex 解决了他们最痛的重复仓库运维问题。
- **信任验证**
  - 用户愿意在真实仓库上持续启用低风险自治，而不仅在演示环境试用。
- **控制验证**
  - 高风险动作的审批、拒绝、暂停、接管与证据链全部闭环。
- **技术验证**
  - webhook ingest、状态机、worktree 执行、GitHub 编排和 handoff pack 形成稳定主链路。
- **扩张验证**
  - 试点用户开始要求从 L0-L2 扩展到更多 L3 场景，说明 trust plane 设计成立。

### Future Vision

如果 MVP 成功，Gitdex 后续应沿着“先扩治理，后扩权力”的方向推进，而不是直接升级成无边界 agent。

#### Post-MVP 方向

- 多仓库 maintenance campaigns 与 repo fleet 治理
- 更丰富的 Git 事务能力，如 backport、stacked PR 编排、复杂补偿流
- 更细粒度的 capability matrix 与策略包
- installation-scoped 到 enterprise-scoped 的部署演进
- deployment governance：environment-aware preflight、promotion orchestration、rollback coordination
- enterprise 级能力：SSO、审计导出、私网连接、自托管、组织级策略管理

#### 2-3 年产品形态愿景

Gitdex 应演进为一个围绕仓库生命周期运行的治理控制平面：

- 对维护者来说，它是一个可信的托管层。
- 对平台团队来说，它是一个 repo governance orchestration layer。
- 对安全与发布团队来说，它是一个可审计、可审批、可接管的操作系统。

最终形态不是“取代人”，而是把仓库管理从脚本拼接和人工值守升级为有状态、可治理、可持续扩展的自治系统。
