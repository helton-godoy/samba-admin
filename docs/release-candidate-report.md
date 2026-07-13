# Relatorio do Candidato a Homologacao Tecnica

Data de referencia: 2026-07-13.

Status: **package tecnico `0.3.5` aprovado apenas para laboratorio somente leitura; no-go para RC1 distribuivel e producao**. O baseline RC0 `0.3.4` permanece preservado e foi exercitado como rollback. E2E browser, CI/scanners/SBOM e TLS institucional continuam gates abertos.

## Avaliacao da base preservada

O monorepo, os dois processos Go, SQLite/WAL, UDS assinado, adapters, feature flags, rotas existentes, aliases legados e telas atuais foram mantidos. O trabalho foi incremental: o OpenAPI passou a verificar a interface do servidor; o front-end passou a consumir o transporte tipado; autenticacao, approval, SSE, agente e adapters receberam controles adicionais sem habilitar escrita FreeBSD.

| Componente anterior | Decisao de compatibilidade |
| --- | --- |
| `samba-admin-api` e `samba-admin-agent` | Preservados como processos separados; coletor e apenas uma ferramenta adicional. |
| Unix Domain Socket com HMAC | Preservado e endurecido com key ID/rotacao, peer, nonce persistente e limites. |
| Adapter `mock` | Preservado para desenvolvimento e E2E deterministico; mutacoes sao simuladas. |
| Adapter `fixture` | Preservado e tornado replay estrito de fixture sanitizada. |
| Adapter FreeBSD bloqueado | Preservado como somente leitura; nenhuma interface mutavel foi adicionada. |
| API e OpenAPI existentes | Rotas preservadas; foram adicionados de forma compativel `GET /api/v1/samba` e `GET /api/v1/cups`, com handlers na interface gerada. |
| `/api/v1/acl` e `/api/v1/principals` | Preservados com `Deprecation`, `Sunset` e `Link`; canonicos permanecem `/acls` e `/identities`. |
| Front-end React/Vite e MSW | Preservados; o mesmo cliente atende API Go e MSW. |
| Jobs, SSE, idempotencia, locks e rollback simulado | Preservados e integrados ao console; nenhum rollback real foi liberado. |

## Incrementos concluidos em codigo

- Geracao e verificacao OpenAPI para Go e TypeScript, interface compilavel e detector de breaking change fixado.
- Login/logout, bootstrap CSRF, renovacao/expiracao, break-glass e MFA TOTP/recovery no backend; login MFA no front-end.
- Invalidação de sessoes em confirmacao, rotacao e revogacao de MFA.
- Workflow persistente de solicitacao, aprovacao segregada, rejeicao, janela e execucao como job.
- Idempotencia reservada por ator/chave antes do efeito em mutacoes expostas.
- Auditoria concorrente com redacao recursiva e verificacao da cadeia.
- Catalogo versionado do agente, executor somente leitura, HMAC com rotacao, nonce persistente, peer credentials, limites e kill de grupo de processos.
- Providers FreeBSD somente leitura para sistema, filesystem, Samba, dominio e CUPS.
- Contratos estruturados Samba/CUPS roteados pela API exclusivamente ao provider UDS no modo real, com RFC 7807/HTTP 503 para agente, capability ou sonda minima indisponivel.
- Coletor Go sanitizado e adapter de replay que nunca cai para execucao real.
- Capabilities no front-end, SSE com replay/reconexao/deduplicacao e polling de contingencia.
- Vinte cenarios Playwright/MSW e testes adicionais de injecao, approval, idempotencia, replay, auditoria e sessoes MFA.
- Package backend inicial com conta restrita, rc.d separado, agente desabilitado, newsyslog, plist, checksum e job nativo FreeBSD.
- Inventario da API roteado exclusivamente pelo agente UDS; o processo sem root nao executa comandos FreeBSD.
- Package `0.3.4` construido e instalado no FreeBSD 15.1, com fixture real sanitizada, ciclo upgrade/downgrade e remocao com servicos ativos exercitados.
- Workflow de CI configurado para contrato, Go, front-end, Playwright, dependencias, segredos, SBOM e FreeBSD 15.1; GitHub Actions nao foi executado nesta rodada.

## Evidencias executadas no RC1

- O front-end passou a consumir `SambaInfo` e `CupsInfo` pelo cliente tipado, mantendo capabilities indisponiveis visiveis e operacoes mutaveis bloqueadas.
- `make openapi-validate`, `make generate`, `make check-generated`, `make test` e `make build` passaram localmente, nessa ordem. Tambem passaram `go vet`, `go test -race`, `staticcheck`, 25 testes Vitest, `sh -n`, `git diff --check` e o sanitizador de artefatos.
- O `frontend-dist` local contem apenas HTML/CSS/JS. O artefato `artifacts/frontend-dist.tar.gz` tem SHA-256 `2add81ef1c34266a17f2279eed56d99ddd3a8b2ccfaadfc36ac29f53ae546823` e nao foi publicado.
- O template nginx foi renderizado com parametros de teste. `nginx -t`, certificado, dominio e ativacao institucional nao foram executados.
- As suites Playwright MSW e API Go real possuem 20 cenarios cada. O sandbox bloqueou Chromium/sockets com `EPERM` antes de assertions da aplicacao; o backend Go/agente fixture por UDS da suite real iniciou, e os artefatos da tentativa passaram pelo sanitizador sem senha, TOTP, trace, PNG ou WebM. Nenhuma suite browser foi aprovada.
- `npm audit` foi inconclusivo por `EAI_AGAIN`. Gitleaks, govulncheck, SBOM e GitHub Actions nao foram executados nesta rodada.
- No FreeBSD passaram testes, race detector, vet e build nativos. Staticcheck nao estava disponivel na VM. O package tecnico `0.3.5`, construido com `RELEASE=no`, passou pelo ciclo nativo e pelo smoke somente leitura.

## Funcionalidades ainda bloqueadas

- Qualquer escrita real de Samba, ACL, filesystem, CUPS, quota ou syslog.
- Ingresso/saida do Active Directory e coleta de sua credencial.
- Samba AD DC, promocao, SYSVOL, replicacao ou restore.
- Parser real e escrita de ACL NFSv4, heranca, ACE ordering e permissoes efetivas.
- LDAP/StartTLS/LDAPS e OIDC para login institucional.
- Tela autenticada de ativacao/rotacao/revogacao MFA e rate limit dedicado ao codigo.
- Endpoint protegido de metricas e transporte SIEM RFC 5424/TLS.
- Atestacao de approval validada pelo agente antes de uma futura mutacao.
- Execucao aprovada do Playwright MSW e da suite contra API Go/agente fixture, alem de AD de homologacao, Windows 11 e rollback mutavel real.
- Publicacao do `frontend-dist`, homologacao do proxy TLS e acesso browser contra a API Go/FreeBSD.
- Execucao real de GitHub Actions, gitleaks, govulncheck e geracao/publicacao do SBOM.
- Metadados, assinatura, dominio e certificado institucionais necessarios a um package/release distribuivel.

## Evidencia FreeBSD e limite

A VM `192.168.122.84` executa FreeBSD `15.1-RELEASE-p1`, ABI `FreeBSD:15:amd64` e Go 1.26.5. O package tecnico `0.3.5` esta instalado; API e agente estao ativos, a API escuta somente em loopback, o adapter e `agent`, o executor e `readonly-freebsd` e `SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS=false`.

O smoke aprovado em `/root/samba-admin-evidence/package-smoke-20260713T101715Z` cobriu endpoints Samba/CUPS, HTTP 503 com correlation ID sem agente, readiness 503 e recuperacao, mutacao bloqueada, deinstall ativo sem processos orfaos e reinstalacao. O relatorio final esta em `/root/samba-admin-evidence/rc1-final-20260713T101536Z`.

Samba e CUPS terminaram parados. `/usr/local/etc/cups/cupsd.conf` manteve SHA-256 `30a33232846d21246f4a59d9f1bb6d8ec54b1ddd429575ff5fb5918f3b82cb5e`; as demais configuracoes observadas permaneceram ausentes. Banco, configuracao e credencial root-only do laboratorio foram preservados. Nao houve credencial de dominio, ingresso, ACL, alteracao Samba/CUPS, filesystem ou AD DC.

## Evidencias preservadas do RC0

Os resultados abaixo registram o baseline anterior e nao devem ser apresentados como validacao do RC1 atual.

- `make openapi-validate`, `make generate`, `make check-generated`, `make test` e `make build`: aprovados na ordem obrigatoria.
- Go: todos os pacotes aprovados, `go vet`, `go test -race` e `staticcheck 2025.1.1` sem achados.
- Dependencias: `govulncheck v1.6.0` e `npm audit --omit=dev` sem vulnerabilidades encontradas.
- Segredos: `gitleaks v8.30.1 --no-git --redact` sem vazamentos.
- Front-end: 13 testes Vitest, build Vite e 20/20 cenarios Playwright/MSW aprovados.
- Contrato: `oasdiff v1.11.7` validado; comparacao de breaking change real depende da base do pull request.
- Scripts/package: `sh -n`, dry-run de staging e verificacao de plist/checksum na CI preparados.
- FreeBSD nativo: todos os testes Go, build CGO, package/checksum, install, login, UDS/peer, inventarios, bloqueio de mutacao, stop, deinstall, upgrade, downgrade e reinstalacao final aprovados.
- O hook de deinstall interrompeu API/agente antes da remocao; nenhum processo orfao permaneceu. O failover de readiness confirmou `healthz=200`, `readyz=503` sem agente e recuperacao para `readyz=200`.
- Banco operacional de laboratorio criado separadamente, com credencial root-only e senha bootstrap removida da configuracao apos validacao do login.
- Fixture real sanitizada validada e incorporada com teste de replay; SHA-256 `f9dee06e83bb32ad9f8a0da969353068a5767b57181d3e3d1afc39f06fa3769a`.

## Package e rollback do RC1

- Package tecnico `0.3.5`: `/root/samba-admin-packages/0.3.5-20260713/samba-admin-0.3.5.pkg`, imutavel e root-only, SHA-256 `72b05ee3aef02a3cb59a1bffd4ffcba637ddc295b8d0f71fbdff96c4d43edd89`.
- Rollback reconstruido/testado `0.3.4`: `/root/samba-admin-rollback/0.3.4-20260713T100657Z/samba-admin-0.3.4.pkg`, imutavel, SHA-256 `333ccfa10efa8bc8a533c0c9712d3cdff32dcd04116cbd160bd9f174245096c1`.
- Checksum historico do artefato original `0.3.4`: `53836c21ea9a3278583a40dac5cc50f8ba93162221cbe2296a6c61f2c4a8b39b`. Ele e preservado como evidencia historica e nao deve ser confundido com o artefato reconstruido.

Durante a rodada houve um rollback intermediario provocado por um bug no procedimento de probe, nao por falha do produto. O rollback `0.3.4` funcionou; depois da correcao do probe, o `0.3.5` foi reinstalado e as sondagens passaram.

Ainda nao ha evidencia consolidada de: workflow GitHub Actions executado, E2E browser aprovado, gitleaks/govulncheck/SBOM desta rodada, `frontend-dist` publicado, proxy TLS homologado, Windows 11, segundo disco UFS2 ou Active Directory de homologacao.

## Decisao recomendada

O package tecnico `0.3.5` esta **aprovado apenas para laboratorio somente leitura**, com rollback conhecido. A decisao formal e **no-go para RC1 distribuivel e producao** ate aprovar os E2E browser, executar CI/scanners/SBOM e homologar TLS/metadados institucionais. Nenhuma capability mutavel pode ser habilitada.
