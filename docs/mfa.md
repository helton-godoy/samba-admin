# MFA TOTP

## Escopo inicial

O primeiro fator adicional e TOTP compativel com RFC 6238. A politica pode torna-lo obrigatorio por papel, rede, operacao ou perfil de risco. MFA nao substitui RBAC, aprovacao ou controle de sessao.

## Ciclo de vida

1. Usuario autenticado inicia ativacao e recebe segredo somente uma vez por canal autenticado.
2. A confirmacao exige codigo TOTP valido antes de marcar o fator como ativo.
3. O segredo e armazenado cifrado com chave fora do banco; logs, auditoria e eventos armazenam apenas metadados.
4. Codigos de recuperacao sao exibidos uma vez e armazenados como hashes individuais com uso unico.
5. Rotacao e revogacao invalidam codigos antigos e sessoes que exigem reautenticacao.

Os endpoints de ativacao, confirmacao, rotacao e revogacao estao implementados. A interface de login conclui desafios TOTP/recovery; a tela autenticada de autoatendimento para gerir o fator ainda e pendente e nao deve ser substituida por operacao manual no banco.

## Controles

- O desafio e de uso unico, vinculado a IP/user-agent e expira rapidamente. Rate limit dedicado por usuario/IP para tentativas de codigo ainda deve ser adicionado antes de liberar MFA institucionalmente.
- Aceitar janela de relogio minima e monitorar deriva de NTP.
- Exigir reautenticacao/MFA recente para acoes de alto risco.
- Recuperacao administrativa requer justificativa, aprovacao quando a politica exigir e auditoria reforcada.
