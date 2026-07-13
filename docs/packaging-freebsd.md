# Empacotamento FreeBSD

## Componentes

O package inicial instala os binarios backend `samba-admin-api` e `samba-admin-agent`, scripts rc.d, wrappers de ambiente, configuracao de exemplo e `newsyslog`. A API executa como usuario `sambaadmin`; o agente so inicia quando `samba_admin_agent_enable=YES` for definido explicitamente. O `dist` do front-end e produzido como artefato separado, preparado para publicacao apos os gates institucionais, e deve ser servido pelo proxy HTTPS; ele nao integra este package.

## Layout previsto

```text
/usr/local/sbin/samba-admin-api
/usr/local/sbin/samba-admin-agent
/usr/local/libexec/samba-admin-*-wrapper
/usr/local/etc/rc.d/samba_admin_api
/usr/local/etc/rc.d/samba_admin_agent
/usr/local/etc/samba-admin/{api.env.sample,agent.env.sample,shared.env.sample}
/usr/local/etc/newsyslog.conf.d/samba-admin.conf
/var/db/samba-admin
/var/db/samba-admin-agent
/var/log/samba-admin
/var/run/samba-admin
```

Configuracoes reais nao sao distribuidas pelo package. Arquivos de ambiente contem apenas valores seguros de exemplo e devem ser copiados pelo operador com dono `root`, grupo minimo necessario e sem permissao de escrita para o usuario de servico.

Mantenedor, URL do projeto e origem do package sao metadados obrigatorios de release. Valores `example.invalid` servem apenas para impedir publicacao acidental e nao constituem identidade institucional. Para um artefato distribuivel, use `RELEASE=yes` e forneca `PACKAGE_ORIGIN`, `PACKAGE_MAINTAINER` e `PACKAGE_WWW`; o build falha se receber os placeholders de laboratorio. O modo padrao produz somente package tecnico local, nao publicavel.

`shared.env` contem a chave HMAC e seu identificador; a rotacao usa chave anterior apenas durante janela documentada. `agent.env` define cache persistente de nonce, limites de mensagem/saida/concorrencia, owner/grupo/mode do socket e UID esperado do processo API. O executor padrao permanece `mock`; `readonly-freebsd` exige configuracao explicita de peer credentials e capabilities homologadas. A API acessa inventarios reais somente pelo UDS; o modo legado `freebsd-readonly` tambem usa o agente e nao executa comandos no processo sem privilegio.

## Build

Construir no FreeBSD homologado porque SQLite/Argon2 usam CGO. O fluxo usa `pkg create -m ... -p pkg-plist`, recusa staging preexistente, gera checksum SHA-256 e nao remove staging automaticamente. O workflow de CI esta configurado para gerar o SBOM CycloneDX do repositorio em job separado; sem uma execucao real do workflow, o SBOM nao deve ser declarado como publicado. Cross-compilation Linux para FreeBSD nao e evidencia de release.

```sh
cd backend
env GOTOOLCHAIN=local go126 test ./...
make GO=go126 build build-tools
make -C packaging/freebsd package
pkg info -F packaging/freebsd/packages/samba-admin-*.pkg
```

O baseline `0.3.4` foi construido no FreeBSD 15.1 com Go `1.26.5`. Em 2026-07-13, testes, race detector, vet e build nativos do `0.3.5` passaram no FreeBSD `15.1-RELEASE-p1`, ABI `FreeBSD:15:amd64`; staticcheck nao estava disponivel na VM. As diretivas CGO usam `/usr/local/include` e `/usr/local/lib` para SQLite/Argon2. Nao copiar binarios Linux para staging FreeBSD.

O build usou `RELEASE=no`, portanto produziu somente um package tecnico de laboratorio, nao distribuivel. O artefato imutavel/root-only esta em `/root/samba-admin-packages/0.3.5-20260713/samba-admin-0.3.5.pkg`, SHA-256 `72b05ee3aef02a3cb59a1bffd4ffcba637ddc295b8d0f71fbdff96c4d43edd89`.

## Instalacao e upgrade

1. Verificar assinatura/checksum e compatibilidade ABI.
2. Inspecionar manifesto e lista de arquivos; instalar o package somente em VM com snapshot.
3. O pre-install cria ou valida a conta `sambaadmin` com grupo dedicado, home `/nonexistent` e shell `/usr/sbin/nologin`; perfil preexistente divergente bloqueia a instalacao.
4. Criar configuracao a partir de amostra, inserir segredo por mecanismo institucional e validar permissoes.
5. Manter ambos desabilitados ate configurar HMAC, nonce, peer UID e capabilities.
6. Habilitar agente `readonly-freebsd` e API `agent` somente na rede administrativa; manter `SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS=false`.
7. Executar readiness e confirmar diretorio `root:sambaadmin:2750`, socket `root:sambaadmin:0660` e peer UID da API.

Upgrade requer backup do banco/configuracao, migracao testada, health check e plano de downgrade. Diretorios de dados, logs, configuracao efetiva e conta de servico sao criados por script e deliberadamente nao pertencem ao plist removivel; a desinstalacao os preserva para recuperacao e auditoria. O hook de pre-deinstall interrompe API e agente antes que `pkg` remova os executaveis, evitando processos orfaos. O rc.d usa o `procname` do binario filho para que `status` e `stop` sejam confiaveis mesmo com supervisao por `daemon(8)`.

## Smoke e evidencia

O script abaixo instala somente o produto, gera segredos efemeros, valida login/inventario/bloqueio de mutacao, para os servicos e desinstala o package. Execute apenas em VM descartavel com snapshot:

```sh
ops/freebsd-lab/smoke-package.sh backend/packaging/freebsd/packages/samba-admin-0.3.5.pkg
```

`--keep-installed` reinstala o package, testa a retomada/readiness e depois encerra os servicos e remove flags/configuracoes efemeras; a configuracao operacional deve ser restaurada em uma etapa separada. O smoke verifica identidade dos processos, ABI, bind loopback, socket, logs, remocao da senha bootstrap, inventarios Samba/CUPS, 503 com correlation ID sem agente, readiness/failover, mutacao bloqueada e desinstalacao com servicos ativos.

A rodada em `192.168.122.84` aprovou o package tecnico `0.3.5`; a evidencia esta em `/root/samba-admin-evidence/package-smoke-20260713T101715Z`. O hook de deinstall nao deixou processos orfaos, a reinstalacao/readiness passou e a configuracao operacional original foi restaurada. Samba/CUPS permaneceram parados e inalterados.

O rollback reconstruido/testado esta em `/root/samba-admin-rollback/0.3.4-20260713T100657Z/samba-admin-0.3.4.pkg`, SHA-256 `333ccfa10efa8bc8a533c0c9712d3cdff32dcd04116cbd160bd9f174245096c1`. O checksum historico do artefato original `0.3.4`, `53836c21ea9a3278583a40dac5cc50f8ba93162221cbe2296a6c61f2c4a8b39b`, continua preservado separadamente.

Durante a rodada, um bug no procedimento de probe provocou rollback intermediario, nao uma falha do produto. O rollback funcionou; depois da correcao do probe, o `0.3.5` foi reinstalado e as sondagens passaram. O workflow de CI deve reproduzir e publicar package, checksum, evidencia e SBOM antes da distribuicao; ele nao foi executado nesta rodada.

## Pendencias para release

- Executar o job GitHub Actions e arquivar package reproduzido, checksum, ABI, manifesto e SBOM.
- Executar gitleaks e govulncheck; `npm audit` foi inconclusivo por `EAI_AGAIN` nesta rodada.
- Homologar `ops/reverse-proxy/nginx-samba-admin.conf.template` com `nginx -t`, dominio e certificado institucionais aprovados; apenas o render parametrizado local passou.
- Substituir mantenedor/URL de exemplo e definir assinatura do pacote.
