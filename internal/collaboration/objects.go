package collaboration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ObjectType represents GitHub collaboration object types.
type ObjectType string

const (
	ObjectTypeIssue       ObjectType = "issue"
	ObjectTypePullRequest ObjectType = "pull_request"
	ObjectTypeDiscussion  ObjectType = "discussion"
	ObjectTypeRelease     ObjectType = "release"
	ObjectTypeCheckRun    ObjectType = "check_run"
)

// CollaborationObject represents a GitHub collaboration object (issue, PR, etc.).
type CollaborationObject struct {
	ObjectID      string     `json:"object_id" yaml:"object_id"`
	ObjectType    ObjectType `json:"object_type" yaml:"object_type"`
	RepoOwner     string     `json:"repo_owner" yaml:"repo_owner"`
	RepoName      string     `json:"repo_name" yaml:"repo_name"`
	Number        int        `json:"number" yaml:"number"`
	Title         string     `json:"title" yaml:"title"`
	State         string     `json:"state" yaml:"state"`
	Author        string     `json:"author" yaml:"author"`
	Assignees     []string   `json:"assignees,omitempty" yaml:"assignees,omitempty"`
	Labels        []string   `json:"labels,omitempty" yaml:"labels,omitempty"`
	Milestone     string     `json:"milestone,omitempty" yaml:"milestone,omitempty"`
	Body          string     `json:"body,omitempty" yaml:"body,omitempty"`
	CreatedAt     time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" yaml:"updated_at"`
	CommentsCount int        `json:"comments_count" yaml:"comments_count"`
	URL           string     `json:"url" yaml:"url"`
}

// ObjectFilter filters collaboration objects for listing.
type ObjectFilter struct {
	ObjectType  ObjectType `json:"object_type,omitempty" yaml:"object_type,omitempty"`
	State       string     `json:"state,omitempty" yaml:"state,omitempty"` // open, closed, all
	Labels      []string   `json:"labels,omitempty" yaml:"labels,omitempty"`
	Assignee    string     `json:"assignee,omitempty" yaml:"assignee,omitempty"`
	Author      string     `json:"author,omitempty" yaml:"author,omitempty"`
	Milestone   string     `json:"milestone,omitempty" yaml:"milestone,omitempty"`
	SearchQuery string     `json:"search_query,omitempty" yaml:"search_query,omitempty"`
	RepoOwner   string     `json:"repo_owner,omitempty" yaml:"repo_owner,omitempty"`
	RepoName    string     `json:"repo_name,omitempty" yaml:"repo_name,omitempty"`
}

// ObjectStore persists and retrieves collaboration objects.
type ObjectStore interface {
	SaveObject(ctx context.Context, obj *CollaborationObject) error
	GetObject(ctx context.Context, objectID string) (*CollaborationObject, error)
	ListObjects(ctx context.Context, filter *ObjectFilter) ([]*CollaborationObject, error)
	GetByRepoAndNumber(ctx context.Context, owner, repo string, number int) (*CollaborationObject, error)
}

// MemoryObjectStore is a thread-safe in-memory implementation of ObjectStore.
type MemoryObjectStore struct {
	mu       sync.RWMutex
	byID     map[string]*CollaborationObject
	byRepoNo map[string]*CollaborationObject // key: "owner/repo#number"
}

// NewMemoryObjectStore creates a new MemoryObjectStore.
func NewMemoryObjectStore() *MemoryObjectStore {
	return &MemoryObjectStore{
		byID:     make(map[string]*CollaborationObject),
		byRepoNo: make(map[string]*CollaborationObject),
	}
}

// SaveObject saves or updates a collaboration object.
func (s *MemoryObjectStore) SaveObject(_ context.Context, obj *CollaborationObject) error {
	if obj == nil {
		return errors.New("object cannot be nil")
	}
	if obj.ObjectID == "" {
		obj.ObjectID = uuid.New().String()
	}
	now := time.Now().UTC()
	obj.UpdatedAt = now
	if obj.CreatedAt.IsZero() {
		obj.CreatedAt = now
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := repoNumberKey(obj.RepoOwner, obj.RepoName, obj.Number)
	if existing, ok := s.byRepoNo[key]; ok && existing.ObjectID != obj.ObjectID {
		return errors.New("object with same repo and number already exists")
	}

	dup := *obj
	dup.Assignees = append([]string(nil), obj.Assignees...)
	dup.Labels = append([]string(nil), obj.Labels...)
	s.byID[dup.ObjectID] = &dup
	s.byRepoNo[key] = &dup
	return nil
}

// GetObject retrieves an object by ID.
func (s *MemoryObjectStore) GetObject(_ context.Context, objectID string) (*CollaborationObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	obj, ok := s.byID[objectID]
	if !ok {
		return nil, errors.New("object not found")
	}
	out := *obj
	out.Assignees = append([]string(nil), obj.Assignees...)
	out.Labels = append([]string(nil), obj.Labels...)
	return &out, nil
}

// ListObjects lists objects matching the filter.
func (s *MemoryObjectStore) ListObjects(_ context.Context, filter *ObjectFilter) ([]*CollaborationObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*CollaborationObject
	for _, obj := range s.byID {
		if filter != nil && !matchFilter(obj, filter) {
			continue
		}
		dup := *obj
		dup.Assignees = append([]string(nil), obj.Assignees...)
		dup.Labels = append([]string(nil), obj.Labels...)
		out = append(out, &dup)
	}
	return out, nil
}

// GetByRepoAndNumber retrieves an object by repo owner, name, and number.
func (s *MemoryObjectStore) GetByRepoAndNumber(_ context.Context, owner, repo string, number int) (*CollaborationObject, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := repoNumberKey(owner, repo, number)
	obj, ok := s.byRepoNo[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	out := *obj
	out.Assignees = append([]string(nil), obj.Assignees...)
	out.Labels = append([]string(nil), obj.Labels...)
	return &out, nil
}

func repoNumberKey(owner, repo string, number int) string {
	return fmt.Sprintf("%s/%s#%d", owner, repo, number)
}

// ObjectRef returns a stable reference string for the object (e.g. "owner/repo#123").
func (c *CollaborationObject) ObjectRef() string {
	return repoNumberKey(c.RepoOwner, c.RepoName, c.Number)
}

func matchFilter(obj *CollaborationObject, filter *ObjectFilter) bool {
	if filter.ObjectType != "" && obj.ObjectType != filter.ObjectType {
		return false
	}
	if filter.State != "" && filter.State != "all" {
		if filter.State != obj.State {
			return false
		}
	}
	if filter.RepoOwner != "" && obj.RepoOwner != filter.RepoOwner {
		return false
	}
	if filter.RepoName != "" && obj.RepoName != filter.RepoName {
		return false
	}
	if len(filter.Labels) > 0 {
		hasAll := true
		for _, l := range filter.Labels {
			found := false
			for _, ol := range obj.Labels {
				if ol == l {
					found = true
					break
				}
			}
			if !found {
				hasAll = false
				break
			}
		}
		if !hasAll {
			return false
		}
	}
	if filter.Assignee != "" {
		found := false
		for _, a := range obj.Assignees {
			if a == filter.Assignee {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if filter.Author != "" && obj.Author != filter.Author {
		return false
	}
	if filter.Milestone != "" && obj.Milestone != filter.Milestone {
		return false
	}
	return true
}
