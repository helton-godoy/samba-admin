# Integracao SIEM e Observabilidade

## Eventos

Registrar login/logout/falha, uso break-glass, approval, job, rollback, chamada ao agente, capability ausente, lock, conflito, timeout e cancelamento. Cada evento inclui `eventId`, timestamp UTC, `correlationId`, `jobId` quando houver, ator, recurso, acao, resultado e severidade.

## Formatos

Disponibilizar JSON estruturado e mapeamento para syslog RFC 5424. Transporte remoto/TLS permanece `nao_verificado` ate que a implementacao, certificado, fila, retencao e comportamento de falha sejam homologados no FreeBSD alvo.

## Privacidade e segredo

Nunca emitir senha, cookie, token, segredo TOTP, chave HMAC, keytab, ticket Kerberos, conteudo de configuracao secreta ou payload bruto de comando. Campos sensiveis devem ser mascarados antes do logger, event broker e auditoria.

## Retencao e correlacao

Definir retencao com seguranca/compliance, sincronizar horario por NTP e enviar alertas para falha de autenticacao, uso break-glass, approval rejeitada, agente indisponivel, replay/HMAC invalido, rollback e quebra de cadeia de auditoria. A exportacao SIEM complementa, mas nao substitui, a auditoria local encadeada.
