package agent

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/id"
)

type Client interface {
	Execute(context.Context, string, map[string]any) (Response, error)
}
type UnixClient struct {
	socket, key, keyID string
	client             *http.Client
}

func NewUnixClient(socket, key string) *UnixClient {
	return NewUnixClientWithKeyID(socket, key, "")
}

func NewUnixClientWithKeyID(socket, key, keyID string) *UnixClient {
	tr := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	return &UnixClient{socket: socket, key: key, keyID: keyID, client: &http.Client{Transport: tr, Timeout: 5*time.Minute + 15*time.Second}}
}
func (c *UnixClient) Execute(ctx context.Context, op string, payload map[string]any) (Response, error) {
	spec, ok := LookupOperation(op)
	if !ok {
		return Response{Success: false, ErrorCode: "operation_not_allowed", Summary: "Operação fora do catálogo"}, fmt.Errorf("operação não permitida")
	}
	r := Request{Operation: op, OperationVersion: spec.Version, RequestID: id.New("req-"), Timestamp: time.Now().UTC(), Nonce: nonce(), Payload: payload}
	if err := SignWithKey(&r, c.key, c.keyID); err != nil {
		return Response{}, err
	}
	b, err := json.Marshal(r)
	if err != nil {
		return Response{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix/v1/execute", bytes.NewReader(b))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer res.Body.Close()
	var out Response
	if err = json.NewDecoder(res.Body).Decode(&out); err != nil {
		return out, err
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return out, fmt.Errorf("agente recusou: %s", out.Summary)
	}
	if !out.Success {
		return out, fmt.Errorf("agente recusou: %s", out.Summary)
	}
	return out, nil
}
func nonce() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

type MockClient struct{ Scenario string }

func (m MockClient) Execute(ctx context.Context, op string, payload map[string]any) (Response, error) {
	spec, ok := LookupOperation(op)
	if !ok {
		return Response{Success: false, ErrorCode: "operation_not_allowed", Summary: "Operação fora do catálogo"}, fmt.Errorf("operação não permitida")
	}
	if err := validateTransportPayload(spec, payload); err != nil {
		return Response{Success: false, ErrorCode: "invalid_payload", Summary: "Payload fora do schema permitido"}, err
	}
	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	case <-time.After(120 * time.Millisecond):
	}
	if m.Scenario == "agent-unavailable" {
		return Response{}, fmt.Errorf("agente indisponível")
	}
	if m.Scenario == "partial-failure" {
		return Response{Success: false, ExitCode: 1, Summary: "Falha parcial simulada", Stderr: "saída sanitizada"}, fmt.Errorf("falha parcial")
	}
	if spec.IsMutation() {
		return Response{Success: true, ExitCode: 0, Summary: "Operação mutável simulada; nenhuma escrita no sistema foi realizada.", Data: map[string]any{"operation": op, "simulated": true, "payloadAccepted": true}}, nil
	}
	return Response{Success: true, ExitCode: 0, Summary: "Operação mockada concluída", Data: map[string]any{"operation": op, "payloadAccepted": true}}, nil
}
