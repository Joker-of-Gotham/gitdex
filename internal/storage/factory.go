package storage

import (
	"context"
	"fmt"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/campaign"
	"github.com/your-org/gitdex/internal/collaboration"
	"github.com/your-org/gitdex/internal/identity"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/policy"
	"github.com/your-org/gitdex/internal/storage/bbolt"
	"github.com/your-org/gitdex/internal/storage/postgres"
	"github.com/your-org/gitdex/internal/storage/sqlite"
)

// NewProvider creates a StorageProvider for the given configuration.
// For postgres, sqlite, and bbolt backends the caller must eventually
// call Close on the returned provider. The memory backend never
// requires cleanup.
func NewProvider(cfg Config) (StorageProvider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	switch cfg.Type {
	case BackendMemory:
		return newMemoryProvider(), nil
	case BackendPostgres:
		maxConns := int32(cfg.MaxOpenConns)
		if maxConns <= 0 {
			maxConns = 10
		}
		return postgres.NewProvider(cfg.DSN, maxConns)
	case BackendSQLite:
		return sqlite.NewProvider(cfg.DSN)
	case BackendBBolt:
		return bbolt.NewProvider(cfg.DSN)
	default:
		return nil, fmt.Errorf("storage: unsupported backend type %q", cfg.Type)
	}
}

// memoryProvider wraps the existing in-memory store implementations so that
// the rest of the codebase can be wired to StorageProvider immediately
// without waiting for a real database backend.
type memoryProvider struct {
	planStore     planning.PlanStore
	taskStore     orchestrator.TaskStore
	auditLedger   audit.AuditLedger
	policyStore   policy.PolicyBundleStore
	identityStore identity.IdentityStore
	objectStore   collaboration.ObjectStore
	contextStore  collaboration.ContextStore
	campaignStore campaign.CampaignStore
	monitorStore  autonomy.MonitorStore
	triggerStore  autonomy.TriggerStore
	autonomyStore autonomy.AutonomyStore
	handoffStore  autonomy.HandoffStore
}

func newMemoryProvider() *memoryProvider {
	return &memoryProvider{
		planStore:     planning.NewMemoryPlanStore(),
		taskStore:     orchestrator.NewMemoryTaskStore(),
		auditLedger:   audit.NewMemoryAuditLedger(),
		policyStore:   policy.NewMemoryBundleStore(),
		identityStore: identity.NewMemoryIdentityStore(),
		objectStore:   collaboration.NewMemoryObjectStore(),
		contextStore:  collaboration.NewMemoryContextStore(),
		campaignStore: campaign.NewMemoryCampaignStore(),
		monitorStore:  autonomy.NewMemoryMonitorStore(),
		triggerStore:  autonomy.NewMemoryTriggerStore(),
		autonomyStore: autonomy.NewMemoryAutonomyStore(),
		handoffStore:  autonomy.NewMemoryHandoffStore(),
	}
}

func (m *memoryProvider) PlanStore() planning.PlanStore               { return m.planStore }
func (m *memoryProvider) TaskStore() orchestrator.TaskStore           { return m.taskStore }
func (m *memoryProvider) AuditLedger() audit.AuditLedger              { return m.auditLedger }
func (m *memoryProvider) PolicyBundleStore() policy.PolicyBundleStore { return m.policyStore }
func (m *memoryProvider) IdentityStore() identity.IdentityStore       { return m.identityStore }
func (m *memoryProvider) ObjectStore() collaboration.ObjectStore      { return m.objectStore }
func (m *memoryProvider) ContextStore() collaboration.ContextStore    { return m.contextStore }
func (m *memoryProvider) CampaignStore() campaign.CampaignStore       { return m.campaignStore }
func (m *memoryProvider) MonitorStore() autonomy.MonitorStore         { return m.monitorStore }
func (m *memoryProvider) TriggerStore() autonomy.TriggerStore         { return m.triggerStore }
func (m *memoryProvider) AutonomyStore() autonomy.AutonomyStore       { return m.autonomyStore }
func (m *memoryProvider) HandoffStore() autonomy.HandoffStore         { return m.handoffStore }

func (m *memoryProvider) Migrate(_ context.Context) error { return nil } //nolint:revive
func (m *memoryProvider) Close() error                    { return nil }
