# Ferramentas do Laboratorio FreeBSD

As ferramentas de inventario sao intencionalmente conservadoras. O smoke de package altera apenas o produto `samba-admin` e `rc.conf` numa VM descartavel; nenhum script altera Samba/CUPS, executa `net ads join`, muda ACL ou edita `fstab`.

| Script | Efeito |
| --- | --- |
| `check-readiness.sh` | Inspeciona prerequisitos de um host FreeBSD; somente leitura. |
| `prepare-lab.sh` | Imprime um plano manual de laboratorio e recusa `--apply`. |
| `collect-fixtures.sh` | Invoca o coletor Go allowlisted e grava um JSON novo, modo `0600`, estritamente sanitizado. |
| `smoke-package.sh` | Instala/configura/testa/para/desinstala o package; exige root, FreeBSD, snapshot e `curl`. `--keep-installed` reinstala e testa readiness, mas encerra os servicos e remove a configuracao efemera ao final; a configuracao operacional deve ser restaurada separadamente. |

Compile e execute a coleta no FreeBSD a partir de checkout controlado; o arquivo de saida deve ser novo:

```sh
make -C backend build-tools
./ops/freebsd-lab/check-readiness.sh
./ops/freebsd-lab/collect-fixtures.sh -output ops/freebsd-lab/fixtures/freebsd-20260712.json
```

Revise as fixtures antes de commitar. A sanitizacao reduz exposicao, mas nao substitui revisao humana nem classificacao institucional de dados.

A fixture homologada desta rodada esta em `backend/testdata/fixtures/freebsd-15.1-samba423-sanitized.json` e e exercitada pelos testes do adapter fixture.

## Rodada tecnica de 2026-07-13

O smoke do package tecnico `0.3.5` foi aprovado na VM `192.168.122.84`. A evidencia root-only esta em `/root/samba-admin-evidence/package-smoke-20260713T101715Z`; o relatorio final esta em `/root/samba-admin-evidence/rc1-final-20260713T101536Z`. Foram exercitados Samba/CUPS somente leitura, 503 com correlation ID sem agente, readiness 503/recuperacao, mutacao bloqueada, deinstall ativo sem processos orfaos e reinstalacao.

O package imutavel esta em `/root/samba-admin-packages/0.3.5-20260713/samba-admin-0.3.5.pkg`, SHA-256 `72b05ee3aef02a3cb59a1bffd4ffcba637ddc295b8d0f71fbdff96c4d43edd89`. O rollback reconstruido/testado `0.3.4` esta em `/root/samba-admin-rollback/0.3.4-20260713T100657Z/samba-admin-0.3.4.pkg`, SHA-256 `333ccfa10efa8bc8a533c0c9712d3cdff32dcd04116cbd160bd9f174245096c1`; o checksum historico do artefato original permanece separado.

Ao final, a configuracao operacional original foi restaurada, API/agente ficaram ativos em loopback, executor `readonly-freebsd`, mutacoes desabilitadas e Samba/CUPS parados. O package foi construido com `RELEASE=no` e esta aprovado apenas para laboratorio, nao para distribuicao ou producao.
