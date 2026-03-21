package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/collaboration"
)

type ContextStore struct {
	pool *pgxpool.Pool
}

func NewContextStore(pool *pgxpool.Pool) *ContextStore {
	return &ContextStore{pool: pool}
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

	_, err := s.pool.Exec(ctx, `
		INSERT INTO task_contexts (context_id, primary_object_ref, linked_objects, related_tasks, notes, created_at)
		VALUES ($1, $2, $3, $4, NULLIF($5,''), $6)
		ON CONFLICT (primary_object_ref) DO UPDATE SET
			context_id = EXCLUDED.context_id,
			linked_objects = EXCLUDED.linked_objects,
			related_tasks = EXCLUDED.related_tasks,
			notes = EXCLUDED.notes
	`, tc.ContextID, tc.PrimaryObjectRef, linkedJSON, relatedJSON, tc.Notes, tc.CreatedAt)
	return err
}

func (s *ContextStore) GetContext(ctx context.Context, contextID string) (*collaboration.TaskContext, error) {
	var tc collaboration.TaskContext
	var linkedJSON, relatedJSON []byte
	var notes *string

	err := s.pool.QueryRow(ctx, `
		SELECT context_id, primary_object_ref, linked_objects, related_tasks, notes, created_at
		FROM task_contexts WHERE context_id = $1
	`, contextID).Scan(&tc.ContextID, &tc.PrimaryObjectRef, &linkedJSON, &relatedJSON, &notes, &tc.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("context not found")
		}
		return nil, err
	}
	if notes != nil {
		tc.Notes = *notes
	}
	_ = json.Unmarshal(linkedJSON, &tc.LinkedObjects)
	_ = json.Unmarshal(relatedJSON, &tc.RelatedTasks)
	return &tc, nil
}

func (s *ContextStore) ListContexts(ctx context.Context) ([]*collaboration.TaskContext, error) {
	rows, err := s.pool.Query(ctx, `SELECT context_id FROM task_contexts ORDER BY created_at`)
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
	err := s.pool.QueryRow(ctx, `SELECT context_id FROM task_contexts WHERE primary_object_ref = $1`, objectRef).Scan(&contextID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("context not found for object %q", objectRef)
		}
		return nil, err
	}
	return s.GetContext(ctx, contextID)
}
