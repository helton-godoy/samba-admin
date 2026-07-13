.DEFAULT_GOAL := help

.PHONY: help generate generate-go generate-ts openapi-validate openapi-breaking check-generated test build frontend-dist

help:
	@printf '%s\n' 'Targets: generate, openapi-validate, openapi-breaking, check-generated, test, build, frontend-dist'

generate: generate-go generate-ts

generate-go:
	$(MAKE) -C backend generate-openapi

generate-ts:
	npm --prefix frontend run generate:openapi

openapi-validate:
	npm --prefix frontend run openapi:validate

# OPENAPI_BASE_REF must identify a ref containing backend/api/openapi.yaml.
# CI supplies the merge-base; a standalone checkout has no baseline to compare.
openapi-breaking:
	./scripts/openapi-breaking.sh

check-generated: openapi-validate
	./scripts/check-generated.sh

test:
	$(MAKE) -C backend test
	npm --prefix frontend test

build: check-generated
	$(MAKE) -C backend build
	npm --prefix frontend run build

frontend-dist:
	npm --prefix frontend run build
	./scripts/package-frontend-dist.sh
