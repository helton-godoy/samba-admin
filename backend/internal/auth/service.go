package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/id"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

var (
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrAccountLocked      = errors.New("conta temporariamente bloqueada")
	ErrPasswordExpired    = errors.New("senha expirada")
	ErrMFARequired        = errors.New("verificação MFA necessária")
	ErrNetworkRestricted  = errors.New("origem de rede não autorizada para esta conta")
)

const (
	loginFailureThreshold = 5
	loginLockDuration     = 15 * time.Minute
)

type Service struct {
	store            *storage.Store
	ttl              time.Duration
	challengeTTL     time.Duration
	crypto           *encryption
	mfaRequiredRoles map[string]bool
}

type Options struct {
	TOTPEncryptionKey []byte
	MFARequiredRoles  []string
	ChallengeTTL      time.Duration
}

type LoginResult struct {
	User           models.User
	Session        string
	CSRF           string
	ExpiresAt      time.Time
	MFARequired    bool
	MFAChallengeID string
}

func New(store *storage.Store, ttl time.Duration) *Service {
	return NewWithOptions(store, ttl, Options{})
}

func NewWithOptions(store *storage.Store, ttl time.Duration, options Options) *Service {
	crypto, err := newEncryption(options.TOTPEncryptionKey)
	if err != nil {
		// A malformed key disables MFA rather than silently using unsafe storage.
		crypto = nil
	}
	if options.ChallengeTTL <= 0 {
		options.ChallengeTTL = 5 * time.Minute
	}
	roles := map[string]bool{}
	for _, role := range options.MFARequiredRoles {
		if role = strings.TrimSpace(role); role != "" {
			roles[role] = true
		}
	}
	return &Service{store: store, ttl: ttl, challengeTTL: options.ChallengeTTL, crypto: crypto, mfaRequiredRoles: roles}
}
func (s *Service) Bootstrap(ctx context.Context, user, password string) error {
	for _, r := range []string{"auditor", "operador de arquivos", "operador de impressão", "administrador Samba", "administrador do sistema", "administrador de segurança"} {
		if err := s.store.EnsureRole(ctx, r, r); err != nil {
			return err
		}
	}
	n, err := s.store.UserCount(ctx)
	if err != nil || n > 0 {
		return err
	}
	if password == "" {
		return errors.New("SAMBA_ADMIN_BOOTSTRAP_PASSWORD é obrigatório no primeiro start quando não há usuários")
	}
	h, err := HashPassword(password)
	if err != nil {
		return err
	}
	userID := id.New("user-")
	if err = s.store.InsertUser(ctx, userID, user, "Administrador local de emergência", h); err != nil {
		return err
	}
	if err = s.store.SetBreakGlass(ctx, userID, true); err != nil {
		return err
	}
	return s.store.AssignRole(ctx, userID, "administrador do sistema")
}
func (s *Service) Login(ctx context.Context, username, password, ip, ua string) (models.User, string, string, time.Time, error) {
	result, err := s.BeginLogin(ctx, username, password, ip, ua)
	if err != nil {
		return models.User{}, "", "", time.Time{}, err
	}
	if result.MFARequired {
		return models.User{}, "", "", time.Time{}, ErrMFARequired
	}
	return result.User, result.Session, result.CSRF, result.ExpiresAt, nil
}

func (s *Service) BeginLogin(ctx context.Context, username, password, ip, ua string) (LoginResult, error) {
	u, err := s.store.FindUser(ctx, username)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	if u.LockedUntil.Valid {
		if until, parseErr := time.Parse(time.RFC3339Nano, u.LockedUntil.String); parseErr == nil && until.After(time.Now().UTC()) {
			return LoginResult{}, ErrAccountLocked
		}
	}
	if u.PasswordExpiresAt.Valid {
		if expires, parseErr := time.Parse(time.RFC3339Nano, u.PasswordExpiresAt.String); parseErr == nil && !expires.After(time.Now().UTC()) {
			return LoginResult{}, ErrPasswordExpired
		}
	}
	if u.BreakGlass && !allowedByCIDRs(u.AllowedCIDRs.String, normalizedIP(ip)) {
		return LoginResult{}, ErrNetworkRestricted
	}
	ok, err := VerifyPassword(u.PasswordHash, password)
	if err != nil || !ok {
		locked, _, recordErr := s.store.RegisterLoginFailure(ctx, u.ID, loginFailureThreshold, loginLockDuration)
		if recordErr != nil {
			return LoginResult{}, recordErr
		}
		if locked {
			return LoginResult{}, ErrAccountLocked
		}
		return LoginResult{}, ErrInvalidCredentials
	}
	if err = s.store.ResetLoginFailures(ctx, u.ID); err != nil {
		return LoginResult{}, err
	}
	roles, err := s.store.UserRoles(ctx, u.ID)
	if err != nil {
		return LoginResult{}, err
	}
	user := models.User{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Roles: roles, BreakGlass: u.BreakGlass}
	requiresMFA := u.MFARequired || s.roleRequiresMFA(roles)
	if requiresMFA {
		if _, err = s.store.ActiveTOTP(ctx, u.ID); err != nil {
			return LoginResult{}, ErrMFAEnrollmentRequired
		}
		challengeID, err := randomToken(32)
		if err != nil {
			return LoginResult{}, err
		}
		csrf, err := randomToken(24)
		if err != nil {
			return LoginResult{}, err
		}
		expires := time.Now().UTC().Add(s.challengeTTL)
		if err = s.store.CreateMFAChallenge(ctx, storage.MFAChallengeRecord{ID: challengeID, UserID: u.ID, CSRFToken: csrf, ExpiresAt: expires, IP: normalizedIP(ip), UserAgent: ua}); err != nil {
			return LoginResult{}, err
		}
		return LoginResult{User: user, MFARequired: true, MFAChallengeID: challengeID, ExpiresAt: expires}, nil
	}
	return s.createSession(ctx, user, ip, ua)
}

func (s *Service) CompleteMFA(ctx context.Context, challengeID, code, ip, ua string) (LoginResult, error) {
	challenge, err := s.store.ClaimMFAChallenge(ctx, strings.TrimSpace(challengeID))
	if err != nil {
		return LoginResult{}, ErrMFAInvalid
	}
	if !boundStringEqual(challenge.IP, normalizedIP(ip)) || !boundStringEqual(challenge.UserAgent, ua) {
		return LoginResult{}, ErrMFAInvalid
	}
	if err = s.verifyMFA(ctx, challenge.UserID, strings.TrimSpace(code)); err != nil {
		return LoginResult{}, ErrMFAInvalid
	}
	u, err := s.findUserByID(ctx, challenge.UserID)
	if err != nil {
		return LoginResult{}, ErrMFAInvalid
	}
	u.MFAEnabled = true
	return s.createSession(ctx, u, ip, ua)
}

func (s *Service) createSession(ctx context.Context, user models.User, ip, ua string) (LoginResult, error) {
	session, err := randomToken(32)
	if err != nil {
		return LoginResult{}, err
	}
	csrf, err := randomToken(24)
	if err != nil {
		return LoginResult{}, err
	}
	expires := time.Now().UTC().Add(s.ttl)
	if err := s.store.CreateSession(ctx, session, user.ID, csrf, expires, normalizedIP(ip), ua); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: user, Session: session, CSRF: csrf, ExpiresAt: expires}, nil
}
func (s *Service) Current(ctx context.Context, session string) (models.User, string, error) {
	return s.store.SessionUser(ctx, session)
}
func (s *Service) Logout(ctx context.Context, session string) error {
	return s.store.DeleteSession(ctx, session)
}

func (s *Service) Renew(ctx context.Context, session, ip, ua string) (LoginResult, error) {
	user, _, err := s.Current(ctx, session)
	if err != nil {
		return LoginResult{}, err
	}
	newSession, err := randomToken(32)
	if err != nil {
		return LoginResult{}, err
	}
	csrf, err := randomToken(24)
	if err != nil {
		return LoginResult{}, err
	}
	expires := time.Now().UTC().Add(s.ttl)
	if err = s.store.RotateSession(ctx, session, newSession, csrf, expires, normalizedIP(ip), ua); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: user, Session: newSession, CSRF: csrf, ExpiresAt: expires}, nil
}

func (s *Service) MFARequiredForRoles(roles []string) bool { return s.roleRequiresMFA(roles) }
func randomToken(n int) (string, error) {
	if n < 16 || n > 128 {
		return "", errors.New("tamanho de token aleatório fora da faixa")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Service) roleRequiresMFA(roles []string) bool {
	for _, role := range roles {
		if s.mfaRequiredRoles[role] {
			return true
		}
	}
	return false
}

func (s *Service) findUserByID(ctx context.Context, userID string) (models.User, error) {
	var username string
	if err := s.store.DB.QueryRow(`SELECT username FROM users WHERE id=?`, userID).Scan(&username); err != nil {
		return models.User{}, err
	}
	record, err := s.store.FindUser(ctx, username)
	if err != nil {
		return models.User{}, err
	}
	roles, err := s.store.UserRoles(ctx, record.ID)
	if err != nil {
		return models.User{}, err
	}
	return models.User{ID: record.ID, Username: record.Username, DisplayName: record.DisplayName, Roles: roles, BreakGlass: record.BreakGlass}, nil
}

func normalizedIP(value string) string {
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	return value
}

func boundStringEqual(a, b string) bool {
	if a == "" {
		return true
	}
	if b == "" || len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
