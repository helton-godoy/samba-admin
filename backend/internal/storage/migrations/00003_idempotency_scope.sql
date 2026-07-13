-- +goose Up
BEGIN IMMEDIATE;
ALTER TABLE idempotency_keys RENAME TO idempotency_keys_legacy;
CREATE TABLE idempotency_keys (
  key TEXT NOT NULL,
  actor TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  response_status INTEGER NOT NULL,
  response_body TEXT NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  PRIMARY KEY(actor, key)
);
INSERT OR IGNORE INTO idempotency_keys(key,actor,request_hash,response_status,response_body,created_at,expires_at)
  SELECT key,actor,request_hash,response_status,response_body,created_at,expires_at FROM idempotency_keys_legacy;
DROP TABLE idempotency_keys_legacy;
CREATE INDEX idx_idempotency_expiry ON idempotency_keys(expires_at);
COMMIT;

-- +goose Down
BEGIN IMMEDIATE;
DROP INDEX IF EXISTS idx_idempotency_expiry;
ALTER TABLE idempotency_keys RENAME TO idempotency_keys_scoped;
CREATE TABLE idempotency_keys (
  key TEXT PRIMARY KEY,
  actor TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  response_status INTEGER NOT NULL,
  response_body TEXT NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL
);
INSERT OR IGNORE INTO idempotency_keys(key,actor,request_hash,response_status,response_body,created_at,expires_at)
  SELECT key,actor,request_hash,response_status,response_body,created_at,expires_at FROM idempotency_keys_scoped;
DROP TABLE idempotency_keys_scoped;
COMMIT;
