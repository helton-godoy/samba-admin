// Package fixturecollector coleta e sanitiza evidências somente leitura.
package fixturecollector

import (
	"context"
	"sync"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/adapters"
	"github.com/hu-ufcat/samba-admin-backend/internal/adapters/freebsd"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
)

const SchemaVersion = 1

type CommandCapture struct {
	Executable string   `json:"executable"`
	Arguments  []string `json:"arguments"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	ExitCode   int      `json:"exitCode"`
	Error      string   `json:"error,omitempty"`
}

type Fixture struct {
	SchemaVersion int                 `json:"schemaVersion"`
	CollectedAt   time.Time           `json:"collectedAt"`
	Source        string              `json:"source"`
	Sanitized     bool                `json:"sanitized"`
	System        models.SystemInfo   `json:"system"`
	Filesystems   []models.FileSystem `json:"filesystems"`
	Samba         models.SambaInfo    `json:"samba"`
	Domain        models.DomainState  `json:"domain"`
	Cups          models.CupsInfo     `json:"cups"`
	Capabilities  []models.Capability `json:"capabilities"`
	Commands      []CommandCapture    `json:"commands"`
	Errors        map[string]string   `json:"errors,omitempty"`
}

type Source interface {
	adapters.System
	adapters.Filesystems
	adapters.SambaInventory
	adapters.CUPSInventory
	Domain(context.Context) (models.DomainState, error)
}

type CaptureRunner struct {
	delegate freebsd.Runner
	mu       sync.Mutex
	commands []CommandCapture
}

func NewCaptureRunner(delegate freebsd.Runner) *CaptureRunner {
	return &CaptureRunner{delegate: delegate}
}

func (r *CaptureRunner) Run(ctx context.Context, path string, args ...string) (freebsd.CommandResult, error) {
	result, err := r.delegate.Run(ctx, path, args...)
	capture := CommandCapture{
		Executable: path,
		Arguments:  append([]string(nil), args...),
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		ExitCode:   result.ExitCode,
	}
	if err != nil {
		capture.Error = err.Error()
	}
	r.mu.Lock()
	r.commands = append(r.commands, capture)
	r.mu.Unlock()
	return result, err
}

func (r *CaptureRunner) Commands() []CommandCapture {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]CommandCapture, len(r.commands))
	copy(result, r.commands)
	for i := range result {
		result[i].Arguments = append([]string(nil), result[i].Arguments...)
	}
	return result
}

func Collect(ctx context.Context, source Source, commands []CommandCapture, now time.Time) Fixture {
	fixture := Fixture{
		SchemaVersion: SchemaVersion,
		CollectedAt:   now.UTC(),
		Source:        "freebsd-readonly",
		Sanitized:     false,
		Commands:      append([]CommandCapture(nil), commands...),
		Errors:        map[string]string{},
	}
	var err error
	if fixture.System, err = source.Inspect(ctx); err != nil {
		fixture.Errors["system"] = err.Error()
	}
	if fixture.Filesystems, err = source.List(ctx); err != nil {
		fixture.Errors["filesystems"] = err.Error()
	}
	if fixture.Samba, err = source.Samba(ctx); err != nil {
		fixture.Errors["samba"] = err.Error()
	}
	if fixture.Domain, err = source.Domain(ctx); err != nil {
		fixture.Errors["domain"] = err.Error()
	}
	if fixture.Cups, err = source.Cups(ctx); err != nil {
		fixture.Errors["cups"] = err.Error()
	}
	if fixture.Capabilities, err = source.Capabilities(ctx); err != nil {
		fixture.Errors["capabilities"] = err.Error()
	}
	if len(fixture.Errors) == 0 {
		fixture.Errors = nil
	}
	return fixture
}
