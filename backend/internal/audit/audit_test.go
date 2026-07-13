package audit

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

func TestAuditHashChain(t *testing.T) {
	s, err := storage.Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := New(s)
	ctx := context.Background()
	if err = a.Record(ctx, Entry{Actor: "a", Operation: "share.create", Resource: "share:x", Result: "success", CorrelationID: "c1", After: map[string]any{"name": "x"}}); err != nil {
		t.Fatal(err)
	}
	if err = a.Record(ctx, Entry{Actor: "b", Operation: "share.update", Resource: "share:x", Result: "success", CorrelationID: "c2", After: map[string]any{"name": "y"}}); err != nil {
		t.Fatal(err)
	}
	rows, err := s.DB.Query(`SELECT hash,COALESCE(previous_hash,'') FROM audit_log ORDER BY timestamp`)
	if err != nil {
		t.Fatal(err)
	}
	var hashes, prevs []string
	for rows.Next() {
		var h, p string
		if err = rows.Scan(&h, &p); err != nil {
			t.Fatal(err)
		}
		hashes = append(hashes, h)
		prevs = append(prevs, p)
	}
	if err = rows.Close(); err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 2 || prevs[1] != hashes[0] {
		t.Fatalf("cadeia inválida hashes=%v prevs=%v", hashes, prevs)
	}
	if err = a.Verify(ctx); err != nil {
		t.Fatalf("verificação da cadeia falhou: %v", err)
	}
}

func TestConcurrentAuditRemainsLinearAndRedactsSecrets(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "audit-concurrent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := New(store)
	var group sync.WaitGroup
	for index := 0; index < 32; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if recordErr := service.Record(context.Background(), Entry{Actor: "operador", Operation: "test.concurrent", Resource: "resource", Result: "success", After: map[string]any{"credential": "nao-registrar", "safe": "ok"}}); recordErr != nil {
				t.Errorf("registro concorrente falhou: %v", recordErr)
			}
		}()
	}
	group.Wait()
	if err = service.Verify(context.Background()); err != nil {
		t.Fatalf("cadeia concorrente inválida: %v", err)
	}
	var after string
	if err = store.DB.QueryRow(`SELECT after_json FROM audit_log LIMIT 1`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(after, "nao-registrar") || !strings.Contains(after, "[REDACTED]") {
		t.Fatalf("segredo persistido na auditoria: %s", after)
	}
}

func TestAuditTamperingIsDetected(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "audit-tamper.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := New(store)
	if err = service.Record(context.Background(), Entry{Actor: "a", Operation: "read", Resource: "x", Result: "success"}); err != nil {
		t.Fatal(err)
	}
	if _, err = store.DB.Exec(`UPDATE audit_log SET operation='tampered'`); err != nil {
		t.Fatal(err)
	}
	if err = service.Verify(context.Background()); err == nil {
		t.Fatal("adulteração da cadeia não foi detectada")
	}
}
