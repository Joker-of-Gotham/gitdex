-- Gitdex PostgreSQL schema
-- 001_init

CREATE TABLE IF NOT EXISTS plans (
    plan_id TEXT PRIMARY KEY,
    task_id TEXT,
    status TEXT NOT NULL,
    intent JSONB NOT NULL DEFAULT '{}',
    scope JSONB NOT NULL DEFAULT '{}',
    steps JSONB NOT NULL DEFAULT '[]',
    risk_level TEXT,
    policy_result JSONB,
    execution_mode TEXT,
    deferred_until TIMESTAMPTZ,
    evidence_refs JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_plans_task_id ON plans(task_id);
CREATE INDEX idx_plans_status ON plans(status);

CREATE TABLE IF NOT EXISTS approval_records (
    record_id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL,
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    reason TEXT,
    previous_status TEXT,
    new_status TEXT,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_approval_records_plan_id ON approval_records(plan_id);

CREATE TABLE IF NOT EXISTS tasks (
    task_id TEXT PRIMARY KEY,
    correlation_id TEXT,
    plan_id TEXT,
    status TEXT NOT NULL,
    current_step INT DEFAULT 0,
    steps JSONB DEFAULT '[]',
    result JSONB,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_tasks_correlation_id ON tasks(correlation_id);
CREATE INDEX idx_tasks_plan_id ON tasks(plan_id);
CREATE INDEX idx_tasks_status ON tasks(status);

CREATE TABLE IF NOT EXISTS task_events (
    event_id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    from_status TEXT,
    to_status TEXT,
    step_index INT,
    detail TEXT,
    timestamp TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_task_events_task_id ON task_events(task_id);

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
    evidence_refs JSONB DEFAULT '[]',
    timestamp TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_audit_entries_correlation_id ON audit_entries(correlation_id);
CREATE INDEX idx_audit_entries_task_id ON audit_entries(task_id);
CREATE INDEX idx_audit_entries_plan_id ON audit_entries(plan_id);
CREATE INDEX idx_audit_entries_timestamp ON audit_entries(timestamp);
CREATE INDEX idx_audit_entries_event_type ON audit_entries(event_type);

CREATE TABLE IF NOT EXISTS policy_bundles (
    bundle_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    capability_grants JSONB DEFAULT '[]',
    protected_targets JSONB DEFAULT '[]',
    approval_rules JSONB DEFAULT '[]',
    risk_thresholds JSONB DEFAULT '{}',
    data_handling_rules JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_policy_bundles_is_active ON policy_bundles(is_active) WHERE is_active;

CREATE TABLE IF NOT EXISTS identities (
    identity_id TEXT PRIMARY KEY,
    identity_type TEXT NOT NULL,
    app_id TEXT,
    installation_id TEXT,
    org_scope TEXT,
    repo_scope TEXT,
    capabilities JSONB DEFAULT '[]',
    scope_grants JSONB DEFAULT '[]',
    is_current BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_identities_is_current ON identities(is_current) WHERE is_current;

CREATE TABLE IF NOT EXISTS collaboration_objects (
    object_id TEXT PRIMARY KEY,
    object_type TEXT NOT NULL,
    repo_owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    number INT NOT NULL,
    title TEXT,
    state TEXT,
    author TEXT,
    assignees JSONB DEFAULT '[]',
    labels JSONB DEFAULT '[]',
    milestone TEXT,
    body TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    comments_count INT DEFAULT 0,
    url TEXT,
    UNIQUE(repo_owner, repo_name, number)
);

CREATE INDEX idx_collaboration_objects_repo ON collaboration_objects(repo_owner, repo_name);
CREATE INDEX idx_collaboration_objects_type ON collaboration_objects(object_type);
CREATE INDEX idx_collaboration_objects_state ON collaboration_objects(state);

CREATE TABLE IF NOT EXISTS task_contexts (
    context_id TEXT PRIMARY KEY,
    primary_object_ref TEXT UNIQUE NOT NULL,
    linked_objects JSONB DEFAULT '[]',
    related_tasks JSONB DEFAULT '[]',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_task_contexts_primary_object_ref ON task_contexts(primary_object_ref);

CREATE TABLE IF NOT EXISTS campaigns (
    campaign_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    target_repos JSONB DEFAULT '[]',
    plan_template TEXT,
    policy_bundle_id TEXT,
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_campaigns_status ON campaigns(status);

CREATE TABLE IF NOT EXISTS monitor_configs (
    monitor_id TEXT PRIMARY KEY,
    repo_owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    interval TEXT,
    checks JSONB DEFAULT '[]',
    enabled BOOLEAN DEFAULT TRUE
);

CREATE INDEX idx_monitor_configs_repo ON monitor_configs(repo_owner, repo_name);
CREATE INDEX idx_monitor_configs_enabled ON monitor_configs(enabled);

CREATE TABLE IF NOT EXISTS monitor_events (
    event_id TEXT PRIMARY KEY,
    monitor_id TEXT NOT NULL,
    repo_owner TEXT NOT NULL,
    repo_name TEXT NOT NULL,
    check_name TEXT,
    status TEXT,
    message TEXT,
    timestamp TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_monitor_events_monitor_id ON monitor_events(monitor_id);
CREATE INDEX idx_monitor_events_timestamp ON monitor_events(timestamp);

CREATE TABLE IF NOT EXISTS trigger_configs (
    trigger_id TEXT PRIMARY KEY,
    trigger_type TEXT NOT NULL,
    name TEXT NOT NULL,
    source TEXT,
    pattern TEXT,
    action_template TEXT,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_trigger_configs_enabled ON trigger_configs(enabled);

CREATE TABLE IF NOT EXISTS trigger_events (
    event_id TEXT PRIMARY KEY,
    trigger_id TEXT NOT NULL,
    trigger_type TEXT,
    source_event TEXT,
    resulting_task_id TEXT,
    timestamp TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_trigger_events_trigger_id ON trigger_events(trigger_id);
CREATE INDEX idx_trigger_events_timestamp ON trigger_events(timestamp);

CREATE TABLE IF NOT EXISTS autonomy_configs (
    config_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    capability_autonomies JSONB DEFAULT '[]',
    default_level TEXT,
    is_active BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_autonomy_configs_is_active ON autonomy_configs(is_active) WHERE is_active;

CREATE TABLE IF NOT EXISTS handoff_packages (
    package_id TEXT PRIMARY KEY,
    task_id TEXT UNIQUE NOT NULL,
    task_summary TEXT,
    current_state TEXT,
    completed_steps JSONB DEFAULT '[]',
    pending_steps JSONB DEFAULT '[]',
    context_data JSONB DEFAULT '{}',
    artifacts JSONB DEFAULT '[]',
    recommendations JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_handoff_packages_task_id ON handoff_packages(task_id);
