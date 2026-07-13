package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/id"
	"github.com/hu-ufcat/samba-admin-backend/internal/sqlite"
	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

type Entry struct {
	Actor                                                                 string
	Roles                                                                 []string
	Source, IP, Operation, Resource, Result, CorrelationID, JobID, Reason string
	Before, After                                                         any
}
type Service struct {
	store *storage.Store
	mu    sync.Mutex
}

func New(s *storage.Store) *Service { return &Service{store: s} }
func (s *Service) Record(ctx context.Context, e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	roles, err := json.Marshal(e.Roles)
	if err != nil {
		return fmt.Errorf("serializar papéis de auditoria: %w", err)
	}
	before, err := json.Marshal(redactAuditValue(e.Before))
	if err != nil {
		return fmt.Errorf("serializar estado anterior de auditoria: %w", err)
	}
	after, err := json.Marshal(redactAuditValue(e.After))
	if err != nil {
		return fmt.Errorf("serializar estado posterior de auditoria: %w", err)
	}
	var prev sqlite.NullString
	_ = s.store.DB.QueryRow(`SELECT hash FROM audit_log ORDER BY rowid DESC LIMIT 1`).Scan(&prev)
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	raw := ts + e.Actor + e.Operation + e.Resource + e.Result + e.CorrelationID + string(before) + string(after) + prev.String
	sum := sha256.Sum256([]byte(raw))
	_, err = s.store.DB.Exec(`INSERT INTO audit_log(id,timestamp,actor,roles_json,source,ip,operation,resource,before_json,after_json,result,correlation_id,job_id,reason,hash,previous_hash) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id.New("audit-"), ts, e.Actor, string(roles), e.Source, e.IP, e.Operation, e.Resource, nullJSON(before), nullJSON(after), e.Result, e.CorrelationID, nullString(e.JobID), nullString(e.Reason), hex.EncodeToString(sum[:]), nullString(prev.String))
	return err
}

func (s *Service) Verify(ctx context.Context) error {
	rows, err := s.store.DB.Query(`SELECT timestamp,actor,operation,resource,result,correlation_id,
		COALESCE(before_json,''),COALESCE(after_json,''),hash,COALESCE(previous_hash,'') FROM audit_log ORDER BY rowid`)
	if err != nil {
		return err
	}
	defer rows.Close()
	previous := ""
	for rows.Next() {
		var timestamp, actor, operation, resource, result, correlation, before, after, hash, linked string
		if err = rows.Scan(&timestamp, &actor, &operation, &resource, &result, &correlation, &before, &after, &hash, &linked); err != nil {
			return err
		}
		if linked != previous {
			return errors.New("encadeamento de auditoria divergente")
		}
		raw := timestamp + actor + operation + resource + result + correlation + jsonOrNull(before) + jsonOrNull(after) + linked
		sum := sha256.Sum256([]byte(raw))
		if !strings.EqualFold(hash, hex.EncodeToString(sum[:])) {
			return errors.New("hash de auditoria divergente")
		}
		previous = hash
	}
	return rows.Err()
}

func redactAuditValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			if auditSensitiveKey(key) {
				result[key] = "[REDACTED]"
			} else {
				result[key] = redactAuditValue(child)
			}
		}
		return result
	case []any:
		result := make([]any, len(typed))
		for index, child := range typed {
			result[index] = redactAuditValue(child)
		}
		return result
	default:
		return value
	}
}

func auditSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(key))
	for _, marker := range []string{"password", "passwd", "secret", "token", "credential", "authorization", "cookie", "keytab", "privatekey", "totp", "otp", "recoverycode"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func jsonOrNull(value string) string {
	if value == "" {
		return "null"
	}
	return value
}
func nullJSON(v []byte) any {
	if string(v) == "null" {
		return nil
	}
	return string(v)
}
func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
