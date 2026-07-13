// Command samba-admin-e2e-mfa-seed prepara exclusivamente o banco temporário
// da suíte integrada. Ele não é compilado nem instalado pelo package FreeBSD.
package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/auth"
	"github.com/hu-ufcat/samba-admin-backend/internal/config"
	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

func main() {
	if os.Getenv("SAMBA_ADMIN_E2E_SEED_CONFIRM") != "sim" {
		log.Fatal("seed MFA permitido somente com confirmação explícita da suíte E2E")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if len(cfg.TOTPEncryptionKey) != 32 {
		log.Fatal("chave de proteção TOTP inválida para a suíte E2E")
	}
	secret := strings.TrimSpace(os.Getenv("SAMBA_ADMIN_E2E_TOTP_SECRET"))
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil || len(decoded) < 16 {
		log.Fatal("segredo TOTP E2E inválido")
	}

	store, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	service := auth.NewWithOptions(store, time.Hour, auth.Options{TOTPEncryptionKey: cfg.TOTPEncryptionKey})
	ctx := context.Background()
	if err = service.Bootstrap(ctx, cfg.BootstrapUser, cfg.BootstrapPassword); err != nil {
		log.Fatal(err)
	}
	user, err := store.FindUser(ctx, cfg.BootstrapUser)
	if err != nil {
		log.Fatal(err)
	}
	ciphertext, err := encrypt(cfg.TOTPEncryptionKey, secret)
	if err != nil {
		log.Fatal(err)
	}
	if err = store.SavePendingTOTP(ctx, user.ID, ciphertext); err != nil {
		log.Fatal(err)
	}
	if err = store.ConfirmPendingTOTP(ctx, user.ID); err != nil {
		log.Fatal(err)
	}
	if err = store.SetMFARequired(ctx, user.ID, true); err != nil {
		log.Fatal(err)
	}
}

func encrypt(key []byte, value string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	if value == "" {
		return "", errors.New("segredo TOTP vazio")
	}
	sealed := gcm.Seal(nil, nonce, []byte(value), nil)
	return base64.RawURLEncoding.EncodeToString(append(nonce, sealed...)), nil
}
