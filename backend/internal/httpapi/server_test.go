package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	agentadapter "github.com/hu-ufcat/samba-admin-backend/internal/adapters/agent"
	"github.com/hu-ufcat/samba-admin-backend/internal/agent"
	"github.com/hu-ufcat/samba-admin-backend/internal/audit"
	"github.com/hu-ufcat/samba-admin-backend/internal/auth"
	"github.com/hu-ufcat/samba-admin-backend/internal/config"
	"github.com/hu-ufcat/samba-admin-backend/internal/events"
	"github.com/hu-ufcat/samba-admin-backend/internal/jobs"
	"github.com/hu-ufcat/samba-admin-backend/internal/rbac"
	"github.com/hu-ufcat/samba-admin-backend/internal/storage"
)

type readOnlyAgentClient struct {
	failures map[string]error
}

func (c readOnlyAgentClient) Execute(_ context.Context, operation string, _ map[string]any) (agent.Response, error) {
	if err := c.failures[operation]; err != nil {
		return agent.Response{}, err
	}
	switch operation {
	case "system.inspect":
		return agent.Response{Success: true, Data: map[string]any{"system": map[string]any{
			"hostname": "lab", "fqdn": "lab.invalid", "freebsdVersion": "15.1-RELEASE",
		}}}, nil
	case "samba.inspect":
		return agent.Response{Success: true, Data: map[string]any{"samba": map[string]any{
			"version": "4.23.8", "configurationPath": "/usr/local/etc/smb4.conf", "profile": "domain-member",
			"binaries": []any{"/usr/local/sbin/smbd"}, "shares": []any{}, "vfsModules": []any{}, "packageBuildOptions": []any{},
		}}}, nil
	case "cups.inspect":
		return agent.Response{Success: true, Data: map[string]any{"cups": map[string]any{
			"service": "valid", "printers": []any{}, "backends": []any{}, "ppds": []any{}, "errors": []any{},
		}}}, nil
	default:
		return agent.Response{Success: true, Data: map[string]any{}}, nil
	}
}

func testServer(t *testing.T) (*httptest.Server, *storage.Store) {
	t.Helper()
	handler, store := testHandler(t)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		store.Close()
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			t.Skip("sandbox não permite abrir listener TCP de loopback")
		}
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	return server, store
}

func testHandler(t *testing.T) (http.Handler, *storage.Store) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{AuthMode: "development-bypass", AdapterMode: "mock", AllowedOrigins: []string{"http://localhost:5173"}, MaxBodyBytes: 2 << 20, SessionTTL: time.Hour, DevScenario: "success"}
	broker := events.New()
	auditSvc := audit.New(store)
	jobMgr := jobs.New(store, broker, agent.MockClient{}, auditSvc)
	s := New(cfg, store, auth.New(store, time.Hour), rbac.New(), jobMgr, broker, auditSvc, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	return s.Router(), store
}

func testReadOnlyHandler(t *testing.T, client agent.Client) (http.Handler, *storage.Store) {
	t.Helper()
	server, store := testReadOnlyServer(t, client)
	return server.Router(), store
}

func testReadOnlyServer(t *testing.T, client agent.Client) (*Server, *storage.Store) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "api-readonly.db"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{AuthMode: "development-bypass", AdapterMode: "agent", AllowedOrigins: []string{"http://localhost:5173"}, MaxBodyBytes: 2 << 20, SessionTTL: time.Hour, DevScenario: "success"}
	broker := events.New()
	auditSvc := audit.New(store)
	jobMgr := jobs.New(store, broker, client, auditSvc)
	s := New(cfg, store, auth.New(store, time.Hour), rbac.New(), jobMgr, broker, auditSvc, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	provider, err := agentadapter.New(client)
	if err != nil {
		store.Close()
		t.Fatal(err)
	}
	s.SetReadOnlyProvider(provider)
	return s, store
}

func TestReadinessChecksAgentInRealMode(t *testing.T) {
	for _, tc := range []struct {
		name       string
		client     readOnlyAgentClient
		wantStatus int
	}{
		{name: "agente disponível", client: readOnlyAgentClient{failures: map[string]error{}}, wantStatus: http.StatusOK},
		{name: "agente indisponível", client: readOnlyAgentClient{failures: map[string]error{"system.inspect": errors.New("socket indisponível")}}, wantStatus: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, store := testReadOnlyHandler(t, tc.client)
			defer store.Close()
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if recorder.Code != tc.wantStatus {
				t.Fatalf("readiness retornou %d, esperado %d: %s", recorder.Code, tc.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestPrintersReturnsUnavailableWhenAgentFails(t *testing.T) {
	handler, store := testReadOnlyHandler(t, readOnlyAgentClient{failures: map[string]error{"cups.inspect": errors.New("socket indisponível")}})
	defer store.Close()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/printers", nil))
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "capability_unavailable") {
		t.Fatalf("falha CUPS foi mascarada: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSambaAndCupsUseAgentProvider(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		path      string
		operation string
		serve     func(*Server, http.ResponseWriter, *http.Request)
		wantField string
	}{
		{name: "Samba", path: "/api/v1/samba", operation: "samba.inspect", serve: func(server *Server, writer http.ResponseWriter, request *http.Request) { server.samba(writer, request) }, wantField: `"version":"4.23.8"`},
		{name: "CUPS", path: "/api/v1/cups", operation: "cups.inspect", serve: func(server *Server, writer http.ResponseWriter, request *http.Request) { server.cups(writer, request) }, wantField: `"service":"valid"`},
	} {
		t.Run(testCase.name+" disponível", func(t *testing.T) {
			server, store := testReadOnlyServer(t, readOnlyAgentClient{failures: map[string]error{}})
			defer store.Close()
			recorder := httptest.NewRecorder()
			testCase.serve(server, recorder, httptest.NewRequest(http.MethodGet, testCase.path, nil))
			if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), testCase.wantField) {
				t.Fatalf("inventário %s inválido: status=%d body=%s", testCase.name, recorder.Code, recorder.Body.String())
			}
		})
		t.Run(testCase.name+" indisponível", func(t *testing.T) {
			server, store := testReadOnlyServer(t, readOnlyAgentClient{failures: map[string]error{testCase.operation: errors.New("socket indisponível")}})
			defer store.Close()
			recorder := httptest.NewRecorder()
			testCase.serve(server, recorder, httptest.NewRequest(http.MethodGet, testCase.path, nil))
			if recorder.Code != http.StatusServiceUnavailable || !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/problem+json") || !strings.Contains(recorder.Body.String(), "capability_unavailable") {
				t.Fatalf("falha %s não retornou RFC 7807/503: status=%d body=%s", testCase.name, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestGeneratedContractRouter(t *testing.T) {
	handler, store := testHandler(t)
	defer store.Close()

	legacy := httptest.NewRecorder()
	handler.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, "/api/v1/acl", nil))
	if legacy.Code != http.StatusOK {
		t.Fatalf("alias legado retornou %d: %s", legacy.Code, legacy.Body.String())
	}
	if legacy.Header().Get("Deprecation") != "true" || legacy.Header().Get("Link") == "" {
		t.Fatalf("cabeçalhos de depreciação ausentes: %v", legacy.Header())
	}

	missingIdempotency := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shares", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(missingIdempotency, req)
	if missingIdempotency.Code != http.StatusBadRequest {
		t.Fatalf("Idempotency-Key ausente deveria retornar 400, recebeu %d", missingIdempotency.Code)
	}
	if !strings.HasPrefix(missingIdempotency.Header().Get("Content-Type"), "application/problem+json") {
		t.Fatalf("erro de contrato não usa RFC 7807: %q", missingIdempotency.Header().Get("Content-Type"))
	}
}

func TestSharePreviewRejectsConfigurationInjection(t *testing.T) {
	handler, store := testHandler(t)
	defer store.Close()
	payload := `{"id":"pending","name":"Seguro","description":"linha válida\nadmin users = invasor","path":"/srv/dados/seguro","enabled":true,"readOnly":false,"guestAccess":false,"allowedPrincipals":["EBSERHNET\\\\Domain Users"],"deniedPrincipals":[],"encryption":"desired","signing":"mandatory","auditProfile":"security","recycleBin":false,"dfs":false,"vfsModules":["acl_xattr"],"maxConnections":10,"aclModel":"nfsv4"}`
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shares/preview", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("injeção de diretiva Samba deveria retornar 422, recebeu %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestIdempotencyReservationReplaysWithoutDuplicatingShare(t *testing.T) {
	handler, store := testHandler(t)
	defer store.Close()
	payload := `{"id":"pending","name":"Reserva","description":"Teste de reserva idempotente","path":"/srv/dados/reserva","enabled":true,"readOnly":false,"guestAccess":false,"allowedPrincipals":["EBSERHNET\\\\Domain Users"],"deniedPrincipals":[],"encryption":"desired","signing":"mandatory","auditProfile":"security","recycleBin":false,"dfs":false,"vfsModules":["acl_xattr"],"maxConnections":10,"aclModel":"nfsv4"}`
	do := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/shares", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "share-reservation-001")
		handler.ServeHTTP(recorder, req)
		return recorder
	}
	first := do(payload)
	if first.Code != http.StatusCreated {
		t.Fatalf("primeira resposta inesperada: %d %s", first.Code, first.Body.String())
	}
	replay := do(payload)
	if replay.Code != http.StatusCreated || replay.Header().Get("Idempotency-Replayed") != "true" || replay.Body.String() != first.Body.String() {
		t.Fatalf("replay divergente: status=%d replay=%q body=%s", replay.Code, replay.Header().Get("Idempotency-Replayed"), replay.Body.String())
	}
	conflict := do(strings.Replace(payload, "Teste de reserva idempotente", "Conteúdo diferente", 1))
	if conflict.Code != http.StatusConflict {
		t.Fatalf("reuso com payload diferente deveria retornar 409, recebeu %d", conflict.Code)
	}
}
func TestFrontendCompatibilityEndpoints(t *testing.T) {
	srv, store := testServer(t)
	defer srv.Close()
	defer store.Close()
	for _, path := range []string{"/api/v1/system", "/api/v1/samba", "/api/v1/cups", "/api/v1/filesystems", "/api/v1/shares", "/api/v1/acl", "/api/v1/principals", "/api/v1/domain", "/api/v1/printers", "/api/v1/print-drivers", "/api/v1/dfs", "/api/v1/quotas", "/api/v1/services", "/api/v1/audit", "/api/v1/jobs", "/api/v1/capabilities"} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Errorf("%s retornou %d", path, res.StatusCode)
		}
	}
}
func TestSharePreviewAndPersistentJob(t *testing.T) {
	srv, store := testServer(t)
	defer srv.Close()
	defer store.Close()
	share := map[string]any{"id": "pending", "name": "Projetos", "description": "Documentos de projetos institucionais", "path": "/srv/dados/projetos", "enabled": true, "readOnly": false, "guestAccess": false, "allowedPrincipals": []string{"EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL"}, "deniedPrincipals": []string{}, "encryption": "desired", "signing": "mandatory", "auditProfile": "security", "recycleBin": true, "dfs": false, "vfsModules": []string{"acl_xattr", "recycle", "full_audit"}, "maxConnections": 200, "aclModel": "nfsv4"}
	b, _ := json.Marshal(share)
	res, err := http.Post(srv.URL+"/api/v1/shares/preview", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	preview, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !bytes.Contains(preview, []byte("testparm")) {
		t.Fatalf("preview status=%d body=%s", res.StatusCode, preview)
	}
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/shares", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "share-persistent-job-001")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 201 {
		t.Fatalf("create status=%d body=%s", res.StatusCode, body)
	}
	var payload struct {
		Job struct {
			ID string `json:"id"`
		} `json:"job"`
	}
	if err = json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Job.ID == "" {
		t.Fatal("job não retornado")
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		j, err := store.GetJob(context.Background(), payload.Job.ID)
		if err == nil && j.Status == "success" {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	j, _ := store.GetJob(context.Background(), payload.Job.ID)
	t.Fatalf("job não concluiu: %+v", j)
}
func TestPathTraversalRejected(t *testing.T) {
	srv, store := testServer(t)
	defer srv.Close()
	defer store.Close()
	payload := `{"id":"pending","name":"Invalido","description":"Teste inválido","path":"/srv/dados/../../root","enabled":true,"readOnly":false,"guestAccess":false,"allowedPrincipals":[],"deniedPrincipals":[],"encryption":"desired","signing":"mandatory","auditProfile":"off","recycleBin":false,"dfs":false,"vfsModules":[],"maxConnections":10,"aclModel":"nfsv4"}`
	res, err := http.Post(srv.URL+"/api/v1/shares/preview", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 422 {
		t.Fatalf("esperado 422, obtido %d", res.StatusCode)
	}
}
func TestSSEConnects(t *testing.T) {
	srv, store := testServer(t)
	defer srv.Close()
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/v1/events", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	buf := make([]byte, 128)
	n, err := res.Body.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(buf[:n]), "event: ready") {
		t.Fatalf("SSE inesperado: %s", buf[:n])
	}
}

func TestShareCreationIdempotency(t *testing.T) {
	srv, store := testServer(t)
	defer srv.Close()
	defer store.Close()
	share := map[string]any{"id": "pending", "name": "Idempotente", "description": "Teste de idempotência", "path": "/srv/dados/idempotente", "enabled": true, "readOnly": false, "guestAccess": false, "allowedPrincipals": []string{"EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL"}, "deniedPrincipals": []string{}, "encryption": "desired", "signing": "mandatory", "auditProfile": "security", "recycleBin": true, "dfs": false, "vfsModules": []string{"acl_xattr"}, "maxConnections": 100, "aclModel": "nfsv4"}
	body, _ := json.Marshal(share)
	do := func(payload []byte) (*http.Response, []byte) {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/shares", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "share-create-20260711-001")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		return res, b
	}
	first, firstBody := do(body)
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("primeira criação: status=%d body=%s", first.StatusCode, firstBody)
	}
	second, secondBody := do(body)
	if second.StatusCode != http.StatusCreated || second.Header.Get("Idempotency-Replayed") != "true" {
		t.Fatalf("replay inesperado: status=%d replay=%q body=%s", second.StatusCode, second.Header.Get("Idempotency-Replayed"), secondBody)
	}
	if !bytes.Equal(firstBody, secondBody) {
		t.Fatalf("resposta idempotente divergente\nprimeira=%s\nsegunda=%s", firstBody, secondBody)
	}
	share["description"] = "conteúdo diferente"
	changed, _ := json.Marshal(share)
	conflict, conflictBody := do(changed)
	if conflict.StatusCode != http.StatusConflict {
		t.Fatalf("reuso com payload diferente deveria retornar 409: status=%d body=%s", conflict.StatusCode, conflictBody)
	}
}

func TestJobCanBeCancelled(t *testing.T) {
	srv, store := testServer(t)
	defer srv.Close()
	defer store.Close()
	share := map[string]any{"id": "pending", "name": "Cancelar", "description": "Teste de cancelamento", "path": "/srv/dados/cancelar", "enabled": true, "readOnly": false, "guestAccess": false, "allowedPrincipals": []string{"EBSERHNET\\Domain Users"}, "deniedPrincipals": []string{}, "encryption": "desired", "signing": "mandatory", "auditProfile": "security", "recycleBin": false, "dfs": false, "vfsModules": []string{"acl_xattr"}, "maxConnections": 10, "aclModel": "nfsv4"}
	body, _ := json.Marshal(share)
	createRequest, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/shares", bytes.NewReader(body))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Idempotency-Key", "share-cancel-job-001")
	res, err := http.DefaultClient.Do(createRequest)
	if err != nil {
		t.Fatal(err)
	}
	created, _ := io.ReadAll(res.Body)
	res.Body.Close()
	var payload struct {
		Job struct {
			ID string `json:"id"`
		} `json:"job"`
	}
	if err = json.Unmarshal(created, &payload); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/jobs/"+payload.Job.ID+"/cancel", nil)
	req.Header.Set("Idempotency-Key", "job-cancel-20260712-001")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("cancelamento retornou %d", res.StatusCode)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		job, getErr := store.GetJob(context.Background(), payload.Job.ID)
		if getErr == nil && job.Status == "cancelled" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	job, _ := store.GetJob(context.Background(), payload.Job.ID)
	t.Fatalf("job não foi cancelado: %+v", job)
}
