package freebsd

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/hu-ufcat/samba-admin-backend/internal/adapters"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
)

type Provider struct {
	Runner  Runner
	Now     func() time.Time
	Timeout time.Duration
}

func New(runner Runner) Provider {
	if runner == nil {
		runner = ExecRunner{}
	}
	return Provider{Runner: runner, Now: time.Now, Timeout: 15 * time.Second}
}

func (p Provider) Inspect(ctx context.Context) (models.SystemInfo, error) {
	p = p.withDefaults()
	hostname, err := p.output(ctx, "/bin/hostname")
	if err != nil {
		return models.SystemInfo{}, fmt.Errorf("hostname: %w", err)
	}
	release, err := p.output(ctx, "/sbin/sysctl", "-n", "kern.osrelease")
	if err != nil {
		return models.SystemInfo{}, fmt.Errorf("freebsd release: %w", err)
	}
	architecture, _ := p.output(ctx, "/sbin/sysctl", "-n", "hw.machine")
	cpuCount := intValue(p.optional(ctx, "/sbin/sysctl", "-n", "hw.ncpu"))
	memoryBytes := uint64Value(p.optional(ctx, "/sbin/sysctl", "-n", "hw.physmem"))
	mounts, _ := p.List(ctx)
	diskPercent := aggregateDiskPercent(mounts)
	samba, _ := p.Samba(ctx)
	domain, _ := p.Domain(ctx)
	interfaces := strings.Fields(p.optional(ctx, "/sbin/ifconfig", "-l"))
	cups, _ := p.Cups(ctx)
	info := models.SystemInfo{
		Hostname:       hostname,
		FQDN:           p.fqdn(ctx, hostname),
		FreeBSDVersion: release,
		SambaVersion:   samba.Version,
		SambaPackage:   samba.Package,
		SambaOrigin:    samba.Origin,
		Profile:        samba.Profile,
		Health:         healthFromSamba(samba),
		DomainJoined:   domain.Joined,
		DomainName:     domain.DNSDomain,
		PreferredDC:    first(domain.PreferredDCs),
		TimeSync:       p.ntpStatus(ctx),
		DNSHealth:      dnsHealth(p.dnsServers(ctx)),
		DiskPercent:    diskPercent,
		SMBSessions:    samba.Sessions,
		OpenFiles:      samba.OpenFiles,
		SharesCount:    len(samba.Shares),
		PrintQueues:    len(cups.Printers),
		LogForwarding:  "nao_verificado",
		Architecture:   architecture,
		Uptime:         p.optional(ctx, "/usr/bin/uptime"),
		CPUCount:       cpuCount,
		MemoryBytes:    memoryBytes,
		LoadAverage:    parseLoadAverage(p.optional(ctx, "/sbin/sysctl", "-n", "vm.loadavg")),
		Interfaces:     interfaces,
		DNSServers:     p.dnsServers(ctx),
		NTPStatus:      p.ntpStatus(ctx),
		Timezone:       p.optional(ctx, "/bin/date", "+%Z"),
	}
	if info.SambaVersion == "" {
		info.SambaVersion = "indisponível"
	}
	if info.SambaPackage == "" {
		info.SambaPackage = "indisponível"
	}
	if info.Profile == "" {
		info.Profile = "standalone"
	}
	return info, nil
}

func (p Provider) Capabilities(ctx context.Context) ([]models.Capability, error) {
	p = p.withDefaults()
	samba, _ := p.Samba(ctx)
	_, fstabErr := p.output(ctx, "/bin/cat", "/etc/fstab")
	_, cupsErr := p.output(ctx, "/usr/local/sbin/cupsd", "-t")
	domain, _ := p.Domain(ctx)
	domainState := "nao_verificado"
	if domain.Realm != "" || domain.NetBIOS != "" {
		domainState = "suportado_com_restricoes"
	}
	capabilities := []models.Capability{
		{ID: "system.inspect", Feature: "Inventário do sistema", State: "suportado", Scope: "FreeBSD", Evidence: "sysctl, mount, df e hostname estão disponíveis."},
		{ID: "filesystem.read", Feature: "Inventário de sistemas de arquivos", State: stateFor(fstabErr == nil, "nao_verificado"), Scope: "FreeBSD/UFS", Evidence: "mount -p, df -k e /etc/fstab somente leitura."},
		{ID: "acl.nfsv4.read", Feature: "Leitura de ACL NFSv4", State: "nao_verificado", Scope: "UFS2", Evidence: "Exige fixture de getfacl e homologação em UFS2."},
		{ID: "acl.nfsv4.write", Feature: "Escrita de ACL NFSv4", State: "indisponivel", Scope: "UFS2", Evidence: "Bloqueada por política nesta release."},
		{ID: "samba.inspect", Feature: "Inventário Samba", State: stateFor(samba.Version != "", "indisponivel"), Scope: "Samba", Evidence: "Versão, pacote e configuração são consultados por comandos fixos somente leitura."},
		{ID: "samba.testparm", Feature: "Validação Samba testparm", State: stateFor(samba.TestparmAvailable, "indisponivel"), Scope: "Samba", Evidence: "Disponibilidade do binário testparm e validação somente leitura."},
		{ID: "domain.member.diagnose", Feature: "Diagnóstico de membro AD", State: domainState, Scope: "Samba member server", Evidence: "Somente perfil Samba, DNS configurado e ping do winbind são consultados sem credenciais; SRV, Kerberos, LDAP e SMB permanecem pendentes."},
		{ID: "domain.member.join", Feature: "Ingresso como membro AD", State: "indisponivel", Scope: "Samba member server", Evidence: "Feature flag de escrita permanece desabilitada."},
		{ID: "samba.ad_dc", Feature: "Samba AD DC", State: "nao_verificado", Scope: "Módulo separado", Evidence: "Opções de pacote não substituem homologação de AD DC."},
		{ID: "cups.inspect", Feature: "Inventário CUPS", State: stateFor(cupsErr == nil, "indisponivel"), Scope: "CUPS", Evidence: "cupsd -t e lpstat somente leitura."},
		{ID: "quota.ufs.read", Feature: "Leitura de quotas UFS", State: "nao_verificado", Scope: "UFS", Evidence: "Exige ferramenta e opções de quota verificadas."},
		{ID: "quota.ufs.write", Feature: "Escrita de quotas UFS", State: "indisponivel", Scope: "UFS", Evidence: "Bloqueada por política nesta release."},
		{ID: "syslog.tls", Feature: "Syslog remoto TLS", State: "nao_verificado", Scope: "SIEM", Evidence: "Requer agente ou daemon TLS homologado."},
	}
	return capabilities, nil
}

func (p Provider) List(ctx context.Context) ([]models.FileSystem, error) {
	p = p.withDefaults()
	mountOutput, err := p.output(ctx, "/sbin/mount", "-p")
	if err != nil {
		return nil, err
	}
	dfOutput, _ := p.output(ctx, "/bin/df", "-k")
	fstabOutput, _ := p.output(ctx, "/bin/cat", "/etc/fstab")
	usage := parseDF(dfOutput)
	fstab := parseFSTAB(fstabOutput)
	fileSystems := make([]models.FileSystem, 0)
	for _, entry := range parseMountP(mountOutput) {
		if entry.Type != "ufs" && entry.Type != "zfs" {
			continue
		}
		used, size := usage[entry.MountPoint].used, usage[entry.MountPoint].size
		options := splitOptions(entry.Options)
		aclModel := "none"
		if contains(options, "nfsv4acls") {
			aclModel = "nfsv4"
		} else if contains(options, "acls") {
			aclModel = "posix"
		}
		fileSystems = append(fileSystems, models.FileSystem{
			ID:                 filesystemID(entry.Device, entry.MountPoint),
			Device:             entry.Device,
			MountPoint:         entry.MountPoint,
			Type:               entry.Type,
			SizeGiB:            kibToGiB(size),
			UsedGiB:            kibToGiB(used),
			Writable:           contains(options, "rw"),
			MountOptions:       options,
			ACLModel:           aclModel,
			ExtendedAttributes: aclModel != "none",
			UserQuota:          contains(options, "userquota"),
			GroupQuota:         contains(options, "groupquota"),
			FSTABPersistent:    fstab[entry.MountPoint] != "",
			AssociatedShares:   []string{},
			FSTABEntry:         fstab[entry.MountPoint],
		})
	}
	return fileSystems, nil
}

// ExportACL remains intentionally unavailable until real NFSv4 parser and
// restore tests finish. The adapter has no mutating method.
func (Provider) ExportACL(context.Context, string) (adapters.BackupArtifact, error) {
	return adapters.BackupArtifact{}, adapters.ErrNotVerified
}

func (p Provider) Samba(ctx context.Context) (models.SambaInfo, error) {
	p = p.withDefaults()
	info := models.SambaInfo{
		ConfigurationPath:   "/usr/local/etc/smb4.conf",
		Origin:              "FreeBSD Ports / pkg",
		Binaries:            []string{},
		Shares:              []string{},
		VFSModules:          []string{},
		PackageBuildOptions: []string{},
	}
	version, err := p.output(ctx, "/usr/local/sbin/smbd", "-V")
	if err != nil || strings.TrimSpace(version) == "" {
		return models.SambaInfo{}, fmt.Errorf("sondagem mínima do Samba (smbd -V): %w", unavailableProbeError(err))
	}
	info.Version = strings.TrimPrefix(version, "Version ")
	info.Binaries = append(info.Binaries, "/usr/local/sbin/smbd")
	if pkg, err := p.output(ctx, "/usr/local/sbin/pkg", "query", "%n-%v", "samba423"); err == nil {
		info.Package = pkg
	}
	if packageInfo, err := p.output(ctx, "/usr/local/sbin/pkg", "info", "samba423"); err == nil {
		info.PackageBuildOptions = parsePackageOptions(packageInfo)
	}
	if config, err := p.output(ctx, "/usr/local/bin/testparm", "-s"); err == nil {
		info.TestparmAvailable = true
		info.EffectiveConfig = redactSambaConfig(config)
		info.Profile = parseSambaProfile(config)
		info.Shares = parseSambaShares(config)
		info.VFSModules = parseVFSModules(config)
		info.FullAuditAvailable = contains(info.VFSModules, "full_audit") || strings.Contains(config, "vfs_full_audit")
		info.DFSAvailable = strings.Contains(strings.ToLower(config), "msdfs")
		info.PrintingAvailable = strings.Contains(strings.ToLower(config), "printing")
		info.Binaries = append(info.Binaries, "/usr/local/bin/testparm")
	}
	if shares, err := p.output(ctx, "/usr/local/bin/smbstatus", "--shares"); err == nil {
		info.Sessions = countDataLines(shares)
	}
	if locks, err := p.output(ctx, "/usr/local/bin/smbstatus", "--locks"); err == nil {
		info.OpenFiles = countDataLines(locks)
	}
	return info, nil
}

func (p Provider) Domain(ctx context.Context) (models.DomainState, error) {
	p = p.withDefaults()
	state := models.DomainState{Tests: []models.DomainTest{}}
	config, configErr := p.output(ctx, "/usr/local/bin/testparm", "-s")
	if configErr != nil {
		state.Tests = append(state.Tests, models.DomainTest{Name: "Samba configuration", State: "indisponivel", Details: "testparm não está disponível."})
		return state, nil
	}
	values := parseSambaAssignments(config)
	state.DNSDomain = strings.ToLower(values["realm"])
	state.Realm = values["realm"]
	state.NetBIOS = values["workgroup"]
	state.IDMapStrategy = parseIDMapStrategy(config)
	state.UIDRange, state.GIDRange = parseIDMapRanges(config)
	state.Joined = parseSambaProfile(config) == "domain-member"
	state.Tests = append(state.Tests, models.DomainTest{Name: "Samba profile", State: stateFor(state.Joined, "atencao"), Details: "Perfil detectado por testparm sem alterar a configuração."})
	if servers := p.dnsServers(ctx); len(servers) > 0 {
		state.Tests = append(state.Tests, models.DomainTest{Name: "DNS", State: "suportado", Details: "resolv.conf contém servidores DNS configurados."})
	} else {
		state.Tests = append(state.Tests, models.DomainTest{Name: "DNS", State: "atencao", Details: "Nenhum nameserver foi identificado em resolv.conf."})
	}
	if result, err := runWithTimeout(ctx, p.Runner, p.Timeout, "/usr/local/bin/wbinfo", "--ping-dc"); err == nil && result.ExitCode == 0 {
		state.Tests = append(state.Tests, models.DomainTest{Name: "winbind/DC", State: "suportado", Details: "wbinfo --ping-dc foi concluído."})
	} else {
		state.Tests = append(state.Tests, models.DomainTest{Name: "winbind/DC", State: "atencao", Details: "winbind ou DC não está disponível; nenhum ingresso foi tentado."})
	}
	state.Tests = append(state.Tests, models.DomainTest{Name: "Kerberos", State: "nao_verificado", Details: "A verificação autenticada requer credenciais temporárias e permanece fora do adapter de leitura."})
	state.Tests = append(state.Tests, models.DomainTest{Name: "LDAP", State: "nao_verificado", Details: "A verificação autenticada requer credenciais temporárias e permanece fora do adapter de leitura."})
	return state, nil
}

func (p Provider) Cups(ctx context.Context) (models.CupsInfo, error) {
	p = p.withDefaults()
	info := models.CupsInfo{Service: "indisponivel", Printers: []models.Printer{}, Backends: []string{}, PPDs: []string{}, Errors: []string{}}
	if pkg, err := p.output(ctx, "/usr/local/sbin/pkg", "query", "%n-%v", "cups"); err == nil {
		info.Version = pkg
	}
	if _, err := p.output(ctx, "/usr/local/sbin/cupsd", "-t"); err != nil {
		return models.CupsInfo{}, fmt.Errorf("sondagem mínima do CUPS (cupsd -t): %w", err)
	}
	info.Service = "valid"
	if printers, err := p.output(ctx, "/usr/local/bin/lpstat", "-p"); err == nil {
		for _, name := range parseLPStatPrinters(printers) {
			info.Printers = append(info.Printers, models.Printer{ID: "cups-" + strings.ToLower(name), Name: name, Shared: false, Windows11Validated: false})
		}
	} else {
		info.Errors = append(info.Errors, "filas CUPS indisponíveis para consulta")
	}
	if devices, err := p.output(ctx, "/usr/local/bin/lpstat", "-v"); err == nil {
		info.Backends = parseLPStatDevices(devices)
	} else {
		info.Errors = append(info.Errors, "backends CUPS indisponíveis para consulta")
	}
	return info, nil
}

func unavailableProbeError(err error) error {
	if err != nil {
		return err
	}
	return adapters.ErrNotVerified
}

func (p Provider) withDefaults() Provider {
	if p.Runner == nil {
		p.Runner = ExecRunner{}
	}
	if p.Now == nil {
		p.Now = time.Now
	}
	if p.Timeout <= 0 {
		p.Timeout = 15 * time.Second
	}
	return p
}

func (p Provider) output(ctx context.Context, path string, args ...string) (string, error) {
	return commandOutput(ctx, p.Runner, p.Timeout, path, args...)
}

func (p Provider) optional(ctx context.Context, path string, args ...string) string {
	value, _ := p.output(ctx, path, args...)
	return value
}

func (p Provider) fqdn(ctx context.Context, hostname string) string {
	if value, err := p.output(ctx, "/bin/hostname", "-f"); err == nil && strings.Contains(value, ".") {
		return value
	}
	return hostname
}

func (p Provider) dnsServers(ctx context.Context) []string {
	content, err := p.output(ctx, "/bin/cat", "/etc/resolv.conf")
	if err != nil {
		return []string{}
	}
	servers := []string{}
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "nameserver" {
			servers = append(servers, fields[1])
		}
	}
	return servers
}

func (p Provider) ntpStatus(ctx context.Context) string {
	if output, err := p.output(ctx, "/usr/bin/ntpq", "-pn"); err == nil && strings.Contains(output, "*") {
		return "suportado"
	}
	return "nao_verificado"
}

func stateFor(ok bool, otherwise string) string {
	if ok {
		return "suportado"
	}
	return otherwise
}

func healthFromSamba(samba models.SambaInfo) string {
	if samba.Version == "" {
		return "atencao"
	}
	if samba.TestparmAvailable {
		return "suportado"
	}
	return "atencao"
}

func dnsHealth(servers []string) string {
	if len(servers) == 0 {
		return "atencao"
	}
	return "suportado"
}

func aggregateDiskPercent(filesystems []models.FileSystem) float64 {
	var used, total float64
	for _, filesystem := range filesystems {
		used += filesystem.UsedGiB
		total += filesystem.SizeGiB
	}
	if total == 0 {
		return 0
	}
	return math.Round((used/total)*1000) / 10
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func intValue(value string) int {
	number, _ := strconv.Atoi(strings.TrimSpace(value))
	return number
}

func uint64Value(value string) uint64 {
	number, _ := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	return number
}

func filesystemID(device, mountPoint string) string {
	value := strings.NewReplacer("/", "-", " ", "-", "\\", "-").Replace(strings.TrimPrefix(device+"-"+mountPoint, "/"))
	return "fs-" + strings.Trim(value, "-")
}

// IsReadOnly is used by callers to assert that this provider cannot be used
// for a mutation path before FreeBSD homologation is complete.
func (Provider) IsReadOnly() bool { return true }

var _ adapters.SambaInventory = Provider{}
var _ adapters.CUPSInventory = Provider{}

var _ adapters.System = Provider{}
var _ adapters.Filesystems = Provider{}
