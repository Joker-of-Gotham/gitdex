package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/campaign"
	"github.com/your-org/gitdex/internal/collaboration"
	"github.com/your-org/gitdex/internal/identity"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/policy"
)

// Provider implements storage.StorageProvider using PostgreSQL.
type Provider struct {
	pool *pgxpool.Pool
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

// NewProvider creates a PostgreSQL storage provider. maxConns defaults to 10 if 0.
func NewProvider(dsn string, maxConns int32) (*Provider, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	if maxConns <= 0 {
		maxConns = 10
	}
	config.MaxConns = maxConns

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	p := &Provider{pool: pool}
	p.planStore = NewPlanStore(pool)
	p.taskStore = NewTaskStore(pool)
	p.auditLedger = NewAuditStore(pool)
	p.policyStore = NewPolicyStore(pool)
	p.identityStore = NewIdentityStore(pool)
	p.objectStore = NewObjectStore(pool)
	p.contextStore = NewContextStore(pool)
	p.campaignStore = NewCampaignStore(pool)
	p.monitorStore = NewMonitorStore(pool)
	p.triggerStore = NewTriggerStore(pool)
	p.autonomyStore = NewAutonomyStore(pool)
	p.handoffStore = NewHandoffStore(pool)

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
	return runMigrations(ctx, p.pool)
}

// Close releases the connection pool.
func (p *Provider) Close() error {
	p.pool.Close()
	return nil
}
