-- +goose Up
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  failed_attempts INTEGER NOT NULL DEFAULT 0,
  locked_until TEXT,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS roles (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS permissions (id TEXT PRIMARY KEY, action TEXT NOT NULL, resource TEXT NOT NULL, UNIQUE(action, resource));
CREATE TABLE IF NOT EXISTS user_roles (user_id TEXT NOT NULL, role_id TEXT NOT NULL, PRIMARY KEY(user_id, role_id), FOREIGN KEY(user_id) REFERENCES users(id), FOREIGN KEY(role_id) REFERENCES roles(id));
CREATE TABLE IF NOT EXISTS role_permissions (role_id TEXT NOT NULL, permission_id TEXT NOT NULL, PRIMARY KEY(role_id, permission_id), FOREIGN KEY(role_id) REFERENCES roles(id), FOREIGN KEY(permission_id) REFERENCES permissions(id));
CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  csrf_token TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  ip TEXT,
  user_agent TEXT,
  FOREIGN KEY(user_id) REFERENCES users(id)
);
CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  requested_by TEXT NOT NULL,
  requested_at TEXT NOT NULL,
  started_at TEXT,
  completed_at TEXT,
  operation TEXT NOT NULL,
  resource TEXT NOT NULL,
  status TEXT NOT NULL,
  progress INTEGER NOT NULL,
  current_step TEXT,
  summary TEXT NOT NULL,
  error TEXT,
  correlation_id TEXT NOT NULL,
  cancellable INTEGER NOT NULL,
  rollback_available INTEGER NOT NULL,
  payload_json TEXT,
  lock_key TEXT
);
CREATE TABLE IF NOT EXISTS events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_type TEXT NOT NULL,
  resource_id TEXT,
  payload_json TEXT NOT NULL,
  correlation_id TEXT,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS audit_log (
  id TEXT PRIMARY KEY,
  timestamp TEXT NOT NULL,
  actor TEXT NOT NULL,
  roles_json TEXT NOT NULL,
  source TEXT NOT NULL,
  ip TEXT,
  operation TEXT NOT NULL,
  resource TEXT NOT NULL,
  before_json TEXT,
  after_json TEXT,
  result TEXT NOT NULL,
  correlation_id TEXT NOT NULL,
  job_id TEXT,
  reason TEXT,
  approval_json TEXT,
  hash TEXT NOT NULL,
  previous_hash TEXT
);
CREATE TABLE IF NOT EXISTS configurations (
  id TEXT PRIMARY KEY,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  current_version INTEGER NOT NULL,
  content_json TEXT NOT NULL,
  etag TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(resource_type, resource_id)
);
CREATE TABLE IF NOT EXISTS configuration_versions (
  id TEXT PRIMARY KEY,
  configuration_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  content_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  created_by TEXT NOT NULL,
  reason TEXT,
  UNIQUE(configuration_id, version)
);
CREATE TABLE IF NOT EXISTS backups (
  id TEXT PRIMARY KEY,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  artifact_path TEXT NOT NULL,
  checksum TEXT NOT NULL,
  created_at TEXT NOT NULL,
  created_by TEXT NOT NULL,
  status TEXT NOT NULL,
  metadata_json TEXT
);
CREATE TABLE IF NOT EXISTS resource_locks (
  resource_key TEXT PRIMARY KEY,
  owner_job_id TEXT NOT NULL,
  acquired_at TEXT NOT NULL,
  expires_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS idempotency_keys (
  key TEXT PRIMARY KEY,
  actor TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  response_status INTEGER NOT NULL,
  response_body TEXT NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_jobs_requested_at ON jobs(requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp DESC);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS resource_locks;
DROP TABLE IF EXISTS backups;
DROP TABLE IF EXISTS configuration_versions;
DROP TABLE IF EXISTS configurations;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
