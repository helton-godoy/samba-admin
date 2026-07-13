package fixtures

import (
	"sort"

	"github.com/hu-ufcat/samba-admin-backend/internal/models"
)

func System() models.SystemInfo {
	return models.SystemInfo{Hostname: "CAT-VP-FS01", FQDN: "cat-vp-fs01.ebserhnet.ebserh.gov.br", FreeBSDVersion: "15.1-RELEASE-p0 (simulado)", SambaVersion: "4.23.x (a verificar no host)", SambaPackage: "samba423 (simulado)", SambaOrigin: "FreeBSD Ports / pkg", Profile: "domain-member", Health: "atencao", DomainJoined: true, DomainName: "EBSERHNET.EBSERH.GOV.BR", PreferredDC: "CAT-PVW-AD1.ebserhnet.ebserh.gov.br", TimeSync: "saudavel", DNSHealth: "saudavel", CPUPercent: 18, MemoryPercent: 62, DiskPercent: 71, SMBSessions: 42, OpenFiles: 186, SharesCount: 3, PrintQueues: 2, LogForwarding: "atencao"}
}
func Samba() models.SambaInfo {
	system := System()
	shares := Shares()
	shareNames := make([]string, 0, len(shares))
	modules := map[string]bool{}
	for _, share := range shares {
		shareNames = append(shareNames, share.Name)
		for _, module := range share.VFSModules {
			modules[module] = true
		}
	}
	vfsModules := make([]string, 0, len(modules))
	for module := range modules {
		vfsModules = append(vfsModules, module)
	}
	sort.Strings(vfsModules)
	return models.SambaInfo{
		Version: system.SambaVersion, Package: system.SambaPackage, Origin: system.SambaOrigin,
		Binaries: []string{"/usr/local/sbin/smbd", "/usr/local/bin/testparm"}, ConfigurationPath: "/usr/local/etc/smb4.conf",
		Profile: system.Profile, Shares: shareNames, Sessions: system.SMBSessions, OpenFiles: system.OpenFiles,
		VFSModules: vfsModules, TestparmAvailable: true, FullAuditAvailable: true, DFSAvailable: true,
		PrintingAvailable: true, PackageBuildOptions: []string{"ADS: on", "FULL_AUDIT: on"},
	}
}
func Cups() models.CupsInfo {
	return models.CupsInfo{
		Version: "cups-2.4.x (simulado)", Service: "valid", Printers: Printers(),
		Backends: []string{"ipp", "socket"}, PPDs: []string{}, Errors: []string{},
	}
}
func Capabilities() []models.Capability {
	return []models.Capability{
		{ID: "smb.windows11", Feature: "SMB 3 para Windows 11", State: "suportado", Scope: "Samba membro de domínio", Evidence: "A política bloqueia SMB1 e acesso guest."},
		{ID: "system.inspect", Feature: "Inventário do sistema", State: "suportado", Scope: "Adapter de leitura", Evidence: "Disponível em mock; exige evidência no host FreeBSD para promoção de estado."},
		{ID: "filesystem.read", Feature: "Inventário de sistemas de arquivos", State: "suportado", Scope: "Adapter de leitura", Evidence: "Modelos de ACL são reportados por ponto de montagem."},
		{ID: "acl.nfsv4.read", Feature: "Leitura de ACL NFSv4", State: "suportado_com_restricoes", Scope: "UFS2", Evidence: "Parser depende de fixture ou host FreeBSD homologado."},
		{ID: "acl.nfsv4.write", Feature: "Escrita de ACL NFSv4", State: "indisponivel", Scope: "UFS2", Evidence: "Bloqueada até homologação com Windows 11 e rollback."},
		{ID: "samba.inspect", Feature: "Inventário Samba", State: "suportado", Scope: "Samba", Evidence: "Disponível no provider mock somente leitura."},
		{ID: "samba.testparm", Feature: "Validação testparm", State: "suportado", Scope: "Samba", Evidence: "Validação é simulada fora do adapter FreeBSD homologado."},
		{ID: "share.write", Feature: "Aplicação de compartilhamentos", State: "experimental", Scope: "Samba", Evidence: "Nenhuma escrita real é habilitada nesta release candidata."},
		{ID: "domain.member.diagnose", Feature: "Diagnóstico de membro AD", State: "suportado", Scope: "Member server", Evidence: "Não promove nem ingressa o servidor no domínio."},
		{ID: "domain.member.join", Feature: "Ingresso como membro AD", State: "indisponivel", Scope: "Member server", Evidence: "Protegido por feature flag até concluir homologação somente leitura."},
		{ID: "samba.ad_dc", Feature: "Samba AD DC no FreeBSD", State: "nao_verificado", Scope: "Módulo isolado", Evidence: "Servidor de arquivos não precisa ser AD DC para integrar-se ao Active Directory."},
		{ID: "cups.inspect", Feature: "Inventário CUPS", State: "suportado_com_restricoes", Scope: "Impressão", Evidence: "Exige coleta real de filas, PPD e backends."},
		{ID: "quota.ufs.read", Feature: "Leitura de quotas UFS", State: "nao_verificado", Scope: "UFS", Evidence: "Depende de opções de montagem e ferramenta disponível."},
		{ID: "quota.ufs.write", Feature: "Escrita de quotas UFS", State: "indisponivel", Scope: "UFS", Evidence: "Bloqueada neste ciclo."},
		{ID: "syslog.tls", Feature: "Syslog remoto TLS", State: "suportado_com_restricoes", Scope: "SIEM", Evidence: "Pode exigir syslog-ng, rsyslog ou agente institucional adicional."},
		{ID: "dfs.namespace.read", Feature: "DFS Namespace", State: "suportado_com_restricoes", Scope: "Referrals SMB", Evidence: "Não inclui DFS-R."},
	}
}
func Filesystems() []models.FileSystem {
	return []models.FileSystem{{ID: "fs-data", Device: "/dev/nda1p2", MountPoint: "/srv/dados", Type: "ufs2", SizeGiB: 2048, UsedGiB: 1460, Writable: true, MountOptions: []string{"rw", "nfsv4acls", "userquota", "groupquota"}, ACLModel: "nfsv4", ExtendedAttributes: true, UserQuota: true, GroupQuota: true, FSTABPersistent: true, AssociatedShares: []string{"Assistencial", "Administrativo", "Projetos"}}, {ID: "fs-print", Device: "/dev/nda1p3", MountPoint: "/var/spool/cups", Type: "ufs2", SizeGiB: 128, UsedGiB: 34, Writable: true, MountOptions: []string{"rw", "acls"}, ACLModel: "posix", ExtendedAttributes: true, FSTABPersistent: true, AssociatedShares: []string{"print$"}}, {ID: "fs-root", Device: "/dev/nda1p1", MountPoint: "/", Type: "ufs2", SizeGiB: 96, UsedGiB: 24, Writable: true, MountOptions: []string{"rw"}, ACLModel: "none", ExtendedAttributes: true, FSTABPersistent: true, AssociatedShares: []string{}}}
}
func Shares() []models.Share {
	return []models.Share{{ID: "share-assistencial", Name: "Assistencial", Description: "Documentos operacionais de apoio assistencial", Path: "/srv/dados/assistencial", Enabled: true, ReadOnly: false, GuestAccess: false, AllowedPrincipals: []string{"EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL"}, DeniedPrincipals: []string{}, Encryption: "desired", Signing: "mandatory", AuditProfile: "security", RecycleBin: true, DFS: true, VFSModules: []string{"acl_xattr", "recycle", "full_audit"}, MaxConnections: 300, ACLModel: "nfsv4"}, {ID: "share-administrativo", Name: "Administrativo", Description: "Área administrativa institucional", Path: "/srv/dados/administrativo", Enabled: true, ReadOnly: false, GuestAccess: false, AllowedPrincipals: []string{"EBSERHNET\\GDL-HUUFCAT-ADMINISTRATIVO"}, DeniedPrincipals: []string{"EBSERHNET\\Domain Guests"}, Encryption: "required", Signing: "mandatory", AuditProfile: "changes", RecycleBin: true, DFS: false, VFSModules: []string{"acl_xattr", "recycle", "full_audit"}, MaxConnections: 150, ACLModel: "nfsv4"}, {ID: "share-print", Name: "print$", Description: "Drivers de impressão publicados pelo Samba", Path: "/var/lib/samba/printers", Enabled: true, ReadOnly: true, GuestAccess: false, AllowedPrincipals: []string{"EBSERHNET\\Domain Users"}, DeniedPrincipals: []string{}, Encryption: "desired", Signing: "mandatory", AuditProfile: "minimum", RecycleBin: false, DFS: false, VFSModules: []string{}, MaxConnections: 50, ACLModel: "posix"}}
}
func ACLs() []models.Ace {
	return []models.Ace{{ID: "ace-1", Principal: "EBSERHNET\\GLO-SEC-HUUFCAT-FS-ADM", Source: "active-directory", Type: "ALLOW", Permissions: []string{"read_data", "write_data", "append_data", "execute", "delete", "read_acl", "write_acl", "write_owner"}, Flags: []string{"file_inherit", "directory_inherit"}, Order: 1}, {ID: "ace-2", Principal: "EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL", Source: "active-directory", Type: "ALLOW", Permissions: []string{"read_data", "write_data", "append_data", "execute", "read_acl"}, Flags: []string{"file_inherit", "directory_inherit"}, Order: 2}, {ID: "ace-3", Principal: "everyone@", Source: "special", Type: "DENY", Permissions: []string{"write_acl", "write_owner"}, Flags: []string{"file_inherit", "directory_inherit"}, Order: 3}}
}
func Principals() []models.Principal {
	gid1, gid2 := 2014101, 2014201
	return []models.Principal{{ID: "p-owner", Name: "owner@", DisplayName: "Proprietário do objeto", Kind: "special", Source: "special", MappingStable: true}, {ID: "p-group", Name: "group@", DisplayName: "Grupo proprietário", Kind: "special", Source: "special", MappingStable: true}, {ID: "p-everyone", Name: "everyone@", DisplayName: "Todos", Kind: "special", Source: "special", MappingStable: true}, {ID: "p-admin", Name: "EBSERHNET\\GLO-SEC-HUUFCAT-FS-ADM", DisplayName: "Administradores do servidor de arquivos", Kind: "group", Source: "active-directory", SID: "S-1-5-21-100000000-200000000-300000000-4101", GID: &gid1, MappingStable: true}, {ID: "p-assist", Name: "EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL", DisplayName: "Equipe assistencial HU-UFCAT", Kind: "group", Source: "active-directory", SID: "S-1-5-21-100000000-200000000-300000000-4201", GID: &gid2, MappingStable: true}, {ID: "p-orphan", Name: "S-1-5-21-100000000-200000000-300000000-9999", DisplayName: "SID sem correspondência", Kind: "user", Source: "active-directory", SID: "S-1-5-21-100000000-200000000-300000000-9999", MappingStable: false}}
}
func Domain() models.DomainState {
	return models.DomainState{Joined: true, DNSDomain: "ebserhnet.ebserh.gov.br", Realm: "EBSERHNET.EBSERH.GOV.BR", NetBIOS: "EBSERHNET", ComputerOU: "OU=Servidores,OU=HU-UFCAT,OU=EBSERH,DC=ebserhnet,DC=ebserh,DC=gov,DC=br", PreferredDCs: []string{"CAT-PVW-AD1.ebserhnet.ebserh.gov.br", "CAT-PVW-AD2.ebserhnet.ebserh.gov.br"}, Site: "HU-UFCAT", IDMapStrategy: "rid", UIDRange: "2000000-2999999", GIDRange: "2000000-2999999", Tests: []models.DomainTest{{Name: "DNS SRV", State: "saudavel", Details: "Registros LDAP e Kerberos encontrados."}, {Name: "Kerberos", State: "saudavel", Details: "Realm e sincronização de horário válidos."}, {Name: "LDAP", State: "saudavel", Details: "Bind SASL/GSSAPI disponível."}, {Name: "SMB/DC", State: "saudavel", Details: "Controlador responde em SMB 3."}, {Name: "ID mapping", State: "atencao", Details: "Uma identidade simulada sem correspondência estável."}}}
}
func Printers() []models.Printer {
	return []models.Printer{{ID: "prn-1", Name: "HUUFCAT-ADM-COR-01", URI: "ipp://10.20.30.41/ipp/print", Model: "Multifuncional corporativa A3", Driver: "Microsoft IPP Class Driver", Shared: true, Jobs: 2, Windows11Validated: true}, {ID: "prn-2", Name: "HUUFCAT-ASSIST-PB-01", URI: "socket://10.20.30.52:9100", Model: "Laser monocromática A4", Driver: "Fabricante Universal PCL6", Shared: true, Jobs: 7, Windows11Validated: false}}
}
func Drivers() []models.PrintDriver {
	return []models.PrintDriver{{ID: "drv-1", Name: "Microsoft IPP Class Driver", Version: "10.0", Architecture: "x64", Signed: true, PackageAware: true, Windows11Compatible: true}, {ID: "drv-2", Name: "Fabricante Universal PCL6", Version: "4.2.1", Architecture: "x64", Signed: true, PackageAware: true, Windows11Compatible: false}}
}
func DFS() []models.DfsRoot {
	return []models.DfsRoot{{ID: "dfs-1", Name: "HUUFCAT", UNC: "\\\\EBSERHNET\\HUUFCAT", Enabled: true, Links: []models.DfsLink{{Name: "Assistencial", Path: "\\\\EBSERHNET\\HUUFCAT\\Assistencial", Targets: []string{"\\\\CAT-VP-FS01\\Assistencial", "\\\\CAT-VP-FS02\\Assistencial"}, Health: "atencao"}}}}
}
func Quotas() []models.Quota {
	return []models.Quota{{ID: "quota-1", FilesystemID: "fs-data", Principal: "EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL", Kind: "group", UsedGiB: 312, SoftGiB: 400, HardGiB: 450, FilesUsed: 142300, GraceDays: 7, MappingStable: true}, {ID: "quota-2", FilesystemID: "fs-data", Principal: "S-1-5-21-...-9999", Kind: "user", UsedGiB: 3, SoftGiB: 10, HardGiB: 12, FilesUsed: 321, GraceDays: 7, MappingStable: false}}
}
func Services() []models.ServiceInfo {
	p1, p2 := 2941, 1773
	return []models.ServiceInfo{{ID: "svc-smbd", Name: "smbd", Description: "Serviço de arquivos SMB", State: "running", PID: &p1, EnabledAtBoot: true, ProfileScope: []string{"standalone", "domain-member"}, LastMessage: "ready to serve connections"}, {ID: "svc-winbindd", Name: "winbindd", Description: "Resolução de identidades do domínio", State: "running", PID: &p2, EnabledAtBoot: true, ProfileScope: []string{"domain-member"}, LastMessage: "DC connection healthy"}, {ID: "svc-cupsd", Name: "cupsd", Description: "Servidor de impressão CUPS", State: "running", EnabledAtBoot: true, ProfileScope: []string{"standalone", "domain-member"}, LastMessage: "Scheduler is running"}}
}
func Audit() []models.AuditEvent {
	return []models.AuditEvent{{ID: "audit-1", Timestamp: "2026-07-11T14:33:10-03:00", Server: "CAT-VP-FS01", Share: "Assistencial", User: "adm.hlgodoy", Domain: "EBSERHNET", Client: "CAT-PW-001", IP: "10.20.30.91", Operation: "set_nt_acl", Result: "success", Path: "/srv/dados/assistencial/protocolos", CorrelationID: "corr-20260711-001"}}
}
