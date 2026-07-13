package agent

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFixtureExecutorReturnsStructuredInventories(t *testing.T) {
	executor, err := NewFixtureExecutor(filepath.Join("..", "..", "testdata", "fixtures", "freebsd-15.1-samba423-sanitized.json"))
	if err != nil {
		t.Fatal(err)
	}

	for operation, field := range map[string]string{
		"system.inspect":         "system",
		"capabilities.inspect":   "capabilities",
		"filesystem.inspect":     "filesystems",
		"samba.inspect":          "samba",
		"domain.member.diagnose": "domain",
		"cups.inspect":           "cups",
	} {
		spec := Catalog[operation]
		response, executeErr := executor.Execute(context.Background(), Request{RequestID: "req-fixture"}, spec)
		if executeErr != nil {
			t.Fatalf("%s falhou: %v", operation, executeErr)
		}
		if !response.Success || response.Data[field] == nil {
			t.Fatalf("%s não retornou o campo %s: %+v", operation, field, response)
		}
	}
}

func TestFixtureExecutorRejectsOperationOutsideReplay(t *testing.T) {
	executor, err := NewFixtureExecutor(filepath.Join("..", "..", "testdata", "fixtures", "freebsd-15.1-samba423-sanitized.json"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := executor.Execute(context.Background(), Request{RequestID: "req-fixture"}, Catalog["quota.read"])
	if err == nil || response.ErrorCode != "fixture_operation_unavailable" {
		t.Fatalf("operação não reproduzida deveria falhar fechada: %+v, %v", response, err)
	}
}
