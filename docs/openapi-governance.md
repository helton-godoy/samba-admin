# Governanca OpenAPI

## Fonte de verdade

`backend/api/openapi.yaml` e a fonte normativa para rotas publicas, schemas, erros, enumeracoes, paginação, filtros, tarefas e capabilities. Tipos manuais, mocks e telas devem convergir para esse contrato; eles nao podem criar campos ou semantica publica propria.

## Artefatos gerados

Os artefatos Go ficam em `backend/api/generated/`; os tipos TypeScript ficam em `frontend/src/api/generated/openapi.ts`. O transporte tipado e implementado por `createOpenApiClient`, sem duplicar schemas. Arquivos gerados sao versionados no repositorio e nao recebem edicao manual.

O pipeline deve executar, na ordem:

```text
make openapi-validate
make generate
make check-generated
```

O workflow tambem executa `git diff --exit-code` depois da geracao, pois gerar antes de comparar poderia mascarar artefato desatualizado no commit. O build do servidor possui verificacao de interface contra o handler gerado.

Por padrao, o backend fixa `oapi-codegen v2.7.0` via `go run`. Builders offline homologados podem definir `OAPI_CODEGEN` com o caminho absoluto de um binario dessa mesma versao, previamente verificado; `make generate` e `make check-generated` preservam os mesmos argumentos e comparacoes.

`oapi-codegen v2.7.0` ainda avisa que o suporte OpenAPI 3.1 e incompleto. Redocly valida a especificacao 3.1 e os testes/builds validam os artefatos, mas uma atualizacao do gerador ou migracao para ferramenta com suporte 3.1 integral permanece risco acompanhado.

## Compatibilidade

- Rotas canonicas: `/api/v1/acls` e `/api/v1/identities`.
- Aliases legados: `/api/v1/acl` e `/api/v1/principals` permanecem por janela publicada, marcados `deprecated: true` no OpenAPI e com headers `Deprecation` e `Sunset` na resposta.
- Uma remocao exige nota de release, periodo de migracao, telemetria de uso e aprovacao de arquitetura.
- Alterar tipo, obrigatoriedade, enum, semantica de erro ou comportamento de idempotencia e uma breaking change, salvo prova documentada em contrario.

## Revisao de mudancas

Todo pull request que alterar OpenAPI deve conter:

1. diff de contrato e classificacao compatível/incompativel;
2. atualizacao de cliente, servidor, mocks e testes;
3. exemplo de request/response para endpoint novo ou alterado;
4. politica de deprecacao se houver substituicao;
5. decisao registrada para breaking change.

O CI valida sintaxe, lint, geracao limpa e compara a especificacao contra a base do pull request. Uma alteracao incompatível sem aprovacao explicitamente documentada bloqueia merge.

## Convencoes obrigatorias

- Erros usam schema unico com codigo estavel, mensagem segura, detalhes validaveis e `correlationId`.
- Mutacoes declaram `Idempotency-Key`, versao/ETag quando aplicavel e identificador de tarefa quando assincronas.
- Capabilities retornam estado, justificativa, pre-requisitos e escopo; ausencia de capability nao pode ser interpretada como suporte.
- Campos secretos nunca sao retornados nem incluidos em exemplos.
