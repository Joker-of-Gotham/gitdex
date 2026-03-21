package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/audit"
)

type AuditStore struct {
	pool *pgxpool.Pool
}

func NewAuditStore(pool *pgxpool.Pool) *AuditStore {
	return &AuditStore{pool: pool}
}

func (s *AuditStore) Append(entry *audit.AuditEntry) error {
	if entry == nil {
		return fmt.Errorf("cannot append nil entry")
	}
	if entry.EntryID == "" {
		entry.EntryID = audit.GenerateEntryID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	evidenceRefsJSON, _ := json.Marshal(orNilSlice(entry.EvidenceRefs))

	_, err := s.pool.Exec(ctx(), `
		INSERT INTO audit_entries (entry_id, correlation_id, task_id, plan_id, event_type, actor, action, target, policy_result, evidence_refs, timestamp)
		VALUES ($1, NULLIF($2,''), NULLIF($3,''), NULLIF($4,''), $5, NULLIF($6,''), NULLIF($7,''), NULLIF($8,''), NULLIF($9,''), $10, $11)
	`, entry.EntryID, entry.CorrelationID, entry.TaskID, entry.PlanID, entry.EventType, entry.Actor, entry.Action, entry.Target, entry.PolicyResult, evidenceRefsJSON, entry.Timestamp)
	return err
}

func (s *AuditStore) Query(filters audit.AuditFilter) ([]*audit.AuditEntry, error) {
	query := `SELECT entry_id, correlation_id, task_id, plan_id, event_type, actor, action, target, policy_result, evidence_refs, timestamp FROM audit_entries WHERE 1=1`
	args := []interface{}{}
	n := 1

	if filters.EntryID != "" {
		query += fmt.Sprintf(" AND entry_id = $%d", n)
		args = append(args, filters.EntryID)
		n++
	}
	if filters.EventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", n)
		args = append(args, filters.EventType)
		n++
	}
	if filters.TaskID != "" {
		query += fmt.Sprintf(" AND task_id = $%d", n)
		args = append(args, filters.TaskID)
		n++
	}
	if filters.CorrelationID != "" {
		query += fmt.Sprintf(" AND correlation_id = $%d", n)
		args = append(args, filters.CorrelationID)
		n++
	}
	if filters.FromTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", n)
		args = append(args, *filters.FromTime)
		n++
	}
	if filters.ToTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", n)
		args = append(args, *filters.ToTime)
		n++
	}
	query += " ORDER BY timestamp"

	rows, err := s.pool.Query(ctx(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*audit.AuditEntry
	for rows.Next() {
		var e audit.AuditEntry
		var corrID, taskID, planID, actor, action, target, policyResult *string
		var evidenceRefsJSON []byte
		if err := rows.Scan(&e.EntryID, &corrID, &taskID, &planID, &e.EventType, &actor, &action, &target, &policyResult, &evidenceRefsJSON, &e.Timestamp); err != nil {
			return nil, err
		}
		if corrID != nil {
			e.CorrelationID = *corrID
		}
		if taskID != nil {
			e.TaskID = *taskID
		}
		if planID != nil {
			e.PlanID = *planID
		}
		if actor != nil {
			e.Actor = *actor
		}
		if action != nil {
			e.Action = *action
		}
		if target != nil {
			e.Target = *target
		}
		if policyResult != nil {
			e.PolicyResult = *policyResult
		}
		_ = json.Unmarshal(evidenceRefsJSON, &e.EvidenceRefs)
		result = append(result, &e)
	}
	return result, rows.Err()
}

func (s *AuditStore) GetByCorrelation(correlationID string) ([]*audit.AuditEntry, error) {
	return s.Query(audit.AuditFilter{CorrelationID: correlationID})
}

func (s *AuditStore) GetByTask(taskID string) ([]*audit.AuditEntry, error) {
	return s.Query(audit.AuditFilter{TaskID: taskID})
}

func (s *AuditStore) GetByEntryID(entryID string) (*audit.AuditEntry, bool) {
	var e audit.AuditEntry
	var corrID, taskID, planID, actor, action, target, policyResult *string
	var evidenceRefsJSON []byte

	err := s.pool.QueryRow(ctx(), `
		SELECT entry_id, correlation_id, task_id, plan_id, event_type, actor, action, target, policy_result, evidence_refs, timestamp
		FROM audit_entries WHERE entry_id = $1
	`, entryID).Scan(&e.EntryID, &corrID, &taskID, &planID, &e.EventType, &actor, &action, &target, &policyResult, &evidenceRefsJSON, &e.Timestamp)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, false
		}
		return nil, false
	}

	if corrID != nil {
		e.CorrelationID = *corrID
	}
	if taskID != nil {
		e.TaskID = *taskID
	}
	if planID != nil {
		e.PlanID = *planID
	}
	if actor != nil {
		e.Actor = *actor
	}
	if action != nil {
		e.Action = *action
	}
	if target != nil {
		e.Target = *target
	}
	if policyResult != nil {
		e.PolicyResult = *policyResult
	}
	_ = json.Unmarshal(evidenceRefsJSON, &e.EvidenceRefs)
	return &e, true
}
