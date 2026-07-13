# Fixtures Sanitizadas

Este diretorio recebe somente fixtures geradas por `../collect-fixtures.sh`. Nao armazene saida bruta, keytabs, tickets Kerberos, senhas, chaves privadas, cookies, tokens ou configuracao institucional secreta.

Cada coleta possui `manifest.json` e arquivos JSON por comando. O manifest registra versao do schema, horario, politica de redacao e hash de cada registro. Antes de versionar, um revisor deve procurar identificadores institucionais residuais e confirmar que a fixture representa um ambiente de laboratorio.
