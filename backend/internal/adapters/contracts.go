// Package adapters defines the privileged operating-system boundary.
// HTTP handlers and business services depend on these contracts instead of
// invoking FreeBSD utilities directly.
package adapters

import (
	"context"
	"errors"

	"github.com/hu-ufcat/samba-admin-backend/internal/models"
)

var ErrNotVerified = errors.New("capacidade ainda não verificada no host FreeBSD")

type System interface {
	Inspect(context.Context) (models.SystemInfo, error)
	Capabilities(context.Context) ([]models.Capability, error)
}

type SambaInventory interface {
	Samba(context.Context) (models.SambaInfo, error)
}

type CUPSInventory interface {
	Cups(context.Context) (models.CupsInfo, error)
}

type Filesystems interface {
	List(context.Context) ([]models.FileSystem, error)
	ExportACL(context.Context, string) (BackupArtifact, error)
}

type ACLs interface {
	Read(context.Context, string) ([]models.Ace, error)
	Effective(context.Context, string, string) (EffectivePermissions, error)
	SimulateConversion(context.Context, ACLConversionRequest) (ACLConversionReport, error)
	Apply(context.Context, ACLApplyRequest) (OperationResult, error)
}

type Shares interface {
	List(context.Context) ([]models.Share, error)
	Preview(context.Context, models.Share) (ConfigurationPreview, error)
	Apply(context.Context, models.Share) (OperationResult, error)
}

type Identities interface {
	Search(context.Context, string) ([]models.Principal, error)
	Resolve(context.Context, string) (models.Principal, error)
}

type Domain interface {
	State(context.Context) (models.DomainState, error)
	Diagnose(context.Context, DomainRequest) ([]models.DomainTest, error)
	Join(context.Context, DomainJoinRequest) (OperationResult, error)
}

type Printing interface {
	Printers(context.Context) ([]models.Printer, error)
	Drivers(context.Context) ([]models.PrintDriver, error)
}

type DFS interface {
	List(context.Context) ([]models.DfsLink, error)
}

type Quotas interface {
	List(context.Context) ([]models.Quota, error)
	Apply(context.Context, QuotaRequest) (OperationResult, error)
}

type Logging interface {
	Inspect(context.Context) (map[string]any, error)
	TestRemote(context.Context, LoggingTestRequest) (OperationResult, error)
}

type Services interface {
	List(context.Context) ([]models.ServiceInfo, error)
	Action(context.Context, ServiceActionRequest) (OperationResult, error)
}

type Configuration interface {
	Read(context.Context, string) (ConfigurationFile, error)
	Validate(context.Context, ConfigurationChange) (ConfigurationPreview, error)
	Apply(context.Context, ConfigurationChange) (OperationResult, error)
}

type Backup interface {
	Create(context.Context, BackupRequest) (BackupArtifact, error)
}

type Rollback interface {
	Execute(context.Context, RollbackRequest) (OperationResult, error)
}

type Suite struct {
	System        System
	Samba         SambaInventory
	CUPS          CUPSInventory
	Filesystems   Filesystems
	ACLs          ACLs
	Shares        Shares
	Identities    Identities
	Domain        Domain
	Printing      Printing
	DFS           DFS
	Quotas        Quotas
	Logging       Logging
	Services      Services
	Configuration Configuration
	Backup        Backup
	Rollback      Rollback
}

type OperationResult struct {
	Success       bool           `json:"success"`
	Summary       string         `json:"summary"`
	ReloadNeeded  bool           `json:"reloadNeeded,omitempty"`
	RestartNeeded bool           `json:"restartNeeded,omitempty"`
	Data          map[string]any `json:"data,omitempty"`
}

type ConfigurationPreview struct {
	Valid   bool           `json:"valid"`
	Content string         `json:"content"`
	Diff    string         `json:"diff"`
	Impact  map[string]any `json:"impact"`
}

type EffectivePermissions struct {
	Principal string   `json:"principal"`
	Allowed   []string `json:"allowed"`
	Denied    []string `json:"denied"`
	Warnings  []string `json:"warnings"`
}

type ACLConversionRequest struct {
	FilesystemID string `json:"filesystemId"`
	TargetModel  string `json:"targetModel"`
}

type ACLConversionReport struct {
	Preserved        int      `json:"preserved"`
	Approximated     int      `json:"approximated"`
	NotRepresentable int      `json:"notRepresentable"`
	Risks            []string `json:"risks"`
}

type ACLApplyRequest struct {
	Path    string       `json:"path"`
	Entries []models.Ace `json:"entries"`
	DryRun  bool         `json:"dryRun"`
}

type DomainRequest struct {
	DNSDomain   string   `json:"dnsDomain"`
	Realm       string   `json:"realm"`
	PreferredDC []string `json:"preferredDc"`
}

type DomainJoinRequest struct {
	DomainRequest
	ComputerOU  string `json:"computerOu"`
	JoinAccount string `json:"joinAccount"`
	Secret      []byte `json:"-"`
}

type QuotaRequest struct {
	FilesystemID string `json:"filesystemId"`
	PrincipalID  string `json:"principalId"`
	SoftBytes    int64  `json:"softBytes"`
	HardBytes    int64  `json:"hardBytes"`
}

type LoggingTestRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type ServiceActionRequest struct {
	ServiceID string `json:"serviceId"`
	Action    string `json:"action"`
}

type ConfigurationFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	ETag    string `json:"etag"`
}

type ConfigurationChange struct {
	Path          string `json:"path"`
	Content       string `json:"content"`
	ExpectedETag  string `json:"expectedEtag"`
	ChangeReason  string `json:"changeReason"`
	Configuration int    `json:"configurationVersion"`
}

type BackupRequest struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
}

type BackupArtifact struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
}

type RollbackRequest struct {
	BackupID string `json:"backupId"`
	Reason   string `json:"reason"`
}
