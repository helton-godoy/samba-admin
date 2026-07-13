# Estado da Implementacao

Este caminho e mantido por compatibilidade com referencias anteriores. O inventario normativo e atualizado esta em:

- [`docs/current-state.md`](docs/current-state.md): estados operacionais e evidencias.
- [`docs/release-candidate-report.md`](docs/release-candidate-report.md): preservacao, incrementos concluidos e bloqueios.
- [`docs/risk-matrix.md`](docs/risk-matrix.md): riscos e gates.
- [`docs/release-candidate-plan.md`](docs/release-candidate-plan.md): decisao e rollout do primeiro RC.

## Resumo em 2026-07-13

| Area | Estado |
| --- | --- |
| OpenAPI e contratos Go/TypeScript | `GET /api/v1/samba` e `GET /api/v1/cups` foram adicionados, gerados e validados nos cinco comandos obrigatorios executados em ordem. |
| Front-end/API | Login, sessao, MFA, erros, capabilities, approval e SSE/polling implementados; Samba/CUPS usam o cliente tipado. |
| Autenticacao/MFA | Backend local completo para TOTP/recovery; UI de gestao do fator e LDAP/OIDC pendentes. |
| Aprovacao | Persistente, segregada e executada como job; agente ainda nao valida atestacao independente. |
| Agente | Endurecido para mock e leitura FreeBSD; toda mutacao real bloqueada. |
| FreeBSD | Package tecnico `0.3.5` aprovado para laboratorio somente leitura no FreeBSD 15.1; API/agente ativos em loopback, executor `readonly-freebsd` e mutacoes desabilitadas. |
| ACL NFSv4 | Parser real e escrita bloqueados. |
| Active Directory | Diagnostico parcial; ingresso/saida e AD DC bloqueados. |
| Empacotamento | Build/test/race/vet nativos, package `0.3.5`, ABI, smoke, deinstall ativo, reinstalacao e readiness aprovados; SHA-256 `72b05ee3aef02a3cb59a1bffd4ffcba637ddc295b8d0f71fbdff96c4d43edd89`. O `0.3.4` reconstruido e testado permanece como rollback. |
| Testes browser | Suites MSW e API Go real possuem 20 cenarios cada, mas o sandbox bloqueou navegador/sockets com `EPERM` antes de assertions; o E2E permanece nao aprovado. |
| Distribuicao do front-end | `frontend-dist` local aprovado quanto a conteudo/sanitizacao, SHA-256 `2add81ef1c34266a17f2279eed56d99ddd3a8b2ccfaadfc36ac29f53ae546823`; template renderizado, mas `nginx -t`, certificado, dominio e ativacao institucional nao executados. |
| Seguranca/CI | Sanitizador de artefatos passou. `npm audit` foi inconclusivo por `EAI_AGAIN`; gitleaks, govulncheck, SBOM e GitHub Actions nao foram executados nesta rodada. |

O package tecnico `0.3.5` esta aprovado apenas para laboratorio somente leitura. A decisao permanece **no-go para RC1 distribuivel e producao**. Nenhuma capability mutavel esta liberada, e nenhum resultado de GitHub Actions, scanner ou SBOM deve ser inferido desta tabela.
