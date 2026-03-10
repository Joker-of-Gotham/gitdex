package memory

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSnapshotReturnsDeepCopy(t *testing.T) {
	store := &Store{
		data: MemoryData{
			Version:     1,
			Preferences: map[string]string{"theme": "dark"},
			Repos: map[string]*RepoMemory{
				"repo": {
					Fingerprint:    "repo",
					Patterns:       []string{"workflow:sync"},
					ResolvedIssues: []string{"ship release"},
				},
			},
		},
	}

	snapshot := store.Snapshot()
	snapshot.Preferences["theme"] = "light"
	snapshot.Repos["repo"].Patterns[0] = "workflow:cleanup"
	snapshot.Repos["repo"].ResolvedIssues[0] = "other"

	assert.Equal(t, "dark", store.data.Preferences["theme"])
	assert.Equal(t, "workflow:sync", store.data.Repos["repo"].Patterns[0])
	assert.Equal(t, "ship release", store.data.Repos["repo"].ResolvedIssues[0])
}

func TestSaveWritesFile(t *testing.T) {
	dir := t.TempDir()
	store := &Store{
		data: MemoryData{
			Version:     1,
			Preferences: map[string]string{"language": "zh"},
			Repos:       map[string]*RepoMemory{},
		},
		path: filepath.Join(dir, "memory.json"),
	}

	err := store.Save()
	assert.NoError(t, err)

	data, err := os.ReadFile(store.path)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"language\": \"zh\"")
	assert.False(t, store.Snapshot().UpdatedAt.IsZero())
}
