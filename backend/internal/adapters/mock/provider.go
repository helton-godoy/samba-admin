package mock

import (
	"context"

	"github.com/hu-ufcat/samba-admin-backend/internal/adapters"
	"github.com/hu-ufcat/samba-admin-backend/internal/fixtures"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
)

// Provider is the deterministic read-only adapter used by development and tests.
type Provider struct{}

func (Provider) Inspect(context.Context) (models.SystemInfo, error) { return fixtures.System(), nil }
func (Provider) Capabilities(context.Context) ([]models.Capability, error) {
	return fixtures.Capabilities(), nil
}
func (Provider) List(context.Context) ([]models.FileSystem, error) {
	return fixtures.Filesystems(), nil
}
func (Provider) ExportACL(context.Context, string) (adapters.BackupArtifact, error) {
	return adapters.BackupArtifact{}, adapters.ErrNotVerified
}
func (Provider) Samba(context.Context) (models.SambaInfo, error)    { return fixtures.Samba(), nil }
func (Provider) Domain(context.Context) (models.DomainState, error) { return fixtures.Domain(), nil }
func (Provider) Cups(context.Context) (models.CupsInfo, error)      { return fixtures.Cups(), nil }
func (Provider) IsReadOnly() bool                                   { return true }

var _ adapters.System = Provider{}
var _ adapters.Filesystems = Provider{}
var _ adapters.SambaInventory = Provider{}
var _ adapters.CUPSInventory = Provider{}
