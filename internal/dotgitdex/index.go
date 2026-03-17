package dotgitdex

import (
	"os"

	"gopkg.in/yaml.v3"
)

// IndexEntry represents one knowledge file in index.yaml.
type IndexEntry struct {
	KnowledgeID string   `yaml:"knowledge_id"`
	Path        string   `yaml:"path"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"short_description"`
	Tags        []string `yaml:"tags,omitempty"`
	Domain      string   `yaml:"domain"` // git, github, ci, pages, actions, docs
	Priority    int      `yaml:"priority,omitempty"`
	TrustLevel  string   `yaml:"trust_level,omitempty"` // builtin, user
}

type indexFile struct {
	Entries []IndexEntry `yaml:"entries"`
}

// WriteIndex writes index entries to maintain/index.yaml.
func (m *Manager) WriteIndex(entries []IndexEntry) error {
	data, err := yaml.Marshal(indexFile{Entries: entries})
	if err != nil {
		return err
	}
	return os.WriteFile(m.IndexPath(), data, 0o644)
}

// ReadIndex reads all index entries from maintain/index.yaml.
func (m *Manager) ReadIndex() ([]IndexEntry, error) {
	data, err := os.ReadFile(m.IndexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f indexFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Entries, nil
}

// ReadIndexText returns the raw index.yaml content as a string for LLM consumption.
func (m *Manager) ReadIndexText() (string, error) {
	data, err := os.ReadFile(m.IndexPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}
