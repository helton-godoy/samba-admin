# Adapters FreeBSD Somente Leitura

## Politica

Nesta fase, adapters executam apenas comandos fixos de leitura. A coleta nao chama `net ads join`, `service restart`, `setfacl`, `mount`, `umount`, `pw`, `pkg install`, `sysrc` ou qualquer comando que altere estado. Resultado ausente ou erro de comando gera capability `nao_verificado`, nao suposicao positiva.

## Matriz de inventario

| Adapter | Dados | Evidencia de leitura |
| --- | --- | --- |
| SystemAdapter | release, arquitetura, hostname/FQDN, uptime, CPU, memoria, carga, rede, DNS, NTP, timezone | `uname`, `sysctl`, `uptime`, `ifconfig`, `route`, `resolv.conf`, `date` em allowlist. |
| FilesystemAdapter | device, mount, tipo, uso, opcoes, fstab, ACL, quota, xattrs, shares associados | `mount`, `df`, `fstyp`, leitura sanitizada de `fstab`, `getfacl` somente quando presente. |
| SambaAdapter | versao/pacote, binarios, configuracao efetiva, shares, sessoes, arquivos abertos, VFS, testparm, DFS/impressao | `pkg info`, `testparm -s`, `smbstatus` e descoberta de binarios. |
| DomainAdapter | dominio, realm, join, DC/site, DNS, Kerberos, LDAP/SMB, winbind, idmap, resolucao e relogio | consultas de diagnostico sem credencial, como `wbinfo --ping-dc` quando disponivel. |
| CupsAdapter | versao, servico, filas, trabalhos, backends, PPD e erros | `pkg info`, `lpstat`, `cupsctl` somente leitura quando disponivel. |

O provider Go, o runner com limite de saida e os parsers de inventario estao implementados em `backend/internal/adapters/freebsd`. Na API, tanto `agent` quanto o alias controlado `freebsd-readonly` acessam o provider exclusivamente pelo UDS assinado; a API sem root nao executa utilitarios do sistema. O modo `fixture` reproduz somente vetores registrados e nunca cai silenciosamente para execucao real.

## Regras de implementacao

1. Cada invocacao usa binario absoluto allowlisted e argumentos constantes.
2. Definir timeout, limite de saida, codigo de erro estruturado e mascaramento antes de persistir/emitir evento.
3. Parser recebe fixture sanitizada e possui teste de caso valido, ausente, formato inesperado e erro de comando.
4. Capability inclui estado, versao detectada, pre-requisito e evidencia coletada.
5. Nenhuma credencial, keytab, ticket Kerberos, chave privada ou configuracao secreta e lida.

## Homologacao

Comparar cada resposta com a saida manual do FreeBSD e com fixture versionada. Validar pelo menos um host sem Samba, um host Samba membro e um host com capability ausente para evitar falsos positivos.

Nesta rodada o inventario foi exercitado no FreeBSD `15.1-RELEASE-p1` com Samba `4.23.8_1` e CUPS `2.4.19_1`; a fixture sanitizada foi incorporada ao replay. O host nao possui configuracao Samba ativa, AD ou segundo volume UFS2. A leitura de `nfsv4acls` nas opcoes de montagem nao equivale a interpretar ACEs: o parser NFSv4 real, heranca, ordenacao e permissoes efetivas permanece pendente, por isso `acl.nfsv4.read` nao pode ser elevado.
