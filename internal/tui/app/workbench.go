package app

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/panes"
	"github.com/your-org/gitdex/internal/tui/views"
)

func focusLabel(area FocusArea) string {
	switch area {
	case FocusNav:
		return "Nav"
	case FocusContent:
		return "Main"
	case FocusComposer:
		return "Composer"
	case FocusInspector:
		return "Inspector"
	case FocusPalette:
		return "Palette"
	default:
		return "Main"
	}
}

func viewLabel(id views.ID) string {
	switch id {
	case views.ViewDashboard:
		return "Cockpit"
	case views.ViewChat:
		return "Chat"
	case views.ViewExplorer:
		return "Explorer"
	case views.ViewWorkspace:
		return "Workspace"
	case views.ViewSettings:
		return "Settings"
	default:
		return strings.Title(string(id))
	}
}

func navPathForView(id views.ID) string {
	switch id {
	case views.ViewDashboard:
		return "dashboard"
	case views.ViewChat:
		return "chat"
	case views.ViewExplorer:
		return "explorer"
	case views.ViewWorkspace:
		return "workspace"
	case views.ViewSettings:
		return "settings"
	case views.ViewReflog:
		return "reflog"
	default:
		return ""
	}
}

func labelToProgress(label repo.StateLabel) float64 {
	switch label {
	case repo.Healthy:
		return 1
	case repo.Unknown:
		return 0.45
	case repo.Drifting:
		return 0.6
	case repo.Degraded:
		return 0.3
	case repo.Blocked:
		return 0.1
	default:
		return 0.4
	}
}

func (m *Model) syncChrome() {
	activeView := viewLabel(m.router.ActiveID())
	switch m.router.ActiveID() {
	case views.ViewExplorer:
		if name := strings.TrimSpace(m.explorerView.ActiveTabName()); name != "" {
			activeView = activeView + " / " + name
		}
	case views.ViewWorkspace:
		if name := strings.TrimSpace(m.workspaceView.ActiveTabName()); name != "" {
			activeView = activeView + " / " + name
		}
	}
	if m.statusBar != nil {
		m.statusBar.SetViewName(activeView)
		m.statusBar.SetFocusName(focusLabel(m.focus))
		m.statusBar.SetThemeName(m.paletteName)
		if m.activeRepo != nil {
			repoName := m.activeRepo.FullName
			if repoName == "" {
				repoName = m.activeRepo.Name
			}
			m.statusBar.SetRepoName(repoName)
			if m.activeRepo.DefaultBranch != "" {
				m.statusBar.SetBranch(m.activeRepo.DefaultBranch)
			}
		}
	}
	if m.navPane != nil {
		m.navPane.SelectPath(navPathForView(m.router.ActiveID()))
	}
	if m.inspectorPane != nil {
		ctx := panes.InspectorContext{
			ActiveView: activeView,
			Focus:      focusLabel(m.focus),
			ThemeName:  m.paletteName,
		}
		if m.activeRepo != nil {
			ctx.Repo = m.activeRepo.FullName
			if ctx.Repo == "" {
				ctx.Repo = m.activeRepo.Name
			}
			ctx.Branch = m.activeRepo.DefaultBranch
			if m.activeRepo.IsLocal {
				ctx.Location = m.activeRepo.LocalPath()
			}
		}
		m.inspectorPane.SetContext(ctx)
		if m.settingsView != nil {
			settings := m.settingsView.InspectorData()
			m.inspectorPane.SetSettings(panes.InspectorSettingsData{
				CurrentSection:     settings.CurrentSection,
				Profile:            settings.Profile,
				IdentityMode:       settings.IdentityMode,
				EffectiveAuth:      settings.EffectiveAuth,
				EffectiveHost:      settings.EffectiveHost,
				SaveTarget:         settings.SaveTarget,
				RecommendedAction:  settings.RecommendedAction,
				DirtyCount:         settings.DirtyCount,
				RepositoryDetected: settings.RepositoryDetected,
				GlobalConfig:       settings.GlobalConfig,
				RepoConfig:         settings.RepoConfig,
				ActiveFiles:        settings.ActiveFiles,
				OverrideFields:     settings.OverrideFields,
				Warnings:           settings.Warnings,
			})
		}
	}
}

func (m *Model) syncDerivedWorkspace(summary *repo.RepoSummary) {
	if summary == nil || m.workspaceView == nil {
		return
	}
	if m.currentBootstrapApp().StorageProvider != nil {
		return
	}

	totalSignals := 5
	completedSignals := 0
	for _, score := range []float64{
		labelToProgress(summary.Local.Label),
		labelToProgress(summary.Remote.Label),
		labelToProgress(summary.Collaboration.Label),
		labelToProgress(summary.Workflows.Label),
		labelToProgress(summary.Deployments.Label),
	} {
		if score >= 0.95 {
			completedSignals++
		}
	}

	planStatus := "in_progress"
	if summary.OverallLabel == repo.Healthy {
		planStatus = "completed"
	} else if summary.OverallLabel == repo.Blocked || summary.OverallLabel == repo.Degraded {
		planStatus = "blocked"
	}

	tasks := make([]views.TaskItem, 0, len(summary.Risks)+len(summary.NextActions)+3)
	for i, risk := range summary.Risks {
		status := "blocked"
		if risk.Severity == repo.RiskMedium {
			status = "in_progress"
		}
		title := risk.Action
		if title == "" {
			title = risk.Description
		}
		tasks = append(tasks, views.TaskItem{
			ID:           fmt.Sprintf("R%02d", i+1),
			Title:        title,
			Status:       status,
			AssignedPlan: "Health",
			Priority:     len(summary.Risks) - i,
		})
	}
	for i, action := range summary.NextActions {
		tasks = append(tasks, views.TaskItem{
			ID:           fmt.Sprintf("A%02d", i+1),
			Title:        action.Action,
			Status:       "queued",
			AssignedPlan: "Follow-up",
			Priority:     len(summary.NextActions) - i,
		})
	}
	if len(tasks) == 0 {
		tasks = append(tasks, views.TaskItem{
			ID:           "H00",
			Title:        "Repository signals are healthy",
			Status:       "completed",
			AssignedPlan: "Health",
			Priority:     1,
		})
	}
	m.workspaceView.Tasks().SetTasks(tasks)

	plans := []views.PlanSummary{
		{
			Title:          "Stabilize repository health",
			Status:         planStatus,
			Scope:          summary.Owner + "/" + summary.Repo,
			StepCount:      totalSignals,
			CompletedSteps: completedSignals,
			RiskLevel:      string(summary.OverallLabel),
		},
	}
	if summary.Collaboration.OpenPRCount > 0 || summary.Collaboration.OpenIssueCount > 0 {
		plans = append(plans, views.PlanSummary{
			Title:          "Reduce collaboration backlog",
			Status:         "in_progress",
			Scope:          fmt.Sprintf("%d PRs / %d issues", summary.Collaboration.OpenPRCount, summary.Collaboration.OpenIssueCount),
			StepCount:      max(summary.Collaboration.OpenPRCount+summary.Collaboration.OpenIssueCount, 1),
			CompletedSteps: 0,
			RiskLevel:      string(summary.Collaboration.Label),
		})
	}
	m.workspaceView.Plans().SetPlans(plans)

	now := time.Now()
	entries := []views.EvidenceEntry{
		{Timestamp: now, Action: "Local", Result: string(summary.Local.Label), Detail: summary.Local.Detail, Success: summary.Local.Label == repo.Healthy},
		{Timestamp: now.Add(-time.Minute), Action: "Remote", Result: string(summary.Remote.Label), Detail: summary.Remote.Detail, Success: summary.Remote.Label == repo.Healthy},
		{Timestamp: now.Add(-2 * time.Minute), Action: "Collaboration", Result: string(summary.Collaboration.Label), Detail: summary.Collaboration.Detail, Success: summary.Collaboration.Label == repo.Healthy},
		{Timestamp: now.Add(-3 * time.Minute), Action: "Workflows", Result: string(summary.Workflows.Label), Detail: summary.Workflows.Detail, Success: summary.Workflows.Label == repo.Healthy},
		{Timestamp: now.Add(-4 * time.Minute), Action: "Deployments", Result: string(summary.Deployments.Label), Detail: summary.Deployments.Detail, Success: summary.Deployments.Label == repo.Healthy},
	}
	m.workspaceView.Evidence().SetEntries(entries)

	m.setWorkspaceInspectorEvidence(entries)
}

func (m *Model) setWorkspaceInspectorEvidence(entries []views.EvidenceEntry) {
	if m.inspectorPane == nil {
		return
	}
	evidence := make([]panes.InspectorEvidence, 0, len(entries))
	for _, entry := range entries {
		evidence = append(evidence, panes.InspectorEvidence{
			Timestamp: entry.Timestamp.Format("15:04:05"),
			Title:     entry.Action,
			Result:    entry.Result,
			Detail:    entry.Detail,
			Success:   entry.Success,
		})
	}
	m.inspectorPane.SetEvidence(evidence)
}

func (m *Model) loadWorkspaceFromStores() tea.Cmd {
	app := m.currentBootstrapApp()
	if app.StorageProvider == nil {
		return nil
	}
	sp := app.StorageProvider
	return func() tea.Msg {
		plans, err1 := sp.PlanStore().List()
		tasks, err2 := sp.TaskStore().ListTasks()
		entries, err3 := sp.AuditLedger().Query(audit.AuditFilter{})
		err := err1
		if err == nil && err2 != nil {
			err = err2
		}
		if err == nil && err3 != nil {
			err = err3
		}
		if err != nil {
			return views.WorkspaceStoresMsg{Err: err}
		}
		vplans := mapPlansToSummaries(plans)
		sort.Slice(vplans, func(i, j int) bool { return vplans[i].Title < vplans[j].Title })
		vtasks := mapTasksToItems(tasks)
		sort.Slice(vtasks, func(i, j int) bool { return vtasks[i].ID < vtasks[j].ID })
		for i := range vplans {
			pid := strings.TrimSpace(vplans[i].PlanID)
			if pid == "" {
				continue
			}
			for _, t := range vtasks {
				if strings.TrimSpace(t.AssignedPlan) == pid {
					vplans[i].StepLines = append(vplans[i].StepLines, fmt.Sprintf("  Task %s [%s]: %s", t.ID, t.Status, t.Title))
				}
			}
		}
		evidence := mapAuditToEvidence(entries)
		sort.Slice(evidence, func(i, j int) bool { return evidence[i].Timestamp.After(evidence[j].Timestamp) })
		if len(evidence) > 80 {
			evidence = evidence[:80]
		}
		return views.WorkspaceStoresMsg{Plans: vplans, Tasks: vtasks, Evidence: evidence}
	}
}

func formatPlanSteps(steps []planning.PlanStep) []string {
	out := make([]string, 0, len(steps))
	for i, s := range steps {
		seq := s.Sequence
		if seq == 0 {
			seq = i + 1
		}
		act := strings.TrimSpace(s.Action)
		if act == "" {
			act = "(no action)"
		}
		line := fmt.Sprintf("%d. %s", seq, act)
		if d := strings.TrimSpace(s.Description); d != "" {
			line += " — " + d
		}
		if t := strings.TrimSpace(s.Target); t != "" {
			line += fmt.Sprintf(" (%s)", t)
		}
		out = append(out, line)
	}
	return out
}

func mapPlansToSummaries(plans []*planning.Plan) []views.PlanSummary {
	out := make([]views.PlanSummary, 0, len(plans))
	for _, p := range plans {
		if p == nil {
			continue
		}
		title := strings.TrimSpace(p.Intent.RawInput)
		if title == "" {
			title = p.PlanID
		}
		completed := 0
		if p.Status == planning.PlanCompleted {
			completed = len(p.Steps)
		}
		scope := strings.TrimSpace(p.Scope.Owner + "/" + p.Scope.Repo)
		if scope == "/" {
			scope = ""
		}
		out = append(out, views.PlanSummary{
			Title:          title,
			Status:         string(p.Status),
			Scope:          scope,
			StepCount:      len(p.Steps),
			CompletedSteps: completed,
			RiskLevel:      string(p.RiskLevel),
			PlanID:         p.PlanID,
			StepLines:      formatPlanSteps(p.Steps),
		})
	}
	return out
}

func mapTasksToItems(tasks []*orchestrator.Task) []views.TaskItem {
	out := make([]views.TaskItem, 0, len(tasks))
	for _, t := range tasks {
		if t == nil {
			continue
		}
		title := t.TaskID
		for _, s := range t.Steps {
			if a := strings.TrimSpace(s.Action); a != "" {
				title = a
				break
			}
		}
		out = append(out, views.TaskItem{
			ID:           t.TaskID,
			Title:        title,
			Status:       string(t.Status),
			AssignedPlan: t.PlanID,
			Priority:     t.CurrentStep,
		})
	}
	return out
}

func mapAuditToEvidence(entries []*audit.AuditEntry) []views.EvidenceEntry {
	out := make([]views.EvidenceEntry, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		ok := e.EventType != audit.EventTaskFailed
		detail := strings.TrimSpace(e.Action)
		if tgt := strings.TrimSpace(e.Target); tgt != "" {
			if detail != "" {
				detail = detail + " → " + tgt
			} else {
				detail = tgt
			}
		}
		res := e.PolicyResult
		if res == "" {
			res = string(e.EventType)
		}
		out = append(out, views.EvidenceEntry{
			Timestamp: e.Timestamp,
			Action:    string(e.EventType),
			Result:    res,
			Detail:    detail,
			Success:   ok,
		})
	}
	return out
}
