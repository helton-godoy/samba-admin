# Samba Console para FreeBSD 15.1 — protótipo web

Protótipo funcional de front-end para administração de Samba em **FreeBSD 15.1-RELEASE**, com foco em **UFS2**, **SMB 3**, clientes **Windows 11**, ACLs NFSv4, integração opcional com Active Directory, CUPS, DFS Namespace, quotas, auditoria e encaminhamento de logs.

> Este projeto é exclusivamente um protótipo. Todos os dados e operações são simulados por Mock Service Worker. O navegador não executa comandos de shell e não recebe privilégios do sistema operacional.

## 1. Premissas e decisões técnicas

- O protocolo de acesso de clientes é SMB 3. “NFSv4” refere-se somente ao modelo de ACL armazenado no UFS2.
- ACL POSIX.1e e ACL NFSv4 são tratadas como propriedades mutuamente exclusivas do ponto de montagem UFS2.
- O modelo de ACL não pode ser selecionado por pasta dentro do mesmo sistema de arquivos.
- Uma conversão de ACL é tratada como migração semântica com inventário, relatório de perdas, manutenção, backup e rollback; nunca como conversão trivial.
- Os perfis standalone, membro de domínio, AD DC e DC adicional são mutuamente exclusivos na interface.
- AD DC permanece bloqueado até validação da versão, opções de compilação, dependências, Kerberos, DNS, LDB, ACL e ferramentas de backup/restore do pacote efetivamente instalado.
- SMB1 e acesso guest não são oferecidos nos fluxos padrão.
- O editor de arquivos utiliza allowlist e não expõe segredos.
- O backend futuro deverá receber operações estruturadas e usar adaptadores predefinidos; não deverá concatenar strings para formar shell arbitrário.

## 2. Riscos e limitações

- O protótipo não prova que uma compilação específica do Samba no FreeBSD fornece AD DC completo. A detecção deve ocorrer no host alvo.
- ACL NFSv4, ACL NT do Windows e ACL POSIX não possuem equivalência perfeita.
- Módulos VFS, parâmetros do `smb.conf` e nomes de eventos do `vfs_full_audit` variam conforme a versão do Samba.
- Shadow Copies em UFS2 dependem de uma estratégia de snapshots tecnicamente válida e testada; o recurso permanece “ainda não verificado”.
- DFS Namespace não replica dados. DFS-R não é presumido.
- Instalação de driver de impressão no Windows 11 depende de assinatura, pacote, arquitetura, Point and Print, GPO e elevação.
- Syslog com TLS pode exigir `syslog-ng`, `rsyslog` ou agente adicional.
- Quotas para contas de domínio dependem de SID ↔ UID/GID estável.
- A API atual é mockada e não implementa autenticação real, persistência, concorrência distribuída ou execução privilegiada.

## 3. Matriz resumida de compatibilidade

| Recurso | Estado no protótipo | Observação |
|---|---|---|
| SMB 3 / Windows 11 | Suportado | SMB1 e guest bloqueados por padrão. |
| UFS2 com ACL NFSv4 | Suportado | Detectado por ponto de montagem. |
| UFS2 com ACL POSIX.1e | Suportado | Mutuamente exclusivo com NFSv4. |
| Conversão POSIX ↔ NFSv4 | Suportado com restrições | Somente simulação e relatório de perdas. |
| Membro de Active Directory | Suportado | Fluxo separado de AD DC. |
| Samba AD DC | Ainda não verificado | Depende do pacote e da compilação reais. |
| DC adicional | Ainda não verificado | Requer testes de DRS, DNS, SYSVOL e backup. |
| CUPS + compartilhamento Samba | Suportado com restrições | Drivers Windows dependem de políticas de segurança. |
| DFS Namespace | Suportado com restrições | Sem promessa de replicação DFS-R. |
| Quotas UFS | Suportado | Mapeamento de identidade deve ser estável. |
| `vfs_full_audit` | Ainda não verificado | Sintaxe e eventos devem ser detectados na versão instalada. |
| Syslog remoto TLS | Suportado com restrições | Pode depender de pacote adicional. |
| Shadow Copies em UFS2 | Ainda não verificado | Exige implementação de snapshots comprovada. |

## 4. Arquitetura

```text
Navegador React/TypeScript
        │ HTTPS + JSON
        ▼
API REST /api/v1
  - autenticação e MFA
  - RBAC e segregação de funções
  - validação de entrada
  - CSRF, rate limit e auditoria
        │
        ▼
Orquestrador de tarefas
  - lock e controle de concorrência
  - inventário e avaliação de impacto
  - geração de configuração
  - diff, backup, aplicação e health check
  - rollback e correlação
        │
        ▼
Adaptadores privilegiados
  - Samba/testparm/net/wbinfo/smbclient
  - mount/fstab/getfacl/setfacl/quotas UFS
  - CUPS
  - syslog/log rotation
  - serviço e rc.d do FreeBSD
```

O frontend não conhece comandos concretos. Exemplo: envia `POST /api/v1/services/smbd/action` com `{ "action": "reload" }`; o backend autoriza, seleciona o adaptador, executa a operação permitida, captura saída, audita e verifica saúde.

## 5. Mapa de telas

- Painel
- Sistemas de arquivos e assistente de ACL
- Compartilhamentos e criação com diff
- Permissões e ACLs
- Gerenciador de arquivos
- Editor de configuração
- Active Directory — membro
- Samba AD DC — módulo separado
- Impressão e drivers
- DFS Namespace
- Cotas UFS
- Logs e SIEM
- Auditoria
- Serviços
- Central de tarefas
- Arquitetura e API

## 6. Estrutura do projeto

```text
src/
  api/client.ts                 cliente REST tipado
  components/                  layout e componentes reutilizáveis
  mocks/data.ts                dados simulados
  mocks/handlers.ts            API simulada com MSW
  pages/                       telas do console
  test/                        testes de fluxos principais
  types.ts                     contratos TypeScript
examples/
  smb4.conf.example
  acl-conversion-report.json
  share.diff
```

## 7. Instalação

Requisitos de desenvolvimento:

- Node.js 22 ou superior
- npm 10 ou superior

```sh
npm install
npm run dev
```

Acesse o endereço informado pelo Vite, normalmente `http://localhost:5173`.

## 8. Compilação e testes

```sh
npm run test
npm run build
npm run preview
```

Os testes cobrem:

- carregamento do painel e matriz de viabilidade;
- bloqueio de caminho não autorizado;
- geração de prévia do `smb4.conf`;
- simulação de conversão de ACL com perdas;
- separação entre membro do domínio e AD DC.

## 9. API futura — resumo

### Leitura

- `GET /api/v1/system`
- `GET /api/v1/filesystems`
- `GET /api/v1/shares`
- `GET /api/v1/acl`
- `GET /api/v1/principals`
- `GET /api/v1/domain`
- `GET /api/v1/printers`
- `GET /api/v1/print-drivers`
- `GET /api/v1/dfs`
- `GET /api/v1/quotas`
- `GET /api/v1/audit`
- `GET /api/v1/services`
- `GET /api/v1/jobs`

### Alterações controladas

- `POST /api/v1/shares/preview`
- `POST /api/v1/shares`
- `POST /api/v1/acl/effective`
- `POST /api/v1/acl/conversion/simulate`
- `POST /api/v1/domain/test`
- `POST /api/v1/configuration/validate`
- `POST /api/v1/services/{id}/action`
- `POST /api/v1/jobs/{id}/rollback`

Operações mutáveis devem aceitar chave de idempotência, versão esperada do recurso, justificativa, janela de mudança e identificador de aprovação quando a política exigir.

## 10. Fluxo obrigatório de aplicação

1. edição;
2. validação local;
3. validação técnica no backend;
4. geração determinística da configuração;
5. diff;
6. avaliação de impacto;
7. lock e backup;
8. confirmação/autorização;
9. aplicação atômica;
10. teste técnico;
11. recarga ou reinicialização;
12. health check;
13. auditoria e correlação;
14. rollback em falha.

## 11. Funcionalidades dependentes do backend

- autenticação local de emergência, AD, OIDC e MFA;
- RBAC real e segregação de funções;
- inspeção de pacotes, opções de compilação e módulos VFS;
- leitura e alteração de montagens UFS2;
- exportação e reaplicação de ACLs;
- geração e validação real de `smb4.conf`;
- ingresso/saída de domínio;
- provisionamento e backup de AD DC;
- administração CUPS e drivers;
- quotas UFS;
- syslog, TLS, certificados e rotação;
- controle de serviços via rc.d;
- locking, filas persistentes, cancelamento e rollback;
- armazenamento imutável de auditoria.

## 12. Recomendações para implantação segura

- Executar o backend como serviço dedicado, sem root permanente; usar privilégio mínimo por adaptador.
- Expor a API somente em rede administrativa segregada e HTTPS com certificado institucional.
- Usar autenticação forte, MFA, RBAC e sessão curta para operações privilegiadas.
- Manter conta local de emergência protegida, monitorada e testada.
- Utilizar allowlists de caminho, arquivo, serviço e operação.
- Nunca repassar entrada do usuário a shell genérico.
- Usar gravação atômica, `fsync`, ownership/mode esperados e validação pós-gravação.
- Registrar quem, quando, origem, intenção, diff, resultado, correlação e rollback, sem segredos.
- Testar restore de configuração, ACL e AD DC em ambiente isolado.
- Implantar inicialmente como ferramenta somente leitura e liberar operações por módulos após homologação.

## 13. Referências técnicas consideradas

- Manual `mount(8)` do FreeBSD: opções `acls` e `nfsv4acls` mutuamente exclusivas.
- FreeBSD Handbook: suporte a ACL NFSv4 em UFS e OpenZFS.
- Documentação Samba: separação de serviços de membro de domínio e AD DC.
- Árvore de Ports do FreeBSD: disponibilidade e evolução dos ports Samba devem ser confirmadas no repositório e pacote usados pelo host.

## 14. Próximos incrementos

1. Implementar backend somente leitura para inventário real do FreeBSD e Samba.
2. Adicionar autenticação OIDC/AD, MFA, RBAC e trilha de auditoria imutável.
3. Implementar mecanismo de capability discovery por versão, pacote, opções de build e módulos VFS.
4. Homologar geração determinística de `smb4.conf` e validação real com `testparm`.
5. Implementar executor privilegiado com allowlist, sem shell genérico, e tarefas persistentes.
6. Homologar exportação/reaplicação de ACLs em cópia de dados representativa.
7. Integrar CUPS e validar Point and Print com as GPOs institucionais.
8. Integrar SIEM, estimativa de volume e política de retenção.
9. Executar testes end-to-end com Playwright em desktop e tablet.
10. Implantar piloto somente leitura antes de liberar alterações.

## Integração com o backend Go

O front-end agora permite alternar entre MSW e a API real:

```bash
cp .env.example .env
# VITE_USE_MSW=false
npm run dev
```

O Vite encaminha `/api`, `/healthz` e `/readyz` para `http://localhost:8080`. As requisições usam cookies de sessão e aceitam token CSRF armazenado em `sessionStorage` na chave `samba-admin-csrf`.
