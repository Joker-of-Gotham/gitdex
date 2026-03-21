-- Gitdex SQLite schema rollback
-- 001_init

DROP TABLE IF EXISTS handoff_packages;
DROP TABLE IF EXISTS autonomy_configs;
DROP TABLE IF EXISTS trigger_events;
DROP TABLE IF EXISTS trigger_configs;
DROP TABLE IF EXISTS monitor_events;
DROP TABLE IF EXISTS monitor_configs;
DROP TABLE IF EXISTS campaigns;
DROP TABLE IF EXISTS task_contexts;
DROP TABLE IF EXISTS collaboration_objects;
DROP TABLE IF EXISTS identities;
DROP TABLE IF EXISTS policy_bundles;
DROP TABLE IF EXISTS audit_entries;
DROP TABLE IF EXISTS task_events;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS approval_records;
DROP TABLE IF EXISTS plans;
