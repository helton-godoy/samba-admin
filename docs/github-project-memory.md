# Memória operacional no GitHub

## Finalidade

O GitHub é a fonte durável de contexto, planejamento, decisão e evidência do projeto. Uma nova sessão de agente deve conseguir recuperar o estado sem depender de histórico de chat: o código e os documentos explicam as invariantes; as Issues explicam o trabalho; os Milestones e o Project explicam sequência e estado; Pull Requests e checks preservam as evidências.

O backlog canônico está em [`.github/project/backlog.tsv`](../.github/project/backlog.tsv) e o roadmap detalhado em [`docs/v1-roadmap.md`](v1-roadmap.md). O TSV é deliberadamente simples para ser consultado por shell, `git`, `awk` e agentes sem bibliotecas adicionais.

## Protocolo obrigatório para agentes

Antes de planejar ou alterar código:

1. confirmar o repositório e a branch com `git remote -v` e `git status --short --branch`;
2. ler `AGENTS.md`, `README.md`, `docs/current-state.md` e `docs/v1-roadmap.md`;
3. recuperar a Issue designada com `gh issue view <numero> --comments`;
4. consultar dependências pelo código estável, por exemplo `gh issue list --state all --search 'SMB-009 in:title'`;
5. verificar Pull Requests e checks relacionados antes de presumir que uma dependência está concluída;
6. confrontar a descrição da Issue com o código atual e registrar qualquer divergência;
7. executar somente o escopo da Issue, mantendo recursos não homologados bloqueados;
8. anexar à Pull Request testes, evidências, riscos residuais e procedimento de rollback;
9. atualizar `docs/current-state.md` quando uma capability mudar de estado.

Comandos mínimos de recuperação:

```bash
gh repo view --json nameWithOwner,defaultBranchRef,url
gh issue list --state open --limit 200 --json number,title,labels,milestone,assignees
gh issue view <numero> --comments
gh pr list --state all --search '<CODIGO>'
gh project list --owner helton-godoy
gh project item-list <numero-do-project> --owner helton-godoy --limit 500
git log --oneline --decorate --all --max-count=50
git grep -n '<CODIGO>'
```

## Chaves estáveis

Cada trabalho possui um código imutável, como `GOV-001`, `CICD-003` ou `ACL-007`. O código deve aparecer:

- no início do título da Issue: `[CODIGO] descrição`;
- no título ou corpo da Pull Request;
- nos commits relevantes, quando a política de commits permitir;
- em ADR, runbook ou evidência associada;
- no TSV, que é a fonte de reconciliação da automação.

Nunca reutilize um código para outro objetivo. Se o escopo mudar materialmente, abra uma nova Issue e relacione a anterior.

## Modelo das Issues executáveis

O script `scripts/bootstrap-github-project.sh` cria uma Issue por linha do TSV. Cada corpo inclui:

- resultado esperado e contexto arquitetural;
- objetivo e gates da Sprint;
- escopo específico do item;
- dependências por código estável;
- critérios de aceite compartilhados e específicos;
- evidências obrigatórias;
- restrições de segurança;
- comandos de recuperação;
- metadados para Milestone, Project e labels.

A Issue é uma unidade executável, não uma frase solta. O agente ainda deve inspecionar o repositório porque uma Issue descreve intenção e critérios, não substitui o estado atual do código.

## Normalização dos Milestones

O roadmap de origem contém nomes oficiais de Milestones e algumas linhas intermediárias de “Entrega” com numeração anterior. Para eliminar ambiguidade, a conversão usa a seguinte matriz canônica:

| Milestone | Sprints | Condição de saída resumida |
|---|---|---|
| `v0.4.0 — Engineering Baseline` | 0–1 | Governança, identidade, formatação e qualidade estática verificáveis. |
| `v0.5.0 — Verified CI/CD` | 2–3 | CI, supply chain e release engineering executados com evidência. |
| `v0.6.0 — Production Front-end` | 4 | Front-end real e E2E aprovados contra API/agente. |
| `v0.7.0 — FreeBSD Read-only` | 5–6 | Observabilidade e inventário FreeBSD completos, ainda sem mutação. |
| `v0.8.0 — AD Member Diagnostics` | 7 | Pré-requisitos de member server diagnosticados, sem ingresso. |
| `v0.9.0 — Safe Samba Mutations` | 8 | Primeira mutação Samba com aprovação, backup e rollback. |
| `v0.9.5 — NFSv4 ACL and Domain Join` | 9–10 | ACL NFSv4 e ingresso como membro homologados com controles críticos. |
| `v1.0.0 — Stable` | 11–12 | Segurança dinâmica, homologação e release estável aprovadas. |

Essa decisão deve ser revista em `GOV-006`, juntamente com a unificação das versões do frontend, backend e pacote.

## Project v2

Nome: `Samba Admin — Roadmap v1.0.0`.

Campos mínimos:

| Campo | Tipo | Valores |
|---|---|---|
| Status | single select | Backlog, Ready, In progress, Review, Homologation, Done, Blocked |
| Sprint | single select | Sprint 0 a Sprint 12 |
| Story Points | number | Valor do TSV |
| Prioridade | single select | P0, P1, P2, P3 |
| Risco | single select | Low, Medium, High, Critical |
| Área | single select | Frontend, Backend, Agent, FreeBSD, Samba, AD, ACL, Security, Observability |

Visualizações recomendadas:

- Roadmap por Milestone;
- execução atual por Status;
- backlog por Sprint;
- riscos High/Critical;
- trabalho por Área;
- homologação e releases.

O `gh` ainda não oferece uma interface estável para criar todas as visualizações do Project. A automação cria o Project, campos e itens; as visualizações devem ser configuradas na interface e verificadas em `GOV-001`.

## Definição global de pronto

Uma Issue só pode ser encerrada quando seus critérios forem atendidos e houver evidência proporcional ao risco. Em código, isso inclui formatação, lint, testes unitários e integração, E2E quando aplicável, contrato OpenAPI e código gerado atualizados, documentação, logs sanitizados, feature flag/capability, análise de segurança e rollback testado. Mudanças críticas em agente, ACL, AD, autenticação ou mutações Samba exigem dois revisores.

Fechar uma Issue sem evidência não muda uma capability para `suportado`. A fonte operacional dessa classificação é `docs/current-state.md`.

## Bootstrap e reconciliação

Pré-requisitos:

```bash
gh auth status
gh auth refresh -s repo,project,read:project
```

Visualizar o plano sem modificar o GitHub:

```bash
./scripts/bootstrap-github-project.sh
```

Aplicar ou reconciliar:

```bash
./scripts/bootstrap-github-project.sh --apply
```

O script é idempotente por título de label/Milestone/Project e pelo código estável das Issues. Ele não fecha Issues, não altera código do produto e não habilita operações reais.

