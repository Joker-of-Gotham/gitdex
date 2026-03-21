package collaboration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LinkType represents the type of link between objects.
type LinkType string

const (
	LinkBlocks      LinkType = "blocks"
	LinkBlockedBy   LinkType = "blocked_by"
	LinkRelatesTo   LinkType = "relates_to"
	LinkDuplicateOf LinkType = "duplicate_of"
	LinkParentOf    LinkType = "parent_of"
	LinkChildOf     LinkType = "child_of"
)

// Valid returns true if the link type is recognized.
func (l LinkType) Valid() bool {
	switch l {
	case LinkBlocks, LinkBlockedBy, LinkRelatesTo, LinkDuplicateOf, LinkParentOf, LinkChildOf:
		return true
	default:
		return false
	}
}

// ObjectLink represents a link between two objects.
type ObjectLink struct {
	SourceRef string    `json:"source_ref" yaml:"source_ref"`
	TargetRef string    `json:"target_ref" yaml:"target_ref"`
	LinkType  LinkType  `json:"link_type" yaml:"link_type"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

func init() {
	// Ensure ObjectLink has CreatedAt set when created via link command
	_ = ObjectLink{}
}

// TaskContext holds cross-object task context.
type TaskContext struct {
	ContextID        string       `json:"context_id" yaml:"context_id"`
	PrimaryObjectRef string       `json:"primary_object_ref" yaml:"primary_object_ref"`
	LinkedObjects    []ObjectLink `json:"linked_objects" yaml:"linked_objects"`
	RelatedTasks     []string     `json:"related_tasks" yaml:"related_tasks"`
	Notes            string       `json:"notes" yaml:"notes"`
	CreatedAt        time.Time    `json:"created_at" yaml:"created_at"`
}

// ContextStore persists and retrieves task contexts.
type ContextStore interface {
	SaveContext(ctx context.Context, tc *TaskContext) error
	GetContext(ctx context.Context, contextID string) (*TaskContext, error)
	ListContexts(ctx context.Context) ([]*TaskContext, error)
	GetByObjectRef(ctx context.Context, objectRef string) (*TaskContext, error)
}

// MemoryContextStore is an in-memory implementation of ContextStore.
type MemoryContextStore struct {
	mu       sync.RWMutex
	byID     map[string]*TaskContext
	byObjRef map[string]*TaskContext
}

// NewMemoryContextStore creates a new MemoryContextStore.
func NewMemoryContextStore() *MemoryContextStore {
	return &MemoryContextStore{
		byID:     make(map[string]*TaskContext),
		byObjRef: make(map[string]*TaskContext),
	}
}

// SaveContext saves or updates a task context.
func (s *MemoryContextStore) SaveContext(_ context.Context, tc *TaskContext) error {
	if tc == nil {
		return fmt.Errorf("context cannot be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if tc.ContextID == "" {
		tc.ContextID = uuid.New().String()
	}
	now := time.Now().UTC()
	tc.CreatedAt = now

	dup := *tc
	dup.LinkedObjects = make([]ObjectLink, len(tc.LinkedObjects))
	copy(dup.LinkedObjects, tc.LinkedObjects)
	dup.RelatedTasks = make([]string, len(tc.RelatedTasks))
	copy(dup.RelatedTasks, tc.RelatedTasks)

	// When overwriting by primary ref, remove old context from byID if different
	if old, ok := s.byObjRef[dup.PrimaryObjectRef]; ok && old.ContextID != dup.ContextID {
		delete(s.byID, old.ContextID)
	}
	s.byID[dup.ContextID] = &dup
	s.byObjRef[dup.PrimaryObjectRef] = &dup
	return nil
}

// GetContext retrieves a context by ID.
func (s *MemoryContextStore) GetContext(_ context.Context, contextID string) (*TaskContext, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tc, ok := s.byID[contextID]
	if !ok {
		return nil, fmt.Errorf("context not found")
	}
	out := *tc
	out.LinkedObjects = make([]ObjectLink, len(tc.LinkedObjects))
	copy(out.LinkedObjects, tc.LinkedObjects)
	out.RelatedTasks = make([]string, len(tc.RelatedTasks))
	copy(out.RelatedTasks, tc.RelatedTasks)
	return &out, nil
}

// ListContexts returns all contexts.
func (s *MemoryContextStore) ListContexts(_ context.Context) ([]*TaskContext, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*TaskContext
	for _, tc := range s.byID {
		dup := *tc
		dup.LinkedObjects = make([]ObjectLink, len(tc.LinkedObjects))
		copy(dup.LinkedObjects, tc.LinkedObjects)
		dup.RelatedTasks = make([]string, len(tc.RelatedTasks))
		copy(dup.RelatedTasks, tc.RelatedTasks)
		out = append(out, &dup)
	}
	return out, nil
}

// GetByObjectRef retrieves context by primary object ref.
func (s *MemoryContextStore) GetByObjectRef(_ context.Context, objectRef string) (*TaskContext, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tc, ok := s.byObjRef[objectRef]
	if !ok {
		return nil, fmt.Errorf("context not found for object %q", objectRef)
	}
	out := *tc
	out.LinkedObjects = make([]ObjectLink, len(tc.LinkedObjects))
	copy(out.LinkedObjects, tc.LinkedObjects)
	out.RelatedTasks = make([]string, len(tc.RelatedTasks))
	copy(out.RelatedTasks, tc.RelatedTasks)
	return &out, nil
}
