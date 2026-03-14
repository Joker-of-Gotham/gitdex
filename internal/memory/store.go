package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
)

const memoryFileName = "memory.json"
const maxRepoEntries = 10
const maxPatterns = 20
const maxResolvedHistory = 30
const defaultMaxRecentEvents = 40
const defaultMaxArtifactNotes = 20
const defaultMaxEpisodes = 24
const defaultMaxPromptEpisodes = 8
const defaultMaxSemanticFacts = 12
const defaultMaxPromptFacts = 8
const defaultMaxEvidence = 6
const defaultCompressionThreshold = 20
const defaultMinSemanticScore = 0.10
const defaultSemanticDecay = 0.02
const defaultMaxTaskConstraints = 8
const defaultMaxTaskPending = 8
const memoryFileMode = 0o600

// Store manages persistent long-term memory across sessions.
type Store struct {
	mu       sync.RWMutex
	data     MemoryData
	path     string
	settings memorySettings
}

type memorySettings struct {
	maxRecentEvents      int
	maxArtifactNotes     int
	maxEpisodes          int
	maxPromptEpisodes    int
	maxSemanticFacts     int
	maxPromptFacts       int
	maxEvidence          int
	compressionThreshold int
	minSemanticScore     float64
	defaultSemanticDecay float64
	maxTaskConstraints   int
	maxTaskPending       int
}

type MemoryData struct {
	Version     int                    `json:"version"`
	Preferences map[string]string      `json:"preferences,omitempty"`
	Repos       map[string]*RepoMemory `json:"repos,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type RepoMemory struct {
	Fingerprint    string    `json:"fingerprint"`
	Patterns       []string  `json:"patterns,omitempty"`
	ResolvedIssues []string  `json:"resolved_issues,omitempty"`
	RecentEvents   []string  `json:"recent_events,omitempty"`
	ArtifactNotes  []string  `json:"artifact_notes,omitempty"`
	Episodes       []Episode `json:"episodes,omitempty"`
	SemanticFacts  []Fact    `json:"semantic_facts,omitempty"`
	Task           *Task     `json:"task,omitempty"`
	CommitStyle    string    `json:"commit_style,omitempty"`
	BranchNaming   string    `json:"branch_naming,omitempty"`
	LastSummarized time.Time `json:"last_summarized,omitempty"`
	LastAccessed   time.Time `json:"last_accessed"`
}

type EvidenceRef struct {
	Kind  string `json:"kind,omitempty"`
	Ref   string `json:"ref,omitempty"`
	Label string `json:"label,omitempty"`
}

type Episode struct {
	ID           string        `json:"id,omitempty"`
	At           time.Time     `json:"at,omitempty"`
	Kind         string        `json:"kind,omitempty"`
	Surface      string        `json:"surface,omitempty"`
	Action       string        `json:"action,omitempty"`
	Summary      string        `json:"summary,omitempty"`
	Result       string        `json:"result,omitempty"`
	WorkflowID   string        `json:"workflow_id,omitempty"`
	CapabilityID string        `json:"capability_id,omitempty"`
	Flow         string        `json:"flow,omitempty"`
	Operation    string        `json:"operation,omitempty"`
	Confidence   float64       `json:"confidence,omitempty"`
	Evidence     []string      `json:"evidence,omitempty"`
	EvidenceRefs []EvidenceRef `json:"evidence_refs,omitempty"`
	LedgerID     string        `json:"ledger_id,omitempty"`
}

type Fact struct {
	Fact          string        `json:"fact,omitempty"`
	Confidence    float64       `json:"confidence,omitempty"`
	Evidence      []string      `json:"evidence,omitempty"`
	EvidenceRefs  []EvidenceRef `json:"evidence_refs,omitempty"`
	LastValidated time.Time     `json:"last_validated,omitempty"`
	Decay         float64       `json:"decay,omitempty"`
}

type Task struct {
	Goal        string    `json:"goal,omitempty"`
	WorkflowID  string    `json:"workflow_id,omitempty"`
	Status      string    `json:"status,omitempty"`
	Constraints []string  `json:"constraints,omitempty"`
	Pending     []string  `json:"pending,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// NewStore creates a memory store, loading from disk if available.
func NewStore() *Store {
	s := &Store{
		data: MemoryData{
			Version:     1,
			Preferences: map[string]string{},
			Repos:       map[string]*RepoMemory{},
		},
		settings: resolveMemorySettings(config.Get()),
	}
	s.path = defaultPath()
	s.load()
	return s
}

func resolveMemorySettings(cfg *config.Config) memorySettings {
	settings := memorySettings{
		maxRecentEvents:      defaultMaxRecentEvents,
		maxArtifactNotes:     defaultMaxArtifactNotes,
		maxEpisodes:          defaultMaxEpisodes,
		maxPromptEpisodes:    defaultMaxPromptEpisodes,
		maxSemanticFacts:     defaultMaxSemanticFacts,
		maxPromptFacts:       defaultMaxPromptFacts,
		maxEvidence:          defaultMaxEvidence,
		compressionThreshold: defaultCompressionThreshold,
		minSemanticScore:     defaultMinSemanticScore,
		defaultSemanticDecay: defaultSemanticDecay,
		maxTaskConstraints:   defaultMaxTaskConstraints,
		maxTaskPending:       defaultMaxTaskPending,
	}
	if cfg == nil {
		return settings
	}
	if v := cfg.Memory.Episodic.MaxRecentEvents; v > 0 {
		settings.maxRecentEvents = v
	}
	if v := cfg.Memory.Episodic.MaxArtifactNotes; v > 0 {
		settings.maxArtifactNotes = v
	}
	if v := cfg.Memory.Episodic.MaxEpisodes; v > 0 {
		settings.maxEpisodes = v
	}
	if v := cfg.Memory.Episodic.MaxPromptEpisodes; v > 0 {
		settings.maxPromptEpisodes = v
	}
	if v := cfg.Memory.Episodic.CompressionThreshold; v > 0 {
		settings.compressionThreshold = v
	}
	if v := cfg.Memory.Semantic.MaxFacts; v > 0 {
		settings.maxSemanticFacts = v
	}
	if v := cfg.Memory.Semantic.MaxPromptFacts; v > 0 {
		settings.maxPromptFacts = v
	}
	if v := cfg.Memory.Semantic.MaxEvidence; v > 0 {
		settings.maxEvidence = v
	}
	if v := cfg.Memory.Semantic.MinScore; v >= 0 {
		settings.minSemanticScore = v
	}
	if v := cfg.Memory.Semantic.DefaultDecay; v > 0 {
		settings.defaultSemanticDecay = v
	}
	if v := cfg.Memory.Task.MaxConstraints; v > 0 {
		settings.maxTaskConstraints = v
	}
	if v := cfg.Memory.Task.MaxPending; v > 0 {
		settings.maxTaskPending = v
	}
	return settings
}

func (s *Store) effectiveSettings() memorySettings {
	settings := s.settings
	if settings.maxRecentEvents <= 0 {
		return resolveMemorySettings(config.Get())
	}
	return settings
}

func defaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "."+config.AppName, memoryFileName)
}

func legacyDefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "."+config.LegacyAppName, memoryFileName)
}

func (s *Store) load() {
	path := s.path
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		legacyPath := legacyDefaultPath()
		if legacyPath == "" || legacyPath == path {
			return
		}
		data, err = os.ReadFile(legacyPath)
		if err != nil {
			return
		}
	}
	var m MemoryData
	if json.Unmarshal(data, &m) == nil && m.Version > 0 {
		s.data = m
		if s.data.Preferences == nil {
			s.data.Preferences = map[string]string{}
		}
		if s.data.Repos == nil {
			s.data.Repos = map[string]*RepoMemory{}
		}
	}
}

// Save persists memory to disk.
func (s *Store) Save() error {
	if s.path == "" {
		return nil
	}
	s.mu.Lock()
	s.data.UpdatedAt = time.Now()
	snapshot := cloneMemoryData(s.data)
	s.mu.Unlock()
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, memoryFileMode)
}

// Snapshot returns a defensive copy of the current memory state.
func (s *Store) Snapshot() MemoryData {
	if s == nil {
		return MemoryData{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneMemoryData(s.data)
}

// Path returns the backing file path for the store.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

// SetPreference stores a global user preference.
func (s *Store) SetPreference(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Preferences[key] = value
}

// GetPreference retrieves a global preference.
func (s *Store) GetPreference(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Preferences[key]
}

// RecordRepoPattern records a pattern for the current repo.
func (s *Store) RecordRepoPattern(fingerprint, pattern string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := s.effectiveSettings()
	repo := s.ensureRepo(fingerprint)
	for _, p := range repo.Patterns {
		if p == pattern {
			return
		}
	}
	repo.Patterns = append(repo.Patterns, pattern)
	if len(repo.Patterns) > maxPatterns {
		repo.Patterns = repo.Patterns[len(repo.Patterns)-maxPatterns:]
	}
	upsertFact(repo, Fact{
		Fact:          "repo pattern: " + pattern,
		Confidence:    0.7,
		Evidence:      []string{pattern},
		EvidenceRefs:  []EvidenceRef{{Kind: "pattern", Ref: pattern, Label: pattern}},
		LastValidated: time.Now(),
		Decay:         s.settings.defaultSemanticDecay,
	}, settings)
}

// RecordResolvedIssue records that an issue was resolved in this repo.
func (s *Store) RecordResolvedIssue(fingerprint, issue string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := s.effectiveSettings()
	repo := s.ensureRepo(fingerprint)
	repo.ResolvedIssues = append(repo.ResolvedIssues, issue)
	if len(repo.ResolvedIssues) > maxResolvedHistory {
		repo.ResolvedIssues = repo.ResolvedIssues[len(repo.ResolvedIssues)-maxResolvedHistory:]
	}
	upsertFact(repo, Fact{
		Fact:          "resolved issue: " + issue,
		Confidence:    0.95,
		Evidence:      []string{issue},
		EvidenceRefs:  []EvidenceRef{{Kind: "issue", Ref: issue, Label: issue}},
		LastValidated: time.Now(),
		Decay:         0.01,
	}, settings)
}

// RecordOperationEvent records a recent episodic event for the current repo.
func (s *Store) RecordOperationEvent(fingerprint, event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := s.effectiveSettings()
	event = strings.TrimSpace(event)
	if event == "" {
		return
	}
	repo := s.ensureRepo(fingerprint)
	s.recordEpisodeLocked(repo, newEpisodeFromEvent(event, repo.Task, settings.maxEvidence), settings)
}

// RecordArtifactNote stores an artifact-oriented note for the current repo.
func (s *Store) RecordArtifactNote(fingerprint, note string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := s.effectiveSettings()
	note = strings.TrimSpace(note)
	if note == "" {
		return
	}
	repo := s.ensureRepo(fingerprint)
	repo.ArtifactNotes = append(repo.ArtifactNotes, note)
	if len(repo.ArtifactNotes) > settings.maxArtifactNotes {
		repo.ArtifactNotes = repo.ArtifactNotes[len(repo.ArtifactNotes)-settings.maxArtifactNotes:]
	}
	s.recordEpisodeLocked(repo, Episode{
		At:         time.Now(),
		Kind:       "artifact",
		Surface:    "artifact",
		Action:     "note",
		Summary:    note,
		Result:     "noted",
		Confidence: 0.72,
		Evidence:   []string{note},
		EvidenceRefs: []EvidenceRef{{
			Kind:  "artifact_note",
			Ref:   note,
			Label: note,
		}},
	}, settings)
}

// RecordTypedEvent records a fully structured episodic event for the current repo.
func (s *Store) RecordTypedEvent(fingerprint string, episode Episode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := s.effectiveSettings()
	repo := s.ensureRepo(fingerprint)
	s.recordEpisodeLocked(repo, normalizeEpisode(episode, repo.Task, settings.maxEvidence), settings)
}

// SetRepoCommitStyle records the detected commit style for this repo.
func (s *Store) SetRepoCommitStyle(fingerprint, style string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := s.effectiveSettings()
	repo := s.ensureRepo(fingerprint)
	repo.CommitStyle = style
	upsertFact(repo, Fact{
		Fact:          "preferred commit style: " + style,
		Confidence:    0.85,
		Evidence:      []string{style},
		EvidenceRefs:  []EvidenceRef{{Kind: "preference", Ref: "commit_style", Label: style}},
		LastValidated: time.Now(),
		Decay:         s.settings.defaultSemanticDecay,
	}, settings)
}

// UpdateTaskState stores the active task layer for the current repo.
func (s *Store) UpdateTaskState(fingerprint, goal, workflowID, status string, constraints, pending []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings := s.effectiveSettings()
	repo := s.ensureRepo(fingerprint)
	goal = strings.TrimSpace(goal)
	workflowID = strings.TrimSpace(workflowID)
	status = strings.TrimSpace(status)
	constraints = compactStrings(constraints, settings.maxTaskConstraints)
	pending = compactStrings(pending, settings.maxTaskPending)
	if goal == "" && workflowID == "" && status == "" && len(constraints) == 0 && len(pending) == 0 {
		repo.Task = nil
		return
	}
	repo.Task = &Task{
		Goal:        goal,
		WorkflowID:  workflowID,
		Status:      status,
		Constraints: append([]string(nil), constraints...),
		Pending:     append([]string(nil), pending...),
		UpdatedAt:   time.Now(),
	}
}

// ToPromptMemory converts stored memory into prompt-injectable format.
func (s *Store) ToPromptMemory(fingerprint string) *prompt.MemoryContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	settings := s.effectiveSettings()

	mc := &prompt.MemoryContext{
		UserPreferences: map[string]string{},
	}
	for k, v := range s.data.Preferences {
		mc.UserPreferences[k] = v
	}
	if repo, ok := s.data.Repos[fingerprint]; ok {
		mc.RepoPatterns = append(mc.RepoPatterns, repo.Patterns...)
		mc.ResolvedIssues = append(mc.ResolvedIssues, repo.ResolvedIssues...)
		mc.RecentEvents = append(mc.RecentEvents, repo.RecentEvents...)
		mc.ArtifactNotes = append(mc.ArtifactNotes, repo.ArtifactNotes...)
		episodes := rankedEpisodes(repo.Episodes, repo.Task, settings)
		for _, item := range episodes {
			mc.Episodes = append(mc.Episodes, prompt.MemoryEpisode{
				ID:           item.ID,
				At:           item.At,
				Kind:         item.Kind,
				Surface:      item.Surface,
				Action:       item.Action,
				Summary:      item.Summary,
				Result:       item.Result,
				WorkflowID:   item.WorkflowID,
				CapabilityID: item.CapabilityID,
				Flow:         item.Flow,
				Operation:    item.Operation,
				Confidence:   item.Confidence,
				Evidence:     append([]string(nil), item.Evidence...),
				EvidenceRefs: toPromptEvidenceRefs(item.EvidenceRefs),
				LedgerID:     item.LedgerID,
			})
		}
		for _, item := range rankedFacts(repo.SemanticFacts, settings) {
			score := scoreFact(item, time.Now())
			mc.SemanticFacts = append(mc.SemanticFacts, prompt.SemanticFact{
				Fact:          item.Fact,
				Confidence:    item.Confidence,
				Evidence:      append([]string(nil), item.Evidence...),
				EvidenceRefs:  toPromptEvidenceRefs(item.EvidenceRefs),
				LastValidated: item.LastValidated,
				Decay:         item.Decay,
				CurrentScore:  score,
				Stale:         factIsStale(item, time.Now(), settings),
			})
		}
		if repo.Task != nil {
			mc.TaskState = &prompt.TaskMemory{
				Goal:        repo.Task.Goal,
				WorkflowID:  repo.Task.WorkflowID,
				Status:      repo.Task.Status,
				Constraints: append([]string(nil), repo.Task.Constraints...),
				Pending:     append([]string(nil), repo.Task.Pending...),
				UpdatedAt:   repo.Task.UpdatedAt,
			}
		}
		if repo.CommitStyle != "" {
			mc.UserPreferences["commit_style"] = repo.CommitStyle
		}
	}
	return mc
}

func (s *Store) ensureRepo(fingerprint string) *RepoMemory {
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		fingerprint = "unknown"
	}
	repo, ok := s.data.Repos[fingerprint]
	if !ok {
		repo = &RepoMemory{Fingerprint: fingerprint}
		s.data.Repos[fingerprint] = repo

		// Evict oldest if too many repos tracked
		if len(s.data.Repos) > maxRepoEntries {
			var oldestKey string
			var oldestTime time.Time
			for k, r := range s.data.Repos {
				if oldestKey == "" || r.LastAccessed.Before(oldestTime) {
					oldestKey = k
					oldestTime = r.LastAccessed
				}
			}
			if oldestKey != "" && oldestKey != fingerprint {
				delete(s.data.Repos, oldestKey)
			}
		}
	}
	repo.LastAccessed = time.Now()
	return repo
}

// RepoFingerprint generates a simple fingerprint from repo state.
func RepoFingerprint(remoteURL, branch string) string {
	remote := strings.TrimSpace(remoteURL)
	if remote == "" {
		remote = "local"
	}
	remote = strings.TrimSuffix(remote, ".git")
	parts := strings.Split(remote, "/")
	if len(parts) > 2 {
		remote = strings.Join(parts[len(parts)-2:], "/")
	}
	return remote
}

func cloneMemoryData(in MemoryData) MemoryData {
	out := MemoryData{
		Version:     in.Version,
		Preferences: make(map[string]string, len(in.Preferences)),
		Repos:       make(map[string]*RepoMemory, len(in.Repos)),
		UpdatedAt:   in.UpdatedAt,
	}
	for key, value := range in.Preferences {
		out.Preferences[key] = value
	}
	for key, repo := range in.Repos {
		out.Repos[key] = cloneRepoMemory(repo)
	}
	return out
}

func cloneRepoMemory(in *RepoMemory) *RepoMemory {
	if in == nil {
		return nil
	}
	episodes := make([]Episode, len(in.Episodes))
	for i, episode := range in.Episodes {
		episodes[i] = Episode{
			ID:           episode.ID,
			At:           episode.At,
			Kind:         episode.Kind,
			Surface:      episode.Surface,
			Action:       episode.Action,
			Summary:      episode.Summary,
			Result:       episode.Result,
			WorkflowID:   episode.WorkflowID,
			CapabilityID: episode.CapabilityID,
			Flow:         episode.Flow,
			Operation:    episode.Operation,
			Confidence:   episode.Confidence,
			Evidence:     append([]string(nil), episode.Evidence...),
			EvidenceRefs: cloneEvidenceRefs(episode.EvidenceRefs),
			LedgerID:     episode.LedgerID,
		}
	}
	facts := make([]Fact, len(in.SemanticFacts))
	for i, fact := range in.SemanticFacts {
		facts[i] = Fact{
			Fact:          fact.Fact,
			Confidence:    fact.Confidence,
			Evidence:      append([]string(nil), fact.Evidence...),
			EvidenceRefs:  cloneEvidenceRefs(fact.EvidenceRefs),
			LastValidated: fact.LastValidated,
			Decay:         fact.Decay,
		}
	}
	var task *Task
	if in.Task != nil {
		task = &Task{
			Goal:        in.Task.Goal,
			WorkflowID:  in.Task.WorkflowID,
			Status:      in.Task.Status,
			Constraints: append([]string(nil), in.Task.Constraints...),
			Pending:     append([]string(nil), in.Task.Pending...),
			UpdatedAt:   in.Task.UpdatedAt,
		}
	}
	return &RepoMemory{
		Fingerprint:    in.Fingerprint,
		Patterns:       append([]string(nil), in.Patterns...),
		ResolvedIssues: append([]string(nil), in.ResolvedIssues...),
		RecentEvents:   append([]string(nil), in.RecentEvents...),
		ArtifactNotes:  append([]string(nil), in.ArtifactNotes...),
		Episodes:       episodes,
		SemanticFacts:  facts,
		Task:           task,
		CommitStyle:    in.CommitStyle,
		BranchNaming:   in.BranchNaming,
		LastSummarized: in.LastSummarized,
		LastAccessed:   in.LastAccessed,
	}
}

func classifySurface(event string) string {
	event = strings.ToLower(strings.TrimSpace(event))
	switch {
	case strings.HasPrefix(event, "platform "):
		return "platform"
	case strings.HasPrefix(event, "goal:"):
		return "goal"
	case strings.HasPrefix(event, "workflow:"):
		return "workflow"
	case strings.HasPrefix(event, "automation:"):
		return "automation"
	case strings.HasPrefix(event, "file "):
		return "file"
	case strings.HasPrefix(event, "command "):
		return "git"
	default:
		return "session"
	}
}

func classifyEventKind(event string) string {
	event = strings.ToLower(strings.TrimSpace(event))
	switch {
	case strings.HasPrefix(event, "platform "):
		return "platform_action"
	case strings.HasPrefix(event, "goal:"):
		return "goal"
	case strings.HasPrefix(event, "workflow:"):
		return "workflow"
	case strings.HasPrefix(event, "automation:"):
		return "automation"
	case strings.HasPrefix(event, "file "):
		return "artifact"
	case strings.HasPrefix(event, "command "):
		return "git_command"
	default:
		return "session_event"
	}
}

func classifyEventResult(event string) string {
	event = strings.ToLower(strings.TrimSpace(event))
	switch {
	case strings.Contains(event, "failed"):
		return "failed"
	case strings.Contains(event, "success"), strings.Contains(event, "succeeded"):
		return "success"
	case strings.Contains(event, "viewed"):
		return "viewed"
	default:
		return "observed"
	}
}

func classifyEventAction(event string) string {
	event = strings.ToLower(strings.TrimSpace(event))
	switch {
	case strings.HasPrefix(event, "platform success:"):
		return "platform_success"
	case strings.HasPrefix(event, "platform failed:"):
		return "platform_failure"
	case strings.HasPrefix(event, "command succeeded:"):
		return "command_success"
	case strings.HasPrefix(event, "command failed:"):
		return "command_failure"
	case strings.HasPrefix(event, "file "):
		return "file_write"
	case strings.HasPrefix(event, "workflow:"):
		return "workflow_selected"
	case strings.HasPrefix(event, "goal:"):
		return "goal_update"
	case strings.HasPrefix(event, "automation scheduled:"):
		return "automation_schedule"
	case strings.HasPrefix(event, "automation "):
		return "automation_event"
	default:
		return "session_event"
	}
}

func classifyEventConfidence(event string) float64 {
	switch classifyEventResult(event) {
	case "failed":
		return 0.82
	case "success":
		return 0.86
	case "viewed":
		return 0.58
	default:
		return 0.64
	}
}

func newEpisodeFromEvent(event string, task *Task, evidenceLimit int) Episode {
	now := time.Now()
	episode := Episode{
		At:         now,
		Kind:       classifyEventKind(event),
		Surface:    classifySurface(event),
		Action:     classifyEventAction(event),
		Summary:    strings.TrimSpace(event),
		Result:     classifyEventResult(event),
		Confidence: classifyEventConfidence(event),
		Evidence:   []string{strings.TrimSpace(event)},
	}
	episode.EvidenceRefs = eventEvidenceRefs(episode.Summary)
	if task != nil {
		episode.WorkflowID = strings.TrimSpace(task.WorkflowID)
	}
	return normalizeEpisode(episode, task, evidenceLimit)
}

func normalizeEpisode(episode Episode, task *Task, evidenceLimit int) Episode {
	episode.At = firstNonZeroTime(episode.At, time.Now())
	episode.Kind = strings.TrimSpace(firstNonEmptyString(episode.Kind, "session_event"))
	episode.Surface = strings.TrimSpace(firstNonEmptyString(episode.Surface, "session"))
	episode.Action = strings.TrimSpace(firstNonEmptyString(episode.Action, episode.Kind))
	episode.Summary = strings.TrimSpace(episode.Summary)
	episode.Result = strings.TrimSpace(firstNonEmptyString(episode.Result, "observed"))
	if episode.Confidence <= 0 {
		episode.Confidence = 0.64
	}
	if evidenceLimit <= 0 {
		evidenceLimit = defaultMaxEvidence
	}
	episode.Evidence = mergeEvidence(nil, episode.Evidence, evidenceLimit)
	episode.EvidenceRefs = mergeEvidenceRefs(nil, episode.EvidenceRefs, evidenceLimit)
	for _, ref := range episode.EvidenceRefs {
		switch strings.ToLower(strings.TrimSpace(ref.Kind)) {
		case "workflow":
			episode.WorkflowID = strings.TrimSpace(firstNonEmptyString(episode.WorkflowID, ref.Ref))
		case "capability":
			episode.CapabilityID = strings.TrimSpace(firstNonEmptyString(episode.CapabilityID, ref.Ref))
		case "flow":
			episode.Flow = strings.TrimSpace(firstNonEmptyString(episode.Flow, ref.Ref))
		case "operation":
			episode.Operation = strings.TrimSpace(firstNonEmptyString(episode.Operation, ref.Ref))
		}
	}
	if task != nil && strings.TrimSpace(episode.WorkflowID) == "" {
		episode.WorkflowID = strings.TrimSpace(task.WorkflowID)
	}
	if episode.ID == "" {
		episode.ID = episodeFingerprint(episode)
	}
	return episode
}

func eventEvidenceRefs(event string) []EvidenceRef {
	event = strings.TrimSpace(event)
	if event == "" {
		return nil
	}
	lower := strings.ToLower(event)
	switch {
	case strings.HasPrefix(lower, "platform success: "), strings.HasPrefix(lower, "platform failed: "):
		identity := strings.TrimSpace(event[strings.Index(event, ":")+1:])
		return platformIdentityRefs(strings.TrimSpace(identity))
	case strings.HasPrefix(lower, "command succeeded: "), strings.HasPrefix(lower, "command failed: "):
		command := strings.TrimSpace(event[strings.Index(event, ":")+1:])
		return []EvidenceRef{{Kind: "command", Ref: command, Label: command}}
	case strings.HasPrefix(lower, "file "):
		parts := strings.SplitN(event, ": ", 2)
		action := strings.TrimSpace(strings.TrimPrefix(parts[0], "file"))
		path := ""
		if len(parts) == 2 {
			path = strings.TrimSpace(parts[1])
		}
		refs := []EvidenceRef{}
		if action != "" {
			refs = append(refs, EvidenceRef{Kind: "operation", Ref: action, Label: action})
		}
		if path != "" {
			refs = append(refs, EvidenceRef{Kind: "file", Ref: path, Label: path})
		}
		return refs
	case strings.HasPrefix(lower, "workflow:"):
		value := strings.TrimSpace(strings.TrimPrefix(event, "workflow:"))
		return []EvidenceRef{{Kind: "workflow", Ref: value, Label: value}}
	case strings.HasPrefix(lower, "goal:"):
		value := strings.TrimSpace(strings.TrimPrefix(event, "goal:"))
		return []EvidenceRef{{Kind: "goal", Ref: value, Label: value}}
	case strings.HasPrefix(lower, "automation scheduled:"):
		value := strings.TrimSpace(strings.TrimPrefix(event, "automation scheduled:"))
		return []EvidenceRef{{Kind: "schedule", Ref: value, Label: value}}
	default:
		return nil
	}
}

func platformIdentityRefs(identity string) []EvidenceRef {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return nil
	}
	parts := strings.Split(identity, ":")
	refs := []EvidenceRef{{Kind: "platform_identity", Ref: identity, Label: identity}}
	if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
		refs = append(refs, EvidenceRef{Kind: "capability", Ref: strings.TrimSpace(parts[0]), Label: strings.TrimSpace(parts[0])})
	}
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		refs = append(refs, EvidenceRef{Kind: "flow", Ref: strings.TrimSpace(parts[1]), Label: strings.TrimSpace(parts[1])})
	}
	if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
		refs = append(refs, EvidenceRef{Kind: "operation", Ref: strings.TrimSpace(parts[2]), Label: strings.TrimSpace(parts[2])})
	}
	if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
		refs = append(refs, EvidenceRef{Kind: "resource", Ref: strings.TrimSpace(parts[3]), Label: strings.TrimSpace(parts[3])})
	}
	return refs
}

func episodeFingerprint(episode Episode) string {
	parts := []string{
		episode.At.UTC().Format(time.RFC3339Nano),
		episode.Kind,
		episode.Surface,
		episode.Action,
		episode.WorkflowID,
		episode.CapabilityID,
		episode.Flow,
		episode.Operation,
		episode.Summary,
	}
	return strings.Join(parts, "|")
}

func (s *Store) recordEpisodeLocked(repo *RepoMemory, episode Episode, settings memorySettings) {
	if repo == nil {
		return
	}
	episode = normalizeEpisode(episode, repo.Task, settings.maxEvidence)
	if episode.Summary != "" {
		repo.RecentEvents = append(repo.RecentEvents, episode.Summary)
		if len(repo.RecentEvents) > settings.maxRecentEvents {
			repo.RecentEvents = repo.RecentEvents[len(repo.RecentEvents)-settings.maxRecentEvents:]
		}
	}
	repo.Episodes = append(repo.Episodes, episode)
	if len(repo.Episodes) > settings.maxEpisodes {
		compressEpisodesLocked(repo, settings)
		if len(repo.Episodes) > settings.maxEpisodes {
			repo.Episodes = repo.Episodes[len(repo.Episodes)-settings.maxEpisodes:]
		}
	}
}

func rankedEpisodes(episodes []Episode, task *Task, settings memorySettings) []Episode {
	if len(episodes) == 0 {
		return nil
	}
	out := append([]Episode(nil), episodes...)
	now := time.Now()
	sort.SliceStable(out, func(i, j int) bool {
		left := scoreEpisode(out[i], task, now)
		right := scoreEpisode(out[j], task, now)
		if left == right {
			return out[i].At.After(out[j].At)
		}
		return left > right
	})
	if len(out) > settings.maxPromptEpisodes {
		out = out[:settings.maxPromptEpisodes]
	}
	return out
}

func scoreEpisode(episode Episode, task *Task, now time.Time) float64 {
	score := episode.Confidence
	if score <= 0 {
		score = 0.64
	}
	if !episode.At.IsZero() {
		ageDays := now.Sub(episode.At).Hours() / 24.0
		recency := 1.0 - minFloat(ageDays/14.0, 1.0)
		score += recency * 0.45
	}
	switch strings.ToLower(strings.TrimSpace(episode.Result)) {
	case "failed":
		score += 0.12
	case "success":
		score += 0.08
	case "viewed":
		score += 0.02
	}
	if len(episode.EvidenceRefs) > 0 {
		score += 0.05
	}
	if task != nil {
		if strings.EqualFold(strings.TrimSpace(task.WorkflowID), strings.TrimSpace(episode.WorkflowID)) && strings.TrimSpace(task.WorkflowID) != "" {
			score += 0.35
		}
		if summaryMentionsTask(episode.Summary, task) {
			score += 0.18
		}
	}
	return score
}

func summaryMentionsTask(summary string, task *Task) bool {
	summary = strings.ToLower(strings.TrimSpace(summary))
	if summary == "" || task == nil {
		return false
	}
	for _, token := range strings.Fields(strings.ToLower(strings.TrimSpace(task.Goal))) {
		token = strings.TrimSpace(token)
		if len(token) < 4 {
			continue
		}
		if strings.Contains(summary, token) {
			return true
		}
	}
	return false
}

func compressEpisodesLocked(repo *RepoMemory, settings memorySettings) {
	if repo == nil {
		return
	}
	now := time.Now()
	decayFactsLocked(repo, now, settings)
	if len(repo.Episodes) < settings.compressionThreshold && now.Sub(repo.LastSummarized) < 24*time.Hour {
		return
	}
	keep := settings.maxEpisodes / 2
	if keep < 8 {
		keep = 8
	}
	if len(repo.Episodes) <= keep {
		repo.LastSummarized = now
		return
	}
	cut := len(repo.Episodes) - keep
	if cut > 10 {
		cut = 10
	}
	if cut <= 0 {
		return
	}
	batch := repo.Episodes[:cut]
	bySurface := map[string][]Episode{}
	for _, episode := range batch {
		surface := strings.TrimSpace(episode.Surface)
		if surface == "" {
			surface = "session"
		}
		bySurface[surface] = append(bySurface[surface], episode)
	}
	for surface, episodes := range bySurface {
		evidence := make([]string, 0, minInt(3, len(episodes)))
		for _, episode := range episodes {
			if len(evidence) >= 3 {
				break
			}
			evidence = append(evidence, episode.Summary)
		}
		upsertFact(repo, Fact{
			Fact:          "repeated " + surface + " activity observed",
			Confidence:    0.55 + (0.05 * float64(minInt(5, len(episodes)))),
			Evidence:      evidence,
			EvidenceRefs: []EvidenceRef{{
				Kind:  "surface",
				Ref:   surface,
				Label: surface,
			}},
			LastValidated: now,
			Decay:         0.03,
		}, settings)
	}
	repo.Episodes = append([]Episode(nil), repo.Episodes[cut:]...)
	repo.LastSummarized = now
}

func upsertFact(repo *RepoMemory, fact Fact, settings memorySettings) {
	if repo == nil || strings.TrimSpace(fact.Fact) == "" {
		return
	}
	fact.Fact = strings.TrimSpace(fact.Fact)
	if fact.Confidence <= 0 {
		fact.Confidence = 0.5
	}
	if fact.Decay <= 0 {
		fact.Decay = settings.defaultSemanticDecay
	}
	for idx := range repo.SemanticFacts {
		if strings.EqualFold(repo.SemanticFacts[idx].Fact, fact.Fact) {
			repo.SemanticFacts[idx].Confidence = blendConfidence(repo.SemanticFacts[idx].Confidence, fact.Confidence)
			repo.SemanticFacts[idx].Evidence = mergeEvidence(repo.SemanticFacts[idx].Evidence, fact.Evidence, settings.maxEvidence)
			repo.SemanticFacts[idx].EvidenceRefs = mergeEvidenceRefs(repo.SemanticFacts[idx].EvidenceRefs, fact.EvidenceRefs, settings.maxEvidence)
			repo.SemanticFacts[idx].LastValidated = fact.LastValidated
			repo.SemanticFacts[idx].Decay = fact.Decay
			return
		}
	}
	repo.SemanticFacts = append(repo.SemanticFacts, fact)
	if len(repo.SemanticFacts) > settings.maxSemanticFacts {
		ranked := rankedFacts(repo.SemanticFacts, settings)
		repo.SemanticFacts = append([]Fact(nil), ranked[:settings.maxSemanticFacts]...)
	}
}

func rankedFacts(facts []Fact, settings memorySettings) []Fact {
	out := append([]Fact(nil), facts...)
	now := time.Now()
	sort.SliceStable(out, func(i, j int) bool {
		left := scoreFact(out[i], now)
		right := scoreFact(out[j], now)
		if left == right {
			return out[i].Fact < out[j].Fact
		}
		return left > right
	})
	if len(out) > settings.maxPromptFacts {
		out = out[:settings.maxPromptFacts]
	}
	return out
}

func scoreFact(f Fact, now time.Time) float64 {
	score := f.Confidence
	if !f.LastValidated.IsZero() {
		hours := now.Sub(f.LastValidated).Hours()
		score -= (hours / 24.0) * f.Decay
	}
	return score
}

func factIsStale(f Fact, now time.Time, settings memorySettings) bool {
	if f.LastValidated.IsZero() {
		return false
	}
	if now.Sub(f.LastValidated) > 30*24*time.Hour {
		return true
	}
	return scoreFact(f, now) <= maxFloat(settings.minSemanticScore*2, 0.2)
}

func decayFactsLocked(repo *RepoMemory, now time.Time, settings memorySettings) {
	if repo == nil || len(repo.SemanticFacts) == 0 {
		return
	}
	out := make([]Fact, 0, len(repo.SemanticFacts))
	for _, fact := range repo.SemanticFacts {
		if scoreFact(fact, now) <= settings.minSemanticScore {
			continue
		}
		out = append(out, fact)
	}
	repo.SemanticFacts = out
}

func mergeEvidence(existing, incoming []string, limit int) []string {
	out := append([]string(nil), existing...)
	seen := map[string]struct{}{}
	for _, item := range out {
		seen[item] = struct{}{}
	}
	for _, item := range incoming {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		out = append(out, item)
		seen[item] = struct{}{}
		if len(out) >= limit {
			break
		}
	}
	return out
}

func mergeEvidenceRefs(existing, incoming []EvidenceRef, limit int) []EvidenceRef {
	out := append([]EvidenceRef(nil), existing...)
	seen := map[string]struct{}{}
	for _, item := range out {
		key := strings.ToLower(strings.TrimSpace(item.Kind)) + "|" + strings.TrimSpace(item.Ref)
		seen[key] = struct{}{}
	}
	for _, item := range incoming {
		item.Kind = strings.TrimSpace(item.Kind)
		item.Ref = strings.TrimSpace(item.Ref)
		item.Label = strings.TrimSpace(firstNonEmptyString(item.Label, item.Ref))
		if item.Kind == "" || item.Ref == "" {
			continue
		}
		key := strings.ToLower(item.Kind) + "|" + item.Ref
		if _, ok := seen[key]; ok {
			continue
		}
		out = append(out, item)
		seen[key] = struct{}{}
		if len(out) >= limit {
			break
		}
	}
	return out
}

func cloneEvidenceRefs(in []EvidenceRef) []EvidenceRef {
	if len(in) == 0 {
		return nil
	}
	out := make([]EvidenceRef, len(in))
	copy(out, in)
	return out
}

func toPromptEvidenceRefs(in []EvidenceRef) []prompt.EvidenceRef {
	if len(in) == 0 {
		return nil
	}
	out := make([]prompt.EvidenceRef, 0, len(in))
	for _, item := range in {
		out = append(out, prompt.EvidenceRef{
			Kind:  item.Kind,
			Ref:   item.Ref,
			Label: item.Label,
		})
	}
	return out
}

func compactStrings(in []string, limit int) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func blendConfidence(existing, incoming float64) float64 {
	if existing <= 0 {
		return incoming
	}
	if incoming <= 0 {
		return existing
	}
	score := (existing * 0.7) + (incoming * 0.3)
	if score < 0.05 {
		return 0.05
	}
	if score > 0.99 {
		return 0.99
	}
	return score
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
