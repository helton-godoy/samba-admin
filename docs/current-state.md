# Estado Atual e Limites Operacionais

## Objetivo deste documento

Este documento e a referencia operacional para distinguir uma funcao implementada, uma simulacao e uma funcao ainda nao homologada. Uma tela, endpoint ou capability nao e evidencia suficiente de suporte em producao.

## Regra de estado

| Estado | Significado operacional |
| --- | --- |
| `suportado` | Implementado, testado e homologado para a combinacao de versao, pacote e ambiente declarada. |
| `suportado_com_restricoes` | Implementado com limites documentados e controles compensatorios. |
| `experimental` | Implementado apenas para laboratorio; nao liberar para producao. |
| `indisponivel` | Nao existe implementacao segura para o ambiente atual. |
| `nao_verificado` | Nao houve evidencia suficiente no host alvo; tratar como deny by default. |

## Inventario em 2026-07-13

| Area | Estado atual | Evidencia e limite |
| --- | --- | --- |
| API, SQLite, auditoria, jobs, locks e RBAC | Implementado e testado localmente | Nao autoriza alteracao no host. Auditoria possui verificacao da cadeia e redacao recursiva; rollback real continua bloqueado. |
| OpenAPI | Fonte efetiva do contrato | Os contratos aditivos `GET /api/v1/samba` e `GET /api/v1/cups` e seus modelos estruturados foram incorporados. Validacao, geracao, verificacao de gerados, testes e build passaram na ordem obrigatoria; o suporte OpenAPI 3.1 do `oapi-codegen` ainda emite aviso e permanece risco de ferramenta. |
| Aliases legados | Preservados e deprecated | `/api/v1/acl` e `/api/v1/principals` retornam cabecalhos de deprecacao; os caminhos canonicos sao `/api/v1/acls` e `/api/v1/identities`. |
| Front-end | Integrado ao cliente tipado | Login, logout, expiracao, MFA, erros, capabilities, Samba/CUPS e SSE/polling usam o cliente tipado. Vinte e cinco testes Vitest passaram. As suites Playwright MSW e API Go real foram listadas com 20 cenarios cada, mas o sandbox bloqueou navegador/sockets com `EPERM` antes de assertions; nenhuma delas foi aprovada nesta rodada. |
| Autenticacao local e TOTP | Implementado e testado localmente | Break-glass, Argon2id, bloqueio, CIDR, sessao/CSRF, TOTP, recovery e revogacao de sessoes existem. Tela de autoatendimento para ativar/rotacionar/revogar MFA ainda nao existe. |
| LDAP/OIDC institucional | Preparado apenas em arquitetura | Nao ha autenticador LDAP, StartTLS, LDAPS ou OIDC executavel nesta versao. |
| Aprovacao | Implementada e testada localmente | Risco, justificativa, impacto, rollback, janela e segregacao sao validados na API. Operacoes reais de alto risco continuam indisponiveis. |
| Agente privilegiado | Endurecido e homologado com restricoes no FreeBSD 15.1 | UDS `0660`, diretorio root controlado, peer UID nativo, HMAC, timestamp, nonce persistente, limites e catalogo versionado foram exercitados via package. Nenhuma mutacao FreeBSD esta implementada. |
| Coletor/replay de fixtures | Implementado e testado no FreeBSD | A fixture real sanitizada `backend/testdata/fixtures/freebsd-15.1-samba423-sanitized.json` foi incorporada e possui teste de replay fail-closed. |
| Adapter FreeBSD | Somente leitura, homologado com restricoes no laboratorio | No package tecnico `0.3.5`, sistema, filesystem, Samba e CUPS percorrem API → UDS → agente. O smoke confirmou Samba/CUPS, RFC 7807/503 com correlation ID sem agente, readiness 503/recuperacao e mutacao bloqueada. |
| ACL NFSv4 | Inventario de mount option apenas | Parser real de ACE, ordem, heranca e permissoes efetivas ainda nao existe; `acl.nfsv4.read` permanece `nao_verificado` no adapter real. |
| Samba member server | Inventario/diagnostico restrito | `GET /api/v1/samba` consulta o provider somente leitura pelo agente; perfil, DNS configurado e ping winbind permanecem diagnosticos separados. SRV, Kerberos, LDAP, SMB, idmap completo e ingresso real continuam bloqueados. `/api/v1/shares` nao foi convertido em inventario real. |
| Samba AD DC | `nao_verificado` | Modulo separado, sem promocao e fora do fluxo de member server. |
| CUPS e quotas | Inventario parcial | `GET /api/v1/cups` e `/api/v1/printers` usam o inventario CUPS somente leitura via agente e retornam 503 quando a sonda minima/capability falha. Escrita CUPS, drivers Windows e quotas mutaveis permanecem bloqueados. |
| Metricas e SIEM | Nao implementados | Ha logs estruturados/correlation ID, mas endpoint protegido de metricas e transporte RFC 5424/TLS ainda nao existem. |
| Empacotamento FreeBSD | `0.3.5` tecnico aprovado para laboratorio | FreeBSD 15.1-p1, ABI `FreeBSD:15:amd64`, Go 1.26.5, testes/race/vet/build nativos, package, checksum, smoke, deinstall ativo sem orfaos e reinstalacao passaram. O build usou `RELEASE=no`, portanto nao e artefato distribuivel. |

## Incremento RC1 validado tecnicamente com restricoes

O RC1 prioriza os inventarios Samba/CUPS via API → UDS → agente, a integracao real do front-end, a suite Playwright sem MSW, o `frontend-dist` e uma configuracao de referencia para reverse proxy HTTPS. O codigo, os testes locais e o ciclo de package nativo foram validados. O `frontend-dist` local contem somente HTML/CSS/JS e tem SHA-256 `2add81ef1c34266a17f2279eed56d99ddd3a8b2ccfaadfc36ac29f53ae546823`.

Nenhum dominio, certificado, URL ou contato institucional foi inferido. O package instalado na VM `192.168.122.84` e `samba-admin-0.3.5`, com API em loopback, adapter `agent`, executor `readonly-freebsd` e `SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS=false`. A decisao e **no-go para RC1 distribuivel/producao** porque os E2E browser nao foram aprovados, GitHub Actions/scanners/SBOM nao foram executados e TLS institucional nao foi homologado.

## Invariantes de seguranca

1. O navegador nunca acessa o agente diretamente.
2. Nenhuma entrada do usuario e concatenada em um comando de shell.
3. Toda escrita real requer capability confirmada, lock, backup, diff, auditoria e rollback verificavel.
4. Credenciais de ingresso no dominio existem somente em memoria durante a operacao.
5. A ausencia de evidencia e tratada como `indisponivel` ou `nao_verificado`, nunca como permissao implicita.

## Evidencias necessarias para mudar um estado

Uma alteracao para `suportado` exige, no minimo: versao do FreeBSD e do pacote Samba/CUPS, fixture sanitizada, testes automatizados, registro do experimento, plano de rollback executado e aprovacao tecnica. Para recursos que afetam Active Directory, ACLs ou disponibilidade SMB, a evidencia deve incluir teste em Windows 11 e ambiente AD de homologacao.

## Evidencia FreeBSD RC0 preservada

Antes do RC1, a VM limpa `192.168.122.84` executava FreeBSD `15.1-RELEASE-p1` amd64/UFS com o package `samba-admin-0.3.4`. O checksum historico do artefato original permanece `53836c21ea9a3278583a40dac5cc50f8ba93162221cbe2296a6c61f2c4a8b39b`.

O rollback `0.3.4` reconstruido e testado foi preservado imutavel em `/root/samba-admin-rollback/0.3.4-20260713T100657Z/samba-admin-0.3.4.pkg`, SHA-256 `333ccfa10efa8bc8a533c0c9712d3cdff32dcd04116cbd160bd9f174245096c1`. Esse checksum e do artefato reconstruido e nao substitui o checksum historico original.

## Evidencia FreeBSD RC1

O package tecnico `0.3.5`, SHA-256 `72b05ee3aef02a3cb59a1bffd4ffcba637ddc295b8d0f71fbdff96c4d43edd89`, foi preservado em `/root/samba-admin-packages/0.3.5-20260713/samba-admin-0.3.5.pkg`. Testes, race detector, vet e build nativos passaram; staticcheck nao estava disponivel na VM. O smoke aprovado esta em `/root/samba-admin-evidence/package-smoke-20260713T101715Z` e o relatorio final em `/root/samba-admin-evidence/rc1-final-20260713T101536Z`.

A VM terminou com API/agente ativos, API em loopback, adapter `agent`, executor `readonly-freebsd` e mutacoes desabilitadas. Samba e CUPS permaneceram parados; `/usr/local/etc/cups/cupsd.conf` manteve SHA-256 `30a33232846d21246f4a59d9f1bb6d8ec54b1ddd429575ff5fb5918f3b82cb5e` e as demais configuracoes observadas permaneceram ausentes. Banco, configuracao e credencial root-only do laboratorio foram preservados. Nao houve AD, Windows 11, segundo disco UFS2, ACL ou operacao Samba/CUPS mutavel.

## Atualizacao

Atualize este arquivo no mesmo pull request que mudar uma capability. Registre a data, o ambiente, os testes e a decisao; nao substitua uma limitacao por linguagem promocional.
