package bbolt

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"
)

// Bucket name constants.
var (
	bucketPlans                = []byte("plans")
	bucketApprovalRecords      = []byte("approval_records")
	bucketTasks                = []byte("tasks")
	bucketTaskEvents           = []byte("task_events")
	bucketAuditEntries         = []byte("audit_entries")
	bucketPolicyBundles        = []byte("policy_bundles")
	bucketIdentities           = []byte("identities")
	bucketCollaborationObjs    = []byte("collaboration_objects")
	bucketTaskContexts         = []byte("task_contexts")
	bucketCampaigns            = []byte("campaigns")
	bucketMonitorConfigs       = []byte("monitor_configs")
	bucketMonitorEvents        = []byte("monitor_events")
	bucketTriggerConfigs       = []byte("trigger_configs")
	bucketTriggerEvents        = []byte("trigger_events")
	bucketAutonomyConfigs      = []byte("autonomy_configs")
	bucketHandoffPackages      = []byte("handoff_packages")
	bucketPlansByTaskID        = []byte("plans_by_task_id")
	bucketTasksByCorrelationID = []byte("tasks_by_correlation_id")
	bucketContextsByObjectRef  = []byte("contexts_by_object_ref")
	bucketHandoffsByTaskID     = []byte("handoffs_by_task_id")
	bucketObjectsByRepoNumber  = []byte("objects_by_repo_number")
)

// ActiveKey is the key used for storing the active/current ID in buckets.
const ActiveKey = "_active"

var (
	ErrBucketNotFound = errors.New("bucket not found")
	ErrNilEntry       = errors.New("cannot append nil entry")
)

// jsonMarshal marshals v to JSON bytes.
func jsonMarshal(v any) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("cannot marshal nil")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return data, nil
}

// jsonUnmarshal unmarshals data into v.
func jsonUnmarshal(data []byte, v any) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// bucketOrCreate returns a bucket, creating it if it doesn't exist.
func bucketOrCreate(tx *bbolt.Tx, name []byte) (*bbolt.Bucket, error) {
	b, err := tx.CreateBucketIfNotExists(name)
	if err != nil {
		return nil, fmt.Errorf("create bucket %s: %w", string(name), err)
	}
	return b, nil
}

// getBucket returns an existing bucket.
func getBucket(tx *bbolt.Tx, name []byte) *bbolt.Bucket {
	return tx.Bucket(name)
}
