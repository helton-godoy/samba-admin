package models

import "time"

type SystemInfo struct {
	Hostname       string    `json:"hostname"`
	FQDN           string    `json:"fqdn"`
	FreeBSDVersion string    `json:"freebsdVersion"`
	SambaVersion   string    `json:"sambaVersion"`
	SambaPackage   string    `json:"sambaPackage"`
	SambaOrigin    string    `json:"sambaOrigin"`
	Profile        string    `json:"profile"`
	Health         string    `json:"health"`
	DomainJoined   bool      `json:"domainJoined"`
	DomainName     string    `json:"domainName,omitempty"`
	PreferredDC    string    `json:"preferredDc,omitempty"`
	TimeSync       string    `json:"timeSync"`
	DNSHealth      string    `json:"dnsHealth"`
	CPUPercent     float64   `json:"cpuPercent"`
	MemoryPercent  float64   `json:"memoryPercent"`
	DiskPercent    float64   `json:"diskPercent"`
	SMBSessions    int       `json:"smbSessions"`
	OpenFiles      int       `json:"openFiles"`
	SharesCount    int       `json:"sharesCount"`
	PrintQueues    int       `json:"printQueues"`
	LogForwarding  string    `json:"logForwarding"`
	Architecture   string    `json:"architecture,omitempty"`
	Uptime         string    `json:"uptime,omitempty"`
	CPUCount       int       `json:"cpuCount,omitempty"`
	MemoryBytes    uint64    `json:"memoryBytes,omitempty"`
	LoadAverage    []float64 `json:"loadAverage,omitempty"`
	Interfaces     []string  `json:"interfaces,omitempty"`
	DNSServers     []string  `json:"dnsServers,omitempty"`
	NTPStatus      string    `json:"ntpStatus,omitempty"`
	Timezone       string    `json:"timezone,omitempty"`
}
type Capability struct {
	ID       string `json:"id"`
	Feature  string `json:"feature"`
	State    string `json:"state"`
	Scope    string `json:"scope"`
	Evidence string `json:"evidence"`
}

type FileSystem struct {
	ID                 string   `json:"id"`
	Device             string   `json:"device"`
	MountPoint         string   `json:"mountPoint"`
	Type               string   `json:"type"`
	SizeGiB            float64  `json:"sizeGiB"`
	UsedGiB            float64  `json:"usedGiB"`
	Writable           bool     `json:"writable"`
	MountOptions       []string `json:"mountOptions"`
	ACLModel           string   `json:"aclModel"`
	ExtendedAttributes bool     `json:"extendedAttributes"`
	UserQuota          bool     `json:"userQuota"`
	GroupQuota         bool     `json:"groupQuota"`
	FSTABPersistent    bool     `json:"fstabPersistent"`
	AssociatedShares   []string `json:"associatedShares"`
	FSTABEntry         string   `json:"fstabEntry,omitempty"`
}

type SambaInfo struct {
	Version             string   `json:"version"`
	Package             string   `json:"package"`
	Origin              string   `json:"origin"`
	Binaries            []string `json:"binaries"`
	ConfigurationPath   string   `json:"configurationPath"`
	Profile             string   `json:"profile"`
	EffectiveConfig     string   `json:"effectiveConfig,omitempty"`
	Shares              []string `json:"shares"`
	Sessions            int      `json:"sessions"`
	OpenFiles           int      `json:"openFiles"`
	VFSModules          []string `json:"vfsModules"`
	TestparmAvailable   bool     `json:"testparmAvailable"`
	FullAuditAvailable  bool     `json:"fullAuditAvailable"`
	DFSAvailable        bool     `json:"dfsAvailable"`
	PrintingAvailable   bool     `json:"printingAvailable"`
	PackageBuildOptions []string `json:"packageBuildOptions"`
}

type CupsInfo struct {
	Version  string    `json:"version"`
	Service  string    `json:"service"`
	Printers []Printer `json:"printers"`
	Backends []string  `json:"backends"`
	PPDs     []string  `json:"ppds"`
	Errors   []string  `json:"errors"`
}
type Share struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Enabled           bool     `json:"enabled"`
	ReadOnly          bool     `json:"readOnly"`
	GuestAccess       bool     `json:"guestAccess"`
	AllowedPrincipals []string `json:"allowedPrincipals"`
	DeniedPrincipals  []string `json:"deniedPrincipals"`
	Encryption        string   `json:"encryption"`
	Signing           string   `json:"signing"`
	AuditProfile      string   `json:"auditProfile"`
	RecycleBin        bool     `json:"recycleBin"`
	DFS               bool     `json:"dfs"`
	VFSModules        []string `json:"vfsModules"`
	MaxConnections    int      `json:"maxConnections"`
	ACLModel          string   `json:"aclModel"`
}
type Ace struct {
	ID          string   `json:"id"`
	Principal   string   `json:"principal"`
	Source      string   `json:"source"`
	Type        string   `json:"type"`
	Permissions []string `json:"permissions"`
	Flags       []string `json:"flags"`
	Inherited   bool     `json:"inherited"`
	Order       int      `json:"order"`
}
type Principal struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	Kind          string `json:"kind"`
	Source        string `json:"source"`
	SID           string `json:"sid,omitempty"`
	UID           *int   `json:"uid,omitempty"`
	GID           *int   `json:"gid,omitempty"`
	MappingStable bool   `json:"mappingStable"`
}
type DomainTest struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Details string `json:"details"`
}
type DomainState struct {
	Joined        bool         `json:"joined"`
	DNSDomain     string       `json:"dnsDomain"`
	Realm         string       `json:"realm"`
	NetBIOS       string       `json:"netbios"`
	ComputerOU    string       `json:"computerOu"`
	PreferredDCs  []string     `json:"preferredDcs"`
	Site          string       `json:"site"`
	IDMapStrategy string       `json:"idmapStrategy"`
	UIDRange      string       `json:"uidRange"`
	GIDRange      string       `json:"gidRange"`
	Tests         []DomainTest `json:"tests"`
}
type Printer struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	URI                string `json:"uri"`
	Model              string `json:"model"`
	Driver             string `json:"driver"`
	Shared             bool   `json:"shared"`
	Paused             bool   `json:"paused"`
	Jobs               int    `json:"jobs"`
	Windows11Validated bool   `json:"windows11Validated"`
}
type PrintDriver struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Version             string `json:"version"`
	Architecture        string `json:"architecture"`
	Signed              bool   `json:"signed"`
	PackageAware        bool   `json:"packageAware"`
	Windows11Compatible bool   `json:"windows11Compatible"`
}
type DfsLink struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Targets []string `json:"targets"`
	Health  string   `json:"health"`
}
type DfsRoot struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	UNC     string    `json:"unc"`
	Enabled bool      `json:"enabled"`
	Links   []DfsLink `json:"links"`
}
type Quota struct {
	ID            string  `json:"id"`
	FilesystemID  string  `json:"filesystemId"`
	Principal     string  `json:"principal"`
	Kind          string  `json:"kind"`
	UsedGiB       float64 `json:"usedGiB"`
	SoftGiB       float64 `json:"softGiB"`
	HardGiB       float64 `json:"hardGiB"`
	FilesUsed     int     `json:"filesUsed"`
	GraceDays     int     `json:"graceDays"`
	MappingStable bool    `json:"mappingStable"`
}
type ServiceInfo struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	State         string   `json:"state"`
	PID           *int     `json:"pid,omitempty"`
	EnabledAtBoot bool     `json:"enabledAtBoot"`
	ProfileScope  []string `json:"profileScope"`
	LastMessage   string   `json:"lastMessage"`
}
type AuditEvent struct {
	ID            string `json:"id"`
	Timestamp     string `json:"timestamp"`
	Server        string `json:"server"`
	Share         string `json:"share"`
	User          string `json:"user"`
	Domain        string `json:"domain"`
	Client        string `json:"client"`
	IP            string `json:"ip"`
	Operation     string `json:"operation"`
	Result        string `json:"result"`
	Path          string `json:"path"`
	CorrelationID string `json:"correlationId"`
}
type Job struct {
	ID                string    `json:"id"`
	RequestedBy       string    `json:"requestedBy"`
	RequestedAt       time.Time `json:"requestedAt"`
	Operation         string    `json:"operation"`
	Status            string    `json:"status"`
	Progress          int       `json:"progress"`
	Summary           string    `json:"summary"`
	Error             string    `json:"error,omitempty"`
	Cancellable       bool      `json:"cancellable"`
	RollbackAvailable bool      `json:"rollbackAvailable"`
	CurrentStep       string    `json:"currentStep,omitempty"`
	Resource          string    `json:"resource,omitempty"`
	CorrelationID     string    `json:"correlationId,omitempty"`
}
type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
	BreakGlass  bool     `json:"breakGlass"`
	MFAEnabled  bool     `json:"mfaEnabled"`
}

type ChangeRequest struct {
	ID                 string     `json:"id"`
	Operation          string     `json:"operation"`
	Resource           string     `json:"resource"`
	Risk               string     `json:"risk"`
	Status             string     `json:"status"`
	RequestedBy        string     `json:"requestedBy"`
	RequestedAt        time.Time  `json:"requestedAt"`
	Justification      string     `json:"justification"`
	MaintenanceStart   *time.Time `json:"maintenanceStart,omitempty"`
	MaintenanceEnd     *time.Time `json:"maintenanceEnd,omitempty"`
	Impact             string     `json:"impact"`
	RollbackPlan       string     `json:"rollbackPlan"`
	RequiredCapability string     `json:"requiredCapability,omitempty"`
	JobID              string     `json:"jobId,omitempty"`
	ApprovedBy         string     `json:"approvedBy,omitempty"`
	ApprovedAt         *time.Time `json:"approvedAt,omitempty"`
	DecisionReason     string     `json:"decisionReason,omitempty"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
}
