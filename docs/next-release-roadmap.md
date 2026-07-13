# Roadmap do Proximo Release Candidato

## Objetivo

Entregar uma versao candidata segura para homologacao tecnica, inicialmente somente leitura no FreeBSD. O perfil institucional principal e servidor Samba membro de um Active Directory existente; Samba AD DC nao faz parte do caminho critico.

## Fases e gates

| Fase | Entrega | Gate de saida |
| --- | --- | --- |
| 0 - Contrato | OpenAPI validado, tipos Go e cliente TypeScript gerados, aliases documentados | CI reprova artefato gerado desatualizado e breaking change sem decisao registrada. |
| 1 - Console | Login, sessao, CSRF, MFA, capabilities, SSE com polling de contingencia | Fluxos browser cobertos por Playwright contra API real de teste. |
| 2 - Governanca | RBAC por acao/recurso e aprovacao em duas etapas | Solicitante nao aprova a propria mudanca quando a politica exige segregacao. |
| 3 - Plataforma | Agente endurecido, package inicial, observabilidade e SIEM | UDS, replay, limite de mensagem, segredo e auditoria testados. |
| 4 - Homologacao leitura | Laboratorio FreeBSD, fixtures sanitizadas e adapters somente leitura | Evidencia coletada em FreeBSD homologado sem escrita no host. |
| 5 - Member server | Diagnostico AD e previa de ingresso | DNS, SRV, horario, LDAP, SMB, Kerberos, idmap e conflitos validados. |
| 6 - Escrita limitada | Geracao/validacao em area temporaria e primeiro compartilhamento | Backup, `testparm`, reload, health check e rollback demonstrados. |

## Fora do escopo do release candidato

- Conversao ou escrita recursiva de ACL.
- Alteracao de `/etc/fstab`, mount options ou quotas.
- Ingresso/saida real do dominio.
- Drivers Windows, syslog TLS e operacoes AD DC.
- Promocao automatica de qualquer servidor para controlador de dominio.

## Criterios de liberacao

1. Todos os jobs obrigatorios de CI passam e os artefatos possuem checksum/SBOM.
2. O package e construido e testado em FreeBSD suportado, nao apenas por cross-compilation Linux.
3. A documentacao operacional e o plano de desastre foram exercitados no laboratorio.
4. Nenhum segredo, keytab, ticket ou senha aparece em fixture, log, job, auditoria ou artefato de CI.
5. A matriz de risco e as capabilities exibem explicitamente o que continua bloqueado.

## Rollout gradual

1. Instalar API sem agente habilitado e somente leitura.
2. Comparar inventario com operacao manual e validar alertas/SIEM.
3. Habilitar validacao sem escrita em laboratorio.
4. Autorizar uma unica operacao de compartilhamento sob janela de manutencao aprovada.
5. Expandir somente apos evidencias e rollback comprovado.
