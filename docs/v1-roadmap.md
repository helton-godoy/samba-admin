# Avaliação executiva

O projeto está em um estágio **tecnicamente promissor, porém ainda não elegível para produção**. A arquitetura de segurança é sólida: API sem privilégios, agente separado, comunicação por Unix Domain Socket autenticado, adaptadores mock/fixture/FreeBSD e operações reais bloqueadas por padrão.

O código já contém:

* OpenAPI como contrato;
* autenticação local, MFA TOTP, RBAC e aprovação segregada;
* SQLite/WAL, jobs, locks, auditoria encadeada e SSE;
* adaptador FreeBSD somente leitura;
* pacote técnico FreeBSD `0.3.5`;
* testes unitários, integração e Playwright;
* workflow de CI com OpenAPI, Go, front-end, E2E, segurança, SBOM e build nativo FreeBSD.

Entretanto, a decisão documentada pelo próprio projeto é **no-go para produção**: os E2E em navegador não foram aprovados, os workflows do GitHub Actions ainda não possuem evidência consolidada de execução, scanners e SBOM não foram executados na rodada mais recente, TLS institucional não foi homologado e todas as mutações reais permanecem bloqueadas.

## Diagnóstico de maturidade

| Dimensão                | Nota estimada | Avaliação                                                               |
| ----------------------- | ------------: | ----------------------------------------------------------------------- |
| Arquitetura             |          8/10 | Boa separação de privilégios e contratos bem definidos.                 |
| Segurança por projeto   |          8/10 | Fail-closed, capabilities e agente endurecido.                          |
| Testes locais           |          7/10 | Boa cobertura de cenários, mas falta evidência recorrente na CI.        |
| CI                      |          6/10 | Workflow extenso já existe, porém precisa ser executado e estabilizado. |
| CD e releases           |          3/10 | Há pacote técnico, mas não há release distribuível e promovível.        |
| Governança Git          |          2/10 | Apenas um commit, sem Issues ou Pull Requests encontrados.              |
| Funcionalidade FreeBSD  |          4/10 | Inventário somente leitura; mutações reais bloqueadas.                  |
| Prontidão para produção |          3/10 | Sem TLS institucional, AD/Windows 11 e rollback mutável homologados.    |

O repositório apresenta apenas um commit inicial contendo praticamente toda a solução, o que reduz rastreabilidade, revisão incremental e capacidade de auditoria das decisões.

Também há inconsistências de identidade e versionamento:

* front-end declarado como `0.1.0`;
* pacote FreeBSD declarado como `0.3.5`;
* módulo Go declarado como `github.com/hu-ufcat/samba-admin-backend`, diferente do repositório atual.

Esses pontos devem ser corrigidos antes de iniciar novas funcionalidades de alto risco.

---

# Escopo recomendado da versão estável `v1.0.0`

A primeira versão estável deve ser um **console seguro para servidor de arquivos Samba**, não uma implementação integral de todos os módulos imaginados.

## Incluído em `v1.0.0`

* FreeBSD 15.1-RELEASE;
* UFS2 com ACL NFSv4;
* Samba standalone e membro de Active Directory;
* diagnóstico e ingresso controlado no AD;
* autenticação administrativa, MFA e RBAC;
* inventário do sistema, filesystems, Samba e serviços;
* criação, alteração, desativação e exclusão controlada de compartilhamentos;
* leitura e alteração segura de ACLs NFSv4;
* seleção de usuários e grupos do AD;
* geração estruturada de configuração;
* validação com `testparm`;
* diff, aprovação, backup, aplicação e rollback;
* reload e restart controlados;
* logs estruturados, auditoria e integração SIEM;
* pacote FreeBSD assinado e verificável;
* pipeline CI/CD completo;
* HTTPS institucional;
* testes com FreeBSD, Active Directory e Windows 11.

## Fora de `v1.0.0`

Devem permanecer experimentais ou desabilitados:

* Samba AD DC;
* controlador de domínio adicional;
* conversão POSIX ↔ NFSv4;
* DFS-R;
* instalação automática de drivers Point and Print;
* mutações de CUPS;
* quotas mutáveis;
* editor genérico de arquivos;
* operações recursivas de ACL em grandes volumes;
* alta disponibilidade e replicação física.

Esses recursos devem compor os ciclos `v1.1` e `v2.0`.

---

# Modelo de trabalho

## Cadência

* Sprint 0: uma semana;
* demais Sprints: duas semanas;
* Sprint de ACL: até três semanas;
* estimativa total para `v1.0.0`: **23 a 27 semanas**, considerando um desenvolvedor sênior principal;
* Story Points são relativos e devem ser recalibrados após as duas primeiras Sprints.

## Organização no GitHub

Criar Milestones:

```text
v0.4.0 — Engineering Baseline
v0.5.0 — Verified CI/CD
v0.6.0 — Production Front-end
v0.7.0 — FreeBSD Read-only
v0.8.0 — AD Member Diagnostics
v0.9.0 — Safe Samba Mutations
v0.9.5 — NFSv4 ACL and Domain Join
v1.0.0 — Stable
```

Labels recomendadas:

```text
type:feature
type:bug
type:security
type:test
type:docs
type:refactor
type:ci
type:release

area:frontend
area:backend
area:agent
area:freebsd
area:samba
area:ad
area:acl
area:security
area:observability

priority:P0
priority:P1
priority:P2
priority:P3

risk:low
risk:medium
risk:high
risk:critical

status:blocked
status:ready
status:in-progress
status:review
status:homologation
```

---

# Sprint 0 — Governança e baseline verificável

**Objetivo:** transformar o snapshot inicial em um projeto auditável e orientado a Pull Requests.

**Capacidade estimada:** 30–35 SP.

### Backlog

| ID      | Item                                                                   | SP | Prioridade |
| ------- | ---------------------------------------------------------------------- | -: | ---------- |
| GOV-001 | Criar Milestones, labels e GitHub Project                              |  3 | P0         |
| GOV-002 | Criar `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md` e licença |  5 | P0         |
| GOV-003 | Criar templates de Issue e Pull Request                                |  3 | P1         |
| GOV-004 | Criar `CODEOWNERS` e política de revisão                               |  3 | P0         |
| GOV-005 | Configurar Ruleset da branch `main`                                    |  5 | P0         |
| GOV-006 | Unificar versão do front-end, backend e pacote                         |  5 | P0         |
| GOV-007 | Decidir e corrigir o module path Go                                    |  3 | P0         |
| GOV-008 | Criar tag imutável `v0.3.5-lab`                                        |  2 | P1         |
| GOV-009 | Executar a primeira PR completa pela CI                                |  5 | P0         |

### Proteção da branch

Exigir:

* Pull Request;
* pelo menos uma aprovação;
* duas aprovações para código de segurança, agente, ACL ou AD;
* resolução de todas as conversas;
* branch atualizada;
* commits assinados, quando viável;
* checks obrigatórios;
* proibição de force push;
* proibição de exclusão;
* bloqueio de push direto, inclusive para administradores.

### Critérios de aceite

* nenhuma mudança funcional direta em `main`;
* primeira PR aprovada por todos os checks;
* versions consistentes;
* baseline marcado como laboratório;
* política de vulnerabilidade publicada;
* backlog integral registrado como Issues.

---

# Sprint 1 — Linters, formatadores e qualidade estática

**Objetivo:** tornar formatação e qualidade de código determinísticas.

**Capacidade:** 40 SP.

### Backlog

| ID      | Item                                                        | SP |
| ------- | ----------------------------------------------------------- | -: |
| QLT-001 | Adicionar `gofmt`, `goimports` e verificação de diff        |  3 |
| QLT-002 | Configurar `golangci-lint`                                  |  8 |
| QLT-003 | Adicionar ESLint com configuração flat e análise type-aware |  8 |
| QLT-004 | Adicionar regras React, Hooks e acessibilidade JSX          |  5 |
| QLT-005 | Configurar Prettier                                         |  3 |
| QLT-006 | Adicionar ShellCheck e `shfmt`                              |  3 |
| QLT-007 | Adicionar `actionlint` e `zizmor` para workflows            |  5 |
| QLT-008 | Adicionar Markdownlint e YAMLlint                           |  3 |
| QLT-009 | Criar hooks locais com `pre-commit` ou Lefthook             |  2 |

O front-end atual possui testes e build, mas não apresenta scripts de lint ou format no `package.json`.

No Go, `golangci-lint` permite executar múltiplos analisadores — incluindo `errcheck`, `govet`, `staticcheck`, `ineffassign` e `unused` — e também oferece verificação de formatação. ([GolangCI-Lint][1])

### Regras mínimas do Go

Ativar:

```text
errcheck
govet
staticcheck
unused
ineffassign
revive
gosec
bodyclose
contextcheck
errorlint
nilerr
noctx
prealloc
rowserrcheck
sqlclosecheck
unconvert
unparam
whitespace
```

### Critérios de aceite

* `make format-check` sem diferenças;
* `make lint` aprovado;
* nenhuma supressão sem justificativa;
* supressões contendo issue de acompanhamento;
* CI bloqueando código não formatado;
* zero warnings novos.

---

# Sprint 2 — CI e segurança de supply chain

**Objetivo:** executar e comprovar automaticamente todas as verificações existentes, complementando os gaps de segurança.

**Capacidade:** 45–50 SP.

O workflow atual já define validação OpenAPI, Go test/race/staticcheck/govulncheck, Vitest, duas suítes Playwright, npm audit, Gitleaks, SBOM e build FreeBSD.

### Backlog

| ID       | Item                                                   | SP |
| -------- | ------------------------------------------------------ | -: |
| CICD-001 | Executar e corrigir integralmente o workflow existente |  8 |
| CICD-002 | Separar checks rápidos de PR e checks completos        |  5 |
| CICD-003 | Adicionar CodeQL para Go e JavaScript/TypeScript       |  5 |
| CICD-004 | Adicionar Dependency Review Action                     |  3 |
| CICD-005 | Adicionar OSV-Scanner e Trivy filesystem               |  5 |
| CICD-006 | Adicionar Semgrep com regras Go/TypeScript             |  5 |
| CICD-007 | Fixar GitHub Actions por commit SHA                    |  5 |
| CICD-008 | Configurar OpenSSF Scorecard                           |  3 |
| CICD-009 | Adicionar análise de licenças                          |  3 |
| CICD-010 | Publicar resultados SARIF na aba Security              |  5 |

O CodeQL suporta tanto Go quanto JavaScript/TypeScript e publica os resultados como alertas de code scanning. ([GitHub Docs][2])

A Dependency Review Action pode bloquear uma PR quando uma atualização introduzir dependências vulneráveis ou incompatíveis com a política de licenças. ([GitHub Docs][3])

O OpenSSF Scorecard deve ser utilizado para medir continuamente práticas como atualização de dependências, proteção de branches, workflows perigosos e revisão de código. ([GitHub][4])

### Gates

* zero vulnerabilidades Critical ou High;
* vulnerabilidade Medium somente com exceção registrada e prazo;
* zero segredos confirmados;
* todas as Actions de terceiros fixadas por SHA;
* licenças incompatíveis bloqueadas;
* SARIF publicado;
* Scorecard inicial registrado como baseline.

---

# Sprint 3 — Release engineering e CD

**Objetivo:** produzir releases reproduzíveis, atestadas e promovíveis.

**Capacidade:** 40 SP.

### Backlog

| ID      | Item                                                     | SP |
| ------- | -------------------------------------------------------- | -: |
| REL-001 | Adotar Conventional Commits                              |  3 |
| REL-002 | Configurar Release Please ou equivalente                 |  5 |
| REL-003 | Gerar `CHANGELOG.md` automaticamente                     |  3 |
| REL-004 | Criar workflow de prerelease                             |  5 |
| REL-005 | Produzir pacote FreeBSD e frontend-dist na mesma release |  8 |
| REL-006 | Gerar SBOM separado para Go, npm e pacote final          |  5 |
| REL-007 | Gerar attestations de build                              |  5 |
| REL-008 | Criar checksums e assinatura verificável                 |  3 |
| REL-009 | Preservar artefato anterior para rollback                |  3 |

As attestations do GitHub registram onde e como os artefatos foram construídos e podem ser aplicadas a binários e SBOMs. ([GitHub Docs][5])

### Ambientes de CD

```text
PR efêmera
   ↓
development
   ↓
laboratório FreeBSD
   ↓ aprovação manual
homologação AD/Windows
   ↓ aprovação técnica e de segurança
produção
```

Produção nunca deve receber deploy automático diretamente após merge.

### Critérios de aceite

* release `v0.4.0-rc.1`;
* pacote e frontend com versão idêntica;
* SBOM publicado;
* attestation verificável;
* checksum publicado;
* rollback testado;
* nenhum placeholder `example.invalid`.

Atualmente o Makefile do pacote bloqueia corretamente releases com mantenedor e URL fictícios.

---

# Sprint 4 — Front-end real e E2E

**Objetivo:** aprovar o console contra API e agente reais, sem MSW.

**Capacidade:** 45 SP.

### Backlog

| ID     | Item                                                | SP |
| ------ | --------------------------------------------------- | -: |
| FE-001 | Estabilizar Playwright/MSW em CI                    |  5 |
| FE-002 | Estabilizar Playwright com API Go e agente fixture  |  8 |
| FE-003 | Criar tela de ativação e rotação de MFA             |  8 |
| FE-004 | Implementar gestão de recovery codes                |  5 |
| FE-005 | Melhorar expiração, renovação e revogação de sessão |  5 |
| FE-006 | Integrar axe-core para acessibilidade               |  5 |
| FE-007 | Implantar budgets de bundle e performance           |  3 |
| FE-008 | Testar proxy HTTPS em ambiente efêmero              |  6 |

### Gates

* 100% dos cenários críticos aprovados;
* taxa de flakiness inferior a 1%;
* nenhum trace, screenshot ou vídeo contendo segredos;
* conformidade WCAG 2.2 AA nos fluxos principais;
* tempo de carregamento e bundle dentro dos budgets;
* cliente OpenAPI usado em todas as telas.

Entrega: `v0.5.0`.

---

# Sprint 5 — Observabilidade, resiliência e hardening

**Objetivo:** tornar falhas detectáveis, diagnosticáveis e recuperáveis.

**Capacidade:** 45 SP.

### Backlog

| ID      | Item                                                    | SP |
| ------- | ------------------------------------------------------- | -: |
| OBS-001 | Endpoint protegido de métricas Prometheus               |  8 |
| OBS-002 | Métricas de API, jobs, locks, agente e autenticação     |  5 |
| OBS-003 | Exportação de eventos RFC 5424                          |  8 |
| OBS-004 | TLS para encaminhamento ao SIEM                         |  8 |
| OBS-005 | Backup online e restauração testada do SQLite           |  5 |
| OBS-006 | Rate limiting específico para login e TOTP              |  5 |
| OBS-007 | Testes de indisponibilidade do agente e banco bloqueado |  3 |
| OBS-008 | Runbooks de incidente e recuperação                     |  3 |

### Critérios de aceite

* dashboards e alertas básicos;
* correlation ID do navegador até o agente;
* restauração do banco testada;
* perda do agente não compromete API;
* eventos sensíveis redigidos;
* SIEM recebe eventos de teste;
* SLO inicial documentado.

---

# Sprint 6 — Inventário FreeBSD completo

**Objetivo:** concluir o produto somente leitura antes de introduzir escrita.

**Capacidade:** 50 SP.

### Backlog

| ID      | Item                                                    | SP |
| ------- | ------------------------------------------------------- | -: |
| BSD-001 | Completar inventário de sistema e rede                  |  5 |
| BSD-002 | Completar inventário de UFS2, mounts e `fstab`          |  8 |
| BSD-003 | Transformar `/shares` em inventário Samba real          |  8 |
| BSD-004 | Inventariar sessões e arquivos abertos                  |  5 |
| BSD-005 | Inventariar serviços rc.d                               |  5 |
| BSD-006 | Inventariar quotas em modo leitura                      |  5 |
| BSD-007 | Implementar parser inicial de ACL NFSv4 somente leitura |  8 |
| BSD-008 | Ampliar fixtures sanitizadas e testes de replay         |  6 |

O estado atual ainda limita ACL NFSv4 à identificação da opção de montagem e não possui parser de ACEs, herança ou permissões efetivas.

### Critérios de aceite

* todas as telas de inventário usam API → UDS → agente;
* API sem root não executa utilitários do sistema;
* falha de uma sonda gera capability indisponível;
* nenhuma mutação possível;
* fixtures reais sanitizadas;
* testes em FreeBSD 15.1;
* pacote `v0.6.0` somente leitura.

---

# Sprint 7 — Diagnóstico de membro do Active Directory

**Objetivo:** validar completamente o ambiente antes de permitir ingresso.

**Capacidade:** 50–55 SP.

### Backlog

| ID     | Item                                          | SP |
| ------ | --------------------------------------------- | -: |
| AD-001 | Diagnóstico DNS e registros SRV               |  5 |
| AD-002 | Diagnóstico de sincronização de horário       |  3 |
| AD-003 | Teste Kerberos sem armazenar credencial       |  8 |
| AD-004 | Diagnóstico LDAP/LDAPS/StartTLS               |  8 |
| AD-005 | Diagnóstico SMB e controladores preferenciais |  5 |
| AD-006 | Diagnóstico Winbind                           |  5 |
| AD-007 | Validação de `idmap_rid`                      |  5 |
| AD-008 | Validação de `idmap_ad` e RFC2307             |  8 |
| AD-009 | Assistente de pré-requisitos no front-end     |  8 |

### Critérios de aceite

* nenhum ingresso ainda habilitado;
* credenciais nunca persistidas;
* redaction automática;
* diagnóstico testado em AD de homologação;
* validação de ranges UID/GID;
* teste com DC local e fallback;
* documentação de `idmap_rid` e `idmap_ad`.

Entrega: `v0.7.0`.

---

# Sprint 8 — Primeira mutação: compartilhamentos Samba

**Objetivo:** implementar a primeira operação real com rollback completo.

**Capacidade:** 60 SP.

### Backlog

| ID      | Item                                        | SP |
| ------- | ------------------------------------------- | -: |
| SMB-001 | Gerador estruturado de configuração Samba   |  8 |
| SMB-002 | Validação real com `testparm`               |  5 |
| SMB-003 | Escrita atômica em arquivo temporário       |  8 |
| SMB-004 | Backup versionado da configuração           |  5 |
| SMB-005 | Aplicação e reload controlado               |  8 |
| SMB-006 | Health check após reload                    |  5 |
| SMB-007 | Rollback automático                         |  8 |
| SMB-008 | Impacto por sessões e arquivos abertos      |  5 |
| SMB-009 | Atestação de aprovação validada pelo agente |  8 |

### Operações permitidas

* criar compartilhamento;
* alterar propriedades seguras;
* habilitar;
* desabilitar;
* excluir, mediante aprovação;
* testar configuração;
* efetuar reload.

Não alterar ACL nesta Sprint.

### Critérios de aceite

* diff apresentado antes da aprovação;
* lock por configuração Samba;
* backup obrigatório;
* `testparm` antes e depois;
* rollback testado por falha induzida;
* acesso validado em Windows 11;
* nenhuma interrupção desnecessária;
* auditoria ponta a ponta.

Entrega: `v0.8.0`.

---

# Sprint 9 — ACL NFSv4 segura

**Objetivo:** permitir administração de ACLs NFSv4 em paths autorizados.

**Capacidade:** 65–75 SP, recomendando três semanas.

### Backlog

| ID      | Item                                     | SP |
| ------- | ---------------------------------------- | -: |
| ACL-001 | Parser completo de ACE NFSv4             |  8 |
| ACL-002 | Ordem canônica de ACEs                   |  8 |
| ACL-003 | Flags de herança                         |  8 |
| ACL-004 | Resolução de identidades AD/local        |  8 |
| ACL-005 | Cálculo de permissões efetivas           |  8 |
| ACL-006 | Exportação e restauração de ACL          |  8 |
| ACL-007 | Aplicação atômica em um objeto           |  8 |
| ACL-008 | Operação recursiva limitada e assíncrona |  8 |
| ACL-009 | Comparação com ACL exibida no Windows    |  8 |

### Testes obrigatórios

* property-based tests;
* fuzzing do parser;
* ACE Allow/Deny;
* `owner@`, `group@`, `everyone@`;
* herança em arquivo e diretório;
* no-propagate;
* ordem alterada;
* SID sem correspondência;
* rollback;
* symlink e TOCTOU;
* acesso via Windows 11.

A conversão do modelo de ACL do filesystem permanece fora desta Sprint.

---

# Sprint 10 — Ingresso controlado como membro do domínio

**Objetivo:** ingressar o servidor no AD sem promovê-lo a controlador de domínio.

**Capacidade:** 55–60 SP.

### Backlog

| ID       | Item                                      | SP |
| -------- | ----------------------------------------- | -: |
| JOIN-001 | Gerar configuração member server          |  8 |
| JOIN-002 | Selecionar e validar OU do computador     |  5 |
| JOIN-003 | Coletar credencial somente em memória     |  8 |
| JOIN-004 | Executar ingresso com timeout e redaction |  8 |
| JOIN-005 | Executar `testjoin` e testes Winbind      |  5 |
| JOIN-006 | Validar usuários e grupos                 |  5 |
| JOIN-007 | Implementar rollback de ingresso falho    |  8 |
| JOIN-008 | Implementar saída controlada do domínio   |  8 |
| JOIN-009 | Exigir dupla aprovação                    |  5 |

### Critérios de aceite

* Samba entra como **membro**, não como AD DC;
* credencial descartada após uso;
* nenhuma senha em banco ou logs;
* objeto criado na OU correta;
* idmap consistente;
* usuários/grupos resolvidos;
* Windows 11 acessa compartilhamento;
* falha induzida executa rollback;
* AD DC continua bloqueado.

Entrega: `v0.9.0`.

---

# Sprint 11 — Segurança dinâmica e homologação

**Objetivo:** executar testes ofensivos e homologação funcional completa.

**Capacidade:** 45 SP.

### Backlog

| ID      | Item                                       | SP |
| ------- | ------------------------------------------ | -: |
| SEC-001 | OWASP ZAP baseline em cada RC              |  5 |
| SEC-002 | OWASP ZAP full scan em homologação isolada |  8 |
| SEC-003 | Fuzzing contínuo do agente e parsers       |  8 |
| SEC-004 | Testes de carga e concorrência             |  5 |
| SEC-005 | Pentest manual orientado a risco           |  8 |
| SEC-006 | Teste de recuperação de desastre           |  5 |
| SEC-007 | Revisão de threat model                    |  3 |
| SEC-008 | Correção dos achados                       |  8 |

O ZAP Full Scan executa spidering e ataques ativos; portanto, deve rodar somente contra ambientes descartáveis ou de homologação explicitamente autorizados, nunca contra produção. ([ZAP][6])

### Cenários ofensivos mínimos

* command injection;
* argument injection;
* path traversal;
* symlink race;
* TOCTOU;
* bypass de RBAC;
* bypass de aprovação;
* replay no UDS;
* nonce reuse;
* HMAC inválido;
* CSRF;
* session fixation;
* brute force;
* log injection;
* SSE abuse;
* tampering da cadeia de auditoria;
* indisponibilidade do agente;
* SQLite lock contention.

---

# Sprint 12 — Release estável `v1.0.0`

**Objetivo:** concluir homologação, documentação e promoção controlada.

**Capacidade:** 35–40 SP.

### Backlog

| ID      | Item                                      | SP |
| ------- | ----------------------------------------- | -: |
| STB-001 | Congelamento funcional                    |  2 |
| STB-002 | Resolver todos os defeitos P0 e P1        |  8 |
| STB-003 | Homologação FreeBSD + AD + Windows 11     |  8 |
| STB-004 | Manual do administrador                   |  5 |
| STB-005 | Runbook de instalação, upgrade e rollback |  5 |
| STB-006 | Release notes e matriz de compatibilidade |  3 |
| STB-007 | Gerar artefatos finais assinados          |  5 |
| STB-008 | Testar instalação limpa e upgrade         |  5 |
| STB-009 | Aprovação formal de produção              |  3 |

### Critérios finais de liberação

A versão `v1.0.0` somente poderá ser publicada quando:

* CI verde em `main`;
* E2E MSW e API real aprovados;
* testes FreeBSD aprovados;
* testes Windows 11 aprovados;
* testes AD aprovados;
* zero vulnerabilidades Critical ou High;
* zero segredos;
* CodeQL e Semgrep sem achados bloqueantes;
* SBOM publicado;
* attestation publicada;
* pacote assinado;
* TLS institucional validado;
* rollback executado;
* restauração de banco executada;
* pentest concluído;
* nenhum defeito P0 ou P1;
* documentação aprovada;
* funcionalidades não homologadas desabilitadas.

---

# Pipeline CI/CD recomendado

## Pull Request — caminho rápido

Meta: até 15 minutos.

```text
Format check
→ lint
→ typecheck
→ unit tests
→ OpenAPI validation
→ generated-code check
→ dependency review
→ secret scan
→ build
→ E2E smoke
```

## Merge em `main`

```text
Testes completos
→ race detector
→ CodeQL
→ Semgrep
→ govulncheck
→ npm audit
→ OSV-Scanner
→ Trivy
→ Playwright MSW
→ Playwright API real
→ SBOM
```

## Nightly

```text
Fuzzing
→ mutation testing
→ ZAP baseline
→ testes de carga
→ FreeBSD matrix
→ OpenSSF Scorecard
→ dependências desatualizadas
```

## Tag de release

```text
Build nativo FreeBSD
→ package
→ frontend-dist
→ testes do package
→ SBOM
→ checksum
→ attestation
→ assinatura
→ deploy no laboratório
→ smoke test
→ aprovação manual
→ homologação
```

---

# Ferramentas de qualidade recomendadas

| Categoria         | Ferramentas                                                        |
| ----------------- | ------------------------------------------------------------------ |
| Go format/lint    | `gofmt`, `goimports`, `golangci-lint`, `staticcheck`, `go vet`     |
| Front-end         | ESLint, typescript-eslint, React Hooks, jsx-a11y, Prettier         |
| Shell e workflows | ShellCheck, shfmt, actionlint, zizmor                              |
| Documentação      | Markdownlint, YAMLlint, Vale                                       |
| Dependências      | Dependabot, Dependency Review, govulncheck, npm audit, OSV-Scanner |
| SAST              | CodeQL, Semgrep, gosec                                             |
| Segredos          | Gitleaks e push protection                                         |
| Artefatos         | CycloneDX/Syft, GitHub attestations, checksums                     |
| Repositório       | OpenSSF Scorecard                                                  |
| DAST              | OWASP ZAP                                                          |
| Testes            | Vitest, Playwright, Go race detector, fuzzing                      |
| Cobertura         | Codecov ou cobertura nativa publicada como check                   |
| Performance       | Lighthouse CI, k6 ou Vegeta                                        |

O Dependabot já está configurado semanalmente para npm, Go modules e GitHub Actions, o que deve ser preservado e complementado com agrupamento e políticas de aprovação.

---

# Definition of Done global

Uma Issue somente poderá ser concluída quando:

1. critérios de aceite atendidos;
2. código formatado;
3. lint aprovado;
4. testes unitários e integração aprovados;
5. E2E atualizado quando aplicável;
6. cobertura não reduzida;
7. novo código crítico com cobertura mínima de 90%;
8. análise de segurança aprovada;
9. OpenAPI atualizado;
10. código gerado atualizado;
11. documentação atualizada;
12. threat model atualizado quando necessário;
13. capability e feature flag definidas;
14. logs sanitizados;
15. rollback testado;
16. evidências anexadas à PR;
17. aprovação de dois revisores para mudanças críticas.

---

# Indicadores de excelência

| Indicador                               | Meta para `v1.0.0` |
| --------------------------------------- | -----------------: |
| CI em PR                                |       ≤ 15 minutos |
| Taxa de sucesso E2E                     |               100% |
| Flaky tests                             |               < 1% |
| Cobertura de novo código crítico        |              ≥ 90% |
| Cobertura global                        |              ≥ 80% |
| Vulnerabilidades Critical/High          |                  0 |
| Segredos encontrados                    |                  0 |
| Defeitos P0/P1 em release               |                  0 |
| Rollback testado                        |  100% das mutações |
| OpenSSF Scorecard                       |             ≥ 8/10 |
| Mean Time to Restore em laboratório     |       ≤ 30 minutos |
| Alterações críticas com dupla aprovação |               100% |
| Artefatos com SBOM e provenance         |               100% |

## Caminho crítico

```text
Governança
→ CI comprovada
→ E2E real
→ FreeBSD somente leitura completo
→ diagnóstico AD
→ mutação de compartilhamento
→ ACL NFSv4
→ ingresso como membro
→ pentest
→ v1.0.0
```

O próximo passo operacional é criar os Milestones e converter os itens `GOV-001` a `STB-009` em Issues, começando pela Sprint 0 e por uma Pull Request exclusivamente de governança e CI.

[1]: https://golangci-lint.run/docs/welcome/quick-start/ "Quick Start – Golangci-lint"
[2]: https://docs.github.com/en/code-security/code-scanning/introduction-to-code-scanning/about-code-scanning-with-codeql "Code scanning with CodeQL - GitHub Docs"
[3]: https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/about-dependency-review "Dependency review - GitHub Docs"
[4]: https://github.com/ossf/scorecard "GitHub - ossf/scorecard: OpenSSF Scorecard - Security health metrics for Open Source · GitHub"
[5]: https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations/using-artifact-attestations-to-establish-provenance-for-builds "Using artifact attestations to establish provenance for builds - GitHub Docs"
[6]: https://www.zaproxy.org/docs/docker/full-scan/ "ZAP – ZAP - Full Scan"

