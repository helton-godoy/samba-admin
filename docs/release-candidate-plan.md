# Plano do RC1 Somente Leitura

## Escopo

O RC1 pretende distribuir o console no modo inventario/diagnostico sem liberar alteracao de configuracao. O package tecnico `0.3.5` foi validado no laboratorio; o `0.3.4` permanece como baseline/rollback. A distribuicao continua bloqueada pelos gates browser, CI/scanners/SBOM e TLS institucional.

## Checklist de entrada

| Gate | Estado em 2026-07-13 |
| --- | --- |
| Contrato OpenAPI, cliente e servidor gerados e validados | Aprovado: os cinco comandos obrigatorios passaram localmente na ordem definida. |
| Login/MFA/RBAC e approval cobertos por testes | Testes Go e 25 Vitest passaram; Playwright MSW e real nao foram aprovados porque o sandbox falhou com `EPERM` antes de assertions. |
| Inventarios Samba/CUPS exclusivamente via agente | Aprovado no package tecnico `0.3.5`: API → UDS → agente, 503 estruturado/correlation ID e readiness sem agente exercitados. |
| Agente com socket/segredo revisado e desabilitado por padrao | Aprovado com restricoes no FreeBSD; executor final `readonly-freebsd`, mutacoes desabilitadas. |
| `frontend-dist` e reverse proxy HTTPS | Artefato local HTML/CSS/JS e render do template aprovados; publicacao, `nginx -t`, certificado, dominio e ativacao institucional pendentes. |
| Package construido em FreeBSD e SBOM/checksum publicados | Package tecnico `0.3.5`, checksum, smoke/deinstall/reinstall aprovados. GitHub Actions e SBOM nao foram executados/publicados. |
| Fixtures sanitizadas e adapters de leitura homologados | Fixture FreeBSD 15.1/Samba 4.23 incorporada; AD, ACL e segundo disco ainda pendentes. |
| Diagnostico member server em AD de teste | Pendente; o host consultado esta standalone e sem winbind funcional. |
| Parser ACL NFSv4 e teste Windows 11 | Pendente; bloqueia qualquer escrita ACL. |
| Runbook, DR, SIEM e plano de seguranca revisados pela operacao | Documentos preparados; revisao institucional pendente. |

A decisao atual e **no-go para producao e no-go para promover o RC1 distribuivel**. O `0.3.5` e aprovado apenas como package tecnico de laboratorio somente leitura. O `0.3.4` reconstruido/testado permanece como rollback, sem substituir o checksum historico do artefato original. Nenhuma feature flag de mutacao deve ser habilitada.

## Implantacao

1. Preservar o package `0.3.4` e o snapshot da VM antes de instalar qualquer candidato RC1 — concluido para a rodada tecnica.
2. Configurar API para modo somente leitura e rede administrativa restrita — concluido; API em loopback, adapter `agent`, executor `readonly-freebsd`.
3. Executar smoke, inventario, readiness, deinstall ativo e reinstalacao — concluido para o `0.3.5`.
4. Executar Playwright fora do sandbox restrito, GitHub Actions, gitleaks, govulncheck e SBOM — pendente.
5. Homologar metadados, assinatura, dominio, certificado e `nginx -t` institucionais — pendente.
6. Registrar nova decisao go/no-go com os riscos residuais depois dos gates pendentes.

## Reversao

Desabilitar servicos, revogar sessoes/segredos se necessario, remover o candidato e reinstalar `/root/samba-admin-rollback/0.3.4-20260713T100657Z/samba-admin-0.3.4.pkg`, SHA-256 `333ccfa10efa8bc8a533c0c9712d3cdff32dcd04116cbd160bd9f174245096c1`, retornando ao banco/configuracao preservados. O checksum historico do artefato original `0.3.4` e `53836c21ea9a3278583a40dac5cc50f8ba93162221cbe2296a6c61f2c4a8b39b`; nao confundir os dois artefatos. Nenhum job mutavel e retomado automaticamente apos rollback.
