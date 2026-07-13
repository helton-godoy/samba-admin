# Matriz de Risco Atualizada

| Risco | Impacto | Estado | Controle/Gate |
| --- | --- | --- | --- |
| Escrita privilegiada indevida | Critico | Bloqueado | Agente separado, catalogo fechado, capability, approval e feature flag. |
| Perda de ACL | Critico | Bloqueado | Leitura/parser primeiro; backup/rollback e Windows 11 antes de escrita. |
| Idmap instavel | Alto | Nao verificado | Diagnostico de ranges e estrategia antes de ACL/quota/domain join. |
| Ingresso AD incorreto | Alto | Bloqueado | Diagnostico, credencial em memoria, aprovacao e laboratorio isolado. |
| Comprometimento UDS | Critico | Mitigado com restricoes | Peer credentials, HMAC, nonce persistente, timestamp, limites, diretorio root/setgid e socket `0660` exercitados no FreeBSD 15.1. |
| Vazamento de segredo | Alto | Mitigacao parcial | Redacao, fixtures estritas e sanitizador dos artefatos da tentativa E2E passaram; gitleaks nao foi executado nesta rodada. |
| Falha de rollback | Alto | Simulado | Exercitar restore real em laboratorio antes de mutacao. |
| Pacote/ABI incorreto | Alto | Mitigado no laboratorio | ABI `FreeBSD:15:amd64`, testes/race/vet/build nativos, checksum, smoke, deinstall ativo e reinstalacao foram exercitados no package tecnico `0.3.5`. Publicacao CI/SBOM permanece pendente. |
| Indisponibilidade SMB | Alto | Bloqueado para mutacao | Diff, teste, janela, health check e rollback. |
| Falha/volume SIEM | Medio | Nao verificado | RFC 5424/JSON, retencao e homologacao de transporte. |
| Falsa confianca em E2E | Alto | Nao aprovado no browser | Suites MSW e API Go real possuem 20 cenarios cada, mas `EPERM` bloqueou navegador/sockets antes de assertions. O backend Go/agente fixture UDS iniciou e os artefatos foram sanitizados; executar ambas fora do sandbox continua gate. |
| Integridade da VM de laboratorio | Alto | Mitigado nesta rodada | VM nova `192.168.122.84`, baseline independente e evidencias root-only. O host anterior `.180` permanece excluido da homologacao. |
| Readiness sem agente | Alto | Mitigado com restricoes | Em modo real, `/readyz` consulta o adapter via UDS; failover real confirmou 503 sem agente e 200 apos recuperacao. |
| Divergencia do contrato Samba/CUPS | Alto | Mitigado com restricoes | OpenAPI/gerados/testes passaram; o smoke `0.3.5` confirmou Samba/CUPS via agente e HTTP 503/correlation ID sem agente. E2E browser continua pendente. |
| Distribuicao insegura do front-end | Alto | Mitigacao parcial | Artefato local somente HTML/CSS/JS e checksum aprovados; template renderizado. Publicacao, `nginx -t`, certificado, dominio e ativacao institucional permanecem bloqueios. |
| Regressao do package RC1 | Alto | Mitigado no laboratorio | Package tecnico `0.3.5` passou smoke/deinstall/reinstall/readiness. Rollback reconstruido `0.3.4` funcionou; um bug do procedimento de probe causou rollback intermediario e foi corrigido. CI/distribuicao ainda nao verificadas. |
| Cobertura de dependencias e supply chain | Alto | Nao concluida nesta rodada | `npm audit` foi inconclusivo por `EAI_AGAIN`; gitleaks, govulncheck, SBOM e GitHub Actions nao foram executados. Bloquear distribuicao ate completar o gate. |
