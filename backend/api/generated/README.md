# Código gerado a partir do OpenAPI

Este diretório é reservado aos tipos e contratos gerados por `oapi-codegen`.

O runtime deste incremento permanece compilável sem baixar módulos externos. Em um ambiente com acesso ao proxy de módulos Go, execute:

```bash
make generate-openapi
```

Após o congelamento do contrato, o próximo ciclo deverá substituir gradualmente os DTOs manuais pelos tipos gerados, mantendo testes de compatibilidade com o front-end.
