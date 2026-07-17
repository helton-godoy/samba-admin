# AGENTS.md

## Contexto geral

Monorepo com dois pacotes independentes: `frontend/` (React + TypeScript + Vite) e `backend/` (Go + SQLite + UDS). O frontend consome a API via proxy do Vite ou MSW (mock). A linguagem padrão de interação é **português (pt-BR)** — commits, mensagens de log, testes, erros e esta documentação devem usar pt-BR.

## Comandos essenciais

### Verificação completa (ordenada — respeitar a ordem)

```bash
make openapi-validate   # valida spec OpenAPI 3.1
make generate           # gera código backend (oapi-codegen) + frontend (openapi-typescript)
make check-generated    # verifica se código gerado está em dia com a spec
make test               # backend (go test) + frontend (vitest)
make build              # backend (2 binários Go) + frontend (tsc + vite build)
```

### Backend isolado

```bash
cd backend
GOTOOLCHAIN=local go test ./...          # testes unitários
GOTOOLCHAIN=local go vet ./...           # verificação estática
go run honnef.co/go/tools/cmd/staticcheck@2025.1.1 ./...  # static analysis
make build                               # compila samba-admin-api + samba-admin-agent
./scripts/dev.sh                         # modo mock (API única, sem agente)
./scripts/dev-with-agent.sh              # API + agente em processos separados via UDS
```

### Frontend isolado

```bash
cd frontend
npm ci
npm run test          # vitest (unit)
npm run build         # tsc -b + vite build
npm run test:e2e      # playwright (chromium, precisa de npx playwright install --with-deps chromium)
```

### Geração e contrato OpenAPI

A fonte de verdade é `backend/api/openapi.yaml`. Código gerado:
- Backend: `backend/api/generated/` (models.gen.go + server.gen.go) — via `oapi-codegen v2.7.0`
- Frontend: `frontend/src/api/generated/openapi.ts` — via `openapi-typescript`

**Atenção:** `frontend/src/api/generated.ts` é um wrapper manual de compatibilidade acima do tipo gerado; não é output de codegen. Novos consumers devem usar `createOpenApiClient` de `frontend/src/api/client.ts`.

Para detectar breaking changes (requer `OPENAPI_BASE_REF` em CI):
```bash
./scripts/openapi-breaking.sh
```

## Arquitetura executável

```text
Navegador → frontend (React/Vite)
   ↓ REST / SSE / cookies + CSRF
samba-admin-api (Go, sem root)
   ↓ UDS autenticado por HMAC + nonce + timestamp
samba-admin-agent (Go, privilegiado)
   ↓
adapters: mock | fixture | freebsd (somente leitura, desabilitado por projeto)
```

Dois binários Go: `cmd/samba-admin-api` e `cmd/samba-admin-agent`. A API nunca executa comandos diretamente; delega ao agente por UDS.

## Modos de operação do backend

| Variável | Efeito |
|---|---|
| `SAMBA_ADMIN_ADAPTER_MODE=mock` | Dados simulados em memória (padrão dev) |
| `SAMBA_ADMIN_ADAPTER_MODE=fixture` | Replay de fixture JSON sanitizada, sem acessar o host |
| `SAMBA_ADMIN_ADAPTER_MODE=agent` | API delega ao agente via UDS |
| `SAMBA_ADMIN_ADAPTER_MODE=freebsd-readonly` | Adapter real somente leitura (exige VM FreeBSD) |
| `SAMBA_ADMIN_AUTH_MODE=development-bypass` | Ignora autenticação (padrão dev) |
| `SAMBA_ADMIN_AUTH_MODE=local` | Login local com Argon2id + sessão + CSRF |

## Estrutura de diretórios importante

- `backend/cmd/` — entrypoints dos dois binários
- `backend/internal/adapters/` — contratos (`contracts.go`) e implementações (mock, fixture, freebsd)
- `backend/internal/agent/` — servidor UDS, cliente, executor, protocolo de assinatura
- `backend/internal/httpapi/` — rotas e handlers HTTP
- `backend/internal/storage/` — SQLite WAL + migrações
- `backend/migrations/` — SQL de migração versionado
- `frontend/src/api/client.ts` — cliente REST tipado com `createOpenApiClient`
- `frontend/src/api/generated.ts` — tipos + facade de compatibilidade (não é output direto de codegen)
- `frontend/src/mocks/` — MSW handlers e dados simulados
- `frontend/src/test/` — testes unitários com `renderWithProviders` helper
- `frontend/e2e/` — testes Playwright

## CI (GitHub Actions)

Jobs paralelos em `.github/workflows/ci.yml`:
1. **openapi** — valida spec, gera código, verifica `check-generated`
2. **openapi-breaking-change** — detecta breaking changes (PRs apenas)
3. **backend** — vet + test (com coverage) + race detector + staticcheck + govulncheck + build
4. **frontend** — npm ci + vitest + vite build
5. **frontend-e2e** — Playwright com Chromium
6. **security-and-sbom** — npm audit + gitleaks + CycloneDX SBOM
7. **freebsd** — build + test nativo FreeBSD 15.1 via `vmactions/freebsd-vm`

## Constrangimentos e convenções

- Backend usa CGO para SQLite e Argon2id; `libsqlite3-dev` e `libargon2-dev` são obrigatórios no Debian/Ubuntu
- `GOTOOLCHAIN=local` é necessário nos testes Go para evitar download automático de toolchain
- Frontend testes usam MSW (Mock Service Worker) — `src/test/setup.ts` inicializa o servidor MSW
- `renderWithProviders` em `src/test/render.tsx` é o helper padrão para testes React (QueryClient + MemoryRouter)
- Playwright rodará com `VITE_USE_MSW=true` por padrão (ver `playwright.config.ts`)
- Operações mutáveis na API exigem `Idempotency-Key` e `X-CSRF-Token`
- O agente aceita somente operações listadas em `internal/agent.Catalog` — não há shell genérico
- FreeBSD adapter permanece desabilitado por projeto; não assumir que executa comandos reais
- Spec OpenAPI é a fonte de verdade do contrato; quebras de compatibilidade exigem aprovação explícita
- Commits, mensagens de teste e logs devem seguir pt-BR

## Memória operacional e execução de Issues

O GitHub é a fonte de planejamento e continuidade entre sessões. Antes de executar uma Issue, siga `docs/github-project-memory.md` e recupere o contexto com `gh`/`git`. O roadmap detalhado está em `docs/v1-roadmap.md`; o backlog normalizado e as chaves estáveis estão em `.github/project/backlog.tsv`.

Toda Issue do roadmap deve ser tratada como contrato executável: confirme o estado atual no código, respeite dependências e gates, implemente somente o escopo descrito, preserve as invariantes fail-closed e anexe evidências e rollback à Pull Request. O encerramento de uma Issue não muda por si só o estado de uma capability; atualize `docs/current-state.md` quando houver homologação suficiente.
