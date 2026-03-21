---
stepsCompleted: ['step-05-generate-output']
lastStep: 'step-05-generate-output'
lastSaved: '2026-03-18'
workflowType: 'testarch-test-design'
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/planning-artifacts/epics.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
  - _bmad-output/test-artifacts/test-design-progress.md
---

# Test Design for Architecture: Gitdex Phase 1 Governed Repository Operations

**Purpose:** 面向架构与后端实现团队的系统级测试设计文档，聚焦测试性缺口、架构风险与实现前必须补齐的验证前提。它是 QA 与 Engineering 之间关于“哪些系统性质必须先可测、可控、可追踪”的契约。

**Date:** 2026-03-18  
**Author:** Chika Komari  
**Status:** Architecture Review Pending  
**Project:** Gitdex  
**PRD Reference:** `_bmad-output/planning-artifacts/prd.md`  
**ADR Reference:** `_bmad-output/planning-artifacts/architecture.md`

---

## Executive Summary

**Scope:** Phase 1 的系统级测试设计，覆盖 Gitdex 的受治理写路径、GitHub 事件入口、任务状态机、审计与 handoff、multi-repo campaign、机器接口合约以及 rich TUI / text-only 终端语义一致性。

**Business Context**:

- **Impact:** Gitdex 的产品承诺不是单点自动化，而是可授权、可解释、可接管的仓库控制平面；一旦写路径、审计链或接管链失真，产品的核心价值就失效。
- **Problem:** 用户要把 Gitdex 当作默认终端入口，前提是系统在高权限场景下仍然可预测、可阻断、可复盘，而不是更快地产生混乱。
- **Release posture:** Phase 1 需要先证明 trust plane、governed execution、handoff 和 machine contracts 成立，再扩张自治深度。

**Architecture**:

- **Key Decision 1:** `terminal-first operator experience + daemon-first governed control plane`
- **Key Decision 2:** `Go + Cobra + Bubble Tea + PostgreSQL + GitHub App + git worktree`
- **Key Decision 3:** `webhook-first + async queue + reconciliation`, with `intent -> context -> structured_plan -> policy -> approval -> queue -> execute -> reconcile -> audit_close`

**Expected Scale**:

- Phase 1 baseline: `50` active repositories
- Campaign baseline: `20` repositories per governed campaign
- Concurrency baseline: `100` tracked active tasks

**Risk Summary**:

- **Total risks:** `10`
- **High-priority (`>= 6`)**: `7`
- **Blocking (`= 9`)**: `2`
- **Estimated QA design + automation planning effort:** `~20-21` core scenarios, `~2.5-4 weeks` for one QA owner

---

## Quick Guide

### BLOCKERS - Team Must Decide or Provide Before Mutative Implementation

1. **B-001 / R-001 Shared Write Gate** - 所有可写入口必须进入同一条 plan/policy/approval pipeline；不能允许 CLI、chat、API、webhook 各自绕行。建议 owner: `Architecture + Security`
2. **B-002 / R-005 Deterministic Simulation Harness** - 需要 GitHub App、webhook replay、queue timing、fake clock、fault injection 的统一测试支撑，不然 recovery / autonomy 无法做快反馈验证。建议 owner: `Platform Tooling`
3. **B-003 / R-002 Replay and Reconciliation Contract** - webhook dedup、idempotency key、replay-safe side effects、reconciliation 触发规则必须先定死。建议 owner: `Control Plane`
4. **B-004 / R-004 Audit and Handoff Primitives** - `task_id`、`correlation_id`、audit envelope、handoff schema 必须成为 shared contract，而不是后补字段。建议 owner: `Platform + Audit`
5. **B-005 / R-003 Repo/Ref Lease Model** - `single-writer-per-repo-ref` 的锁/租约模型必须可测试，否则 worktree 隔离仍可能发生 ref 污染。建议 owner: `Git Execution`

**What we need from team:** 以上事项不是优化项，而是后续可写 story 的前置条件。

---

### HIGH PRIORITY - Team Should Validate and Approve

1. **R-006 Contract Compatibility** - CLI/API/plan/report/handoff 的 versioned schema 和 round-trip conformance 应升级为 release gate。建议 owner: `API / Contracts`
2. **R-008 Campaign Isolation** - multi-repo campaign 必须以 partial success 和 per-repo intervention 为默认语义，而不是“全局成功/全局失败”。建议 owner: `Campaign / Orchestrator`
3. **R-007 Terminal Parity** - rich TUI、text-only、PowerShell、bash、zsh 的核心语义要进入 CI matrix，而不是人工冒烟。建议 owner: `CLI / UX`
4. **R-009 Degraded Auth/Rate Handling** - GitHub App token expiry、rate budget 紧张、installation drift 等失败模式必须显式进入 governed states，而不是静默重试。建议 owner: `GitHub Integration`

**What we need from team:** 这些项可以与实现并行推进，但需要在对应 epic 落地前明确验收方法。

---

### INFO ONLY - Solutions Already Chosen

1. **Test strategy split**: E2E 只负责关键跨边界语义；API/integration 承担大部分系统验证；component 聚焦解释和失败路径。
2. **Primary harnesses**: `go test`、Playwright API、schema/contract suites、Pact-compatible contract verification、k6 baseline checks。
3. **Failure philosophy**: 所有自治失败都要以 explainable state、audit lineage 和 handoff pack 收束。
4. **Companion document**: 执行顺序、资源估算和 QA recipe 全部放在 `test-design-qa.md`，避免此文档沦为测试手册。

---

## For Architects and Devs - Open Topics

### Risk Assessment

**Total risks identified:** `10` (`7` high-priority score `>= 6`, `2` medium, `1` low)

#### High-Priority Risks (Score >= 6)

| Risk ID | Category | Description | Probability | Impact | Score | Mitigation | Owner | Timeline |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| **R-001** | **SEC** | 多入口写路径在某一入口绕过 plan/policy/approval，导致未授权高风险动作执行 | 3 | 3 | **9** | 统一 governed write facade + cross-entry conformance suite | Architecture + Security | Before Stories 2.2 / 3.2 |
| **R-002** | **OPS** | webhook 重放、乱序或 dedup 缺陷导致重复副作用、卡死任务或不可解释漂移 | 3 | 3 | **9** | idempotency envelope + replay suite + reconciliation controller drills | Control Plane | Before Story 5.3 |
| **R-003** | **DATA** | isolated worktree / repo-ref lease 失效，导致共享 ref、脏工作区或错误提交目标 | 2 | 3 | **6** | per-repo-ref lease manager + disposable worktree root + protected-ref simulation | Git Execution | Before Story 2.6 |
| **R-004** | **DATA** | audit chain 与 correlation linkage 不完整，导致 handoff、追责与复盘失真 | 2 | 3 | **6** | append-only audit writer + shared IDs + schema-validated handoff fields | Platform + Audit | Before first mutative story closes |
| **R-005** | **TECH** | 缺少确定性 GitHub/Git/failure simulator，无法快速验证 recovery、autonomy 和 latency contract | 3 | 2 | **6** | local simulator + replay fixture catalog + fake clock/fault injection hooks | Platform Tooling | Before Sprint 2 implementation |
| **R-006** | **TECH** | CLI/API/artifact schema 漂移导致 CI/IDE/agent/runtime 集成不兼容 | 2 | 3 | **6** | versioned schemas + round-trip contract tests + compatibility fixtures | API / Contracts | Before Story 7.1 beta |
| **R-008** | **OPS** | campaign 未保证 per-repo isolation / intervention，导致多仓库 blast radius 放大 | 2 | 3 | **6** | per-repo wave model + partial-success invariants + per-row approval/retry semantics | Campaign / Orchestrator | Before Story 6.3 |

#### Medium-Priority Risks (Score 3-5)

| Risk ID | Category | Description | Probability | Impact | Score | Mitigation | Owner |
| --- | --- | --- | --- | --- | --- | --- | --- |
| R-007 | BUS | rich TUI、text-only、cross-shell 输出语义漂移，降低 operator trust | 2 | 2 | 4 | parity matrix + text-only regression suite | CLI / UX |
| R-009 | OPS | GitHub App token、installation scope、rate budget 异常只表现为“慢”而非 explainable state | 2 | 2 | 4 | synthetic auth/rate faults + explicit degraded states | GitHub Integration |

#### Low-Priority Risks (Score 1-2)

| Risk ID | Category | Description | Probability | Impact | Score | Action |
| --- | --- | --- | --- | --- | --- | --- |
| R-010 | BUS | LLM explanation 文字质量与 evidence basis 偶发偏移，影响信心但不直接越权 | 1 | 2 | 2 | Monitor via explanation review set |

#### Risk Category Legend

- **TECH**: 架构、边界、接口设计缺陷
- **SEC**: 越权、授权绕过、敏感路径失控
- **PERF**: 延迟、吞吐、资源耗尽
- **DATA**: 任务状态、审计链、仓库内容一致性问题
- **BUS**: 用户信任、产品承诺、关键体验失真
- **OPS**: webhook、部署、调度、可运行性和恢复问题

---

### Testability Concerns and Architectural Gaps

#### ACTIONABLE CONCERNS

##### 1. Blockers to Fast Feedback

| Concern | Impact | What Architecture Must Provide | Owner | Timeline |
| --- | --- | --- | --- | --- |
| **No deterministic GitHub/control-plane simulator** | 无法并行验证 webhook replay、rate limit、token drift、reconciliation | GitHub App stub, webhook fixture loader, fake clock, queue driver, fault injection seam | Platform Tooling | Before Story 5.3 |
| **No shared governed write facade** | 无法证明 CLI/chat/API/webhook 入口的 policy parity | one mutative facade and mandatory write-path contract tests | Architecture | Before Story 2.2 |
| **No shared audit/handoff envelope** | handoff completeness、traceability、postmortem replay 无法断言 | shared schema for task, audit event, handoff pack, evidence refs | Platform + Audit | Before Story 3.3 |
| **No repo/ref lease contract** | 并发写测试不稳定，无法验证 `single-writer-per-repo-ref` | lease/lock service with deterministic timeout and ownership semantics | Git Execution | Before Story 2.6 |
| **No contract registry / compatibility fixtures** | machine API 与 artifact drift 无法提前暴露 | versioned schema directory, compatibility fixtures, round-trip suite | API / Contracts | Before Story 7.1 |

##### 2. Architectural Improvements Needed

1. **把模拟器做成正式模块，而不是测试私货**
   - **Current problem**: 目前文档定义了行为，但没有为 GitHub、scheduler、webhook、rate budget、clock 提供统一模拟入口。
   - **Required change**: 在 control-plane facade 旁边引入 deterministic test support package，暴露 provider states、replay fixtures、fault knobs。
   - **Impact if not fixed**: recovery、autonomy、NFR timing 只能依赖慢且脆弱的 end-to-end 环境。
   - **Owner**: Platform Tooling
   - **Timeline**: Pre-implementation foundation

2. **把 idempotency 和 replay-safe side effects 提升为 envelope-level contract**
   - **Current problem**: 文档强调 webhook-first 和 replay-safe，但实现约束如果只留给 handler 约定，极易漂移。
   - **Required change**: 为 event envelope、task envelope、execution effect records 定义稳定 idempotency keys 和 duplicate semantics。
   - **Impact if not fixed**: `R-002` 只能在线上或 staging 被动暴露。
   - **Owner**: Control Plane
   - **Timeline**: Before event-driven stories

3. **让 audit 和 handoff 直接复用 shared contracts**
   - **Current problem**: 如果 audit/handoff 由不同模块分别拼装，很难保证字段一致与 lineage completeness。
   - **Required change**: 从 shared contract source 生成 audit event schema 与 handoff pack schema，并在 write path 强制填充必需字段。
   - **Impact if not fixed**: `NFR28-NFR32` 将无法成为硬门。
   - **Owner**: Platform + Audit
   - **Timeline**: Before first mutative story

4. **把 read model 与 effect execution 的断言边界分开**
   - **Current problem**: 如果测试只能观察最终 UI，而不能观察任务状态与投影更新，就会迫使 E2E 覆盖膨胀。
   - **Required change**: 为 task state、repo projection、campaign projection、audit close 提供稳定查询接口。
   - **Impact if not fixed**: 覆盖将重复，且失败定位成本高。
   - **Owner**: Architecture + API
   - **Timeline**: Before Story 7.2

##### 3. ASRs

- **ACTIONABLE**
  - `ASR-A1`: 所有写操作必须穿过同一 governed write pipeline
  - `ASR-A2`: ingress event 必须 replay-safe、deduplicated、idempotent
  - `ASR-A3`: mutative Git work 必须使用 isolated worktree + repo/ref lease
  - `ASR-A4`: audit ledger 与 handoff artifacts 必须具备完整 lineage
  - `ASR-A5`: CLI/API/plan/report/handoff contracts 必须版本化且可 round-trip
- **FYI**
  - `ASR-F1`: GraphQL-first reads / REST-first writes 不改变测试级别划分
  - `ASR-F2`: Deployment 在 Phase 1 是 governance adapter，不是新的 CD engine
  - `ASR-F3`: rich TUI 与 text-only 共享语义，不要求像素级一致

---

### Testability Assessment Summary

#### What Works Well

- 已有显式任务状态机和 campaign 状态机，适合做 deterministic transition assertions。
- control plane、execution plane、policy、audit 的边界已经足够清晰，便于分配 unit/API/E2E 职责。
- audit record minimum fields 与 handoff pack minimum fields 已被预定义，说明 traceability 目标不是事后补丁。
- 项目结构、schema 位置和 naming conventions 已经固定，适合直接挂接 conformance suites。

#### Accepted Trade-offs (No Action Required)

- **Phase 1 不直接控制云资源** - 当前只验证 deployment governance adapter，是正确收敛，不需要为“直接发版”预先扩张测试面。
- **单租户优先** - 在 enterprise self-hosted 之前先验证 local workstation / single-tenant team mode，可接受。

---

### Risk Mitigation Plans (High-Priority Risks >= 6)

#### R-001: Multi-entry Policy Bypass (Score: 9) - CRITICAL

**Mitigation Strategy:**

1. 建立单一 mutative facade，所有 CLI/chat/API/webhook/schedule 写路径只可调用这一入口。
2. 为每个入口生成 allow / block / escalate 三类 conformance fixtures。
3. 在 CI 中加入 cross-entry parity suite，禁止未经过 plan/policy/approval 的写请求合并。

**Owner:** Architecture + Security  
**Timeline:** Before Stories 2.2 / 3.2  
**Status:** Planned  
**Verification:** cross-entry governed write conformance suite `100%` pass

#### R-002: Replay / Dedup / Reconciliation Drift (Score: 9) - CRITICAL

**Mitigation Strategy:**

1. 为 ingress event、task dispatch 和 execution effect 定义稳定 idempotency keys。
2. 构建 duplicate / out-of-order / missing webhook replay fixtures。
3. 在 reconciliation suite 中验证“最终 explainable state”而非仅验证“最终成功”。

**Owner:** Control Plane  
**Timeline:** Before Story 5.3  
**Status:** Planned  
**Verification:** replay/dedup/reconciliation suite on duplicate, reordered, and missing events

#### R-003: Worktree / Lease Isolation Failure (Score: 6) - HIGH

**Mitigation Strategy:**

1. 为 repo/ref lease service 定义 acquire / renew / release / timeout semantics。
2. 所有 mutative Git integration tests 默认使用 disposable worktree root。
3. 在 protected target fixtures 上验证 concurrent writes are denied or serialized.

**Owner:** Git Execution  
**Timeline:** Before Story 2.6  
**Status:** Planned  
**Verification:** concurrent repo/ref isolation suite with protected target fixtures

#### R-004: Audit and Handoff Incompleteness (Score: 6) - HIGH

**Mitigation Strategy:**

1. 将 `task_id`、`correlation_id`、`plan_hash`、`policy_result`、`approval_state` 设为 shared required fields。
2. handoff pack 与 audit event 由 shared schemas 驱动，不允许模块私有字段替代核心字段。
3. 在 every-write-path suite 中强制校验 lineage completeness。

**Owner:** Platform + Audit  
**Timeline:** Before first mutative story closes  
**Status:** Planned  
**Verification:** audit lineage conformance and handoff minimum-field suite

#### R-005: Missing Deterministic Simulation Harness (Score: 6) - HIGH

**Mitigation Strategy:**

1. 交付 local GitHub sandbox、webhook replay driver、fake clock 和 fault injection knobs。
2. 将 simulator 纳入 foundation story，而不是待到 Epic 5 再补。
3. 把 recovery、rate-limit、auth drift、timeout 都变成可重放 provider states。

**Owner:** Platform Tooling  
**Timeline:** Before Sprint 2 implementation  
**Status:** Planned  
**Verification:** deterministic replay suite running in PR without external SaaS dependency

#### R-006: Contract Drift Across CLI/API/Artifacts (Score: 6) - HIGH

**Mitigation Strategy:**

1. 在 `schema/` 下维护 versioned contracts for task, plan, audit, campaign, handoff.
2. 对 CLI JSON/YAML, HTTP API payloads, and exported artifacts 执行 round-trip compatibility tests。
3. 将 compatibility fixtures 与 previous-compatible consumer payloads 一并纳入 repo。

**Owner:** API / Contracts  
**Timeline:** Before Story 7.1 beta  
**Status:** Planned  
**Verification:** round-trip contract suite with previous-compatible fixtures

#### R-008: Campaign Blast Radius Without Per-Repo Isolation (Score: 6) - HIGH

**Mitigation Strategy:**

1. 将 per-repo plan/state/approval/intervention 设为 campaign 核心语义，而不是附加 UI。
2. 在 campaign execution model 中显式支持 partial success、exclude、retry、takeover。
3. 以 multi-repo acceptance run 验证单仓库失败不会污染整轮 campaign。

**Owner:** Campaign / Orchestrator  
**Timeline:** Before Story 6.3  
**Status:** Planned  
**Verification:** campaign matrix acceptance suite with partial-success scenarios

---

### Assumptions and Dependencies

#### Assumptions

1. GitHub App sandbox organization / repositories 可用于非生产 replay 与 auth tests。
2. Postgres 将作为本地与 CI 的 durable state baseline，而不是被 mock 掉的临时依赖。
3. Shared schemas 会在实现早期落盘到仓库中，而不是仅存在于代码结构体。
4. Phase 1 的多仓库 campaign 规模按 PRD/Architecture 的上限控制，不追求提前扩容到 enterprise fleet。

#### Dependencies

1. Fixture repository catalog 与 disposable worktree root - required before Story 2.6
2. GitHub webhook replay fixtures 与 simulator secrets - required before Story 5.3
3. Multi-OS CI runners (`Windows`, `Linux`, `macOS`) - required before Story 1.5 release hardening
4. Contract registry / schema compatibility fixture store - required before Story 7.1 beta

#### Risks to Plan

- **Risk:** simulator build-out 被推迟到 autonomy epic 之后  
  - **Impact:** 关键 recovery / replay / drift 行为只能在慢环境验证  
  - **Contingency:** 把 simulator/fault hooks 提升为 foundation backlog item

- **Risk:** shared contracts 只在实现中隐含存在，未形成独立 schema  
  - **Impact:** API、CLI、artifact drift 难以及时发现  
  - **Contingency:** 强制 schema files 与 code 同步提交

---

**End of Architecture Document**

**Next Steps for Architecture Team:**

1. 先解决 `B-001` 到 `B-005`，再推进可写 story 的实现顺序。
2. 把 high-risk mitigations 明确绑定到对应 epic/story 的 Definition of Done。
3. 确认 simulator、schema registry、audit primitives 属于基础设施，不是 QA 自行补洞。
