package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/audit"
)

type AuditStore struct {
	db *sql.DB
}

func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
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

	_, err := s.db.ExecContext(ctx(), `
		INSERT INTO audit_entries (entry_id, correlation_id, task_id, plan_id, event_type, actor, action, target, policy_result, evidence_refs, timestamp)
		VALUES (?, NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), ?, NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), NULLIF(?,''), ?, ?)
	`, entry.EntryID, entry.CorrelationID, entry.TaskID, entry.PlanID, entry.EventType, entry.Actor, entry.Action, entry.Target, entry.PolicyResult, evidenceRefsJSON, entry.Timestamp.UTC().Format(time.RFC3339))
	return err
}

func (s *AuditStore) Query(filters audit.AuditFilter) ([]*audit.AuditEntry, error) {
	query := `SELECT entry_id, correlation_id, task_id, plan_id, event_type, actor, action, target, policy_result, evidence_refs, timestamp FROM audit_entries WHERE 1=1`
	args := []interface{}{}

	if filters.EntryID != "" {
		query += ` AND entry_id = ?`
		args = append(args, filters.EntryID)
	}
	if filters.EventType != "" {
		query += ` AND event_type = ?`
		args = append(args, filters.EventType)
	}
	if filters.TaskID != "" {
		query += ` AND task_id = ?`
		args = append(args, filters.TaskID)
	}
	if filters.CorrelationID != "" {
		query += ` AND correlation_id = ?`
		args = append(args, filters.CorrelationID)
	}
	if filters.FromTime != nil {
		query += ` AND timestamp >= ?`
		args = append(args, filters.FromTime.UTC().Format(time.RFC3339))
	}
	if filters.ToTime != nil {
		query += ` AND timestamp <= ?`
		args = append(args, filters.ToTime.UTC().Format(time.RFC3339))
	}
	query += ` ORDER BY timestamp`

	rows, err := s.db.QueryContext(ctx(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*audit.AuditEntry
	for rows.Next() {
		var e audit.AuditEntry
		var corrID, taskID, planID, actor, action, target, policyResult sql.NullString
		var evidenceRefsJSON []byte
		var timestampStr string
		if err := rows.Scan(&e.EntryID, &corrID, &taskID, &planID, &e.EventType, &actor, &action, &target, &policyResult, &evidenceRefsJSON, &timestampStr); err != nil {
			return nil, err
		}
		if corrID.Valid {
			e.CorrelationID = corrID.String
		}
		if taskID.Valid {
			e.TaskID = taskID.String
		}
		if planID.Valid {
			e.PlanID = planID.String
		}
		if actor.Valid {
			e.Actor = actor.String
		}
		if action.Valid {
			e.Action = action.String
		}
		if target.Valid {
			e.Target = target.String
		}
		if policyResult.Valid {
			e.PolicyResult = policyResult.String
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
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
	var corrID, taskID, planID, actor, action, target, policyResult sql.NullString
	var evidenceRefsJSON []byte
	var timestampStr string

	err := s.db.QueryRowContext(ctx(), `
		SELECT entry_id, correlation_id, task_id, plan_id, event_type, actor, action, target, policy_result, evidence_refs, timestamp
		FROM audit_entries WHERE entry_id = ?
	`, entryID).Scan(&e.EntryID, &corrID, &taskID, &planID, &e.EventType, &actor, &action, &target, &policyResult, &evidenceRefsJSON, &timestampStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false
		}
		return nil, false
	}

	if corrID.Valid {
		e.CorrelationID = corrID.String
	}
	if taskID.Valid {
		e.TaskID = taskID.String
	}
	if planID.Valid {
		e.PlanID = planID.String
	}
	if actor.Valid {
		e.Actor = actor.String
	}
	if action.Valid {
		e.Action = action.String
	}
	if target.Valid {
		e.Target = target.String
	}
	if policyResult.Valid {
		e.PolicyResult = policyResult.String
	}
	e.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
	_ = json.Unmarshal(evidenceRefsJSON, &e.EvidenceRefs)
	return &e, true
}
