# Samba Admin — FreeBSD, Samba e Active Directory

Monorepo de evolução do protótipo de console web para administração de Samba no FreeBSD.

```text
samba-admin/
├── frontend/   React + TypeScript + Vite
└── backend/    Go + REST + SSE + SQLite + agente UDS
```

## Arquitetura executável

```text
Navegador
   ↓ HTTPS / REST / SSE
frontend
   ↓
samba-admin-api sem root
   ↓ UDS autenticado por HMAC
samba-admin-agent privilegiado
   ↓
adapters mock / fixture / FreeBSD somente leitura
```

## Execução integrada (Modo Mock/Local)

Terminal 1:

```bash
cd backend
./scripts/dev.sh
```

Terminal 2:

```bash
cd frontend
cp .env.example .env
npm ci
npm run dev
```

A variável `VITE_USE_MSW=false` faz o front-end utilizar a API Go pela proxy do Vite. Para retornar ao protótipo totalmente mockado, use `VITE_USE_MSW=true`.

## Verificação Integrada (FreeBSD Real)

Para consultar a VM FreeBSD provisionada e iniciar somente o front-end local, mantenha o tunel no primeiro terminal:

```bash
ssh -F /dev/null -N -L 8080:127.0.0.1:8080 root@192.168.122.84
```

No segundo terminal:

```bash
./scripts/run-integrated.sh
```

O script executa apenas `uname`, `freebsd-version`, consultas `pkg` e `GET /healthz`. Ele nao copia arquivos, nao compila no host remoto, nao instala packages e nao inicia/reinicia servicos. O Vite escuta apenas em `127.0.0.1` e usa a proxy local para o tunel.

Após a execução, acesse [http://localhost:5173](http://localhost:5173) no navegador.

## Evidencia local preservada do RC0

Os resultados abaixo pertencem ao baseline que produziu o package `0.3.4`; eles nao constituem aprovacao do package tecnico `0.3.5` nem da distribuicao do RC1.

```text
OpenAPI: validacao, geracao e verificacao de artefatos
Backend: testes, vet, race detector, staticcheck e build
Front-end: 13 testes unitarios, build e 20 cenarios Playwright/MSW
Seguranca: govulncheck e npm audit sem vulnerabilidades conhecidas
```

## RC0 FreeBSD

O package backend `0.3.4` foi construido e exercitado no FreeBSD `15.1-RELEASE-p1`. Para repetir o ciclo numa VM descartavel com snapshot:

```sh
cd backend
env GOTOOLCHAIN=local go126 test ./...
make GO=go126 build build-tools
make -C packaging/freebsd package
cd ..
ops/freebsd-lab/smoke-package.sh backend/packaging/freebsd/packages/samba-admin-0.3.4.pkg
```

Na VM de homologacao atual a API escuta apenas em `127.0.0.1:8080`. Use tunel SSH, nao exponha a porta diretamente:

```sh
ssh -F /dev/null -L 8080:127.0.0.1:8080 root@192.168.122.84
```

## RC1 somente leitura — evidencia tecnica de laboratorio

O contrato aditivo agora inclui `GET /api/v1/samba` e `GET /api/v1/cups`, com respostas estruturadas e erro RFC 7807/HTTP 503 quando o agente, a capability ou a sondagem minima estiver indisponivel. No modo real, ambos percorrem exclusivamente API sem root → UDS autenticado → agente privilegiado → provider FreeBSD somente leitura. `/api/v1/printers`, `/api/v1/shares` e os aliases legados permanecem preservados.

Em 2026-07-13, os cinco comandos obrigatorios passaram localmente na ordem definida, assim como `go vet`, `go test -race`, `staticcheck`, 25 testes Vitest, verificacao de sintaxe dos scripts, `git diff --check` e sanitizacao dos artefatos E2E. O `frontend-dist` local contem somente HTML/CSS/JS e foi gerado em `artifacts/frontend-dist.tar.gz`, SHA-256 `2add81ef1c34266a17f2279eed56d99ddd3a8b2ccfaadfc36ac29f53ae546823`. O template nginx foi renderizado com parametros de teste; `nginx -t`, certificado, dominio e ativacao institucional nao foram executados.

O package tecnico `samba-admin-0.3.5`, construido nativamente com `RELEASE=no`, foi aprovado apenas para laboratorio FreeBSD somente leitura. Seu SHA-256 e `72b05ee3aef02a3cb59a1bffd4ffcba637ddc295b8d0f71fbdff96c4d43edd89`. A VM `192.168.122.84` terminou com API/agente ativos, API em loopback, adapter `agent`, executor `readonly-freebsd` e mutacoes desabilitadas. O rollback `0.3.4` foi preservado e testado.

As suites Playwright MSW e API Go real possuem 20 cenarios cada, mas nao foram aprovadas nesta rodada: o sandbox bloqueou navegador e sockets com `EPERM`, antes de qualquer assertion da aplicacao. O backend Go/agente fixture por UDS da suite real chegou a iniciar, e os artefatos da tentativa foram sanitizados sem senha, TOTP, trace, PNG ou WebM. GitHub Actions, gitleaks, govulncheck e SBOM nao foram executados; `npm audit` foi inconclusivo por `EAI_AGAIN`.

## Limite deste incremento

O adapter FreeBSD executa somente comandos fixos de leitura no agente privilegiado quando explicitamente habilitado. Toda escrita real, ingresso/saida AD, ACL, CUPS, quota, syslog e AD DC permanece bloqueada. A decisao formal e **no-go para distribuir ou promover o RC1 a producao** enquanto E2E browser, CI/scanners/SBOM e TLS institucional nao forem concluidos. O package `0.3.5` e apenas uma evidencia tecnica de laboratorio somente leitura.

## Situação detalhada

Consulte [`docs/current-state.md`](docs/current-state.md) e [`docs/release-candidate-report.md`](docs/release-candidate-report.md) para distinguir funcionalidades concluidas, simuladas, parciais e dependentes de homologacao.
