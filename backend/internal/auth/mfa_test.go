package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

func TestTOTPLoginAndSingleUseChallenge(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "mfa.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewWithOptions(store, time.Hour, Options{TOTPEncryptionKey: []byte("0123456789abcdef0123456789abcdef")})
	ctx := context.Background()
	if err = service.Bootstrap(ctx, "admin", "Senha-Correta-Muito-Forte-2026!"); err != nil {
		t.Fatal(err)
	}
	user, err := store.FindUser(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	enrollment, err := service.BeginTOTPEnrollment(ctx, user.ID, "Samba Admin Suite", "admin")
	if err != nil {
		t.Fatal(err)
	}
	code, err := totpCode(enrollment.Secret, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	codes, err := service.ConfirmTOTPEnrollment(ctx, user.ID, code)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != recoveryCodeCount {
		t.Fatalf("esperado %d recovery codes, obtido %d", recoveryCodeCount, len(codes))
	}
	if err = store.SetMFARequired(ctx, user.ID, true); err != nil {
		t.Fatal(err)
	}
	begin, err := service.BeginLogin(ctx, "admin", "Senha-Correta-Muito-Forte-2026!", "127.0.0.1", "teste")
	if err != nil {
		t.Fatal(err)
	}
	if !begin.MFARequired || begin.MFAChallengeID == "" || begin.Session != "" {
		t.Fatalf("desafio MFA inesperado: %+v", begin)
	}
	result, err := service.CompleteMFA(ctx, begin.MFAChallengeID, code, "127.0.0.1", "teste")
	if err != nil || result.Session == "" || !result.User.MFAEnabled {
		t.Fatalf("MFA não concluiu: result=%+v err=%v", result, err)
	}
	if _, err = service.CompleteMFA(ctx, begin.MFAChallengeID, code, "127.0.0.1", "teste"); !errors.Is(err, ErrMFAInvalid) {
		t.Fatalf("desafio reutilizado deveria falhar: %v", err)
	}
}

func TestRecoveryCodeIsNormalizedAndConsumed(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "recovery.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewWithOptions(store, time.Hour, Options{TOTPEncryptionKey: []byte("0123456789abcdef0123456789abcdef")})
	ctx := context.Background()
	if err = service.Bootstrap(ctx, "admin", "Senha-Correta-Muito-Forte-2026!"); err != nil {
		t.Fatal(err)
	}
	user, _ := store.FindUser(ctx, "admin")
	enrollment, err := service.BeginTOTPEnrollment(ctx, user.ID, "Samba Admin Suite", "admin")
	if err != nil {
		t.Fatal(err)
	}
	code, _ := totpCode(enrollment.Secret, time.Now().UTC())
	codes, err := service.ConfirmTOTPEnrollment(ctx, user.ID, code)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.SetMFARequired(ctx, user.ID, true); err != nil {
		t.Fatal(err)
	}
	begin, err := service.BeginLogin(ctx, "admin", "Senha-Correta-Muito-Forte-2026!", "127.0.0.1", "teste")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.CompleteMFA(ctx, begin.MFAChallengeID, " "+codes[0]+" ", "127.0.0.1", "teste"); err != nil {
		t.Fatalf("recovery code válido recusado: %v", err)
	}
	begin, err = service.BeginLogin(ctx, "admin", "Senha-Correta-Muito-Forte-2026!", "127.0.0.1", "teste")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.CompleteMFA(ctx, begin.MFAChallengeID, codes[0], "127.0.0.1", "teste"); !errors.Is(err, ErrMFAInvalid) {
		t.Fatalf("recovery code reutilizado deveria falhar: %v", err)
	}
}

func TestMFASecurityChangesInvalidateExistingSessions(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "mfa-sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := NewWithOptions(store, time.Hour, Options{TOTPEncryptionKey: []byte("0123456789abcdef0123456789abcdef")})
	ctx := context.Background()
	if err = service.Bootstrap(ctx, "admin", "Senha-Correta-Muito-Forte-2026!"); err != nil {
		t.Fatal(err)
	}
	user, err := store.FindUser(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}

	initial, err := service.BeginLogin(ctx, "admin", "Senha-Correta-Muito-Forte-2026!", "127.0.0.1", "teste")
	if err != nil {
		t.Fatal(err)
	}
	enrollment, err := service.BeginTOTPEnrollment(ctx, user.ID, "Samba Admin Suite", "admin")
	if err != nil {
		t.Fatal(err)
	}
	code, err := totpCode(enrollment.Secret, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.ConfirmTOTPEnrollment(ctx, user.ID, code); err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.Current(ctx, initial.Session); err == nil {
		t.Fatal("a confirmação do TOTP deveria invalidar sessões existentes")
	}

	if err = store.SetMFARequired(ctx, user.ID, true); err != nil {
		t.Fatal(err)
	}
	begin, err := service.BeginLogin(ctx, "admin", "Senha-Correta-Muito-Forte-2026!", "127.0.0.1", "teste")
	if err != nil {
		t.Fatal(err)
	}
	authenticated, err := service.CompleteMFA(ctx, begin.MFAChallengeID, code, "127.0.0.1", "teste")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.RotateRecoveryCodes(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.Current(ctx, authenticated.Session); err == nil {
		t.Fatal("a rotação dos códigos de recuperação deveria invalidar sessões existentes")
	}

	begin, err = service.BeginLogin(ctx, "admin", "Senha-Correta-Muito-Forte-2026!", "127.0.0.1", "teste")
	if err != nil {
		t.Fatal(err)
	}
	authenticated, err = service.CompleteMFA(ctx, begin.MFAChallengeID, code, "127.0.0.1", "teste")
	if err != nil {
		t.Fatal(err)
	}
	if err = service.RevokeTOTP(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.Current(ctx, authenticated.Session); err == nil {
		t.Fatal("a revogação do TOTP deveria invalidar sessões existentes")
	}
}
