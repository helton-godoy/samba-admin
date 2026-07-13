import type { components, paths } from './generated/openapi';

/*
 * Generated-contract fallback.
 *
 * This file mirrors backend/api/openapi.yaml until automated TypeScript
 * generation is wired into the build. Keep it free of UI concerns so the
 * wrapper in client.ts can be replaced by a generated client without
 * changing callers.
 */

export type CapabilityState =
  | 'suportado'
  | 'restrito'
  | 'suportado_com_restricoes'
  | 'experimental'
  | 'indisponivel'
  | 'nao_verificado'
  | 'supported'
  | 'supported_with_restrictions'
  | 'unavailable'
  | 'not_verified';

export type OpenApiPath = keyof paths;
export type OpenApiSchemas = components['schemas'];

export type HealthState = 'saudavel' | 'atencao' | 'critico' | 'desconhecido';
export type SambaProfile = 'standalone' | 'domain-member' | 'ad-dc' | 'additional-dc';
export type AclModel = 'posix' | 'nfsv4' | 'none';
export type JobStatus =
  | 'queued'
  | 'validating'
  | 'running'
  | 'testing'
  | 'success'
  | 'partial'
  | 'failed'
  | 'rolled-back'
  | 'cancellation-requested'
  | 'cancelled';

export interface ProblemFieldError {
  field: string;
  code: string;
  message: string;
}

export interface ProblemDetails {
  type?: string;
  title?: string;
  status?: number;
  detail?: string;
  instance?: string;
  correlationId?: string;
  errors?: ProblemFieldError[];
}

export type LoginRequest = components['schemas']['LoginRequest'];

export interface User extends Omit<components['schemas']['User'], 'breakGlass' | 'mfaEnabled'> {
  breakGlass?: boolean;
  mfaEnabled?: boolean;
}

export interface LoginResponse {
  user?: User;
  csrfToken?: string;
  expiresAt?: string;
  mfaRequired?: boolean;
  mfaChallengeId?: string;
}

export interface MfaVerificationRequest {
  challengeId: string;
  code: string;
}

export interface SystemInfo {
  hostname: string;
  fqdn: string;
  freebsdVersion: string;
  sambaVersion: string;
  sambaPackage: string;
  sambaOrigin: string;
  profile: SambaProfile;
  health: HealthState;
  domainJoined: boolean;
  domainName?: string | null;
  preferredDc?: string | null;
  timeSync: HealthState;
  dnsHealth: HealthState;
  cpuPercent: number;
  memoryPercent: number;
  diskPercent: number;
  smbSessions: number;
  openFiles: number;
  sharesCount: number;
  printQueues: number;
  logForwarding: HealthState;
}

export type SambaInfo = components['schemas']['SambaInfo'];

export interface FileSystem {
  id: string;
  device: string;
  mountPoint: string;
  type: 'ufs2' | 'zfs' | 'tmpfs' | string;
  sizeGiB: number;
  usedGiB: number;
  writable: boolean;
  mountOptions: string[];
  aclModel: AclModel;
  extendedAttributes: boolean;
  userQuota: boolean;
  groupQuota: boolean;
  fstabPersistent: boolean;
  associatedShares: string[];
}

export interface Share {
  id: string;
  name: string;
  description: string;
  path: string;
  enabled: boolean;
  readOnly: boolean;
  guestAccess: boolean;
  allowedPrincipals: string[];
  deniedPrincipals: string[];
  encryption: 'disabled' | 'desired' | 'required';
  signing: 'default' | 'mandatory';
  auditProfile: 'off' | 'minimum' | 'security' | 'changes' | 'complete';
  recycleBin: boolean;
  dfs: boolean;
  vfsModules: string[];
  maxConnections: number;
  aclModel: AclModel;
}

export type ShareDraft = Share;

export interface SharePreview {
  valid: boolean;
  testparm: string;
  reloadRequired: boolean;
  restartRequired: boolean;
  config: string;
  diff: string;
  impact: {
    activeSessions: number;
    openFiles: number;
    affectedShares: string[];
  };
}

export interface Ace {
  id: string;
  principal: string;
  source: 'local' | 'active-directory' | 'special';
  type: 'ALLOW' | 'DENY';
  permissions: string[];
  flags: string[];
  inherited: boolean;
  order: number;
}

export interface Principal {
  id: string;
  name: string;
  displayName: string;
  kind: 'user' | 'group' | 'special';
  source: 'local' | 'active-directory' | 'special';
  sid?: string;
  uid?: number;
  gid?: number;
  mappingStable: boolean;
}

export interface DomainState {
  joined: boolean;
  dnsDomain: string;
  realm: string;
  netbios: string;
  computerOu: string;
  preferredDcs: string[];
  site: string;
  idmapStrategy: 'rid' | 'ad';
  uidRange: string;
  gidRange: string;
  tests: Array<{ name: string; state: HealthState; details: string }>;
}

export interface Printer {
  id: string;
  name: string;
  uri: string;
  model: string;
  driver: string;
  shared: boolean;
  paused: boolean;
  jobs: number;
  windows11Validated: boolean;
}

export type CupsInfo = components['schemas']['CupsInfo'];

export interface PrintDriver {
  id: string;
  name: string;
  version: string;
  architecture: 'x64' | 'arm64';
  signed: boolean;
  packageAware: boolean;
  windows11Compatible: boolean;
}

export interface DfsRoot {
  id: string;
  name: string;
  unc: string;
  enabled: boolean;
  links: Array<{ name: string; path: string; targets: string[]; health: HealthState }>;
}

export interface Quota {
  id: string;
  filesystemId: string;
  principal: string;
  kind: 'user' | 'group';
  usedGiB: number;
  softGiB: number;
  hardGiB: number;
  filesUsed: number;
  graceDays: number;
  mappingStable: boolean;
}

export interface ServiceInfo {
  id: string;
  name: string;
  description: string;
  state: 'running' | 'stopped' | 'degraded';
  pid?: number;
  enabledAtBoot: boolean;
  profileScope: SambaProfile[];
  lastMessage: string;
}

export interface AuditEvent {
  id: string;
  timestamp: string;
  server: string;
  share: string;
  user: string;
  domain: string;
  client: string;
  ip: string;
  operation: string;
  result: 'success' | 'failure';
  path: string;
  correlationId: string;
}

export interface Job {
  id: string;
  requestedBy: string;
  requestedAt: string;
  operation: string;
  status: JobStatus;
  progress: number;
  summary: string;
  error?: string | null;
  cancellable: boolean;
  rollbackAvailable: boolean;
}

export interface Capability {
  id: string;
  feature: string;
  state: CapabilityState;
  scope: string;
  evidence: string;
  prerequisites?: string[];
}

export interface ConfigurationFile {
  path: string;
  encoding: string;
  eol: string;
  version: string;
  lockedBy: string | null;
  size: number;
  content: string;
  previousContent: string;
}

export interface ConfigurationValidation {
  valid: boolean;
  validators: string[];
  diff: string;
  backupRequired: boolean;
  reloadRequired: boolean;
}

export interface AclConversionReport {
  filesystem: string;
  from: string;
  to: string;
  requiresMaintenance: boolean;
  requiresUnmount: boolean;
  backupArtifact: string;
  affectedShares: string[];
  scannedObjects: number;
  preserved: number;
  approximated: number;
  notRepresentable: number;
  risks: string[];
  plan: string[];
}

export interface EffectiveAcl {
  principal: string;
  effective: string[];
  denied: string[];
  warnings: string[];
}

export interface DomainTestResult {
  success: boolean;
  checks: DomainState['tests'];
  commandsModeled: string[];
}

export interface ServiceActionResponse {
  id: string;
  action: string;
  accepted: boolean;
  warning?: string;
}

export interface RollbackResponse {
  id: string;
  status: string;
  message: string;
  job?: Job;
}

export interface CancelJobResponse {
  jobId: string;
  status: 'cancellation-requested';
}

export type ChangeRequestInput = components['schemas']['ChangeRequestInput'];
export type ChangeDecisionInput = components['schemas']['ChangeDecisionInput'];
export type ChangeRequest = components['schemas']['ChangeRequest'];

export interface OpenApiTransport {
  request<T>(path: OpenApiPath | string, init?: RequestInit): Promise<T>;
}

/**
 * Local generated-client shape. The HTTP wrapper supplies cookies, CSRF,
 * RFC 7807 parsing, idempotency and conditional request headers.
 */
export function createOpenApiClient(transport: OpenApiTransport) {
  const json = (method: string, body?: unknown, headers?: HeadersInit): RequestInit => ({
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body)
  });

  return {
    login: (body: LoginRequest) => transport.request<LoginResponse | User>('/api/v1/auth/login', json('POST', body)),
    verifyMfaLogin: (body: MfaVerificationRequest) => transport.request<LoginResponse | User>('/api/v1/auth/mfa/verify', json('POST', body)),
    renewSession: () => transport.request<LoginResponse | User>('/api/v1/auth/session/renew', json('POST')),
    logout: () => transport.request<void>('/api/v1/auth/logout', json('POST')),
    currentUser: () => transport.request<User>('/api/v1/auth/me'),
    getSystem: () => transport.request<SystemInfo>('/api/v1/system'),
    getSamba: () => transport.request<SambaInfo>('/api/v1/samba'),
    getCups: () => transport.request<CupsInfo>('/api/v1/cups'),
    listCapabilities: () => transport.request<Capability[]>('/api/v1/capabilities'),
    listFilesystems: () => transport.request<FileSystem[]>('/api/v1/filesystems'),
    listShares: () => transport.request<Share[]>('/api/v1/shares'),
    previewShare: (share: ShareDraft) => transport.request<SharePreview>('/api/v1/shares/preview', json('POST', share)),
    createShare: (share: Share, headers?: HeadersInit) => transport.request<{ share: Share; job: Job }>('/api/v1/shares', json('POST', share, headers)),
    listAcls: () => transport.request<Ace[]>('/api/v1/acls'),
    listIdentities: () => transport.request<Principal[]>('/api/v1/identities'),
    calculateEffectiveAcl: (principal: string) => transport.request<EffectiveAcl>('/api/v1/acl/effective', json('POST', { principal })),
    simulateAclConversion: (filesystemId: string, target: string) => transport.request<AclConversionReport>('/api/v1/acl/conversion/simulate', json('POST', { filesystemId, target })),
    getDomain: () => transport.request<DomainState>('/api/v1/domain'),
    testDomain: () => transport.request<DomainTestResult>('/api/v1/domain/test', json('POST')),
    listPrinters: () => transport.request<Printer[]>('/api/v1/printers'),
    listPrintDrivers: () => transport.request<PrintDriver[]>('/api/v1/print-drivers'),
    listDfs: () => transport.request<DfsRoot[]>('/api/v1/dfs'),
    listQuotas: () => transport.request<Quota[]>('/api/v1/quotas'),
    listServices: () => transport.request<ServiceInfo[]>('/api/v1/services'),
    serviceAction: (id: string, action: string, headers?: HeadersInit) => transport.request<ServiceActionResponse>(`/api/v1/services/${encodeURIComponent(id)}/action`, json('POST', { action }, headers)),
    listAudit: () => transport.request<AuditEvent[]>('/api/v1/audit'),
    listJobs: () => transport.request<Job[]>('/api/v1/jobs'),
    cancelJob: (id: string, headers?: HeadersInit) => transport.request<CancelJobResponse>(`/api/v1/jobs/${encodeURIComponent(id)}/cancel`, json('POST', undefined, headers)),
    rollbackJob: (id: string, headers?: HeadersInit) => transport.request<RollbackResponse>(`/api/v1/jobs/${encodeURIComponent(id)}/rollback`, json('POST', undefined, headers)),
    listChangeRequests: (status?: ChangeRequest['status']) => transport.request<ChangeRequest[]>(`/api/v1/change-requests${status ? `?status=${encodeURIComponent(status)}` : ''}`),
    createChangeRequest: (body: ChangeRequestInput, headers?: HeadersInit) => transport.request<ChangeRequest>('/api/v1/change-requests', json('POST', body, headers)),
    approveChangeRequest: (id: string, body: ChangeDecisionInput, headers?: HeadersInit) => transport.request<ChangeRequest>(`/api/v1/change-requests/${encodeURIComponent(id)}/approve`, json('POST', body, headers)),
    rejectChangeRequest: (id: string, body: ChangeDecisionInput, headers?: HeadersInit) => transport.request<ChangeRequest>(`/api/v1/change-requests/${encodeURIComponent(id)}/reject`, json('POST', body, headers)),
    executeApprovedChangeRequest: (id: string, headers?: HeadersInit) => transport.request<Job>(`/api/v1/change-requests/${encodeURIComponent(id)}/execute`, json('POST', undefined, headers)),
    getConfigurationFile: (path: string) => transport.request<ConfigurationFile>(`/api/v1/configuration/file?path=${encodeURIComponent(path)}`),
    validateConfiguration: (path: string, content: string) => transport.request<ConfigurationValidation>('/api/v1/configuration/validate', json('POST', { path, content }))
  };
}
