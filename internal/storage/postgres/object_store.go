package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/collaboration"
)

type ObjectStore struct {
	pool *pgxpool.Pool
}

func NewObjectStore(pool *pgxpool.Pool) *ObjectStore {
	return &ObjectStore{pool: pool}
}

func (s *ObjectStore) SaveObject(ctx context.Context, obj *collaboration.CollaborationObject) error {
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

	assigneesJSON, _ := json.Marshal(orNilSlice(obj.Assignees))
	labelsJSON, _ := json.Marshal(orNilSlice(obj.Labels))

	_, err := s.pool.Exec(ctx, `
		INSERT INTO collaboration_objects (object_id, object_type, repo_owner, repo_name, number, title, state, author, assignees, labels, milestone, body, created_at, updated_at, comments_count, url)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''), NULLIF($8,''), $9, $10, NULLIF($11,''), NULLIF($12,''), $13, $14, COALESCE($15, 0), NULLIF($16,''))
		ON CONFLICT (object_id) DO UPDATE SET
			object_type = EXCLUDED.object_type,
			repo_owner = EXCLUDED.repo_owner,
			repo_name = EXCLUDED.repo_name,
			number = EXCLUDED.number,
			title = EXCLUDED.title,
			state = EXCLUDED.state,
			author = EXCLUDED.author,
			assignees = EXCLUDED.assignees,
			labels = EXCLUDED.labels,
			milestone = EXCLUDED.milestone,
			body = EXCLUDED.body,
			updated_at = EXCLUDED.updated_at,
			comments_count = EXCLUDED.comments_count,
			url = EXCLUDED.url
	`, obj.ObjectID, obj.ObjectType, obj.RepoOwner, obj.RepoName, obj.Number, obj.Title, obj.State, obj.Author, assigneesJSON, labelsJSON, obj.Milestone, obj.Body, obj.CreatedAt, obj.UpdatedAt, obj.CommentsCount, obj.URL)
	return err
}

func (s *ObjectStore) GetObject(ctx context.Context, objectID string) (*collaboration.CollaborationObject, error) {
	var obj collaboration.CollaborationObject
	var assigneesJSON, labelsJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT object_id, object_type, repo_owner, repo_name, number, title, state, author, assignees, labels, milestone, body, created_at, updated_at, comments_count, url
		FROM collaboration_objects WHERE object_id = $1
	`, objectID).Scan(&obj.ObjectID, &obj.ObjectType, &obj.RepoOwner, &obj.RepoName, &obj.Number, &obj.Title, &obj.State, &obj.Author, &assigneesJSON, &labelsJSON, &obj.Milestone, &obj.Body, &obj.CreatedAt, &obj.UpdatedAt, &obj.CommentsCount, &obj.URL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("object not found")
		}
		return nil, err
	}

	_ = json.Unmarshal(assigneesJSON, &obj.Assignees)
	_ = json.Unmarshal(labelsJSON, &obj.Labels)
	return &obj, nil
}

func (s *ObjectStore) ListObjects(ctx context.Context, filter *collaboration.ObjectFilter) ([]*collaboration.CollaborationObject, error) {
	rows, err := s.pool.Query(ctx, `SELECT object_id, object_type, repo_owner, repo_name, number, title, state, author, assignees, labels, milestone, body, created_at, updated_at, comments_count, url FROM collaboration_objects`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*collaboration.CollaborationObject
	for rows.Next() {
		var obj collaboration.CollaborationObject
		var assigneesJSON, labelsJSON []byte
		if err := rows.Scan(&obj.ObjectID, &obj.ObjectType, &obj.RepoOwner, &obj.RepoName, &obj.Number, &obj.Title, &obj.State, &obj.Author, &assigneesJSON, &labelsJSON, &obj.Milestone, &obj.Body, &obj.CreatedAt, &obj.UpdatedAt, &obj.CommentsCount, &obj.URL); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(assigneesJSON, &obj.Assignees)
		_ = json.Unmarshal(labelsJSON, &obj.Labels)
		if filter != nil && !matchFilter(&obj, filter) {
			continue
		}
		result = append(result, &obj)
	}
	return result, rows.Err()
}

func (s *ObjectStore) GetByRepoAndNumber(ctx context.Context, owner, repo string, number int) (*collaboration.CollaborationObject, error) {
	var obj collaboration.CollaborationObject
	var assigneesJSON, labelsJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT object_id, object_type, repo_owner, repo_name, number, title, state, author, assignees, labels, milestone, body, created_at, updated_at, comments_count, url
		FROM collaboration_objects WHERE repo_owner = $1 AND repo_name = $2 AND number = $3
	`, owner, repo, number).Scan(&obj.ObjectID, &obj.ObjectType, &obj.RepoOwner, &obj.RepoName, &obj.Number, &obj.Title, &obj.State, &obj.Author, &assigneesJSON, &labelsJSON, &obj.Milestone, &obj.Body, &obj.CreatedAt, &obj.UpdatedAt, &obj.CommentsCount, &obj.URL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("object not found")
		}
		return nil, err
	}

	_ = json.Unmarshal(assigneesJSON, &obj.Assignees)
	_ = json.Unmarshal(labelsJSON, &obj.Labels)
	return &obj, nil
}

func matchFilter(obj *collaboration.CollaborationObject, filter *collaboration.ObjectFilter) bool {
	if filter.ObjectType != "" && obj.ObjectType != filter.ObjectType {
		return false
	}
	if filter.State != "" && filter.State != "all" && obj.State != filter.State {
		return false
	}
	if filter.RepoOwner != "" && obj.RepoOwner != filter.RepoOwner {
		return false
	}
	if filter.RepoName != "" && obj.RepoName != filter.RepoName {
		return false
	}
	if len(filter.Labels) > 0 {
		for _, l := range filter.Labels {
			found := false
			for _, ol := range obj.Labels {
				if ol == l {
					found = true
					break
				}
			}
			if !found {
				return false
			}
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
