package context

import (
	"embed"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data/knowledge/*.yaml
var knowledgeFS embed.FS

type KnowledgeBase struct {
	Scenarios []Scenario
}

type Scenario struct {
	ID       string         `yaml:"id"`
	Triggers map[string]any `yaml:"triggers"`
	SOP      string         `yaml:"sop"`
	Pitfalls string         `yaml:"pitfalls"`
	Source   string         // filename source
}

type knowledgeFile struct {
	Scenarios []struct {
		ID       string         `yaml:"id"`
		Triggers map[string]any `yaml:"triggers"`
		SOP      string         `yaml:"sop"`
		Pitfalls string         `yaml:"pitfalls"`
	} `yaml:"scenarios"`
}

// LoadKnowledgeBase reads all embedded YAML knowledge files.
func LoadKnowledgeBase() *KnowledgeBase {
	kb := &KnowledgeBase{}
	entries, err := knowledgeFS.ReadDir("data/knowledge")
	if err != nil {
		return kb
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := knowledgeFS.ReadFile("data/knowledge/" + entry.Name())
		if err != nil {
			continue
		}
		var kf knowledgeFile
		if yaml.Unmarshal(data, &kf) != nil {
			continue
		}
		source := strings.TrimSuffix(entry.Name(), ".yaml")
		for _, s := range kf.Scenarios {
			kb.Scenarios = append(kb.Scenarios, Scenario{
				ID:       s.ID,
				Triggers: s.Triggers,
				SOP:      s.SOP,
				Pitfalls: s.Pitfalls,
				Source:   source,
			})
		}
	}
	return kb
}

// FormatScenario returns a readable text representation of a scenario.
func FormatScenario(s Scenario) string {
	var parts []string
	if s.SOP != "" {
		parts = append(parts, strings.TrimSpace(s.SOP))
	}
	if s.Pitfalls != "" {
		parts = append(parts, "Pitfalls:\n"+strings.TrimSpace(s.Pitfalls))
	}
	return strings.Join(parts, "\n\n")
}
