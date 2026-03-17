package knowledge

import (
	"embed"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
)

//go:embed data/knowledge/*.yaml
var knowledgeFS embed.FS

type scenarioFile struct {
	Scenarios []scenario `yaml:"scenarios"`
}

type scenario struct {
	ID       string         `yaml:"id"`
	Summary  string         `yaml:"summary"`
	Triggers map[string]any `yaml:"triggers"`
	SOP      string         `yaml:"sop"`
	Pitfalls string         `yaml:"pitfalls"`
}

// Extract writes all embedded knowledge YAML files to disk under
// .gitdex/maintain/knowledge/ and generates index.yaml.
func Extract(store *dotgitdex.Manager) error {
	knDir := store.KnowledgeDir()
	if err := os.MkdirAll(knDir, 0o755); err != nil {
		return err
	}

	entries, err := knowledgeFS.ReadDir("data/knowledge")
	if err != nil {
		return err
	}

	var indexEntries []dotgitdex.IndexEntry

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		data, err := knowledgeFS.ReadFile("data/knowledge/" + entry.Name())
		if err != nil {
			continue
		}
		destPath := filepath.Join(knDir, entry.Name())
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			continue
		}

		var sf scenarioFile
		if yaml.Unmarshal(data, &sf) != nil {
			continue
		}

		source := strings.TrimSuffix(entry.Name(), ".yaml")
		domain := guessDomain(source)

		for _, s := range sf.Scenarios {
			tags := triggerKeys(s.Triggers)
			summary := strings.TrimSpace(s.Summary)
			if summary == "" {
				summary = s.ID
			}
			indexEntries = append(indexEntries, dotgitdex.IndexEntry{
				KnowledgeID: source + "#" + s.ID,
				Path:        destPath,
				Title:       s.ID,
				Description: summary,
				Tags:        tags,
				Domain:      domain,
				TrustLevel:  "builtin",
			})
		}
	}

	return store.WriteIndex(indexEntries)
}

func guessDomain(source string) string {
	switch {
	case strings.Contains(source, "github"):
		return "github"
	case strings.Contains(source, "bitbucket"):
		return "bitbucket"
	case strings.Contains(source, "gitlab"):
		return "gitlab"
	default:
		return "git"
	}
}

func triggerKeys(triggers map[string]any) []string {
	keys := make([]string, 0, len(triggers))
	for k := range triggers {
		keys = append(keys, k)
	}
	return keys
}
