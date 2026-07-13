# Plano de Testes de Seguranca

## Objetivo

Validar controles em camadas antes de habilitar qualquer escrita real. Cada caso registra ambiente, versao, entrada, resultado esperado, correlation ID e evidencia sanitizada.

## Matriz minima

| Area | Casos obrigatorios |
| --- | --- |
| Entrada e execucao | Command/argument/environment injection, JSON malicioso, body excessivo, path traversal, symlink race e TOCTOU. |
| Autenticacao | Brute force, lockout, timing relevante, session fixation, expiracao, logout, CSRF, cookie seguro e MFA/recovery. |
| Autorizacao | Bypass RBAC, objeto fora de escopo, escalation de papel, approval bypass e autoaprovacao. |
| Agente | Socket spoofing, peer UID/GID invalido, HMAC invalido, nonce reutilizado, replay fora/na janela e mensagem excessiva. |
| Dados | XSS, log injection, segredo em log/evento/auditoria, adulteracao de audit chain e SQLite lock contention. |
| Eventos | Abuso SSE, `Last-Event-ID` malicioso, reconexao, deduplicacao, polling de contingencia e vazamento entre tenants. |
| Concorrencia | Race detector onde suportado, locks, cancelamento, rollback concorrente e idempotencia. |

## Niveis de evidencia

Testes unitarios e de integracao local cobrem validadores, API, SQLite, auditoria, approval, autenticacao, UDS e parsers. Ha 20 cenarios Playwright/MSW e uma segunda suite de 20 cenarios contra API Go, SQLite temporario e agente fixture por UDS, sem MSW. Em 2026-07-13, a execucao browser das duas suites foi bloqueada pelo sandbox com `EPERM` antes de qualquer assertion; o backend Go/agente fixture UDS da suite real iniciou e a listagem/sanitizacao de artefatos passou, mas isso nao aprova o E2E. Falha ou ausencia de execucao de teste critico bloqueia release e qualquer capability mutavel.

Na mesma rodada, os cinco comandos obrigatorios passaram localmente na ordem definida. Tambem passaram `go vet`, `go test -race`, `staticcheck`, 25 testes Vitest, `sh -n`, `git diff --check` e o sanitizador de artefatos. O sanitizador nao encontrou senha, TOTP, trace, PNG ou WebM nos relatorios da tentativa E2E.

No FreeBSD 15.1-p1 passaram testes, race detector, vet, build, package, smoke, deinstall ativo, reinstalacao e failover de readiness; staticcheck nao estava disponivel na VM. `npm audit` foi inconclusivo por `EAI_AGAIN`, e gitleaks, govulncheck, SBOM e GitHub Actions nao foram executados nesta rodada.

## Cobertura automatizada atual

| Controle | Evidencia atual | Pendencia |
| --- | --- | --- |
| Path/argument/config injection | Testes Go de validacao, executor e handlers | Fuzzing prolongado e symlink race em UFS2 real. |
| HMAC/timestamp/nonce/peer | Testes do protocolo e agente | Socket spoofing nativo FreeBSD com UIDs distintos. |
| RBAC/CSRF/idempotencia/approval | Testes HTTP e storage | Teste distribuido com duas instancias da API. |
| Sessao/MFA/recovery | Testes Go e fluxo Playwright | Rate limit dedicado ao codigo MFA e UI de gestao do fator. |
| Audit chain/log redaction | Concorrencia, adulteracao e redacao recursiva | Exportacao/retencao SIEM e verificacao independente. |
| SSE/reconexao/polling | Hook unitario e cenarios Playwright/MSW e API Go implementados | Executar os navegadores fora do sandbox restrito; depois cobrir carga e abuso. |
| SQLite contention/race | Testes concorrentes e `go test -race` passaram localmente e no FreeBSD 15.1-p1 | Teste distribuido prolongado continua pendente. |
| Supply chain | Artefato `frontend-dist` sanitizado e package/checksum nativo verificados | Reexecutar `npm audit` com rede; executar gitleaks, govulncheck, SBOM e GitHub Actions. |

## Regras de execucao

Usar somente dados sinteticos ou sanitizados. Nao colocar credenciais em variaveis de CI visiveis, screenshots, traces ou artefatos. Fuzzing e testes destrutivos ocorrem apenas em VM descartavel com snapshot.
