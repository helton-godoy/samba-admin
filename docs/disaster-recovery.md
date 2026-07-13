# Recuperacao de Desastre

## Objetivos

Restaurar console, configuracao e trilha de auditoria sem introduzir escrita nao verificada em Samba, ACL ou Active Directory. Recuperacao do console nao e autorizacao para alterar o servidor de arquivos.

## Itens de backup

- Banco SQLite consistente, incluindo WAL/SHM quando aplicavel.
- Configuracoes de API/agente sem segredos em texto claro; segredos sao recuperados pelo cofre institucional.
- Versao do package, manifesto, checksum, SBOM e configuracao rc.d.
- Auditoria encadeada e exportacao para destino remoto/imutavel quando disponibilizado.
- Evidencias de laboratorio, fixtures sanitizadas e documentacao de capability.

## Procedimento de restauracao

1. Declarar incidente, congelar mudancas e preservar evidencias.
2. Restaurar em host isolado ou snapshot limpo com mesma ABI FreeBSD quando possivel.
3. Verificar checksums, ownership, permissoes, integridade SQLite e cadeia de auditoria.
4. Recuperar segredos pelo fluxo institucional; nunca copiar de log, fixture ou ticket.
5. Iniciar API somente leitura e manter agente desabilitado.
6. Executar health/readiness, comparar inventario e aprovar retorno gradual.

## Testes

Testar restore periodicamente em laboratorio e registrar RTO/RPO observados. Toda restauracao deve provar que jobs pendentes, locks, sessoes e aprovacoes sao tratados de forma segura, sem executar automaticamente tarefas antigas.

## Limites

Restauracao de AD DC, SYSVOL, replicacao ou ACL em producao esta fora deste ciclo e requer runbook especializado, backup homologado e equipe responsavel pelo diretorio.
