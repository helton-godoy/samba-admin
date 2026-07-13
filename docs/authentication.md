# Autenticacao Institucional e Conta de Emergencia

## Separacao obrigatoria

Identidade administrativa do console, credencial de ingresso do Samba no dominio e identidade de servico sao objetos distintos. A senha usada para `net ads join` nunca e reaproveitada para login web e nunca e persistida.

## Conta break-glass

Uma conta local de emergencia permanece disponivel quando LDAP/OIDC estiver indisponivel. A politica exige senha forte, expiração configuravel, bloqueio temporario, auditoria de uso, alerta imediato e restricao por rede quando configurada. O sistema nao pode remover ou desativar a ultima conta break-glass sem criar outra conta valida na mesma transacao.

## Provedores institucionais

LDAP deve usar LDAPS ou StartTLS com validacao de CA, hostname e politica de certificado. OIDC deve validar emissor, audiencia, assinatura, `nonce`, expiracao e mapeamento de grupos. Falha de provedor externo deve falhar fechada, exceto pelo fluxo break-glass explicitamente autorizado.

## Sessao

Regenerar identificador de sessao no login e apos elevacao de privilegio; expirar sessoes inativas e revogar em logout, troca de senha, revogacao MFA ou desabilitacao de conta. Registrar usuario, origem, resultado e correlation ID, sem senha, token, cookie ou segredo TOTP.
