# Seguranca do Agente Privilegiado

## Fronteira de privilegio

`samba-admin-api` executa sem root. `samba-admin-agent` e o unico processo autorizado a atingir adaptadores privilegiados. O browser jamais conversa com o socket.

## Socket e autenticacao de mensagem

- Usar Unix Domain Socket com diretorio controlado, dono/grupo fixos e modo restritivo.
- Validar peer credentials quando o sistema suportar; rejeitar UID/GID inesperado.
- Exigir HMAC-SHA256, timestamp em janela configuravel, nonce unico e limite de tamanho antes de desserializar payload.
- Armazenar chave fora do SQLite, com permissao minima, rotacao planejada e periodo de sobreposicao auditado.
- Limitar conexoes concorrentes, tamanho de resposta e taxa por par.

O package usa `SAMBA_ADMIN_AGENT_KEY_ID`, chave anterior opcional para rotacao, `SAMBA_ADMIN_AGENT_NONCE_STORE`, TTL/cache de nonce, limites de relogio/mensagem/saida/concorrencia, e owner/grupo/mode/UID esperado do socket. O executor `readonly-freebsd` falha fechada sem esses valores e sem capabilities explicitamente homologadas.

## Catalogo fechado

Cada operacao declara nome, versao, schema, risco, timeout, lock, executaveis/argumentos/caminhos permitidos, backup, approval, capability, rollback, campos sensiveis e estrategia de auditoria. Operacao desconhecida falha fechada.

## Execucao

O executor usa caminho absoluto com `exec.CommandContext(ctx, binary, args...)`, ambiente minimo, diretorio controlado, `umask`, timeout e limite de stdout/stderr. `sh -c`, shell generico, concatenacao de argumentos e escrita em caminhos nao allowlisted sao proibidos.

## Estado atual de liberacao

Socket, peer credentials (Linux e FreeBSD com CGO), HMAC/rotacao, timestamp, nonce persistente, limites e executor somente leitura estao implementados e testados localmente. Antes da homologacao FreeBSD, o catalogo nao autoriza escrita real e a compilacao nativa ainda deve provar `getpeereid`, dono/grupo/modo e replay apos reinicio.

O campo `requiresApproval` do catalogo e metadado de politica; o agente ainda nao valida uma atestacao de aprovacao independente. Isso nao cria exposicao nesta versao porque todos os executores reais recusam mutacoes. Uma credencial de aprovacao assinada, vinculada a operacao/recurso/hash/janela, e gate obrigatorio antes do primeiro executor mutavel.
