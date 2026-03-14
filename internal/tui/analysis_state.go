package tui

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
)

func analysisStateFingerprint(state *status.GitState, goal, mode string) string {
	if state == nil {
		return ""
	}
	hasher := fnv.New128a()
	writeFingerprintPart := func(parts ...string) {
		for _, part := range parts {
			_, _ = hasher.Write([]byte(strings.TrimSpace(part)))
			_, _ = hasher.Write([]byte{0})
		}
	}

	writeFingerprintPart(
		strings.TrimSpace(goal),
		strings.TrimSpace(mode),
		state.LocalBranch.Name,
		state.LocalBranch.Upstream,
		fmt.Sprintf("ahead:%d", state.LocalBranch.Ahead),
		fmt.Sprintf("behind:%d", state.LocalBranch.Behind),
		fmt.Sprintf("commits:%d", state.CommitCount),
		fmt.Sprintf("merge:%t", state.MergeInProgress),
		fmt.Sprintf("rebase:%t", state.RebaseInProgress),
		fmt.Sprintf("cherry:%t", state.CherryInProgress),
		fmt.Sprintf("bisect:%t", state.BisectInProgress),
	)

	for _, file := range state.StagingArea {
		writeFingerprintPart("stage", file.Path, string(file.StagingCode), string(file.WorktreeCode))
	}
	for _, file := range state.WorkingTree {
		writeFingerprintPart("worktree", file.Path, string(file.WorktreeCode), string(file.StagingCode))
	}
	for _, remote := range state.RemoteInfos {
		writeFingerprintPart(
			"remote",
			remote.Name,
			remote.FetchURL,
			remote.PushURL,
			fmt.Sprintf("fetch_valid:%t", remote.FetchURLValid),
			fmt.Sprintf("push_valid:%t", remote.PushURLValid),
			fmt.Sprintf("reachable_checked:%t", remote.ReachabilityChecked),
			fmt.Sprintf("reachable:%t", remote.Reachable),
		)
	}
	if state.CommitSummaryInfo != nil {
		writeFingerprintPart(
			fmt.Sprintf("commit_summary:%t", state.CommitSummaryInfo.UsesConventional),
			strings.TrimSpace(state.CommitSummaryInfo.CommitFrequency),
			strings.TrimSpace(state.CommitSummaryInfo.LastCommitRelative),
		)
	}
	if state.FileInspect != nil {
		writeFingerprintPart("file_inspect:yes")
	}
	if state.ConfigInfo != nil {
		writeFingerprintPart("config:yes")
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func platformSuggestionCommand(op *git.PlatformExecInfo) string {
	if op == nil {
		return localizedText("Inspect platform state", "检查平台状态", "Inspect platform state")
	}
	target := humanCapabilityLabel(op.CapabilityID)
	operation := strings.ReplaceAll(strings.TrimSpace(op.Operation), "_", " ")
	view := strings.ReplaceAll(strings.TrimSpace(op.Query["view"]), "_", " ")
	resource := strings.TrimSpace(op.ResourceID)

	switch strings.ToLower(strings.TrimSpace(op.Flow)) {
	case "inspect":
		switch {
		case view != "":
			return localizedText(
				fmt.Sprintf("Inspect %s %s", target, view),
				fmt.Sprintf("检查%s%s", target, localizedSpaceJoin(view)),
				fmt.Sprintf("Inspect %s %s", target, view),
			)
		case resource != "":
			return localizedText(
				fmt.Sprintf("Inspect %s %s", target, resource),
				fmt.Sprintf("检查%s%s", target, localizedSpaceJoin(resource)),
				fmt.Sprintf("Inspect %s %s", target, resource),
			)
		default:
			return localizedText(
				fmt.Sprintf("Inspect %s state", target),
				fmt.Sprintf("检查%s状态", target),
				fmt.Sprintf("Inspect %s state", target),
			)
		}
	case "validate":
		if operation != "" {
			return localizedText(
				fmt.Sprintf("Validate %s %s", target, operation),
				fmt.Sprintf("校验%s%s", target, localizedSpaceJoin(operation)),
				fmt.Sprintf("Validate %s %s", target, operation),
			)
		}
		return localizedText(fmt.Sprintf("Validate %s state", target), fmt.Sprintf("校验%s状态", target), fmt.Sprintf("Validate %s state", target))
	case "rollback":
		if operation != "" {
			return localizedText(
				fmt.Sprintf("Roll back %s %s", target, operation),
				fmt.Sprintf("回滚%s%s", target, localizedSpaceJoin(operation)),
				fmt.Sprintf("Roll back %s %s", target, operation),
			)
		}
		return localizedText(fmt.Sprintf("Roll back %s change", target), fmt.Sprintf("回滚%s变更", target), fmt.Sprintf("Roll back %s change", target))
	case "mutate":
		if operation != "" {
			return localizedText(
				fmt.Sprintf("Change %s via %s", target, operation),
				fmt.Sprintf("通过%s修改%s", operation, target),
				fmt.Sprintf("Change %s via %s", target, operation),
			)
		}
		return localizedText(fmt.Sprintf("Change %s settings", target), fmt.Sprintf("修改%s设置", target), fmt.Sprintf("Change %s settings", target))
	default:
		return localizedText(
			fmt.Sprintf("%s action for %s", strings.TrimSpace(op.Flow), target),
			fmt.Sprintf("%s：%s", strings.TrimSpace(op.Flow), target),
			fmt.Sprintf("%s action for %s", strings.TrimSpace(op.Flow), target),
		)
	}
}

func localizedSpaceJoin(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(targetLanguage())) {
	case "zh":
		return text
	default:
		return " " + text
	}
}

func targetLanguage() string {
	lang := strings.ToLower(strings.TrimSpace(configuredLanguage()))
	if lang == "" || lang == "auto" {
		return strings.ToLower(strings.TrimSpace(i18n.Lang()))
	}
	return lang
}

func humanCapabilityLabel(capabilityID string) string {
	switch strings.TrimSpace(capabilityID) {
	case "pages":
		return localizedText("GitHub Pages", "GitHub Pages", "GitHub Pages")
	case "release":
		return localizedText("release", "发布", "release")
	case "pull_request":
		return localizedText("pull request", "拉取请求", "pull request")
	case "notifications":
		return localizedText("notifications", "通知", "notifications")
	case "actions":
		return localizedText("GitHub Actions", "GitHub Actions", "GitHub Actions")
	case "codespaces":
		return localizedText("Codespaces", "Codespaces", "Codespaces")
	case "branch_rulesets":
		return localizedText("branch rulesets", "分支规则", "branch rulesets")
	default:
		if capabilityID == "" {
			return localizedText("platform", "平台", "platform")
		}
		return strings.ReplaceAll(capabilityID, "_", " ")
	}
}
