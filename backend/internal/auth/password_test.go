package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

func TestArgon2idHashAndVerify(t *testing.T) {
	h, err := HashPassword("Senha-de-Teste-Muito-Forte-2026!")
	if err != nil {
		t.Fatal(err)
	}
	if len(h) < 20 || h[:9] != "$argon2id" {
		t.Fatalf("hash inesperado: %s", h)
	}
	ok, err := VerifyPassword(h, "Senha-de-Teste-Muito-Forte-2026!")
	if err != nil || !ok {
		t.Fatalf("senha válida recusada: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword(h, "senha-incorreta")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("senha incorreta aceita")
	}
}

func TestLocalLoginLockout(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	svc := New(store, time.Hour)
	ctx := context.Background()
	if err = svc.Bootstrap(ctx, "admin", "Senha-Correta-Muito-Forte-2026!"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < loginFailureThreshold-1; i++ {
		_, _, _, _, err = svc.Login(ctx, "admin", "incorreta", "127.0.0.1", "teste")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("tentativa %d deveria retornar credenciais inválidas: %v", i+1, err)
		}
	}
	_, _, _, _, err = svc.Login(ctx, "admin", "incorreta", "127.0.0.1", "teste")
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("quinta tentativa deveria bloquear a conta: %v", err)
	}
	_, _, _, _, err = svc.Login(ctx, "admin", "Senha-Correta-Muito-Forte-2026!", "127.0.0.1", "teste")
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("conta bloqueada não deveria aceitar login imediato: %v", err)
	}
}
