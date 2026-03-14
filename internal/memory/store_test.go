package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestToPromptMemoryIncludesRecentEventsAndArtifacts(t *testing.T) {
	store := &Store{
		data: MemoryData{
			Version:     1,
			Preferences: map[string]string{"theme": "dark"},
			Repos: map[string]*RepoMemory{
				"repo": {
					Fingerprint:   "repo",
					RecentEvents:  []string{"goal:setup deployment", "command succeeded: git status"},
					ArtifactNotes: []string{"create .github/workflows/deploy.yml"},
				},
			},
		},
	}

	mem := store.ToPromptMemory("repo")
	assert.Equal(t, []string{"goal:setup deployment", "command succeeded: git status"}, mem.RecentEvents)
	assert.Equal(t, []string{"create .github/workflows/deploy.yml"}, mem.ArtifactNotes)
}

func TestToPromptMemoryIncludesThreeTierMemory(t *testing.T) {
	store := &Store{
		data: MemoryData{
			Version:     1,
			Preferences: map[string]string{"theme": "dark"},
			Repos: map[string]*RepoMemory{
				"repo": {
					Fingerprint: "repo",
					Episodes: []Episode{{
						Surface:  "platform",
						Summary:  "platform success: pages mutate",
						Result:   "success",
						Evidence: []string{"platform success: pages mutate"},
					}},
					SemanticFacts: []Fact{{
						Fact:       "preferred commit style: conventional",
						Confidence: 0.91,
						Evidence:   []string{"conventional"},
					}},
					Task: &Task{
						Goal:       "ship pages",
						WorkflowID: "pages_setup",
						Status:     "in_progress",
						Pending:    []string{"inspect pages", "validate domain"},
					},
				},
			},
		},
	}

	mem := store.ToPromptMemory("repo")
	if assert.Len(t, mem.Episodes, 1) {
		assert.Equal(t, "platform", mem.Episodes[0].Surface)
	}
	if assert.Len(t, mem.SemanticFacts, 1) {
		assert.Equal(t, "preferred commit style: conventional", mem.SemanticFacts[0].Fact)
	}
	if assert.NotNil(t, mem.TaskState) {
		assert.Equal(t, "ship pages", mem.TaskState.Goal)
		assert.Equal(t, "pages_setup", mem.TaskState.WorkflowID)
	}
}

func TestRecordOperationEventCompressesEpisodesIntoSemanticFacts(t *testing.T) {
	store := &Store{
		data: MemoryData{
			Version:     1,
			Preferences: map[string]string{},
			Repos:       map[string]*RepoMemory{},
		},
	}

	for i := 0; i < 28; i++ {
		store.RecordOperationEvent("repo", "automation:sync:ahead0->1:behind0->0")
	}
	mem := store.ToPromptMemory("repo")

	assert.NotEmpty(t, mem.Episodes)
	assert.NotEmpty(t, mem.SemanticFacts)
	assert.Contains(t, mem.SemanticFacts[0].Fact, "repeated automation activity observed")
}

func TestRecordTypedEventStoresStructuredEvidenceRefs(t *testing.T) {
	store := &Store{
		data: MemoryData{
			Version:     1,
			Preferences: map[string]string{},
			Repos:       map[string]*RepoMemory{},
		},
	}

	store.RecordTypedEvent("repo", Episode{
		At:           time.Unix(1710000000, 0).UTC(),
		Kind:         "platform_action",
		Surface:      "platform",
		Action:       "platform_success",
		Summary:      "platform success: pages:validate:domain",
		Result:       "success",
		WorkflowID:   "pages_setup",
		CapabilityID: "pages",
		Flow:         "validate",
		Operation:    "domain",
		EvidenceRefs: []EvidenceRef{{Kind: "capability", Ref: "pages", Label: "pages"}},
	})

	snapshot := store.Snapshot()
	if assert.Len(t, snapshot.Repos["repo"].Episodes, 1) {
		episode := snapshot.Repos["repo"].Episodes[0]
		assert.Equal(t, "pages_setup", episode.WorkflowID)
		assert.Equal(t, "pages", episode.CapabilityID)
		assert.Equal(t, "validate", episode.Flow)
		assert.Equal(t, "domain", episode.Operation)
		if assert.Len(t, episode.EvidenceRefs, 1) {
			assert.Equal(t, "capability", episode.EvidenceRefs[0].Kind)
			assert.Equal(t, "pages", episode.EvidenceRefs[0].Ref)
		}
	}
}

func TestToPromptMemoryRanksWorkflowRelevantEpisodesFirst(t *testing.T) {
	store := &Store{
		data: MemoryData{
			Version:     1,
			Preferences: map[string]string{},
			Repos: map[string]*RepoMemory{
				"repo": {
					Fingerprint: "repo",
					Task: &Task{
						Goal:       "validate pages domain",
						WorkflowID: "pages_setup",
						Status:     "in_progress",
					},
					Episodes: []Episode{
						{
							At:         time.Now().Add(-1 * time.Hour),
							Kind:       "git_command",
							Surface:    "git",
							Action:     "command_success",
							Summary:    "command succeeded: git status",
							Result:     "success",
							Confidence: 0.75,
						},
						{
							At:           time.Now().Add(-18 * time.Hour),
							Kind:         "platform_action",
							Surface:      "platform",
							Action:       "platform_success",
							Summary:      "platform success: pages:validate:domain",
							Result:       "success",
							WorkflowID:   "pages_setup",
							CapabilityID: "pages",
							Flow:         "validate",
							Operation:    "domain",
							Confidence:   0.84,
							EvidenceRefs: []EvidenceRef{{Kind: "workflow", Ref: "pages_setup", Label: "pages_setup"}},
						},
					},
				},
			},
		},
		settings: memorySettings{
			maxRecentEvents:      defaultMaxRecentEvents,
			maxArtifactNotes:     defaultMaxArtifactNotes,
			maxEpisodes:          defaultMaxEpisodes,
			maxPromptEpisodes:    2,
			maxSemanticFacts:     defaultMaxSemanticFacts,
			maxPromptFacts:       defaultMaxPromptFacts,
			maxEvidence:          defaultMaxEvidence,
			compressionThreshold: defaultCompressionThreshold,
			minSemanticScore:     defaultMinSemanticScore,
			defaultSemanticDecay: defaultSemanticDecay,
			maxTaskConstraints:   defaultMaxTaskConstraints,
			maxTaskPending:       defaultMaxTaskPending,
		},
	}

	mem := store.ToPromptMemory("repo")
	if assert.Len(t, mem.Episodes, 2) {
		assert.Equal(t, "pages_setup", mem.Episodes[0].WorkflowID)
		assert.Equal(t, "pages", mem.Episodes[0].CapabilityID)
	}
}

func TestToPromptMemoryAnnotatesStaleSemanticFacts(t *testing.T) {
	store := &Store{
		data: MemoryData{
			Version:     1,
			Preferences: map[string]string{},
			Repos: map[string]*RepoMemory{
				"repo": {
					Fingerprint: "repo",
					SemanticFacts: []Fact{{
						Fact:          "preferred validation flow: pages_setup",
						Confidence:    0.94,
						Evidence:      []string{"pages_setup"},
						EvidenceRefs:  []EvidenceRef{{Kind: "workflow", Ref: "pages_setup", Label: "pages_setup"}},
						LastValidated: time.Now().Add(-45 * 24 * time.Hour),
						Decay:         0.01,
					}},
				},
			},
		},
		settings: resolveMemorySettings(nil),
	}

	mem := store.ToPromptMemory("repo")
	if assert.Len(t, mem.SemanticFacts, 1) {
		assert.True(t, mem.SemanticFacts[0].Stale)
		assert.Less(t, mem.SemanticFacts[0].CurrentScore, mem.SemanticFacts[0].Confidence)
		assert.Equal(t, "workflow", mem.SemanticFacts[0].EvidenceRefs[0].Kind)
	}
}
