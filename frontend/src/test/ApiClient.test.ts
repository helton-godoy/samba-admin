import { describe, expect, it } from 'vitest';
import { http, HttpResponse } from 'msw';
import { api, CSRF_STORAGE_KEY } from '../api/client';
import { server } from '../mocks/server';
import { systemInfo } from '../mocks/data';

describe('API client', () => {
  it('usa recursos canônicos de ACL/identidade e captura CSRF após o login', async () => {
    const login = await api.auth.login({ username: 'admin.homologacao', password: 'senha-de-teste' });
    const [acls, identities] = await Promise.all([api.acls(), api.identities()]);

    expect(login).toMatchObject({ status: 'authenticated', user: { username: 'admin.homologacao' } });
    expect(sessionStorage.getItem(CSRF_STORAGE_KEY)).toBe('mock-csrf-token');
    expect(acls.length).toBeGreaterThan(0);
    expect(identities.length).toBeGreaterThan(0);
  });

  it('normalizes RFC 7807 errors with their correlation ID', async () => {
    server.use(http.get('/api/v1/system', () => HttpResponse.json({
      type: 'resource_conflict',
      title: 'Conflito',
      status: 409,
      detail: 'O recurso foi alterado por outra tarefa.',
      correlationId: 'corr-test-problem'
    }, {
      status: 409,
      headers: { 'Content-Type': 'application/problem+json', 'X-Correlation-ID': 'corr-test-problem' }
    })));

    await expect(api.system()).rejects.toMatchObject({
      status: 409,
      code: 'resource_conflict',
      correlationId: 'corr-test-problem',
      message: 'O recurso foi alterado por outra tarefa.'
    });
  });

  it('recupera o CSRF pelo endpoint de sessão antes de renovar', async () => {
    sessionStorage.removeItem(CSRF_STORAGE_KEY);
    server.use(
      http.get('/api/v1/auth/me', () => HttpResponse.json({ id: 'user-1', username: 'admin', displayName: 'Admin', roles: ['administrador do sistema'] }, {
        headers: { 'X-CSRF-Token': 'csrf-bootstrap' }
      })),
      http.post('/api/v1/auth/session/renew', ({ request }) => {
        expect(request.headers.get('X-CSRF-Token')).toBe('csrf-bootstrap');
        return HttpResponse.json({
          user: { id: 'user-1', username: 'admin', displayName: 'Admin', roles: ['administrador do sistema'] },
          csrfToken: 'csrf-rotated',
          mfaRequired: false
        }, { headers: { 'X-CSRF-Token': 'csrf-rotated' } });
      })
    );
    await expect(api.auth.refresh()).resolves.toMatchObject({ username: 'admin' });
    expect(sessionStorage.getItem(CSRF_STORAGE_KEY)).toBe('csrf-rotated');
  });

  it('não envia o token CSRF em requisições GET', async () => {
    await api.auth.login({ username: 'admin.homologacao', password: 'senha-de-teste' });
    server.use(http.get('/api/v1/system', ({ request }) => {
      expect(request.headers.get('X-CSRF-Token')).toBeNull();
      return HttpResponse.json(systemInfo);
    }));

    await expect(api.system()).resolves.toMatchObject({ hostname: systemInfo.hostname });
  });

  it('consome os inventários Samba e CUPS sem substituir printers e shares', async () => {
    const [samba, cups, printers, shares] = await Promise.all([
      api.samba(),
      api.cups(),
      api.printers(),
      api.shares()
    ]);

    expect(samba).toMatchObject({ package: 'samba423-4.23.2 (simulado)', testparmAvailable: true });
    expect(cups).toMatchObject({ service: 'valid' });
    expect(printers[0]?.id).toBe('prn-1');
    expect(shares[0]?.id).toBe('share-assistencial');
  });
});
