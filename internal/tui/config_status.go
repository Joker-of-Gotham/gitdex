package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	gitplatform "github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func (m Model) runConfigSlashCommand(args []string) (tea.Model, tea.Cmd) {
	switch sub := normalizedSlashSubcommand(args); sub {
	case "", "status":
		m.statusMsg = localizedText("Configuration status loaded", "已加载配置状态", "Configuration status loaded")
		m.setCommandResponse(localizedConfigTitle(), m.configStatusBody())
		return m, nil
	case "llm", "model", "provider":
		m.composerFocused = false
		return m.openModelSetup(selectPrimary), nil
	case "automation", "settings":
		m.composerFocused = false
		m = m.openAutomationConfig()
		return m, nil
	case "platform":
		m.statusMsg = localizedText("Platform access status loaded", "已加载平台访问状态", "Platform access status loaded")
		m.setCommandResponse(localizedAccessTitle(), m.platformAccessBody())
		return m, nil
	default:
		m.statusMsg = localizedText("Unknown config command", "未知配置命令", "Unknown config command")
		m.setCommandResponse(localizedConfigTitle(), m.statusMsg)
		return m, nil
	}
}

func normalizedSlashSubcommand(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(args[0]))
}

func (m Model) configStatusBody() string {
	lines := []string{
		fmt.Sprintf(localizedText("Language: %s", "语言：%s", "Language: %s"), m.currentResolvedLanguage()),
		fmt.Sprintf(localizedText("LLM: %s", "LLM：%s", "LLM: %s"), m.llmConfigSummary()),
		fmt.Sprintf(localizedText("Automation: %s", "自动化：%s", "Automation: %s"), m.automationSummaryText()),
		m.automationGoalRequirementText(),
		"",
	}
	lines = append(lines, strings.Split(m.platformAccessBody(), "\n")...)
	lines = append(lines,
		"",
		localizedText("Quick actions:", "快捷操作：", "Quick actions:"),
		"/config llm",
		"/provider",
		"/settings",
		"/mode show",
		"/config platform",
	)
	return strings.Join(lines, "\n")
}

func (m Model) llmConfigSummary() string {
	primary := m.llmConfig.PrimaryRole()
	provider := strings.TrimSpace(config.RoleProvider(primary))
	if provider == "" {
		provider = strings.TrimSpace(primary.Provider)
	}
	model := strings.TrimSpace(primary.Model)
	if provider == "" && model == "" {
		return localizedText("not configured", "未配置", "not configured")
	}
	status := localizedText("ready", "可用", "ready")
	if m.llmProvider == nil {
		status = localizedText("setup required", "需要配置", "setup required")
	}
	return strings.TrimSpace(fmt.Sprintf("%s / %s (%s)", firstNonEmpty(provider, "llm"), firstNonEmpty(model, "-"), status))
}

func (m Model) platformAccessBody() string {
	platforms := []gitplatform.Platform{
		gitplatform.PlatformGitHub,
		gitplatform.PlatformGitLab,
		gitplatform.PlatformBitbucket,
	}
	lines := []string{localizedText("Platform access:", "平台访问：", "Platform access:")}
	for _, platformID := range platforms {
		lines = append(lines, m.platformAccessLine(platformID))
	}
	lines = append(lines,
		"",
		localizedText(
			"If a platform action is unavailable, configure one reachable route first.",
			"如果平台动作当前不可用，请先配置一条可达的访问路径。",
			"If a platform action is unavailable, configure one reachable route first.",
		),
	)
	return strings.Join(lines, "\n")
}

func (m Model) platformAccessLine(platformID gitplatform.Platform) string {
	switch platformID {
	case gitplatform.PlatformGitHub:
		api := localizedText("missing token", "缺少 token", "missing token")
		if strings.TrimSpace(m.platformCfg.GitHubToken) != "" {
			api = localizedText("API ready", "API 已就绪", "API ready")
		}
		gh := localizedText("disabled", "已禁用", "disabled")
		if m.adapterCfg.GitHub.GH.Enabled {
			gh = localizedText("CLI unavailable", "CLI 不可用", "CLI unavailable")
			if binary := strings.TrimSpace(m.adapterCfg.GitHub.GH.Binary); binary != "" {
				if _, err := exec.LookPath(binary); err == nil {
					gh = localizedText("CLI ready", "CLI 已就绪", "CLI ready")
				}
			}
		}
		browser := localizedText("disabled", "已禁用", "disabled")
		if m.adapterCfg.GitHub.Browser.Enabled {
			browser = fmt.Sprintf(
				localizedText("browser ready (%s)", "browser 已就绪（%s）", "browser ready (%s)"),
				firstNonEmpty(strings.TrimSpace(m.adapterCfg.GitHub.Browser.Driver), "default"),
			)
		}
		return fmt.Sprintf(
			localizedText("- GitHub: %s | gh: %s | browser: %s", "- GitHub：%s | gh：%s | browser：%s", "- GitHub: %s | gh: %s | browser: %s"),
			api,
			gh,
			browser,
		)
	case gitplatform.PlatformGitLab:
		api := localizedText("missing token", "缺少 token", "missing token")
		if strings.TrimSpace(m.platformCfg.GitLabToken) != "" {
			api = localizedText("API ready", "API 已就绪", "API ready")
		}
		browser := localizedText("disabled", "已禁用", "disabled")
		if m.adapterCfg.GitLab.Browser.Enabled {
			browser = fmt.Sprintf(
				localizedText("browser ready (%s)", "browser 已就绪（%s）", "browser ready (%s)"),
				firstNonEmpty(strings.TrimSpace(m.adapterCfg.GitLab.Browser.Driver), "default"),
			)
		}
		return fmt.Sprintf(
			localizedText("- GitLab: %s | browser: %s", "- GitLab：%s | browser：%s", "- GitLab: %s | browser: %s"),
			api,
			browser,
		)
	case gitplatform.PlatformBitbucket:
		api := localizedText("missing token", "缺少 token", "missing token")
		if strings.TrimSpace(m.platformCfg.BitbucketToken) != "" {
			api = localizedText("API ready", "API 已就绪", "API ready")
		}
		browser := localizedText("disabled", "已禁用", "disabled")
		if m.adapterCfg.Bitbucket.Browser.Enabled {
			browser = fmt.Sprintf(
				localizedText("browser ready (%s)", "browser 已就绪（%s）", "browser ready (%s)"),
				firstNonEmpty(strings.TrimSpace(m.adapterCfg.Bitbucket.Browser.Driver), "default"),
			)
		}
		return fmt.Sprintf(
			localizedText("- Bitbucket: %s | browser: %s", "- Bitbucket：%s | browser：%s", "- Bitbucket: %s | browser: %s"),
			api,
			browser,
		)
	default:
		return fmt.Sprintf("- %s", platformID.String())
	}
}

func (m Model) summarizePlatformFailure(op *git.PlatformExecInfo, err error) (string, string) {
	raw := strings.TrimSpace(err.Error())
	capability := humanCapabilityLabel(firstNonEmpty(op.CapabilityID, "platform"))
	problems := make([]string, 0, 4)
	addProblem := func(label string) {
		label = strings.TrimSpace(label)
		if label == "" {
			return
		}
		for _, existing := range problems {
			if existing == label {
				return
			}
		}
		problems = append(problems, label)
	}

	lower := strings.ToLower(raw)
	if strings.Contains(lower, "token is not configured") {
		addProblem(localizedText("missing API token", "缺少 API token", "missing API token"))
	}
	if strings.Contains(lower, "gh adapter disabled") {
		addProblem(localizedText("gh adapter disabled", "gh 适配器已禁用", "gh adapter disabled"))
	}
	if strings.Contains(lower, "executable file not found") || strings.Contains(lower, "not found in %path%") || strings.Contains(lower, "\"gh\": executable file not found") {
		addProblem(localizedText("gh CLI not found", "未找到 gh CLI", "gh CLI not found"))
	}
	if strings.Contains(lower, "browser adapter disabled") {
		addProblem(localizedText("browser adapter disabled", "browser 适配器已禁用", "browser adapter disabled"))
	}
	if strings.Contains(lower, "diagnostic blocked execution") {
		addProblem(localizedText("blocked by diagnostics", "被诊断规则拦截", "blocked by diagnostics"))
	}
	if strings.Contains(lower, "cannot handle capability") || strings.Contains(lower, "executor") && strings.Contains(lower, "unavailable") {
		addProblem(localizedText("executor unavailable", "执行器不可用", "executor unavailable"))
	}
	if len(problems) == 0 {
		addProblem(oneLine(raw))
	}

	summary := fmt.Sprintf(
		localizedText("%s unavailable: %s", "%s 当前不可用：%s", "%s unavailable: %s"),
		capability,
		strings.Join(problems, localizedText(", ", "，", ", ")),
	)

	nextSteps := []string{
		"/config status",
		"/config platform",
		"/refresh",
	}
	if fields := platformPlaceholderInputFields(op); len(fields) > 0 {
		nextSteps = append([]string{"/accept"}, nextSteps...)
	}
	if strings.Contains(lower, "token is not configured") {
		nextSteps = append(nextSteps, "platform.github_token / platform.gitlab_token / platform.bitbucket_token")
	}
	if strings.Contains(lower, "gh") {
		nextSteps = append(nextSteps, "adapters.github.gh.enabled / adapters.github.gh.binary")
	}
	if strings.Contains(lower, "browser") {
		nextSteps = append(nextSteps, "adapters.*.browser.enabled / adapters.*.browser.driver")
	}

	bodyLines := []string{
		summary,
		"",
		localizedText("Action:", "动作：", "Action:"),
		platformSuggestionCommand(op),
		"",
		localizedText("Details:", "详细信息：", "Details:"),
		raw,
	}
	if fields := platformPlaceholderInputFields(op); len(fields) > 0 {
		bodyLines = append(bodyLines,
			"",
			localizedText("Missing values:", "缺少的值：", "Missing values:"),
		)
		for _, field := range fields {
			bodyLines = append(bodyLines, "- "+field.Label)
		}
	}
	bodyLines = append(bodyLines,
		"",
		localizedText("Next steps:", "下一步：", "Next steps:"),
	)
	for _, step := range nextSteps {
		bodyLines = append(bodyLines, "- "+step)
	}
	return summary, strings.Join(bodyLines, "\n")
}
