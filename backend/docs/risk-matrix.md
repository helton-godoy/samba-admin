# Matriz de riscos resumida

| Risco | Impacto | Controle |
|---|---|---|
| Execução privilegiada indevida | Crítico | agente mínimo, catálogo fechado, HMAC/UDS |
| Conversão de ACL com perda | Crítico | exportação, simulação, manutenção e rollback |
| idmap instável | Alto | validação SID↔UID/GID antes de ACL/cota |
| Reconfiguração concorrente | Alto | ETag, lock por recurso e jobs persistentes |
| Vazamento de segredo | Alto | redaction, memória temporária e auditoria sem conteúdo sensível |
| Auditoria excessiva | Médio | perfis, estimativa de volume e retenção |
| Recurso Samba não disponível | Alto | capability matrix detectada no host |
