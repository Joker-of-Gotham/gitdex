-- Gitdex SQLite schema
-- 001_init
-- JSONB → TEXT, TIMESTAMPTZ → TEXT (ISO 8601), BOOLEAN → INTEGER (0/1)

CREATE TABLE IF NOT EXISTS plans (
    plan_id TEXT PRIMARY KEY,
    task_id TEXT,
    status TEXT NOT NULL,
    intent TEXT NOT NULL DEFAULT '{}',
    scope TEXT NOT NULL DEFAULT '{}',
    steps TEXT NOT NULL DEFAULT '[]',
    risk_level TEXT,
    policy_result TEXT,
    execution_mode TEXT,
    deferred_until TEXT,
    evidence_refs TEXT DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plans_task_id ON plans(task_id);
CREATE INDEX IF NOT EXISTS idx_plans_status ON plans(status);

CREATE TABLE IF NOT EXISTS approval_records (
    record_id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL,
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    reason TEXT,
    previous_status TEXT,
    new_status TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approval_records_plan_id ON approval_records(plan_id);

CREATE TABLE IF NOT EXISTS tasks (
    task_id TEXT PRIMARY KEY,
    correlation_id TEXT,
    plan_id TEXT,
    status TEXT NOT NULL,
    current_step INTEGER DEFAULT 0,
    steps TEXT DEFAULT '[]',
    result TEXT,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_correlation_id ON tasks(correlation_id);
CREATE INDEX IF NOT EXISTS idx_tasks_plan_id ON tasks(plan_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

CREATE TABLE IF NOT EXISTS task_events (
    event_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    from_status TEXT,
    to_status TEXT,
    step_index INTEGER,
    detail TEXT,
    timestamp TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_events_task_id ON task_events(task_id);

CREATE TABLE IF NOT EXISTS audit_entries (
    entry_id TEXT PRIMARY KEY,
    correlation_id TEXT,
    task_id TEXT,
    plan_id TEXT,
    event_type TEXT NOT NULL,
    actor TEXT,
    action TEXT,
    target TEXT,
    policy_result TEXT,
    evidence_refs TEXT DEFAULT '[]',
    timestamp TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_entries_correlation_id ON audit_entries(correlation_id);
CREATE INDEX IF NOT EXISTS idx_audit_entries_task_id ON audit_entries(task_id);
CREATE INDEX IF NOT EXISTS idx_audit_entries_plan_id ON audit_entries(plan_id);
CREATE INDEX IF NOT EXISTS idx_audit_entries_timestamp ON audit_entries(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_entries_event_type ON audit_entries(event_type);

CREATE TABLE IF NOT EXISTS policy_bundles (
    bundle_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    capability_grants TEXT DEFAULT '[]',
    protected_targets TEXT DEFAULT '[]',
    approval_rules TEXT DEFAULT '[]',
    risk_thresholds TEXT DEFAULT '{}',
    data_handling_rules TEXT DEFAULT '[]',
    is_active INTEGER DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_policy_bundles_is_active ON policy_bundles(is_active) WHERE is_active;

CREATE TABLE IF NOT EXISTS identities (
    identity_id TEXT PRIMARY KEY,
    identity_type TEXT NOT NULL,
    app_id TEXT,
    installation_id TEXT,
    org_scope TEXT,
    repo_scope TEXT,
    capabilities TEXT DEFAULT '[]',
    scope_grants TEXT DEFAULT '[]',
    is_current INTEGER DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_identities_is_current ON identities(is_current) WHERE is_current;

CREATE TABLE IF NOT EXISTS collaboration_objects (
    object_id TEXT PRIMARY KEY,
    object_type TEXT NOT NULL,
    repo_owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    number INTEGER NOT NULL,
    title TEXT,
    state TEXT,
    author TEXT,
    assignees TEXT DEFAULT '[]',
    labels TEXT DEFAULT '[]',
    milestone TEXT,
    body TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    comments_count INTEGER DEFAULT 0,
    url TEXT,
    UNIQUE(repo_owner, repo_name, number)
);

CREATE INDEX IF NOT EXISTS idx_collaboration_objects_repo ON collaboration_objects(repo_owner, repo_name);
CREATE INDEX IF NOT EXISTS idx_collaboration_objects_type ON collaboration_objects(object_type);
CREATE INDEX IF NOT EXISTS idx_collaboration_objects_state ON collaboration_objects(state);

CREATE TABLE IF NOT EXISTS task_contexts (
    context_id TEXT PRIMARY KEY,
    primary_object_ref TEXT UNIQUE NOT NULL,
    linked_objects TEXT DEFAULT '[]',
    related_tasks TEXT DEFAULT '[]',
    notes TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_contexts_primary_object_ref ON task_contexts(primary_object_ref);

CREATE TABLE IF NOT EXISTS campaigns (
    campaign_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    target_repos TEXT DEFAULT '[]',
    plan_template TEXT,
    policy_bundle_id TEXT,
    created_by TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);

CREATE TABLE IF NOT EXISTS monitor_configs (
    monitor_id TEXT PRIMARY KEY,
    repo_owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    interval TEXT,
    checks TEXT DEFAULT '[]',
    enabled INTEGER DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_monitor_configs_repo ON monitor_configs(repo_owner, repo_name);
CREATE INDEX IF NOT EXISTS idx_monitor_configs_enabled ON monitor_configs(enabled);

CREATE TABLE IF NOT EXISTS monitor_events (
    event_id TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL,
    repo_owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    check_name TEXT,
    status TEXT,
    message TEXT,
    timestamp TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_monitor_events_monitor_id ON monitor_events(monitor_id);
CREATE INDEX IF NOT EXISTS idx_monitor_events_timestamp ON monitor_events(timestamp);

CREATE TABLE IF NOT EXISTS trigger_configs (
    trigger_id TEXT PRIMARY KEY,
    trigger_type TEXT NOT NULL,
    name TEXT NOT NULL,
    source TEXT,
    pattern TEXT,
    action_template TEXT,
    enabled INTEGER DEFAULT 1,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_trigger_configs_enabled ON trigger_configs(enabled);

CREATE TABLE IF NOT EXISTS trigger_events (
    event_id TEXT PRIMARY KEY,
    trigger_id TEXT NOT NULL,
    trigger_type TEXT,
    source_event TEXT,
    resulting_task_id TEXT,
    timestamp TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_trigger_events_trigger_id ON trigger_events(trigger_id);
CREATE INDEX IF NOT EXISTS idx_trigger_events_timestamp ON trigger_events(timestamp);

CREATE TABLE IF NOT EXISTS autonomy_configs (
    config_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    capability_autonomies TEXT DEFAULT '[]',
    default_level TEXT,
    is_active INTEGER DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_autonomy_configs_is_active ON autonomy_configs(is_active) WHERE is_active;

CREATE TABLE IF NOT EXISTS handoff_packages (
    package_id TEXT PRIMARY KEY,
    task_id TEXT UNIQUE NOT NULL,
    task_summary TEXT,
    current_state TEXT,
    completed_steps TEXT DEFAULT '[]',
    pending_steps TEXT DEFAULT '[]',
    context_data TEXT DEFAULT '{}',
    artifacts TEXT DEFAULT '[]',
    recommendations TEXT DEFAULT '[]',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_handoff_packages_task_id ON handoff_packages(task_id);
