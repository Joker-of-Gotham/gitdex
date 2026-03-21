package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/autonomy"
)

type HandoffStore struct {
	pool *pgxpool.Pool
}

func NewHandoffStore(pool *pgxpool.Pool) *HandoffStore {
	return &HandoffStore{pool: pool}
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

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO handoff_packages (package_id, task_id, task_summary, current_state, completed_steps, pending_steps, context_data, artifacts, recommendations, created_at)
		VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), $5, $6, $7, $8, $9, $10)
		ON CONFLICT (task_id) DO UPDATE SET
			package_id = EXCLUDED.package_id,
			task_summary = EXCLUDED.task_summary,
			current_state = EXCLUDED.current_state,
			completed_steps = EXCLUDED.completed_steps,
			pending_steps = EXCLUDED.pending_steps,
			context_data = EXCLUDED.context_data,
			artifacts = EXCLUDED.artifacts,
			recommendations = EXCLUDED.recommendations
	`, pkg.PackageID, pkg.TaskID, pkg.TaskSummary, pkg.CurrentState, completedJSON, pendingJSON, contextJSON, artifactsJSON, recommendationsJSON, pkg.CreatedAt)
	return err
}

func (s *HandoffStore) GetPackage(packageID string) (*autonomy.HandoffPackage, error) {
	var pkg autonomy.HandoffPackage
	var completedJSON, pendingJSON, contextJSON, artifactsJSON, recommendationsJSON []byte

	err := s.pool.QueryRow(ctx(), `
		SELECT package_id, task_id, task_summary, current_state, completed_steps, pending_steps, context_data, artifacts, recommendations, created_at
		FROM handoff_packages WHERE package_id = $1
	`, packageID).Scan(&pkg.PackageID, &pkg.TaskID, &pkg.TaskSummary, &pkg.CurrentState, &completedJSON, &pendingJSON, &contextJSON, &artifactsJSON, &recommendationsJSON, &pkg.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("package %q not found", packageID)
		}
		return nil, err
	}

	_ = json.Unmarshal(completedJSON, &pkg.CompletedSteps)
	_ = json.Unmarshal(pendingJSON, &pkg.PendingSteps)
	_ = json.Unmarshal(contextJSON, &pkg.ContextData)
	_ = json.Unmarshal(artifactsJSON, &pkg.Artifacts)
	_ = json.Unmarshal(recommendationsJSON, &pkg.Recommendations)
	return &pkg, nil
}

func (s *HandoffStore) ListPackages() ([]*autonomy.HandoffPackage, error) {
	rows, err := s.pool.Query(ctx(), `SELECT package_id FROM handoff_packages ORDER BY created_at`)
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
	err := s.pool.QueryRow(ctx(), `SELECT package_id FROM handoff_packages WHERE task_id = $1`, taskID).Scan(&packageID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no package for task %q", taskID)
		}
		return nil, err
	}
	return s.GetPackage(packageID)
}
