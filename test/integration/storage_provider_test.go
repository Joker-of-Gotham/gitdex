package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/campaign"
	"github.com/your-org/gitdex/internal/identity"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/policy"
	"github.com/your-org/gitdex/internal/storage"
)

func getProviders(t *testing.T) map[string]storage.StorageProvider {
	t.Helper()
	providers := make(map[string]storage.StorageProvider)

	// Memory - always available
	mem, err := storage.NewProvider(storage.Config{Type: storage.BackendMemory})
	if err != nil {
		t.Fatalf("memory provider: %v", err)
	}
	providers["memory"] = mem

	// SQLite - always available (pure Go)
	sqliteDir := t.TempDir()
	sqliteDB := filepath.Join(sqliteDir, "test.db")
	sqlite, err := storage.NewProvider(storage.Config{Type: storage.BackendSQLite, DSN: sqliteDB})
	if err != nil {
		t.Fatalf("sqlite provider: %v", err)
	}
	if err := sqlite.Migrate(context.Background()); err != nil {
		t.Fatalf("sqlite migrate: %v", err)
	}
	providers["sqlite"] = sqlite

	// BBolt - always available
	bboltDir := t.TempDir()
	bboltDB := filepath.Join(bboltDir, "test.db")
	bbolt, err := storage.NewProvider(storage.Config{Type: storage.BackendBBolt, DSN: bboltDB})
	if err != nil {
		t.Fatalf("bbolt provider: %v", err)
	}
	providers["bbolt"] = bbolt

	// PostgreSQL - only if GITDEX_TEST_POSTGRES_DSN is set
	if dsn := os.Getenv("GITDEX_TEST_POSTGRES_DSN"); dsn != "" {
		pg, err := storage.NewProvider(storage.Config{Type: storage.BackendPostgres, DSN: dsn})
		if err != nil {
			t.Skipf("postgres: %v", err)
		}
		if err := pg.Migrate(context.Background()); err != nil {
			t.Skipf("postgres migrate: %v", err)
		}
		providers["postgres"] = pg
	}

	t.Cleanup(func() {
		for _, p := range providers {
			_ = p.Close()
		}
	})
	return providers
}

func TestPlanStore_SaveAndGet(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.PlanStore()
			plan := &planning.Plan{
				PlanID: "test-plan-1",
				Status: planning.PlanDraft,
				Intent: planning.PlanIntent{Source: "test", RawInput: "hello", ActionType: "sync"},
				Steps:  []planning.PlanStep{{Sequence: 1, Action: "sync", Target: "repo", RiskLevel: planning.RiskLow}},
			}
			if err := store.Save(plan); err != nil {
				t.Fatalf("Save: %v", err)
			}
			got, err := store.Get("test-plan-1")
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if got.PlanID != "test-plan-1" {
				t.Fatalf("PlanID = %q, want test-plan-1", got.PlanID)
			}
			if got.Status != planning.PlanDraft {
				t.Fatalf("Status = %q, want draft", got.Status)
			}
			if len(got.Steps) != 1 {
				t.Fatalf("Steps len = %d, want 1", len(got.Steps))
			}
		})
	}
}

func TestPlanStore_GetByTaskID(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.PlanStore()
			plan := &planning.Plan{
				PlanID: "plan-task-lookup",
				TaskID: "task-123",
				Status: planning.PlanDraft,
				Intent: planning.PlanIntent{Source: "test", RawInput: "x", ActionType: "sync"},
			}
			if err := store.Save(plan); err != nil {
				t.Fatalf("Save: %v", err)
			}
			got, err := store.GetByTaskID("task-123")
			if err != nil {
				t.Fatalf("GetByTaskID: %v", err)
			}
			if got.PlanID != "plan-task-lookup" {
				t.Fatalf("PlanID = %q, want plan-task-lookup", got.PlanID)
			}
		})
	}
}

func TestPlanStore_List(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.PlanStore()
			if err := store.Save(&planning.Plan{PlanID: "p1", Status: planning.PlanDraft, Intent: planning.PlanIntent{}}); err != nil {
				t.Fatalf("Save: %v", err)
			}
			list, err := store.List()
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(list) < 1 {
				t.Fatalf("List len = %d, want at least 1", len(list))
			}
		})
	}
}

func TestPlanStore_UpdateStatus(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.PlanStore()
			if err := store.Save(&planning.Plan{PlanID: "p-status", Status: planning.PlanDraft, Intent: planning.PlanIntent{}}); err != nil {
				t.Fatalf("Save: %v", err)
			}
			if err := store.UpdateStatus("p-status", planning.PlanApproved); err != nil {
				t.Fatalf("UpdateStatus: %v", err)
			}
			got, err := store.Get("p-status")
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if got.Status != planning.PlanApproved {
				t.Fatalf("Status = %q, want approved", got.Status)
			}
		})
	}
}

func TestPlanStore_SaveApproval_GetApprovals(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.PlanStore()
			if err := store.Save(&planning.Plan{PlanID: "p-approval", Status: planning.PlanDraft, Intent: planning.PlanIntent{}}); err != nil {
				t.Fatalf("Save: %v", err)
			}
			rec := &planning.ApprovalRecord{
				PlanID:         "p-approval",
				Action:         planning.ActionApprove,
				Actor:          "alice",
				PreviousStatus: planning.PlanDraft,
				NewStatus:      planning.PlanApproved,
			}
			if err := store.SaveApproval(rec); err != nil {
				t.Fatalf("SaveApproval: %v", err)
			}
			approvals, err := store.GetApprovals("p-approval")
			if err != nil {
				t.Fatalf("GetApprovals: %v", err)
			}
			if len(approvals) != 1 {
				t.Fatalf("approvals len = %d, want 1", len(approvals))
			}
			if approvals[0].Actor != "alice" {
				t.Fatalf("Actor = %q, want alice", approvals[0].Actor)
			}
		})
	}
}

func TestTaskStore_SaveAndGet(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.TaskStore()
			task := &orchestrator.Task{
				TaskID:        "task-1",
				CorrelationID: "corr-1",
				PlanID:        "plan-1",
				Status:        orchestrator.TaskQueued,
			}
			if err := store.SaveTask(task); err != nil {
				t.Fatalf("SaveTask: %v", err)
			}
			got, err := store.GetTask("task-1")
			if err != nil {
				t.Fatalf("GetTask: %v", err)
			}
			if got.TaskID != "task-1" {
				t.Fatalf("TaskID = %q, want task-1", got.TaskID)
			}
			if got.CorrelationID != "corr-1" {
				t.Fatalf("CorrelationID = %q, want corr-1", got.CorrelationID)
			}
		})
	}
}

func TestTaskStore_GetByCorrelationID(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.TaskStore()
			if err := store.SaveTask(&orchestrator.Task{TaskID: "t2", CorrelationID: "corr-x", PlanID: "p", Status: orchestrator.TaskQueued}); err != nil {
				t.Fatalf("SaveTask: %v", err)
			}
			got, err := store.GetByCorrelationID("corr-x")
			if err != nil {
				t.Fatalf("GetByCorrelationID: %v", err)
			}
			if got.TaskID != "t2" {
				t.Fatalf("TaskID = %q, want t2", got.TaskID)
			}
		})
	}
}

func TestTaskStore_ListTasks_UpdateTask(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.TaskStore()
			task := &orchestrator.Task{TaskID: "t-list", CorrelationID: "c", PlanID: "p", Status: orchestrator.TaskQueued}
			if err := store.SaveTask(task); err != nil {
				t.Fatalf("SaveTask: %v", err)
			}
			task.Status = orchestrator.TaskSucceeded
			if err := store.UpdateTask(task); err != nil {
				t.Fatalf("UpdateTask: %v", err)
			}
			got, err := store.GetTask("t-list")
			if err != nil {
				t.Fatalf("GetTask: %v", err)
			}
			if got.Status != orchestrator.TaskSucceeded {
				t.Fatalf("Status = %q, want succeeded", got.Status)
			}
			list, err := store.ListTasks()
			if err != nil {
				t.Fatalf("ListTasks: %v", err)
			}
			if len(list) < 1 {
				t.Fatalf("ListTasks len = %d, want at least 1", len(list))
			}
		})
	}
}

func TestAuditLedger_Append_Query_GetByCorrelation_GetByTask(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			ledger := p.AuditLedger()
			entry := &audit.AuditEntry{
				EntryID:       audit.GenerateEntryID(),
				CorrelationID: "corr-a",
				TaskID:        "task-a",
				PlanID:        "plan-a",
				EventType:     audit.EventPlanCreated,
				Actor:         "system",
				Action:        "create",
				Target:        "plan",
				Timestamp:     time.Now().UTC(),
			}
			if err := ledger.Append(entry); err != nil {
				t.Fatalf("Append: %v", err)
			}
			byCorr, err := ledger.GetByCorrelation("corr-a")
			if err != nil {
				t.Fatalf("GetByCorrelation: %v", err)
			}
			if len(byCorr) != 1 {
				t.Fatalf("GetByCorrelation len = %d, want 1", len(byCorr))
			}
			byTask, err := ledger.GetByTask("task-a")
			if err != nil {
				t.Fatalf("GetByTask: %v", err)
			}
			if len(byTask) != 1 {
				t.Fatalf("GetByTask len = %d, want 1", len(byTask))
			}
			query, err := ledger.Query(audit.AuditFilter{EventType: audit.EventPlanCreated})
			if err != nil {
				t.Fatalf("Query: %v", err)
			}
			if len(query) < 1 {
				t.Fatalf("Query len = %d, want at least 1", len(query))
			}
			e, ok := ledger.GetByEntryID(entry.EntryID)
			if !ok {
				t.Fatal("GetByEntryID: not found")
			}
			if e.Actor != "system" {
				t.Fatalf("Actor = %q, want system", e.Actor)
			}
		})
	}
}

func TestPolicyBundleStore_Save_Get_SetActive_GetActive(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.PolicyBundleStore()
			bundle := &policy.PolicyBundle{BundleID: "bundle-1", Name: "test", Version: "1.0"}
			if err := store.SaveBundle(bundle); err != nil {
				t.Fatalf("SaveBundle: %v", err)
			}
			got, err := store.GetBundle("bundle-1")
			if err != nil {
				t.Fatalf("GetBundle: %v", err)
			}
			if got.Name != "test" {
				t.Fatalf("Name = %q, want test", got.Name)
			}
			active, err := store.GetActiveBundle()
			if err != nil {
				t.Fatalf("GetActiveBundle: %v", err)
			}
			if active != nil && active.BundleID != "bundle-1" {
				t.Fatalf("active bundle = %q, want bundle-1", active.BundleID)
			}
			bundle2 := &policy.PolicyBundle{BundleID: "bundle-2", Name: "test2", Version: "1.0"}
			if err := store.SaveBundle(bundle2); err != nil {
				t.Fatalf("SaveBundle 2: %v", err)
			}
			if err := store.SetActiveBundle("bundle-2"); err != nil {
				t.Fatalf("SetActiveBundle: %v", err)
			}
			active, _ = store.GetActiveBundle()
			if active != nil && active.BundleID != "bundle-2" {
				t.Fatalf("active bundle after set = %q, want bundle-2", active.BundleID)
			}
		})
	}
}

func TestIdentityStore_Save_Get_SetCurrent_GetCurrent(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.IdentityStore()
			id := &identity.AppIdentity{IdentityID: "id-1", IdentityType: identity.IdentityTypeGitHubApp}
			if err := store.SaveIdentity(id); err != nil {
				t.Fatalf("SaveIdentity: %v", err)
			}
			got, err := store.GetIdentity("id-1")
			if err != nil {
				t.Fatalf("GetIdentity: %v", err)
			}
			if got.IdentityID != "id-1" {
				t.Fatalf("IdentityID = %q, want id-1", got.IdentityID)
			}
			if err := store.SetCurrentIdentity("id-1"); err != nil {
				t.Fatalf("SetCurrentIdentity: %v", err)
			}
			cur, err := store.GetCurrentIdentity()
			if err != nil {
				t.Fatalf("GetCurrentIdentity: %v", err)
			}
			if cur != nil && cur.IdentityID != "id-1" {
				t.Fatalf("current identity = %q, want id-1", cur.IdentityID)
			}
		})
	}
}

func TestCampaignStore_Save_Get_ListCampaigns(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.CampaignStore()
			c := &campaign.Campaign{
				CampaignID:  "camp-1",
				Name:        "Test",
				Status:      campaign.StatusDraft,
				TargetRepos: []campaign.RepoTarget{{Owner: "o", Repo: "r", InclusionStatus: campaign.InclusionPending}},
			}
			if err := store.SaveCampaign(c); err != nil {
				t.Fatalf("SaveCampaign: %v", err)
			}
			got, err := store.GetCampaign("camp-1")
			if err != nil {
				t.Fatalf("GetCampaign: %v", err)
			}
			if got.Name != "Test" {
				t.Fatalf("Name = %q, want Test", got.Name)
			}
			list, err := store.ListCampaigns()
			if err != nil {
				t.Fatalf("ListCampaigns: %v", err)
			}
			if len(list) < 1 {
				t.Fatalf("ListCampaigns len = %d, want at least 1", len(list))
			}
		})
	}
}

func TestMonitorStore_Save_Get_ListMonitorConfigs(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.MonitorStore()
			cfg := &autonomy.MonitorConfig{
				MonitorID: "mon-1",
				RepoOwner: "owner",
				RepoName:  "repo",
				Interval:  "5m",
				Checks:    []string{"drift"},
				Enabled:   true,
			}
			if err := store.SaveMonitorConfig(cfg); err != nil {
				t.Fatalf("SaveMonitorConfig: %v", err)
			}
			got, err := store.GetMonitorConfig(cfg.MonitorID)
			if err != nil {
				t.Fatalf("GetMonitorConfig: %v", err)
			}
			if got.RepoOwner != "owner" {
				t.Fatalf("RepoOwner = %q, want owner", got.RepoOwner)
			}
			list, err := store.ListMonitorConfigs()
			if err != nil {
				t.Fatalf("ListMonitorConfigs: %v", err)
			}
			if len(list) < 1 {
				t.Fatalf("ListMonitorConfigs len = %d, want at least 1", len(list))
			}
		})
	}
}

func TestTriggerStore_Save_Get_ListTriggers(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.TriggerStore()
			cfg := &autonomy.TriggerConfig{
				TriggerID:      "tr-1",
				TriggerType:    autonomy.TriggerTypeEvent,
				Name:           "webhook",
				Source:         "github",
				ActionTemplate: "sync",
				Enabled:        true,
			}
			if err := store.SaveTrigger(cfg); err != nil {
				t.Fatalf("SaveTrigger: %v", err)
			}
			got, err := store.GetTrigger("tr-1")
			if err != nil {
				t.Fatalf("GetTrigger: %v", err)
			}
			if got.Name != "webhook" {
				t.Fatalf("Name = %q, want webhook", got.Name)
			}
			list, err := store.ListTriggers()
			if err != nil {
				t.Fatalf("ListTriggers: %v", err)
			}
			if len(list) < 1 {
				t.Fatalf("ListTriggers len = %d, want at least 1", len(list))
			}
		})
	}
}

func TestAutonomyStore_Save_Get_SetActive_GetActive(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.AutonomyStore()
			cfg := &autonomy.AutonomyConfig{
				ConfigID: "cfg-1",
				Name:     "default",
			}
			if err := store.SaveConfig(cfg); err != nil {
				t.Fatalf("SaveConfig: %v", err)
			}
			got, err := store.GetConfig(cfg.ConfigID)
			if err != nil {
				t.Fatalf("GetConfig: %v", err)
			}
			if got.Name != "default" {
				t.Fatalf("Name = %q, want default", got.Name)
			}
			active, err := store.GetActiveConfig()
			if err != nil {
				t.Fatalf("GetActiveConfig: %v", err)
			}
			if active != nil && active.ConfigID != "cfg-1" {
				t.Fatalf("active config = %q, want cfg-1", active.ConfigID)
			}
			cfg2 := &autonomy.AutonomyConfig{ConfigID: "cfg-2", Name: "other"}
			if err := store.SaveConfig(cfg2); err != nil {
				t.Fatalf("SaveConfig 2: %v", err)
			}
			if err := store.SetActiveConfig("cfg-2"); err != nil {
				t.Fatalf("SetActiveConfig: %v", err)
			}
			active, _ = store.GetActiveConfig()
			if active != nil && active.ConfigID != "cfg-2" {
				t.Fatalf("active after set = %q, want cfg-2", active.ConfigID)
			}
		})
	}
}

func TestHandoffStore_SavePackage_GetPackage_GetByTaskID(t *testing.T) {
	for name, p := range getProviders(t) {
		t.Run(name, func(t *testing.T) {
			store := p.HandoffStore()
			pkg := &autonomy.HandoffPackage{
				TaskID:       "task-handoff",
				TaskSummary:  "test handoff",
				CurrentState: "paused",
			}
			if err := store.SavePackage(pkg); err != nil {
				t.Fatalf("SavePackage: %v", err)
			}
			got, err := store.GetPackage(pkg.PackageID)
			if err != nil {
				t.Fatalf("GetPackage: %v", err)
			}
			if got.TaskID != "task-handoff" {
				t.Fatalf("TaskID = %q, want task-handoff", got.TaskID)
			}
			byTask, err := store.GetByTaskID("task-handoff")
			if err != nil {
				t.Fatalf("GetByTaskID: %v", err)
			}
			if byTask.PackageID != pkg.PackageID {
				t.Fatalf("GetByTaskID PackageID = %q", byTask.PackageID)
			}
		})
	}
}
