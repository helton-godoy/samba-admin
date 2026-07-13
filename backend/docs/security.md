# Segurança

- A API não executa shell e o agente não aceita `command.execute`, `shell.run` ou equivalentes.
- O catálogo do agente define operação, executáveis absolutos, timeout, lock, backup e aprovação.
- A comunicação UDS usa HMAC-SHA256, timestamp e nonce contra replay.
- Compartilhamentos e editor usam allowlists de caminhos.
- Senhas locais usam Argon2id; há bloqueio temporário após cinco falhas; cookies são HttpOnly/SameSite e Secure fora do modo de desenvolvimento.
- RBAC é avaliado por ação e recurso, não por `isAdmin`.
- Auditoria usa encadeamento SHA-256 para evidenciar alteração retroativa; imutabilidade forte exigirá destino remoto/WORM.
- Segredos não são registrados nos jobs, eventos ou auditoria.

## Limitações atuais

A validação de symlink deve ser reforçada no adapter real com `openat`, `O_NOFOLLOW`, verificação de device/inode e resolução dentro do root autorizado. O protótipo bloqueia travessia lexical, mas não substitui controles do kernel.

## Origem do cliente e proxy reverso

A API não confia em `X-Forwarded-For` por padrão. Ela registra o endereço do par TCP. Antes de operar atrás de proxy reverso, implemente uma allowlist de proxies confiáveis e só então processe headers encaminhados.

## Controles ainda pendentes para produção

- rate limiting distribuído e política institucional de desbloqueio;
- MFA e federação OIDC/AD;
- secret store externo;
- aprovação em duas etapas;
- auditoria remota imutável;
- testes anti-symlink no adapter FreeBSD real.
