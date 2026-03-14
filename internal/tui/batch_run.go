package tui

import (
	"fmt"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
)

type batchRunCandidate struct {
	Index    int
	Action   string
	Runnable bool
	Reason   string
}

func suggestionCommandForExecution(s git.Suggestion) []string {
	if len(s.Command) > 0 {
		return append([]string(nil), s.Command...)
	}
	if len(s.Steps) > 0 && len(s.Steps[0]) > 0 {
		return append([]string(nil), s.Steps[0]...)
	}
	return nil
}

func (m Model) batchRunCandidates(force bool) []batchRunCandidate {
	candidates := make([]batchRunCandidate, 0, len(m.suggestions))
	for idx, suggestion := range m.suggestions {
		candidates = append(candidates, m.batchRunCandidateForSuggestion(idx, suggestion, force))
	}
	return candidates
}

func (m Model) batchRunCandidateForSuggestion(idx int, suggestion git.Suggestion, force bool) batchRunCandidate {
	candidate := batchRunCandidate{
		Index:  idx,
		Action: strings.TrimSpace(suggestion.Action),
	}
	if idx < len(m.suggExecState) && m.suggExecState[idx] != git.ExecPending {
		candidate.Reason = localizedText("already processed", "已处理", "already processed")
		return candidate
	}

	switch suggestion.Interaction {
	case git.InfoOnly:
		candidate.Runnable = true
		return candidate
	case git.FileWrite:
		if suggestion.FileOp == nil {
			candidate.Reason = localizedText("missing file metadata", "缺少文件元数据", "missing file metadata")
			return candidate
		}
		candidate.Runnable = true
		return candidate
	case git.NeedsInput:
		candidate.Reason = localizedText("manual input required", "需要手动输入", "manual input required")
		return candidate
	case git.CommitMessage, git.ConflictGuide, git.RecoveryGuide:
		candidate.Reason = localizedText("manual review required", "需要人工处理", "manual review required")
		return candidate
	case git.PlatformExec:
		if suggestion.PlatformOp == nil {
			candidate.Reason = localizedText("missing platform metadata", "缺少平台元数据", "missing platform metadata")
			return candidate
		}
		if len(suggestion.Inputs) > 0 {
			candidate.Reason = localizedText("manual input required", "需要手动输入", "manual input required")
			return candidate
		}
		if !force {
			allowed, reason := m.shouldAllowAutomationSuggestion(suggestion)
			if !allowed {
				candidate.Reason = strings.TrimSpace(firstNonEmpty(reason, localizedText("blocked by policy", "被策略阻止", "blocked by policy")))
				return candidate
			}
		}
		if _, err := m.preflightPlatformRequest(platformExecRequest{Op: clonePlatformExecInfo(suggestion.PlatformOp)}); err != nil {
			candidate.Reason = strings.TrimSpace(err.Error())
			return candidate
		}
		candidate.Runnable = true
		return candidate
	default:
		command := suggestionCommandForExecution(suggestion)
		if len(command) == 0 {
			candidate.Reason = localizedText("no executable command", "没有可执行命令", "no executable command")
			return candidate
		}
		if !force {
			copySuggestion := suggestion
			copySuggestion.Command = command
			allowed, reason := m.shouldAllowAutomationSuggestion(copySuggestion)
			if !allowed {
				candidate.Reason = strings.TrimSpace(firstNonEmpty(reason, localizedText("blocked by policy", "被策略阻止", "blocked by policy")))
				return candidate
			}
		}
		candidate.Runnable = true
		return candidate
	}
}

func (m Model) batchRunSummary(force bool) string {
	candidates := m.batchRunCandidates(force)
	runnable := make([]string, 0, len(candidates))
	blocked := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Runnable {
			runnable = append(runnable, "- "+firstNonEmpty(candidate.Action, fmt.Sprintf("#%d", candidate.Index+1)))
			continue
		}
		if strings.TrimSpace(candidate.Reason) == "" || candidate.Reason == localizedText("already processed", "已处理", "already processed") {
			continue
		}
		blocked = append(blocked, fmt.Sprintf("- %s: %s", firstNonEmpty(candidate.Action, fmt.Sprintf("#%d", candidate.Index+1)), candidate.Reason))
	}

	lines := []string{
		fmt.Sprintf(localizedText("Runnable now: %d", "当前可执行：%d", "Runnable now: %d"), len(runnable)),
	}
	if len(runnable) > 0 {
		lines = append(lines, localizedText("Will run in order:", "将按顺序执行：", "Will run in order:"))
		lines = append(lines, runnable...)
	}
	if len(blocked) > 0 {
		lines = append(lines, "")
		lines = append(lines, localizedText("Not runnable yet:", "暂时不可执行：", "Not runnable yet:"))
		lines = append(lines, blocked...)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
