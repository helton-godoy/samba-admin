package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Executor interface {
	Execute(context.Context, Request, OperationSpec) (Response, error)
}

// MockExecutor remains useful for deterministic API tests. The Server still
// rejects mutable operations before this executor can observe them.
type MockExecutor struct{ Scenario string }

func (m MockExecutor) Execute(ctx context.Context, r Request, s OperationSpec) (Response, error) {
	if m.Scenario == "partial-failure" {
		return Response{RequestID: r.RequestID, Success: false, ExitCode: 1, Summary: "Falha parcial simulada", ErrorCode: "partial_failure"}, errors.New("falha parcial")
	}
	select {
	case <-ctx.Done():
		return Response{RequestID: r.RequestID, Success: false, Summary: "Timeout", ErrorCode: "timeout"}, ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}
	if s.IsMutation() {
		return Response{RequestID: r.RequestID, Success: true, Summary: "Operação mutável simulada; nenhuma escrita no sistema foi realizada.", Data: map[string]any{"operation": r.Operation, "simulated": true}}, nil
	}
	return Response{RequestID: r.RequestID, Success: true, Summary: "Operação somente leitura mockada concluída", Data: map[string]any{"operation": r.Operation}}, nil
}

type ServerOptions struct {
	KeyRing                 KeyRing
	Executor                Executor
	Logger                  *slog.Logger
	MaxMessageBytes         int64
	MaxOutputBytes          int
	MaxConcurrent           int
	MaxClockSkew            time.Duration
	NonceTTL                time.Duration
	NonceCacheLimit         int
	NonceStorePath          string
	RequireCapabilities     bool
	EnabledCapabilities     map[string]bool
	EnableMutableOperations bool
}

type Server struct {
	keys                KeyRing
	exec                Executor
	logger              *slog.Logger
	nonces              *NonceStore
	maxMessageBytes     int64
	maxOutputBytes      int
	maxClockSkew        time.Duration
	semaphore           chan struct{}
	requireCapabilities bool
	capabilities        map[string]bool
	mutableEnabled      bool
	mu                  sync.RWMutex
}

func NewServer(key string, exec Executor, logger *slog.Logger) *Server {
	server, err := NewServerWithOptions(ServerOptions{
		KeyRing:         KeyRing{CurrentKey: key, CurrentID: "legacy"},
		Executor:        exec,
		Logger:          logger,
		MaxMessageBytes: 1 << 20,
		MaxOutputBytes:  128 << 10,
		MaxConcurrent:   4,
		MaxClockSkew:    2 * time.Minute,
		NonceTTL:        5 * time.Minute,
		NonceCacheLimit: 4096,
	})
	if err != nil {
		panic(fmt.Sprintf("configuração inválida do agente: %v", err))
	}
	return server
}

func NewServerWithOptions(options ServerOptions) (*Server, error) {
	if err := ValidateCatalog(); err != nil {
		return nil, err
	}
	if err := options.KeyRing.Validate(); err != nil {
		return nil, err
	}
	if options.Executor == nil {
		return nil, errors.New("executor do agente é obrigatório")
	}
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	if options.MaxMessageBytes <= 0 || options.MaxMessageBytes > 8<<20 {
		return nil, errors.New("limite de mensagem do agente deve estar entre 1 e 8 MiB")
	}
	if options.MaxOutputBytes <= 0 || options.MaxOutputBytes > 4<<20 {
		return nil, errors.New("limite de saída do agente deve estar entre 1 e 4 MiB")
	}
	if options.MaxConcurrent < 1 || options.MaxConcurrent > 64 {
		return nil, errors.New("limite de concorrência do agente deve estar entre 1 e 64")
	}
	if options.MaxClockSkew <= 0 || options.MaxClockSkew > 10*time.Minute {
		return nil, errors.New("janela de validade do agente inválida")
	}
	if options.NonceTTL < options.MaxClockSkew || options.NonceTTL > 24*time.Hour {
		return nil, errors.New("TTL de nonce deve cobrir a janela de validade e ser inferior a 24 horas")
	}
	nonces, err := NewNonceStore(options.NonceStorePath, options.NonceTTL, options.NonceCacheLimit)
	if err != nil {
		return nil, err
	}
	capabilities := make(map[string]bool, len(options.EnabledCapabilities))
	for name, enabled := range options.EnabledCapabilities {
		if enabled {
			capabilities[name] = true
		}
	}
	return &Server{
		keys: options.KeyRing, exec: options.Executor, logger: options.Logger,
		nonces: nonces, maxMessageBytes: options.MaxMessageBytes,
		maxOutputBytes: options.MaxOutputBytes, maxClockSkew: options.MaxClockSkew,
		semaphore:           make(chan struct{}, options.MaxConcurrent),
		requireCapabilities: options.RequireCapabilities, capabilities: capabilities,
		mutableEnabled: options.EnableMutableOperations,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/execute", s.handle)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		write(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	return mux
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	request, err := s.decodeRequest(w, r)
	if err != nil {
		return
	}
	if err = validateEnvelope(request); err != nil {
		write(w, http.StatusBadRequest, Response{RequestID: request.RequestID, Success: false, Summary: "Envelope de agente inválido", ErrorCode: "invalid_request"})
		return
	}
	if err = VerifyWithKeyRing(request, s.keys, s.maxClockSkew); err != nil {
		write(w, http.StatusUnauthorized, Response{RequestID: request.RequestID, Success: false, Summary: "Autenticação do agente recusada", ErrorCode: "unauthorized"})
		return
	}
	spec, ok := LookupOperation(request.Operation)
	if !ok {
		write(w, http.StatusForbidden, Response{RequestID: request.RequestID, Success: false, Summary: "Operação não permitida", ErrorCode: "operation_not_allowed"})
		return
	}
	if request.OperationVersion != spec.Version {
		write(w, http.StatusConflict, Response{RequestID: request.RequestID, Success: false, Summary: "Versão de operação incompatível", ErrorCode: "operation_version_mismatch"})
		return
	}
	if !s.capabilityEnabled(spec.RequiredCapability) {
		write(w, http.StatusServiceUnavailable, Response{RequestID: request.RequestID, Success: false, Summary: "Capability necessária não está habilitada", ErrorCode: "capability_unavailable"})
		return
	}
	if err = spec.Payload.Validate(request.Payload); err != nil {
		write(w, http.StatusUnprocessableEntity, Response{RequestID: request.RequestID, Success: false, Summary: "Payload da operação inválido", ErrorCode: "invalid_payload"})
		return
	}
	if spec.IsMutation() && !s.mutableEnabled {
		write(w, http.StatusForbidden, Response{RequestID: request.RequestID, Success: false, Summary: "Operações mutáveis permanecem bloqueadas até homologação", ErrorCode: "mutation_blocked"})
		return
	}
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	default:
		write(w, http.StatusTooManyRequests, Response{RequestID: request.RequestID, Success: false, Summary: "Limite de concorrência atingido", ErrorCode: "concurrency_limited"})
		return
	}
	if err = s.nonces.Use(request.Nonce, time.Now().UTC()); err != nil {
		if errors.Is(err, ErrNonceReplay) {
			write(w, http.StatusConflict, Response{RequestID: request.RequestID, Success: false, Summary: "Nonce já utilizado", ErrorCode: "replay"})
			return
		}
		write(w, http.StatusServiceUnavailable, Response{RequestID: request.RequestID, Success: false, Summary: "Proteção contra replay indisponível", ErrorCode: "replay_store_unavailable"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(spec.TimeoutSeconds)*time.Second)
	defer cancel()
	response, execErr := s.exec.Execute(ctx, request, spec)
	response.RequestID = request.RequestID
	response.Summary = truncateAndRedact(response.Summary, 4096)
	response.Stdout = truncateAndRedact(response.Stdout, s.maxOutputBytes)
	response.Stderr = truncateAndRedact(response.Stderr, s.maxOutputBytes)
	response.Data = redactResponseData(response.Data, spec)
	response.ErrorCode = normalizeErrorCode(response.ErrorCode)
	if execErr != nil && response.ErrorCode == "" {
		response.Success = false
		response.ErrorCode = "execution_failed"
		response.Summary = "Falha na execução controlada"
	}
	if response.Summary == "" {
		response.Success = false
		response.ErrorCode = "invalid_executor_response"
		response.Summary = "Executor retornou resposta inválida"
	}
	s.logger.Info("operação do agente concluída", "operation", request.Operation, "version", request.OperationVersion, "keyId", request.KeyID, "requestId", request.RequestID, "success", response.Success, "errorCode", response.ErrorCode)
	write(w, http.StatusOK, response)
}

func (s *Server) decodeRequest(w http.ResponseWriter, r *http.Request) (Request, error) {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		write(w, http.StatusUnsupportedMediaType, Response{Success: false, Summary: "Content-Type deve ser application/json", ErrorCode: "unsupported_media_type"})
		return Request{}, errors.New("content type inválido")
	}
	if r.ContentLength > s.maxMessageBytes {
		write(w, http.StatusRequestEntityTooLarge, Response{Success: false, Summary: "Mensagem excede o limite permitido", ErrorCode: "message_too_large"})
		return Request{}, errors.New("mensagem muito grande")
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.maxMessageBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request Request
	if err := decoder.Decode(&request); err != nil {
		if isMessageTooLarge(err) {
			write(w, http.StatusRequestEntityTooLarge, Response{Success: false, Summary: "Mensagem excede o limite permitido", ErrorCode: "message_too_large"})
		} else {
			write(w, http.StatusBadRequest, Response{Success: false, Summary: "JSON inválido", ErrorCode: "invalid_json"})
		}
		return Request{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		write(w, http.StatusBadRequest, Response{RequestID: request.RequestID, Success: false, Summary: "JSON deve conter uma única mensagem", ErrorCode: "invalid_json"})
		return Request{}, errors.New("JSON com dados extras")
	}
	if request.Payload == nil {
		request.Payload = map[string]any{}
	}
	return request, nil
}

func validateEnvelope(request Request) error {
	if strings.TrimSpace(request.Operation) == "" || strings.TrimSpace(request.RequestID) == "" || strings.TrimSpace(request.Signature) == "" {
		return errors.New("campos obrigatórios ausentes")
	}
	if len(request.RequestID) > 128 || len(request.Operation) > 128 || len(request.KeyID) > 64 {
		return errors.New("campos de envelope excedem o limite")
	}
	return validateNonce(request.Nonce)
}

func (s *Server) capabilityEnabled(capability string) bool {
	if capability == "" || !s.requireCapabilities {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.capabilities[capability]
}

func isMessageTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}

func truncateAndRedact(value string, limit int) string {
	value = redactOutput(value)
	if len(value) <= limit {
		return value
	}
	marker := "\n[TRUNCATED]"
	if limit <= len(marker) {
		return marker[:limit]
	}
	return value[:limit-len(marker)] + marker
}

func redactResponseData(data map[string]any, spec OperationSpec) map[string]any {
	if data == nil {
		return nil
	}
	redacted := make(map[string]any, len(data))
	for key, value := range data {
		if spec.SensitiveField(key) || sensitiveDataKey(key) {
			redacted[key] = "[REDACTED]"
			continue
		}
		redacted[key] = redactNestedData(value)
	}
	return redacted
}

func redactNestedData(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for key, child := range typed {
			if sensitiveDataKey(key) {
				redacted[key] = "[REDACTED]"
			} else {
				redacted[key] = redactNestedData(child)
			}
		}
		return redacted
	case []any:
		redacted := make([]any, len(typed))
		for index, child := range typed {
			redacted[index] = redactNestedData(child)
		}
		return redacted
	case string:
		return redactOutput(typed)
	default:
		return value
	}
}

func sensitiveDataKey(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(key, "_", ""))
	for _, sensitive := range []string{"password", "passwd", "secret", "token", "credential", "authorization", "cookie", "keytab", "privatekey", "totp", "otp", "recoverycode"} {
		if strings.Contains(key, sensitive) {
			return true
		}
	}
	return false
}

func normalizeErrorCode(value string) string {
	if value == "" {
		return ""
	}
	if len(value) > 80 {
		return "execution_failed"
	}
	for _, character := range value {
		if !(character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '_' || character == '-' || character == '.') {
			return "execution_failed"
		}
	}
	return value
}

func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
