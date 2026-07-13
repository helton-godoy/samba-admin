# Homologacao de ACL NFSv4

## Escopo inicial

Implementar primeiro leitura e parser de ACL NFSv4 em UFS2. A escrita continua bloqueada ate que a matriz de testes seja concluida no pacote Samba e FreeBSD alvo.

## Dados que o parser deve preservar

- Principais `owner@`, `group@`, `everyone@`, usuarios/grupos locais e AD.
- Tipo Allow/Deny, permissoes, flags, heranca, ordem e marca inherited.
- Resolucao de identidade e mapeamento SID/UID/GID quando houver evidencia suficiente.
- Resultado efetivo com limitacoes declaradas, nunca inferencia de equivalencia perfeita com ACL NT.

## Matriz obrigatoria antes de escrita

| Cenario | Evidencia |
| --- | --- |
| UFS2 com `nfsv4acls` | Criacao, leitura e heranca de arquivo/diretorio. |
| Ordem Allow/Deny | Acesso permitido e negado reproduzido localmente e por SMB. |
| Windows 11 | Explorer, criacao, renomeacao, copia e exibicao de seguranca. |
| Samba membro | Resolucao de grupo AD e aplicacao de ACL pelo cliente SMB. |
| Falha e rollback | Exportacao, restauracao e comparacao de ACL antes/depois. |
| Concorrencia | Arquivo aberto, alteracao concorrente e cancelamento de job. |

## Proibicoes ate a homologacao

Nao pressupor equivalencia entre NFSv4 ACL, NT ACL, POSIX mode e POSIX.1e ACL. Nao permitir conversao recursiva, alteracao de mount option, `fstab` ou reparo automatico de ACE ordering.

## Evidencias

Anexar fixture sanitizada, versao de FreeBSD/Samba, opcoes de montagem, caso de teste Windows, saida de parser e hash de backup. Uma diferenca semantica e motivo para manter a capability como `nao_verificado`.
