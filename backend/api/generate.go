// Package api contains OpenAPI generation directives. Runtime handlers remain
// in internal/httpapi and are verified against the generated contract in CI.
package api

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.7.0 -config oapi-codegen.yaml openapi.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.7.0 -config oapi-codegen-server.yaml openapi.yaml
