package agent

import (
	"context"
	"errors"
	"testing"

	transport "github.com/hu-ufcat/samba-admin-backend/internal/agent"
)

type fakeClient struct {
	responses map[string]transport.Response
	err       error
}

func (f fakeClient) Execute(_ context.Context, operation string, _ map[string]any) (transport.Response, error) {
	return f.responses[operation], f.err
}

func TestProviderDecodesStructuredInventory(t *testing.T) {
	provider, err := New(fakeClient{responses: map[string]transport.Response{
		"system.inspect": {
			Success: true,
			Data: map[string]any{"system": map[string]any{
				"hostname": "lab", "fqdn": "lab.invalid", "freebsdVersion": "15.1-RELEASE",
			}},
		},
		"filesystem.inspect": {
			Success: true,
			Data: map[string]any{"filesystems": []any{map[string]any{
				"id": "fs-root", "device": "/dev/vtbd0s1a", "mountPoint": "/", "type": "ufs",
			}}},
		},
		"samba.inspect": {
			Success: true,
			Data: map[string]any{"samba": map[string]any{
				"version": "4.23.8", "configurationPath": "/usr/local/etc/smb4.conf", "profile": "domain-member",
			}},
		},
		"cups.inspect": {
			Success: true,
			Data: map[string]any{"cups": map[string]any{
				"version": "2.4.19", "service": "valid",
			}},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	system, err := provider.Inspect(context.Background())
	if err != nil || system.Hostname != "lab" || system.FreeBSDVersion != "15.1-RELEASE" {
		t.Fatalf("inventário do sistema inválido: %+v err=%v", system, err)
	}
	filesystems, err := provider.List(context.Background())
	if err != nil || len(filesystems) != 1 || filesystems[0].MountPoint != "/" {
		t.Fatalf("inventário de filesystem inválido: %+v err=%v", filesystems, err)
	}
	samba, err := provider.Samba(context.Background())
	if err != nil || samba.Version != "4.23.8" || samba.Shares == nil || samba.VFSModules == nil {
		t.Fatalf("inventário Samba inválido: %+v err=%v", samba, err)
	}
	cups, err := provider.Cups(context.Background())
	if err != nil || cups.Version != "2.4.19" || cups.Printers == nil || cups.Errors == nil {
		t.Fatalf("inventário CUPS inválido: %+v err=%v", cups, err)
	}
}

func TestProviderRejectsMissingOrFailedAgentData(t *testing.T) {
	provider, err := New(fakeClient{responses: map[string]transport.Response{"system.inspect": {Success: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Inspect(context.Background()); err == nil {
		t.Fatal("resposta sem inventário estruturado foi aceita")
	}
	provider, err = New(fakeClient{err: errors.New("socket indisponível")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.Inspect(context.Background()); err == nil {
		t.Fatal("falha de transporte foi ignorada")
	}
}

func TestProviderFailsClosedForIncompleteInventory(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		operation string
		field     string
		collect   func(Provider) error
	}{
		{
			name: "Samba sem caminho de configuração", operation: "samba.inspect", field: "samba",
			collect: func(provider Provider) error { _, err := provider.Samba(context.Background()); return err },
		},
		{
			name: "CUPS sem estado da sonda", operation: "cups.inspect", field: "cups",
			collect: func(provider Provider) error { _, err := provider.Cups(context.Background()); return err },
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			provider, err := New(fakeClient{responses: map[string]transport.Response{
				testCase.operation: {Success: true, Data: map[string]any{testCase.field: map[string]any{"version": "1"}}},
			}})
			if err != nil {
				t.Fatal(err)
			}
			if err = testCase.collect(provider); err == nil {
				t.Fatal("inventário incompleto foi aceito")
			}
		})
	}
}
