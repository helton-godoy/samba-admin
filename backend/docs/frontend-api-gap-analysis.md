# Análise de lacunas entre o front-end e a API

## Cobertura encontrada

O protótipo consumia 22 operações HTTP. Todas foram preservadas no backend. Os caminhos `/api/v1/acl` e `/api/v1/principals` são aliases legados; os recursos canônicos são `/api/v1/acls` e `/api/v1/identities`.

| Tela | Operações principais | Estado |
|---|---|---|
| Painel | system, capabilities, filesystems | Implementado |
| Compartilhamentos | list, preview, create | Implementado com tarefa persistente |
| ACL | list, identities, effective, conversion simulation | Implementado em modo mock |
| Domínio | state, diagnostics | Implementado em modo mock |
| Impressão, DFS, cotas | listagens | Implementado em modo fixture |
| Serviços | list, action | Ação gera tarefa persistente |
| Editor | read allowlisted, validate | Implementado |
| Tarefas | list, cancelamento, rollback, SSE | Implementado |

## Lacunas ainda existentes

- O front-end ainda não possui tela de login; `development-bypass` é usado durante integração.
- O front-end não consome SSE automaticamente; a API já disponibiliza `/api/v1/events`.
- Alterações de impressora, DFS, quotas, logs e auditoria ainda não possuem formulários mutáveis completos.
- ETag está ativo nas informações de sistema e `Idempotency-Key` está persistido na criação de compartilhamentos; a generalização para todas as mutações permanece pendente.
- O adapter FreeBSD real permanece desabilitado.
