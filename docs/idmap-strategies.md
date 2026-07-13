# Estrategias de idmap

## Principio

SID para UID/GID deve ser deterministico, estavel e governado antes de ACLs, quotas ou dados persistentes. Mudancas de idmap podem tornar objetos inacessiveis; portanto exigem classificacao de alto risco, backup e aprovacao em duas etapas.

## Opcoes

| Estrategia | Uso | Validacoes obrigatorias |
| --- | --- | --- |
| Backend padrao interno | Mapeamento de dominios sem requisito RFC2307 | Range reservado, nao sobreposto e documentacao da persistencia. |
| `idmap_rid` | Mapeamento deterministico derivado do RID | Base/range por dominio, ausencia de sobreposicao e avaliacao de compatibilidade historica. |
| `idmap_ad` | Atributos RFC2307 do Active Directory | `uidNumber`, `gidNumber`, unicidade, preenchimento, estabilidade, ownership e governanca institucional. |

## Verificacoes

- Nenhuma faixa UID/GID pode se sobrepor a contas locais, outro dominio ou outro backend.
- Validar usuarios e grupos representativos, inclusive administradores, grupos aninhados e identidades sem atributos RFC2307.
- Registrar versao de `smb.conf`, range anterior/proposto, impacto em ACL/quota e amostra SID-UID-GID antes de aplicar.
- Falha de mapeamento e capability `indisponivel`; nao usar fallback silencioso para escrita de ACL ou quota.

## Mudanca controlada

Exportar inventario de ownership/ACL, gerar diff, executar em laboratorio com copia representativa, testar Windows 11 e rollback. Nao reconfigurar idmap em producao para corrigir um unico usuario sem avaliar todos os SIDs existentes.
