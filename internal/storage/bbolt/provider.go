package bbolt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/campaign"
	"github.com/your-org/gitdex/internal/collaboration"
	"github.com/your-org/gitdex/internal/identity"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/policy"
)

var allBuckets = [][]byte{
	bucketPlans, bucketApprovalRecords, bucketTasks, bucketTaskEvents,
	bucketAuditEntries, bucketPolicyBundles, bucketIdentities,
	bucketCollaborationObjs, bucketTaskContexts, bucketCampaigns,
	bucketMonitorConfigs, bucketMonitorEvents, bucketTriggerConfigs, bucketTriggerEvents,
	bucketAutonomyConfigs, bucketHandoffPackages,
	bucketPlansByTaskID, bucketTasksByCorrelationID, bucketContextsByObjectRef,
	bucketHandoffsByTaskID, bucketObjectsByRepoNumber,
}

// Provider implements storage.StorageProvider using BBolt.
type Provider struct {
	db            *bbolt.DB
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

// NewProvider opens a BBolt database at path and creates all buckets and stores.
func NewProvider(path string) (*Provider, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("bbolt mkdir: %w", err)
	}

	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("bbolt open: %w", err)
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		for _, name := range allBuckets {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return fmt.Errorf("create bucket %s: %w", string(name), err)
			}
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, err
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

// AutonomyStore returns the autonomy config store.
func (p *Provider) AutonomyStore() autonomy.AutonomyStore { return p.autonomyStore }

// HandoffStore returns the handoff package store.
func (p *Provider) HandoffStore() autonomy.HandoffStore { return p.handoffStore }

// Migrate is a no-op for BBolt (no schema migrations).
func (p *Provider) Migrate(_ context.Context) error { return nil } //nolint:revive

// Close closes the database.
func (p *Provider) Close() error {
	if p.db == nil {
		return nil
	}
	return p.db.Close()
}
