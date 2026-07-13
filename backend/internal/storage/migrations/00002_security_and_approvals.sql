-- +goose Up
-- Configuracoes de seguranca ficam separadas dos usuarios para manter esta
-- migracao aditiva em instalacoes anteriores a MFA e aprovacao de mudancas.
CREATE TABLE IF NOT EXISTS user_security (
  user_id TEXT PRIMARY KEY,
  is_break_glass INTEGER NOT NULL DEFAULT 0,
  mfa_required INTEGER NOT NULL DEFAULT 0,
  password_expires_at TEXT,
  allowed_cidrs_json TEXT,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS mfa_totp (
  user_id TEXT PRIMARY KEY,
  secret_ciphertext TEXT NOT NULL,
  pending_ciphertext TEXT,
  enabled_at TEXT,
  rotated_at TEXT,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  code_hash TEXT NOT NULL,
  created_at TEXT NOT NULL,
  used_at TEXT,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_mfa_recovery_user ON mfa_recovery_codes(user_id);
CREATE TABLE IF NOT EXISTS mfa_challenges (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  csrf_token TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  ip TEXT,
  user_agent TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_mfa_challenges_expiry ON mfa_challenges(expires_at);
CREATE TABLE IF NOT EXISTS change_requests (
  id TEXT PRIMARY KEY,
  operation TEXT NOT NULL,
  resource TEXT NOT NULL,
  risk TEXT NOT NULL,
  status TEXT NOT NULL,
  requested_by TEXT NOT NULL,
  requested_at TEXT NOT NULL,
  justification TEXT NOT NULL,
  maintenance_start TEXT,
  maintenance_end TEXT,
  impact TEXT NOT NULL,
  rollback_plan TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  required_capability TEXT,
  job_id TEXT,
  approved_by TEXT,
  approved_at TEXT,
  decision_reason TEXT,
  expires_at TEXT,
  FOREIGN KEY(requested_by) REFERENCES users(id),
  FOREIGN KEY(approved_by) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_change_requests_status ON change_requests(status, requested_at DESC);

-- +goose Down
DROP TABLE IF EXISTS change_requests;
DROP TABLE IF EXISTS mfa_challenges;
DROP TABLE IF EXISTS mfa_recovery_codes;
DROP TABLE IF EXISTS mfa_totp;
DROP TABLE IF EXISTS user_security;
