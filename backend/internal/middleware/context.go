package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/auth"
	"github.com/hu-ufcat/samba-admin-backend/internal/config"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
	"github.com/hu-ufcat/samba-admin-backend/internal/rbac"
)

type key string

const correlationKey key = "correlation"
const userKey key = "user"
const csrfKey key = "csrf"

func CorrelationID(ctx context.Context) string    { v, _ := ctx.Value(correlationKey).(string); return v }
func CurrentUser(ctx context.Context) models.User { v, _ := ctx.Value(userKey).(models.User); return v }
func CSRFToken(ctx context.Context) string        { v, _ := ctx.Value(csrfKey).(string); return v }

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
		if !validCorrelationID(id) {
			b := make([]byte, 12)
			_, _ = rand.Read(b)
			id = "corr-" + hex.EncodeToString(b)
		}
		w.Header().Set("X-Correlation-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), correlationKey, id)))
	})
}
func validCorrelationID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, c := range value {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			continue
		}
		return false
	}
	return true
}
func Recover(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				logger.Error("panic recuperado", "error", v, "stack", string(debug.Stack()), "correlationId", CorrelationID(r.Context()))
				http.Error(w, "falha interna", 500)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
func AccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start), "correlationId", CorrelationID(r.Context()))
	})
}
func BodyLimit(n int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, n)
		}
		next.ServeHTTP(w, r)
	})
}
func TimeoutExcept(d time.Duration, excludedPath string, next http.Handler) http.Handler {
	if d <= 0 {
		return next
	}
	timed := http.TimeoutHandler(next, d, `{"type":"timeout","title":"Tempo limite excedido","status":503,"detail":"A requisição excedeu o limite operacional."}`)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == excludedPath {
			next.ServeHTTP(w, r)
			return
		}
		timed.ServeHTTP(w, r)
	})
}
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
func CORS(origins []string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, X-Correlation-ID, Idempotency-Key, If-Match")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func Authenticate(cfg config.Config, svc *auth.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if cfg.AuthMode == "development-bypass" {
			u := models.User{ID: "dev-admin", Username: "dev-admin", DisplayName: "Administrador de desenvolvimento", Roles: []string{"administrador do sistema"}}
			ctx := context.WithValue(r.Context(), userKey, u)
			ctx = context.WithValue(ctx, csrfKey, "development-csrf")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		c, err := r.Cookie("samba_admin_session")
		if err != nil {
			writeProblem(w, r, http.StatusUnauthorized, "session_expired", "Sessão inválida", "Autentique-se novamente.")
			return
		}
		u, csrf, err := svc.Current(r.Context(), c.Value)
		if err != nil {
			writeProblem(w, r, http.StatusUnauthorized, "session_expired", "Sessão inválida", "Autentique-se novamente.")
			return
		}
		ctx := context.WithValue(r.Context(), userKey, u)
		ctx = context.WithValue(ctx, csrfKey, csrf)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func RequireCSRF(cfg config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions || r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/mfa/verify" {
			next.ServeHTTP(w, r)
			return
		}
		if cfg.AuthMode == "development-bypass" {
			next.ServeHTTP(w, r)
			return
		}
		if subtleEqual(r.Header.Get("X-CSRF-Token"), CSRFToken(r.Context())) {
			next.ServeHTTP(w, r)
			return
		}
		writeProblem(w, r, http.StatusForbidden, "csrf_failed", "Proteção CSRF", "O token CSRF está ausente ou inválido.")
	})
}
func Authorize(engine *rbac.Engine, action, resource string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if engine.Allowed(CurrentUser(r.Context()).Roles, action, resource) {
			next.ServeHTTP(w, r)
			return
		}
		writeProblem(w, r, http.StatusForbidden, "permission_denied", "Permissão insuficiente", "O papel atual não autoriza esta operação.")
	})
}
func publicPath(p string) bool {
	return p == "/healthz" || p == "/readyz" || p == "/api/v1/auth/login" || p == "/api/v1/auth/mfa/verify"
}
func subtleEqual(a, b string) bool {
	if len(a) != len(b) || a == "" {
		return false
	}
	var v byte
	for i := range []byte(a) {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, typ, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": typ, "title": title, "status": status, "detail": detail, "correlationId": CorrelationID(r.Context())})
}
