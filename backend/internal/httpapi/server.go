package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	openapi "github.com/hu-ufcat/samba-admin-backend/api/generated"
	"github.com/hu-ufcat/samba-admin-backend/internal/adapters"
	mockadapter "github.com/hu-ufcat/samba-admin-backend/internal/adapters/mock"
	"github.com/hu-ufcat/samba-admin-backend/internal/approval"
	"github.com/hu-ufcat/samba-admin-backend/internal/audit"
	"github.com/hu-ufcat/samba-admin-backend/internal/auth"
	"github.com/hu-ufcat/samba-admin-backend/internal/config"
	"github.com/hu-ufcat/samba-admin-backend/internal/events"
	"github.com/hu-ufcat/samba-admin-backend/internal/fixtures"
	"github.com/hu-ufcat/samba-admin-backend/internal/id"
	"github.com/hu-ufcat/samba-admin-backend/internal/jobs"
	appmw "github.com/hu-ufcat/samba-admin-backend/internal/middleware"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
	"github.com/hu-ufcat/samba-admin-backend/internal/rbac"
	"github.com/hu-ufcat/samba-admin-backend/internal/sqlite"
	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
	"github.com/hu-ufcat/samba-admin-backend/internal/validation"
)

type Server struct {
	cfg       config.Config
	store     *storage.Store
	auth      *auth.Service
	rbac      *rbac.Engine
	jobs      *jobs.Manager
	broker    *events.Broker
	audit     *audit.Service
	approvals *approval.Service
	readOnly  ReadOnlyProvider
	logger    *slog.Logger
	mu        sync.RWMutex
	idemMu    sync.Mutex
	shares    []models.Share
}

// ReadOnlyProvider is the narrow interface permitted during FreeBSD
// homologation. It deliberately does not expose join, ACL apply or service
// control methods.
type ReadOnlyProvider interface {
	adapters.System
	adapters.Filesystems
	adapters.SambaInventory
	adapters.CUPSInventory
	Domain(context.Context) (models.DomainState, error)
}

func New(cfg config.Config, store *storage.Store, authSvc *auth.Service, rbacEngine *rbac.Engine, jobsMgr *jobs.Manager, broker *events.Broker, auditSvc *audit.Service, logger *slog.Logger) *Server {
	return &Server{cfg: cfg, store: store, auth: authSvc, rbac: rbacEngine, jobs: jobsMgr, broker: broker, audit: auditSvc, approvals: approval.New(store), readOnly: mockadapter.Provider{}, logger: logger, shares: fixtures.Shares()}
}

func (s *Server) SetReadOnlyProvider(provider ReadOnlyProvider) { s.readOnly = provider }
func (s *Server) Router() http.Handler {
	contractHandler := &openAPIHandler{server: s}
	handler := openapi.HandlerWithOptions(contractHandler, openapi.StdHTTPServerOptions{
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			s.logger.Warn("parâmetro HTTP inválido", "correlation_id", appmw.CorrelationID(r.Context()), "error_type", fmt.Sprintf("%T", err))
			problem(w, r, http.StatusBadRequest, "invalid_request_parameter", "Parâmetro inválido", "A requisição contém um parâmetro ausente ou inválido.", nil)
		},
	})
	handler = appmw.RequireCSRF(s.cfg, handler)
	handler = appmw.Authenticate(s.cfg, s.auth, handler)
	handler = appmw.BodyLimit(s.cfg.MaxBodyBytes, handler)
	handler = appmw.TimeoutExcept(s.cfg.RequestTimeout, "/api/v1/events", handler)
	handler = appmw.CORS(s.cfg.AllowedOrigins, handler)
	handler = appmw.SecurityHeaders(handler)
	handler = appmw.AccessLog(s.logger, handler)
	handler = appmw.Recover(s.logger, handler)
	handler = appmw.RequestID(handler)
	return handler
}
func (s *Server) authorize(action, resource string, next http.Handler) http.Handler {
	return appmw.Authorize(s.rbac, action, resource, next)
}

func (s *Server) withIdempotency(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if err := validation.IdempotencyKey(key); err != nil {
			problem(w, r, http.StatusBadRequest, "invalid_idempotency_key", "Chave de idempotência inválida", err.Error(), nil)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			problem(w, r, http.StatusBadRequest, "invalid_request_body", "Corpo inválido", "Não foi possível ler o corpo da requisição.", nil)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		user := appmw.CurrentUser(r.Context())
		actor := user.ID
		if actor == "" {
			actor = user.Username
		}
		requestHash := fullHash(r.Method + "\n" + r.URL.EscapedPath() + "\n" + r.URL.RawQuery + "\n" + string(body))

		s.idemMu.Lock()
		defer s.idemMu.Unlock()
		record, err := s.store.GetIdempotency(r.Context(), key, actor)
		if err == nil {
			s.writeIdempotencyRecord(w, r, record, requestHash)
			return
		}
		if !errors.Is(err, sqlite.ErrNoRows) {
			problem(w, r, http.StatusServiceUnavailable, "idempotency_lookup_failed", "Idempotência indisponível", "Não foi possível consultar a reserva de idempotência.", nil)
			return
		}
		reservation := storage.IdempotencyRecord{Key: key, Actor: actor, RequestHash: requestHash, ExpiresAt: time.Now().UTC().Add(24 * time.Hour)}
		if err = s.store.ReserveIdempotency(r.Context(), reservation); err != nil {
			problem(w, r, http.StatusConflict, "idempotency_reservation_conflict", "Reserva concorrente", "Outra requisição reservou esta chave; consulte o estado antes de repetir.", nil)
			return
		}

		captured := newBufferedResponse()
		next(captured, r)
		if captured.status >= http.StatusInternalServerError {
			_ = s.store.DeletePendingIdempotency(r.Context(), key, actor, requestHash)
			captured.copyTo(w, false)
			return
		}
		if err = s.store.CompleteIdempotency(r.Context(), key, actor, requestHash, captured.status, captured.body.String()); err != nil {
			problem(w, r, http.StatusServiceUnavailable, "idempotency_store_failed", "Idempotência indisponível", "A operação pode ter sido aceita, mas o resultado idempotente não pôde ser persistido; reconcilie pelo correlation ID.", nil)
			return
		}
		captured.copyTo(w, false)
	}
}

func (s *Server) writeIdempotencyRecord(w http.ResponseWriter, r *http.Request, record storage.IdempotencyRecord, requestHash string) {
	if record.RequestHash != requestHash {
		problem(w, r, http.StatusConflict, "idempotency_conflict", "Conflito de idempotência", "A chave já foi usada com outro conteúdo.", nil)
		return
	}
	if record.ResponseStatus == 0 {
		w.Header().Set("Retry-After", "1")
		problem(w, r, http.StatusConflict, "idempotency_in_progress", "Operação em andamento", "A chave está reservada e ainda não possui resultado confirmado.", nil)
		return
	}
	w.Header().Set("Idempotency-Replayed", "true")
	writeRawJSON(w, record.ResponseStatus, []byte(record.ResponseBody))
}

type bufferedResponse struct {
	header      http.Header
	status      int
	wroteHeader bool
	body        bytes.Buffer
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{header: make(http.Header), status: http.StatusOK}
}

func (w *bufferedResponse) Header() http.Header { return w.header }
func (w *bufferedResponse) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
}
func (w *bufferedResponse) Write(value []byte) (int, error) {
	w.wroteHeader = true
	return w.body.Write(value)
}
func (w *bufferedResponse) copyTo(target http.ResponseWriter, replayed bool) {
	for key, values := range w.header {
		target.Header()[key] = append([]string(nil), values...)
	}
	target.Header().Set("Idempotency-Replayed", fmt.Sprint(replayed))
	target.WriteHeader(w.status)
	_, _ = target.Write(w.body.Bytes())
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "timestamp": time.Now().UTC()})
}
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DB.Ping(); err != nil {
		problem(w, r, 503, "not_ready", "Serviço indisponível", "Banco de dados não está pronto.", nil)
		return
	}
	if s.readOnly != nil && (s.cfg.AdapterMode == "agent" || s.cfg.AdapterMode == "freebsd-readonly") {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if _, err := s.readOnly.Inspect(ctx); err != nil {
			problem(w, r, 503, "not_ready", "Serviço indisponível", "Agente de leitura não está pronto.", nil)
			return
		}
	}
	writeJSON(w, 200, map[string]any{"status": "ready", "timestamp": time.Now().UTC(), "adapterMode": s.cfg.AdapterMode})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	result, err := s.auth.BeginLogin(r.Context(), in.Username, in.Password, clientIP(r), r.UserAgent())
	if err != nil {
		_ = s.audit.Record(r.Context(), audit.Entry{Actor: strings.TrimSpace(in.Username), Source: "local", IP: clientIP(r), Operation: "auth.login", Resource: "session", Result: "failure", CorrelationID: appmw.CorrelationID(r.Context()), Reason: safeAuthReason(err)})
		if errors.Is(err, auth.ErrAccountLocked) {
			w.Header().Set("Retry-After", "900")
			problem(w, r, 429, "account_temporarily_locked", "Conta temporariamente bloqueada", "Muitas tentativas inválidas. Tente novamente após o período de bloqueio.", nil)
			return
		}
		if errors.Is(err, auth.ErrMFAEnrollmentRequired) {
			problem(w, r, 403, "mfa_enrollment_required", "MFA obrigatório", "A política exige ativação prévia de MFA para esta conta.", nil)
			return
		}
		if errors.Is(err, auth.ErrPasswordExpired) {
			problem(w, r, 403, "password_expired", "Senha expirada", "A senha local expirou e deve ser trocada por um administrador autorizado.", nil)
			return
		}
		if errors.Is(err, auth.ErrNetworkRestricted) {
			problem(w, r, 403, "network_restricted", "Origem não autorizada", "A conta de emergência está restrita à rede configurada.", nil)
			return
		}
		problem(w, r, 401, "invalid_credentials", "Falha de autenticação", "Usuário ou senha inválidos.", nil)
		return
	}
	if result.MFARequired {
		_ = s.audit.Record(r.Context(), audit.Entry{Actor: result.User.Username, Roles: result.User.Roles, Source: "local", IP: clientIP(r), Operation: "auth.mfa.challenge", Resource: "session", Result: "pending", CorrelationID: appmw.CorrelationID(r.Context())})
		writeJSON(w, http.StatusOK, map[string]any{"mfaRequired": true, "mfaChallengeId": result.MFAChallengeID, "expiresAt": result.ExpiresAt})
		return
	}
	s.setSessionCookie(w, result.Session, result.ExpiresAt)
	w.Header().Set("X-CSRF-Token", result.CSRF)
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: result.User.Username, Roles: result.User.Roles, Source: "local", IP: clientIP(r), Operation: "auth.login", Resource: "session", Result: "success", CorrelationID: appmw.CorrelationID(r.Context())})
	if result.User.BreakGlass {
		_ = s.audit.Record(r.Context(), audit.Entry{Actor: result.User.Username, Roles: result.User.Roles, Source: "local", IP: clientIP(r), Operation: "auth.break_glass.use", Resource: "session", Result: "success", CorrelationID: appmw.CorrelationID(r.Context()), Reason: "conta local de emergência utilizada"})
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": result.User, "csrfToken": result.CSRF, "expiresAt": result.ExpiresAt, "mfaRequired": false})
}

func (s *Server) verifyMFA(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ChallengeID string `json:"challengeId"`
		Code        string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	result, err := s.auth.CompleteMFA(r.Context(), in.ChallengeID, in.Code, clientIP(r), r.UserAgent())
	if err != nil {
		_ = s.audit.Record(r.Context(), audit.Entry{Actor: "mfa-challenge", Source: "local", IP: clientIP(r), Operation: "auth.mfa.verify", Resource: "session", Result: "failure", CorrelationID: appmw.CorrelationID(r.Context()), Reason: "código MFA recusado"})
		problem(w, r, http.StatusUnauthorized, "invalid_mfa_code", "Falha de MFA", "O desafio expirou ou o código é inválido.", nil)
		return
	}
	s.setSessionCookie(w, result.Session, result.ExpiresAt)
	w.Header().Set("X-CSRF-Token", result.CSRF)
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: result.User.Username, Roles: result.User.Roles, Source: "local", IP: clientIP(r), Operation: "auth.mfa.verify", Resource: "session", Result: "success", CorrelationID: appmw.CorrelationID(r.Context())})
	writeJSON(w, http.StatusOK, map[string]any{"user": result.User, "csrfToken": result.CSRF, "expiresAt": result.ExpiresAt, "mfaRequired": false})
}

func (s *Server) renewSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("samba_admin_session")
	if err != nil {
		problem(w, r, http.StatusUnauthorized, "session_expired", "Sessão expirada", "Autentique-se novamente.", nil)
		return
	}
	result, err := s.auth.Renew(r.Context(), cookie.Value, clientIP(r), r.UserAgent())
	if err != nil {
		problem(w, r, http.StatusUnauthorized, "session_expired", "Sessão expirada", "Autentique-se novamente.", nil)
		return
	}
	s.setSessionCookie(w, result.Session, result.ExpiresAt)
	w.Header().Set("X-CSRF-Token", result.CSRF)
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: result.User.Username, Roles: result.User.Roles, Source: "local", IP: clientIP(r), Operation: "auth.session.renew", Resource: "session", Result: "success", CorrelationID: appmw.CorrelationID(r.Context())})
	writeJSON(w, http.StatusOK, map[string]any{"user": result.User, "csrfToken": result.CSRF, "expiresAt": result.ExpiresAt, "mfaRequired": false})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("samba_admin_session"); err == nil {
		_ = s.auth.Logout(r.Context(), c.Value)
	}
	user := appmw.CurrentUser(r.Context())
	if user.ID != "" {
		_ = s.audit.Record(r.Context(), audit.Entry{Actor: user.Username, Roles: user.Roles, Source: "local", IP: clientIP(r), Operation: "auth.logout", Resource: "session", Result: "success", CorrelationID: appmw.CorrelationID(r.Context())})
	}
	http.SetCookie(w, &http.Cookie{Name: "samba_admin_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.cfg.AuthMode != "development-bypass", SameSite: http.SameSiteStrictMode})
	w.WriteHeader(204)
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-CSRF-Token", appmw.CSRFToken(r.Context()))
	writeJSON(w, 200, appmw.CurrentUser(r.Context()))
}

func (s *Server) beginTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	user := appmw.CurrentUser(r.Context())
	enrollment, err := s.auth.BeginTOTPEnrollment(r.Context(), user.ID, "Samba Admin Suite", user.Username)
	if err != nil {
		problem(w, r, http.StatusConflict, "mfa_unavailable", "MFA indisponível", "A configuração segura de MFA não está disponível.", nil)
		return
	}
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: user.Username, Roles: user.Roles, Source: "local", IP: clientIP(r), Operation: "auth.mfa.enrollment.begin", Resource: "user:" + user.ID, Result: "success", CorrelationID: appmw.CorrelationID(r.Context())})
	writeJSON(w, http.StatusCreated, map[string]any{"otpauthUri": enrollment.URI, "manualSecret": enrollment.Secret})
}

func (s *Server) confirmTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	user := appmw.CurrentUser(r.Context())
	codes, err := s.auth.ConfirmTOTPEnrollment(r.Context(), user.ID, in.Code)
	if err != nil {
		problem(w, r, http.StatusUnprocessableEntity, "invalid_mfa_code", "Falha de MFA", "O código de confirmação é inválido.", nil)
		return
	}
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: user.Username, Roles: user.Roles, Source: "local", IP: clientIP(r), Operation: "auth.mfa.enrollment.confirm", Resource: "user:" + user.ID, Result: "success", CorrelationID: appmw.CorrelationID(r.Context())})
	s.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"codes": codes})
}

func (s *Server) rotateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	user := appmw.CurrentUser(r.Context())
	codes, err := s.auth.RotateRecoveryCodes(r.Context(), user.ID)
	if err != nil {
		problem(w, r, http.StatusConflict, "mfa_unavailable", "MFA indisponível", "Não foi possível gerar códigos de recuperação.", nil)
		return
	}
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: user.Username, Roles: user.Roles, Source: "local", IP: clientIP(r), Operation: "auth.mfa.recovery.rotate", Resource: "user:" + user.ID, Result: "success", CorrelationID: appmw.CorrelationID(r.Context())})
	s.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"codes": codes})
}

func (s *Server) revokeTOTP(w http.ResponseWriter, r *http.Request) {
	user := appmw.CurrentUser(r.Context())
	if s.auth.MFARequiredForRoles(user.Roles) {
		problem(w, r, http.StatusConflict, "mfa_required_by_policy", "MFA obrigatório", "A política do papel impede a revogação de MFA.", nil)
		return
	}
	if err := s.auth.RevokeTOTP(r.Context(), user.ID); err != nil {
		problem(w, r, http.StatusInternalServerError, "mfa_revoke_failed", "Falha de MFA", "Não foi possível revogar MFA.", nil)
		return
	}
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: user.Username, Roles: user.Roles, Source: "local", IP: clientIP(r), Operation: "auth.mfa.revoke", Resource: "user:" + user.ID, Result: "success", CorrelationID: appmw.CorrelationID(r.Context())})
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) setSessionCookie(w http.ResponseWriter, value string, expires time.Time) {
	secure := s.cfg.AuthMode != "development-bypass"
	http.SetCookie(w, &http.Cookie{Name: "samba_admin_session", Value: value, Path: "/", Expires: expires, MaxAge: int(time.Until(expires).Seconds()), Secure: secure, HttpOnly: true, SameSite: http.SameSiteStrictMode})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "samba_admin_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.cfg.AuthMode != "development-bypass", SameSite: http.SameSiteStrictMode})
	w.Header().Set("X-Session-Reauthentication", "required")
}

func safeAuthReason(err error) string {
	switch {
	case errors.Is(err, auth.ErrAccountLocked):
		return "conta temporariamente bloqueada"
	case errors.Is(err, auth.ErrMFAEnrollmentRequired):
		return "MFA obrigatório não configurado"
	case errors.Is(err, auth.ErrPasswordExpired):
		return "senha expirada"
	case errors.Is(err, auth.ErrNetworkRestricted):
		return "origem de rede não autorizada"
	default:
		return "credenciais inválidas"
	}
}
func (s *Server) system(w http.ResponseWriter, r *http.Request) {
	v := fixtures.System()
	if s.readOnly != nil {
		var err error
		v, err = s.readOnly.Inspect(r.Context())
		if err != nil {
			problem(w, r, http.StatusServiceUnavailable, "capability_unavailable", "Inventário indisponível", "O adapter FreeBSD de leitura não pôde concluir o inventário.", nil)
			return
		}
	}
	etag := etag(v)
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(304)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) samba(w http.ResponseWriter, r *http.Request) {
	if s.readOnly == nil {
		problem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Inventário Samba indisponível", "O provider somente leitura não está disponível.", nil)
		return
	}
	info, err := s.readOnly.Samba(r.Context())
	if err != nil {
		problem(w, r, http.StatusServiceUnavailable, "capability_unavailable", "Inventário Samba indisponível", "O agente não pôde concluir a sondagem mínima do Samba.", nil)
		return
	}
	writeJSON(w, http.StatusOK, info)
}
func (s *Server) cups(w http.ResponseWriter, r *http.Request) {
	if s.readOnly == nil {
		problem(w, r, http.StatusServiceUnavailable, "agent_unavailable", "Inventário CUPS indisponível", "O provider somente leitura não está disponível.", nil)
		return
	}
	info, err := s.readOnly.Cups(r.Context())
	if err != nil {
		problem(w, r, http.StatusServiceUnavailable, "capability_unavailable", "Inventário CUPS indisponível", "O agente não pôde concluir a sondagem mínima do CUPS.", nil)
		return
	}
	writeJSON(w, http.StatusOK, info)
}
func (s *Server) capabilities(w http.ResponseWriter, r *http.Request) {
	if s.readOnly != nil {
		capabilities, err := s.readOnly.Capabilities(r.Context())
		if err != nil {
			problem(w, r, http.StatusServiceUnavailable, "capability_unavailable", "Capacidades indisponíveis", "O adapter FreeBSD de leitura não pôde concluir a detecção.", nil)
			return
		}
		writeJSON(w, http.StatusOK, capabilities)
		return
	}
	writeJSON(w, 200, fixtures.Capabilities())
}
func (s *Server) filesystems(w http.ResponseWriter, r *http.Request) {
	if s.readOnly != nil {
		fileSystems, err := s.readOnly.List(r.Context())
		if err != nil {
			problem(w, r, http.StatusServiceUnavailable, "capability_unavailable", "Inventário indisponível", "O adapter FreeBSD de leitura não pôde listar os sistemas de arquivos.", nil)
			return
		}
		writeJSON(w, http.StatusOK, fileSystems)
		return
	}
	writeJSON(w, 200, fixtures.Filesystems())
}
func (s *Server) listShares(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]models.Share(nil), s.shares...)
	filter := strings.ToLower(r.URL.Query().Get("filter"))
	if filter != "" {
		tmp := out[:0]
		for _, x := range out {
			if strings.Contains(strings.ToLower(x.Name+" "+x.Description+" "+x.Path), filter) {
				tmp = append(tmp, x)
			}
		}
		out = tmp
	}
	writeJSON(w, 200, out)
}
func (s *Server) previewShare(w http.ResponseWriter, r *http.Request) {
	var sh models.Share
	if !decode(w, r, &sh) {
		return
	}
	if err := validateShare(sh); err != nil {
		validationProblem(w, r, "share", err.Error())
		return
	}
	cfg := renderShare(sh)
	writeJSON(w, 200, map[string]any{"valid": true, "testparm": "Loaded services file OK. Server role: ROLE_DOMAIN_MEMBER (simulado)", "reloadRequired": true, "restartRequired": false, "config": cfg, "diff": "--- smb4.conf.anterior\n+++ smb4.conf.proposto\n@@\n+" + strings.ReplaceAll(strings.TrimSuffix(cfg, "\n"), "\n", "\n+"), "impact": map[string]any{"activeSessions": fixtures.System().SMBSessions, "openFiles": fixtures.System().OpenFiles, "affectedShares": []string{sh.Name}}})
}
func (s *Server) createShare(w http.ResponseWriter, r *http.Request) {
	var sh models.Share
	if !decode(w, r, &sh) {
		return
	}
	if err := validateShare(sh); err != nil {
		validationProblem(w, r, "share", err.Error())
		return
	}
	if sh.GuestAccess {
		problem(w, r, 403, "policy_denied", "Operação bloqueada", "A política institucional bloqueia acesso guest.", nil)
		return
	}
	if !s.mutableOperationsEnabled() {
		s.mutableOperationBlocked(w, r)
		return
	}

	actor := appmw.CurrentUser(r.Context()).Username
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.shares {
		if strings.EqualFold(x.Name, sh.Name) {
			problem(w, r, 409, "resource_conflict", "Conflito", "Já existe compartilhamento com esse nome.", nil)
			return
		}
	}
	sh.ID = id.New("share-")
	job, err := s.jobs.Create(r.Context(), actor, "Criar compartilhamento "+sh.Name, "share:"+sh.ID, appmw.CorrelationID(r.Context()), "share.create", map[string]any{"share": sh, "dryRun": false}, true)
	if err != nil {
		problem(w, r, 500, "job_creation_failed", "Falha ao criar tarefa", err.Error(), nil)
		return
	}
	s.shares = append(s.shares, sh)
	response := map[string]any{"share": sh, "job": job}
	w.Header().Set("Location", "/api/v1/jobs/"+job.ID)
	writeJSON(w, http.StatusCreated, response)
}
func (s *Server) acls(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, fixtures.ACLs()) }
func (s *Server) legacyACLs(w http.ResponseWriter, r *http.Request) {
	setDeprecationHeaders(w, "/api/v1/acls")
	s.acls(w, r)
}
func (s *Server) principals(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, fixtures.Principals())
}
func (s *Server) legacyPrincipals(w http.ResponseWriter, r *http.Request) {
	setDeprecationHeaders(w, "/api/v1/identities")
	s.principals(w, r)
}
func (s *Server) effectiveACL(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Principal string `json:"principal"`
	}
	if !decode(w, r, &in) {
		return
	}
	writeJSON(w, 200, map[string]any{"principal": in.Principal, "effective": []string{"Ler e executar", "Criar arquivos", "Criar pastas", "Excluir próprios arquivos"}, "denied": []string{"Alterar proprietário", "Alterar ACL"}, "warnings": []string{"Resultado mockado; grupos aninhados e o token real devem ser resolvidos no backend FreeBSD/AD."}})
}
func (s *Server) simulateACLConversion(w http.ResponseWriter, r *http.Request) {
	var in struct {
		FilesystemID string `json:"filesystemId"`
		Target       string `json:"target"`
	}
	if !decode(w, r, &in) {
		return
	}
	var fs *models.FileSystem
	for _, x := range fixtures.Filesystems() {
		if x.ID == in.FilesystemID {
			v := x
			fs = &v
			break
		}
	}
	if fs == nil {
		problem(w, r, 404, "not_found", "Não encontrado", "Sistema de arquivos não encontrado.", nil)
		return
	}
	if in.Target != "posix" && in.Target != "nfsv4" {
		validationProblem(w, r, "target", "Modelo de ACL inválido.")
		return
	}
	writeJSON(w, 200, map[string]any{"filesystem": fs.MountPoint, "from": fs.ACLModel, "to": in.Target, "requiresMaintenance": true, "requiresUnmount": true, "backupArtifact": "/var/backups/samba-admin/acl-" + in.FilesystemID + "-simulado.jsonl", "affectedShares": fs.AssociatedShares, "scannedObjects": 184203, "preserved": 173910, "approximated": 9911, "notRepresentable": 382, "risks": []string{"A ordem e a precedência de ACEs DENY podem sofrer alteração.", "Máscaras POSIX não possuem equivalência perfeita no modelo NFSv4.", "Entradas com identidades não resolvidas exigem mapeamento manual.", "A reversão restaura o inventário exportado, mas não garante equivalência semântica perfeita."}, "plan": []string{"Bloquear novas alterações e notificar usuários.", "Encerrar sessões e verificar arquivos abertos.", "Exportar ACLs, donos, grupos e metadados.", "Desmontar o sistema de arquivos em janela de manutenção.", "Alterar /etc/fstab de acls para nfsv4acls.", "Montar, reaplicar ACLs mapeadas e executar testes de acesso.", "Reabrir o serviço apenas após verificação de saúde."}})
}
func (s *Server) domain(w http.ResponseWriter, r *http.Request) {
	if s.readOnly != nil {
		state, err := s.readOnly.Domain(r.Context())
		if err != nil {
			problem(w, r, http.StatusServiceUnavailable, "capability_unavailable", "Diagnóstico indisponível", "O adapter FreeBSD de leitura não pôde diagnosticar o domínio.", nil)
			return
		}
		writeJSON(w, http.StatusOK, state)
		return
	}
	writeJSON(w, 200, fixtures.Domain())
}
func (s *Server) domainTest(w http.ResponseWriter, r *http.Request) {
	d := fixtures.Domain()
	modeled := []string{"DNS SRV", "diferença de horário", "Kerberos com credencial efêmera", "LDAP TLS", "SMB 3", "winbind", "resolução de usuário", "resolução de grupo"}
	executed := []string{}
	if s.readOnly != nil {
		state, err := s.readOnly.Domain(r.Context())
		if err != nil {
			problem(w, r, http.StatusServiceUnavailable, "capability_unavailable", "Diagnóstico indisponível", "O adapter FreeBSD de leitura não pôde diagnosticar o domínio.", nil)
			return
		}
		d = state
		if s.cfg.AdapterMode == "freebsd-readonly" || s.cfg.AdapterMode == "agent" {
			executed = []string{"/usr/local/bin/testparm -s", "/bin/cat /etc/resolv.conf", "/usr/local/bin/wbinfo --ping-dc"}
		}
	}
	writeJSON(w, 202, map[string]any{"success": true, "checks": d.Tests, "commandsExecuted": executed, "commandsModeled": modeled})
}
func (s *Server) sambaProfiles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, []map[string]any{{"id": "standalone", "name": "Servidor autônomo", "selected": false, "requiresReprovision": false}, {"id": "domain-member", "name": "Membro de Active Directory existente", "selected": true, "requiresReprovision": false}, {"id": "ad-dc", "name": "Controlador de domínio Samba AD DC", "selected": false, "requiresReprovision": true, "capability": "nao_verificado"}, {"id": "additional-dc", "name": "Controlador adicional", "selected": false, "requiresReprovision": true, "capability": "nao_verificado"}})
}
func (s *Server) printers(w http.ResponseWriter, r *http.Request) {
	if s.readOnly != nil {
		cups, err := s.readOnly.Cups(r.Context())
		if err != nil {
			problem(w, r, http.StatusServiceUnavailable, "capability_unavailable", "Impressão indisponível", "O inventário CUPS não pôde ser consultado pelo agente.", nil)
			return
		}
		writeJSON(w, http.StatusOK, cups.Printers)
		return
	}
	writeJSON(w, 200, fixtures.Printers())
}
func (s *Server) drivers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, fixtures.Drivers())
}
func (s *Server) dfs(w http.ResponseWriter, r *http.Request)    { writeJSON(w, 200, fixtures.DFS()) }
func (s *Server) quotas(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, fixtures.Quotas()) }
func (s *Server) logging(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"local": true, "remoteEnabled": true, "destination": "siem.exemplo.interno", "port": 6514, "protocol": "tcp-tls", "format": "RFC5424", "tlsCapability": "restrito", "queueEnabled": true, "retentionDays": 90})
}
func (s *Server) auditEvents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, fixtures.Audit())
}
func (s *Server) services(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, fixtures.Services())
}
func (s *Server) serviceAction(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		Action string `json:"action"`
	}
	if !decode(w, r, &in) {
		return
	}
	actionLabels := map[string]string{
		"start":        "Iniciar",
		"stop":         "Parar",
		"restart":      "Reiniciar",
		"reload":       "Recarregar",
		"health-check": "Verificar saúde do",
	}
	actionLabel, allowed := actionLabels[in.Action]
	if !allowed {
		validationProblem(w, r, "action", "Ação de serviço inválida.")
		return
	}
	if in.Action == "restart" && fixtures.System().SMBSessions > 0 {
		problem(w, r, 428, "approval_required", "Aprovação obrigatória", "Reinício do Samba com sessões ativas exige solicitação, aprovação segregada e janela de manutenção.", map[string]any{"operation": "samba.service.restart", "activeSessions": fixtures.System().SMBSessions})
		return
	}
	if in.Action != "health-check" && !s.mutableOperationsEnabled() {
		s.mutableOperationBlocked(w, r)
		return
	}
	op := map[string]string{
		"health-check": "service.status",
		"start":        "service.start",
		"stop":         "service.stop",
		"reload":       "service.reload",
		"restart":      "service.restart",
	}[in.Action]
	actor := appmw.CurrentUser(r.Context()).Username
	job, err := s.jobs.Create(r.Context(), actor, actionLabel+" serviço "+id, "service:"+id, appmw.CorrelationID(r.Context()), op, map[string]any{"serviceId": id, "action": in.Action}, in.Action == "restart" || in.Action == "stop")
	if err != nil {
		problem(w, r, 500, "job_creation_failed", "Falha ao criar tarefa", err.Error(), nil)
		return
	}
	writeJSON(w, 202, map[string]any{"id": id, "action": in.Action, "accepted": true, "warning": func() string {
		if in.Action == "restart" || in.Action == "stop" {
			return "Sessões SMB e trabalhos de impressão podem ser interrompidos."
		}
		return ""
	}(), "job": job})
}
func (s *Server) configurationFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if err := validation.ConfigPath(path); err != nil {
		problem(w, r, 403, "path_not_allowed", "Acesso bloqueado", err.Error(), nil)
		return
	}
	content := configContent(path)
	previous := strings.Replace(content, "map to guest = Never", "map to guest = Bad User", 1)
	writeJSON(w, 200, map[string]any{"path": path, "encoding": "UTF-8", "eol": "LF", "version": "sha256:" + shortHash(content), "lockedBy": nil, "size": len(content), "content": content, "previousContent": previous})
}
func (s *Server) validateConfiguration(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := validation.ConfigPath(in.Path); err != nil {
		problem(w, r, 403, "path_not_allowed", "Acesso bloqueado", err.Error(), nil)
		return
	}
	if len(in.Content) > 1<<20 {
		validationProblem(w, r, "content", "Arquivo excede o limite de 1 MiB.")
		return
	}
	if strings.Contains(in.Content, "server min protocol = NT1") {
		validationProblem(w, r, "content", "SMB1/NT1 é bloqueado pela política de segurança.")
		return
	}
	writeJSON(w, 200, map[string]any{"valid": true, "validators": []string{"sintaxe", "política de segurança", "testparm conceitual"}, "diff": "--- versão anterior\n+++ versão proposta\n@@ validação mockada para " + in.Path + "\n", "backupRequired": true, "reloadRequired": strings.HasSuffix(in.Path, "smb4.conf")})
}
func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	j, err := s.store.ListJobs(r.Context())
	if err != nil {
		problem(w, r, 500, "storage_error", "Falha de persistência", err.Error(), nil)
		return
	}
	writeJSON(w, 200, j)
}
func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if err := s.jobs.Cancel(jobID); err != nil {
		problem(w, r, 409, "job_not_cancellable", "Tarefa não cancelável", err.Error(), nil)
		return
	}
	writeJSON(w, 202, map[string]any{"jobId": jobID, "status": "cancellation-requested"})
}

func (s *Server) rollbackJob(w http.ResponseWriter, r *http.Request) {
	if !s.mutableOperationsEnabled() {
		s.mutableOperationBlocked(w, r)
		return
	}
	j, err := s.jobs.Rollback(r.Context(), r.PathValue("id"), appmw.CurrentUser(r.Context()).Username, appmw.CorrelationID(r.Context()))
	if err != nil {
		problem(w, r, 404, "rollback_unavailable", "Rollback indisponível", err.Error(), nil)
		return
	}
	writeJSON(w, 202, map[string]any{"id": r.PathValue("id"), "status": "queued", "message": "Rollback simulado agendado.", "job": j})
}
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		problem(w, r, 500, "stream_not_supported", "SSE indisponível", "Servidor HTTP não suporta streaming.", nil)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID == "" {
		// Browser EventSource cannot set an arbitrary header for a manually
		// recreated connection, so this query fallback keeps replay available.
		lastEventID = r.URL.Query().Get("lastEventId")
	}
	_, ch, unsubscribe, replay := s.broker.SubscribeAfter(lastEventID)
	defer unsubscribe()
	fmt.Fprintf(w, "event: ready\ndata: {\"status\":\"connected\",\"replayed\":%t}\n\n", len(replay) > 0)
	for _, event := range replay {
		writeSSEEvent(w, event)
	}
	flusher.Flush()
	keep := time.NewTicker(20 * time.Second)
	defer keep.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, e)
			flusher.Flush()
		case <-keep.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
func writeSSEEvent(w http.ResponseWriter, event events.Event) {
	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.JSON())
}
func (s *Server) backups(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, []map[string]any{{"id": "backup-simulado-001", "resourceType": "samba-config", "resourceId": "global", "artifactPath": "/var/backups/samba-admin/smb4.conf.simulado", "checksum": "sha256:simulado", "status": "verified", "createdAt": "2026-07-11T15:00:00Z"}})
}
func (s *Server) rollbackByBackup(w http.ResponseWriter, r *http.Request) {
	if !s.mutableOperationsEnabled() {
		s.mutableOperationBlocked(w, r)
		return
	}
	var in struct {
		BackupID string `json:"backupId"`
		Reason   string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.BackupID == "" {
		validationProblem(w, r, "backupId", "Backup obrigatório.")
		return
	}
	j, err := s.jobs.Create(r.Context(), appmw.CurrentUser(r.Context()).Username, "Restaurar backup "+in.BackupID, "backup:"+in.BackupID, appmw.CorrelationID(r.Context()), "rollback.execute", map[string]any{"backupId": in.BackupID, "reason": in.Reason}, false)
	if err != nil {
		problem(w, r, 500, "job_creation_failed", "Falha ao criar tarefa", err.Error(), nil)
		return
	}
	writeJSON(w, 202, j)
}

func (s *Server) listChangeRequests(w http.ResponseWriter, r *http.Request) {
	requests, err := s.approvals.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		problem(w, r, http.StatusInternalServerError, "storage_error", "Falha de persistência", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, requests)
}

func (s *Server) createChangeRequest(w http.ResponseWriter, r *http.Request) {
	var input approval.RequestInput
	if !decode(w, r, &input) {
		return
	}
	user := appmw.CurrentUser(r.Context())
	request, err := s.approvals.Request(r.Context(), user, input)
	if err != nil {
		validationProblem(w, r, "changeRequest", err.Error())
		return
	}
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: user.Username, Roles: user.Roles, Source: "api", IP: clientIP(r), Operation: "change.request", Resource: request.Resource, Result: "success", CorrelationID: appmw.CorrelationID(r.Context()), Reason: request.Justification, After: map[string]any{"changeRequestId": request.ID, "operation": request.Operation, "risk": request.Risk, "status": request.Status}})
	writeJSON(w, http.StatusCreated, request)
}

func (s *Server) getChangeRequest(w http.ResponseWriter, r *http.Request) {
	request, err := s.approvals.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		problem(w, r, http.StatusNotFound, "not_found", "Não encontrado", "Solicitação de mudança não encontrada.", nil)
		return
	}
	writeJSON(w, http.StatusOK, request)
}

func (s *Server) approveChangeRequest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &input) {
		return
	}
	user := appmw.CurrentUser(r.Context())
	request, err := s.approvals.Approve(r.Context(), r.PathValue("id"), user, input.Reason)
	if err != nil {
		problem(w, r, http.StatusConflict, "approval_denied", "Aprovação recusada", err.Error(), nil)
		return
	}
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: user.Username, Roles: user.Roles, Source: "api", IP: clientIP(r), Operation: "change.approve", Resource: request.Resource, Result: "success", CorrelationID: appmw.CorrelationID(r.Context()), Reason: request.DecisionReason, After: map[string]any{"changeRequestId": request.ID, "operation": request.Operation}})
	writeJSON(w, http.StatusOK, request)
}

func (s *Server) rejectChangeRequest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &input) {
		return
	}
	user := appmw.CurrentUser(r.Context())
	request, err := s.approvals.Reject(r.Context(), r.PathValue("id"), user, input.Reason)
	if err != nil {
		problem(w, r, http.StatusConflict, "approval_denied", "Decisão recusada", err.Error(), nil)
		return
	}
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: user.Username, Roles: user.Roles, Source: "api", IP: clientIP(r), Operation: "change.reject", Resource: request.Resource, Result: "success", CorrelationID: appmw.CorrelationID(r.Context()), Reason: request.DecisionReason, After: map[string]any{"changeRequestId": request.ID, "operation": request.Operation}})
	writeJSON(w, http.StatusOK, request)
}

func (s *Server) executeChangeRequest(w http.ResponseWriter, r *http.Request) {
	request, err := s.approvals.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		problem(w, r, http.StatusNotFound, "not_found", "Não encontrado", "Solicitação de mudança não encontrada.", nil)
		return
	}
	if request.Status != "approved" {
		problem(w, r, http.StatusConflict, "approval_required", "Aprovação pendente", "A solicitação deve estar aprovada antes da execução.", nil)
		return
	}
	policy := approval.PolicyFor(request.Operation)
	now := time.Now().UTC()
	if policy.RequiresWindow {
		if request.MaintenanceStart == nil || request.MaintenanceEnd == nil {
			problem(w, r, http.StatusConflict, "maintenance_window_required", "Janela obrigatória", "A solicitação aprovada não possui janela de manutenção válida.", nil)
			return
		}
		if now.Before(*request.MaintenanceStart) {
			problem(w, r, http.StatusConflict, "maintenance_window_not_started", "Fora da janela", "A janela de manutenção ainda não começou.", map[string]any{"maintenanceStart": request.MaintenanceStart})
			return
		}
		if !now.Before(*request.MaintenanceEnd) {
			problem(w, r, http.StatusConflict, "maintenance_window_expired", "Janela expirada", "A janela de manutenção terminou; solicite nova aprovação.", nil)
			return
		}
	}
	if policy.RequiredCapability != "" && !s.capabilityEnabled(policy.RequiredCapability) {
		problem(w, r, http.StatusServiceUnavailable, "capability_unavailable", "Capacidade indisponível", "A capacidade necessária não está homologada neste host.", map[string]any{"capability": policy.RequiredCapability})
		return
	}
	if !s.mutableOperationsEnabled() {
		s.mutableOperationBlocked(w, r)
		return
	}
	record, err := s.store.GetChangeRequest(r.Context(), request.ID)
	if err != nil {
		problem(w, r, http.StatusInternalServerError, "storage_error", "Falha de persistência", err.Error(), nil)
		return
	}
	payload := map[string]any{}
	if err = json.Unmarshal([]byte(record.PayloadJSON), &payload); err != nil {
		problem(w, r, http.StatusInternalServerError, "invalid_change_payload", "Falha de persistência", "O payload aprovado não pode ser recuperado.", nil)
		return
	}
	user := appmw.CurrentUser(r.Context())
	jobID := id.New("JOB-")
	if err = s.approvals.MarkExecuting(r.Context(), request.ID, jobID); err != nil {
		problem(w, r, http.StatusConflict, "change_state_conflict", "Conflito", "A solicitação foi alterada antes do início da tarefa.", nil)
		return
	}
	job, err := s.jobs.CreateWithID(r.Context(), jobID, user.Username, "Executar mudança aprovada: "+request.Operation, request.Resource, appmw.CorrelationID(r.Context()), request.Operation, payload, policy.Risk != approval.RiskRead)
	if err != nil {
		_ = s.approvals.ReleaseExecution(r.Context(), request.ID, jobID)
		problem(w, r, http.StatusInternalServerError, "job_creation_failed", "Falha ao criar tarefa", err.Error(), nil)
		return
	}
	go s.trackChangeRequest(request.ID, job.ID)
	_ = s.audit.Record(r.Context(), audit.Entry{Actor: user.Username, Roles: user.Roles, Source: "api", IP: clientIP(r), Operation: "change.execute", Resource: request.Resource, Result: "accepted", CorrelationID: appmw.CorrelationID(r.Context()), JobID: job.ID, After: map[string]any{"changeRequestId": request.ID, "operation": request.Operation}})
	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) trackChangeRequest(changeID, jobID string) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(24 * time.Hour)
	defer timeout.Stop()
	for {
		select {
		case <-timeout.C:
			return
		case <-ticker.C:
			job, err := s.store.GetJob(context.Background(), jobID)
			if err != nil {
				return
			}
			status := ""
			switch job.Status {
			case "success":
				status = "completed"
			case "failed", "partial":
				status = "failed"
			case "cancelled":
				status = "cancelled"
			case "rolled-back":
				status = "rolled_back"
			}
			if status != "" {
				_ = s.approvals.Complete(context.Background(), changeID, status)
				return
			}
		}
	}
}

func (s *Server) capabilityEnabled(id string) bool {
	for _, capability := range fixtures.Capabilities() {
		if capability.ID != id {
			continue
		}
		return capability.State == "suportado" || capability.State == "supported"
	}
	return false
}

func (s *Server) mutableOperationsEnabled() bool {
	if s.cfg.AdapterMode == "mock" {
		return true
	}
	// A feature flag can only enable another simulated path. The real FreeBSD
	// executor remains read-only in this release candidate.
	return s.cfg.AdapterMode == "agent" && s.cfg.AgentExecutorMode == "mock" && s.cfg.EnableMutableOperations
}

func (s *Server) mutableOperationBlocked(w http.ResponseWriter, r *http.Request) {
	problem(w, r, http.StatusServiceUnavailable, "mutation_blocked", "Operação mutável bloqueada", "A homologação atual permite somente leitura; a feature flag de mutação permanece desabilitada.", nil)
}

func validateShare(s models.Share) error {
	if err := validation.ShareName(s.Name); err != nil {
		return err
	}
	if err := validation.SharePath(s.Path); err != nil {
		return err
	}
	if s.GuestAccess {
		return fmt.Errorf("acesso guest não permitido")
	}
	if err := validation.SingleLine("descrição", s.Description, 255, false); err != nil {
		return err
	}
	if len(s.AllowedPrincipals) == 0 || len(s.AllowedPrincipals) > 256 || len(s.DeniedPrincipals) > 256 {
		return fmt.Errorf("lista de principals fora da faixa permitida")
	}
	principals := append(append([]string(nil), s.AllowedPrincipals...), s.DeniedPrincipals...)
	for _, principal := range principals {
		if err := validation.Principal(principal); err != nil {
			return err
		}
	}
	if !oneOf(s.Encryption, "disabled", "desired", "required") {
		return fmt.Errorf("política de criptografia inválida")
	}
	if !oneOf(s.Signing, "default", "mandatory") {
		return fmt.Errorf("política de assinatura inválida")
	}
	if !oneOf(s.AuditProfile, "off", "minimum", "security", "changes", "complete") {
		return fmt.Errorf("perfil de auditoria inválido")
	}
	if !oneOf(s.ACLModel, "posix", "nfsv4", "none") {
		return fmt.Errorf("modelo de ACL inválido")
	}
	if len(s.VFSModules) > 8 {
		return fmt.Errorf("módulos VFS excedem o limite")
	}
	for _, module := range s.VFSModules {
		if !oneOf(module, "acl_xattr", "recycle", "full_audit") {
			return fmt.Errorf("módulo VFS não permitido: %s", module)
		}
	}
	if s.MaxConnections < 1 || s.MaxConnections > 10000 {
		return fmt.Errorf("limite de conexões fora da faixa")
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
func renderShare(s models.Share) string {
	enc := s.Encryption
	if enc == "disabled" {
		enc = "off"
	}
	valid := strings.Join(s.AllowedPrincipals, " ")
	if valid == "" {
		valid = "@EBSERHNET\\Domain Users"
	}
	return fmt.Sprintf("[%s]\n  comment = %s\n  path = %s\n  browseable = yes\n  read only = %s\n  guest ok = no\n  server smb encrypt = %s\n  server signing = %s\n  valid users = %s\n  vfs objects = %s\n", s.Name, s.Description, s.Path, yesNo(s.ReadOnly), enc, s.Signing, valid, strings.Join(s.VFSModules, " "))
}
func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
func configContent(path string) string {
	switch filepath.Clean(path) {
	case "/usr/local/etc/smb4.conf":
		return "[global]\n  workgroup = EBSERHNET\n  realm = EBSERHNET.EBSERH.GOV.BR\n  security = ADS\n  server min protocol = SMB2_10\n  map to guest = Never\n  idmap config * : backend = tdb\n  idmap config * : range = 1000000-1999999\n  idmap config EBSERHNET : backend = rid\n  idmap config EBSERHNET : range = 2000000-2999999\n"
	case "/etc/fstab":
		return "/dev/nda1p1 / ufs rw 1 1\n/dev/nda1p2 /srv/dados ufs rw,nfsv4acls,userquota,groupquota 2 2\n"
	default:
		return "# Arquivo mockado: " + path + "\n# O adapter real aplicará validação, locking, backup e rollback.\n"
	}
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		problem(w, r, 400, "invalid_json", "Requisição inválida", err.Error(), nil)
		return false
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		problem(w, r, 400, "invalid_json", "Requisição inválida", "A requisição deve conter um único documento JSON.", nil)
		return false
	}
	return true
}
func validationProblem(w http.ResponseWriter, r *http.Request, field, msg string) {
	problem(w, r, 422, "validation_error", "Falha de validação", msg, []map[string]string{{"field": field, "code": "invalid_value", "message": msg}})
}
func problem(w http.ResponseWriter, r *http.Request, status int, typ, title, detail string, errs any) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": typ, "title": title, "status": status, "detail": detail, "instance": r.URL.Path, "correlationId": appmw.CorrelationID(r.Context()), "errors": errs})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeRawJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
func etag(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}
func shortHash(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:8]) }
func fullHash(s string) string  { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }
func setDeprecationHeaders(w http.ResponseWriter, successor string) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Sunset", "Wed, 31 Dec 2027 23:59:59 GMT")
	w.Header().Set("Link", "<"+successor+">; rel=\"successor-version\"")
}
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
