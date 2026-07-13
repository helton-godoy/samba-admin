# Samba como Membro de Active Directory

## Perfil principal

O servidor opera como membro de um Active Directory existente para fornecer arquivos e impressao. Isso permite autenticar usuarios do dominio, resolver grupos e aplicar ACLs sem promover o servidor a controlador de dominio.

Samba AD DC e um modulo separado, desabilitado por padrao e `nao_verificado`. Um servidor membro nao deve ser chamado de "promovido", nem reutilizar configuracao, banco ou fluxo de AD DC.

## Parametros institucionais de exemplo

Os valores abaixo sao exemplos de configuracao, nunca credenciais:

```text
Dominio DNS: ebserhnet.ebserh.gov.br
Realm: EBSERHNET.EBSERH.GOV.BR
DC preferencial 1: CAT-PVW-AD1.ebserhnet.ebserh.gov.br
DC preferencial 2: CAT-PVW-AD2.ebserhnet.ebserh.gov.br
```

## Diagnostico antes de qualquer ingresso

O diagnostico somente leitura deve validar hostname, FQDN, DNS, SRV, realm, NetBIOS, conectividade LDAP e SMB, Kerberos, diferenca de horario, DC/site, OU pretendida, portas, winbind, idmap, faixas UID/GID e contas locais conflitantes. Cada teste retorna evidencia, timeout e motivo de bloqueio.

## Fluxo futuro protegido

1. Diagnostico e validacao de pre-requisitos.
2. Previa de configuracao e diff sem escrita.
3. Solicitacao aprovada com janela, impacto e rollback.
4. Coleta de credencial em memoria, sem log ou persistencia.
5. Ingresso por operacao allowlisted e teste de resolucao de usuario/grupo.
6. Auditoria, descarte de credencial e rollback se qualquer verificacao falhar.

O feature flag para ingresso permanece desligado ate que adapters de leitura e rollback sejam homologados em dominio de testes.

## Limites

Nao automatizar ingresso em AD real, nao alterar GPO, nao criar contas no dominio e nao tentar promocao para AD DC. Falha de DNS, horario ou idmap bloqueia o fluxo; nao deve existir modo de ignorar essas validacoes na interface.
