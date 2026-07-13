import type {
  Ace,
  AuditEvent,
  Capability,
  CupsInfo,
  DfsRoot,
  DomainState,
  FileSystem,
  Job,
  Printer,
  PrintDriver,
  Principal,
  Quota,
  SambaInfo,
  ServiceInfo,
  Share,
  SystemInfo
} from '../types';

export const systemInfo: SystemInfo = {
  hostname: 'CAT-VP-FS01',
  fqdn: 'cat-vp-fs01.ebserhnet.ebserh.gov.br',
  freebsdVersion: '15.1-RELEASE-p0',
  sambaVersion: '4.23.2 (simulado)',
  sambaPackage: 'samba423-4.23.2 (simulado)',
  sambaOrigin: 'FreeBSD Ports / pkg',
  profile: 'domain-member',
  health: 'atencao',
  domainJoined: true,
  domainName: 'EBSERHNET.EBSERH.GOV.BR',
  preferredDc: 'CAT-PVW-AD1.ebserhnet.ebserh.gov.br',
  timeSync: 'saudavel',
  dnsHealth: 'saudavel',
  cpuPercent: 18,
  memoryPercent: 62,
  diskPercent: 71,
  smbSessions: 42,
  openFiles: 186,
  sharesCount: 6,
  printQueues: 3,
  logForwarding: 'atencao'
};

export const sambaInfo: SambaInfo = {
  version: '4.23.2 (simulado)',
  package: 'samba423-4.23.2 (simulado)',
  origin: 'FreeBSD Ports / pkg',
  binaries: ['/usr/local/sbin/smbd', '/usr/local/bin/testparm'],
  configurationPath: '/usr/local/etc/smb4.conf',
  profile: 'domain-member',
  shares: ['Assistencial', 'Administrativo', 'print$'],
  sessions: 42,
  openFiles: 186,
  vfsModules: ['acl_xattr', 'recycle', 'full_audit'],
  testparmAvailable: true,
  fullAuditAvailable: true,
  dfsAvailable: true,
  printingAvailable: true,
  packageBuildOptions: ['ADS: on', 'CUPS: on', 'QUOTAS: on', 'SYSLOG: on']
};

export const filesystems: FileSystem[] = [
  {
    id: 'fs-data',
    device: '/dev/nda1p2',
    mountPoint: '/srv/dados',
    type: 'ufs2',
    sizeGiB: 2048,
    usedGiB: 1460,
    writable: true,
    mountOptions: ['rw', 'nfsv4acls', 'userquota', 'groupquota'],
    aclModel: 'nfsv4',
    extendedAttributes: true,
    userQuota: true,
    groupQuota: true,
    fstabPersistent: true,
    associatedShares: ['Assistencial', 'Administrativo', 'Projetos']
  },
  {
    id: 'fs-print',
    device: '/dev/nda1p3',
    mountPoint: '/var/spool/cups',
    type: 'ufs2',
    sizeGiB: 128,
    usedGiB: 34,
    writable: true,
    mountOptions: ['rw', 'acls'],
    aclModel: 'posix',
    extendedAttributes: true,
    userQuota: false,
    groupQuota: false,
    fstabPersistent: true,
    associatedShares: ['print$']
  },
  {
    id: 'fs-root',
    device: '/dev/nda1p1',
    mountPoint: '/',
    type: 'ufs2',
    sizeGiB: 96,
    usedGiB: 24,
    writable: true,
    mountOptions: ['rw'],
    aclModel: 'none',
    extendedAttributes: true,
    userQuota: false,
    groupQuota: false,
    fstabPersistent: true,
    associatedShares: []
  }
];

export const shares: Share[] = [
  {
    id: 'share-assistencial',
    name: 'Assistencial',
    description: 'Documentos operacionais de apoio assistencial',
    path: '/srv/dados/assistencial',
    enabled: true,
    readOnly: false,
    guestAccess: false,
    allowedPrincipals: ['EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL'],
    deniedPrincipals: [],
    encryption: 'desired',
    signing: 'mandatory',
    auditProfile: 'security',
    recycleBin: true,
    dfs: true,
    vfsModules: ['acl_xattr', 'recycle', 'full_audit'],
    maxConnections: 300,
    aclModel: 'nfsv4'
  },
  {
    id: 'share-administrativo',
    name: 'Administrativo',
    description: 'Área administrativa institucional',
    path: '/srv/dados/administrativo',
    enabled: true,
    readOnly: false,
    guestAccess: false,
    allowedPrincipals: ['EBSERHNET\\GDL-HUUFCAT-ADMINISTRATIVO'],
    deniedPrincipals: ['EBSERHNET\\Domain Guests'],
    encryption: 'required',
    signing: 'mandatory',
    auditProfile: 'changes',
    recycleBin: true,
    dfs: false,
    vfsModules: ['acl_xattr', 'recycle', 'full_audit'],
    maxConnections: 150,
    aclModel: 'nfsv4'
  },
  {
    id: 'share-print',
    name: 'print$',
    description: 'Drivers de impressão publicados pelo Samba',
    path: '/var/lib/samba/printers',
    enabled: true,
    readOnly: true,
    guestAccess: false,
    allowedPrincipals: ['EBSERHNET\\Domain Users'],
    deniedPrincipals: [],
    encryption: 'desired',
    signing: 'mandatory',
    auditProfile: 'minimum',
    recycleBin: false,
    dfs: false,
    vfsModules: [],
    maxConnections: 50,
    aclModel: 'posix'
  }
];

export const principals: Principal[] = [
  {
    id: 'p-owner',
    name: 'owner@',
    displayName: 'Proprietário do objeto',
    kind: 'special',
    source: 'special',
    mappingStable: true
  },
  {
    id: 'p-group',
    name: 'group@',
    displayName: 'Grupo proprietário',
    kind: 'special',
    source: 'special',
    mappingStable: true
  },
  {
    id: 'p-everyone',
    name: 'everyone@',
    displayName: 'Todos',
    kind: 'special',
    source: 'special',
    mappingStable: true
  },
  {
    id: 'p-admin',
    name: 'EBSERHNET\\GLO-SEC-HUUFCAT-FS-ADM',
    displayName: 'Administradores do servidor de arquivos',
    kind: 'group',
    source: 'active-directory',
    sid: 'S-1-5-21-100000000-200000000-300000000-4101',
    gid: 2014101,
    mappingStable: true
  },
  {
    id: 'p-assist',
    name: 'EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL',
    displayName: 'Equipe assistencial HU-UFCAT',
    kind: 'group',
    source: 'active-directory',
    sid: 'S-1-5-21-100000000-200000000-300000000-4201',
    gid: 2014201,
    mappingStable: true
  },
  {
    id: 'p-orphan',
    name: 'S-1-5-21-100000000-200000000-300000000-9999',
    displayName: 'SID sem correspondência',
    kind: 'user',
    source: 'active-directory',
    sid: 'S-1-5-21-100000000-200000000-300000000-9999',
    mappingStable: false
  }
];

export const aclEntries: Ace[] = [
  {
    id: 'ace-1',
    principal: 'EBSERHNET\\GLO-SEC-HUUFCAT-FS-ADM',
    source: 'active-directory',
    type: 'ALLOW',
    permissions: ['read_data', 'write_data', 'append_data', 'execute', 'delete', 'read_acl', 'write_acl', 'write_owner'],
    flags: ['file_inherit', 'directory_inherit'],
    inherited: false,
    order: 1
  },
  {
    id: 'ace-2',
    principal: 'EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL',
    source: 'active-directory',
    type: 'ALLOW',
    permissions: ['read_data', 'write_data', 'append_data', 'execute', 'read_acl'],
    flags: ['file_inherit', 'directory_inherit'],
    inherited: false,
    order: 2
  },
  {
    id: 'ace-3',
    principal: 'everyone@',
    source: 'special',
    type: 'DENY',
    permissions: ['write_acl', 'write_owner'],
    flags: ['file_inherit', 'directory_inherit'],
    inherited: false,
    order: 3
  }
];

export const domainState: DomainState = {
  joined: true,
  dnsDomain: 'ebserhnet.ebserh.gov.br',
  realm: 'EBSERHNET.EBSERH.GOV.BR',
  netbios: 'EBSERHNET',
  computerOu: 'OU=Servidores,OU=HU-UFCAT,OU=EBSERH,DC=ebserhnet,DC=ebserh,DC=gov,DC=br',
  preferredDcs: [
    'CAT-PVW-AD1.ebserhnet.ebserh.gov.br',
    'CAT-PVW-AD2.ebserhnet.ebserh.gov.br'
  ],
  site: 'HU-UFCAT',
  idmapStrategy: 'rid',
  uidRange: '2000000-2999999',
  gidRange: '2000000-2999999',
  tests: [
    { name: 'DNS SRV', state: 'saudavel', details: 'Registros LDAP e Kerberos encontrados.' },
    { name: 'Kerberos', state: 'saudavel', details: 'Realm e sincronização de horário válidos.' },
    { name: 'LDAP', state: 'saudavel', details: 'Bind SASL/GSSAPI disponível.' },
    { name: 'SMB/DC', state: 'saudavel', details: 'Controlador responde em SMB 3.' },
    { name: 'ID mapping', state: 'atencao', details: 'Uma identidade simulada sem correspondência estável.' }
  ]
};

export const printers: Printer[] = [
  {
    id: 'prn-1',
    name: 'HUUFCAT-ADM-COR-01',
    uri: 'ipp://10.20.30.41/ipp/print',
    model: 'Multifuncional corporativa A3',
    driver: 'Microsoft IPP Class Driver',
    shared: true,
    paused: false,
    jobs: 2,
    windows11Validated: true
  },
  {
    id: 'prn-2',
    name: 'HUUFCAT-ASSIST-PB-01',
    uri: 'socket://10.20.30.52:9100',
    model: 'Laser monocromática A4',
    driver: 'Fabricante Universal PCL6',
    shared: true,
    paused: false,
    jobs: 7,
    windows11Validated: false
  }
];

export const cupsInfo: CupsInfo = {
  version: 'cups-2.4.11 (simulado)',
  service: 'valid',
  printers,
  backends: ['ipp', 'ipps', 'socket'],
  ppds: [],
  errors: []
};

export const printDrivers: PrintDriver[] = [
  {
    id: 'drv-1',
    name: 'Microsoft IPP Class Driver',
    version: '10.0',
    architecture: 'x64',
    signed: true,
    packageAware: true,
    windows11Compatible: true
  },
  {
    id: 'drv-2',
    name: 'Fabricante Universal PCL6',
    version: '4.2.1',
    architecture: 'x64',
    signed: true,
    packageAware: true,
    windows11Compatible: false
  }
];

export const dfsRoots: DfsRoot[] = [
  {
    id: 'dfs-1',
    name: 'HUUFCAT',
    unc: '\\\\EBSERHNET\\HUUFCAT',
    enabled: true,
    links: [
      {
        name: 'Assistencial',
        path: '\\\\EBSERHNET\\HUUFCAT\\Assistencial',
        targets: ['\\\\CAT-VP-FS01\\Assistencial', '\\\\CAT-VP-FS02\\Assistencial'],
        health: 'atencao'
      }
    ]
  }
];

export const quotas: Quota[] = [
  {
    id: 'quota-1',
    filesystemId: 'fs-data',
    principal: 'EBSERHNET\\GDL-HUUFCAT-ASSISTENCIAL',
    kind: 'group',
    usedGiB: 312,
    softGiB: 400,
    hardGiB: 450,
    filesUsed: 142300,
    graceDays: 7,
    mappingStable: true
  },
  {
    id: 'quota-2',
    filesystemId: 'fs-data',
    principal: 'S-1-5-21-...-9999',
    kind: 'user',
    usedGiB: 3,
    softGiB: 10,
    hardGiB: 12,
    filesUsed: 321,
    graceDays: 7,
    mappingStable: false
  }
];

export const services: ServiceInfo[] = [
  {
    id: 'svc-smbd',
    name: 'smbd',
    description: 'Serviço de arquivos SMB',
    state: 'running',
    pid: 1882,
    enabledAtBoot: true,
    profileScope: ['standalone', 'domain-member'],
    lastMessage: 'Aceitando conexões SMB 3.'
  },
  {
    id: 'svc-winbindd',
    name: 'winbindd',
    description: 'Resolução de identidades do Active Directory',
    state: 'running',
    pid: 1891,
    enabledAtBoot: true,
    profileScope: ['domain-member'],
    lastMessage: 'DC CAT-PVW-AD1 online.'
  },
  {
    id: 'svc-cupsd',
    name: 'cupsd',
    description: 'Servidor de impressão CUPS',
    state: 'degraded',
    pid: 1117,
    enabledAtBoot: true,
    profileScope: ['standalone', 'domain-member', 'ad-dc', 'additional-dc'],
    lastMessage: 'Driver drv-2 ainda não validado no Windows 11.'
  },
  {
    id: 'svc-backend',
    name: 'samba-admin-api',
    description: 'Backend privilegiado da interface',
    state: 'running',
    pid: 2201,
    enabledAtBoot: true,
    profileScope: ['standalone', 'domain-member', 'ad-dc', 'additional-dc'],
    lastMessage: 'API v1 disponível somente em HTTPS.'
  }
];

export const auditEvents: AuditEvent[] = [
  {
    id: 'audit-1',
    timestamp: '2026-07-11T14:12:01-03:00',
    server: 'CAT-VP-FS01',
    share: 'Assistencial',
    user: 'maria.silva',
    domain: 'EBSERHNET',
    client: 'CAT-PDW-1022',
    ip: '10.20.40.22',
    operation: 'rename',
    result: 'success',
    path: '/srv/dados/assistencial/Protocolos/protocolo-v3.pdf',
    correlationId: 'corr-7e519'
  },
  {
    id: 'audit-2',
    timestamp: '2026-07-11T14:15:43-03:00',
    server: 'CAT-VP-FS01',
    share: 'Administrativo',
    user: 'joao.souza',
    domain: 'EBSERHNET',
    client: 'CAT-PDW-1120',
    ip: '10.20.40.120',
    operation: 'open',
    result: 'failure',
    path: '/srv/dados/administrativo/Diretoria/restrito.xlsx',
    correlationId: 'corr-8f001'
  }
];

export const jobs: Job[] = [
  {
    id: 'JOB-2026-0711-0042',
    requestedBy: 'EBSERHNET\\adm.hlgodoy',
    requestedAt: '2026-07-11T13:58:00-03:00',
    operation: 'Validar e recarregar configuração do compartilhamento Assistencial',
    status: 'success',
    progress: 100,
    summary: 'Backup criado, testparm aprovado e smbd recarregado.',
    cancellable: false,
    rollbackAvailable: true
  },
  {
    id: 'JOB-2026-0711-0043',
    requestedBy: 'EBSERHNET\\adm.hlgodoy',
    requestedAt: '2026-07-11T14:07:00-03:00',
    operation: 'Teste de encaminhamento TLS para o SIEM',
    status: 'partial',
    progress: 100,
    summary: 'Conexão TLS estabelecida; confirmação de ingestão pendente.',
    error: 'O coletor não retornou confirmação de processamento.',
    cancellable: false,
    rollbackAvailable: false
  }
];

export const capabilities: Capability[] = [
  {
    id: 'system.inspect',
    feature: 'Inventário do sistema',
    state: 'suportado',
    scope: 'FreeBSD',
    evidence: 'Provider de fixture somente leitura disponível.'
  },
  {
    id: 'filesystem.read',
    feature: 'Inventário de sistemas de arquivos',
    state: 'suportado',
    scope: 'FreeBSD/UFS',
    evidence: 'Montagens e uso são consultados sem alterar o host.'
  },
  {
    id: 'samba.inspect',
    feature: 'Inventário Samba',
    state: 'suportado',
    scope: 'Samba',
    evidence: 'Versão, pacote e configuração são consultados por comandos fixos somente leitura.'
  },
  {
    id: 'samba.testparm',
    feature: 'Validacao Samba com testparm',
    state: 'suportado',
    scope: 'smbd',
    evidence: 'Versao do Samba detectada; testparm esta disponivel no ambiente de homologacao.'
  },
  {
    id: 'acl.nfsv4.read',
    feature: 'ACL NFSv4 em UFS2',
    state: 'suportado',
    scope: 'Sistema de arquivos',
    evidence: 'Ponto de montagem /srv/dados usa nfsv4acls.'
  },
  {
    id: 'acl.nfsv4.write',
    feature: 'Escrita de ACL NFSv4',
    state: 'indisponivel',
    scope: 'UFS2',
    evidence: 'Bloqueada por política nesta release.'
  },
  {
    id: 'cap-shadow',
    feature: 'Shadow Copies em UFS2',
    state: 'nao_verificado',
    scope: 'Módulo VFS / estratégia de snapshots',
    evidence: 'UFS2 não oferece snapshots equivalentes ao ZFS; implementação precisa ser detectada e testada.'
  },
  {
    id: 'domain.member.diagnose',
    feature: 'Diagnostico de membro de dominio Active Directory',
    state: 'suportado',
    scope: 'winbindd / Kerberos',
    evidence: 'Ingresso e testes do domínio simulados com sucesso.'
  },
  {
    id: 'domain.member.join',
    feature: 'Ingresso como membro AD',
    state: 'indisponivel',
    scope: 'Samba member server',
    evidence: 'Feature flag de escrita permanece desabilitada.'
  },
  {
    id: 'samba.ad_dc',
    feature: 'Samba AD DC no pacote instalado',
    state: 'nao_verificado',
    scope: 'Compilação do pacote e dependências',
    evidence: 'Exige inspeção de opções de build, Kerberos, DNS e testes de provisionamento/replicação.'
  },
  {
    id: 'cap-dfs-r',
    feature: 'Replicação DFS-R',
    state: 'indisponivel',
    scope: 'Replicação física de dados',
    evidence: 'O módulo demonstra DFS Namespace e referrals; não presume DFS-R.'
  },
  {
    id: 'cups.inspect',
    feature: 'Inventário CUPS',
    state: 'suportado',
    scope: 'CUPS',
    evidence: 'cupsd -t e lpstat são consultados em modo somente leitura.'
  },
  {
    id: 'quota.ufs.read',
    feature: 'Leitura de quotas UFS',
    state: 'nao_verificado',
    scope: 'UFS',
    evidence: 'Exige ferramenta e opções de quota verificadas.'
  },
  {
    id: 'quota.ufs.write',
    feature: 'Escrita de quotas UFS',
    state: 'indisponivel',
    scope: 'UFS',
    evidence: 'Bloqueada por política nesta release.'
  },
  {
    id: 'syslog.tls',
    feature: 'Syslog remoto com TLS',
    state: 'nao_verificado',
    scope: 'Logging',
    evidence: 'Requer agente ou daemon TLS homologado; nenhum envio real é executado no RC1.'
  },
  {
    id: 'samba.service.manage',
    feature: 'Operacoes controladas de servico Samba',
    state: 'indisponivel',
    scope: 'smbd / winbindd',
    evidence: 'Operações mutáveis permanecem bloqueadas no RC1 somente leitura.'
  }
];
