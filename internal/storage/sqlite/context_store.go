package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/collaboration"
)

type ContextStore struct {
	db *sql.DB
}

func NewContextStore(db *sql.DB) *ContextStore {
	return &ContextStore{db: db}
}

func (s *ContextStore) SaveContext(ctx context.Context, tc *collaboration.TaskContext) error {
	if tc == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if tc.ContextID == "" {
		tc.ContextID = uuid.New().String()
	}
	now := time.Now().UTC()
	tc.CreatedAt = now

	linkedJSON, _ := json.Marshal(orNilSlice(tc.LinkedObjects))
	relatedJSON, _ := json.Marshal(orNilSlice(tc.RelatedTasks))

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO task_contexts (context_id, primary_object_ref, linked_objects, related_tasks, notes, created_at)
		VALUES (?, ?, ?, ?, NULLIF(?,''), ?)
		ON CONFLICT (primary_object_ref) DO UPDATE SET
			context_id = excluded.context_id,
			linked_objects = excluded.linked_objects,
			related_tasks = excluded.related_tasks,
			notes = excluded.notes
	`, tc.ContextID, tc.PrimaryObjectRef, linkedJSON, relatedJSON, tc.Notes, formatTime(tc.CreatedAt))
	return err
}

func (s *ContextStore) GetContext(ctx context.Context, contextID string) (*collaboration.TaskContext, error) {
	var tc collaboration.TaskContext
	var linkedJSON, relatedJSON []byte
	var notes sql.NullString
	var createdAtStr string

	err := s.db.QueryRowContext(ctx, `
		SELECT context_id, primary_object_ref, linked_objects, related_tasks, notes, created_at
		FROM task_contexts WHERE context_id = ?
	`, contextID).Scan(&tc.ContextID, &tc.PrimaryObjectRef, &linkedJSON, &relatedJSON, &notes, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("context not found")
		}
		return nil, err
	}
	if notes.Valid {
		tc.Notes = notes.String
	}
	_ = json.Unmarshal(linkedJSON, &tc.LinkedObjects)
	_ = json.Unmarshal(relatedJSON, &tc.RelatedTasks)
	tc.CreatedAt, _ = parseTime(createdAtStr)
	return &tc, nil
}

func (s *ContextStore) ListContexts(ctx context.Context) ([]*collaboration.TaskContext, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT context_id FROM task_contexts ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]*collaboration.TaskContext, 0, len(ids))
	for _, id := range ids {
		tc, err := s.GetContext(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, tc)
	}
	return result, nil
}

func (s *ContextStore) GetByObjectRef(ctx context.Context, objectRef string) (*collaboration.TaskContext, error) {
	var contextID string
	err := s.db.QueryRowContext(ctx, `SELECT context_id FROM task_contexts WHERE primary_object_ref = ?`, objectRef).Scan(&contextID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("context not found for object %q", objectRef)
		}
		return nil, err
	}
	return s.GetContext(ctx, contextID)
}
