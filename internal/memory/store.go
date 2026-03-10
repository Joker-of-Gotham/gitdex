package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
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
const memoryFileMode = 0o600

// Store manages persistent long-term memory across sessions.
type Store struct {
	mu   sync.RWMutex
	data MemoryData
	path string
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
	CommitStyle    string    `json:"commit_style,omitempty"`
	BranchNaming   string    `json:"branch_naming,omitempty"`
	LastAccessed   time.Time `json:"last_accessed"`
}

// NewStore creates a memory store, loading from disk if available.
func NewStore() *Store {
	s := &Store{
		data: MemoryData{
			Version:     1,
			Preferences: map[string]string{},
			Repos:       map[string]*RepoMemory{},
		},
	}
	s.path = defaultPath()
	s.load()
	return s
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
}

// RecordResolvedIssue records that an issue was resolved in this repo.
func (s *Store) RecordResolvedIssue(fingerprint, issue string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	repo := s.ensureRepo(fingerprint)
	repo.ResolvedIssues = append(repo.ResolvedIssues, issue)
	if len(repo.ResolvedIssues) > maxResolvedHistory {
		repo.ResolvedIssues = repo.ResolvedIssues[len(repo.ResolvedIssues)-maxResolvedHistory:]
	}
}

// SetRepoCommitStyle records the detected commit style for this repo.
func (s *Store) SetRepoCommitStyle(fingerprint, style string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	repo := s.ensureRepo(fingerprint)
	repo.CommitStyle = style
}

// ToPromptMemory converts stored memory into prompt-injectable format.
func (s *Store) ToPromptMemory(fingerprint string) *prompt.MemoryContext {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mc := &prompt.MemoryContext{
		UserPreferences: map[string]string{},
	}
	for k, v := range s.data.Preferences {
		mc.UserPreferences[k] = v
	}
	if repo, ok := s.data.Repos[fingerprint]; ok {
		mc.RepoPatterns = append(mc.RepoPatterns, repo.Patterns...)
		mc.ResolvedIssues = append(mc.ResolvedIssues, repo.ResolvedIssues...)
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
	return &RepoMemory{
		Fingerprint:    in.Fingerprint,
		Patterns:       append([]string(nil), in.Patterns...),
		ResolvedIssues: append([]string(nil), in.ResolvedIssues...),
		CommitStyle:    in.CommitStyle,
		BranchNaming:   in.BranchNaming,
		LastAccessed:   in.LastAccessed,
	}
}
