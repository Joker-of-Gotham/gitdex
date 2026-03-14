package tui

import (
	"regexp"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
)

var platformPlaceholderRE = regexp.MustCompile(`<([^>]+)>`)

func platformPlaceholderInputFields(op *git.PlatformExecInfo) []git.InputField {
	if op == nil {
		return nil
	}

	seen := map[string]struct{}{}
	placeholders := make([]string, 0, 4)
	appendMatches := func(text string) {
		for _, match := range platformPlaceholderRE.FindAllStringSubmatch(text, -1) {
			if len(match) < 2 {
				continue
			}
			key := strings.TrimSpace(match[1])
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			placeholders = append(placeholders, key)
		}
	}

	appendMatches(op.ResourceID)
	for _, value := range op.Scope {
		appendMatches(value)
	}
	for _, value := range op.Query {
		appendMatches(value)
	}
	appendMatches(string(op.Payload))
	appendMatches(string(op.ValidatePayload))
	appendMatches(string(op.RollbackPayload))

	sort.Strings(placeholders)
	fields := make([]git.InputField, 0, len(placeholders))
	for _, key := range placeholders {
		token := "<" + key + ">"
		fields = append(fields, git.InputField{
			Key:         token,
			Label:       platformPlaceholderLabel(key),
			Placeholder: platformPlaceholderHint(key),
			ArgIndex:    -1,
		})
	}
	return fields
}

func ensurePlatformSuggestionInputs(s git.Suggestion) git.Suggestion {
	if s.Interaction != git.PlatformExec || s.PlatformOp == nil || len(s.Inputs) > 0 {
		return s
	}
	s.Inputs = platformPlaceholderInputFields(s.PlatformOp)
	return s
}

func platformInputsNote(fields []git.InputField) string {
	if len(fields) == 0 {
		return ""
	}
	labels := make([]string, 0, len(fields))
	for _, field := range fields {
		labels = append(labels, field.Label)
	}
	return localizedText(
		"Missing values: "+strings.Join(labels, ", ")+". Run /accept to fill them before execution.",
		"缺少以下值："+strings.Join(labels, "、")+"。运行 /accept 后先填写再执行。",
		"Missing values: "+strings.Join(labels, ", ")+". Run /accept to fill them before execution.",
	)
}

func platformPlaceholderLabel(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "repo_owner":
		return localizedText("Repository owner", "仓库所有者", "Repository owner")
	case "repo_name":
		return localizedText("Repository name", "仓库名称", "Repository name")
	case "branch":
		return localizedText("Branch name", "分支名称", "Branch name")
	case "environment":
		return localizedText("Environment name", "环境名称", "Environment name")
	case "domain":
		return localizedText("Domain", "域名", "Domain")
	case "tag":
		return localizedText("Tag", "标签", "Tag")
	default:
		title := cases.Title(language.English).String(strings.ReplaceAll(strings.TrimSpace(key), "_", " "))
		return localizedText(title, strings.ReplaceAll(strings.TrimSpace(key), "_", " "), title)
	}
}

func platformPlaceholderHint(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "repo_owner":
		return localizedText("e.g. octocat", "例如 octocat", "e.g. octocat")
	case "repo_name":
		return localizedText("e.g. hello-world", "例如 hello-world", "e.g. hello-world")
	case "branch":
		return localizedText("e.g. main", "例如 main", "e.g. main")
	case "environment":
		return localizedText("e.g. production", "例如 production", "e.g. production")
	case "domain":
		return localizedText("e.g. docs.example.com", "例如 docs.example.com", "e.g. docs.example.com")
	default:
		return localizedText("Enter a value", "请输入值", "Enter a value")
	}
}
