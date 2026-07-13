package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Request is the complete, signed intent sent from the unprivileged API to the
// privileged agent. OperationVersion and KeyID are part of the signature.
type Request struct {
	Operation        string         `json:"operation"`
	OperationVersion int            `json:"operationVersion"`
	RequestID        string         `json:"requestId"`
	Timestamp        time.Time      `json:"timestamp"`
	Nonce            string         `json:"nonce"`
	KeyID            string         `json:"keyId,omitempty"`
	Payload          map[string]any `json:"payload"`
	Signature        string         `json:"signature"`
}

type Response struct {
	RequestID string         `json:"requestId"`
	Success   bool           `json:"success"`
	ExitCode  int            `json:"exitCode"`
	Summary   string         `json:"summary"`
	Stdout    string         `json:"stdout,omitempty"`
	Stderr    string         `json:"stderr,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	ErrorCode string         `json:"errorCode,omitempty"`
}

type RiskLevel string

const (
	RiskRead        RiskLevel = "read"
	RiskLow         RiskLevel = "low"
	RiskMedium      RiskLevel = "medium"
	RiskHigh        RiskLevel = "high"
	RiskDestructive RiskLevel = "destructive"
	RiskEmergency   RiskLevel = "emergency"
)

type AccessMode string

const (
	AccessReadOnly AccessMode = "read-only"
	AccessMutating AccessMode = "mutating"
)

type RollbackPolicy string

const (
	RollbackNone     RollbackPolicy = "none"
	RollbackManual   RollbackPolicy = "manual"
	RollbackRequired RollbackPolicy = "required"
)

type AuditStrategy string

const (
	AuditMinimal  AuditStrategy = "minimal"
	AuditSecurity AuditStrategy = "security"
	AuditFull     AuditStrategy = "full"
)

type FieldType string

const (
	FieldString FieldType = "string"
	FieldBool   FieldType = "boolean"
	FieldNumber FieldType = "number"
	FieldObject FieldType = "object"
	FieldArray  FieldType = "array"
)

// FieldRule deliberately models a small, auditable subset of JSON Schema.
// The agent only needs enough expressiveness to reject unexpected privileged
// inputs; richer validation also happens in the API.
type FieldRule struct {
	Type      FieldType `json:"type"`
	Required  bool      `json:"required,omitempty"`
	Enum      []string  `json:"enum,omitempty"`
	MaxLength int       `json:"maxLength,omitempty"`
	Sensitive bool      `json:"sensitive,omitempty"`
}

type PayloadSchema struct {
	Fields               map[string]FieldRule `json:"fields"`
	AdditionalProperties bool                 `json:"additionalProperties"`
}

// OperationSpec is the policy boundary for every privileged intent. Mutable
// entries stay in the catalog for traceability but the agent rejects them
// until a separately homologated release deliberately enables them.
type OperationSpec struct {
	Name               string              `json:"name"`
	Operation          string              `json:"operation"`
	Version            int                 `json:"version"`
	Payload            PayloadSchema       `json:"payloadSchema"`
	Risk               RiskLevel           `json:"risk"`
	Access             AccessMode          `json:"access"`
	TimeoutSeconds     int                 `json:"timeoutSeconds"`
	LockRequired       bool                `json:"lockRequired"`
	LockResource       string              `json:"lockResource,omitempty"`
	Executables        []string            `json:"executables"`
	AllowedArguments   map[string][]string `json:"allowedArguments,omitempty"`
	AllowedPaths       []string            `json:"allowedPaths,omitempty"`
	BackupRequired     bool                `json:"backupRequired"`
	ApprovalRequired   bool                `json:"approvalRequired"`
	RequiredCapability string              `json:"requiredCapability,omitempty"`
	Rollback           RollbackPolicy      `json:"rollbackPolicy"`
	SensitiveFields    []string            `json:"sensitiveFields,omitempty"`
	Audit              AuditStrategy       `json:"auditStrategy"`
}

func readOperation(name, capability string, timeout int, executable string, arguments []string) OperationSpec {
	allowed := map[string][]string{}
	if executable != "" {
		allowed[executable] = append([]string(nil), arguments...)
	}
	return OperationSpec{
		Name: name, Operation: name, Version: 1,
		Payload: PayloadSchema{Fields: map[string]FieldRule{}},
		Risk:    RiskRead, Access: AccessReadOnly, TimeoutSeconds: timeout,
		Executables: []string{executable}, AllowedArguments: allowed,
		RequiredCapability: capability, Rollback: RollbackNone, Audit: AuditMinimal,
	}
}

type catalogCommand struct {
	path string
	args []string
}

func readInventoryOperation(name, capability string, timeout, version int, commands ...catalogCommand) OperationSpec {
	spec := readOperation(name, capability, timeout, "", nil)
	spec.Version = version
	spec.Executables = []string{}
	spec.AllowedArguments = map[string][]string{}
	for _, command := range commands {
		if !contains(spec.Executables, command.path) {
			spec.Executables = append(spec.Executables, command.path)
		}
		for _, argument := range command.args {
			if !contains(spec.AllowedArguments[command.path], argument) {
				spec.AllowedArguments[command.path] = append(spec.AllowedArguments[command.path], argument)
			}
		}
	}
	return spec
}

func mutatingOperation(name, capability string, risk RiskLevel, timeout int) OperationSpec {
	return OperationSpec{
		Name: name, Operation: name, Version: 1,
		Payload: PayloadSchema{Fields: map[string]FieldRule{}, AdditionalProperties: false},
		Risk:    risk, Access: AccessMutating, TimeoutSeconds: timeout,
		LockRequired: true, LockResource: "operation", BackupRequired: true,
		ApprovalRequired: true, RequiredCapability: capability,
		Rollback: RollbackRequired, Audit: AuditFull,
	}
}

// Catalog is intentionally closed. No endpoint can provide an executable,
// argument vector, or arbitrary path to this process.
var Catalog = func() map[string]OperationSpec {
	filesystemCommands := []catalogCommand{
		{path: "/sbin/mount", args: []string{"-p"}},
		{path: "/bin/df", args: []string{"-k"}},
		{path: "/bin/cat", args: []string{"/etc/fstab"}},
	}
	sambaCommands := []catalogCommand{
		{path: "/usr/local/sbin/smbd", args: []string{"-V"}},
		{path: "/usr/local/sbin/pkg", args: []string{"query", "%n-%v", "samba423", "info"}},
		{path: "/usr/local/bin/testparm", args: []string{"-s"}},
		{path: "/usr/local/bin/smbstatus", args: []string{"--shares", "--locks"}},
	}
	domainCommands := []catalogCommand{
		{path: "/usr/local/bin/testparm", args: []string{"-s"}},
		{path: "/bin/cat", args: []string{"/etc/resolv.conf"}},
		{path: "/usr/local/bin/wbinfo", args: []string{"--ping-dc"}},
	}
	cupsCommands := []catalogCommand{
		{path: "/usr/local/sbin/pkg", args: []string{"query", "%n-%v", "cups"}},
		{path: "/usr/local/sbin/cupsd", args: []string{"-t"}},
		{path: "/usr/local/bin/lpstat", args: []string{"-p", "-v"}},
	}
	systemCommands := []catalogCommand{
		{path: "/bin/hostname", args: []string{"-f"}},
		{path: "/sbin/sysctl", args: []string{"-n", "kern.osrelease", "hw.machine", "hw.ncpu", "hw.physmem", "vm.loadavg"}},
		{path: "/sbin/ifconfig", args: []string{"-l"}},
		{path: "/usr/bin/uptime"},
		{path: "/usr/bin/ntpq", args: []string{"-pn"}},
		{path: "/bin/date", args: []string{"+%Z"}},
	}
	systemCommands = append(systemCommands, filesystemCommands...)
	systemCommands = append(systemCommands, sambaCommands...)
	systemCommands = append(systemCommands, domainCommands...)
	systemCommands = append(systemCommands, cupsCommands...)
	systemInspect := readInventoryOperation("system.inspect", "system.inspect", 30, 2, systemCommands...)
	systemInspect.Payload = PayloadSchema{Fields: map[string]FieldRule{
		"dryRun": {Type: FieldBool},
	}}
	capabilityCommands := append([]catalogCommand{}, sambaCommands...)
	capabilityCommands = append(capabilityCommands, domainCommands...)
	capabilityCommands = append(capabilityCommands,
		catalogCommand{path: "/bin/cat", args: []string{"/etc/fstab"}},
		catalogCommand{path: "/usr/local/sbin/cupsd", args: []string{"-t"}},
	)
	capabilitiesInspect := readInventoryOperation("capabilities.inspect", "capabilities.inspect", 30, 1, capabilityCommands...)
	filesystemInspect := readInventoryOperation("filesystem.inspect", "filesystem.inspect", 20, 2, filesystemCommands...)
	sambaInspect := readInventoryOperation("samba.inspect", "samba.inspect", 30, 2, sambaCommands...)
	sambaValidate := readOperation("samba.config.validate", "samba.testparm", 30, "/usr/local/bin/testparm", []string{"-s"})
	domainDiagnose := readInventoryOperation("domain.member.diagnose", "domain.member.diagnose", 30, 2, domainCommands...)
	cupsInspect := readInventoryOperation("cups.inspect", "cups.inspect", 30, 2, cupsCommands...)
	quotaRead := readOperation("quota.read", "quota.ufs.read", 30, "/usr/bin/quota", []string{"-v"})
	serviceStatus := readOperation("service.status", "service.status", 15, "/usr/sbin/service", nil)
	serviceStatus.Payload = PayloadSchema{Fields: map[string]FieldRule{
		"serviceId": {Type: FieldString, Required: true, Enum: []string{"svc-smbd", "svc-winbindd", "svc-cupsd"}, MaxLength: 32},
		"action":    {Type: FieldString, Required: true, Enum: []string{"health-check"}, MaxLength: 32},
	}}
	serviceStatus.AllowedArguments = map[string][]string{"/usr/sbin/service": {"samba_server", "winbindd", "cupsd", "status"}}

	aclRead := readOperation("acl.read", "acl.nfsv4.read", 30, "/bin/getfacl", nil)
	aclRead.Payload = PayloadSchema{Fields: map[string]FieldRule{
		"path": {Type: FieldString, Required: true, MaxLength: 1024},
	}}
	aclRead.AllowedPaths = []string{"/srv/dados", "/var/lib/samba/printers"}

	shareCreate := mutatingOperation("share.create", "share.write", RiskMedium, 90)
	shareCreate.Payload = PayloadSchema{Fields: map[string]FieldRule{
		"share":  {Type: FieldObject, Required: true},
		"dryRun": {Type: FieldBool},
	}}
	shareUpdate := mutatingOperation("share.update", "share.write", RiskMedium, 90)
	shareDisable := mutatingOperation("share.disable", "share.write", RiskHigh, 60)
	configApply := mutatingOperation("samba.config.apply", "samba.config.write", RiskHigh, 60)
	configApply.Payload = PayloadSchema{Fields: map[string]FieldRule{
		"path":    {Type: FieldString, Required: true, MaxLength: 1024},
		"content": {Type: FieldString, Required: true, MaxLength: 1 << 20, Sensitive: true},
	}}
	configApply.AllowedPaths = []string{"/usr/local/etc/smb4.conf"}
	serviceReload := mutatingOperation("samba.service.reload", "samba.service.reload", RiskHigh, 30)
	serviceRestart := mutatingOperation("samba.service.restart", "samba.service.restart", RiskHigh, 60)
	serviceGenericRestart := mutatingOperation("service.restart", "service.restart", RiskHigh, 60)
	serviceStart := mutatingOperation("service.start", "service.start", RiskMedium, 60)
	serviceStop := mutatingOperation("service.stop", "service.stop", RiskHigh, 60)
	genericServiceReload := mutatingOperation("service.reload", "service.reload", RiskMedium, 60)
	servicePayload := PayloadSchema{Fields: map[string]FieldRule{
		"serviceId": {Type: FieldString, Required: true, Enum: []string{"svc-smbd", "svc-winbindd", "svc-cupsd"}, MaxLength: 32},
		"action":    {Type: FieldString, Required: true, Enum: []string{"start", "stop", "restart", "reload", "health-check"}, MaxLength: 32},
	}}
	genericServiceReload.Payload = servicePayload
	serviceRestart.Payload = servicePayload
	serviceGenericRestart.Payload = servicePayload
	serviceStart.Payload = servicePayload
	serviceStop.Payload = servicePayload
	serviceReload.Payload = servicePayload
	aclApply := mutatingOperation("acl.apply", "acl.nfsv4.write", RiskDestructive, 300)
	quotaApply := mutatingOperation("quota.apply", "quota.ufs.write", RiskMedium, 60)
	loggingTest := mutatingOperation("logging.test", "syslog.tls", RiskMedium, 30)
	backupCreate := mutatingOperation("backup.create", "backup.create", RiskLow, 120)
	backupCreate.Access = AccessMutating
	backupCreate.BackupRequired = false
	backupCreate.Rollback = RollbackNone
	rollbackExecute := mutatingOperation("rollback.execute", "rollback.execute", RiskHigh, 300)
	rollbackExecute.Payload = PayloadSchema{Fields: map[string]FieldRule{
		"sourceJobId": {Type: FieldString, MaxLength: 128},
		"backupId":    {Type: FieldString, MaxLength: 128},
		"reason":      {Type: FieldString, MaxLength: 1024},
	}}

	return map[string]OperationSpec{
		"system.inspect":         systemInspect,
		"capabilities.inspect":   capabilitiesInspect,
		"filesystem.inspect":     filesystemInspect,
		"samba.inspect":          sambaInspect,
		"samba.config.validate":  sambaValidate,
		"domain.member.diagnose": domainDiagnose,
		"cups.inspect":           cupsInspect,
		"quota.read":             quotaRead,
		"service.status":         serviceStatus,
		"acl.read":               aclRead,
		"share.create":           shareCreate,
		"share.update":           shareUpdate,
		"share.disable":          shareDisable,
		"samba.config.apply":     configApply,
		"samba.service.reload":   serviceReload,
		"samba.service.restart":  serviceRestart,
		"service.restart":        serviceGenericRestart,
		"service.start":          serviceStart,
		"service.stop":           serviceStop,
		"service.reload":         genericServiceReload,
		"acl.apply":              aclApply,
		"quota.apply":            quotaApply,
		"logging.test":           loggingTest,
		"backup.create":          backupCreate,
		"rollback.execute":       rollbackExecute,
	}
}()

func LookupOperation(name string) (OperationSpec, bool) {
	spec, ok := Catalog[name]
	return spec, ok
}

func ValidateCatalog() error {
	if len(Catalog) == 0 {
		return errors.New("catálogo de operações vazio")
	}
	for key, spec := range Catalog {
		if key == "" || spec.Name != key || spec.Operation != key {
			return fmt.Errorf("catálogo contém nome inconsistente para %q", key)
		}
		if spec.Version < 1 || spec.TimeoutSeconds < 1 {
			return fmt.Errorf("catálogo contém versão ou timeout inválido para %q", key)
		}
		if spec.Access != AccessReadOnly && spec.Access != AccessMutating {
			return fmt.Errorf("catálogo contém modo de acesso inválido para %q", key)
		}
		if !validRisk(spec.Risk) || !validRollback(spec.Rollback) || !validAudit(spec.Audit) {
			return fmt.Errorf("catálogo incompleto para %q", key)
		}
		for _, executable := range spec.Executables {
			if executable == "" || !filepath.IsAbs(executable) {
				return fmt.Errorf("executável não absoluto em %q", key)
			}
		}
		for _, root := range spec.AllowedPaths {
			if !filepath.IsAbs(root) || filepath.Clean(root) != root {
				return fmt.Errorf("caminho permitido inválido em %q", key)
			}
		}
		for executable, arguments := range spec.AllowedArguments {
			if !contains(spec.Executables, executable) || !filepath.IsAbs(executable) {
				return fmt.Errorf("allowlist de argumentos inválida em %q", key)
			}
			for _, argument := range arguments {
				if argument == "" || strings.ContainsAny(argument, "\r\n\x00") {
					return fmt.Errorf("argumento inválido no catálogo em %q", key)
				}
			}
		}
		for field, rule := range spec.Payload.Fields {
			if field == "" || !validFieldType(rule.Type) || rule.MaxLength < 0 {
				return fmt.Errorf("schema de payload inválido em %q", key)
			}
			if len(rule.Enum) > 0 && rule.Type != FieldString {
				return fmt.Errorf("enum deve ser textual no catálogo em %q", key)
			}
		}
	}
	return nil
}

func validRisk(value RiskLevel) bool {
	switch value {
	case RiskRead, RiskLow, RiskMedium, RiskHigh, RiskDestructive, RiskEmergency:
		return true
	default:
		return false
	}
}

func validRollback(value RollbackPolicy) bool {
	switch value {
	case RollbackNone, RollbackManual, RollbackRequired:
		return true
	default:
		return false
	}
}

func validAudit(value AuditStrategy) bool {
	switch value {
	case AuditMinimal, AuditSecurity, AuditFull:
		return true
	default:
		return false
	}
}

func validFieldType(value FieldType) bool {
	switch value {
	case FieldString, FieldBool, FieldNumber, FieldObject, FieldArray:
		return true
	default:
		return false
	}
}

func (s PayloadSchema) Validate(payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	for field, rule := range s.Fields {
		value, exists := payload[field]
		if rule.Required && !exists {
			return fmt.Errorf("campo obrigatório ausente: %s", field)
		}
		if !exists || value == nil {
			continue
		}
		if err := validateField(field, value, rule); err != nil {
			return err
		}
	}
	if !s.AdditionalProperties {
		for field := range payload {
			if _, ok := s.Fields[field]; !ok {
				return fmt.Errorf("campo não permitido: %s", field)
			}
		}
	}
	return nil
}

// validateTransportPayload gives in-process mock callers the same JSON shape
// the privileged server receives from the Unix socket.
func validateTransportPayload(spec OperationSpec, payload map[string]any) error {
	if payload == nil {
		return spec.Payload.Validate(nil)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var normalized map[string]any
	if err = json.Unmarshal(encoded, &normalized); err != nil {
		return err
	}
	return spec.Payload.Validate(normalized)
}

func validateField(field string, value any, rule FieldRule) error {
	switch rule.Type {
	case FieldString:
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("campo %s deve ser texto", field)
		}
		if rule.MaxLength > 0 && len(text) > rule.MaxLength {
			return fmt.Errorf("campo %s excede o tamanho permitido", field)
		}
		if len(rule.Enum) > 0 && !contains(rule.Enum, text) {
			return fmt.Errorf("campo %s possui valor não permitido", field)
		}
	case FieldBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("campo %s deve ser booleano", field)
		}
	case FieldNumber:
		switch value.(type) {
		case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		default:
			return fmt.Errorf("campo %s deve ser numérico", field)
		}
	case FieldObject:
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("campo %s deve ser objeto", field)
		}
	case FieldArray:
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("campo %s deve ser lista", field)
		}
	default:
		return fmt.Errorf("campo %s possui schema inválido", field)
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (s OperationSpec) IsMutation() bool { return s.Access == AccessMutating }

func (s OperationSpec) SensitiveField(field string) bool {
	if contains(s.SensitiveFields, field) {
		return true
	}
	rule, ok := s.Payload.Fields[field]
	return ok && rule.Sensitive
}

func (s OperationSpec) AllowsPath(path string) bool {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return false
	}
	for _, root := range s.AllowedPaths {
		if clean == root || strings.HasPrefix(clean, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
