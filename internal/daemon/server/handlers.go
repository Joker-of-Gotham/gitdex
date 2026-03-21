package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/campaign"
	"github.com/your-org/gitdex/internal/planning"
	ghwebhook "github.com/your-org/gitdex/internal/platform/github"
)

func isNotFoundErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not found")
}

func errParamRequired(name string) error {
	return fmt.Errorf("path parameter %q is required", name)
}

func errNotFound(resource, id string) error {
	return fmt.Errorf("%s not found: %s", resource, id)
}

func writeJSON(w http.ResponseWriter, data any) {
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, err error, status int) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func (s *Server) handleListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := s.provider.PlanStore().List()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, plans)
}

func (s *Server) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	planID := chi.URLParam(r, "planID")
	if planID == "" {
		writeError(w, errParamRequired("planID"), http.StatusBadRequest)
		return
	}
	plan, err := s.provider.PlanStore().Get(planID)
	if err != nil {
		if isNotFoundErr(err) {
			writeError(w, errNotFound("plan", planID), http.StatusNotFound)
		} else {
			writeError(w, err, http.StatusInternalServerError)
		}
		return
	}
	if plan == nil {
		writeError(w, errNotFound("plan", planID), http.StatusNotFound)
		return
	}
	writeJSON(w, plan)
}

func (s *Server) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	var plan planning.Plan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		writeError(w, fmt.Errorf("invalid JSON: %w", err), http.StatusBadRequest)
		return
	}
	if plan.PlanID == "" {
		plan.PlanID = "plan_" + uuid.New().String()[:8]
	}
	if plan.Status == "" {
		plan.Status = planning.PlanDraft
	}
	if err := s.provider.PlanStore().Save(&plan); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", "/api/v1/plans/"+plan.PlanID)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, plan)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.provider.TaskStore().ListTasks()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, tasks)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		writeError(w, errParamRequired("taskID"), http.StatusBadRequest)
		return
	}
	task, err := s.provider.TaskStore().GetTask(taskID)
	if err != nil {
		if isNotFoundErr(err) {
			writeError(w, errNotFound("task", taskID), http.StatusNotFound)
		} else {
			writeError(w, err, http.StatusInternalServerError)
		}
		return
	}
	if task == nil {
		writeError(w, errNotFound("task", taskID), http.StatusNotFound)
		return
	}
	writeJSON(w, task)
}

func (s *Server) handleQueryAudit(w http.ResponseWriter, r *http.Request) {
	// Parse optional query params: event_type, task_id, correlation_id, from_time, to_time
	filters := audit.AuditFilter{}
	if v := r.URL.Query().Get("event_type"); v != "" {
		filters.EventType = audit.EventType(v)
	}
	if v := r.URL.Query().Get("task_id"); v != "" {
		filters.TaskID = v
	}
	if v := r.URL.Query().Get("correlation_id"); v != "" {
		filters.CorrelationID = v
	}
	entries, err := s.provider.AuditLedger().Query(filters)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, entries)
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	campaigns, err := s.provider.CampaignStore().ListCampaigns()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, campaigns)
}

func (s *Server) handleGetCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "campaignID")
	if campaignID == "" {
		writeError(w, errParamRequired("campaignID"), http.StatusBadRequest)
		return
	}
	campaign, err := s.provider.CampaignStore().GetCampaign(campaignID)
	if err != nil {
		if isNotFoundErr(err) {
			writeError(w, errNotFound("campaign", campaignID), http.StatusNotFound)
		} else {
			writeError(w, err, http.StatusInternalServerError)
		}
		return
	}
	if campaign == nil {
		writeError(w, errNotFound("campaign", campaignID), http.StatusNotFound)
		return
	}
	writeJSON(w, campaign)
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var c campaign.Campaign
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, fmt.Errorf("invalid JSON: %w", err), http.StatusBadRequest)
		return
	}
	if c.Status == "" {
		c.Status = campaign.StatusDraft
	}
	if err := s.provider.CampaignStore().SaveCampaign(&c); err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Location", "/api/v1/campaigns/"+c.CampaignID)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, c)
}

func (s *Server) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.provider.MonitorStore().ListMonitorConfigs()
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, monitors)
}

func (s *Server) handleGetMonitor(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "monitorID")
	if monitorID == "" {
		writeError(w, errParamRequired("monitorID"), http.StatusBadRequest)
		return
	}
	monitor, err := s.provider.MonitorStore().GetMonitorConfig(monitorID)
	if err != nil {
		if isNotFoundErr(err) {
			writeError(w, errNotFound("monitor", monitorID), http.StatusNotFound)
		} else {
			writeError(w, err, http.StatusInternalServerError)
		}
		return
	}
	if monitor == nil {
		writeError(w, errNotFound("monitor", monitorID), http.StatusNotFound)
		return
	}
	writeJSON(w, monitor)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, fmt.Errorf("read webhook body: %w", err), http.StatusBadRequest)
		return
	}

	secret := strings.TrimSpace(os.Getenv("GITDEX_GITHUB_WEBHOOK_SECRET"))
	handler := ghwebhook.NewWebhookHandler(secret)
	validated := false
	if secret != "" {
		if err := handler.ValidateSignature(r, body); err != nil {
			writeError(w, fmt.Errorf("invalid webhook signature: %w", err), http.StatusUnauthorized)
			return
		}
		validated = true
	}

	eventType, payload, err := handler.ParseEvent(r, body)
	if err != nil {
		writeError(w, fmt.Errorf("parse webhook: %w", err), http.StatusBadRequest)
		return
	}

	repoFullName, action := parseWebhookEnvelope(payload)
	matched, err := s.appendTriggerEventsForWebhook(r.Context(), eventType, repoFullName, action)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	writeJSON(w, map[string]any{
		"received":         true,
		"validated":        validated,
		"event_type":       eventType,
		"delivery_id":      r.Header.Get("X-GitHub-Delivery"),
		"repository":       repoFullName,
		"action":           action,
		"matched_triggers": matched,
	})
}

func parseWebhookEnvelope(payload []byte) (repoFullName, action string) {
	var envelope struct {
		Action     string `json:"action"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return "", ""
	}
	return strings.TrimSpace(envelope.Repository.FullName), strings.TrimSpace(envelope.Action)
}

func (s *Server) appendTriggerEventsForWebhook(ctx context.Context, eventType, repoFullName, action string) (int, error) {
	configs, err := s.provider.TriggerStore().ListTriggers()
	if err != nil {
		return 0, fmt.Errorf("list triggers: %w", err)
	}

	matched := 0
	for _, cfg := range configs {
		if !matchesWebhookTrigger(cfg, eventType, repoFullName) {
			continue
		}
		ev := &autonomy.TriggerEvent{
			TriggerID:       cfg.TriggerID,
			TriggerType:     cfg.TriggerType,
			SourceEvent:     firstNonEmptyWebhook(action, eventType),
			ResultingTaskID: "",
		}
		if s.onGitHubWebhook != nil {
			if err := s.onGitHubWebhook(ctx, cfg, repoFullName, ev); err != nil {
				return matched, fmt.Errorf("handle matched trigger %s: %w", cfg.TriggerID, err)
			}
		}
		if err := s.provider.TriggerStore().AppendTriggerEvent(ev); err != nil {
			return matched, fmt.Errorf("append trigger event: %w", err)
		}
		matched++
	}
	return matched, nil
}

func matchesWebhookTrigger(cfg *autonomy.TriggerConfig, eventType, repoFullName string) bool {
	if cfg == nil || !cfg.Enabled || cfg.TriggerType != autonomy.TriggerTypeEvent {
		return false
	}
	source := strings.ToLower(strings.TrimSpace(cfg.Source))
	eventType = strings.ToLower(strings.TrimSpace(eventType))
	repoFullName = strings.ToLower(strings.TrimSpace(repoFullName))
	if source == "" {
		return true
	}
	switch {
	case source == eventType:
		return true
	case source == repoFullName:
		return true
	case source == repoFullName+":"+eventType:
		return true
	case source == repoFullName+"#"+eventType:
		return true
	case strings.HasPrefix(source, "event:") && strings.TrimPrefix(source, "event:") == eventType:
		return true
	case strings.HasPrefix(source, "repo:") && strings.TrimPrefix(source, "repo:") == repoFullName:
		return true
	default:
		return false
	}
}

func firstNonEmptyWebhook(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *Server) handleCruiseStatus(w http.ResponseWriter, r *http.Request) {
	if s.cruise == nil {
		writeError(w, fmt.Errorf("cruise engine not configured"), http.StatusServiceUnavailable)
		return
	}
	out := map[string]any{
		"state":       s.cruise.State(),
		"cycle_count": s.cruise.CycleCount(),
		"metrics":     s.cruise.MetricsSummary(),
		"dead_letter": s.cruise.DeadLetterSummaries(),
	}
	if rep := s.cruise.LastReport(); rep != nil {
		out["last_report"] = rep
	}
	writeJSON(w, out)
}

func (s *Server) handleCruisePause(w http.ResponseWriter, r *http.Request) {
	if s.cruise == nil {
		writeError(w, fmt.Errorf("cruise engine not configured"), http.StatusServiceUnavailable)
		return
	}
	s.cruise.Pause()
	writeJSON(w, map[string]any{"ok": true, "state": s.cruise.State()})
}

func (s *Server) handleCruiseResume(w http.ResponseWriter, r *http.Request) {
	if s.cruise == nil {
		writeError(w, fmt.Errorf("cruise engine not configured"), http.StatusServiceUnavailable)
		return
	}
	s.cruise.Resume()
	writeJSON(w, map[string]any{"ok": true, "state": s.cruise.State()})
}

func (s *Server) handleListApprovals(w http.ResponseWriter, r *http.Request) {
	if s.cruise == nil {
		writeError(w, fmt.Errorf("cruise engine not configured"), http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, s.cruise.PendingApprovals())
}

func (s *Server) handleApprovalApprove(w http.ResponseWriter, r *http.Request) {
	if s.cruise == nil {
		writeError(w, fmt.Errorf("cruise engine not configured"), http.StatusServiceUnavailable)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, errParamRequired("id"), http.StatusBadRequest)
		return
	}
	if err := s.cruise.Approve(r.Context(), id); err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "plan_id": id, "action": "approved"})
}

func (s *Server) handleApprovalReject(w http.ResponseWriter, r *http.Request) {
	if s.cruise == nil {
		writeError(w, fmt.Errorf("cruise engine not configured"), http.StatusServiceUnavailable)
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, errParamRequired("id"), http.StatusBadRequest)
		return
	}
	if err := s.cruise.Reject(r.Context(), id); err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "plan_id": id, "action": "rejected"})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if s.cruise == nil {
		writeError(w, fmt.Errorf("cruise engine not configured"), http.StatusServiceUnavailable)
		return
	}
	rep := s.cruise.Reporter()
	var history []autonomy.CruiseReport
	if rep != nil {
		history = rep.List()
	}
	writeJSON(w, map[string]any{
		"health":         "ok",
		"cruise_state":   s.cruise.State(),
		"run_history":    history,
		"failure_triage": s.cruise.DeadLetterSummaries(),
		"metrics":        s.cruise.MetricsSummary(),
	})
}
