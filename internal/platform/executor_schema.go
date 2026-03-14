package platform

import (
	"sort"
	"strings"
)

// ExecutorSchemaHint describes the request shape a platform_exec suggestion
// should follow for a specific capability.
type ExecutorSchemaHint struct {
	CapabilityID string   `json:"capability_id"`
	Label        string   `json:"label"`
	Summary      string   `json:"summary"`
	InspectViews []string `json:"inspect_views,omitempty"`
	MutateOps    []string `json:"mutate_ops,omitempty"`
	ScopeKeys    []string `json:"scope_keys,omitempty"`
	QueryKeys    []string `json:"query_keys,omitempty"`
	FieldRules   []string `json:"field_rules,omitempty"`
	Notes        []string `json:"notes,omitempty"`
	Example      string   `json:"example,omitempty"`
}

var executorSchemaCatalog = map[Platform]map[string]ExecutorSchemaHint{
	PlatformGitHub: {
		"actions": {
			CapabilityID: "actions",
			Label:        "GitHub Actions",
			Summary:      "Inspect workflows, runs, and repository Actions policy, or update repository Actions settings and workflow state.",
			InspectViews: []string{"workflows", "workflow", "runs", "run", "workflow_usage", "artifacts", "caches", "repo_policy", "permissions", "allowed_actions"},
			MutateOps:    []string{"permissions_update", "allowed_actions_update", "enable_workflow", "disable_workflow", "dispatch", "rerun", "cancel_run"},
			ScopeKeys:    []string{"workflow_id", "run_id"},
			QueryKeys:    []string{"view", "workflow_id", "run_id", "branch", "event", "status", "page", "per_page"},
			FieldRules: []string{
				`permissions_update payload typically includes "enabled", "allowed_actions", and optional "sha_pinning_required"`,
				`allowed_actions_update payload typically includes "github_owned_allowed", "verified_allowed", and "patterns"`,
				`enable_workflow/disable_workflow require resource_id or scope.workflow_id`,
				`dispatch requires resource_id or scope.workflow_id plus payload.ref and optional payload.inputs`,
				`rerun/cancel_run require resource_id or scope.run_id`,
			},
			Notes: []string{
				`workflow dispatch and run control are not fully rollbackable via GitHub API`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"actions","flow":"mutate","operation":"permissions_update","payload":{"enabled":true,"allowed_actions":"selected"}}`,
		},
		"release": {
			CapabilityID: "release",
			Label:        "Release management",
			Summary:      "Manage GitHub releases, release notes, and release assets.",
			InspectViews: []string{"default list", "latest", "by_tag", "assets", "asset"},
			MutateOps:    []string{"create", "update", "delete", "publish_draft", "generate_notes", "asset_upload", "asset_delete"},
			QueryKeys:    []string{"view", "tag", "release_id", "asset_id"},
			FieldRules: []string{
				`mutate create/update: payload usually includes "tag_name", and may include "name", "body", "draft", "prerelease", "discussion_category_name", "generate_release_notes"`,
				`publish_draft requires resource_id with the release id`,
				`asset_upload requires scope.release_id or payload.release_id plus payload.name and one of payload.file_path | payload.content_base64 | payload.content`,
				`inspect by_tag: set query.view="by_tag" and query.tag or resource_id`,
				`inspect assets: set query.view="assets" and query.release_id`,
			},
			Notes: []string{
				`use validate_payload as a subset of the desired release state`,
				`asset_delete rollback succeeds only when a prior local path, inline content, or downloadable asset source exists`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"release","flow":"mutate","operation":"create","payload":{"tag_name":"v1.2.0","name":"v1.2.0","draft":false}}`,
		},
		"pull_request": {
			CapabilityID: "pull_request",
			Label:        "Pull request lifecycle",
			Summary:      "Inspect, create, update, close, reopen, merge, or manage auto-merge for pull requests.",
			InspectViews: []string{"default list", "single pull", "files", "commits"},
			MutateOps:    []string{"create", "update", "close", "reopen", "merge", "enable_auto_merge", "disable_auto_merge"},
			ScopeKeys:    []string{"pull_number"},
			QueryKeys:    []string{"view", "state", "head", "base", "sort", "direction", "page", "per_page"},
			FieldRules: []string{
				`create payload usually includes "title", "head", "base", and may include "body" and "draft"`,
				`update payload usually includes "title", "body", "base", "state", or "maintainer_can_modify"`,
				`close/reopen/merge/auto-merge require resource_id or scope.pull_number`,
				`merge payload may include "commit_title", "commit_message", "sha", and "merge_method"`,
				`enable_auto_merge payload may include "merge_method", "commit_title", "commit_message", and "author_email"`,
			},
			Notes: []string{
				`merge rollback is not fully automatic because GitHub API does not provide pull-request unmerge`,
				`auto-merge uses GitHub GraphQL even though inspect/create/update/merge are REST-backed`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"pull_request","flow":"mutate","operation":"create","payload":{"title":"Release v1.2.0","head":"release/v1.2.0","base":"main","body":"Prepare ship PR","draft":false}}`,
		},
		"pr_review": {
			CapabilityID: "pr_review",
			Label:        "Pull request review and approval",
			Summary:      "Inspect review state or perform review actions for a pull request.",
			InspectViews: []string{"default pull", "reviews", "review", "requested_reviewers"},
			MutateOps:    []string{"approve", "request_changes", "comment", "dismiss", "request_reviewers", "remove_reviewers"},
			ScopeKeys:    []string{"pull_number", "review_id"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`every mutate action requires scope.pull_number`,
				`dismiss requires resource_id or scope.review_id`,
				`request_reviewers/remove_reviewers payload uses "reviewers" and/or "team_reviewers"`,
			},
			Notes: []string{
				`approve/request_changes/comment create a review that can be rolled back by dismissal`,
				`inspect requested reviewers with query.view="requested_reviewers"`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"pr_review","flow":"mutate","operation":"approve","scope":{"pull_number":"42"},"payload":{"body":"LGTM"}}`,
		},
		"deploy_keys": {
			CapabilityID: "deploy_keys",
			Label:        "Deploy keys",
			Summary:      "Inspect repository deploy keys or create/delete them.",
			MutateOps:    []string{"create", "delete"},
			FieldRules: []string{
				`create payload requires "title" and "key", and may include "read_only"`,
				`delete requires resource_id with the deploy key id`,
			},
			Notes:   []string{`there is no update operation; replace by create/delete when rotation is needed`},
			Example: `{"interaction":"platform_exec","capability_id":"deploy_keys","flow":"mutate","operation":"create","payload":{"title":"CI key","key":"ssh-ed25519 AAAA...","read_only":true}}`,
		},
		"actions_secrets_variables": {
			CapabilityID: "actions_secrets_variables",
			Label:        "Actions secrets and variables",
			Summary:      "Manage repository-level or environment-level GitHub Actions secrets and variables.",
			MutateOps:    []string{"create", "update", "delete"},
			ScopeKeys:    []string{"kind", "scope", "environment"},
			FieldRules: []string{
				`scope.kind must be "secret" or "variable"`,
				`scope.scope is usually "repository" or "environment"`,
				`environment scope requires scope.environment`,
				`payload requires "name"; variables use "value"; secrets use "value" and are write-only after creation`,
			},
			Notes: []string{
				`secret rollback requires rollback_payload with the previous plaintext value`,
				`prefer repository scope unless the secret or variable is truly environment-specific`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"actions_secrets_variables","flow":"mutate","operation":"create","scope":{"kind":"secret","scope":"environment","environment":"production"},"payload":{"name":"OPENAI_API_KEY","value":"<api-key>"}}`,
		},
		"codespaces": {
			CapabilityID: "codespaces",
			Label:        "Codespaces",
			Summary:      "Inspect repository Codespaces availability, devcontainer templates, and create/start/stop/delete codespaces.",
			InspectViews: []string{"list", "single", "devcontainers", "repo_policy", "prebuilds", "permissions_check"},
			MutateOps:    []string{"create", "start", "stop", "delete"},
			ScopeKeys:    []string{"codespace_name"},
			QueryKeys:    []string{"view", "codespace_name", "page", "per_page"},
			FieldRules: []string{
				`create payload usually includes "ref", "devcontainer_path", "machine", "location", or other codespace creation fields`,
				`start/stop/delete require resource_id or scope.codespace_name`,
			},
			Notes: []string{
				`deleted codespaces cannot be recreated automatically through rollback`,
				`repo_policy and prebuilds are derived from repository machine/prebuild availability surfaces`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"codespaces","flow":"mutate","operation":"create","payload":{"ref":"main","devcontainer_path":".devcontainer/devcontainer.json"}}`,
		},
		"codespaces_secrets": {
			CapabilityID: "codespaces_secrets",
			Label:        "Codespaces secrets",
			Summary:      "Manage repository Codespaces secrets used during bootstrap or development.",
			InspectViews: []string{"default list", "public_key"},
			MutateOps:    []string{"create", "update", "delete"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`payload requires "name"; create/update also requires "value"`,
				`inspect public key with query.view="public_key" only when debugging encryption flow`,
			},
			Notes:   []string{`rollback requires rollback_payload with the previous plaintext value`},
			Example: `{"interaction":"platform_exec","capability_id":"codespaces_secrets","flow":"mutate","operation":"create","payload":{"name":"NPM_TOKEN","value":"<token>"}}`,
		},
		"dependabot_secrets": {
			CapabilityID: "dependabot_secrets",
			Label:        "Dependabot secrets",
			Summary:      "Manage repository Dependabot secrets for private registries and update automation.",
			InspectViews: []string{"default list", "public_key"},
			MutateOps:    []string{"create", "update", "delete"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`payload requires "name"; create/update also requires "value"`,
				`use these only for Dependabot registry access, not for Actions runtime secrets`,
			},
			Notes:   []string{`rollback requires rollback_payload with the previous plaintext value`},
			Example: `{"interaction":"platform_exec","capability_id":"dependabot_secrets","flow":"mutate","operation":"create","payload":{"name":"PRIVATE_REGISTRY_TOKEN","value":"<token>"}}`,
		},
		"dependabot_config": {
			CapabilityID: "dependabot_config",
			Label:        "Dependabot configuration",
			Summary:      "Inspect or mutate .github/dependabot.yml so grouped updates, version updates, registries, and schedules stay coherent.",
			InspectViews: []string{"file"},
			MutateOps:    []string{"create", "update", "delete"},
			FieldRules: []string{
				`mutate create/update payload accepts either "content" with full dependabot.yml text or structured "config" with ecosystems/directories/schedule/grouped_updates/version_updates/security_updates/open_pull_limit/labels/assignees`,
				`rollback restores the previous file contents automatically when the previous snapshot exists`,
			},
			Notes:   []string{`use this for grouped/version update policy because GitHub exposes those settings primarily through dependabot.yml rather than repository settings APIs`, `validation checks schema correctness, deterministic re-encode, and no-op diffs`},
			Example: `{"interaction":"platform_exec","capability_id":"dependabot_config","flow":"mutate","operation":"update","payload":{"config":{"version":2,"updates":[{"ecosystem":"github-actions","directories":["/"],"schedule":{"interval":"weekly"}}]},"message":"Update dependabot policy"}}`,
		},
		"webhooks": {
			CapabilityID: "webhooks",
			Label:        "Webhooks",
			Summary:      "Inspect, create, update, ping, or delete repository webhooks.",
			MutateOps:    []string{"create", "update", "ping", "delete"},
			QueryKeys:    []string{"include_deliveries"},
			FieldRules: []string{
				`create/update payload usually includes "name", "active", "events", and "config"`,
				`ping requires resource_id with the webhook id`,
				`inspect deliveries by setting resource_id and query.include_deliveries="true"`,
			},
			Notes:   []string{`document the downstream consumer and secret expectation in the reason text`},
			Example: `{"interaction":"platform_exec","capability_id":"webhooks","flow":"mutate","operation":"create","payload":{"name":"web","active":true,"events":["push"],"config":{"url":"<endpoint>","content_type":"json","secret":"<secret>"}}}`,
		},
		"pages": {
			CapabilityID: "pages",
			Label:        "GitHub Pages",
			Summary:      "Inspect site configuration, build status, domain state, and manage Pages publishing.",
			InspectViews: []string{"default config", "build_history", "build_detail", "latest_build", "health", "domain", "dns"},
			MutateOps:    []string{"create", "update", "build", "rebuild", "delete"},
			QueryKeys:    []string{"view", "build_id"},
			FieldRules: []string{
				`create/update payload may include "source", "build_type", "cname", and "https_enforced"`,
				`inspect build history with query.view="build_history"`,
				`inspect a specific build with query.view="build_detail" and query.build_id or resource_id`,
				`query.view="dns" or payload.cname should be validated against external DNS rather than assumed from GitHub config alone`,
			},
			Notes: []string{
				`prefer inspect before mutate so the model sees the existing source branch/path and domain state`,
				`build is a separate mutate op and should usually be followed by validate`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"pages","flow":"inspect","query":{"view":"latest_build"}}`,
		},
		"deployment": {
			CapabilityID: "deployment",
			Label:        "Deployments",
			Summary:      "Create deployments, inspect status, or post deployment statuses.",
			InspectViews: []string{"default list", "single deployment", "statuses"},
			MutateOps:    []string{"create", "status", "delete"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`create payload usually includes "ref" and "environment", and may include "task", "payload", "description", "production_environment"`,
				`status requires resource_id with the deployment id and payload.state`,
				`inspect statuses with resource_id and query.view="statuses"`,
			},
			Notes:   []string{`deployments are safer when paired with environments and post-deploy validation`},
			Example: `{"interaction":"platform_exec","capability_id":"deployment","flow":"mutate","operation":"create","payload":{"ref":"main","environment":"production","description":"ship v1.2.0"}}`,
		},
		"environments": {
			CapabilityID: "environments",
			Label:        "Environments",
			Summary:      "Inspect or manage repository environments and protection rules.",
			MutateOps:    []string{"create", "update", "delete"},
			FieldRules: []string{
				`resource_id is the environment name, or payload.name for create/update`,
				`payload may include "wait_timer", "reviewers", "deployment_branch_policy", "prevent_self_review", and "can_admins_bypass"`,
			},
			Notes:   []string{`environment create and update share the same executor path; use update when the environment already exists`},
			Example: `{"interaction":"platform_exec","capability_id":"environments","flow":"mutate","operation":"update","resource_id":"production","payload":{"wait_timer":0,"prevent_self_review":true}}`,
		},
		"rulesets": {
			CapabilityID: "rulesets",
			Label:        "Repository rulesets",
			Summary:      "Inspect repository rulesets, create/update/delete them, or inspect rule suites.",
			InspectViews: []string{"default list", "single ruleset", "rule_suites", "rule_suite"},
			MutateOps:    []string{"create", "update", "delete"},
			QueryKeys:    []string{"view", "rule_suite_id"},
			FieldRules: []string{
				`create/update payload usually includes "name", "target", "enforcement", "conditions", "rules", and optional "bypass_actors"`,
				`inspect a specific rule suite with query.view="rule_suite" and query.rule_suite_id`,
			},
			Notes:   []string{`prefer rulesets over ad hoc branch-protection drift when the goal is repository-wide governance`},
			Example: `{"interaction":"platform_exec","capability_id":"rulesets","flow":"inspect","query":{"view":"rule_suites"}}`,
		},
		"branch_rulesets": {
			CapabilityID: "branch_rulesets",
			Label:        "Branch rulesets",
			Summary:      "Inspect how branch-specific protection resolves for a branch and manage branch-targeted rulesets.",
			InspectViews: []string{"default list", "branch_rules"},
			MutateOps:    []string{"create", "update", "delete"},
			QueryKeys:    []string{"view", "branch"},
			FieldRules: []string{
				`inspect resolved branch rules with query.view="branch_rules" and query.branch`,
				`mutate payload shape matches repository rulesets but target/conditions should be branch-specific`,
			},
			Notes:   []string{`use this when the goal is branch protection or branch-specific merge policy, not generic repository governance`},
			Example: `{"interaction":"platform_exec","capability_id":"branch_rulesets","flow":"inspect","query":{"view":"branch_rules","branch":"main"}}`,
		},
		"check_runs_failure_threshold": {
			CapabilityID: "check_runs_failure_threshold",
			Label:        "Check runs failure threshold",
			Summary:      "Inspect or update ruleset-backed required-check policy and merge blocking thresholds for repository branches.",
			InspectViews: []string{"default list", "rule_suites", "branch_rules"},
			MutateOps:    []string{"create", "update", "delete"},
			QueryKeys:    []string{"view", "branch", "rule_suite_id"},
			FieldRules: []string{
				`this surface is implemented through rulesets; payload should focus on required status checks and related merge gates`,
			},
			Notes: []string{
				`GitHub does not expose a standalone check-failure-threshold surface; use rulesets payloads for this capability`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"check_runs_failure_threshold","flow":"inspect","query":{"view":"branch_rules","branch":"main"}}`,
		},
		"packages": {
			CapabilityID: "packages",
			Label:        "Packages",
			Summary:      "Inspect package metadata and versions, or delete/restore package resources.",
			InspectViews: []string{"package list", "single package", "versions", "latest_version", "single version", "assets"},
			MutateOps:    []string{"delete", "restore"},
			ScopeKeys:    []string{"owner_type", "package_type", "package_name", "version_id", "registry", "namespace"},
			QueryKeys:    []string{"view", "owner_type", "package_type", "package_name", "version_id", "visibility", "state"},
			FieldRules: []string{
				`package_type is required for all package operations`,
				`package_name is required for single-package and version flows`,
				`assets inspection requires a version identifier and returns registry metadata when the registry exposes it`,
				`version-specific delete/restore uses resource_id or scope.version_id`,
				`scope may be "org", "user", or "repo"; repo scope preserves repository-scoped identity while REST resolution still uses the owner namespace endpoints`,
			},
			Notes:   []string{`owner_type can still be set explicitly to force the user/org namespace lookup path`},
			Example: `{"interaction":"platform_exec","capability_id":"packages","flow":"inspect","scope":{"owner_type":"org","package_type":"container","package_name":"gitdex"},"query":{"view":"versions"}}`,
		},
		"notifications": {
			CapabilityID: "notifications",
			Label:        "Notifications",
			Summary:      "Inspect repository or thread notification subscriptions and update watch state.",
			InspectViews: []string{"repo_subscription", "repo_notifications", "global_inbox", "participating_inbox", "thread", "thread_subscription"},
			MutateOps:    []string{"watch", "update", "unwatch", "delete", "mark_read"},
			ScopeKeys:    []string{"thread_id"},
			QueryKeys:    []string{"view", "all", "participating", "reason", "since", "before", "page", "per_page"},
			FieldRules: []string{
				`repo-level watch state uses query.view="repo_subscription" or the default`,
				`thread-specific flows require scope.thread_id and query.view="thread" or "thread_subscription"`,
				`watch/update payload usually uses "subscribed" and "ignored"`,
			},
			Notes:   []string{`mark_read cannot be rolled back via GitHub API`},
			Example: `{"interaction":"platform_exec","capability_id":"notifications","flow":"mutate","operation":"watch","payload":{"subscribed":true,"ignored":false}}`,
		},
		"email_notifications": {
			CapabilityID: "email_notifications",
			Label:        "Email notifications",
			Summary:      "Use the same subscription executor when the goal is email notification routing or subscription state.",
			InspectViews: []string{"repo_subscription", "repo_notifications", "global_inbox", "participating_inbox", "thread", "thread_subscription"},
			MutateOps:    []string{"watch", "update", "unwatch", "delete", "mark_read"},
			ScopeKeys:    []string{"thread_id"},
			QueryKeys:    []string{"view", "all", "participating", "reason", "since", "before", "page", "per_page"},
			FieldRules: []string{
				`prefer repo_subscription or thread_subscription views when the goal is email notification routing`,
			},
			Notes:   []string{`GitHub exposes subscription state; granular account-level email preferences are not repository admin mutations`},
			Example: `{"interaction":"platform_exec","capability_id":"email_notifications","flow":"inspect","query":{"view":"repo_subscription"}}`,
		},
		"security": {
			CapabilityID: "security",
			Label:        "Security overview",
			Summary:      "Inspect or update repository security_and_analysis posture as an aggregated security surface.",
			InspectViews: []string{"summary", "configuration", "default"},
			MutateOps:    []string{"update"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`security mutate expects a full repository-level security_and_analysis payload or a capability-specific subset already wrapped under security_and_analysis`,
			},
			Notes: []string{
				`use this aggregated surface when the goal spans multiple security controls rather than a single scanner or alert family`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"security","flow":"inspect","query":{"view":"summary"}}`,
		},
		"advanced_security": {
			CapabilityID: "advanced_security",
			Label:        "Advanced Security",
			Summary:      "Inspect or update repository security_and_analysis posture, security configuration association, and security rollout summary.",
			InspectViews: []string{"summary", "configuration", "default"},
			MutateOps:    []string{"update", "enable", "disable"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`mutate update may pass a full "security_and_analysis" object or a capability-specific object that will be wrapped for the matching security toggle`,
				`inspect summary aggregates repository security settings, automated security fixes, and code security configuration when available`,
			},
			Notes:   []string{`use this for layered security posture changes before drilling down into alert-level executors`},
			Example: `{"interaction":"platform_exec","capability_id":"advanced_security","flow":"inspect","query":{"view":"summary"}}`,
		},
		"dependabot_posture": {
			CapabilityID: "dependabot_posture",
			Label:        "Dependabot posture",
			Summary:      "Inspect repository Dependabot posture and toggle automated security fixes through the public API surface.",
			InspectViews: []string{"default", "automated_security_fixes"},
			MutateOps:    []string{"enable", "disable", "update"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`enable/update maps to automated-security-fixes while inspect also summarizes the repository posture`,
			},
			Notes:   []string{`use dependabot_config for ecosystem-specific cadence and grouping policy`},
			Example: `{"interaction":"platform_exec","capability_id":"dependabot_posture","flow":"mutate","operation":"enable"}`,
		},
		"dependency_graph": {
			CapabilityID: "dependency_graph",
			Label:        "Dependency graph",
			Summary:      "Inspect the repository dependency graph export and SBOM state.",
			InspectViews: []string{"sbom"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`inspect uses the SBOM export surface; repository-level mutation is not exposed through this executor`,
			},
			Notes:   []string{`pair this with Dependabot and code scanning flows when auditing supply-chain posture`},
			Example: `{"interaction":"platform_exec","capability_id":"dependency_graph","flow":"inspect","query":{"view":"sbom"}}`,
		},
		"dependabot": {
			CapabilityID: "dependabot",
			Label:        "Dependabot controls",
			Summary:      "Inspect or update repository Dependabot-related security settings exposed via security_and_analysis.",
			InspectViews: []string{"default"},
			MutateOps:    []string{"update", "enable", "disable"},
			FieldRules: []string{
				`mutate payload can be a "status" object or a full "security_and_analysis" wrapper`,
			},
			Notes:   []string{`use dependabot_alerts for alert triage and dependabot_security_updates for automated fix toggles`},
			Example: `{"interaction":"platform_exec","capability_id":"dependabot","flow":"mutate","operation":"enable"}`,
		},
		"dependabot_security_updates": {
			CapabilityID: "dependabot_security_updates",
			Label:        "Dependabot security updates",
			Summary:      "Inspect or toggle automated security fixes for a repository.",
			InspectViews: []string{"automated_security_fixes"},
			MutateOps:    []string{"enable", "disable", "update"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`enable/update maps to the automated-security-fixes endpoint; disable deletes that configuration`,
			},
			Notes:   []string{`pair with dependabot_alerts when the goal is to reduce existing security backlog`},
			Example: `{"interaction":"platform_exec","capability_id":"dependabot_security_updates","flow":"mutate","operation":"enable"}`,
		},
		"grouped_security_updates": {
			CapabilityID: "grouped_security_updates",
			Label:        "Grouped security updates",
			Summary:      "Inspect or update repository-level Dependabot security grouping posture exposed via security settings.",
			MutateOps:    []string{"update", "enable", "disable"},
			FieldRules: []string{
				`repository-level mutation is limited to exposed security settings; grouping policy details may still live in dependabot.yml`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"grouped_security_updates","flow":"mutate","operation":"enable"}`,
		},
		"dependabot_version_updates": {
			CapabilityID: "dependabot_version_updates",
			Label:        "Dependabot version updates",
			Summary:      "Inspect or update repository-level Dependabot version-update posture exposed via security settings.",
			MutateOps:    []string{"update", "enable", "disable"},
			FieldRules: []string{
				`repository-level mutation is limited to exposed security settings; detailed update cadence may still live in dependabot.yml`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"dependabot_version_updates","flow":"mutate","operation":"enable"}`,
		},
		"dependabot_alerts": {
			CapabilityID: "dependabot_alerts",
			Label:        "Dependabot alerts",
			Summary:      "Inspect, dismiss, or reopen repository Dependabot alerts.",
			InspectViews: []string{"alerts", "alert"},
			MutateOps:    []string{"dismiss", "reopen", "update"},
			ScopeKeys:    []string{"alert_number"},
			QueryKeys:    []string{"view", "state", "severity", "ecosystem", "package", "manifest", "sort", "direction", "page", "per_page"},
			FieldRules: []string{
				`dismiss/update payload typically uses "state", "dismissed_reason", and "dismissed_comment"`,
				`resource_id or scope.alert_number is required for alert mutation`,
			},
			Notes:   []string{`use validate after dismiss/reopen so the final alert state is explicit`},
			Example: `{"interaction":"platform_exec","capability_id":"dependabot_alerts","flow":"mutate","operation":"dismiss","resource_id":"17"}`,
		},
		"secret_scanning_settings": {
			CapabilityID: "secret_scanning_settings",
			Label:        "Secret scanning settings",
			Summary:      "Inspect or update repository-level secret scanning posture through security_and_analysis.",
			InspectViews: []string{"default"},
			MutateOps:    []string{"update", "enable", "disable"},
			FieldRules: []string{
				`mutate payload can be a simple status object and will be wrapped to the secret_scanning security_and_analysis field`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"secret_scanning_settings","flow":"mutate","operation":"enable"}`,
		},
		"secret_scanning_alerts": {
			CapabilityID: "secret_scanning_alerts",
			Label:        "Secret scanning alerts",
			Summary:      "Inspect secret scanning alerts and resolve or reopen them.",
			InspectViews: []string{"alerts", "alert", "locations", "push_protection_bypasses"},
			MutateOps:    []string{"resolve", "reopen", "update"},
			ScopeKeys:    []string{"alert_number"},
			QueryKeys:    []string{"view", "state", "secret_type", "resolution", "page", "per_page", "alert_number"},
			FieldRules: []string{
				`resolve/update payload typically uses "state", "resolution", and "resolution_comment"`,
				`locations view requires resource_id or scope.alert_number`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"secret_scanning_alerts","flow":"inspect","query":{"view":"locations","alert_number":"17"}}`,
		},
		"code_scanning": {
			CapabilityID: "code_scanning",
			Label:        "Code scanning",
			Summary:      "Inspect code scanning alerts, instances, analyses, default setup, and mutate alert state or default setup.",
			InspectViews: []string{"alerts", "alert", "instances", "analyses", "analysis", "default_setup"},
			MutateOps:    []string{"dismiss", "reopen", "update", "default_setup_update", "delete_analysis"},
			ScopeKeys:    []string{"alert_number", "analysis_id"},
			QueryKeys:    []string{"view", "state", "tool_name", "severity", "ref", "page", "per_page", "analysis_id", "alert_number"},
			FieldRules: []string{
				`alert mutation requires resource_id or scope.alert_number`,
				`default_setup_update writes the repository default setup surface rather than alert state`,
				`delete_analysis requires resource_id or scope.analysis_id`,
			},
			Notes:   []string{`use codeql_analysis when the goal is primarily default setup or analysis inventory rather than alert triage`},
			Example: `{"interaction":"platform_exec","capability_id":"code_scanning","flow":"inspect","query":{"view":"analyses"}}`,
		},
		"code_scanning_tool_settings": {
			CapabilityID: "code_scanning_tool_settings",
			Label:        "Code scanning tool settings",
			Summary:      "Inspect the code security configuration surface when the goal is tool-level settings review.",
			InspectViews: []string{"tool_settings", "configuration"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`this public surface is inspect-only; use it to review code security configuration assignments and tool posture`,
			},
			Notes:   []string{`there is no standalone public mutate endpoint for all tool settings`},
			Example: `{"interaction":"platform_exec","capability_id":"code_scanning_tool_settings","flow":"inspect","query":{"view":"tool_settings"}}`,
		},
		"code_scanning_default_setup": {
			CapabilityID: "code_scanning_default_setup",
			Label:        "Code scanning default setup",
			Summary:      "Inspect or update the repository code scanning default setup surface.",
			InspectViews: []string{"default_setup"},
			MutateOps:    []string{"default_setup_update"},
			QueryKeys:    []string{"view"},
			FieldRules: []string{
				`default_setup_update payload typically uses the fields returned by the default setup surface`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"code_scanning_default_setup","flow":"inspect","query":{"view":"default_setup"}}`,
		},
		"codeql_setup": {
			CapabilityID: "codeql_setup",
			Label:        "CodeQL setup",
			Summary:      "Inspect or update CodeQL default setup and review analysis inventory.",
			InspectViews: []string{"default_setup", "analyses", "analysis"},
			MutateOps:    []string{"default_setup_update", "delete_analysis"},
			ScopeKeys:    []string{"analysis_id"},
			QueryKeys:    []string{"view", "analysis_id", "ref", "page", "per_page"},
			FieldRules: []string{
				`use default_setup_update for CodeQL default setup changes and delete_analysis only for cleanup flows`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"codeql_setup","flow":"inspect","query":{"view":"default_setup"}}`,
		},
		"codeql_analysis": {
			CapabilityID: "codeql_analysis",
			Label:        "CodeQL analysis",
			Summary:      "Inspect or update CodeQL default setup and analysis inventory through the code scanning surfaces.",
			InspectViews: []string{"default_setup", "analyses", "analysis"},
			MutateOps:    []string{"default_setup_update", "delete_analysis"},
			ScopeKeys:    []string{"analysis_id"},
			QueryKeys:    []string{"view", "analysis_id", "ref", "page", "per_page"},
			FieldRules: []string{
				`default_setup_update payload typically uses the fields returned by the default setup surface`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"codeql_analysis","flow":"inspect","query":{"view":"default_setup"}}`,
		},
		"copilot_autofix": {
			CapabilityID: "copilot_autofix",
			Label:        "Copilot Autofix",
			Summary:      "Inspect code scanning alerts with Copilot Autofix context and available autofix suggestions or commits.",
			InspectViews: []string{"alerts", "alert", "autofix", "autofix_commits"},
			ScopeKeys:    []string{"alert_number"},
			QueryKeys:    []string{"view", "alert_number"},
			FieldRules: []string{
				`autofix and autofix_commits require resource_id or scope.alert_number`,
			},
			Notes:   []string{`mutation of Autofix output itself is not exposed; use code_scanning alert/state executors around it`},
			Example: `{"interaction":"platform_exec","capability_id":"copilot_autofix","flow":"inspect","resource_id":"17","query":{"view":"autofix"}}`,
		},
		"secret_protection": {
			CapabilityID: "secret_protection",
			Label:        "Secret protection",
			Summary:      "Inspect secret scanning alerts, locations, and push protection bypasses; resolve or reopen alerts.",
			InspectViews: []string{"alerts", "alert", "locations", "push_protection_bypasses"},
			MutateOps:    []string{"resolve", "reopen", "update"},
			ScopeKeys:    []string{"alert_number"},
			QueryKeys:    []string{"view", "state", "secret_type", "resolution", "page", "per_page", "alert_number"},
			FieldRules: []string{
				`resolve/update payload typically uses "state", "resolution", and "resolution_comment"`,
				`locations view requires resource_id or scope.alert_number`,
			},
			Notes:   []string{`use push_protection capability for repository-level push-protection setting changes`},
			Example: `{"interaction":"platform_exec","capability_id":"secret_protection","flow":"inspect","query":{"view":"push_protection_bypasses"}}`,
		},
		"private_vulnerability_reporting": {
			CapabilityID: "private_vulnerability_reporting",
			Label:        "Private vulnerability reporting",
			Summary:      "Inspect or update repository private vulnerability reporting through security_and_analysis.",
			MutateOps:    []string{"update", "enable", "disable"},
			FieldRules: []string{
				`mutate payload can be a simple status object and will be wrapped for the repository security_and_analysis surface`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"private_vulnerability_reporting","flow":"mutate","operation":"enable"}`,
		},
		"protection_rules": {
			CapabilityID: "protection_rules",
			Label:        "Protection rules",
			Summary:      "Inspect or update non-provider secret scanning protection-rule posture exposed via repository security settings.",
			MutateOps:    []string{"update", "enable", "disable"},
			FieldRules: []string{
				`this executor covers repository-level security setting toggles, not branch rulesets or merge rules`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"protection_rules","flow":"mutate","operation":"enable"}`,
		},
		"push_protection": {
			CapabilityID: "push_protection",
			Label:        "Push protection",
			Summary:      "Inspect or update repository push-protection posture for secret scanning.",
			MutateOps:    []string{"update", "enable", "disable"},
			FieldRules: []string{
				`mutate payload can be a simple status object and will be wrapped to secret_scanning_push_protection`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"push_protection","flow":"mutate","operation":"enable"}`,
		},
		"copilot_code_review": {
			CapabilityID: "copilot_code_review",
			Label:        "Copilot code review admin",
			Summary:      "Inspect organization Copilot billing, seats, metrics, and content exclusions relevant to code review rollout.",
			InspectViews: []string{"billing", "seats", "metrics", "content_exclusions"},
			MutateOps:    []string{"update_content_exclusions", "delete_content_exclusions"},
			QueryKeys:    []string{"view", "page", "per_page", "since", "before"},
			FieldRules: []string{
				`content exclusion mutation is organization-scoped and expects the exact content_exclusions document shape returned by GitHub`,
			},
			Notes:   []string{`Copilot org-admin endpoints generally require organization ownership and may be unavailable on user-owned repositories`},
			Example: `{"interaction":"platform_exec","capability_id":"copilot_code_review","flow":"inspect","query":{"view":"content_exclusions"}}`,
		},
		"copilot_coding_agent": {
			CapabilityID: "copilot_coding_agent",
			Label:        "Copilot coding agent admin",
			Summary:      "Inspect organization Copilot billing, seat consumption, metrics, and content exclusions relevant to agent-mode rollout.",
			InspectViews: []string{"billing", "seats", "metrics", "content_exclusions"},
			MutateOps:    []string{"update_content_exclusions", "delete_content_exclusions"},
			QueryKeys:    []string{"view", "page", "per_page", "since", "before"},
			FieldRules: []string{
				`content exclusion mutation is organization-scoped and expects the exact content_exclusions document shape returned by GitHub`,
			},
			Notes:   []string{`coding-agent-specific controls are narrower than general Copilot billing/admin surfaces and may be plan-gated`},
			Example: `{"interaction":"platform_exec","capability_id":"copilot_coding_agent","flow":"inspect","query":{"view":"metrics"}}`,
		},
		"copilot_seat_management": {
			CapabilityID: "copilot_seat_management",
			Label:        "Copilot seat management",
			Summary:      "Inspect Copilot seat assignments and add/remove selected users or teams.",
			InspectViews: []string{"seats", "seat_assignments", "billing"},
			MutateOps:    []string{"add_users", "remove_users", "add_teams", "remove_teams"},
			QueryKeys:    []string{"view", "page", "per_page"},
			FieldRules: []string{
				`add_users/remove_users payload uses "selected_usernames"`,
				`add_teams/remove_teams payload uses "selected_team_slugs"`,
			},
			Notes:   []string{`Copilot seat APIs are organization-scoped and require org-owner level authorization`},
			Example: `{"interaction":"platform_exec","capability_id":"copilot_seat_management","flow":"mutate","operation":"add_users","payload":{"selected_usernames":["octocat"]}}`,
		},
	},
	PlatformGitLab: {
		"merge_requests": {
			CapabilityID: "merge_requests",
			Label:        "GitLab merge requests",
			Summary:      "Inspect merge requests and drive create/update/close/reopen flows through the public GitLab API.",
			InspectViews: []string{"list", "single", "changes", "approvals"},
			MutateOps:    []string{"create", "update", "close", "reopen"},
			ScopeKeys:    []string{"merge_request_iid"},
			QueryKeys:    []string{"view", "state", "source_branch", "target_branch", "labels", "page", "per_page"},
			FieldRules: []string{
				`create payload usually includes "title", "source_branch", "target_branch", and optional "description" or "draft"`,
				`update/close/reopen require resource_id or scope.merge_request_iid`,
			},
			Notes:   []string{`merged-state reversal is not automatically rollbackable`},
			Example: `{"interaction":"platform_exec","capability_id":"merge_requests","flow":"mutate","operation":"create","payload":{"title":"Ship pages","source_branch":"feature/pages","target_branch":"main"}}`,
		},
		"pipelines": {
			CapabilityID: "pipelines",
			Label:        "GitLab pipelines",
			Summary:      "Inspect project pipelines and run-control operations such as create, retry, or cancel.",
			InspectViews: []string{"list", "single", "jobs", "bridges"},
			MutateOps:    []string{"create", "retry", "cancel"},
			ScopeKeys:    []string{"pipeline_id"},
			QueryKeys:    []string{"view", "ref", "status", "source", "page", "per_page"},
			FieldRules: []string{
				`create payload usually includes "ref" and optional "variables"`,
				`retry/cancel require resource_id or scope.pipeline_id`,
			},
			Notes:   []string{`retry and cancel are explicit run-control operations and should be treated as non-reversible`},
			Example: `{"interaction":"platform_exec","capability_id":"pipelines","flow":"inspect","query":{"view":"list","ref":"main"}}`,
		},
		"environments": {
			CapabilityID: "environments",
			Label:        "GitLab environments",
			Summary:      "Inspect environments and deployment state, or stop an active environment when supported.",
			InspectViews: []string{"list", "single", "deployments"},
			MutateOps:    []string{"stop"},
			ScopeKeys:    []string{"environment_id"},
			QueryKeys:    []string{"view", "name", "page", "per_page"},
			FieldRules: []string{
				`stop requires resource_id or scope.environment_id`,
			},
			Notes:   []string{`environment policy parity is broader than the stop endpoint`},
			Example: `{"interaction":"platform_exec","capability_id":"environments","flow":"inspect","query":{"view":"deployments"}}`,
		},
		"pages": {
			CapabilityID: "pages",
			Label:        "GitLab Pages",
			Summary:      "Inspect GitLab Pages deployment, domain, and certificate posture through the project Pages surface.",
			InspectViews: []string{"settings", "domains", "domain", "health"},
			MutateOps:    []string{"verify_domain"},
			ScopeKeys:    []string{"domain"},
			QueryKeys:    []string{"view", "domain"},
			FieldRules: []string{
				`verify_domain requires resource_id or scope.domain`,
			},
			Notes:   []string{`DNS and certificate readiness still depend on external systems`},
			Example: `{"interaction":"platform_exec","capability_id":"pages","flow":"inspect","query":{"view":"domains"}}`,
		},
		"security": {
			CapabilityID: "security",
			Label:        "GitLab security posture",
			Summary:      "Inspect GitLab security dashboard posture and analyzer coverage where public project APIs expose it.",
			InspectViews: []string{"dashboard", "vulnerabilities", "policies"},
			QueryKeys:    []string{"view", "severity", "state", "page", "per_page"},
			FieldRules: []string{
				`this surface is inspect-only until a stable mutate/rollback model is added per analyzer or policy family`,
			},
			Notes:   []string{`treat this as inspect_only in unattended flows`},
			Example: `{"interaction":"platform_exec","capability_id":"security","flow":"inspect","query":{"view":"dashboard"}}`,
		},
	},
	PlatformBitbucket: {
		"pull_requests": {
			CapabilityID: "pull_requests",
			Label:        "Bitbucket pull requests",
			Summary:      "Inspect pull requests and drive create/update/decline/reopen flows through the public Bitbucket API.",
			InspectViews: []string{"list", "single", "activity", "diffstat"},
			MutateOps:    []string{"create", "update", "decline", "reopen"},
			ScopeKeys:    []string{"pull_request_id"},
			QueryKeys:    []string{"view", "state", "source_branch", "destination_branch", "page", "pagelen"},
			FieldRules: []string{
				`create payload usually includes "title", "source.branch.name", and "destination.branch.name"`,
				`update/decline/reopen require resource_id or scope.pull_request_id`,
			},
			Notes:   []string{`merged pull requests cannot be automatically unmerged`},
			Example: `{"interaction":"platform_exec","capability_id":"pull_requests","flow":"mutate","operation":"create","payload":{"title":"Ship release","source":{"branch":{"name":"release/v1"}},"destination":{"branch":{"name":"main"}}}}`,
		},
		"pipelines": {
			CapabilityID: "pipelines",
			Label:        "Bitbucket pipelines",
			Summary:      "Inspect pipelines and trigger or stop runs using the public Bitbucket pipeline APIs.",
			InspectViews: []string{"list", "single", "steps"},
			MutateOps:    []string{"create", "stop", "rerun"},
			ScopeKeys:    []string{"pipeline_uuid"},
			QueryKeys:    []string{"view", "ref", "state", "page", "pagelen"},
			FieldRules: []string{
				`create payload usually includes "target" with a ref or selector`,
				`stop/rerun require resource_id or scope.pipeline_uuid`,
			},
			Notes:   []string{`run-control actions are non-reversible`},
			Example: `{"interaction":"platform_exec","capability_id":"pipelines","flow":"inspect","query":{"view":"list"}}`,
		},
		"deployments": {
			CapabilityID: "deployments",
			Label:        "Bitbucket deployments",
			Summary:      "Inspect deployment environments and deployment history available through Bitbucket deployment APIs.",
			InspectViews: []string{"environments", "history"},
			QueryKeys:    []string{"view", "environment", "page", "pagelen"},
			FieldRules: []string{
				`use inspect flows to audit deployment state before composing pipeline or environment actions`,
			},
			Notes:   []string{`mutation parity is narrower than inspection today`},
			Example: `{"interaction":"platform_exec","capability_id":"deployments","flow":"inspect","query":{"view":"environments"}}`,
		},
		"branch_restrictions": {
			CapabilityID: "branch_restrictions",
			Label:        "Bitbucket branch restrictions",
			Summary:      "Inspect and mutate repository branch restrictions through the public Bitbucket repository settings APIs.",
			InspectViews: []string{"list"},
			MutateOps:    []string{"create", "update", "delete"},
			ScopeKeys:    []string{"restriction_id"},
			QueryKeys:    []string{"kind", "pattern", "page", "pagelen"},
			FieldRules: []string{
				`create/update payload typically includes "kind", "pattern", and branch restriction matchers`,
			},
			Example: `{"interaction":"platform_exec","capability_id":"branch_restrictions","flow":"inspect","query":{"kind":"push"}}`,
		},
		"webhooks": {
			CapabilityID: "webhooks",
			Label:        "Bitbucket webhooks",
			Summary:      "Inspect, create, update, or delete repository webhooks.",
			InspectViews: []string{"list", "single"},
			MutateOps:    []string{"create", "update", "delete"},
			ScopeKeys:    []string{"webhook_uuid"},
			QueryKeys:    []string{"page", "pagelen"},
			FieldRules: []string{
				`create/update payload usually includes "description", "url", "active", and "events"`,
			},
			Notes:   []string{`delivery rollback depends on downstream system state`},
			Example: `{"interaction":"platform_exec","capability_id":"webhooks","flow":"mutate","operation":"create","payload":{"description":"CI","url":"https://example.test/hook","active":true}}`,
		},
		"repository_variables": {
			CapabilityID: "repository_variables",
			Label:        "Bitbucket repository variables",
			Summary:      "Inspect and mutate repository variables used by pipelines and deployment flows.",
			InspectViews: []string{"list", "single"},
			MutateOps:    []string{"create", "update", "delete"},
			ScopeKeys:    []string{"variable_uuid"},
			QueryKeys:    []string{"page", "pagelen"},
			FieldRules: []string{
				`create/update payload usually includes "key", "value", and optional "secured"`,
			},
			Notes:   []string{`rollback of secured values requires a previous plaintext source`},
			Example: `{"interaction":"platform_exec","capability_id":"repository_variables","flow":"mutate","operation":"create","payload":{"key":"DEPLOY_ENV","value":"prod","secured":false}}`,
		},
	},
}

// ExecutorSchemaFor returns the known schema hint for a capability.
func ExecutorSchemaFor(p Platform, capabilityID string) (ExecutorSchemaHint, bool) {
	items := executorSchemaCatalog[p]
	if len(items) == 0 {
		return ExecutorSchemaHint{}, false
	}
	hint, ok := items[strings.TrimSpace(capabilityID)]
	return hint, ok
}

// RecommendedExecutorSchemas returns the most relevant executor schema hints for a goal.
func RecommendedExecutorSchemas(p Platform, goal string, preferredIDs []string, limit int) []ExecutorSchemaHint {
	catalog := executorSchemaCatalog[p]
	if len(catalog) == 0 {
		return nil
	}

	type scored struct {
		hint  ExecutorSchemaHint
		score int
	}

	goal = normalizeGoal(goal)
	seen := map[string]struct{}{}
	var ranked []scored

	for index, id := range preferredIDs {
		hint, ok := catalog[strings.TrimSpace(id)]
		if !ok {
			continue
		}
		if _, exists := seen[hint.CapabilityID]; exists {
			continue
		}
		seen[hint.CapabilityID] = struct{}{}
		ranked = append(ranked, scored{
			hint:  hint,
			score: 1000 - index,
		})
	}

	if goal != "" {
		for _, capability := range CapabilityCatalog(p) {
			hint, ok := catalog[capability.ID]
			if !ok {
				continue
			}
			score := capabilityScore(goal, capability)
			if score <= 0 {
				continue
			}
			if _, exists := seen[hint.CapabilityID]; exists {
				continue
			}
			seen[hint.CapabilityID] = struct{}{}
			ranked = append(ranked, scored{
				hint:  hint,
				score: score,
			})
		}
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].hint.Label < ranked[j].hint.Label
		}
		return ranked[i].score > ranked[j].score
	})

	if limit <= 0 {
		limit = 4
	}
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	out := make([]ExecutorSchemaHint, 0, len(ranked))
	for _, item := range ranked {
		out = append(out, item.hint)
	}
	return out
}
