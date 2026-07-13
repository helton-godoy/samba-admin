// Package agent implements the read-only adapter boundary over the signed
// Unix Domain Socket. The API process never executes operating-system tools.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hu-ufcat/samba-admin-backend/internal/adapters"
	transport "github.com/hu-ufcat/samba-admin-backend/internal/agent"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
)

type Provider struct {
	client transport.Client
}

func New(client transport.Client) (Provider, error) {
	if client == nil {
		return Provider{}, errors.New("cliente do agente é obrigatório")
	}
	return Provider{client: client}, nil
}

func (p Provider) Inspect(ctx context.Context) (models.SystemInfo, error) {
	var value models.SystemInfo
	return value, p.execute(ctx, "system.inspect", "system", &value)
}

func (p Provider) Capabilities(ctx context.Context) ([]models.Capability, error) {
	var value []models.Capability
	return value, p.execute(ctx, "capabilities.inspect", "capabilities", &value)
}

func (p Provider) List(ctx context.Context) ([]models.FileSystem, error) {
	var value []models.FileSystem
	return value, p.execute(ctx, "filesystem.inspect", "filesystems", &value)
}

func (Provider) ExportACL(context.Context, string) (adapters.BackupArtifact, error) {
	return adapters.BackupArtifact{}, adapters.ErrNotVerified
}

func (p Provider) Samba(ctx context.Context) (models.SambaInfo, error) {
	var value models.SambaInfo
	if err := p.execute(ctx, "samba.inspect", "samba", &value); err != nil {
		return models.SambaInfo{}, err
	}
	if value.Version == "" || value.ConfigurationPath == "" {
		return models.SambaInfo{}, errors.New("agente retornou inventário Samba sem a sondagem mínima")
	}
	if value.Binaries == nil {
		value.Binaries = []string{}
	}
	if value.Shares == nil {
		value.Shares = []string{}
	}
	if value.VFSModules == nil {
		value.VFSModules = []string{}
	}
	if value.PackageBuildOptions == nil {
		value.PackageBuildOptions = []string{}
	}
	return value, nil
}

func (p Provider) Domain(ctx context.Context) (models.DomainState, error) {
	var value models.DomainState
	return value, p.execute(ctx, "domain.member.diagnose", "domain", &value)
}

func (p Provider) Cups(ctx context.Context) (models.CupsInfo, error) {
	var value models.CupsInfo
	if err := p.execute(ctx, "cups.inspect", "cups", &value); err != nil {
		return models.CupsInfo{}, err
	}
	if value.Service == "" {
		return models.CupsInfo{}, errors.New("agente retornou inventário CUPS sem a sondagem mínima")
	}
	if value.Printers == nil {
		value.Printers = []models.Printer{}
	}
	if value.Backends == nil {
		value.Backends = []string{}
	}
	if value.PPDs == nil {
		value.PPDs = []string{}
	}
	if value.Errors == nil {
		value.Errors = []string{}
	}
	return value, nil
}

func (p Provider) execute(ctx context.Context, operation, field string, target any) error {
	response, err := p.client.Execute(ctx, operation, nil)
	if err != nil {
		return fmt.Errorf("agente %s: %w", operation, err)
	}
	if !response.Success {
		return fmt.Errorf("agente %s recusou a coleta estruturada", operation)
	}
	value, ok := response.Data[field]
	if !ok {
		return fmt.Errorf("agente %s não retornou o campo estruturado %q", operation, field)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("codificar resposta do agente %s: %w", operation, err)
	}
	if err = json.Unmarshal(encoded, target); err != nil {
		return fmt.Errorf("decodificar resposta do agente %s: %w", operation, err)
	}
	return nil
}

func (Provider) IsReadOnly() bool { return true }

var _ adapters.System = Provider{}
var _ adapters.Filesystems = Provider{}
var _ adapters.SambaInventory = Provider{}
var _ adapters.CUPSInventory = Provider{}
