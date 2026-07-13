# Plano de homologação no FreeBSD 15.1

- Criar VM isolada com snapshot e dois UFS2 separados: um com `nfsv4acls`, outro com `acls`.
- Instalar exatamente o pacote Samba pretendido e capturar versão/origem/opções.
- Habilitar primeiro apenas inventário; comparar fixtures com saídas reais.
- Validar DNS, NTP, Kerberos, winbind e idmap no domínio de teste.
- Criar compartilhamento descartável, executar testparm, SMB 3, ACL e acesso Windows 11.
- Testar concorrência, arquivo aberto, rollback e interrupção de serviço.
- Homologar auditoria e volume no SIEM.
- Somente depois liberar escrita por feature flag e por papel RBAC.
