package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // TOTP is standardized on HMAC-SHA-1 by RFC 6238.
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/id"
	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

const (
	totpPeriod        = 30 * time.Second
	totpDigits        = 6
	recoveryCodeCount = 10
)

var (
	ErrMFAUnavailable        = errors.New("MFA indisponível: chave de proteção não configurada")
	ErrMFAEnrollmentRequired = errors.New("MFA obrigatório, mas ainda não configurado")
	ErrMFAInvalid            = errors.New("código MFA inválido")
)

type MFAEnrollment struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

type encryption struct{ key []byte }

func newEncryption(key []byte) (*encryption, error) {
	if len(key) == 0 {
		return nil, nil
	}
	if len(key) != 32 {
		return nil, errors.New("a chave de criptografia TOTP deve possuir 32 bytes")
	}
	return &encryption{key: append([]byte(nil), key...)}, nil
}

func (e *encryption) encrypt(plain string) (string, error) {
	if e == nil {
		return "", ErrMFAUnavailable
	}
	block, err := aes.NewCipher(e.key)
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
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	return base64.RawURLEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func (e *encryption) decrypt(value string) (string, error) {
	if e == nil {
		return "", ErrMFAUnavailable
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", errors.New("segredo TOTP armazenado inválido")
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("segredo TOTP armazenado inválido")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("segredo TOTP não pôde ser decifrado")
	}
	return string(plain), nil
}

func (s *Service) BeginTOTPEnrollment(ctx context.Context, userID, issuer, account string) (MFAEnrollment, error) {
	if s.crypto == nil {
		return MFAEnrollment{}, ErrMFAUnavailable
	}
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(issuer) == "" || strings.TrimSpace(account) == "" {
		return MFAEnrollment{}, errors.New("usuário, emissor e conta são obrigatórios")
	}
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		return MFAEnrollment{}, err
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	ciphertext, err := s.crypto.encrypt(secret)
	if err != nil {
		return MFAEnrollment{}, err
	}
	if err = s.store.SavePendingTOTP(ctx, userID, ciphertext); err != nil {
		return MFAEnrollment{}, err
	}
	issuer = strings.ReplaceAll(issuer, ":", "")
	account = strings.ReplaceAll(account, ":", "")
	uri := "otpauth://totp/" + urlEscape(issuer+":"+account) + "?secret=" + secret + "&issuer=" + urlEscape(issuer) + "&period=30&digits=6"
	return MFAEnrollment{Secret: secret, URI: uri}, nil
}

func (s *Service) ConfirmTOTPEnrollment(ctx context.Context, userID, code string) ([]string, error) {
	if s.crypto == nil {
		return nil, ErrMFAUnavailable
	}
	pending, err := s.store.PendingTOTP(ctx, userID)
	if err != nil {
		return nil, ErrMFAInvalid
	}
	secret, err := s.crypto.decrypt(pending)
	if err != nil || !validTOTP(secret, code, time.Now().UTC()) {
		return nil, ErrMFAInvalid
	}
	if err = s.store.ConfirmPendingTOTP(ctx, userID); err != nil {
		return nil, err
	}
	return s.RotateRecoveryCodes(ctx, userID)
}

func (s *Service) RotateRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	codes := make([]string, 0, recoveryCodeCount)
	type pendingCode struct{ id, hash string }
	pending := make([]pendingCode, 0, recoveryCodeCount)
	for i := 0; i < recoveryCodeCount; i++ {
		code, err := randomRecoveryCode()
		if err != nil {
			return nil, err
		}
		hash, err := HashPassword(normalizeRecoveryCode(code))
		if err != nil {
			return nil, err
		}
		pending = append(pending, pendingCode{id: id.New("recovery-"), hash: hash})
		codes = append(codes, code)
	}
	if err := s.store.DeleteRecoveryCodes(ctx, userID); err != nil {
		return nil, err
	}
	for _, code := range pending {
		if err := s.store.CreateRecoveryCode(ctx, code.id, userID, code.hash); err != nil {
			return nil, err
		}
	}
	if err := s.store.DeleteUserSessions(ctx, userID); err != nil {
		return nil, err
	}
	return codes, nil
}

func (s *Service) RevokeTOTP(ctx context.Context, userID string) error {
	if err := s.store.RevokeTOTP(ctx, userID); err != nil {
		return err
	}
	if err := s.store.DeleteRecoveryCodes(ctx, userID); err != nil {
		return err
	}
	return s.store.DeleteUserSessions(ctx, userID)
}

func (s *Service) verifyMFA(ctx context.Context, userID, code string) error {
	if s.crypto == nil {
		return ErrMFAUnavailable
	}
	if ciphertext, err := s.store.ActiveTOTP(ctx, userID); err == nil {
		secret, decryptErr := s.crypto.decrypt(ciphertext)
		if decryptErr == nil && validTOTP(secret, code, time.Now().UTC()) {
			return nil
		}
	}
	// Recovery codes are deliberately checked after TOTP and are consumed once.
	for _, recovery := range mustUnusedRecoveryCodes(ctx, s.store, userID) {
		valid, err := VerifyPassword(recovery.CodeHash, normalizeRecoveryCode(code))
		if err == nil && valid {
			if err = s.store.MarkRecoveryCodeUsed(ctx, recovery.ID); err == nil {
				return nil
			}
		}
	}
	return ErrMFAInvalid
}

func mustUnusedRecoveryCodes(ctx context.Context, store *storage.Store, userID string) []storage.RecoveryCodeRecord {
	codes, err := store.UnusedRecoveryCodes(ctx, userID)
	if err != nil {
		return nil
	}
	return codes
}

func validTOTP(secret, code string, now time.Time) bool {
	if len(code) != totpDigits {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	for offset := int64(-1); offset <= 1; offset++ {
		expected, err := totpCode(secret, now.Add(time.Duration(offset)*totpPeriod))
		if err == nil && subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

func totpCode(secret string, at time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil || len(key) < 16 {
		return "", errors.New("segredo TOTP inválido")
	}
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(at.UTC().Unix()/int64(totpPeriod/time.Second)))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(counter[:])
	sum := mac.Sum(nil)
	offset := int(sum[len(sum)-1] & 0x0f)
	value := (int(sum[offset])&0x7f)<<24 | int(sum[offset+1])<<16 | int(sum[offset+2])<<8 | int(sum[offset+3])
	return fmt.Sprintf("%06d", value%1_000_000), nil
}

func randomRecoveryCode() (string, error) {
	var raw [10]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	encoded := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw[:]))
	return encoded[:5] + "-" + encoded[5:10] + "-" + encoded[10:15] + "-" + encoded[15:], nil
}

func normalizeRecoveryCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
}

func allowedByCIDRs(serialized, ip string) bool {
	if strings.TrimSpace(serialized) == "" {
		return true
	}
	value := strings.Trim(serialized, "[] ")
	if value == "" {
		return true
	}
	parsed, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, item := range strings.Split(value, ",") {
		cidr := strings.Trim(strings.TrimSpace(item), `"`)
		prefix, err := netip.ParsePrefix(cidr)
		if err == nil && prefix.Contains(parsed) {
			return true
		}
	}
	return false
}

func urlEscape(value string) string {
	var out strings.Builder
	for _, b := range []byte(value) {
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '-' || b == '_' || b == '.' {
			out.WriteByte(b)
			continue
		}
		out.WriteByte('%')
		out.WriteString(strings.ToUpper(strconv.FormatInt(int64(b), 16)))
	}
	return out.String()
}
