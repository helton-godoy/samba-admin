import { createHmac } from 'node:crypto';
import { expect, request, test, type APIRequestContext, type Page } from '@playwright/test';

const password = process.env.SAMBA_ADMIN_E2E_PASSWORD;
const totpSecret = process.env.SAMBA_ADMIN_E2E_TOTP_SECRET;
if (!password || !totpSecret) {
  throw new Error('As credenciais efêmeras da suíte integrada não foram fornecidas.');
}

const username = 'admin-e2e';
const primaryURL = `http://127.0.0.1:${process.env.SAMBA_ADMIN_E2E_API_PORT || '18080'}`;
const unavailableURL = `http://127.0.0.1:${process.env.SAMBA_ADMIN_E2E_UNAVAILABLE_PORT || '18081'}`;
const expiryURL = `http://127.0.0.1:${process.env.SAMBA_ADMIN_E2E_EXPIRY_PORT || '18082'}`;
const mfaURL = `http://127.0.0.1:${process.env.SAMBA_ADMIN_E2E_MFA_PORT || '18083'}`;

interface AuthenticatedSession {
  api: APIRequestContext;
  cookie: string;
  csrf: string;
}

interface Problem {
  type: string;
  title: string;
  status: number;
  detail: string;
  correlationId: string;
}

async function authenticate(baseURL = primaryURL): Promise<AuthenticatedSession> {
  const api = await request.newContext({ baseURL, userAgent: 'samba-admin-e2e-real' });
  const response = await api.post('/api/v1/auth/login', {
    data: { username, password }
  });
  if (response.status() !== 200) {
    await api.dispose();
    throw new Error(`Login E2E recusado com HTTP ${response.status()}.`);
  }
  const body = await response.json() as { csrfToken?: string; mfaRequired?: boolean };
  if (body.mfaRequired) {
    await api.dispose();
    throw new Error('A API primária solicitou MFA inesperadamente.');
  }
  const cookie = response.headers()['set-cookie']?.split(';', 1)[0];
  const csrf = response.headers()['x-csrf-token'] || body.csrfToken;
  if (!cookie || !csrf) {
    await api.dispose();
    throw new Error('A sessão E2E não retornou cookie e CSRF.');
  }
  return { api, cookie, csrf };
}

function authenticatedHeaders(session: AuthenticatedSession, mutation = false): Record<string, string> {
  const headers: Record<string, string> = { Cookie: session.cookie };
  if (mutation) {
    headers['X-CSRF-Token'] = session.csrf;
    headers['Idempotency-Key'] = `e2e-${crypto.randomUUID()}`;
  }
  return headers;
}

async function authenticateInBrowser(page: Page): Promise<void> {
  await page.goto('/login');
  await page.getByLabel('Usuario').fill(username);
  await page.getByLabel('Senha').fill(password);
  await page.getByRole('button', { name: 'Entrar' }).click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByRole('heading', { name: 'Painel' })).toBeVisible();
}

async function createHealthCheckJob(): Promise<{ id: string }> {
  const session = await authenticate();
  try {
    const response = await session.api.post('/api/v1/services/svc-smbd/action', {
      headers: authenticatedHeaders(session, true),
      data: { action: 'health-check' }
    });
    if (response.status() !== 202) {
      throw new Error(`Criação da tarefa de leitura falhou com HTTP ${response.status()}.`);
    }
    const body = await response.json() as { job?: { id?: string } };
    if (!body.job?.id) throw new Error('A API não retornou o identificador da tarefa.');
    return { id: body.job.id };
  } finally {
    await session.api.dispose();
  }
}

function decodeBase32(value: string): Buffer {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
  let bits = '';
  for (const char of value.toUpperCase().replaceAll('=', '')) {
    const index = alphabet.indexOf(char);
    if (index < 0) throw new Error('Segredo TOTP E2E inválido.');
    bits += index.toString(2).padStart(5, '0');
  }
  const bytes: number[] = [];
  for (let offset = 0; offset + 8 <= bits.length; offset += 8) {
    bytes.push(Number.parseInt(bits.slice(offset, offset + 8), 2));
  }
  return Buffer.from(bytes);
}

function currentTotp(secret: string): string {
  const counter = Buffer.alloc(8);
  counter.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30_000)));
  const digest = createHmac('sha1', decodeBase32(secret)).update(counter).digest();
  const offset = digest[digest.length - 1] & 0x0f;
  const value = ((digest[offset] & 0x7f) << 24)
    | (digest[offset + 1] << 16)
    | (digest[offset + 2] << 8)
    | digest[offset + 3];
  return String(value % 1_000_000).padStart(6, '0');
}

test('1. login local pela interface contra a API Go', async ({ page }) => {
  await authenticateInBrowser(page);
  await expect(page.getByText(/Administrador local de emergência/i)).toBeVisible();
});

test('2. logout invalida a sessão real', async ({ page }) => {
  await authenticateInBrowser(page);
  await page.getByTitle('Encerrar sessao').click();
  await expect(page).toHaveURL(/\/login$/);
  const me = await page.request.get('/api/v1/auth/me');
  expect(me.status()).toBe(401);
});

test('3. sessão expirada retorna RFC 7807 e exige nova autenticação', async () => {
  const session = await authenticate(expiryURL);
  try {
    await new Promise((resolve) => setTimeout(resolve, 1_500));
    const response = await session.api.get('/api/v1/auth/me', { headers: { Cookie: session.cookie } });
    expect(response.status()).toBe(401);
    const problem = await response.json() as Problem;
    expect(problem.type).toContain('session_expired');
    expect(problem.correlationId).toBeTruthy();
  } finally {
    await session.api.dispose();
  }
});

test('4. CSRF ausente bloqueia mutação autenticada', async () => {
  const session = await authenticate();
  try {
    const response = await session.api.post('/api/v1/services/svc-smbd/action', {
      headers: { Cookie: session.cookie, 'Idempotency-Key': `e2e-${crypto.randomUUID()}` },
      data: { action: 'health-check' }
    });
    expect(response.status()).toBe(403);
    const problem = await response.json() as Problem;
    expect(problem.type).toContain('csrf_failed');
    expect(problem.correlationId).toBeTruthy();
  } finally {
    await session.api.dispose();
  }
});

test('5. MFA TOTP conclui o desafio de login real', async () => {
  const api = await request.newContext({ baseURL: mfaURL, userAgent: 'samba-admin-e2e-mfa' });
  try {
    const begin = await api.post('/api/v1/auth/login', { data: { username, password } });
    expect(begin.status()).toBe(200);
    const challenge = await begin.json() as { mfaRequired: boolean; mfaChallengeId?: string };
    expect(challenge.mfaRequired).toBe(true);
    expect(challenge.mfaChallengeId).toBeTruthy();

    const verification = await api.post('/api/v1/auth/mfa/verify', {
      data: { challengeId: challenge.mfaChallengeId, code: currentTotp(totpSecret) }
    });
    expect(verification.status()).toBe(200);
    const authenticated = await verification.json() as { mfaRequired: boolean; user?: { username: string } };
    expect(authenticated.mfaRequired).toBe(false);
    expect(authenticated.user?.username).toBe(username);
  } finally {
    await api.dispose();
  }
});

test('6. dashboard usa respostas da API Go sem MSW', async ({ page }) => {
  await authenticateInBrowser(page);
  await expect(page.getByText('Matriz de viabilidade detectada')).toBeVisible();
  await expect(page.getByText('15.1-RELEASE-p1')).toBeVisible();
});

test('7. capabilities reais preservam IDs canônicos e justificativas', async () => {
  const session = await authenticate();
  try {
    const response = await session.api.get('/api/v1/capabilities', { headers: authenticatedHeaders(session) });
    expect(response.status()).toBe(200);
    const capabilities = await response.json() as Array<{ id: string; state: string; evidence: string }>;
    expect(capabilities).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: 'system.inspect' }),
      expect.objectContaining({ id: 'cups.inspect' })
    ]));
    expect(capabilities.every((capability) => capability.evidence.length > 0)).toBe(true);
  } finally {
    await session.api.dispose();
  }
});

test('8. inventário de sistema vem do executor fixture pelo agente UDS', async () => {
  const session = await authenticate();
  try {
    const response = await session.api.get('/api/v1/system', { headers: authenticatedHeaders(session) });
    expect(response.status()).toBe(200);
    const system = await response.json() as { hostname: string; freebsdVersion: string };
    expect(system.hostname).toBe('{HOSTNAME}');
    expect(system.freebsdVersion).toBe('15.1-RELEASE-p1');
  } finally {
    await session.api.dispose();
  }
});

test('9. inventário Samba estruturado atravessa API e agente', async () => {
  const session = await authenticate();
  try {
    const response = await session.api.get('/api/v1/samba', { headers: authenticatedHeaders(session) });
    expect(response.status()).toBe(200);
    const samba = await response.json() as { version: string; package: string; packageBuildOptions: string[] };
    expect(samba.version).toBe('4.23.8');
    expect(samba.package).toContain('samba423');
    expect(samba.packageBuildOptions.length).toBeGreaterThan(0);
  } finally {
    await session.api.dispose();
  }
});

test('10. inventário CUPS estruturado atravessa API e agente', async () => {
  const session = await authenticate();
  try {
    const response = await session.api.get('/api/v1/cups', { headers: authenticatedHeaders(session) });
    expect(response.status()).toBe(200);
    const cups = await response.json() as { version: string; service: string; printers: unknown[]; errors: string[] };
    expect(cups.version).toContain('cups-2.4.19');
    expect(cups.service).toBe('valid');
    expect(cups.printers).toEqual([]);
    expect(cups.errors.length).toBeGreaterThan(0);
    expect(cups.errors.every((error) => error.length > 0)).toBe(true);
  } finally {
    await session.api.dispose();
  }
});

test('11. inventário de domínio permanece diagnóstico somente leitura', async () => {
  const session = await authenticate();
  try {
    const response = await session.api.get('/api/v1/domain', { headers: authenticatedHeaders(session) });
    expect(response.status()).toBe(200);
    const domain = await response.json() as { joined: boolean; tests: Array<{ name: string }> };
    expect(domain.joined).toBe(false);
    expect(domain.tests.length).toBeGreaterThan(0);
  } finally {
    await session.api.dispose();
  }
});

test('12. inventário de filesystem exibe a montagem coletada', async ({ page }) => {
  await authenticateInBrowser(page);
  await page.goto('/sistemas-arquivos');
  await expect(page.getByRole('heading', { name: 'Sistemas de arquivos' })).toBeVisible();
  await expect(page.getByText('/dev/vtbd0s1a')).toBeVisible();
  await expect(page.getByText('/', { exact: true })).toBeVisible();
});

test('13. jobs persistentes incluem tarefa de verificação somente leitura', async () => {
  const created = await createHealthCheckJob();
  const session = await authenticate();
  try {
    await expect.poll(async () => {
      const response = await session.api.get('/api/v1/jobs', { headers: authenticatedHeaders(session) });
      const jobs = await response.json() as Array<{ id: string }>;
      return jobs.some((job) => job.id === created.id);
    }).toBe(true);
  } finally {
    await session.api.dispose();
  }
});

test('14. SSE real entrega atualização de job ao navegador', async ({ page }) => {
  await authenticateInBrowser(page);
  await page.goto('/tarefas');
  await expect(page.getByText('Atualizacao em tempo real conectada.', { exact: true })).toBeVisible();
  const created = await createHealthCheckJob();
  await expect(page.getByText(created.id)).toBeVisible();
});

test('15. SSE reconecta usando o último evento recebido', async ({ page, context }) => {
  await authenticateInBrowser(page);
  await page.goto('/tarefas');
  await expect(page.getByText('Atualizacao em tempo real conectada.', { exact: true })).toBeVisible();
  const created = await createHealthCheckJob();
  await expect(page.getByText(created.id)).toBeVisible();
  const lastEventId = await expect.poll(async () => page.evaluate(() => sessionStorage.getItem('samba-admin:last-job-event-id'))).not.toBeNull();
  void lastEventId;

  await context.setOffline(true);
  await expect(page.getByText(/Reconectando ao fluxo de eventos/)).toBeVisible();
  const reconnect = page.waitForRequest((candidate) => {
    const url = new URL(candidate.url());
    return url.pathname === '/api/v1/events' && Boolean(url.searchParams.get('lastEventId'));
  });
  await context.setOffline(false);
  await reconnect;
  await expect(page.getByText('Atualizacao em tempo real conectada.', { exact: true })).toBeVisible();
});

test('16. falhas consecutivas do SSE ativam polling de contingência', async ({ page }) => {
  let jobRequests = 0;
  page.on('request', (candidate) => {
    if (new URL(candidate.url()).pathname === '/api/v1/jobs') jobRequests += 1;
  });
  await page.route('**/api/v1/events**', (route) => route.abort('failed'));
  await authenticateInBrowser(page);
  await page.goto('/tarefas');
  await expect(page.getByText(/SSE indisponivel; atualizacao por polling/)).toBeVisible({ timeout: 30_000 });
  await expect.poll(() => jobRequests, { timeout: 12_000 }).toBeGreaterThan(1);
});

test('17. agente indisponível produz 503 estruturado', async () => {
  const session = await authenticate(unavailableURL);
  try {
    const response = await session.api.get('/api/v1/system', { headers: authenticatedHeaders(session) });
    expect(response.status()).toBe(503);
    const problem = await response.json() as Problem;
    expect(problem.type).toContain('capability_unavailable');
    expect(problem.correlationId).toBeTruthy();
  } finally {
    await session.api.dispose();
  }
});

test('18. readiness retorna 503 quando o agente não está disponível', async () => {
  const api = await request.newContext({ baseURL: unavailableURL });
  try {
    const response = await api.get('/readyz');
    expect(response.status()).toBe(503);
    const problem = await response.json() as Problem;
    expect(problem.type).toContain('not_ready');
    expect(problem.correlationId).toBeTruthy();
  } finally {
    await api.dispose();
  }
});

test('19. falha do inventário CUPS retorna 503 sem fallback mutável', async () => {
  const session = await authenticate(unavailableURL);
  try {
    const response = await session.api.get('/api/v1/cups', { headers: authenticatedHeaders(session) });
    expect(response.status()).toBe(503);
    const problem = await response.json() as Problem;
    expect(problem.title).toContain('CUPS');
    expect(problem.correlationId).toBeTruthy();
  } finally {
    await session.api.dispose();
  }
});

test('20. mutação bloqueada preserva correlation ID de ponta a ponta', async () => {
  const session = await authenticate();
  const correlationId = `corr-e2e-${crypto.randomUUID()}`;
  try {
    const response = await session.api.post('/api/v1/services/svc-cupsd/action', {
      headers: {
        ...authenticatedHeaders(session, true),
        'X-Correlation-ID': correlationId
      },
      data: { action: 'stop' }
    });
    expect(response.status()).toBe(503);
    const problem = await response.json() as Problem;
    expect(problem.type).toContain('mutation_blocked');
    expect(problem.correlationId).toBe(correlationId);
    expect(response.headers()['x-correlation-id']).toBe(correlationId);
  } finally {
    await session.api.dispose();
  }
});
