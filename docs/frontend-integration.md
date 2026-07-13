# Integracao do Front-end

## Regra de consumo

O front-end consome `createOpenApiClient` em `frontend/src/api/client.ts`, apoiado pelos tipos gerados em `frontend/src/api/generated/openapi.ts`. `frontend/src/api/generated.ts` e uma facade manual de compatibilidade e nao deve receber tipos que contradigam o contrato. MSW continua permitido somente para desenvolvimento e testes deterministas; seus handlers devem acompanhar o OpenAPI.

O build de producao nunca ativa MSW. `make frontend-dist` gera `artifacts/frontend-dist.tar.gz` e `artifacts/frontend-dist.SHA256` de forma deterministica; o empacotamento falha se encontrar worker MSW, source map, trace, video, arquivo `.env` ou link simbolico. Em 2026-07-13, o artefato local continha somente HTML/CSS/JS e recebeu SHA-256 `2add81ef1c34266a17f2279eed56d99ddd3a8b2ccfaadfc36ac29f53ae546823`. Ele ainda nao foi publicado por CI.

## Sessao e erros

- Login cria sessao por cookie `HttpOnly`, `Secure` fora de desenvolvimento e `SameSite` adequado ao fluxo institucional.
- O token CSRF e obtido por endpoint/response autorizado e enviado apenas em mutacoes.
- `401` inicia fluxo de expiracao/renovacao ou redireciona para login; `403`, `409`, `422`, `423`, `503` e capability indisponivel recebem componentes distintos.
- Erros tecnicos exibem `correlationId`, mas nunca stack trace, segredo ou resposta interna bruta.

## Capabilities

Telas leem capabilities reais antes de habilitar uma acao. Acoes indisponiveis continuam visiveis, desabilitadas e acompanhadas de justificativa/pre-requisito. AD DC deve exibir que um servidor de arquivos nao precisa ser controlador de dominio para integrar-se ao AD.

## Eventos e contingencia

Para tarefas, usar SSE com `Last-Event-ID`, deduplicacao por identificador, backoff limitado e recuperacao de estado por `GET /jobs/{id}` apos reconexao. Quando SSE estiver indisponivel, usar polling com limite de frequencia e cancelar o polling quando a tarefa atingir estado terminal.

## Teste de integracao

A suite Playwright/MSW possui 20 cenarios para login, logout, expiracao, MFA, capability bloqueada, compartilhamento, validacao, diff, approval, tarefa, SSE, perda/reconexao, cancelamento, rollback, conflito, erro do agente, traversal e RBAC insuficiente. Uma segunda suite, tambem com 20 cenarios, executa sem MSW contra API Go, SQLite temporario e agente fixture por UDS autenticado; ela cobre ainda Samba, CUPS, readiness sem agente, RFC 7807/503, CSRF e bloqueio de mutacao.

Na rodada local de 2026-07-13, as duas suites foram listadas com seus 20 cenarios, mas o sandbox recusou navegador/sockets com `EPERM`. O ambiente Go/SQLite/agente fixture por UDS da segunda suite iniciou corretamente; nenhuma assertion de aplicacao foi atingida em qualquer das suites, portanto MSW e API real permanecem sem aprovacao de execucao. O sanitizador confirmou que os relatorios da tentativa nao continham senha, segredo TOTP, trace, screenshot, PNG, video ou WebM.

## Proxy HTTPS de referencia

`ops/reverse-proxy/nginx-samba-admin.conf.template` serve o `frontend-dist` e encaminha `/api/`, `/healthz` e `/readyz` para a API em `127.0.0.1:8080`. O endpoint SSE possui buffering e cache desabilitados, timeout dedicado e `X-Accel-Buffering: no`. A configuracao tambem limita body e timeouts, reforca o cookie de sessao e aplica HSTS, CSP e demais headers defensivos.

Os valores abaixo sao parametros obrigatorios de release e nao possuem default no repositorio:

- `@PUBLIC_HOST@`: nome institucional realmente aprovado;
- `@TLS_CERTIFICATE_PATH@`: cadeia PEM emitida/fornecida pela instituicao;
- `@TLS_CERTIFICATE_KEY_PATH@`: chave privada correspondente, fora do repositorio e legivel apenas pelo proxy;
- `@FRONTEND_ROOT@`: diretorio absoluto extraido do artefato validado.

O operador deve substituir todos os placeholders, confirmar que nenhum `@...@` permaneceu, executar `nginx -t` e obter aprovacao institucional antes de ativar o virtual host. HSTS so deve ser publicado depois de o certificado e a renovacao estarem operacionais. Dominio, contato, URL, emissor e caminho do certificado nao podem ser inferidos pelo build.

O script `scripts/render-nginx-config.sh` faz a substituicao de forma validada e recusa sobrescrever uma configuracao existente. Ele exige `SAMBA_ADMIN_PUBLIC_HOST`, `SAMBA_ADMIN_TLS_CERTIFICATE`, `SAMBA_ADMIN_TLS_CERTIFICATE_KEY` e `SAMBA_ADMIN_FRONTEND_ROOT`; nenhum valor institucional e armazenado no repositorio.

O render parametrizado passou localmente em 2026-07-13. Isso nao equivale a homologacao do proxy: `nginx -t`, certificado, dominio, renovacao e ativacao institucional nao foram executados.

A API deve manter `SAMBA_ADMIN_HTTP_ADDR=127.0.0.1:8080`, autenticacao local e cookies `Secure`, `HttpOnly` e `SameSite=Strict`. Somente o proxy HTTPS pode escutar na rede; acesso de laboratorio sem proxy deve usar tunel SSH.
