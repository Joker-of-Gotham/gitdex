package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/planning"
)

type PlanStore struct {
	pool *pgxpool.Pool
}

func NewPlanStore(pool *pgxpool.Pool) *PlanStore {
	return &PlanStore{pool: pool}
}

func (s *PlanStore) Save(plan *planning.Plan) error {
	if plan == nil {
		return fmt.Errorf("cannot save nil plan")
	}
	if plan.PlanID == "" {
		plan.PlanID = planning.GeneratePlanID()
	}
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = time.Now().UTC()
	}
	plan.UpdatedAt = time.Now().UTC()

	intentJSON, _ := json.Marshal(plan.Intent)
	scopeJSON, _ := json.Marshal(plan.Scope)
	stepsJSON, _ := json.Marshal(plan.Steps)
	var policyResultJSON []byte
	if plan.PolicyResult != nil {
		policyResultJSON, _ = json.Marshal(plan.PolicyResult)
	}
	evidenceRefsJSON, _ := json.Marshal(orNilSlice(plan.EvidenceRefs))

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO plans (plan_id, task_id, status, intent, scope, steps, risk_level, policy_result, execution_mode, deferred_until, evidence_refs, created_at, updated_at)
		VALUES ($1, NULLIF($2,''), $3, $4, $5, $6, NULLIF($7,''), $8, NULLIF($9,''), $10, $11, $12, $13)
		ON CONFLICT (plan_id) DO UPDATE SET
			task_id = EXCLUDED.task_id,
			status = EXCLUDED.status,
			intent = EXCLUDED.intent,
			scope = EXCLUDED.scope,
			steps = EXCLUDED.steps,
			risk_level = EXCLUDED.risk_level,
			policy_result = EXCLUDED.policy_result,
			execution_mode = EXCLUDED.execution_mode,
			deferred_until = EXCLUDED.deferred_until,
			evidence_refs = EXCLUDED.evidence_refs,
			updated_at = EXCLUDED.updated_at
	`, plan.PlanID, plan.TaskID, plan.Status, intentJSON, scopeJSON, stepsJSON, plan.RiskLevel, policyResultJSON, plan.ExecutionMode, plan.DeferredUntil, evidenceRefsJSON, plan.CreatedAt, plan.UpdatedAt)
	return err
}

func (s *PlanStore) Get(planID string) (*planning.Plan, error) {
	var p planning.Plan
	var taskID *string
	var intentJSON, scopeJSON, stepsJSON []byte
	var policyResultJSON []byte
	var evidenceRefsJSON []byte
	var deferredUntil *time.Time

	err := s.pool.QueryRow(ctx(), `
		SELECT plan_id, task_id, status, intent, scope, steps, risk_level, policy_result, execution_mode, deferred_until, evidence_refs, created_at, updated_at
		FROM plans WHERE plan_id = $1
	`, planID).Scan(&p.PlanID, &taskID, &p.Status, &intentJSON, &scopeJSON, &stepsJSON, &p.RiskLevel, &policyResultJSON, &p.ExecutionMode, &deferredUntil, &evidenceRefsJSON, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("plan %q not found", planID)
		}
		return nil, err
	}

	if taskID != nil {
		p.TaskID = *taskID
	}
	_ = json.Unmarshal(intentJSON, &p.Intent)
	_ = json.Unmarshal(scopeJSON, &p.Scope)
	_ = json.Unmarshal(stepsJSON, &p.Steps)
	if len(policyResultJSON) > 0 {
		p.PolicyResult = &planning.PolicyResult{}
		_ = json.Unmarshal(policyResultJSON, p.PolicyResult)
	}
	p.DeferredUntil = deferredUntil
	_ = json.Unmarshal(evidenceRefsJSON, &p.EvidenceRefs)
	return &p, nil
}

func (s *PlanStore) GetByTaskID(taskID string) (*planning.Plan, error) {
	var planID string
	err := s.pool.QueryRow(ctx(), `SELECT plan_id FROM plans WHERE task_id = $1`, taskID).Scan(&planID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no plan found for task %q", taskID)
		}
		return nil, err
	}
	return s.Get(planID)
}

func (s *PlanStore) List() ([]*planning.Plan, error) {
	rows, err := s.pool.Query(ctx(), `SELECT plan_id FROM plans ORDER BY created_at`)
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

	result := make([]*planning.Plan, 0, len(ids))
	for _, id := range ids {
		p, err := s.Get(id)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

func (s *PlanStore) UpdateStatus(planID string, status planning.PlanStatus) error {
	cmd, err := s.pool.Exec(ctx(), `UPDATE plans SET status = $1, updated_at = $2 WHERE plan_id = $3`, status, time.Now().UTC(), planID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("plan %q not found", planID)
	}
	return nil
}

func (s *PlanStore) SaveApproval(record *planning.ApprovalRecord) error {
	if record == nil {
		return fmt.Errorf("cannot save nil approval record")
	}
	if record.RecordID == "" {
		record.RecordID = planning.GenerateApprovalID()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO approval_records (record_id, plan_id, action, actor, reason, previous_status, new_status, created_at)
		VALUES ($1, $2, $3, $4, NULLIF($5,''), $6, $7, $8)
	`, record.RecordID, record.PlanID, record.Action, record.Actor, record.Reason, record.PreviousStatus, record.NewStatus, record.CreatedAt)
	return err
}

func (s *PlanStore) GetApprovals(planID string) ([]*planning.ApprovalRecord, error) {
	rows, err := s.pool.Query(ctx(), `
		SELECT record_id, plan_id, action, actor, reason, previous_status, new_status, created_at
		FROM approval_records WHERE plan_id = $1 ORDER BY created_at
	`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*planning.ApprovalRecord
	for rows.Next() {
		var r planning.ApprovalRecord
		var reason *string
		if err := rows.Scan(&r.RecordID, &r.PlanID, &r.Action, &r.Actor, &reason, &r.PreviousStatus, &r.NewStatus, &r.CreatedAt); err != nil {
			return nil, err
		}
		if reason != nil {
			r.Reason = *reason
		}
		result = append(result, &r)
	}
	return result, rows.Err()
}
