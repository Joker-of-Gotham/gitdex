package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/autonomy"
)

type HandoffStore struct {
	db *sql.DB
}

func NewHandoffStore(db *sql.DB) *HandoffStore {
	return &HandoffStore{db: db}
}

func (s *HandoffStore) SavePackage(pkg *autonomy.HandoffPackage) error {
	if pkg == nil {
		return fmt.Errorf("cannot save nil package")
	}
	if pkg.TaskID == "" {
		return fmt.Errorf("TaskID cannot be empty")
	}
	if pkg.PackageID == "" {
		pkg.PackageID = "pkg_" + generateShortID()
	}
	if pkg.CreatedAt.IsZero() {
		pkg.CreatedAt = time.Now().UTC()
	}

	completedJSON, _ := json.Marshal(orNilSlice(pkg.CompletedSteps))
	pendingJSON, _ := json.Marshal(orNilSlice(pkg.PendingSteps))
	contextJSON, _ := json.Marshal(orNilMap(pkg.ContextData))
	artifactsJSON, _ := json.Marshal(orNilSlice(pkg.Artifacts))
	recommendationsJSON, _ := json.Marshal(orNilSlice(pkg.Recommendations))

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO handoff_packages (package_id, task_id, task_summary, current_state, completed_steps, pending_steps, context_data, artifacts, recommendations, created_at)
		VALUES (?, ?, NULLIF(?,''), NULLIF(?,''), ?, ?, ?, ?, ?, ?)
		ON CONFLICT (task_id) DO UPDATE SET
			package_id = excluded.package_id,
			task_summary = excluded.task_summary,
			current_state = excluded.current_state,
			completed_steps = excluded.completed_steps,
			pending_steps = excluded.pending_steps,
			context_data = excluded.context_data,
			artifacts = excluded.artifacts,
			recommendations = excluded.recommendations
	`, pkg.PackageID, pkg.TaskID, pkg.TaskSummary, pkg.CurrentState, completedJSON, pendingJSON, contextJSON, artifactsJSON, recommendationsJSON, formatTime(pkg.CreatedAt))
	return err
}

func (s *HandoffStore) GetPackage(packageID string) (*autonomy.HandoffPackage, error) {
	var pkg autonomy.HandoffPackage
	var completedJSON, pendingJSON, contextJSON, artifactsJSON, recommendationsJSON []byte
	var createdAtStr string

	err := s.db.QueryRowContext(ctx(), `
		SELECT package_id, task_id, task_summary, current_state, completed_steps, pending_steps, context_data, artifacts, recommendations, created_at
		FROM handoff_packages WHERE package_id = ?
	`, packageID).Scan(&pkg.PackageID, &pkg.TaskID, &pkg.TaskSummary, &pkg.CurrentState, &completedJSON, &pendingJSON, &contextJSON, &artifactsJSON, &recommendationsJSON, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("package %q not found", packageID)
		}
		return nil, err
	}

	_ = json.Unmarshal(completedJSON, &pkg.CompletedSteps)
	_ = json.Unmarshal(pendingJSON, &pkg.PendingSteps)
	_ = json.Unmarshal(contextJSON, &pkg.ContextData)
	_ = json.Unmarshal(artifactsJSON, &pkg.Artifacts)
	_ = json.Unmarshal(recommendationsJSON, &pkg.Recommendations)
	pkg.CreatedAt, _ = parseTime(createdAtStr)
	return &pkg, nil
}

func (s *HandoffStore) ListPackages() ([]*autonomy.HandoffPackage, error) {
	rows, err := s.db.QueryContext(ctx(), `SELECT package_id FROM handoff_packages ORDER BY created_at`)
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

	result := make([]*autonomy.HandoffPackage, 0, len(ids))
	for _, id := range ids {
		pkg, err := s.GetPackage(id)
		if err != nil {
			return nil, err
		}
		result = append(result, pkg)
	}
	return result, nil
}

func (s *HandoffStore) GetByTaskID(taskID string) (*autonomy.HandoffPackage, error) {
	var packageID string
	err := s.db.QueryRowContext(ctx(), `SELECT package_id FROM handoff_packages WHERE task_id = ?`, taskID).Scan(&packageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no package for task %q", taskID)
		}
		return nil, err
	}
	return s.GetPackage(packageID)
}
