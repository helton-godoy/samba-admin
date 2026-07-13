package storage

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/models"
	"github.com/hu-ufcat/samba-admin-backend/internal/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct{ DB *sqlite.DB }

type UserRecord struct {
	ID, Username, DisplayName, PasswordHash string
	Failed                                  int
	LockedUntil                             sqlite.NullString
	BreakGlass                              bool
	MFARequired                             bool
	PasswordExpiresAt                       sqlite.NullString
	AllowedCIDRs                            sqlite.NullString
}

func Open(path string) (*Store, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			return nil, err
		}
	}
	db, err := sqlite.Open(path)
	if err != nil {
		return nil, err
	}
	for _, stmt := range []string{"PRAGMA journal_mode=WAL;", "PRAGMA foreign_keys=ON;", "PRAGMA busy_timeout=5000;"} {
		if _, err = db.Exec(stmt); err != nil {
			db.Close()
			return nil, err
		}
	}
	if err = applyMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}
	return &Store{DB: db}, nil
}

// applyMigrations records applied files so new schema changes are safe on both
// fresh installations and databases created by earlier releases.
func applyMigrations(db *sqlite.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			versions = append(versions, entry.Name())
		}
	}
	sort.Strings(versions)
	for _, version := range versions {
		var recorded string
		err := db.QueryRow(`SELECT version FROM schema_migrations WHERE version=?`, version).Scan(&recorded)
		if err == nil {
			continue
		}
		if !errors.Is(err, sqlite.ErrNoRows) {
			return err
		}
		migration, err := migrations.ReadFile("migrations/" + version)
		if err != nil {
			return err
		}
		if err = db.ExecScript(stripGooseDirectives(string(migration))); err != nil {
			return fmt.Errorf("%s: %w", version, err)
		}
		if _, err = db.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`, version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	return nil
}
func stripGooseDirectives(s string) string {
	lines := make([]byte, 0, len(s))
	for _, line := range splitLines(s) {
		text := string(line)
		if text == "-- +goose Down" {
			break
		}
		if len(text) >= 9 && text[:9] == "-- +goose" {
			continue
		}
		lines = append(lines, line...)
		lines = append(lines, '\n')
	}
	return string(lines)
}

func splitLines(s string) [][]byte {
	var out [][]byte
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, []byte(s[start:i]))
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, []byte(s[start:]))
	}
	return out
}
func (s *Store) Close() error { return s.DB.Close() }
func (s *Store) InsertJob(ctx context.Context, j models.Job, payload any, lockKey string) error {
	b, _ := json.Marshal(payload)
	_, err := s.DB.Exec(`INSERT INTO jobs(id,requested_by,requested_at,operation,resource,status,progress,current_step,summary,error,correlation_id,cancellable,rollback_available,payload_json,lock_key) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, j.ID, j.RequestedBy, j.RequestedAt.UTC().Format(time.RFC3339Nano), j.Operation, j.Resource, j.Status, j.Progress, j.CurrentStep, j.Summary, nullIfEmpty(j.Error), j.CorrelationID, boolInt(j.Cancellable), boolInt(j.RollbackAvailable), string(b), lockKey)
	return err
}
func (s *Store) UpdateJob(ctx context.Context, j models.Job) error {
	var started, completed any
	if j.Status == "running" || j.Status == "validating" || j.Status == "testing" {
		started = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if terminal(j.Status) {
		completed = time.Now().UTC().Format(time.RFC3339Nano)
	}
	_, err := s.DB.Exec(`UPDATE jobs SET status=?,progress=?,current_step=?,summary=?,error=?,started_at=COALESCE(started_at,?),completed_at=COALESCE(?,completed_at),cancellable=?,rollback_available=? WHERE id=?`, j.Status, j.Progress, j.CurrentStep, j.Summary, nullIfEmpty(j.Error), started, completed, boolInt(j.Cancellable), boolInt(j.RollbackAvailable), j.ID)
	return err
}
func terminal(status string) bool {
	switch status {
	case "success", "partial", "failed", "rolled-back", "cancelled":
		return true
	}
	return false
}
func (s *Store) GetJob(ctx context.Context, id string) (models.Job, error) {
	return scanJob(s.DB.QueryRow(`SELECT id,requested_by,requested_at,operation,resource,status,progress,COALESCE(current_step,''),summary,COALESCE(error,''),correlation_id,cancellable,rollback_available FROM jobs WHERE id=?`, id))
}
func (s *Store) ListJobs(ctx context.Context) ([]models.Job, error) {
	rows, err := s.DB.Query(`SELECT id,requested_by,requested_at,operation,resource,status,progress,COALESCE(current_step,''),summary,COALESCE(error,''),correlation_id,cancellable,rollback_available FROM jobs ORDER BY requested_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Job{}
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scanJob(s scanner) (models.Job, error) {
	var j models.Job
	var ts string
	var c, r int
	err := s.Scan(&j.ID, &j.RequestedBy, &ts, &j.Operation, &j.Resource, &j.Status, &j.Progress, &j.CurrentStep, &j.Summary, &j.Error, &j.CorrelationID, &c, &r)
	if err != nil {
		return j, err
	}
	j.RequestedAt, _ = time.Parse(time.RFC3339Nano, ts)
	j.Cancellable = c == 1
	j.RollbackAvailable = r == 1
	return j, nil
}
func (s *Store) AcquireLock(ctx context.Context, key, jobID string, ttl time.Duration) error {
	now := time.Now().UTC()
	_, _ = s.DB.Exec(`DELETE FROM resource_locks WHERE expires_at < ?`, now.Format(time.RFC3339Nano))
	_, err := s.DB.Exec(`INSERT INTO resource_locks(resource_key,owner_job_id,acquired_at,expires_at) VALUES(?,?,?,?)`, key, jobID, now.Format(time.RFC3339Nano), now.Add(ttl).Format(time.RFC3339Nano))
	return err
}
func (s *Store) ReleaseLock(ctx context.Context, key, jobID string) {
	_, _ = s.DB.Exec(`DELETE FROM resource_locks WHERE resource_key=? AND owner_job_id=?`, key, jobID)
}
func (s *Store) PutEvent(ctx context.Context, eventType, resource string, payload any, correlation string) error {
	b, _ := json.Marshal(payload)
	_, err := s.DB.Exec(`INSERT INTO events(event_type,resource_id,payload_json,correlation_id,created_at) VALUES(?,?,?,?,?)`, eventType, resource, string(b), correlation, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) EnsureRole(ctx context.Context, id, name string) error {
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO roles(id,name) VALUES(?,?)`, id, name)
	return err
}
func (s *Store) UserCount(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}
func (s *Store) InsertUser(ctx context.Context, id, user, display, hash string) error {
	_, err := s.DB.Exec(`INSERT INTO users(id,username,display_name,password_hash,created_at) VALUES(?,?,?,?,?)`, id, user, display, hash, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`INSERT OR IGNORE INTO user_security(user_id) VALUES(?)`, id)
	return err
}
func (s *Store) AssignRole(ctx context.Context, userID, roleID string) error {
	_, err := s.DB.Exec(`INSERT OR IGNORE INTO user_roles(user_id,role_id) VALUES(?,?)`, userID, roleID)
	return err
}
func (s *Store) FindUser(ctx context.Context, username string) (UserRecord, error) {
	var u UserRecord
	var breakGlass, mfaRequired int
	err := s.DB.QueryRow(`SELECT u.id,u.username,u.display_name,u.password_hash,u.failed_attempts,u.locked_until,
		COALESCE(sec.is_break_glass, 0),COALESCE(sec.mfa_required, 0),sec.password_expires_at,sec.allowed_cidrs_json
		FROM users u LEFT JOIN user_security sec ON sec.user_id=u.id WHERE u.username=?`, username).Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.Failed, &u.LockedUntil, &breakGlass, &mfaRequired, &u.PasswordExpiresAt, &u.AllowedCIDRs,
	)
	u.BreakGlass = breakGlass == 1
	u.MFARequired = mfaRequired == 1
	return u, err
}

func (s *Store) SetBreakGlass(ctx context.Context, userID string, enabled bool) error {
	_, err := s.DB.Exec(`INSERT INTO user_security(user_id,is_break_glass) VALUES(?,?)
		ON CONFLICT(user_id) DO UPDATE SET is_break_glass=excluded.is_break_glass`, userID, boolInt(enabled))
	return err
}

func (s *Store) SetMFARequired(ctx context.Context, userID string, required bool) error {
	_, err := s.DB.Exec(`INSERT INTO user_security(user_id,mfa_required) VALUES(?,?)
		ON CONFLICT(user_id) DO UPDATE SET mfa_required=excluded.mfa_required`, userID, boolInt(required))
	return err
}

func (s *Store) SetPasswordExpiry(ctx context.Context, userID string, expiresAt *time.Time) error {
	var value any
	if expiresAt != nil {
		value = expiresAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.DB.Exec(`INSERT INTO user_security(user_id,password_expires_at) VALUES(?,?)
		ON CONFLICT(user_id) DO UPDATE SET password_expires_at=excluded.password_expires_at`, userID, value)
	return err
}

func (s *Store) CountBreakGlassUsers(ctx context.Context) (int, error) {
	var count int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM user_security WHERE is_break_glass=1`).Scan(&count)
	return count, err
}

func (s *Store) SavePendingTOTP(ctx context.Context, userID, pendingCiphertext string) error {
	_, err := s.DB.Exec(`INSERT INTO mfa_totp(user_id,secret_ciphertext,pending_ciphertext) VALUES(?,?,?)
		ON CONFLICT(user_id) DO UPDATE SET pending_ciphertext=excluded.pending_ciphertext`, userID, "", pendingCiphertext)
	return err
}

func (s *Store) PendingTOTP(ctx context.Context, userID string) (string, error) {
	var value sqlite.NullString
	err := s.DB.QueryRow(`SELECT pending_ciphertext FROM mfa_totp WHERE user_id=?`, userID).Scan(&value)
	if err != nil {
		return "", err
	}
	if !value.Valid || value.String == "" {
		return "", sqlite.ErrNoRows
	}
	return value.String, nil
}

func (s *Store) ActiveTOTP(ctx context.Context, userID string) (string, error) {
	var value sqlite.NullString
	err := s.DB.QueryRow(`SELECT secret_ciphertext FROM mfa_totp WHERE user_id=?`, userID).Scan(&value)
	if err != nil {
		return "", err
	}
	if !value.Valid || value.String == "" {
		return "", sqlite.ErrNoRows
	}
	return value.String, nil
}

func (s *Store) ConfirmPendingTOTP(ctx context.Context, userID string) error {
	result, err := s.DB.Exec(`UPDATE mfa_totp SET secret_ciphertext=pending_ciphertext,pending_ciphertext=NULL,
		enabled_at=?,rotated_at=? WHERE user_id=? AND pending_ciphertext IS NOT NULL AND pending_ciphertext<>''`,
		time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), userID)
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return sqlite.ErrNoRows
	}
	return nil
}

func (s *Store) RevokeTOTP(ctx context.Context, userID string) error {
	_, err := s.DB.Exec(`DELETE FROM mfa_totp WHERE user_id=?`, userID)
	return err
}

func (s *Store) CreateRecoveryCode(ctx context.Context, id, userID, hash string) error {
	_, err := s.DB.Exec(`INSERT INTO mfa_recovery_codes(id,user_id,code_hash,created_at) VALUES(?,?,?,?)`, id, userID, hash, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

type RecoveryCodeRecord struct {
	ID, UserID, CodeHash string
}

func (s *Store) UnusedRecoveryCodes(ctx context.Context, userID string) ([]RecoveryCodeRecord, error) {
	rows, err := s.DB.Query(`SELECT id,user_id,code_hash FROM mfa_recovery_codes WHERE user_id=? AND used_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var codes []RecoveryCodeRecord
	for rows.Next() {
		var code RecoveryCodeRecord
		if err := rows.Scan(&code.ID, &code.UserID, &code.CodeHash); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

func (s *Store) MarkRecoveryCodeUsed(ctx context.Context, id string) error {
	result, err := s.DB.Exec(`UPDATE mfa_recovery_codes SET used_at=? WHERE id=? AND used_at IS NULL`, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return sqlite.ErrNoRows
	}
	return nil
}

func (s *Store) DeleteRecoveryCodes(ctx context.Context, userID string) error {
	_, err := s.DB.Exec(`DELETE FROM mfa_recovery_codes WHERE user_id=?`, userID)
	return err
}

type MFAChallengeRecord struct {
	ID, UserID, CSRFToken, IP, UserAgent string
	ExpiresAt                            time.Time
}

func (s *Store) CreateMFAChallenge(ctx context.Context, c MFAChallengeRecord) error {
	_, err := s.DB.Exec(`DELETE FROM mfa_challenges WHERE expires_at<?`, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`INSERT INTO mfa_challenges(id,user_id,csrf_token,expires_at,ip,user_agent,created_at) VALUES(?,?,?,?,?,?,?)`,
		c.ID, c.UserID, c.CSRFToken, c.ExpiresAt.UTC().Format(time.RFC3339Nano), c.IP, c.UserAgent, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) ClaimMFAChallenge(ctx context.Context, id string) (MFAChallengeRecord, error) {
	var c MFAChallengeRecord
	var expires string
	err := s.DB.QueryRow(`SELECT id,user_id,csrf_token,expires_at,COALESCE(ip,''),COALESCE(user_agent,'') FROM mfa_challenges WHERE id=?`, id).Scan(
		&c.ID, &c.UserID, &c.CSRFToken, &expires, &c.IP, &c.UserAgent,
	)
	if err != nil {
		return c, err
	}
	c.ExpiresAt, err = time.Parse(time.RFC3339Nano, expires)
	if err != nil || !c.ExpiresAt.After(time.Now().UTC()) {
		_, _ = s.DB.Exec(`DELETE FROM mfa_challenges WHERE id=?`, id)
		return MFAChallengeRecord{}, sqlite.ErrNoRows
	}
	result, err := s.DB.Exec(`DELETE FROM mfa_challenges WHERE id=?`, id)
	if err != nil {
		return MFAChallengeRecord{}, err
	}
	if result.RowsAffected != 1 {
		return MFAChallengeRecord{}, sqlite.ErrNoRows
	}
	return c, nil
}
func (s *Store) UserRoles(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.DB.Query(`SELECT r.name FROM roles r JOIN user_roles ur ON ur.role_id=r.id WHERE ur.user_id=?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var x string
		if err := rows.Scan(&x); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Store) CreateSession(ctx context.Context, id, userID, csrf string, expires time.Time, ip, ua string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.DB.Exec(`INSERT INTO sessions(id,user_id,csrf_token,expires_at,created_at,last_seen_at,ip,user_agent) VALUES(?,?,?,?,?,?,?,?)`, id, userID, csrf, expires.UTC().Format(time.RFC3339Nano), now, now, ip, ua)
	return err
}
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE id=?`, id)
	return err
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE user_id=?`, userID)
	return err
}

// RotateSession creates a new opaque session before invalidating the old one,
// preventing session fixation while preserving the user's authenticated state.
func (s *Store) RotateSession(ctx context.Context, oldID, newID, csrf string, expires time.Time, ip, ua string) error {
	var userID string
	var oldExpires string
	if err := s.DB.QueryRow(`SELECT user_id,expires_at FROM sessions WHERE id=?`, oldID).Scan(&userID, &oldExpires); err != nil {
		return err
	}
	oldExpiry, err := time.Parse(time.RFC3339Nano, oldExpires)
	if err != nil || !oldExpiry.After(time.Now().UTC()) {
		return errors.New("sessão expirada")
	}
	if err := s.CreateSession(ctx, newID, userID, csrf, expires, ip, ua); err != nil {
		return err
	}
	if err := s.DeleteSession(ctx, oldID); err != nil {
		_ = s.DeleteSession(ctx, newID)
		return err
	}
	return nil
}
func (s *Store) SessionUser(ctx context.Context, id string) (models.User, string, error) {
	var u models.User
	var expiry, csrf string
	var breakGlass, mfaEnabled int
	err := s.DB.QueryRow(`SELECT u.id,u.username,u.display_name,s.expires_at,s.csrf_token,
		COALESCE(sec.is_break_glass,0),CASE WHEN totp.secret_ciphertext IS NOT NULL AND totp.secret_ciphertext<>'' THEN 1 ELSE 0 END
		FROM sessions s JOIN users u ON u.id=s.user_id
		LEFT JOIN user_security sec ON sec.user_id=u.id
		LEFT JOIN mfa_totp totp ON totp.user_id=u.id WHERE s.id=?`, id).Scan(&u.ID, &u.Username, &u.DisplayName, &expiry, &csrf, &breakGlass, &mfaEnabled)
	if err != nil {
		return u, "", err
	}
	u.BreakGlass = breakGlass == 1
	u.MFAEnabled = mfaEnabled == 1
	t, _ := time.Parse(time.RFC3339Nano, expiry)
	if time.Now().After(t) {
		return u, "", errors.New("sessão expirada")
	}
	roles, err := s.UserRoles(ctx, u.ID)
	u.Roles = roles
	return u, csrf, err
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

type IdempotencyRecord struct {
	Key            string
	Actor          string
	RequestHash    string
	ResponseStatus int
	ResponseBody   string
	ExpiresAt      time.Time
}

func (s *Store) RegisterLoginFailure(ctx context.Context, userID string, threshold int, lockFor time.Duration) (bool, time.Time, error) {
	if threshold < 1 {
		threshold = 5
	}
	lockedUntil := time.Now().UTC().Add(lockFor)
	_, err := s.DB.Exec(`UPDATE users
		SET failed_attempts = failed_attempts + 1,
		    locked_until = CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE locked_until END
		WHERE id = ?`, threshold, lockedUntil.Format(time.RFC3339Nano), userID)
	if err != nil {
		return false, time.Time{}, err
	}
	var attempts int
	var stored sqlite.NullString
	if err = s.DB.QueryRow(`SELECT failed_attempts, locked_until FROM users WHERE id=?`, userID).Scan(&attempts, &stored); err != nil {
		return false, time.Time{}, err
	}
	if stored.Valid {
		t, parseErr := time.Parse(time.RFC3339Nano, stored.String)
		if parseErr == nil && t.After(time.Now().UTC()) {
			return true, t, nil
		}
	}
	return attempts >= threshold, lockedUntil, nil
}

func (s *Store) ResetLoginFailures(ctx context.Context, userID string) error {
	_, err := s.DB.Exec(`UPDATE users SET failed_attempts=0, locked_until=NULL WHERE id=?`, userID)
	return err
}

func (s *Store) GetIdempotency(ctx context.Context, key, actor string) (IdempotencyRecord, error) {
	var rec IdempotencyRecord
	var expires string
	err := s.DB.QueryRow(`SELECT key,actor,request_hash,response_status,response_body,expires_at
		FROM idempotency_keys WHERE key=? AND actor=? AND expires_at>?`, key, actor, time.Now().UTC().Format(time.RFC3339Nano)).Scan(
		&rec.Key, &rec.Actor, &rec.RequestHash, &rec.ResponseStatus, &rec.ResponseBody, &expires,
	)
	if err != nil {
		return rec, err
	}
	rec.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
	return rec, nil
}

func (s *Store) PutIdempotency(ctx context.Context, rec IdempotencyRecord) error {
	_, err := s.DB.Exec(`INSERT INTO idempotency_keys(key,actor,request_hash,response_status,response_body,created_at,expires_at)
		VALUES(?,?,?,?,?,?,?)`, rec.Key, rec.Actor, rec.RequestHash, rec.ResponseStatus, rec.ResponseBody,
		time.Now().UTC().Format(time.RFC3339Nano), rec.ExpiresAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) ReserveIdempotency(ctx context.Context, rec IdempotencyRecord) error {
	now := time.Now().UTC()
	if _, err := s.DB.Exec(`DELETE FROM idempotency_keys WHERE key=? AND actor=? AND expires_at<=?`, rec.Key, rec.Actor, now.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	_, err := s.DB.Exec(`INSERT INTO idempotency_keys(key,actor,request_hash,response_status,response_body,created_at,expires_at)
		VALUES(?,?,?,0,'',?,?)`, rec.Key, rec.Actor, rec.RequestHash, now.Format(time.RFC3339Nano), rec.ExpiresAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) CompleteIdempotency(ctx context.Context, key, actor, requestHash string, status int, body string) error {
	result, err := s.DB.Exec(`UPDATE idempotency_keys SET response_status=?,response_body=?
		WHERE key=? AND actor=? AND request_hash=? AND response_status=0`, status, body, key, actor, requestHash)
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("reserva de idempotência ausente ou já concluída")
	}
	return nil
}

func (s *Store) DeletePendingIdempotency(ctx context.Context, key, actor, requestHash string) error {
	_, err := s.DB.Exec(`DELETE FROM idempotency_keys WHERE key=? AND actor=? AND request_hash=? AND response_status=0`, key, actor, requestHash)
	return err
}

// ChangeRequestRecord stores the immutable request metadata and the later
// approval decision. PayloadJSON must never contain credentials or TOTP data.
type ChangeRequestRecord struct {
	ID, Operation, Resource, Risk, Status, RequestedBy, Justification string
	RequestedAt                                                       time.Time
	MaintenanceStart, MaintenanceEnd                                  *time.Time
	Impact, RollbackPlan, PayloadJSON, RequiredCapability, JobID      string
	ApprovedBy                                                        string
	ApprovedAt                                                        *time.Time
	DecisionReason                                                    string
	ExpiresAt                                                         *time.Time
}

func (s *Store) CreateChangeRequest(ctx context.Context, r ChangeRequestRecord) error {
	_, err := s.DB.Exec(`INSERT INTO change_requests(
		id,operation,resource,risk,status,requested_by,requested_at,justification,maintenance_start,maintenance_end,
		impact,rollback_plan,payload_json,required_capability,job_id,approved_by,approved_at,decision_reason,expires_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.Operation, r.Resource, r.Risk, r.Status, r.RequestedBy, r.RequestedAt.UTC().Format(time.RFC3339Nano), r.Justification,
		timeValue(r.MaintenanceStart), timeValue(r.MaintenanceEnd), r.Impact, r.RollbackPlan, r.PayloadJSON, nullIfEmpty(r.RequiredCapability),
		nullIfEmpty(r.JobID), nullIfEmpty(r.ApprovedBy), timeValue(r.ApprovedAt), nullIfEmpty(r.DecisionReason), timeValue(r.ExpiresAt),
	)
	return err
}

func (s *Store) GetChangeRequest(ctx context.Context, id string) (ChangeRequestRecord, error) {
	return scanChangeRequest(s.DB.QueryRow(`SELECT id,operation,resource,risk,status,requested_by,requested_at,justification,
		maintenance_start,maintenance_end,impact,rollback_plan,payload_json,COALESCE(required_capability,''),COALESCE(job_id,''),
		COALESCE(approved_by,''),approved_at,COALESCE(decision_reason,''),expires_at FROM change_requests WHERE id=?`, id))
}

func (s *Store) ListChangeRequests(ctx context.Context, status string) ([]ChangeRequestRecord, error) {
	query := `SELECT id,operation,resource,risk,status,requested_by,requested_at,justification,
		maintenance_start,maintenance_end,impact,rollback_plan,payload_json,COALESCE(required_capability,''),COALESCE(job_id,''),
		COALESCE(approved_by,''),approved_at,COALESCE(decision_reason,''),expires_at FROM change_requests`
	args := []any{}
	if status != "" {
		query += ` WHERE status=?`
		args = append(args, status)
	}
	query += ` ORDER BY requested_at DESC LIMIT 200`
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	requests := []ChangeRequestRecord{}
	for rows.Next() {
		r, err := scanChangeRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}
	return requests, rows.Err()
}

func (s *Store) DecideChangeRequest(ctx context.Context, id, approverID, status, reason string) (ChangeRequestRecord, error) {
	r, err := s.GetChangeRequest(ctx, id)
	if err != nil {
		return r, err
	}
	if r.Status != "pending_approval" {
		return r, fmt.Errorf("solicitação não está pendente de aprovação")
	}
	if r.ExpiresAt != nil && !r.ExpiresAt.After(time.Now().UTC()) {
		_, _ = s.DB.Exec(`UPDATE change_requests SET status='expired' WHERE id=? AND status='pending_approval'`, id)
		return r, fmt.Errorf("solicitação expirada")
	}
	if status != "approved" && status != "rejected" {
		return r, fmt.Errorf("decisão inválida")
	}
	result, err := s.DB.Exec(`UPDATE change_requests SET status=?,approved_by=?,approved_at=?,decision_reason=?
		WHERE id=? AND status='pending_approval'`, status, approverID, time.Now().UTC().Format(time.RFC3339Nano), nullIfEmpty(reason), id)
	if err != nil {
		return r, err
	}
	if result.RowsAffected != 1 {
		return r, fmt.Errorf("solicitação alterada concorrentemente")
	}
	return s.GetChangeRequest(ctx, id)
}

func (s *Store) SetChangeRequestJob(ctx context.Context, id, jobID string) error {
	result, err := s.DB.Exec(`UPDATE change_requests SET status='executing',job_id=? WHERE id=? AND status='approved'`, jobID, id)
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("solicitação não aprovada para execução")
	}
	return nil
}

func (s *Store) ResetChangeRequestJob(ctx context.Context, id, jobID string) error {
	result, err := s.DB.Exec(`UPDATE change_requests SET status='approved',job_id=NULL WHERE id=? AND status='executing' AND job_id=?`, id, jobID)
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("reserva de execução não pôde ser liberada")
	}
	return nil
}

func (s *Store) CompleteChangeRequest(ctx context.Context, id, status string) error {
	if status != "completed" && status != "failed" && status != "cancelled" && status != "rolled_back" {
		return fmt.Errorf("estado final inválido")
	}
	result, err := s.DB.Exec(`UPDATE change_requests SET status=? WHERE id=? AND status='executing'`, status, id)
	if err != nil {
		return err
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("solicitação não está em execução")
	}
	return nil
}

func timeValue(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.UTC().Format(time.RFC3339Nano)
}

func scanChangeRequest(s scanner) (ChangeRequestRecord, error) {
	var r ChangeRequestRecord
	var requested, maintenanceStart, maintenanceEnd, approvedAt, expires sqlite.NullString
	err := s.Scan(&r.ID, &r.Operation, &r.Resource, &r.Risk, &r.Status, &r.RequestedBy, &requested, &r.Justification,
		&maintenanceStart, &maintenanceEnd, &r.Impact, &r.RollbackPlan, &r.PayloadJSON, &r.RequiredCapability, &r.JobID,
		&r.ApprovedBy, &approvedAt, &r.DecisionReason, &expires)
	if err != nil {
		return r, err
	}
	r.RequestedAt, _ = time.Parse(time.RFC3339Nano, requested.String)
	r.MaintenanceStart = parseOptionalTime(maintenanceStart)
	r.MaintenanceEnd = parseOptionalTime(maintenanceEnd)
	r.ApprovedAt = parseOptionalTime(approvedAt)
	r.ExpiresAt = parseOptionalTime(expires)
	return r, nil
}

func parseOptionalTime(v sqlite.NullString) *time.Time {
	if !v.Valid || v.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, v.String)
	if err != nil {
		return nil
	}
	return &t
}
