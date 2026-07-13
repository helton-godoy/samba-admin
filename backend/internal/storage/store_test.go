package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/models"
)

func TestSQLitePersistenceAndWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var mode string
	if err = s.DB.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Fatalf("modo esperado wal, obtido %s", mode)
	}
	j := models.Job{ID: "JOB-1", RequestedBy: "tester", RequestedAt: time.Now().UTC(), Operation: "Teste", Resource: "share:x", Status: "queued", Summary: "fila", CorrelationID: "corr-1", Cancellable: true, RollbackAvailable: true}
	if err = s.InsertJob(context.Background(), j, map[string]any{"x": 1}, "share:x"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.GetJob(context.Background(), "JOB-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Operation != "Teste" {
		t.Fatalf("persistência falhou: %+v", got)
	}
}

func TestIdempotencyIsScopedByActorAndReservedBeforeCompletion(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "idempotency.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	expires := time.Now().UTC().Add(time.Hour)
	for _, actor := range []string{"actor-a", "actor-b"} {
		if err = store.ReserveIdempotency(ctx, IdempotencyRecord{Key: "same-key", Actor: actor, RequestHash: "hash", ExpiresAt: expires}); err != nil {
			t.Fatalf("chave deveria ser independente por ator %s: %v", actor, err)
		}
	}
	if err = store.ReserveIdempotency(ctx, IdempotencyRecord{Key: "same-key", Actor: "actor-a", RequestHash: "hash", ExpiresAt: expires}); err == nil {
		t.Fatal("reserva concorrente do mesmo ator foi aceita")
	}
	if err = store.CompleteIdempotency(ctx, "same-key", "actor-a", "hash", 201, `{"ok":true}`); err != nil {
		t.Fatal(err)
	}
	record, err := store.GetIdempotency(ctx, "same-key", "actor-a")
	if err != nil || record.ResponseStatus != 201 {
		t.Fatalf("resultado idempotente não persistido: %+v, %v", record, err)
	}
}
func TestResourceLockConflict(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "lock.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if err = s.AcquireLock(ctx, "share:x", "job-1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err = s.AcquireLock(ctx, "share:x", "job-2", time.Minute); err == nil {
		t.Fatal("lock concorrente deveria falhar")
	}
}
