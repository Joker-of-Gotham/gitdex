package tui

import (
	"os/exec"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func (m Model) prepareSuggestionsForDisplay(in []git.Suggestion) ([]git.Suggestion, []string, int) {
	if len(in) == 0 {
		return nil, nil, 0
	}
	out := make([]git.Suggestion, 0, len(in))
	notes := make([]string, 0, len(in))
	dropped := 0
	for _, suggestion := range in {
		if !m.suggestionRelevantToRepo(suggestion, m.gitState, m.session.ActiveGoal) {
			dropped++
			continue
		}
		suggestion = ensurePlatformSuggestionInputs(suggestion)
		out = append(out, suggestion)
		notes = append(notes, m.previewSuggestionAvailability(suggestion))
	}
	return out, notes, dropped
}

func (m Model) previewSuggestionAvailability(s git.Suggestion) string {
	if s.Interaction != git.PlatformExec || s.PlatformOp == nil {
		return ""
	}

	notes := make([]string, 0, 3)
	if len(s.Inputs) > 0 {
		notes = append(notes, platformInputsNote(s.Inputs))
	}

	diagnostics, _ := platform.DiagnosePlatformOperation(m.detectedPlatform(), m.gitState, clonePlatformExecInfo(s.PlatformOp))
	if diagnostics.Decision == platform.DiagnosticBlocked {
		for _, item := range diagnostics.Items {
			if strings.TrimSpace(item.Detail) == "" {
				continue
			}
			switch strings.TrimSpace(item.Code) {
			case "boundary_inspect_only":
				notes = append(notes, localizedText(
					"Current platform only supports inspection for this surface. Use /why to review the boundary.",
					"当前平台对此能力只支持检查。可用 /why 查看边界原因。",
					"Current platform only supports inspection for this surface. Use /why to review the boundary.",
				))
			case "placeholders_unresolved":
				if len(s.Inputs) == 0 {
					notes = append(notes, localizedText(
						"Platform request still has unresolved placeholders. Run /accept to fill them.",
						"平台请求仍有未填写占位符。运行 /accept 先填写。",
						"Platform request still has unresolved placeholders. Run /accept to fill them.",
					))
				}
			}
		}
	}

	switch strings.TrimSpace(s.PlatformOp.CapabilityID) {
	case "pages", "release", "actions", "codespaces", "notifications", "branch_rulesets":
		if note := m.previewPlatformRouteAvailability(); note != "" {
			notes = append(notes, note)
		}
	}

	return strings.Join(compactStringList(notes, 3), localizedText(" | ", " ｜ ", " | "))
}

func (m Model) previewPlatformRouteAvailability() string {
	hasToken := strings.TrimSpace(m.platformCfg.GitHubToken) != ""
	hasGH := false
	if m.adapterCfg.GitHub.GH.Enabled {
		if binary := strings.TrimSpace(m.adapterCfg.GitHub.GH.Binary); binary != "" {
			_, err := exec.LookPath(binary)
			hasGH = err == nil
		}
	}
	hasBrowser := m.adapterCfg.GitHub.Browser.Enabled

	problems := make([]string, 0, 3)
	if !hasToken {
		problems = append(problems, localizedText("GitHub API token missing", "缺少 GitHub API token", "GitHub API token missing"))
	}
	if m.adapterCfg.GitHub.GH.Enabled && !hasGH {
		problems = append(problems, localizedText("gh CLI unavailable", "gh CLI 不可用", "gh CLI unavailable"))
	}
	if !hasBrowser {
		problems = append(problems, localizedText("browser adapter disabled", "browser 适配器已禁用", "browser adapter disabled"))
	}

	switch {
	case hasToken || hasGH || hasBrowser:
		return ""
	case len(problems) == 0:
		return localizedText("Platform access needs configuration. Run /config status.", "平台访问尚未配置。运行 /config status。", "Platform access needs configuration. Run /config status.")
	default:
		return localizedText(
			"Route unavailable: "+strings.Join(problems, ", ")+". Run /config platform.",
			"访问路径不可用："+strings.Join(problems, "、")+"。运行 /config platform。",
			"Route unavailable: "+strings.Join(problems, ", ")+". Run /config platform.",
		)
	}
}

func (m Model) suggestionRelevantToRepo(_ git.Suggestion, _ *status.GitState, _ string) bool {
	return true
}
