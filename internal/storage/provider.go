package storage

import (
	"context"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/campaign"
	"github.com/your-org/gitdex/internal/collaboration"
	"github.com/your-org/gitdex/internal/identity"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/policy"
)

// StorageProvider is the top-level abstraction that returns domain-specific
// stores. Implementations exist for PostgreSQL, SQLite and BBolt. The CLI
// bootstrap path calls NewProvider once and threads the returned provider
// through the command tree so every handler shares the same backend.
type StorageProvider interface {
	PlanStore() planning.PlanStore
	TaskStore() orchestrator.TaskStore
	AuditLedger() audit.AuditLedger
	PolicyBundleStore() policy.PolicyBundleStore
	IdentityStore() identity.IdentityStore
	ObjectStore() collaboration.ObjectStore
	ContextStore() collaboration.ContextStore
	CampaignStore() campaign.CampaignStore
	MonitorStore() autonomy.MonitorStore
	TriggerStore() autonomy.TriggerStore
	AutonomyStore() autonomy.AutonomyStore
	HandoffStore() autonomy.HandoffStore

	// Migrate applies pending schema migrations (no-op for BBolt).
	Migrate(ctx context.Context) error

	// Close releases connections / file handles held by the provider.
	Close() error
}
