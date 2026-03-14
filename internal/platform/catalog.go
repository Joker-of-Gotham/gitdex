package platform

import (
	"sort"
	"strings"
)

type Capability struct {
	ID       string
	Label    string
	Category string
	DocsURL  string
	Keywords []string
}

type CapabilityProbe struct {
	CapabilityID string
	RelativePath string
}

type CapabilityPlaybook struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Category string   `json:"category"`
	DocsURL  string   `json:"docs_url,omitempty"`
	Inspect  []string `json:"inspect,omitempty"`
	Apply    []string `json:"apply,omitempty"`
	Verify   []string `json:"verify,omitempty"`
	Score    int      `json:"score,omitempty"`
}

type playbookTemplate struct {
	Inspect []string
	Apply   []string
	Verify  []string
}

var capabilityCatalog = map[Platform][]Capability{
	PlatformGitHub:    githubCapabilityList(),
	PlatformGitLab:    gitlabCapabilityList(),
	PlatformBitbucket: bitbucketCapabilityList(),
}

var capabilityProbeCatalog = map[Platform][]CapabilityProbe{
	PlatformGitHub: {
		{CapabilityID: "actions", RelativePath: "/actions/workflows"},
		{CapabilityID: "release", RelativePath: "/releases"},
		{CapabilityID: "pull_request", RelativePath: "/pulls"},
		{CapabilityID: "deployment", RelativePath: "/deployments"},
		{CapabilityID: "environments", RelativePath: "/environments"},
		{CapabilityID: "webhooks", RelativePath: "/hooks"},
		{CapabilityID: "pages", RelativePath: "/pages"},
		{CapabilityID: "rulesets", RelativePath: "/rulesets"},
		{CapabilityID: "branch_rulesets", RelativePath: "/rulesets"},
		{CapabilityID: "check_runs_failure_threshold", RelativePath: "/rulesets"},
		{CapabilityID: "security", RelativePath: "/code-scanning/default-setup"},
		{CapabilityID: "dependabot_alerts", RelativePath: "/dependabot/alerts"},
		{CapabilityID: "advanced_security", RelativePath: "/code-scanning/default-setup"},
		{CapabilityID: "dependabot_posture", RelativePath: "/automated-security-fixes"},
		{CapabilityID: "dependency_graph", RelativePath: "/dependency-graph/sbom"},
		{CapabilityID: "dependabot_security_updates", RelativePath: "/automated-security-fixes"},
		{CapabilityID: "grouped_security_updates", RelativePath: "/automated-security-fixes"},
		{CapabilityID: "dependabot_version_updates", RelativePath: "/automated-security-fixes"},
		{CapabilityID: "secret_scanning_settings", RelativePath: "/repos/{owner}/{repo}"},
		{CapabilityID: "secret_scanning_alerts", RelativePath: "/secret-scanning/alerts"},
		{CapabilityID: "code_scanning_tool_settings", RelativePath: "/code-security-configuration"},
		{CapabilityID: "code_scanning", RelativePath: "/code-scanning/alerts"},
		{CapabilityID: "code_scanning_default_setup", RelativePath: "/code-scanning/default-setup"},
		{CapabilityID: "codeql_setup", RelativePath: "/code-scanning/default-setup"},
		{CapabilityID: "codeql_analysis", RelativePath: "/code-scanning/default-setup"},
		{CapabilityID: "copilot_autofix", RelativePath: "/code-scanning/alerts"},
		{CapabilityID: "secret_protection", RelativePath: "/secret-scanning/alerts"},
		{CapabilityID: "push_protection", RelativePath: "/secret-scanning/push-protection-bypasses"},
		{CapabilityID: "actions_secrets_variables", RelativePath: "/actions/variables"},
		{CapabilityID: "actions_secrets_variables", RelativePath: "/actions/secrets"},
		{CapabilityID: "codespaces", RelativePath: "/codespaces"},
		{CapabilityID: "codespaces_secrets", RelativePath: "/codespaces/secrets"},
		{CapabilityID: "dependabot_secrets", RelativePath: "/dependabot/secrets"},
		{CapabilityID: "dependabot_config", RelativePath: "/contents/.github/dependabot.yml"},
		{CapabilityID: "pr_review", RelativePath: "/pulls"},
		{CapabilityID: "packages", RelativePath: "/packages"},
		{CapabilityID: "deploy_keys", RelativePath: "/keys"},
		{CapabilityID: "notifications", RelativePath: "/notifications"},
		{CapabilityID: "email_notifications", RelativePath: "/subscription"},
		{CapabilityID: "copilot_seat_management", RelativePath: "/copilot/billing/seats"},
	},
	PlatformBitbucket: {
		{CapabilityID: "pull_requests", RelativePath: "/pullrequests"},
		{CapabilityID: "pipelines", RelativePath: "/pipelines/"},
		{CapabilityID: "deployments", RelativePath: "/deployments_config/environments"},
		{CapabilityID: "webhooks", RelativePath: "/hooks"},
	},
	PlatformGitLab: {
		{CapabilityID: "merge_requests", RelativePath: "/merge_requests"},
		{CapabilityID: "pipelines", RelativePath: "/pipelines"},
		{CapabilityID: "environments", RelativePath: "/environments"},
		{CapabilityID: "pages", RelativePath: "/pages"},
	},
}

var categoryPlaybooks = map[string]playbookTemplate{
	"delivery": {
		Inspect: []string{
			"Inspect current delivery state, environment readiness, and existing platform settings.",
			"Collect the current workflow/deployment/Page configuration before proposing changes.",
		},
		Apply: []string{
			"Prefer declarative configuration files and idempotent API mutations over manual one-off clicks.",
			"Stage delivery changes in a reversible order: config, validation, rollout, post-checks.",
		},
		Verify: []string{
			"Verify generated workflows or deployment definitions locally before rollout.",
			"Confirm the platform surface reports the expected state after apply.",
		},
	},
	"automation": {
		Inspect: []string{
			"Check workflow definitions, permissions, triggers, and failure signals before proposing automation changes.",
		},
		Apply: []string{
			"Prefer reusable workflows and least-privilege permissions.",
			"Keep automation changes observable with explicit logs, artifacts, and rollback guidance.",
		},
		Verify: []string{
			"Verify workflow syntax, required secrets, and trigger coverage.",
		},
	},
	"security": {
		Inspect: []string{
			"Inspect the current security posture, enabled scanners, and outstanding alerts before mutating policy.",
		},
		Apply: []string{
			"Apply hardening in a layered order: visibility, detection, prevention, then enforcement.",
			"Prefer additive controls that preserve repository operability.",
		},
		Verify: []string{
			"Verify scanners, alert surfaces, and protection toggles all reflect the intended state.",
		},
	},
	"governance": {
		Inspect: []string{
			"Inspect branch policies, merge gates, and ruleset precedence before editing governance settings.",
		},
		Apply: []string{
			"Prefer explicit rulesets over ad hoc branch-only policy drift.",
		},
		Verify: []string{
			"Verify required checks, review gates, and branch targeting match repository policy.",
		},
	},
	"integration": {
		Inspect: []string{
			"Inspect existing integrations, endpoints, delivery health, and event subscriptions.",
		},
		Apply: []string{
			"Apply integration changes with clear ownership, secrets handling, and retry expectations.",
		},
		Verify: []string{
			"Verify endpoint reachability, delivery status, and event coverage after changes.",
		},
	},
	"credentials": {
		Inspect: []string{
			"Inspect secret scope, variable scope, and current credential consumers before mutation.",
		},
		Apply: []string{
			"Use the narrowest feasible credential scope and document all downstream consumers.",
		},
		Verify: []string{
			"Verify each secret/variable is referenced by an existing workflow, environment, or integration.",
		},
	},
	"collaboration": {
		Inspect: []string{
			"Inspect review state, branch readiness, and merge policy before proposing collaboration steps.",
		},
		Apply: []string{
			"Keep review actions aligned with branch policy, approval thresholds, and CI readiness.",
		},
		Verify: []string{
			"Verify review status, approvals, and branch checks align before merge or approval.",
		},
	},
	"copilot": {
		Inspect: []string{
			"Inspect Copilot feature availability, seat/license assumptions, and repository opt-in state.",
		},
		Apply: []string{
			"Keep Copilot changes aligned with repository policy, code review, and security posture.",
		},
		Verify: []string{
			"Verify the requested Copilot feature is available on the current plan and repository.",
		},
	},
	"artifacts": {
		Inspect: []string{
			"Inspect package sources, registries, and publish permissions before changing artifact flows.",
		},
		Apply: []string{
			"Keep package changes reproducible and tied to release metadata.",
		},
		Verify: []string{
			"Verify package visibility, version semantics, and publish authentication.",
		},
	},
	"devex": {
		Inspect: []string{
			"Inspect onboarding, dev environment bootstrap, and repository defaults before changing developer experience settings.",
		},
		Apply: []string{
			"Prefer reproducible dev environment configuration over manual setup instructions.",
		},
		Verify: []string{
			"Verify new contributors can bootstrap from a clean environment using the declared setup.",
		},
	},
	"notifications": {
		Inspect: []string{
			"Inspect current notification routing, critical events, and escalation requirements.",
		},
		Apply: []string{
			"Keep notifications scoped to actionable signals and avoid redundant noise.",
		},
		Verify: []string{
			"Verify critical actions still emit notifications after configuration changes.",
		},
	},
}

var capabilityPlaybookOverrides = map[string]playbookTemplate{
	"release": {
		Inspect: []string{
			"Inspect existing tags, release workflow definitions, asset build outputs, and changelog sources.",
		},
		Apply: []string{
			"Create or update release automation, asset packaging, and changelog generation in a single path.",
		},
		Verify: []string{
			"Verify tags, release notes, assets, and checksum outputs all match the target version.",
		},
	},
	"deployment": {
		Inspect: []string{
			"Inspect deployment environments, rollout gates, secrets, and runtime targets.",
		},
		Apply: []string{
			"Model deployment as environment-aware automation with prechecks and rollback steps.",
		},
		Verify: []string{
			"Verify deployment status, environment health, and post-deploy checks.",
		},
	},
	"ai_assistant_deployment": {
		Inspect: []string{
			"Inspect model/provider configuration, runtime secrets, deployment topology, and observability requirements for the AI assistant.",
		},
		Apply: []string{
			"Deploy the assistant with explicit provider settings, health checks, logs, and environment gating.",
		},
		Verify: []string{
			"Verify the assistant can reach configured model backends, secrets resolve correctly, and runtime endpoints are healthy.",
		},
	},
	"pages": {
		Inspect: []string{
			"Inspect site source path, build command, publish target, custom domain, and Pages environment state.",
		},
		Apply: []string{
			"Use a dedicated Pages build/publish workflow or supported native publishing path.",
		},
		Verify: []string{
			"Verify build output, publish source, environment, and domain/DNS readiness.",
		},
	},
	"advanced_security": {
		Inspect: []string{
			"Inspect Advanced Security feature availability, scanner status, and alert backlogs.",
		},
		Apply: []string{
			"Enable scanners and protections in a layered order: dependency graph, Dependabot, code scanning, secret scanning, then enforcement.",
		},
		Verify: []string{
			"Verify alert surfaces, scanning workflows, and protection toggles all report healthy status.",
		},
	},
	"rulesets": {
		Inspect: []string{
			"Inspect repository rulesets, branch targeting, bypass lists, and policy precedence.",
		},
		Apply: []string{
			"Prefer shared rulesets with explicit targeting rather than duplicated branch-only rules.",
		},
		Verify: []string{
			"Verify mergeability, check requirements, and bypass behavior on representative branches.",
		},
	},
	"branch_rulesets": {
		Inspect: []string{
			"Inspect resolved branch rules for the target branch before mutating branch-specific policy.",
		},
		Apply: []string{
			"Keep branch-targeted rulesets aligned with repository-wide governance instead of duplicating policy blindly.",
		},
		Verify: []string{
			"Verify the target branch resolves to the intended checks, review gates, and bypass behavior.",
		},
	},
	"actions_secrets_variables": {
		Inspect: []string{
			"Inspect Actions secrets, variables, required environments, and workflow consumers.",
		},
		Apply: []string{
			"Create or update secrets and variables with the narrowest possible scope.",
		},
		Verify: []string{
			"Verify every referenced variable/secret exists at the intended scope and is consumed by a live workflow.",
		},
	},
	"deploy_keys": {
		Inspect: []string{
			"Inspect current deploy keys, repository SSH consumers, and read-only vs write needs before rotating keys.",
		},
		Apply: []string{
			"Rotate or add deploy keys with explicit ownership and minimum required access.",
		},
		Verify: []string{
			"Verify the key exists with the intended read_only setting and downstream consumer mapping.",
		},
	},
	"codespaces_secrets": {
		Inspect: []string{
			"Inspect Codespaces bootstrap, dotfiles/devcontainer config, and required developer secrets.",
		},
		Apply: []string{
			"Scope Codespaces secrets to the required repository or organization context only.",
		},
		Verify: []string{
			"Verify a fresh Codespace can bootstrap with the configured secrets and variables.",
		},
	},
	"dependabot_secrets": {
		Inspect: []string{
			"Inspect private registry dependencies, registries config, and Dependabot credential consumers.",
		},
		Apply: []string{
			"Keep Dependabot secrets aligned with private registry definitions and update scopes.",
		},
		Verify: []string{
			"Verify Dependabot can access private registries without over-broad credentials.",
		},
	},
	"dependabot_config": {
		Inspect: []string{
			"Inspect .github/dependabot.yml, private registries, grouped updates, version-update cadence, and security-update rules before editing automation policy.",
		},
		Apply: []string{
			"Prefer updating dependabot.yml atomically so grouped/version update policy, schedules, and registries stay coherent.",
		},
		Verify: []string{
			"Verify dependabot.yml remains syntactically valid and matches the intended grouped/version update policy after mutation.",
		},
	},
	"pr_review": {
		Inspect: []string{
			"Inspect the pull request, its review state, requested reviewers, and merge blockers before taking review actions.",
		},
		Apply: []string{
			"Separate review approval, requested-changes, reviewer routing, and merge-readiness into explicit steps.",
		},
		Verify: []string{
			"Verify review state, requested reviewers, and required checks after each review mutation.",
		},
	},
	"pull_request": {
		Inspect: []string{
			"Inspect whether a pull request already exists for the target head/base pair, then inspect merge blockers, draft state, and auto-merge readiness.",
		},
		Apply: []string{
			"Keep pull request creation, metadata changes, merge, and auto-merge as explicit lifecycle steps instead of collapsing them into one opaque action.",
		},
		Verify: []string{
			"Verify pull request metadata, merge state, and auto-merge state separately after each mutation.",
		},
	},
	"webhooks": {
		Inspect: []string{
			"Inspect webhook destinations, event subscriptions, recent delivery history, and secret configuration.",
		},
		Apply: []string{
			"Keep webhook configuration idempotent and document each downstream consumer.",
		},
		Verify: []string{
			"Verify deliveries succeed and downstream consumers handle retry/idempotency safely.",
		},
	},
	"packages": {
		Inspect: []string{
			"Inspect package owner scope, package type, versions, and retention state before deletion or restore.",
		},
		Apply: []string{
			"Prefer version-scoped cleanup over deleting the entire package unless the goal explicitly requires removal.",
		},
		Verify: []string{
			"Verify the expected package or package version exists, or is absent after deletion, in the intended owner scope.",
		},
	},
	"notifications": {
		Inspect: []string{
			"Inspect whether the goal is repository watch state, thread subscription, or inbox visibility before mutating notifications.",
		},
		Apply: []string{
			"Use repository-level subscription for broad signal routing and thread-level subscription for targeted triage.",
		},
		Verify: []string{
			"Verify the resulting subscribed/ignored state on the exact repository or thread surface that was changed.",
		},
	},
	"email_notifications": {
		Inspect: []string{
			"Inspect repository and thread subscription state before proposing email-notification changes.",
		},
		Apply: []string{
			"Keep email notification changes tied to actionable repository or thread subscriptions, not vague account-wide assumptions.",
		},
		Verify: []string{
			"Verify the subscription state changed on the exact repository or thread surface requested by the goal.",
		},
	},
	"copilot_seat_management": {
		Inspect: []string{
			"Inspect current Copilot seat assignments, seat consumption, and org billing posture before assigning or removing seats.",
		},
		Apply: []string{
			"Mutate Copilot seats with explicit selected users or teams so seat ownership remains auditable.",
		},
		Verify: []string{
			"Verify the expected users or teams appear in the Copilot seat assignment surface after mutation.",
		},
	},
}

var capabilityKeywordOverrides = map[string][]string{
	"release":                   {"发布", "版本发布", "release assets", "release notes"},
	"deployment":                {"部署", "deployment", "rollout", "上线", "环境部署"},
	"ai_assistant_deployment":   {"AI 助手部署", "assistant deployment", "模型服务部署"},
	"pr_review":                 {"PR 审查", "PR 批准", "pull request 审查", "requested reviewers"},
	"actions":                   {"Actions", "工作流", "CI", "自动化"},
	"rulesets":                  {"规则集", "repo rules", "合并策略"},
	"branch_rulesets":           {"分支规则", "分支保护", "required checks"},
	"webhooks":                  {"Webhook", "事件订阅", "回调"},
	"pages":                     {"Pages", "静态站点", "自定义域名", "build history"},
	"environments":              {"环境", "environment protection", "deployment gate"},
	"actions_secrets_variables": {"Actions secrets", "Actions variables", "环境密钥", "环境变量"},
	"codespaces_secrets":        {"Codespaces secrets", "开发环境密钥"},
	"dependabot_secrets":        {"Dependabot secrets", "私有 registry", "private registry"},
	"dependabot_config":         {"dependabot.yml", "dependabot config", "grouped security updates", "version updates"},
	"deploy_keys":               {"deploy key", "部署密钥", "SSH 密钥"},
	"packages":                  {"packages", "registry", "包发布", "包版本"},
	"notifications":             {"notifications", "订阅", "watch", "thread subscription"},
	"email_notifications":       {"邮件通知", "email notifications", "notification routing"},
	"advanced_security":         {"高级安全", "GHAS", "advanced security"},
	"copilot_seat_management":   {"copilot seats", "seat management", "copilot billing seats", "copilot users"},
}

func CapabilityCatalog(p Platform) []Capability {
	return copyCapabilities(capabilityCatalog[p])
}

func CapabilityProbes(p Platform) []CapabilityProbe {
	items := capabilityProbeCatalog[p]
	if len(items) == 0 {
		return nil
	}
	out := make([]CapabilityProbe, len(items))
	copy(out, items)
	return out
}

func CapabilityIDs(p Platform) []string {
	catalog := CapabilityCatalog(p)
	out := make([]string, 0, len(catalog))
	for _, item := range catalog {
		out = append(out, item.ID)
	}
	return out
}

func CapabilityPlaybookFor(cap Capability) CapabilityPlaybook {
	playbook := CapabilityPlaybook{
		ID:       cap.ID,
		Label:    cap.Label,
		Category: cap.Category,
		DocsURL:  cap.DocsURL,
	}
	if base, ok := categoryPlaybooks[cap.Category]; ok {
		playbook.Inspect = append(playbook.Inspect, base.Inspect...)
		playbook.Apply = append(playbook.Apply, base.Apply...)
		playbook.Verify = append(playbook.Verify, base.Verify...)
	}
	if override, ok := capabilityPlaybookOverrides[cap.ID]; ok {
		playbook.Inspect = mergePlaybookSteps(playbook.Inspect, override.Inspect)
		playbook.Apply = mergePlaybookSteps(playbook.Apply, override.Apply)
		playbook.Verify = mergePlaybookSteps(playbook.Verify, override.Verify)
	}
	return playbook
}

func RecommendCapabilityPlaybooks(p Platform, goal string, limit int) []CapabilityPlaybook {
	goal = normalizeGoal(goal)
	if goal == "" {
		return nil
	}

	type scored struct {
		cap   Capability
		score int
	}

	var scoredCaps []scored
	for _, cap := range CapabilityCatalog(p) {
		score := capabilityScore(goal, cap)
		if score <= 0 {
			continue
		}
		scoredCaps = append(scoredCaps, scored{cap: cap, score: score})
	}

	sort.SliceStable(scoredCaps, func(i, j int) bool {
		if scoredCaps[i].score == scoredCaps[j].score {
			if scoredCaps[i].cap.Category == scoredCaps[j].cap.Category {
				return scoredCaps[i].cap.Label < scoredCaps[j].cap.Label
			}
			return scoredCaps[i].cap.Category < scoredCaps[j].cap.Category
		}
		return scoredCaps[i].score > scoredCaps[j].score
	})

	if limit <= 0 {
		limit = 5
	}
	if len(scoredCaps) > limit {
		scoredCaps = scoredCaps[:limit]
	}

	out := make([]CapabilityPlaybook, 0, len(scoredCaps))
	for _, item := range scoredCaps {
		playbook := CapabilityPlaybookFor(item.cap)
		playbook.Score = item.score
		out = append(out, playbook)
	}
	return out
}

func ApplyGoalRecommendations(state *PlatformState, goal string, limit int) {
	if state == nil {
		return
	}
	playbooks := RecommendCapabilityPlaybooks(ParsePlatform(state.Detected), goal, limit)
	if len(playbooks) == 0 {
		state.Playbooks = nil
		return
	}
	state.Playbooks = playbooks
	labels := make([]string, 0, len(playbooks))
	for _, playbook := range playbooks {
		labels = append(labels, playbook.ID)
	}
	state.AdminSummary = append(state.AdminSummary, "recommended="+strings.Join(labels, ","))
}

func copyCapabilities(items []Capability) []Capability {
	if len(items) == 0 {
		return nil
	}
	out := make([]Capability, len(items))
	for i, item := range items {
		out[i] = item
		if len(item.Keywords) > 0 {
			out[i].Keywords = append([]string(nil), item.Keywords...)
		}
	}
	return out
}

func mergePlaybookSteps(base, override []string) []string {
	if len(override) == 0 {
		return append([]string(nil), base...)
	}
	return append([]string(nil), override...)
}

func normalizeGoal(goal string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(goal)), " "))
}

func capabilityScore(goal string, cap Capability) int {
	score := 0
	tokens := []string{cap.ID, cap.Label, cap.Category}
	tokens = append(tokens, cap.Keywords...)
	tokens = append(tokens, capabilityKeywordOverrides[cap.ID]...)
	for _, token := range tokens {
		token = normalizeGoal(token)
		if token == "" {
			continue
		}
		switch {
		case strings.Contains(goal, token):
			score += 4
		case tokenContainsAny(token, goal):
			score++
		}
	}
	return score
}

func tokenContainsAny(token, goal string) bool {
	for _, part := range strings.FieldsFunc(token, func(r rune) bool {
		return r == ' ' || r == '_' || r == '-' || r == '/' || r == '|'
	}) {
		part = strings.TrimSpace(part)
		if len(part) < 3 {
			continue
		}
		if strings.Contains(goal, part) {
			return true
		}
	}
	return false
}

func githubCapabilityList() []Capability {
	const docsBase = "https://docs.github.com/en"
	return []Capability{
		{ID: "pull_request", Label: "Pull requests", Category: "collaboration", DocsURL: docsBase + "/rest/pulls/pulls", Keywords: []string{"pull request", "create pr", "merge pr", "auto merge", "draft pr"}},
		{ID: "release", Label: "Release management", Category: "delivery", DocsURL: docsBase + "/repositories/releasing-projects-on-github", Keywords: []string{"release", "发布", "tag", "changelog", "asset"}},
		{ID: "deployment", Label: "Deployments", Category: "delivery", DocsURL: docsBase + "/rest/deployments", Keywords: []string{"deployment", "deploy", "部署", "rollout", "environment"}},
		{ID: "ai_assistant_deployment", Label: "AI assistant deployment", Category: "delivery", DocsURL: docsBase + "/actions/deployment", Keywords: []string{"ai assistant", "assistant deployment", "部署ai", "部署 assistant", "模型服务"}},
		{ID: "pr_review", Label: "Pull request review/approval", Category: "collaboration", DocsURL: docsBase + "/pull-requests", Keywords: []string{"pull request", "pr", "review", "approval", "审查", "批准", "merge readiness"}},
		{ID: "actions", Label: "GitHub Actions", Category: "automation", DocsURL: docsBase + "/actions", Keywords: []string{"actions", "workflow", "ci", "automation", "自动化"}},
		{ID: "security", Label: "Security overview", Category: "security", DocsURL: docsBase + "/code-security", Keywords: []string{"security", "安全", "hardening", "扫描", "policy"}},
		{ID: "rulesets", Label: "Repository rulesets", Category: "governance", DocsURL: docsBase + "/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets", Keywords: []string{"ruleset", "rulesets", "规则集", "repo rules"}},
		{ID: "branch_rulesets", Label: "Branch rulesets", Category: "governance", DocsURL: docsBase + "/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets", Keywords: []string{"branch ruleset", "branch protection", "分支规则", "分支保护"}},
		{ID: "webhooks", Label: "Webhooks", Category: "integration", DocsURL: docsBase + "/webhooks", Keywords: []string{"webhook", "hooks", "集成", "delivery"}},
		{ID: "copilot_code_review", Label: "Copilot code review", Category: "copilot", DocsURL: docsBase + "/copilot", Keywords: []string{"copilot review", "code review", "copilot code review", "代码审查"}},
		{ID: "copilot_coding_agent", Label: "Copilot coding agent", Category: "copilot", DocsURL: docsBase + "/copilot", Keywords: []string{"copilot agent", "coding agent", "agent", "编程代理"}},
		{ID: "environments", Label: "Environments", Category: "delivery", DocsURL: docsBase + "/actions/deployment/targeting-different-environments", Keywords: []string{"environment", "environments", "环境", "protection rules"}},
		{ID: "packages", Label: "Packages", Category: "artifacts", DocsURL: docsBase + "/packages", Keywords: []string{"packages", "registry", "package publish", "包管理", "发布包"}},
		{ID: "codespaces", Label: "Codespaces", Category: "devex", DocsURL: docsBase + "/codespaces", Keywords: []string{"codespaces", "devcontainer", "云开发环境", "开发环境"}},
		{ID: "pages", Label: "GitHub Pages", Category: "delivery", DocsURL: docsBase + "/pages", Keywords: []string{"pages", "static site", "站点", "静态站点", "custom domain"}},
		{ID: "notifications", Label: "Notifications", Category: "notifications", DocsURL: docsBase + "/account-and-profile/managing-subscriptions-and-notifications-on-github/setting-up-notifications/configuring-notifications", Keywords: []string{"notifications", "订阅", "watch", "thread subscription"}},
		{ID: "advanced_security", Label: "Advanced Security", Category: "security", DocsURL: docsBase + "/code-security/getting-started/github-security-features", Keywords: []string{"advanced security", "高级安全", "ghas", "security suite"}},
		{ID: "dependabot_posture", Label: "Dependabot posture", Category: "security", DocsURL: docsBase + "/code-security/dependabot/dependabot-security-updates", Keywords: []string{"dependabot posture", "automated security fixes", "security updates posture"}},
		{ID: "private_vulnerability_reporting", Label: "Private vulnerability reporting", Category: "security", DocsURL: docsBase + "/code-security/security-advisories/working-with-repository-security-advisories/about-coordinated-disclosure-of-security-vulnerabilities", Keywords: []string{"private vulnerability reporting", "漏洞报告", "security advisory"}},
		{ID: "dependency_graph", Label: "Dependency graph", Category: "security", DocsURL: docsBase + "/code-security/supply-chain-security/understanding-your-software-supply-chain/about-the-dependency-graph", Keywords: []string{"dependency graph", "依赖图", "supply chain"}},
		{ID: "dependabot", Label: "Dependabot", Category: "security", DocsURL: docsBase + "/code-security/dependabot", Keywords: []string{"dependabot", "依赖更新", "security updates"}},
		{ID: "dependabot_alerts", Label: "Dependabot alerts", Category: "security", DocsURL: docsBase + "/code-security/dependabot/dependabot-alerts", Keywords: []string{"dependabot alerts", "依赖告警", "alerts"}},
		{ID: "dependabot_security_updates", Label: "Dependabot security updates", Category: "security", DocsURL: docsBase + "/code-security/dependabot/dependabot-security-updates", Keywords: []string{"security updates", "安全更新", "dependabot security"}},
		{ID: "grouped_security_updates", Label: "Grouped security updates", Category: "security", DocsURL: docsBase + "/code-security/dependabot/working-with-dependabot/grouping-dependabot-updates", Keywords: []string{"grouped updates", "grouped security updates", "分组更新"}},
		{ID: "dependabot_version_updates", Label: "Dependabot version updates", Category: "security", DocsURL: docsBase + "/code-security/dependabot/dependabot-version-updates", Keywords: []string{"version updates", "版本更新", "dependabot version"}},
		{ID: "secret_scanning_settings", Label: "Secret scanning settings", Category: "security", DocsURL: docsBase + "/code-security/secret-scanning/introduction/about-secret-scanning", Keywords: []string{"secret scanning settings", "secret scanning", "scanner posture"}},
		{ID: "secret_scanning_alerts", Label: "Secret scanning alerts", Category: "security", DocsURL: docsBase + "/code-security/secret-scanning/working-with-secret-scanning-and-push-protection/working-with-secret-scanning-alerts", Keywords: []string{"secret scanning alerts", "secret alerts", "敏感信息告警"}},
		{ID: "code_scanning", Label: "Code scanning", Category: "security", DocsURL: docsBase + "/code-security/code-scanning", Keywords: []string{"code scanning", "代码扫描", "扫描告警"}},
		{ID: "code_scanning_tool_settings", Label: "Code scanning tool settings", Category: "security", DocsURL: docsBase + "/code-security/securing-your-organization/enabling-security-features-in-your-organization/configuring-a-code-security-configuration", Keywords: []string{"code scanning tool settings", "code security configuration", "tool settings"}},
		{ID: "code_scanning_default_setup", Label: "Code scanning default setup", Category: "security", DocsURL: docsBase + "/code-security/code-scanning/automatically-scanning-your-code-for-vulnerabilities-and-errors/configuring-code-scanning", Keywords: []string{"code scanning default setup", "default setup", "default code scanning"}},
		{ID: "codeql_setup", Label: "CodeQL setup", Category: "security", DocsURL: docsBase + "/code-security/code-scanning/automatically-scanning-your-code-for-vulnerabilities-and-errors/configuring-code-scanning", Keywords: []string{"codeql setup", "codeql configuration", "codeql default setup"}},
		{ID: "codeql_analysis", Label: "CodeQL analysis", Category: "security", DocsURL: docsBase + "/code-security/code-scanning/automatically-scanning-your-code-for-vulnerabilities-and-errors/configuring-code-scanning", Keywords: []string{"codeql", "codeql analysis", "代码ql", "扫描工作流"}},
		{ID: "copilot_autofix", Label: "Copilot Autofix", Category: "security", DocsURL: docsBase + "/code-security/code-scanning/managing-code-scanning-alerts/responsible-use-autofix-code-scanning", Keywords: []string{"autofix", "copilot autofix", "自动修复"}},
		{ID: "protection_rules", Label: "Protection rules", Category: "security", DocsURL: docsBase + "/code-security/secret-scanning/introduction/about-push-protection", Keywords: []string{"protection rules", "保护规则", "push protection"}},
		{ID: "check_runs_failure_threshold", Label: "Check runs failure threshold", Category: "governance", DocsURL: docsBase + "/repositories/configuring-branches-and-merges-in-your-repository/defining-the-mergeability-of-pull-requests/about-protected-branches", Keywords: []string{"check runs", "failure threshold", "检查阈值", "required checks"}},
		{ID: "secret_protection", Label: "Secret protection", Category: "security", DocsURL: docsBase + "/code-security/secret-scanning", Keywords: []string{"secret scanning", "secret protection", "密钥保护", "敏感信息"}},
		{ID: "push_protection", Label: "Push protection", Category: "security", DocsURL: docsBase + "/code-security/secret-scanning/introduction/about-push-protection", Keywords: []string{"push protection", "推送保护", "secret push protection"}},
		{ID: "deploy_keys", Label: "Deploy keys", Category: "credentials", DocsURL: docsBase + "/authentication/connecting-to-github-with-ssh/managing-deploy-keys", Keywords: []string{"deploy keys", "部署密钥", "ssh key"}},
		{ID: "actions_secrets_variables", Label: "Actions secrets and variables", Category: "credentials", DocsURL: docsBase + "/actions/security-guides/using-secrets-in-github-actions", Keywords: []string{"actions secrets", "actions variables", "变量", "secrets", "actions credentials"}},
		{ID: "codespaces_secrets", Label: "Codespaces secrets", Category: "credentials", DocsURL: docsBase + "/codespaces/managing-your-codespaces/managing-encrypted-secrets-for-your-codespaces", Keywords: []string{"codespaces secrets", "codespaces 变量", "开发环境密钥"}},
		{ID: "dependabot_secrets", Label: "Dependabot secrets", Category: "credentials", DocsURL: docsBase + "/code-security/dependabot/working-with-dependabot/configuring-access-to-private-registries-for-dependabot", Keywords: []string{"dependabot secrets", "私有仓库凭据", "dependabot registry"}},
		{ID: "dependabot_config", Label: "Dependabot configuration", Category: "automation", DocsURL: docsBase + "/code-security/dependabot/working-with-dependabot/configuring-dependabot-version-updates", Keywords: []string{"dependabot.yml", "dependabot config", "grouped updates", "version updates"}},
		{ID: "email_notifications", Label: "Email notifications", Category: "notifications", DocsURL: docsBase + "/account-and-profile/managing-subscriptions-and-notifications-on-github/setting-up-notifications/configuring-notifications", Keywords: []string{"email notifications", "邮件通知", "notification routing"}},
		{ID: "copilot_seat_management", Label: "Copilot seat management", Category: "copilot", DocsURL: docsBase + "/rest/copilot/copilot-user-management", Keywords: []string{"copilot seats", "copilot billing seats", "seat management", "copilot users"}},
	}
}

func gitlabCapabilityList() []Capability {
	return []Capability{
		{ID: "merge_requests", Label: "Merge requests", Category: "collaboration", DocsURL: "https://docs.gitlab.com/ee/user/project/merge_requests/", Keywords: []string{"merge request", "mr", "review", "审查"}},
		{ID: "pipelines", Label: "Pipelines", Category: "automation", DocsURL: "https://docs.gitlab.com/ee/ci/", Keywords: []string{"pipelines", "ci", "automation", "流水线"}},
		{ID: "environments", Label: "Environments", Category: "delivery", DocsURL: "https://docs.gitlab.com/ee/ci/environments/", Keywords: []string{"environment", "deploy", "部署", "环境"}},
		{ID: "pages", Label: "GitLab Pages", Category: "delivery", DocsURL: "https://docs.gitlab.com/ee/user/project/pages/", Keywords: []string{"pages", "static site", "站点"}},
		{ID: "security", Label: "Security dashboard", Category: "security", DocsURL: "https://docs.gitlab.com/ee/user/application_security/", Keywords: []string{"security", "安全", "dashboard"}},
	}
}

func bitbucketCapabilityList() []Capability {
	return []Capability{
		{ID: "pull_requests", Label: "Pull requests", Category: "collaboration", DocsURL: "https://support.atlassian.com/bitbucket-cloud/docs/create-a-pull-request/", Keywords: []string{"pull request", "pr", "review", "审查"}},
		{ID: "pipelines", Label: "Pipelines", Category: "automation", DocsURL: "https://support.atlassian.com/bitbucket-cloud/docs/get-started-with-bitbucket-pipelines/", Keywords: []string{"pipelines", "ci", "automation", "流水线"}},
		{ID: "deployments", Label: "Deployments", Category: "delivery", DocsURL: "https://support.atlassian.com/bitbucket-cloud/docs/set-up-and-monitor-deployments/", Keywords: []string{"deployments", "deploy", "部署", "environment"}},
		{ID: "branch_restrictions", Label: "Branch restrictions", Category: "governance", DocsURL: "https://support.atlassian.com/bitbucket-cloud/docs/use-branch-permissions/", Keywords: []string{"branch restrictions", "分支限制", "branch policy"}},
		{ID: "webhooks", Label: "Webhooks", Category: "integration", DocsURL: "https://support.atlassian.com/bitbucket-cloud/docs/manage-webhooks/", Keywords: []string{"webhooks", "hooks", "集成"}},
		{ID: "repository_variables", Label: "Repository variables", Category: "credentials", DocsURL: "https://support.atlassian.com/bitbucket-cloud/docs/variables-and-secrets/", Keywords: []string{"variables", "secrets", "变量", "密钥"}},
	}
}
