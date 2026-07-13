import { delay, http, HttpResponse } from 'msw';
import {
  aclEntries,
  auditEvents,
  capabilities,
  cupsInfo,
  dfsRoots,
  domainState,
  filesystems,
  jobs,
  printers,
  printDrivers,
  principals,
  quotas,
  sambaInfo,
  services,
  shares,
  systemInfo
} from './data';
import type { Job, Share } from '../types';
import type { ChangeRequest, ChangeRequestInput } from '../api/generated';

const createdShares: Share[] = [];
const createdJobs: Job[] = [];
const recoveredJob: Job = {
  id: 'JOB-RECOVERED-001',
  requestedBy: 'admin.mock',
  requestedAt: '2026-07-12T08:00:00-03:00',
  operation: 'Tarefa recuperada após reconexão',
  status: 'running',
  progress: 73,
  summary: 'Estado recuperado pela API após o último evento conhecido.',
  cancellable: true,
  rollbackAvailable: true
};
const initialChangeRequest = (): ChangeRequest => ({
  id: 'change-0001',
  operation: 'samba.service.restart',
  resource: 'service:smbd',
  risk: 'high',
  status: 'pending_approval',
  requestedBy: 'solicitante.mock',
  requestedAt: '2026-07-12T00:00:00-03:00',
  justification: 'Aplicar manutenção aprovada no serviço de arquivos.',
  maintenanceStart: '2026-07-13T01:00:00-03:00',
  maintenanceEnd: '2026-07-13T02:00:00-03:00',
  impact: 'Sessões SMB ativas podem ser temporariamente interrompidas.',
  rollbackPlan: 'Restaurar a configuração anterior e reiniciar o serviço.'
});
const changeRequests: ChangeRequest[] = [initialChangeRequest()];
let authenticated = true;

const mockUser = {
  id: 'mock-admin',
  username: 'admin.mock',
  displayName: 'Administrador de homologacao',
  roles: ['administrador do sistema']
};

export function resetMockState() {
  createdShares.splice(0, createdShares.length);
  createdJobs.splice(0, createdJobs.length);
  changeRequests.splice(0, changeRequests.length, initialChangeRequest());
  authenticated = true;
}

const simulate = async (ms = 280) => delay(ms);
const mockScenario = (request: Request) => request.headers.get('X-Mock-Scenario') || '';

function problem(status: number, type: string, title: string, detail: string, correlationId: string) {
  return HttpResponse.json({ type, title, status, detail, correlationId }, {
    status,
    headers: { 'Content-Type': 'application/problem+json', 'X-Correlation-ID': correlationId }
  });
}

export const handlers = [
  http.post('/api/v1/auth/login', async ({ request }) => {
    await simulate(180);
    const credentials = await request.json() as { username?: string; password?: string };
    if (!credentials.username || !credentials.password || credentials.password === 'invalida') {
      return HttpResponse.json({
        type: 'invalid_credentials',
        title: 'Falha de autenticacao',
        status: 401,
        detail: 'Usuario ou senha invalidos.',
        correlationId: 'corr-mock-login-failed'
      }, { status: 401, headers: { 'Content-Type': 'application/problem+json', 'X-Correlation-ID': 'corr-mock-login-failed' } });
    }
    authenticated = true;
    if (credentials.password === 'mfa-required') {
      return HttpResponse.json({
        expiresAt: new Date(Date.now() + 15 * 60_000).toISOString(),
        mfaRequired: true,
        mfaChallengeId: 'mock-mfa-challenge'
      }, { headers: { 'X-Correlation-ID': 'corr-mock-login-mfa' } });
    }
    return HttpResponse.json({
      user: { ...mockUser, username: credentials.username },
      csrfToken: 'mock-csrf-token',
      expiresAt: new Date(Date.now() + 15 * 60_000).toISOString(),
      mfaRequired: false
    }, {
      headers: {
        'X-CSRF-Token': 'mock-csrf-token',
        'X-Correlation-ID': 'corr-mock-login-ok'
      }
    });
  }),
  http.post('/api/v1/auth/mfa/verify', async ({ request }) => {
    await simulate(120);
    const verification = await request.json() as { challengeId?: string; code?: string };
    if (verification.challengeId !== 'mock-mfa-challenge' || verification.code !== '000000') {
      return HttpResponse.json({
        type: 'mfa_invalid', title: 'Codigo MFA invalido', status: 401,
        detail: 'O codigo informado nao foi aceito.', correlationId: 'corr-mock-mfa-failed'
      }, { status: 401, headers: { 'Content-Type': 'application/problem+json' } });
    }
    return HttpResponse.json({
      user: mockUser,
      csrfToken: 'mock-csrf-token',
      expiresAt: new Date(Date.now() + 15 * 60_000).toISOString(),
      mfaRequired: false
    }, { headers: { 'X-CSRF-Token': 'mock-csrf-token' } });
  }),
  http.post('/api/v1/auth/session/renew', async ({ request }) => {
    await simulate(60);
    if (!authenticated || mockScenario(request) === 'session-expired') {
      return HttpResponse.json({
        type: 'session_invalid', title: 'Sessao invalida', status: 401,
        detail: 'A sessao administrativa expirou.', correlationId: 'corr-mock-session-expired'
      }, { status: 401, headers: { 'Content-Type': 'application/problem+json' } });
    }
    return HttpResponse.json({
      user: mockUser,
      csrfToken: 'mock-csrf-token',
      expiresAt: new Date(Date.now() + 15 * 60_000).toISOString(),
      mfaRequired: false
    }, { headers: { 'X-CSRF-Token': 'mock-csrf-token' } });
  }),
  http.post('/api/v1/auth/logout', async () => {
    await simulate(80);
    authenticated = false;
    return new HttpResponse(null, { status: 204 });
  }),
  http.get('/api/v1/auth/me', async () => {
    await simulate(80);
    if (!authenticated) {
      return HttpResponse.json({
        type: 'session_invalid',
        title: 'Sessao invalida',
        status: 401,
        detail: 'A sessao administrativa expirou.',
        correlationId: 'corr-mock-session-expired'
      }, { status: 401, headers: { 'Content-Type': 'application/problem+json', 'X-Correlation-ID': 'corr-mock-session-expired' } });
    }
    return HttpResponse.json(mockUser, { headers: { 'X-CSRF-Token': 'mock-csrf-token' } });
  }),
  http.get('/api/v1/system', async () => {
    await simulate();
    return HttpResponse.json(systemInfo);
  }),
  http.get('/api/v1/samba', async ({ request }) => {
    await simulate();
    if (mockScenario(request) === 'agent-unavailable') {
      return problem(503, 'agent_unavailable', 'Agente indisponível', 'O inventário Samba exige conexão com o agente somente leitura.', 'corr-mock-agent-samba');
    }
    return HttpResponse.json(sambaInfo);
  }),
  http.get('/api/v1/cups', async ({ request }) => {
    await simulate();
    const scenario = mockScenario(request);
    if (scenario === 'agent-unavailable' || scenario === 'cups-error') {
      return problem(503, scenario === 'cups-error' ? 'capability_unavailable' : 'agent_unavailable', 'Inventário CUPS indisponível', 'O agente ou a capability CUPS somente leitura não está disponível.', 'corr-mock-agent-cups');
    }
    return HttpResponse.json(cupsInfo);
  }),
  http.get('/api/v1/filesystems', async () => {
    await simulate();
    return HttpResponse.json(filesystems);
  }),
  http.get('/api/v1/shares', async () => {
    await simulate();
    return HttpResponse.json([...shares, ...createdShares]);
  }),
  http.post('/api/v1/shares/preview', async ({ request }) => {
    await simulate(450);
    const share = (await request.json()) as Partial<Share>;
    if (!share.name || !share.path) {
      return HttpResponse.json(
        { code: 'VALIDATION_REFUSED', message: 'Nome e caminho são obrigatórios.' },
        { status: 422 }
      );
    }
    if (!String(share.path).startsWith('/srv/dados/')) {
      return HttpResponse.json(
        {
          code: 'PATH_NOT_ALLOWED',
          message: 'O caminho deve estar em um volume autorizado. Diretórios do sistema são bloqueados.'
        },
        { status: 422 }
      );
    }
    const config = `[${share.name}]\n  comment = ${share.description || 'Compartilhamento gerenciado'}\n  path = ${share.path}\n  browseable = yes\n  read only = ${share.readOnly ? 'yes' : 'no'}\n  guest ok = no\n  server smb encrypt = ${share.encryption === 'required' ? 'required' : 'desired'}\n  server signing = ${share.signing === 'mandatory' ? 'mandatory' : 'default'}\n  valid users = ${(share.allowedPrincipals || []).join(' ') || '@EBSERHNET\\Domain Users'}\n  vfs objects = ${(share.vfsModules || ['acl_xattr']).join(' ')}\n`;
    return HttpResponse.json({
      valid: true,
      testparm: 'Loaded services file OK. Server role: ROLE_DOMAIN_MEMBER',
      reloadRequired: true,
      restartRequired: false,
      config,
      diff: `--- smb4.conf.anterior\n+++ smb4.conf.proposto\n@@\n+${config.replaceAll('\n', '\n+')}`,
      impact: {
        activeSessions: systemInfo.smbSessions,
        openFiles: systemInfo.openFiles,
        affectedShares: [share.name]
      }
    });
  }),
  http.post('/api/v1/shares', async ({ request }) => {
    await simulate(500);
    if (mockScenario(request) === 'conflict') {
      return problem(409, 'edit_conflict', 'Conflito de edição', 'A configuração foi alterada por outro administrador.', 'corr-mock-conflict');
    }
    if (mockScenario(request) === 'agent-error') {
      return problem(502, 'agent_unavailable', 'Agente indisponível', 'O agente privilegiado recusou a operação.', 'corr-mock-agent-error');
    }
    const payload = (await request.json()) as Share;
    if (payload.guestAccess) {
      return HttpResponse.json(
        { code: 'POLICY_DENIED', message: 'A política do protótipo bloqueia acesso guest.' },
        { status: 403 }
      );
    }
    const newShare = { ...payload, id: `share-${Date.now()}` };
    createdShares.push(newShare);
    const job: Job = {
      id: `JOB-${Date.now()}`,
      requestedBy: 'EBSERHNET\\adm.hlgodoy',
      requestedAt: new Date().toISOString(),
      operation: `Criar compartilhamento ${payload.name}`,
      status: 'running',
      progress: 62,
      summary: 'Backup e validação concluídos; aguardando verificação de saúde.',
      cancellable: true,
      rollbackAvailable: true
    };
    createdJobs.push(job);
    return HttpResponse.json({ share: newShare, job }, { status: 201 });
  }),
  http.get('/api/v1/acl', async () => {
    await simulate();
    return HttpResponse.json(aclEntries, { headers: { Deprecation: 'true', Link: '</api/v1/acls>; rel="successor-version"' } });
  }),
  http.get('/api/v1/acls', async () => {
    await simulate();
    return HttpResponse.json(aclEntries);
  }),
  http.get('/api/v1/principals', async () => {
    await simulate();
    return HttpResponse.json(principals, { headers: { Deprecation: 'true', Link: '</api/v1/identities>; rel="successor-version"' } });
  }),
  http.get('/api/v1/identities', async () => {
    await simulate();
    return HttpResponse.json(principals);
  }),
  http.post('/api/v1/acl/effective', async ({ request }) => {
    await simulate(400);
    const body = (await request.json()) as { principal: string };
    return HttpResponse.json({
      principal: body.principal,
      effective: ['Ler e executar', 'Criar arquivos', 'Criar pastas', 'Excluir próprios arquivos'],
      denied: ['Alterar proprietário', 'Alterar ACL'],
      warnings: ['Resultado simulado; grupos aninhados e tokens reais devem ser resolvidos no backend.']
    });
  }),
  http.post('/api/v1/acl/conversion/simulate', async ({ request }) => {
    await simulate(900);
    const body = (await request.json()) as { filesystemId: string; target: string };
    const fs = filesystems.find((item) => item.id === body.filesystemId);
    if (!fs) {
      return HttpResponse.json({ message: 'Sistema de arquivos não encontrado.' }, { status: 404 });
    }
    return HttpResponse.json({
      filesystem: fs.mountPoint,
      from: fs.aclModel,
      to: body.target,
      requiresMaintenance: true,
      requiresUnmount: true,
      backupArtifact: `/var/backups/samba-admin/acl-${body.filesystemId}-20260711.jsonl`,
      affectedShares: fs.associatedShares,
      scannedObjects: 184203,
      preserved: 173910,
      approximated: 9911,
      notRepresentable: 382,
      risks: [
        'A ordem e a precedência de ACEs DENY podem sofrer alteração.',
        'Máscaras POSIX não possuem equivalência perfeita no modelo NFSv4.',
        'Entradas com identidades não resolvidas exigem mapeamento manual.',
        'A reversão restaura o inventário exportado, mas não garante equivalência semântica perfeita.'
      ],
      plan: [
        'Bloquear novas alterações e notificar usuários.',
        'Encerrar sessões e verificar arquivos abertos.',
        'Exportar ACLs, donos, grupos e metadados.',
        'Desmontar o sistema de arquivos em janela de manutenção.',
        'Alterar /etc/fstab de acls para nfsv4acls.',
        'Montar, reaplicar ACLs mapeadas e executar testes de acesso.',
        'Reabrir o serviço apenas após verificação de saúde.'
      ]
    });
  }),
  http.get('/api/v1/domain', async () => {
    await simulate();
    return HttpResponse.json(domainState);
  }),
  http.post('/api/v1/domain/test', async () => {
    await simulate(850);
    return HttpResponse.json({
      success: true,
      checks: domainState.tests,
      commandsModeled: ['host', 'drill', 'kinit', 'klist', 'net ads testjoin', 'wbinfo --ping-dc', 'getent passwd', 'smbclient']
    });
  }),
  http.get('/api/v1/printers', async () => {
    await simulate();
    return HttpResponse.json(printers);
  }),
  http.get('/api/v1/print-drivers', async () => {
    await simulate();
    return HttpResponse.json(printDrivers);
  }),
  http.get('/api/v1/dfs', async () => {
    await simulate();
    return HttpResponse.json(dfsRoots);
  }),
  http.get('/api/v1/quotas', async () => {
    await simulate();
    return HttpResponse.json(quotas);
  }),
  http.get('/api/v1/services', async ({ request }) => {
    await simulate();
    if (mockScenario(request) === 'permission-denied') {
      return problem(403, 'permission_denied', 'Permissão insuficiente', 'O papel atual não pode consultar serviços.', 'corr-mock-denied');
    }
    return HttpResponse.json(services);
  }),
  http.post('/api/v1/services/:id/action', async ({ params, request }) => {
    await simulate(600);
    const body = (await request.json()) as { action: string };
    return HttpResponse.json({
      id: params.id,
      action: body.action,
      accepted: true,
      warning: body.action === 'restart' ? 'Sessões SMB e trabalhos de impressão podem ser interrompidos.' : undefined
    });
  }),
  http.get('/api/v1/audit', async () => {
    await simulate();
    return HttpResponse.json(auditEvents);
  }),
  http.get('/api/v1/jobs', async ({ request }) => {
    await simulate();
    const scenario = mockScenario(request);
    if (scenario === 'agent-error') {
      return problem(502, 'agent_unavailable', 'Agente indisponível', 'Não foi possível consultar o estado da tarefa no agente.', 'corr-mock-agent-jobs');
    }
    if (scenario === 'sse-recover') {
      return HttpResponse.json([recoveredJob, ...createdJobs, ...jobs]);
    }
    return HttpResponse.json([...createdJobs, ...jobs]);
  }),
  http.post('/api/v1/jobs/:id/cancel', async ({ params }) => {
    await simulate(180);
    return HttpResponse.json({ jobId: params.id, status: 'cancellation-requested' }, { status: 202 });
  }),
  http.post('/api/v1/jobs/:id/rollback', async ({ params }) => {
    await simulate(700);
    return HttpResponse.json({
      id: params.id,
      status: 'rolled-back',
      message: 'Rollback simulado concluído e verificação de saúde aprovada.'
    });
  }),
  http.get('/api/v1/events', ({ request }) => {
    const encoder = new TextEncoder();
    const scenario = mockScenario(request);
    const lastEventId = new URL(request.url).searchParams.get('lastEventId');
    const stream = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(encoder.encode(`event: ready\ndata: {"type":"ready"}\n\n`));
        if (scenario === 'sse-recover' && lastEventId) {
          const payload = JSON.stringify({ type: 'job.updated', payload: recoveredJob, timestamp: new Date().toISOString() });
          controller.enqueue(encoder.encode(`id: event-recovered-001\nevent: job.updated\ndata: ${payload}\n\n`));
        }
        if (scenario === 'sse-loss') controller.close();
      }
    });
    return new HttpResponse(stream, {
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        Connection: 'keep-alive'
      }
    });
  }),
  http.get('/api/v1/change-requests', async ({ request }) => {
    await simulate(120);
    const status = new URL(request.url).searchParams.get('status');
    return HttpResponse.json(status ? changeRequests.filter((change) => change.status === status) : changeRequests);
  }),
  http.post('/api/v1/change-requests', async ({ request }) => {
    await simulate(180);
    const input = await request.json() as ChangeRequestInput;
    if (!input.maintenanceStart || !input.maintenanceEnd) {
      return problem(422, 'maintenance_window_required', 'Janela obrigatória', 'Mudanças de alto risco exigem início e fim da janela.', 'corr-mock-window');
    }
    const change: ChangeRequest = {
      id: `change-${Date.now()}`,
      operation: input.operation,
      resource: input.resource,
      risk: 'high',
      status: 'pending_approval',
      requestedBy: mockUser.username,
      requestedAt: new Date().toISOString(),
      justification: input.justification,
      maintenanceStart: input.maintenanceStart,
      maintenanceEnd: input.maintenanceEnd,
      impact: input.impact,
      rollbackPlan: input.rollbackPlan
    };
    changeRequests.unshift(change);
    return HttpResponse.json(change, { status: 201 });
  }),
  http.post('/api/v1/change-requests/:id/approve', async ({ params, request }) => {
    await simulate(140);
    const change = changeRequests.find((item) => item.id === params.id);
    if (!change) return problem(404, 'change_not_found', 'Solicitação não encontrada', 'A solicitação informada não existe.', 'corr-mock-change-not-found');
    if (change.requestedBy === mockUser.username && ['high', 'destructive', 'emergency'].includes(change.risk)) {
      return problem(409, 'self_approval_denied', 'Segregação obrigatória', 'O solicitante não pode aprovar a própria mudança.', 'corr-mock-self-approval');
    }
    const decision = await request.json() as { reason?: string };
    change.status = 'approved';
    change.approvedBy = mockUser.username;
    change.approvedAt = new Date().toISOString();
    change.decisionReason = decision.reason;
    return HttpResponse.json(change);
  }),
  http.post('/api/v1/change-requests/:id/reject', async ({ params, request }) => {
    await simulate(140);
    const change = changeRequests.find((item) => item.id === params.id);
    if (!change) return problem(404, 'change_not_found', 'Solicitação não encontrada', 'A solicitação informada não existe.', 'corr-mock-change-not-found');
    const decision = await request.json() as { reason?: string };
    change.status = 'rejected';
    change.approvedBy = mockUser.username;
    change.approvedAt = new Date().toISOString();
    change.decisionReason = decision.reason;
    return HttpResponse.json(change);
  }),
  http.post('/api/v1/change-requests/:id/execute', async ({ params }) => {
    await simulate(180);
    const change = changeRequests.find((item) => item.id === params.id);
    if (!change) return problem(404, 'change_not_found', 'Solicitação não encontrada', 'A solicitação informada não existe.', 'corr-mock-change-not-found');
    if (change.status !== 'approved') return problem(409, 'approval_required', 'Aprovação obrigatória', 'A mudança precisa estar aprovada antes da execução.', 'corr-mock-approval-required');
    const job: Job = {
      id: `JOB-CHANGE-${Date.now()}`,
      requestedBy: change.requestedBy,
      requestedAt: new Date().toISOString(),
      operation: change.operation,
      status: 'queued',
      progress: 0,
      summary: 'Mudança aprovada aguardando execução simulada.',
      cancellable: true,
      rollbackAvailable: true
    };
    createdJobs.unshift(job);
    change.status = 'executing';
    change.jobId = job.id;
    return HttpResponse.json(job, { status: 202 });
  }),
  http.get('/api/v1/capabilities', async () => {
    await simulate();
    return HttpResponse.json(capabilities);
  }),
  http.get('/api/v1/configuration/file', async ({ request }) => {
    await simulate();
    const url = new URL(request.url);
    const path = url.searchParams.get('path') || '/usr/local/etc/smb4.conf';
    const allowed = [
      '/usr/local/etc/smb4.conf',
      '/etc/rc.conf',
      '/etc/fstab',
      '/etc/krb5.conf',
      '/etc/nsswitch.conf',
      '/etc/syslog.conf',
      '/usr/local/etc/cups/cupsd.conf'
    ];
    if (!allowed.includes(path)) {
      return HttpResponse.json({ message: 'Arquivo fora da lista autorizada.' }, { status: 403 });
    }
    const content = path.endsWith('smb4.conf')
      ? `[global]\n  workgroup = EBSERHNET\n  realm = EBSERHNET.EBSERH.GOV.BR\n  security = ADS\n  server min protocol = SMB2_10\n  map to guest = Never\n  idmap config * : backend = tdb\n  idmap config * : range = 1000000-1999999\n  idmap config EBSERHNET : backend = rid\n  idmap config EBSERHNET : range = 2000000-2999999\n`
      : path.endsWith('fstab')
        ? `/dev/nda1p1 / ufs rw 1 1\n/dev/nda1p2 /srv/dados ufs rw,nfsv4acls,userquota,groupquota 2 2\n`
        : `# Arquivo simulado: ${path}\n# O backend real aplicará validação, locking, backup e rollback.\n`;
    return HttpResponse.json({
      path,
      encoding: 'UTF-8',
      eol: 'LF',
      version: 'sha256:2b6292e0',
      lockedBy: null,
      size: content.length,
      content,
      previousContent: content.replace('map to guest = Never', 'map to guest = Bad User')
    });
  }),
  http.post('/api/v1/configuration/validate', async ({ request }) => {
    await simulate(650);
    const body = (await request.json()) as { path: string; content: string };
    if (body.content.includes('server min protocol = NT1')) {
      return HttpResponse.json(
        { valid: false, errors: ['SMB1/NT1 é bloqueado pela política de segurança.'] },
        { status: 422 }
      );
    }
    return HttpResponse.json({
      valid: true,
      validators: ['sintaxe', 'política de segurança', 'testparm conceitual'],
      diff: `--- versão anterior\n+++ versão proposta\n@@ validação simulada para ${body.path}\n`,
      backupRequired: true,
      reloadRequired: body.path.endsWith('smb4.conf')
    });
  })
];
