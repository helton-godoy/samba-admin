# Laboratorio FreeBSD Reproduzivel

## Escopo e seguranca

O laboratorio valida FreeBSD 15.1-RELEASE, ou versao explicitamente aprovada, sem tocar no AD de producao. A rede deve ser isolada; a integracao com dominio de testes e opcional e requer confirmacao humana separada. Scripts deste repositorio nao executam ingresso no dominio, mudanca de ACL, `fstab`, Samba ou CUPS.

## Topologia minima

- Uma VM FreeBSD com dois discos: sistema e volume UFS2 descartavel para testes.
- Snapshot/checkpoint antes de cada rodada e identificador registrado na evidencia.
- Rede administrativa isolada, DNS/NTP de laboratorio e firewall com portas minimas.
- Cliente Windows 11 de teste para SMB/ACL quando a fase exigir.
- Opcionalmente, dois DCs de homologacao; nunca usar credenciais ou controladores de producao.

## Plataforma preferida: Proxmox VE

1. Criar VM com UEFI/BIOS conforme padrao institucional, CPU/RAM suficientes e dois discos virtuais.
2. Instalar FreeBSD a partir de imagem verificada por checksum/assinatura.
3. Formatar o segundo disco como UFS2 de laboratorio e mantem-lo separado da raiz.
4. Criar checkpoint `baseline-clean` antes de instalar pacotes ou habilitar servicos.
5. Anexar somente VLAN de laboratorio e registrar MAC, snapshot e versao em ticket.

VirtualBox e bhyve sao opcoes para desenvolvimento; devem seguir a mesma separacao de disco e rede. O procedimento manual e permitido quando produz a mesma ficha de evidencias.

## Preparacao manual controlada

Instalar Go, SQLite, Argon2, Samba e CUPS somente depois de registrar a versao exata dos pacotes. Criar usuario administrativo restrito para acesso humano e usuario de servico somente pelo package. Nao habilitar o agente por padrao. Antes de qualquer escrita futura, restaurar o checkpoint limpo e repetir a coleta somente leitura.

## Uso dos scripts

- `scripts/run-integrated.sh`: consulta release e pacotes do FreeBSD, verifica uma API já provisionada e inicia apenas o frontend local. O script não sincroniza arquivos, compila, reinicia serviços nem altera o host remoto.
- `ops/freebsd-lab/check-readiness.sh`: apenas inspeciona prerequisitos e emite relatorio.
- `ops/freebsd-lab/collect-fixtures.sh`: invoca o binario Go `backend/bin/samba-admin-fixture-collector`, que executa a allowlist do adapter e cria um unico JSON novo com modo `0600`.
- `ops/freebsd-lab/prepare-lab.sh`: gera plano de comandos; recusa modo de aplicacao.
- `ops/freebsd-lab/smoke-package.sh`: altera apenas o produto `samba-admin` em VM descartavel; valida instalacao, rc.d, UDS, login, leitura, bloqueio de mutacao e desinstalacao. Exige snapshot e nunca altera Samba/CUPS/AD/ACL.

## Evidencias por rodada

Registrar versao do SO, pacote Samba/CUPS, topologia, snapshot inicial/final, hashes das fixtures, resultados de testes, diferencas observadas, riscos e decisao de liberar ou bloquear. Restaurar checkpoint apos testes destrutivos ou quando houver incerteza de estado.

## Estado da VM desta rodada

A VM `192.168.122.84` iniciou limpa em FreeBSD `15.1-RELEASE-p1`, amd64/UFS, ABI `FreeBSD:15:amd64`, sem residuos do produto. Foram instalados para homologacao Go `1.26.5`, SQLite `3.53.3`, Argon2, Samba `4.23.8_1`, CUPS `2.4.19_1`, Kerberos, LDAP e `curl`. Samba e CUPS permaneceram desabilitados e parados.

O package tecnico `samba-admin-0.3.5`, construido com `RELEASE=no`, esta instalado. API e agente executam via rc.d; a API escuta somente em `127.0.0.1:8080`, usa adapter `agent`, o agente usa executor `readonly-freebsd` e as mutacoes permanecem desabilitadas. Testes, race detector, vet e build nativos passaram; staticcheck nao estava disponivel na VM.

O package imutavel/root-only esta em `/root/samba-admin-packages/0.3.5-20260713/samba-admin-0.3.5.pkg`, SHA-256 `72b05ee3aef02a3cb59a1bffd4ffcba637ddc295b8d0f71fbdff96c4d43edd89`. O smoke aprovado esta em `/root/samba-admin-evidence/package-smoke-20260713T101715Z`; o relatorio final esta em `/root/samba-admin-evidence/rc1-final-20260713T101536Z`.

## Estado do incremento RC1

O smoke confirmou `GET /api/v1/samba` e `GET /api/v1/cups` via API → UDS → agente, 503 estruturado com correlation ID quando o agente foi parado, readiness 503 e recuperacao, mutacao bloqueada, deinstall ativo sem processos orfaos e reinstalacao. Banco e configuracao originais foram preservados; a credencial do laboratorio permaneceu root-only e nao foi reutilizada pelo smoke.

Samba e CUPS terminaram parados. O arquivo `/usr/local/etc/cups/cupsd.conf` manteve SHA-256 `30a33232846d21246f4a59d9f1bb6d8ec54b1ddd429575ff5fb5918f3b82cb5e`; as demais configuracoes observadas permaneceram ausentes. Nao houve alteracao de Samba, CUPS, AD, ACL ou filesystem.

Durante a rodada, um bug no procedimento de probe provocou rollback intermediario, sem falha do produto. O rollback `0.3.4` funcionou; depois da correcao do probe, o `0.3.5` foi reinstalado e as sondagens passaram.

A conta de laboratorio utiliza o banco `/var/db/samba-admin/homologacao-0.3.4.db`. A credencial recuperavel por root fica em `/root/samba-admin-evidence/rc0-access-0.3.4.txt`, modo `0600`; a senha bootstrap foi removida do arquivo de ambiente depois da criacao da conta.

O rollback reconstruido/testado foi preservado imutavel em `/root/samba-admin-rollback/0.3.4-20260713T100657Z/samba-admin-0.3.4.pkg`, SHA-256 `333ccfa10efa8bc8a533c0c9712d3cdff32dcd04116cbd160bd9f174245096c1`. Esse e o checksum do artefato reconstruido; o checksum historico do artefato original `0.3.4` permanece `53836c21ea9a3278583a40dac5cc50f8ba93162221cbe2296a6c61f2c4a8b39b`.

## Acesso

A API nao deve ser exposta diretamente. Use a configuracao SSH sem herdar a configuracao global e mantenha o tunel em loopback:

```sh
ssh -F /dev/null -N -L 8080:127.0.0.1:8080 root@192.168.122.84
```

A credencial permanece somente no arquivo root-only indicado acima. Nunca copie seu conteudo para terminal compartilhado, log, trace, screenshot, relatorio ou repositorio.

A VM possui apenas o disco raiz observado. Portanto ela nao atende ainda ao requisito de segundo volume UFS2 descartavel e nao autoriza homologacao de ACL, quota, mount option ou rollback de filesystem. O identificador do snapshot/checkpoint permanece evidencia externa do hipervisor e deve ser anexado ao ticket.

O host anterior `192.168.122.180` permanece fora desta rodada por integridade de baseline incerta; nao reutilizar suas evidencias como homologacao formal.
