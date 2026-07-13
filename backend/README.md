# Samba Admin Backend

Backend em Go para o console web de administração Samba/FreeBSD. O projeto executa em ambiente de desenvolvimento sem FreeBSD e mantém os adapters reais desabilitados.

## Estado do incremento

Implementado:

- API REST v1 compatível com o front-end existente;
- OpenAPI 3.1 com 36 paths;
- SQLite real em modo WAL;
- migração inicial versionada;
- autenticação local e Argon2id;
- RBAC por ação e recurso;
- sessões, cookies seguros e CSRF;
- auditoria encadeada por SHA-256;
- jobs persistentes, locks, progresso e rollback simulado;
- Server-Sent Events;
- API e agente em processos separados;
- Unix Domain Socket com HMAC, timestamp e nonce;
- catálogo fechado de operações privilegiadas;
- adapters mockados e fixtures;
- allowlists de caminhos e arquivos;
- testes unitários e de integração.

Ainda depende de ambiente FreeBSD:

- inventário real do host;
- execução de `testparm`, `smbstatus`, `wbinfo`, `service` e ferramentas ACL/quota;
- aplicação real de configurações;
- ingresso real no domínio;
- CUPS, DFS e syslog reais;
- empacotamento `.pkg` homologado.

## Dependências de sistema para compilação

O núcleo não depende de módulos Go externos. Para SQLite e Argon2id, a compilação usa CGO:

### Debian/Ubuntu

```bash
sudo apt install build-essential libsqlite3-dev libargon2-1
```

### FreeBSD

```sh
pkg install go sqlite3 libargon2
```

O projeto foi validado com Go 1.23.2 no ambiente de geração. O `go.mod` indica `toolchain go1.26.5`, versão estável de referência no momento da criação.

## Execução rápida — modo mock

```bash
cp config.example.env .env
set -a
. ./.env
set +a
./scripts/dev.sh
```

A API ficará em `http://localhost:8080`.

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/api/v1/system
```

## Execução com API e agente separados

```bash
export SAMBA_ADMIN_AGENT_KEY='substitua-por-chave-aleatoria-com-32-bytes-ou-mais'
./scripts/dev-with-agent.sh
```

Neste modo, a API envia intenções estruturadas pelo socket `./var/samba-admin-agent.sock`. O executor continua mockado.

## Autenticação local

O modo padrão de desenvolvimento é:

```text
SAMBA_ADMIN_AUTH_MODE=development-bypass
```

Para testar autenticação local:

```bash
export SAMBA_ADMIN_AUTH_MODE=local
export SAMBA_ADMIN_BOOTSTRAP_USER=admin
export SAMBA_ADMIN_BOOTSTRAP_PASSWORD='Senha-Muito-Forte-Temporaria-2026!'
go run ./cmd/samba-admin-api
```

A senha inicial é exigida somente quando o banco ainda não possui usuários. Não a coloque em arquivos versionados.

Login:

```bash
curl -i -c cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"Senha-Muito-Forte-Temporaria-2026!"}' \
  http://localhost:8080/api/v1/auth/login
```

O header `X-CSRF-Token` retornado deve ser enviado nas operações mutáveis.

## Testes

```bash
make test
```

Cobertura funcional atual:

- Argon2id;
- RBAC;
- SQLite/WAL e persistência;
- lock concorrente;
- cadeia hash da auditoria;
- assinatura e comunicação UDS;
- catálogo fechado do agente;
- endpoints usados pelo front-end;
- criação de compartilhamento e job;
- conclusão de tarefa;
- path traversal;
- SSE.

## OpenAPI e geração

A fonte de contrato está em `api/openapi.yaml`.

```bash
make generate-openapi
```

Esse alvo usa `oapi-codegen` e exige acesso ao repositório de módulos Go. O runtime não depende do código gerado; isso permite compilar o protótipo em ambientes offline. A adoção dos tipos gerados no runtime está prevista para o próximo ciclo, após congelamento do contrato.

## Segurança operacional

- Não existe endpoint de shell genérico.
- O agente aceita somente operações presentes em `internal/agent.Catalog`.
- Executáveis reais deverão ter caminhos absolutos e argumentos separados.
- O navegador nunca acessa diretamente o agente.
- O adapter FreeBSD deve revalidar todos os caminhos e usar controles anti-symlink.
- O perfil AD recomendado para servidor de arquivos é `domain-member`; AD DC permanece módulo isolado e não verificado.

Consulte `docs/` para arquitetura, riscos, homologação e backlog.

## Controles adicionais implementados

- bloqueio local por 15 minutos após cinco tentativas inválidas;
- timeout HTTP, preservando o endpoint SSE de longa duração;
- endereço de auditoria obtido do par TCP, sem confiar automaticamente em `X-Forwarded-For`;
- `Idempotency-Key` persistida em SQLite para criação de compartilhamentos;
- cancelamento de tarefas em execução;
- contratos explícitos de adapters em `internal/adapters`.

O estado detalhado de cada fase está em `../IMPLEMENTATION_STATUS.md`.
