package agent

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidTimestamp = errors.New("timestamp fora da janela permitida")
	ErrInvalidSignature = errors.New("assinatura inválida")
	ErrUnknownKeyID     = errors.New("identificador de chave desconhecido")
)

// KeyRing permits a bounded key-rotation window. The previous key is only
// accepted for verification and is never used to sign new requests.
type KeyRing struct {
	CurrentKey  string
	CurrentID   string
	PreviousKey string
	PreviousID  string
}

func (k KeyRing) Validate() error {
	if len(k.CurrentKey) < 32 {
		return errors.New("a chave HMAC atual deve possuir ao menos 32 bytes")
	}
	if strings.TrimSpace(k.CurrentID) == "" {
		return errors.New("identificador da chave HMAC atual é obrigatório")
	}
	if k.PreviousKey != "" {
		if len(k.PreviousKey) < 32 {
			return errors.New("a chave HMAC anterior deve possuir ao menos 32 bytes")
		}
		if strings.TrimSpace(k.PreviousID) == "" {
			return errors.New("identificador da chave HMAC anterior é obrigatório")
		}
		if k.PreviousID == k.CurrentID {
			return errors.New("identificadores das chaves HMAC devem ser distintos")
		}
	}
	return nil
}

func canonical(r Request) ([]byte, error) {
	payloadValue := r.Payload
	if payloadValue == nil {
		payloadValue = map[string]any{}
	}
	payload, err := json.Marshal(payloadValue)
	if err != nil {
		return nil, fmt.Errorf("serialização do payload assinado: %w", err)
	}
	parts := []string{
		r.Operation,
		fmt.Sprintf("%d", r.OperationVersion),
		r.RequestID,
		r.Timestamp.UTC().Format(time.RFC3339Nano),
		r.Nonce,
		r.KeyID,
		string(payload),
	}
	return []byte(strings.Join(parts, "\n")), nil
}

// Sign is retained for callers from the initial protocol. New callers should
// use SignWithKey so the key identity is explicit in the signed envelope.
func Sign(r *Request, key string) {
	_ = SignWithKey(r, key, "")
}

func SignWithKey(r *Request, key, keyID string) error {
	if r == nil {
		return errors.New("requisição ausente")
	}
	r.KeyID = keyID
	message, err := canonical(*r)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(message)
	r.Signature = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return nil
}

// Verify retains the old single-key entry point for compatibility with unit
// tests and callers compiled before key rotation support was introduced.
func Verify(r Request, key string, maxSkew time.Duration) error {
	return VerifyWithKeyRing(r, KeyRing{CurrentKey: key, CurrentID: "legacy"}, maxSkew)
}

func VerifyWithKeyRing(r Request, keys KeyRing, maxSkew time.Duration) error {
	if err := keys.Validate(); err != nil {
		return err
	}
	if r.Timestamp.IsZero() || maxSkew <= 0 {
		return ErrInvalidTimestamp
	}
	now := time.Now().UTC()
	if r.Timestamp.Before(now.Add(-maxSkew)) || r.Timestamp.After(now.Add(maxSkew)) {
		return ErrInvalidTimestamp
	}
	signature, err := base64.RawURLEncoding.DecodeString(r.Signature)
	if err != nil {
		return ErrInvalidSignature
	}
	message, err := canonical(r)
	if err != nil {
		return err
	}
	keysToTry, err := keys.verificationKeys(r.KeyID)
	if err != nil {
		return err
	}
	for _, key := range keysToTry {
		mac := hmac.New(sha256.New, []byte(key))
		_, _ = mac.Write(message)
		if hmac.Equal(signature, mac.Sum(nil)) {
			return nil
		}
	}
	return ErrInvalidSignature
}

func (k KeyRing) verificationKeys(keyID string) ([]string, error) {
	if keyID == "" {
		keys := []string{k.CurrentKey}
		if k.PreviousKey != "" {
			keys = append(keys, k.PreviousKey)
		}
		return keys, nil
	}
	if keyID == k.CurrentID {
		return []string{k.CurrentKey}, nil
	}
	if k.PreviousKey != "" && keyID == k.PreviousID {
		return []string{k.PreviousKey}, nil
	}
	return nil, ErrUnknownKeyID
}
