# Runbook Operacional

## Inicio seguro

1. Confirmar versao do package, checksum, configuracao e dono/modo dos diretorios.
2. Para inventario real, habilitar `samba_admin_agent` com executor `readonly-freebsd` e depois `samba_admin_api` em modo `agent`; manter todas as feature flags mutaveis desabilitadas.
3. Confirmar que a API escuta apenas em loopback e que somente o proxy HTTPS institucional ou um tunel SSH oferece acesso.
4. Verificar `healthz`, `readyz`, logs estruturados, banco WAL, auditoria e capabilities.
5. Registrar baseline de inventario antes de qualquer mudanca.

## Baseline tecnico de laboratorio em 2026-07-13

A VM `192.168.122.84` executa o package tecnico `0.3.5`, SHA-256 `72b05ee3aef02a3cb59a1bffd4ffcba637ddc295b8d0f71fbdff96c4d43edd89`. API e agente estao ativos, a API escuta em `127.0.0.1:8080`, o adapter e `agent`, o executor e `readonly-freebsd` e `SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS=false`.

Samba e CUPS devem permanecer parados. A referencia observada de `/usr/local/etc/cups/cupsd.conf` e SHA-256 `30a33232846d21246f4a59d9f1bb6d8ec54b1ddd429575ff5fb5918f3b82cb5e`; as demais configuracoes Samba/CUPS observadas estavam ausentes. Qualquer divergencia exige interromper a rodada e investigar sem editar esses servicos.

O acesso de laboratorio deve usar tunel local e ignorar a configuracao SSH global:

```sh
ssh -F /dev/null -N -L 8080:127.0.0.1:8080 root@192.168.122.84
```

Nao exponha a porta 8080 na rede. A credencial de laboratorio permanece somente no arquivo root-only documentado em `docs/freebsd-lab.md`; nunca imprima ou copie seu conteudo para logs, traces, screenshots ou artefatos.

## Operacao diaria

- Monitorar falhas de login, uso break-glass, filas de jobs, locks, timeouts, rollback e perda de conexao SSE.
- Revisar alerts de capability ausente ou mudanca de versao de Samba/FreeBSD.
- Rotacionar logs e verificar espaco de banco sem apagar evidencias fora da politica de retencao.
- Conferir se o agente continua restrito ao catalogo somente leitura e se `SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS=false` permanece efetivo.

## Mudanca controlada

Validar RBAC, capability, idempotencia, ETag, lock, diff, impacto, janela, aprovacao e backup. Acompanhar job ate estado terminal, coletar correlation ID e validar health check. Se falhar, acionar rollback documentado; nao repetir comando manual arbitrario no host.

## Incidente

Em suspeita de comprometimento: bloquear acesso externo, preservar logs/auditoria, revogar sessoes/segredos conforme plano, desabilitar agente, coletar evidencias somente leitura e escalar para seguranca. Nao limpar banco, logs ou socket antes de preservar cadeia de custodia.

## Rollback do package tecnico

1. Preservar banco, configuracao e evidencias antes de remover o candidato.
2. Verificar o package reconstruido/root-only `/root/samba-admin-rollback/0.3.4-20260713T100657Z/samba-admin-0.3.4.pkg` contra SHA-256 `333ccfa10efa8bc8a533c0c9712d3cdff32dcd04116cbd160bd9f174245096c1`.
3. Parar API/agente, remover o candidato e instalar o rollback segundo o procedimento de package; restaurar somente a configuracao preservada, sem reutilizar credenciais efemeras do smoke.
4. Confirmar processos, loopback, executor `readonly-freebsd`, mutacoes desabilitadas, `healthz`, `readyz` e hashes/estado de Samba e CUPS.

O checksum historico do artefato original `0.3.4` e `53836c21ea9a3278583a40dac5cc50f8ba93162221cbe2296a6c61f2c4a8b39b`; ele nao identifica o package reconstruido acima. O rollback reconstruido foi exercitado com sucesso durante a rodada, apos um bug no procedimento de probe, e o `0.3.5` foi reinstalado com sondagens aprovadas.

As evidencias da rodada ficam em `/root/samba-admin-evidence/package-smoke-20260713T101715Z` e `/root/samba-admin-evidence/rc1-final-20260713T101536Z`. Esses diretorios sao root-only e nao devem ser publicados sem sanitizacao.

## Gate de distribuicao

O package `0.3.5` usa `RELEASE=no` e e aprovado somente para laboratorio. Nao distribuir nem promover a producao ate que Playwright MSW/real passem fora do sandbox, GitHub Actions, gitleaks, govulncheck e SBOM sejam executados, `npm audit` conclua com rede funcional e proxy/certificado/metadados institucionais sejam homologados.

## Escalonamento

Inclua correlation ID, job ID, usuario, recurso, horario, versao de package, capability, diff/hash, resultado, snapshot e acoes de rollback. Nunca inclua senha, cookie, chave HMAC, TOTP, keytab ou ticket Kerberos.
