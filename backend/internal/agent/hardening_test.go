package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const currentTestKey = "current-agent-test-key-0123456789-abcdef"
const previousTestKey = "previous-agent-test-key-0123456789-abcdef"

func hardenedTestServer(t *testing.T) *Server {
	t.Helper()
	nonceDir := secureTempDir(t)
	server, err := NewServerWithOptions(ServerOptions{
		KeyRing: KeyRing{
			CurrentKey: currentTestKey, CurrentID: "key-current",
			PreviousKey: previousTestKey, PreviousID: "key-previous",
		},
		Executor:        MockExecutor{},
		Logger:          slog.New(slog.NewTextHandler(os.Stderr, nil)),
		MaxMessageBytes: 64 << 10,
		MaxOutputBytes:  8 << 10,
		MaxConcurrent:   1,
		MaxClockSkew:    time.Minute,
		NonceTTL:        2 * time.Minute,
		NonceCacheLimit: 128,
		NonceStorePath:  filepath.Join(nonceDir, "nonces.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func invokeSigned(t *testing.T, server *Server, key, keyID, nonce, operation string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	httpRequest := signedHTTPRequest(t, key, keyID, nonce, operation, payload)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httpRequest)
	return recorder
}

func signedHTTPRequest(t *testing.T, key, keyID, nonce, operation string, payload map[string]any) *http.Request {
	t.Helper()
	spec, ok := LookupOperation(operation)
	if !ok {
		t.Fatalf("operação ausente: %s", operation)
	}
	request := Request{
		Operation: operation, OperationVersion: spec.Version, RequestID: "req-test-" + nonce,
		Timestamp: time.Now().UTC(), Nonce: nonce, Payload: payload,
	}
	if err := SignWithKey(&request, key, keyID); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	httpRequest := httptest.NewRequest(http.MethodPost, "/v1/execute", bytes.NewReader(body))
	httpRequest.Header.Set("Content-Type", "application/json")
	return httpRequest
}

func TestCatalogIsVersionedAndPolicyComplete(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatal(err)
	}
	for name, spec := range Catalog {
		if spec.Name != name || spec.Operation != name || spec.Version < 1 {
			t.Fatalf("spec inválido para %s: %+v", name, spec)
		}
		if spec.Risk == "" || spec.Access == "" || spec.Rollback == "" || spec.Audit == "" {
			t.Fatalf("política incompleta para %s", name)
		}
	}
	if !Catalog["share.create"].IsMutation() {
		t.Fatal("criação de compartilhamento deve permanecer classificada como mutação")
	}
	if Catalog["system.inspect"].RequiredCapability != "system.inspect" {
		t.Fatal("capability de inventário ausente")
	}
	for operation, capability := range map[string]string{"samba.inspect": "samba.inspect", "cups.inspect": "cups.inspect"} {
		if Catalog[operation].RequiredCapability != capability || Catalog[operation].IsMutation() {
			t.Fatalf("operação de inventário %s possui política incorreta: %+v", operation, Catalog[operation])
		}
	}
}

func TestCurrentAndPreviousHMACKeysAreAccepted(t *testing.T) {
	server := hardenedTestServer(t)
	for _, key := range []struct {
		secret string
		id     string
		nonce  string
	}{
		{currentTestKey, "key-current", "nonce-current-key-0001"},
		{previousTestKey, "key-previous", "nonce-previous-key-0001"},
	} {
		response := invokeSigned(t, server, key.secret, key.id, key.nonce, "system.inspect", map[string]any{"dryRun": true})
		if response.Code != http.StatusOK {
			t.Fatalf("chave %s recusada: status=%d body=%s", key.id, response.Code, response.Body.String())
		}
	}
	unknown := invokeSigned(t, server, currentTestKey, "unknown", "nonce-unknown-key-0001", "system.inspect", nil)
	if unknown.Code != http.StatusUnauthorized {
		t.Fatalf("key id desconhecido deveria ser recusado: %d", unknown.Code)
	}
}

func TestInvalidHMACAndVersionAreRejected(t *testing.T) {
	server := hardenedTestServer(t)
	spec := Catalog["system.inspect"]
	request := Request{
		Operation: "system.inspect", OperationVersion: spec.Version + 1, RequestID: "req-invalid-version",
		Timestamp: time.Now().UTC(), Nonce: "nonce-invalid-version-0001", Payload: map[string]any{},
	}
	if err := SignWithKey(&request, currentTestKey, "key-current"); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/execute", bytes.NewReader(body)))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("versão incompatível deveria retornar 409, recebeu %d", recorder.Code)
	}
	request.OperationVersion = spec.Version
	request.Nonce = "nonce-invalid-hmac-0001"
	request.Signature = "not-a-valid-signature"
	body, err = json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/execute", bytes.NewReader(body)))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("HMAC inválido deveria retornar 401, recebeu %d", recorder.Code)
	}
}

func TestReplayAndSchemaViolationsAreRejected(t *testing.T) {
	server := hardenedTestServer(t)
	nonce := "nonce-replay-check-0001"
	first := invokeSigned(t, server, currentTestKey, "key-current", nonce, "system.inspect", nil)
	if first.Code != http.StatusOK {
		t.Fatalf("primeira chamada recusada: %d %s", first.Code, first.Body.String())
	}
	replay := invokeSigned(t, server, currentTestKey, "key-current", nonce, "system.inspect", nil)
	if replay.Code != http.StatusConflict {
		t.Fatalf("replay deveria retornar 409, recebeu %d", replay.Code)
	}
	invalid := invokeSigned(t, server, currentTestKey, "key-current", "nonce-schema-check-0001", "system.inspect", map[string]any{"unexpected": true})
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("payload fora do schema deveria retornar 422, recebeu %d", invalid.Code)
	}
}

func TestRequiredCapabilityFailsClosed(t *testing.T) {
	for index, operation := range []string{"system.inspect", "samba.inspect", "cups.inspect"} {
		t.Run(operation, func(t *testing.T) {
			server := hardenedTestServer(t)
			server.requireCapabilities = true
			server.capabilities = map[string]bool{}
			nonce := fmt.Sprintf("nonce-capability-check-%04d", index+1)
			response := invokeSigned(t, server, currentTestKey, "key-current", nonce, operation, nil)
			if response.Code != http.StatusServiceUnavailable {
				t.Fatalf("capability de %s ausente deveria retornar 503, recebeu %d", operation, response.Code)
			}
		})
	}
}

func TestMutableOperationIsBlockedBeforeExecutor(t *testing.T) {
	server := hardenedTestServer(t)
	response := invokeSigned(t, server, currentTestKey, "key-current", "nonce-mutation-block-0001", "share.create", map[string]any{
		"share":  map[string]any{"name": "teste"},
		"dryRun": false,
	})
	if response.Code != http.StatusForbidden {
		t.Fatalf("mutação deveria estar bloqueada: %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "mutation_blocked") {
		t.Fatalf("erro esperado não retornado: %s", response.Body.String())
	}
}

func TestMutableFeatureFlagOnlyPermitsMockSimulation(t *testing.T) {
	server := hardenedTestServer(t)
	server.mutableEnabled = true
	response := invokeSigned(t, server, currentTestKey, "key-current", "nonce-mutation-simulation-0001", "share.create", map[string]any{
		"share":  map[string]any{"name": "teste"},
		"dryRun": false,
	})
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "simulated") {
		t.Fatalf("simulação mockada deveria ser explicitamente indicada: %d %s", response.Code, response.Body.String())
	}
}

type blockingExecutor struct {
	started chan struct{}
	release chan struct{}
}

func (e *blockingExecutor) Execute(ctx context.Context, request Request, _ OperationSpec) (Response, error) {
	select {
	case e.started <- struct{}{}:
	default:
	}
	select {
	case <-ctx.Done():
		return Response{RequestID: request.RequestID, Success: false, ErrorCode: "cancelled"}, ctx.Err()
	case <-e.release:
		return Response{RequestID: request.RequestID, Success: true, Summary: "concluída"}, nil
	}
}

func TestConcurrencyLimitRejectsExcessRequest(t *testing.T) {
	server := hardenedTestServer(t)
	blocker := &blockingExecutor{started: make(chan struct{}, 1), release: make(chan struct{})}
	server.exec = blocker
	firstRequest := signedHTTPRequest(t, currentTestKey, "key-current", "nonce-concurrency-first-0001", "system.inspect", nil)
	firstRecorder := httptest.NewRecorder()
	firstDone := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(firstRecorder, firstRequest)
		close(firstDone)
	}()
	select {
	case <-blocker.started:
	case <-time.After(time.Second):
		t.Fatal("primeira execução não iniciou")
	}
	second := invokeSigned(t, server, currentTestKey, "key-current", "nonce-concurrency-second-0001", "system.inspect", nil)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("limite de concorrência deveria retornar 429, recebeu %d", second.Code)
	}
	close(blocker.release)
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("primeira execução não finalizou")
	}
}

func TestMessageLimitIsEnforcedBeforeJSONDecode(t *testing.T) {
	server := hardenedTestServer(t)
	server.maxMessageBytes = 32
	request := httptest.NewRequest(http.MethodPost, "/v1/execute", strings.NewReader(strings.Repeat("x", 64)))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("mensagem excessiva deveria retornar 413, recebeu %d", recorder.Code)
	}
}

func TestNonceStorePersistsReplayProtection(t *testing.T) {
	path := filepath.Join(secureTempDir(t), "nonces.json")
	store, err := NewNonceStore(path, time.Minute, 128)
	if err != nil {
		t.Fatal(err)
	}
	nonce := "nonce-persistent-check-0001"
	if err = store.Use(nonce, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissão do arquivo de nonce insegura: %o", info.Mode().Perm())
	}
	reloaded, err := NewNonceStore(path, time.Minute, 128)
	if err != nil {
		t.Fatal(err)
	}
	if err = reloaded.Use(nonce, time.Now().UTC()); err == nil {
		t.Fatal("nonce persistido foi aceito após reinício")
	}
}

type recordingRunner struct {
	command commandSpec
}

type inventoryRunner struct {
	results map[string]commandResult
}

func (r inventoryRunner) Run(_ context.Context, command commandSpec, _ int) (commandResult, error) {
	key := command.path + " " + strings.Join(command.args, " ")
	result, ok := r.results[key]
	if !ok {
		return commandResult{}, errors.New("sonda ausente")
	}
	return result, nil
}

func TestReadOnlyExecutorReturnsStructuredSambaAndCupsInventory(t *testing.T) {
	runner := inventoryRunner{results: map[string]commandResult{
		"/usr/local/sbin/smbd -V":  {stdout: "Version 4.23.8"},
		"/usr/local/sbin/cupsd -t": {stdout: "configuração válida"},
		"/usr/local/bin/lpstat -p": {stdout: ""},
		"/usr/local/bin/lpstat -v": {stdout: ""},
	}}
	executor := newReadOnlyExecutorForTest(runner, 4096)
	for _, testCase := range []struct {
		operation string
		field     string
	}{
		{operation: "samba.inspect", field: "samba"},
		{operation: "cups.inspect", field: "cups"},
	} {
		response, err := executor.Execute(context.Background(), Request{Operation: testCase.operation, RequestID: "req-" + testCase.field}, Catalog[testCase.operation])
		if err != nil || !response.Success || response.Data[testCase.field] == nil {
			t.Fatalf("inventário %s não atravessou o executor estruturado: response=%+v err=%v", testCase.operation, response, err)
		}
	}
}

func (r *recordingRunner) Run(_ context.Context, command commandSpec, _ int) (commandResult, error) {
	r.command = command
	return commandResult{stdout: "password=topsecret", exitCode: 0}, nil
}

func TestReadOnlyExecutorUsesFixedCommandAndRedactsOutput(t *testing.T) {
	runner := &recordingRunner{}
	executor := newReadOnlyExecutorForTest(runner, 1024)
	request := Request{Operation: "samba.config.validate", RequestID: "req-readonly"}
	response, err := executor.Execute(context.Background(), request, Catalog["samba.config.validate"])
	if err != nil || !response.Success {
		t.Fatalf("executor falhou: response=%+v err=%v", response, err)
	}
	if runner.command.path != "/usr/local/bin/testparm" || len(runner.command.args) != 1 || runner.command.args[0] != "-s" {
		t.Fatalf("comando não foi fixado pela política: %+v", runner.command)
	}
	if strings.Contains(response.Stdout, "topsecret") || !strings.Contains(response.Stdout, "[REDACTED]") {
		t.Fatalf("saída sensível não foi mascarada: %q", response.Stdout)
	}
	response, err = executor.Execute(context.Background(), Request{Operation: "share.create", RequestID: "req-write"}, Catalog["share.create"])
	if !errors.Is(err, ErrMutationBlocked) || response.ErrorCode != "mutation_blocked" {
		t.Fatalf("executor aceitou mutação: response=%+v err=%v", response, err)
	}
}

func TestResponseDataRedactsSensitiveFields(t *testing.T) {
	data := redactResponseData(map[string]any{
		"password": "topsecret",
		"nested":   map[string]any{"token": "abc", "safe": "value"},
	}, Catalog["system.inspect"])
	if data["password"] != "[REDACTED]" {
		t.Fatalf("senha não foi mascarada: %#v", data)
	}
	nested, ok := data["nested"].(map[string]any)
	if !ok || nested["token"] != "[REDACTED]" || nested["safe"] != "value" {
		t.Fatalf("dados aninhados não foram mascarados corretamente: %#v", data)
	}
}

func TestSocketPolicyRejectsRegularFileAndAppliesMode(t *testing.T) {
	dir := secureTempDir(t)
	regular := filepath.Join(dir, "not-a-socket")
	if err := os.WriteFile(regular, []byte("do not remove"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ListenWithOptions(regular, SocketOptions{Mode: 0o600, ExpectedPeerUID: -1}); err == nil {
		t.Fatal("arquivo regular não pode ser removido para criar socket")
	}
	if _, err := os.Stat(regular); err != nil {
		t.Fatalf("arquivo regular foi removido: %v", err)
	}
	socket := filepath.Join(dir, "agent.sock")
	listener, err := ListenWithOptions(socket, SocketOptions{Mode: 0o600, ExpectedPeerUID: -1})
	if err != nil {
		if errors.Is(err, syscall.EPERM) {
			t.Skip("sandbox não permite criar socket Unix")
		}
		t.Fatal(err)
	}
	defer listener.Close()
	info, err := os.Stat(socket)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("modo do socket incorreto: %o", info.Mode().Perm())
	}
}

func TestPrepareSocketDirectoryKeepsRootControlledSetgidPolicy(t *testing.T) {
	dir := filepath.Join(secureTempDir(t), "run")
	if err := PrepareSocketDirectory(dir, "", ""); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o750 || info.Mode()&os.ModeSetgid == 0 {
		t.Fatalf("modo do diretório do socket incorreto: %v", info.Mode())
	}
}

func TestPrepareSocketDirectoryRejectsSymlinkWithoutChangingTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "socket-dir")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := PrepareSocketDirectory(link, "", ""); err == nil {
		t.Fatal("symlink foi aceito como diretório do socket")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("modo do alvo foi alterado: %o", info.Mode().Perm())
	}
}

func secureTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}
