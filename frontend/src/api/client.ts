import {
  createOpenApiClient,
  type Ace,
  type AclConversionReport,
  type AuditEvent,
  type Capability,
  type ChangeRequest,
  type ChangeRequestInput,
  type ConfigurationFile,
  type ConfigurationValidation,
  type CupsInfo,
  type DfsRoot,
  type DomainState,
  type DomainTestResult,
  type EffectiveAcl,
  type FileSystem,
  type Job,
  type LoginResponse,
  type LoginRequest,
  type MfaVerificationRequest,
  type PrintDriver,
  type Printer,
  type Principal,
  type ProblemDetails,
  type Quota,
  type RollbackResponse,
  type SambaInfo,
  type ServiceActionResponse,
  type ServiceInfo,
  type Share,
  type SharePreview,
  type SystemInfo,
  type User
} from './generated';

export const CSRF_STORAGE_KEY = 'samba-admin-csrf';
export const SESSION_EXPIRED_EVENT = 'samba-admin:session-expired';

export class ApiError extends Error {
  readonly status: number;
  readonly code?: string;
  readonly correlationId?: string;
  readonly fieldErrors: ProblemDetails['errors'];
  readonly retryAfterSeconds?: number;
  readonly problem?: ProblemDetails;

  constructor({
    message,
    status,
    code,
    correlationId,
    fieldErrors,
    retryAfterSeconds,
    problem
  }: {
    message: string;
    status: number;
    code?: string;
    correlationId?: string;
    fieldErrors?: ProblemDetails['errors'];
    retryAfterSeconds?: number;
    problem?: ProblemDetails;
  }) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.correlationId = correlationId;
    this.fieldErrors = fieldErrors;
    this.retryAfterSeconds = retryAfterSeconds;
    this.problem = problem;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

function getSessionStorage(): Storage | undefined {
  if (typeof window === 'undefined') return undefined;
  try {
    return window.sessionStorage;
  } catch {
    return undefined;
  }
}

export function getCsrfToken(): string | undefined {
  return getSessionStorage()?.getItem(CSRF_STORAGE_KEY) || undefined;
}

export function setCsrfToken(token: string | null | undefined): void {
  const storage = getSessionStorage();
  if (!storage) return;
  if (token) storage.setItem(CSRF_STORAGE_KEY, token);
  else storage.removeItem(CSRF_STORAGE_KEY);
}

function randomIdentifier(prefix: string): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function idempotencyKey(): string {
  return randomIdentifier('ui');
}

function parseRetryAfter(value: string | null): number | undefined {
  if (!value) return undefined;
  const seconds = Number.parseInt(value, 10);
  return Number.isFinite(seconds) ? seconds : undefined;
}

function asProblem(payload: unknown): ProblemDetails | undefined {
  if (!payload || typeof payload !== 'object') return undefined;
  const candidate = payload as Record<string, unknown>;
  const hasProblemField = ['title', 'detail', 'type', 'correlationId', 'errors'].some((key) => key in candidate);
  if (!hasProblemField) return undefined;
  return {
    type: typeof candidate.type === 'string' ? candidate.type : undefined,
    title: typeof candidate.title === 'string' ? candidate.title : undefined,
    status: typeof candidate.status === 'number' ? candidate.status : undefined,
    detail: typeof candidate.detail === 'string' ? candidate.detail : undefined,
    instance: typeof candidate.instance === 'string' ? candidate.instance : undefined,
    correlationId: typeof candidate.correlationId === 'string' ? candidate.correlationId : undefined,
    errors: Array.isArray(candidate.errors)
      ? candidate.errors.filter((entry): entry is { field: string; code: string; message: string } => {
          return Boolean(entry) && typeof entry === 'object' && typeof (entry as Record<string, unknown>).field === 'string' && typeof (entry as Record<string, unknown>).code === 'string' && typeof (entry as Record<string, unknown>).message === 'string';
        })
      : undefined
  };
}

function errorFromResponse(response: Response, payload: unknown, text: string): ApiError {
  const problem = asProblem(payload);
  const legacy = payload && typeof payload === 'object' ? (payload as { code?: unknown; message?: unknown }) : undefined;
  const correlationId = problem?.correlationId || response.headers.get('X-Correlation-ID') || undefined;
  const code = typeof legacy?.code === 'string'
    ? legacy.code
    : problem?.type?.split('/').filter(Boolean).at(-1);
  const message = problem?.detail
    || (typeof legacy?.message === 'string' ? legacy.message : undefined)
    || problem?.title
    || text.trim()
    || `Falha HTTP ${response.status}`;

  return new ApiError({
    message,
    status: response.status,
    code,
    correlationId,
    fieldErrors: problem?.errors,
    retryAfterSeconds: parseRetryAfter(response.headers.get('Retry-After')),
    problem
  });
}

function notifySessionExpired(error: ApiError): void {
  if (error.status !== 401 || typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(SESSION_EXPIRED_EVENT, { detail: error }));
}

async function parsePayload(response: Response): Promise<{ payload: unknown; text: string }> {
  if (response.status === 204 || response.status === 205) return { payload: undefined, text: '' };
  const text = await response.text();
  if (!text) return { payload: undefined, text };
  try {
    return { payload: JSON.parse(text) as unknown, text };
  } catch {
    return { payload: undefined, text };
  }
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  const method = (init.method || 'GET').toUpperCase();
  headers.set('Accept', 'application/json, application/problem+json');
  if (init.body !== undefined && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json');

  const csrfToken = getCsrfToken();
  if (csrfToken && ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method) && !headers.has('X-CSRF-Token')) {
    headers.set('X-CSRF-Token', csrfToken);
  }
  if (!headers.has('X-Correlation-ID')) headers.set('X-Correlation-ID', randomIdentifier('corr-ui'));

  let response: Response;
  try {
    response = await fetch(path, {
      ...init,
      headers,
      credentials: 'include'
    });
  } catch (error) {
    if (typeof DOMException !== 'undefined' && error instanceof DOMException && error.name === 'AbortError') throw error;
    throw new ApiError({
      message: 'A interface nao conseguiu alcançar a API.',
      status: 0,
      code: 'network_unavailable'
    });
  }

  const nextCsrfToken = response.headers.get('X-CSRF-Token');
  if (nextCsrfToken) setCsrfToken(nextCsrfToken);

  const { payload, text } = await parsePayload(response);
  if (!response.ok) {
    const error = errorFromResponse(response, payload, text);
    notifySessionExpired(error);
    throw error;
  }

  return payload as T;
}

const contract = createOpenApiClient({ request });

export interface MutationOptions {
  idempotencyKey?: string;
  ifMatch?: string;
}

export type AuthenticationResult =
  | { status: 'authenticated'; user: User; expiresAt?: string }
  | { status: 'mfa_required'; challengeId: string };

function isUser(value: LoginResponse | User): value is User {
  return typeof value === 'object' && value !== null && 'username' in value && 'roles' in value;
}

function normalizeAuthentication(value: LoginResponse | User): AuthenticationResult {
  if (isUser(value)) return { status: 'authenticated', user: value };
  if (value.csrfToken) setCsrfToken(value.csrfToken);
  if (value.mfaRequired) {
    if (!value.mfaChallengeId) {
      throw new ApiError({ message: 'A API solicitou MFA sem informar desafio.', status: 502, code: 'invalid_mfa_challenge' });
    }
    return { status: 'mfa_required', challengeId: value.mfaChallengeId };
  }
  if (value.user) return { status: 'authenticated', user: value.user, expiresAt: value.expiresAt };
  throw new ApiError({ message: 'A API retornou uma sessao sem usuario.', status: 502, code: 'invalid_login_response' });
}

function mutationHeaders(options: MutationOptions = {}): Headers {
  const headers = new Headers();
  headers.set('Idempotency-Key', options.idempotencyKey || idempotencyKey());
  if (options.ifMatch) headers.set('If-Match', options.ifMatch);
  return headers;
}

/**
 * Compatibility facade for existing pages. New consumers should use the
 * canonical plural resource methods below; legacy aliases remain temporarily.
 */
export const api = {
  auth: {
    login: async (credentials: LoginRequest): Promise<AuthenticationResult> => normalizeAuthentication(await contract.login(credentials)),
    verifyMfa: async (verification: MfaVerificationRequest): Promise<AuthenticationResult> => normalizeAuthentication(await contract.verifyMfaLogin(verification)),
    logout: async (): Promise<void> => {
      await contract.logout();
      setCsrfToken(undefined);
    },
    me: (): Promise<User> => contract.currentUser(),
    refresh: async (): Promise<User> => {
      const currentUser = await contract.currentUser();
      try {
        const result = normalizeAuthentication(await contract.renewSession());
        if (result.status === 'authenticated') return result.user;
        throw new ApiError({ message: 'A renovacao da sessao requer MFA.', status: 401, code: 'mfa_required' });
      } catch (error) {
        // Older API installations expose only /auth/me. Keep controlled compatibility.
        if (isApiError(error) && error.status === 404) return currentUser;
        throw error;
      }
    }
  },
  system: (): Promise<SystemInfo> => contract.getSystem(),
  samba: (): Promise<SambaInfo> => contract.getSamba(),
  cups: (): Promise<CupsInfo> => contract.getCups(),
  filesystems: (): Promise<FileSystem[]> => contract.listFilesystems(),
  shares: (): Promise<Share[]> => contract.listShares(),
  previewShare: (share: Share): Promise<SharePreview> => contract.previewShare(share),
  createShare: (share: Share, options?: MutationOptions): Promise<{ share: Share; job: Job }> => contract.createShare(share, mutationHeaders(options)),
  acls: (): Promise<Ace[]> => contract.listAcls(),
  identities: (): Promise<Principal[]> => contract.listIdentities(),
  /** @deprecated Use acls(). The server retains /api/v1/acl only for old clients. */
  acl: (): Promise<Ace[]> => contract.listAcls(),
  /** @deprecated Use identities(). The server retains /api/v1/principals only for old clients. */
  principals: (): Promise<Principal[]> => contract.listIdentities(),
  effectiveAcl: (principal: string): Promise<EffectiveAcl> => contract.calculateEffectiveAcl(principal),
  simulateAclConversion: (filesystemId: string, target: string): Promise<AclConversionReport> => contract.simulateAclConversion(filesystemId, target),
  domain: (): Promise<DomainState> => contract.getDomain(),
  testDomain: (): Promise<DomainTestResult> => contract.testDomain(),
  printers: (): Promise<Printer[]> => contract.listPrinters(),
  printDrivers: (): Promise<PrintDriver[]> => contract.listPrintDrivers(),
  dfs: (): Promise<DfsRoot[]> => contract.listDfs(),
  quotas: (): Promise<Quota[]> => contract.listQuotas(),
  services: (): Promise<ServiceInfo[]> => contract.listServices(),
  serviceAction: (id: string, action: string, options?: MutationOptions): Promise<ServiceActionResponse> => contract.serviceAction(id, action, mutationHeaders(options)),
  audit: (): Promise<AuditEvent[]> => contract.listAudit(),
  jobs: (): Promise<Job[]> => contract.listJobs(),
  cancelJob: (id: string, options?: MutationOptions) => contract.cancelJob(id, mutationHeaders(options)),
  rollbackJob: (id: string, options?: MutationOptions): Promise<RollbackResponse> => contract.rollbackJob(id, mutationHeaders(options)),
  changeRequests: (status?: ChangeRequest['status']) => contract.listChangeRequests(status),
  createChangeRequest: (change: ChangeRequestInput, options?: MutationOptions) => contract.createChangeRequest(change, mutationHeaders(options)),
  approveChangeRequest: (id: string, reason: string, options?: MutationOptions) => contract.approveChangeRequest(id, { reason }, mutationHeaders(options)),
  rejectChangeRequest: (id: string, reason: string, options?: MutationOptions) => contract.rejectChangeRequest(id, { reason }, mutationHeaders(options)),
  executeChangeRequest: (id: string, options?: MutationOptions) => contract.executeApprovedChangeRequest(id, mutationHeaders(options)),
  capabilities: (): Promise<Capability[]> => contract.listCapabilities(),
  configFile: (path: string): Promise<ConfigurationFile> => contract.getConfigurationFile(path),
  validateConfig: (path: string, content: string): Promise<ConfigurationValidation> => contract.validateConfiguration(path, content),
  eventsUrl: (lastEventId?: string): string => {
    if (!lastEventId) return '/api/v1/events';
    return `/api/v1/events?lastEventId=${encodeURIComponent(lastEventId)}`;
  }
};
