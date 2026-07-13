// Package fixture reproduz inventários sanitizados sem acessar um host real.
package fixture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hu-ufcat/samba-admin-backend/internal/adapters"
	"github.com/hu-ufcat/samba-admin-backend/internal/adapters/freebsd"
	"github.com/hu-ufcat/samba-admin-backend/internal/fixturecollector"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
)

const maxFixtureBytes = 32 << 20

type Provider struct {
	Root     string
	delegate freebsd.Provider
	loaded   bool
}

func Open(path string) (Provider, error) {
	file, err := os.Open(path)
	if err != nil {
		return Provider{}, fmt.Errorf("abrir fixture: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Provider{}, fmt.Errorf("inspecionar fixture: %w", err)
	}
	if info.Size() > maxFixtureBytes {
		return Provider{}, fmt.Errorf("fixture excede o limite de %d bytes", maxFixtureBytes)
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxFixtureBytes+1))
	decoder.DisallowUnknownFields()
	var data fixturecollector.Fixture
	if err = decoder.Decode(&data); err != nil {
		return Provider{}, fmt.Errorf("decodificar fixture: %w", err)
	}
	if data.SchemaVersion != fixturecollector.SchemaVersion {
		return Provider{}, fmt.Errorf("versão de fixture não suportada: %d", data.SchemaVersion)
	}
	if !data.Sanitized {
		return Provider{}, errors.New("fixture não sanitizada foi recusada")
	}
	if err = decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Provider{}, errors.New("fixture contém conteúdo JSON adicional")
	}
	return New(data), nil
}

func New(data fixturecollector.Fixture) Provider {
	runner := replayRunner{commands: make(map[string]fixturecollector.CommandCapture, len(data.Commands))}
	for _, command := range data.Commands {
		runner.commands[commandKey(command.Executable, command.Arguments)] = command
	}
	provider := freebsd.New(runner)
	return Provider{delegate: provider, loaded: true}
}

func (p Provider) Inspect(ctx context.Context) (models.SystemInfo, error) {
	if !p.loaded {
		return models.SystemInfo{}, errors.New("provider de fixture não inicializado")
	}
	return p.delegate.Inspect(ctx)
}

func (p Provider) Capabilities(ctx context.Context) ([]models.Capability, error) {
	if !p.loaded {
		return nil, errors.New("provider de fixture não inicializado")
	}
	return p.delegate.Capabilities(ctx)
}

func (p Provider) List(ctx context.Context) ([]models.FileSystem, error) {
	if !p.loaded {
		return nil, errors.New("provider de fixture não inicializado")
	}
	return p.delegate.List(ctx)
}

func (p Provider) ExportACL(context.Context, string) (adapters.BackupArtifact, error) {
	return adapters.BackupArtifact{}, adapters.ErrNotVerified
}

func (p Provider) Samba(ctx context.Context) (models.SambaInfo, error) {
	if !p.loaded {
		return models.SambaInfo{}, errors.New("provider de fixture não inicializado")
	}
	return p.delegate.Samba(ctx)
}

func (p Provider) Domain(ctx context.Context) (models.DomainState, error) {
	if !p.loaded {
		return models.DomainState{}, errors.New("provider de fixture não inicializado")
	}
	return p.delegate.Domain(ctx)
}

func (p Provider) Cups(ctx context.Context) (models.CupsInfo, error) {
	if !p.loaded {
		return models.CupsInfo{}, errors.New("provider de fixture não inicializado")
	}
	return p.delegate.Cups(ctx)
}

func (Provider) IsReadOnly() bool { return true }

type replayRunner struct {
	commands map[string]fixturecollector.CommandCapture
}

func (r replayRunner) Run(_ context.Context, path string, args ...string) (freebsd.CommandResult, error) {
	command, ok := r.commands[commandKey(path, args)]
	if !ok {
		return freebsd.CommandResult{}, fmt.Errorf("comando ausente na fixture: %s: %w", path, adapters.ErrNotVerified)
	}
	result := freebsd.CommandResult{Stdout: command.Stdout, Stderr: command.Stderr, ExitCode: command.ExitCode}
	if command.Error != "" {
		return result, errors.New(command.Error)
	}
	return result, nil
}

func commandKey(path string, args []string) string {
	return path + "\x00" + strings.Join(args, "\x00")
}

var _ adapters.System = Provider{}
var _ adapters.Filesystems = Provider{}
var _ adapters.SambaInventory = Provider{}
var _ adapters.CUPSInventory = Provider{}
