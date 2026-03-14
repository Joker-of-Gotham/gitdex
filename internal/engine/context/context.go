package context

import (
	"embed"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed data/*.yaml
var dataFS embed.FS

var (
	once     sync.Once
	instance *GitContext
)

type PlaceholderData struct {
	ExactPatterns []string `yaml:"exact_patterns"`
	PathPatterns  []string `yaml:"path_patterns"`
}

type SubcommandInfo struct {
	RequiresMessage         bool     `yaml:"requires_message"`
	MessageFlags            []string `yaml:"message_flags"`
	SkipMessageFlags        []string `yaml:"skip_message_flags"`
	DefaultInputLabel       string   `yaml:"default_input_label"`
	DefaultInputPlaceholder string   `yaml:"default_input_placeholder"`
	DefaultLabel            string   `yaml:"default_label"`
}

type LabelHint struct {
	Match []string `yaml:"match"`
	Label string   `yaml:"label"`
}

type TemplateData struct {
	InputDefaults map[string]map[string]string `yaml:"input_defaults"`
	LabelHints    []LabelHint                  `yaml:"label_hints"`
}

type WorkflowDefinition struct {
	ID            string                   `yaml:"id"`
	Label         string                   `yaml:"label"`
	Goal          string                   `yaml:"goal"`
	Prerequisites []string                 `yaml:"prerequisites"`
	Capabilities  []string                 `yaml:"capabilities,omitempty"`
	Prefill       []WorkflowPlatformAction `yaml:"prefill,omitempty"`
}

type WorkflowPlatformAction struct {
	CapabilityID string            `yaml:"capability_id"`
	Flow         string            `yaml:"flow,omitempty"`
	Operation    string            `yaml:"operation,omitempty"`
	ResourceID   string            `yaml:"resource_id,omitempty"`
	Scope        map[string]string `yaml:"scope,omitempty"`
	Query        map[string]string `yaml:"query,omitempty"`
	Payload      any               `yaml:"payload,omitempty"`
	Validate     any               `yaml:"validate,omitempty"`
	Rollback     any               `yaml:"rollback,omitempty"`
}

type WorkflowData struct {
	Workflows []WorkflowDefinition `yaml:"workflows"`
}

type GitContext struct {
	Placeholders PlaceholderData
	Subcommands  map[string]SubcommandInfo
	Templates    TemplateData
	Workflows    WorkflowData
}

func Get() *GitContext {
	once.Do(func() {
		instance = &GitContext{
			Subcommands: make(map[string]SubcommandInfo),
		}
		load("data/placeholders.yaml", &instance.Placeholders)
		load("data/subcommands.yaml", &instance.Subcommands)
		load("data/templates.yaml", &instance.Templates)
		load("data/workflows.yaml", &instance.Workflows)
	})
	return instance
}

func load(path string, target interface{}) {
	data, err := dataFS.ReadFile(path)
	if err != nil {
		return
	}
	_ = yaml.Unmarshal(data, target)
}

func (c *GitContext) IsPlaceholder(token string) bool {
	if strings.HasPrefix(token, "<") && strings.HasSuffix(token, ">") {
		return true
	}
	lower := strings.ToLower(token)
	for _, p := range c.Placeholders.ExactPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	for _, p := range c.Placeholders.PathPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func (c *GitContext) SubcommandLabel(sub string) string {
	if info, ok := c.Subcommands[strings.ToLower(sub)]; ok && info.DefaultLabel != "" {
		return info.DefaultLabel
	}
	return ""
}

func (c *GitContext) CommitInfo() SubcommandInfo {
	return c.Subcommands["commit"]
}

func (c *GitContext) GuessLabel(token string, subcommand string) string {
	lower := strings.ToLower(token)
	for _, hint := range c.Templates.LabelHints {
		for _, m := range hint.Match {
			if strings.Contains(lower, m) {
				return hint.Label
			}
		}
	}
	if label := c.SubcommandLabel(subcommand); label != "" {
		return label
	}
	cleaned := strings.Trim(token, "<>")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	cleaned = strings.ReplaceAll(cleaned, "_", " ")
	if cleaned != "" {
		return strings.Title(cleaned) //nolint:staticcheck
	}
	return "Value"
}

func (c *GitContext) DefaultPlaceholder(key, label string, preferSSH bool) string {
	combined := strings.ToLower(strings.TrimSpace(key + " " + label))
	if strings.Contains(combined, "remote") || strings.Contains(combined, "url") {
		defaults := c.Templates.InputDefaults["remote_url"]
		if preferSSH {
			if v, ok := defaults["ssh"]; ok {
				return v
			}
		}
		if v, ok := defaults["https"]; ok {
			return v
		}
	}
	if g, ok := c.Templates.InputDefaults["generic"]; ok {
		if v, ok2 := g["default"]; ok2 {
			return v
		}
	}
	return "Enter value..."
}

func (c *GitContext) IsMessageFlag(flag string) bool {
	info := c.CommitInfo()
	for _, f := range info.MessageFlags {
		if flag == f {
			return true
		}
	}
	return false
}

func (c *GitContext) IsSkipMessageFlag(flag string) bool {
	info := c.CommitInfo()
	for _, f := range info.SkipMessageFlags {
		if flag == f {
			return true
		}
	}
	return false
}

func (c *GitContext) WorkflowList() []WorkflowDefinition {
	if c == nil {
		return nil
	}
	out := make([]WorkflowDefinition, 0, len(c.Workflows.Workflows))
	out = append(out, c.Workflows.Workflows...)
	return out
}
