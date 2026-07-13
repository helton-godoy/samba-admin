package agent

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSignedUnixSocketRequest(t *testing.T) {
	dir := t.TempDir()
	socket := filepath.Join(dir, "agent.sock")
	key := "chave-de-teste-com-mais-de-32-bytes"
	srv := NewServer(key, MockExecutor{}, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	ln, err := Listen(socket, srv.Handler())
	if err != nil {
		if errors.Is(err, syscall.EPERM) {
			t.Skip("sandbox não permite criar socket Unix")
		}
		t.Fatal(err)
	}
	httpSrv := &http.Server{Handler: srv.Handler()}
	go httpSrv.Serve(ln)
	defer httpSrv.Shutdown(context.Background())
	client := NewUnixClient(socket, key)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for _, operation := range []string{"system.inspect", "samba.inspect", "cups.inspect"} {
		payload := map[string]any(nil)
		if operation == "system.inspect" {
			payload = map[string]any{"dryRun": true}
		}
		res, executeErr := client.Execute(ctx, operation, payload)
		if executeErr != nil {
			t.Fatalf("%s pelo UDS: %v", operation, executeErr)
		}
		if !res.Success {
			t.Fatalf("resposta de %s: %+v", operation, res)
		}
	}
}
func TestUnknownOperationDenied(t *testing.T) {
	client := MockClient{}
	if _, err := client.Execute(context.Background(), "shell.run", nil); err == nil {
		t.Fatal("operação genérica foi aceita")
	}
}
