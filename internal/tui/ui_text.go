package tui

import (
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
)

func localizedText(en, zh, ja string) string {
	switch strings.ToLower(strings.TrimSpace(i18n.Lang())) {
	case "zh":
		if strings.TrimSpace(zh) != "" {
			return zh
		}
	case "ja":
		if strings.TrimSpace(ja) != "" {
			return ja
		}
	}
	return en
}

func localizedStatusText(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "platform unavailable":
		return localizedText("Platform action unavailable", "平台操作当前不可用", "Platform action unavailable")
	case "platform failed":
		return localizedText("Platform action failed", "平台操作失败", "Platform action failed")
	case "platform inspect success":
		return localizedText("Platform inspect succeeded", "平台检查成功", "Platform inspect succeeded")
	case "platform mutate success":
		return localizedText("Platform mutation succeeded", "平台变更成功", "Platform mutation succeeded")
	case "platform validate success":
		return localizedText("Platform validation succeeded", "平台校验成功", "Platform validation succeeded")
	case "platform rollback success":
		return localizedText("Platform rollback succeeded", "平台回滚成功", "Platform rollback succeeded")
	case "file success":
		return localizedText("File operation succeeded", "文件操作成功", "File operation succeeded")
	case "file failed":
		return localizedText("File operation failed", "文件操作失败", "File operation failed")
	case "success":
		return localizedText("Succeeded", "成功", "Succeeded")
	case "failed":
		return localizedText("Failed", "失败", "Failed")
	case "advisory viewed":
		return localizedText("Viewed", "已查看", "Viewed")
	default:
		return status
	}
}

func localizedPromptTitle() string {
	return localizedText("Prompt", "输入", "Prompt")
}

func localizedPromptPlaceholder() string {
	return localizedText(
		"Describe what you want gitdex to do next...",
		"描述你希望 gitdex 下一步做什么……",
		"Describe what you want gitdex to do next...",
	)
}

func localizedPromptHintIdle() string {
	return localizedText(
		"Type / to open commands, or enter a goal directly.",
		"输入 / 打开命令列表，或直接输入目标。",
		"Type / to open commands, or enter a goal directly.",
	)
}

func localizedPromptHintActive() string {
	return localizedText(
		"Tab/Up/Down: command suggestions  Enter: run  Esc: unfocus",
		"Tab/上下：命令候选  Enter：执行  Esc：取消聚焦",
		"Tab/Up/Down: command suggestions  Enter: run  Esc: unfocus",
	)
}

func localizedCommandsTitle() string {
	return localizedText("Commands", "命令", "Commands")
}

func localizedAutomationTitle() string {
	return localizedText("Automation", "自动化", "Automation")
}

func localizedAssistantTitle() string {
	return localizedText("Assistant", "助手响应", "Assistant")
}

func localizedLatestResultTitle() string {
	return localizedText("Latest result", "最近结果", "Latest result")
}

func localizedGoalLabel() string {
	return localizedText("Goal", "目标", "Goal")
}

func localizedConfigTitle() string {
	return localizedText("Configuration", "配置", "Configuration")
}

func localizedAccessTitle() string {
	return localizedText("Access status", "访问状态", "Access status")
}
