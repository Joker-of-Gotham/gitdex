package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/collaboration"
)

type ObjectStore struct {
	db *sql.DB
}

func NewObjectStore(db *sql.DB) *ObjectStore {
	return &ObjectStore{db: db}
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

	commentsCount := 0
	if obj.CommentsCount > 0 {
		commentsCount = obj.CommentsCount
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO collaboration_objects (object_id, object_type, repo_owner, repo_name, number, title, state, author, assignees, labels, milestone, body, created_at, updated_at, comments_count, url)
		VALUES (?, ?, ?, ?, ?, NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), ?, ?, NULLIF(?,''), NULLIF(?,''), ?, ?, ?, NULLIF(?,''))
		ON CONFLICT (object_id) DO UPDATE SET
			object_type = excluded.object_type,
			repo_owner = excluded.repo_owner,
			repo_name = excluded.repo_name,
			number = excluded.number,
			title = excluded.title,
			state = excluded.state,
			author = excluded.author,
			assignees = excluded.assignees,
			labels = excluded.labels,
			milestone = excluded.milestone,
			body = excluded.body,
			updated_at = excluded.updated_at,
			comments_count = excluded.comments_count,
			url = excluded.url
	`, obj.ObjectID, obj.ObjectType, obj.RepoOwner, obj.RepoName, obj.Number, obj.Title, obj.State, obj.Author, assigneesJSON, labelsJSON, obj.Milestone, obj.Body, formatTime(obj.CreatedAt), formatTime(obj.UpdatedAt), commentsCount, obj.URL)
	return err
}

func (s *ObjectStore) GetObject(ctx context.Context, objectID string) (*collaboration.CollaborationObject, error) {
	var obj collaboration.CollaborationObject
	var assigneesJSON, labelsJSON []byte
	var createdAtStr, updatedAtStr string

	err := s.db.QueryRowContext(ctx, `
		SELECT object_id, object_type, repo_owner, repo_name, number, title, state, author, assignees, labels, milestone, body, created_at, updated_at, comments_count, url
		FROM collaboration_objects WHERE object_id = ?
	`, objectID).Scan(&obj.ObjectID, &obj.ObjectType, &obj.RepoOwner, &obj.RepoName, &obj.Number, &obj.Title, &obj.State, &obj.Author, &assigneesJSON, &labelsJSON, &obj.Milestone, &obj.Body, &createdAtStr, &updatedAtStr, &obj.CommentsCount, &obj.URL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("object not found")
		}
		return nil, err
	}

	_ = json.Unmarshal(assigneesJSON, &obj.Assignees)
	_ = json.Unmarshal(labelsJSON, &obj.Labels)
	obj.CreatedAt, _ = parseTime(createdAtStr)
	obj.UpdatedAt, _ = parseTime(updatedAtStr)
	return &obj, nil
}

func (s *ObjectStore) ListObjects(ctx context.Context, filter *collaboration.ObjectFilter) ([]*collaboration.CollaborationObject, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT object_id, object_type, repo_owner, repo_name, number, title, state, author, assignees, labels, milestone, body, created_at, updated_at, comments_count, url FROM collaboration_objects`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*collaboration.CollaborationObject
	for rows.Next() {
		var obj collaboration.CollaborationObject
		var assigneesJSON, labelsJSON []byte
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&obj.ObjectID, &obj.ObjectType, &obj.RepoOwner, &obj.RepoName, &obj.Number, &obj.Title, &obj.State, &obj.Author, &assigneesJSON, &labelsJSON, &obj.Milestone, &obj.Body, &createdAtStr, &updatedAtStr, &obj.CommentsCount, &obj.URL); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(assigneesJSON, &obj.Assignees)
		_ = json.Unmarshal(labelsJSON, &obj.Labels)
		obj.CreatedAt, _ = parseTime(createdAtStr)
		obj.UpdatedAt, _ = parseTime(updatedAtStr)
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
	var createdAtStr, updatedAtStr string

	err := s.db.QueryRowContext(ctx, `
		SELECT object_id, object_type, repo_owner, repo_name, number, title, state, author, assignees, labels, milestone, body, created_at, updated_at, comments_count, url
		FROM collaboration_objects WHERE repo_owner = ? AND repo_name = ? AND number = ?
	`, owner, repo, number).Scan(&obj.ObjectID, &obj.ObjectType, &obj.RepoOwner, &obj.RepoName, &obj.Number, &obj.Title, &obj.State, &obj.Author, &assigneesJSON, &labelsJSON, &obj.Milestone, &obj.Body, &createdAtStr, &updatedAtStr, &obj.CommentsCount, &obj.URL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("object not found")
		}
		return nil, err
	}

	_ = json.Unmarshal(assigneesJSON, &obj.Assignees)
	_ = json.Unmarshal(labelsJSON, &obj.Labels)
	obj.CreatedAt, _ = parseTime(createdAtStr)
	obj.UpdatedAt, _ = parseTime(updatedAtStr)
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
