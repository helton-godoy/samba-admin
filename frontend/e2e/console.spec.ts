import { expect, test, type Page } from '@playwright/test';

async function autenticar(page: Page, senha = 'senha-de-teste') {
  await page.goto('/login');
  await page.getByLabel('Usuario').fill('admin.homologacao');
  await page.getByLabel('Senha').fill(senha);
  await page.getByRole('button', { name: 'Entrar' }).click();
}

async function preencherCompartilhamento(page: Page, nome: string, caminho = `/srv/dados/${nome.toLowerCase()}`) {
  await page.goto('/compartilhamentos/novo');
  await expect(page.getByRole('heading', { name: 'Novo compartilhamento' })).toBeVisible();
  await page.getByLabel('Nome').fill(nome);
  await page.getByLabel('Descrição').fill(`Documentos de homologação para ${nome}`);
  await page.getByLabel('Caminho autorizado').fill(caminho);
  await page.getByLabel('Usuário ou grupo permitido').selectOption({ index: 1 });
}

async function gerarPrevia(page: Page, nome: string) {
  await preencherCompartilhamento(page, nome);
  await page.getByRole('button', { name: 'Validar e gerar prévia' }).click();
  await expect(page.getByRole('heading', { name: 'Prévia antes da aplicação' })).toBeVisible();
}

test('1. autentica pela tela de login', async ({ page }) => {
  await autenticar(page);
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByRole('heading', { name: 'Painel' })).toBeVisible();
});

test('2. encerra a sessão administrativa', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Painel' })).toBeVisible();
  await page.getByTitle('Encerrar sessao').click();
  await expect(page).toHaveURL(/\/login$/);
});

test('3. redireciona ao login quando a sessão expira', async ({ page }) => {
  await page.setExtraHTTPHeaders({ 'X-Mock-Scenario': 'session-expired' });
  await page.goto('/');
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole('heading', { name: 'Console Samba' })).toBeVisible();
});

test('4. conclui autenticação com MFA TOTP', async ({ page }) => {
  await autenticar(page, 'mfa-required');
  await expect(page.getByText('A politica da sua conta exige confirmacao TOTP.')).toBeVisible();
  await page.getByLabel('Codigo de autenticacao').fill('000000');
  await page.getByRole('button', { name: 'Confirmar MFA' }).click();
  await expect(page).toHaveURL(/\/$/);
});

test('5. carrega o painel e a matriz de capabilities', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Painel' })).toBeVisible();
  await expect(page.getByText('Matriz de viabilidade detectada')).toBeVisible();
});

test('6. mantém capability AD DC não verificada desabilitada', async ({ page }) => {
  await page.goto('/ad-dc');
  await expect(page.getByRole('button', { name: 'Provisionar nova floresta' })).toBeDisabled();
  await expect(page.getByText('Perfil não habilitado')).toBeVisible();
});

test('7. cria compartilhamento apenas como tarefa simulada', async ({ page }) => {
  await gerarPrevia(page, 'E2ECriacao');
  await page.getByRole('button', { name: 'Confirmar aplicação simulada' }).click();
  await expect(page.getByText('Compartilhamento criado.')).toBeVisible();
  await expect(page.getByText(/JOB-/)).toBeVisible();
});

test('8. valida a configuração antes da aplicação', async ({ page }) => {
  await gerarPrevia(page, 'E2EValidacao');
  await expect(page.getByText('Loaded services file OK. Server role: ROLE_DOMAIN_MEMBER')).toBeVisible();
  await expect(page.getByText('Recarregar')).toBeVisible();
});

test('9. apresenta diff da configuração proposta', async ({ page }) => {
  await gerarPrevia(page, 'E2EDiff');
  await expect(page.getByText('Diff', { exact: true })).toBeVisible();
  await expect(page.getByText('+++ smb4.conf.proposto')).toBeVisible();
});

test('10. aprova mudança com aprovador segregado', async ({ page }) => {
  await page.goto('/mudancas');
  await expect(page.getByRole('heading', { name: 'Aprovação de mudanças' })).toBeVisible();
  await page.getByRole('button', { name: 'Aprovar change-0001' }).click();
  await expect(page.getByText('Aprovada', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Executar change-0001' })).toBeVisible();
});

test('11. lista tarefas persistentes', async ({ page }) => {
  await page.goto('/tarefas');
  await expect(page.getByRole('heading', { name: 'Central de tarefas' })).toBeVisible();
  await expect(page.getByText('JOB-2026-0711-0042')).toBeVisible();
});

test('12. conecta o acompanhamento SSE', async ({ page }) => {
  await page.goto('/tarefas');
  await expect(page.locator('.stream-status').getByText('Atualizacao em tempo real conectada.', { exact: true })).toBeVisible();
});

test('13. solicita cancelamento controlado de tarefa', async ({ page }) => {
  await gerarPrevia(page, 'E2ECancelar');
  await page.getByRole('button', { name: 'Confirmar aplicação simulada' }).click();
  await expect(page.getByText('Compartilhamento criado.')).toBeVisible();
  await page.getByRole('link', { name: 'Central de tarefas' }).click();
  await page.getByRole('button', { name: 'Cancelar' }).first().click();
  await expect(page.getByText(/Cancelamento solicitado para/)).toBeVisible();
});

test('14. executa rollback simulado de tarefa', async ({ page }) => {
  await page.goto('/tarefas');
  await page.getByRole('button', { name: 'Rollback' }).first().click();
  await expect(page.getByText('Rollback simulado concluído e verificação de saúde aprovada.')).toBeVisible();
});

test('15. apresenta conflito de edição com correlação', async ({ page }) => {
  await page.setExtraHTTPHeaders({ 'X-Mock-Scenario': 'conflict' });
  await gerarPrevia(page, 'E2EConflito');
  await page.getByRole('button', { name: 'Confirmar aplicação simulada' }).click();
  await expect(page.getByText('Conflito operacional detectado.')).toBeVisible();
  await expect(page.getByText('corr-mock-conflict')).toBeVisible();
});

test('16. apresenta erro estruturado do agente', async ({ page }) => {
  await page.setExtraHTTPHeaders({ 'X-Mock-Scenario': 'agent-error' });
  await page.goto('/tarefas');
  await expect(page.getByText('Não foi possível consultar o estado da tarefa no agente.')).toBeVisible();
  await expect(page.getByText('corr-mock-agent-jobs')).toBeVisible();
});

test('17. detecta perda da conexão SSE', async ({ page }) => {
  await page.setExtraHTTPHeaders({ 'X-Mock-Scenario': 'sse-loss' });
  await page.goto('/tarefas');
  await expect(page.locator('.stream-status').getByText(/Reconectando ao fluxo de eventos/)).toBeVisible();
});

test('18. retoma SSE a partir do último evento conhecido', async ({ page }) => {
  await page.addInitScript(() => sessionStorage.setItem('samba-admin:last-job-event-id', 'event-before-disconnect'));
  await page.setExtraHTTPHeaders({ 'X-Mock-Scenario': 'sse-recover' });
  const recoveryRequest = page.waitForRequest((request) => {
    const url = new URL(request.url());
    return url.pathname === '/api/v1/events' && url.searchParams.get('lastEventId') === 'event-before-disconnect';
  });
  await page.goto('/tarefas');
  await recoveryRequest;
  await expect(page.getByText('Tarefa recuperada após reconexão')).toBeVisible();
  await expect(page.getByText('73%')).toBeVisible();
});

test('19. bloqueia path traversal ainda no formulário', async ({ page }) => {
  await preencherCompartilhamento(page, 'E2ETraversal', '/srv/dados/../../root');
  await page.getByRole('button', { name: 'Validar e gerar prévia' }).click();
  await expect(page.getByText('Travessia de diretórios não é permitida')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Prévia antes da aplicação' })).not.toBeVisible();
});

test('20. apresenta permissão insuficiente sem ocultar o recurso', async ({ page }) => {
  await page.setExtraHTTPHeaders({ 'X-Mock-Scenario': 'permission-denied' });
  await page.goto('/servicos');
  await expect(page.getByText('Permissao ou protecao CSRF recusada.')).toBeVisible();
  await expect(page.getByText('corr-mock-denied')).toBeVisible();
});
