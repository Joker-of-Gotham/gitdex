package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/campaign"
	"github.com/your-org/gitdex/internal/collaboration"
	"github.com/your-org/gitdex/internal/identity"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/policy"
)

// Provider implements storage.StorageProvider using SQLite.
type Provider struct {
	db *sql.DB
	// individual stores
	planStore     *PlanStore
	taskStore     *TaskStore
	auditLedger   *AuditStore
	policyStore   *PolicyStore
	identityStore *IdentityStore
	objectStore   *ObjectStore
	contextStore  *ContextStore
	campaignStore *CampaignStore
	monitorStore  *MonitorStore
	triggerStore  *TriggerStore
	autonomyStore *AutonomyStore
	handoffStore  *HandoffStore
}

// NewProvider creates a SQLite storage provider. Opens database and sets pragmas.
func NewProvider(dsn string) (*Provider, error) {
	if err := ensureParentDir(dsn); err != nil {
		return nil, err
	}

	// Register driver: modernc.org/sqlite registers as "sqlite"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("sqlite: pragma %s: %w", pragma, err)
		}
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: ping: %w", err)
	}

	p := &Provider{db: db}
	p.planStore = NewPlanStore(db)
	p.taskStore = NewTaskStore(db)
	p.auditLedger = NewAuditStore(db)
	p.policyStore = NewPolicyStore(db)
	p.identityStore = NewIdentityStore(db)
	p.objectStore = NewObjectStore(db)
	p.contextStore = NewContextStore(db)
	p.campaignStore = NewCampaignStore(db)
	p.monitorStore = NewMonitorStore(db)
	p.triggerStore = NewTriggerStore(db)
	p.autonomyStore = NewAutonomyStore(db)
	p.handoffStore = NewHandoffStore(db)

	return p, nil
}

// PlanStore returns the plan store.
func (p *Provider) PlanStore() planning.PlanStore { return p.planStore }

// TaskStore returns the task store.
func (p *Provider) TaskStore() orchestrator.TaskStore { return p.taskStore }

// AuditLedger returns the audit ledger.
func (p *Provider) AuditLedger() audit.AuditLedger { return p.auditLedger }

// PolicyBundleStore returns the policy bundle store.
func (p *Provider) PolicyBundleStore() policy.PolicyBundleStore { return p.policyStore }

// IdentityStore returns the identity store.
func (p *Provider) IdentityStore() identity.IdentityStore { return p.identityStore }

// ObjectStore returns the collaboration object store.
func (p *Provider) ObjectStore() collaboration.ObjectStore { return p.objectStore }

// ContextStore returns the task context store.
func (p *Provider) ContextStore() collaboration.ContextStore { return p.contextStore }

// CampaignStore returns the campaign store.
func (p *Provider) CampaignStore() campaign.CampaignStore { return p.campaignStore }

// MonitorStore returns the monitor store.
func (p *Provider) MonitorStore() autonomy.MonitorStore { return p.monitorStore }

// TriggerStore returns the trigger store.
func (p *Provider) TriggerStore() autonomy.TriggerStore { return p.triggerStore }

// AutonomyStore returns the autonomy store.
func (p *Provider) AutonomyStore() autonomy.AutonomyStore { return p.autonomyStore }

// HandoffStore returns the handoff store.
func (p *Provider) HandoffStore() autonomy.HandoffStore { return p.handoffStore }

// Migrate applies pending schema migrations.
func (p *Provider) Migrate(ctx context.Context) error {
	return runMigrations(ctx, p.db)
}

// Close releases the database connection.
func (p *Provider) Close() error {
	return p.db.Close()
}

func ensureParentDir(dsn string) error {
	path := strings.TrimSpace(dsn)
	switch {
	case path == "", path == ":memory:", strings.HasPrefix(path, "file::memory:"):
		return nil
	case strings.HasPrefix(path, "file:"):
		trimmed := strings.TrimPrefix(path, "file:")
		if idx := strings.Index(trimmed, "?"); idx >= 0 {
			trimmed = trimmed[:idx]
		}
		if trimmed == "" {
			return nil
		}
		switch {
		case strings.HasPrefix(trimmed, "///"):
			if len(trimmed) >= 5 && trimmed[4] == ':' {
				path = trimmed[3:]
			} else {
				path = trimmed[2:]
			}
		default:
			path = trimmed
		}
	}

	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("sqlite: create parent dir: %w", err)
	}
	return nil
}
