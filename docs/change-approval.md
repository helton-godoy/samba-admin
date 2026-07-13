# Aprovacao de Mudancas e Segregacao de Funcoes

## Classificacao

| Classe | Exemplo | Regra |
| --- | --- | --- |
| Leitura | Inventario e diagnostico | Sem aprovacao, com auditoria. |
| Baixo risco | Validacao sem escrita | Solicitante autorizado e registro. |
| Medio risco | Escrita em area temporaria | Justificativa, lock e controle de mudanca. |
| Alto risco | Perfil Samba, idmap, domain join | Dois responsaveis, janela, impacto e rollback. |
| Destrutiva | Exclusao de share, ACL recursiva | Aprovacao reforcada e evidencia de backup/restore. |
| Emergencia | Correcao de disponibilidade critica | Politica excepcional, registro imediato e revisao posterior. |

## Duas etapas

Mudancas de alto risco, destrutivas e AD DC exigem solicitante e aprovador diferentes. A solicitacao contem recurso, diff, impacto, janela de manutencao, plano de rollback, ticket externo quando aplicavel e expiracao. Aprovacao expirada, revogada ou pertencente ao solicitante e invalida a execucao.

## Operacoes que exigem aprovacao

- Modelo de ACL, `/etc/fstab`, mount options e ACL recursiva.
- Ingresso/saida do dominio, perfil Samba e idmap.
- Reinicio Samba com sessoes ativas e exclusao de compartilhamento.
- Drivers de impressao, syslog remoto, rollback global e qualquer AD DC.

## Execucao e auditoria

O job revalida RBAC, approval, versao do recurso, capability, janela e lock imediatamente antes da execucao. A auditoria registra quem solicitou, quem aprovou, quando, diff/hash, resultado e rollback, sem segredos.
