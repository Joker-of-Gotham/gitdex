package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/planning"
)

type PlanStore struct {
	db *sql.DB
}

func NewPlanStore(db *sql.DB) *PlanStore {
	return &PlanStore{db: db}
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

	deferredStr := ""
	if plan.DeferredUntil != nil {
		deferredStr = plan.DeferredUntil.UTC().Format(time.RFC3339)
	}

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO plans (plan_id, task_id, status, intent, scope, steps, risk_level, policy_result, execution_mode, deferred_until, evidence_refs, created_at, updated_at)
		VALUES (?, NULLIF(?,''), ?, ?, ?, ?, NULLIF(?,''), ?, NULLIF(?,''), NULLIF(?,''), ?, ?, ?)
		ON CONFLICT (plan_id) DO UPDATE SET
			task_id = excluded.task_id,
			status = excluded.status,
			intent = excluded.intent,
			scope = excluded.scope,
			steps = excluded.steps,
			risk_level = excluded.risk_level,
			policy_result = excluded.policy_result,
			execution_mode = excluded.execution_mode,
			deferred_until = excluded.deferred_until,
			evidence_refs = excluded.evidence_refs,
			updated_at = excluded.updated_at
	`, plan.PlanID, plan.TaskID, plan.Status, intentJSON, scopeJSON, stepsJSON, plan.RiskLevel, policyResultJSON, plan.ExecutionMode, deferredStr, evidenceRefsJSON, plan.CreatedAt.UTC().Format(time.RFC3339), plan.UpdatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *PlanStore) Get(planID string) (*planning.Plan, error) {
	var p planning.Plan
	var taskID sql.NullString
	var riskLevel sql.NullString
	var executionMode sql.NullString
	var intentJSON, scopeJSON, stepsJSON []byte
	var policyResultJSON []byte
	var evidenceRefsJSON []byte
	var deferredStr sql.NullString
	var createdAtStr, updatedAtStr string

	err := s.db.QueryRowContext(ctx(), `
		SELECT plan_id, task_id, status, intent, scope, steps, risk_level, policy_result, execution_mode, deferred_until, evidence_refs, created_at, updated_at
		FROM plans WHERE plan_id = ?
	`, planID).Scan(&p.PlanID, &taskID, &p.Status, &intentJSON, &scopeJSON, &stepsJSON, &riskLevel, &policyResultJSON, &executionMode, &deferredStr, &evidenceRefsJSON, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("plan %q not found", planID)
		}
		return nil, err
	}

	if taskID.Valid {
		p.TaskID = taskID.String
	}
	if riskLevel.Valid {
		p.RiskLevel = planning.RiskLevel(riskLevel.String)
	}
	if executionMode.Valid {
		p.ExecutionMode = planning.ExecutionMode(executionMode.String)
	}
	_ = json.Unmarshal(intentJSON, &p.Intent)
	_ = json.Unmarshal(scopeJSON, &p.Scope)
	_ = json.Unmarshal(stepsJSON, &p.Steps)
	if len(policyResultJSON) > 0 {
		p.PolicyResult = &planning.PolicyResult{}
		_ = json.Unmarshal(policyResultJSON, p.PolicyResult)
	}
	if deferredStr.Valid && deferredStr.String != "" {
		t, _ := time.Parse(time.RFC3339, deferredStr.String)
		p.DeferredUntil = &t
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	_ = json.Unmarshal(evidenceRefsJSON, &p.EvidenceRefs)
	return &p, nil
}

func (s *PlanStore) GetByTaskID(taskID string) (*planning.Plan, error) {
	var planID string
	err := s.db.QueryRowContext(ctx(), `SELECT plan_id FROM plans WHERE task_id = ?`, taskID).Scan(&planID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no plan found for task %q", taskID)
		}
		return nil, err
	}
	return s.Get(planID)
}

func (s *PlanStore) List() ([]*planning.Plan, error) {
	rows, err := s.db.QueryContext(ctx(), `SELECT plan_id FROM plans ORDER BY created_at`)
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
	res, err := s.db.ExecContext(ctx(), `UPDATE plans SET status = ?, updated_at = ? WHERE plan_id = ?`, status, time.Now().UTC().Format(time.RFC3339), planID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
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

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO approval_records (record_id, plan_id, action, actor, reason, previous_status, new_status, created_at)
		VALUES (?, ?, ?, ?, NULLIF(?,''), ?, ?, ?)
	`, record.RecordID, record.PlanID, record.Action, record.Actor, record.Reason, record.PreviousStatus, record.NewStatus, record.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *PlanStore) GetApprovals(planID string) ([]*planning.ApprovalRecord, error) {
	rows, err := s.db.QueryContext(ctx(), `
		SELECT record_id, plan_id, action, actor, reason, previous_status, new_status, created_at
		FROM approval_records WHERE plan_id = ? ORDER BY created_at
	`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*planning.ApprovalRecord
	for rows.Next() {
		var r planning.ApprovalRecord
		var reason sql.NullString
		var createdAtStr string
		if err := rows.Scan(&r.RecordID, &r.PlanID, &r.Action, &r.Actor, &reason, &r.PreviousStatus, &r.NewStatus, &createdAtStr); err != nil {
			return nil, err
		}
		if reason.Valid {
			r.Reason = reason.String
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		result = append(result, &r)
	}
	return result, rows.Err()
}
