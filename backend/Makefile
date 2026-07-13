GO?=go
OAPI_CODEGEN?=$(GO) run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.7.0

.PHONY: test build build-tools check-migrations run run-agent dev generate-openapi clean

test: check-migrations
	GOTOOLCHAIN=local $(GO) test ./...

build: check-migrations
	mkdir -p bin
	GOTOOLCHAIN=local $(GO) build -trimpath -o bin/samba-admin-api ./cmd/samba-admin-api
	GOTOOLCHAIN=local $(GO) build -trimpath -o bin/samba-admin-agent ./cmd/samba-admin-agent

build-tools:
	mkdir -p bin
	GOTOOLCHAIN=local $(GO) build -trimpath -o bin/samba-admin-fixture-collector ./cmd/samba-admin-fixture-collector

check-migrations:
	@for migration in migrations/*.sql; do name=$${migration##*/}; cmp -s "$$migration" "internal/storage/migrations/$$name" || { echo "migracao divergente ou ausente: $$name" >&2; exit 1; }; done
	@for migration in internal/storage/migrations/*.sql; do name=$${migration##*/}; test -f "migrations/$$name" || { echo "migracao publica ausente: $$name" >&2; exit 1; }; done

run:
	./scripts/dev.sh

run-agent:
	./scripts/dev-with-agent.sh

generate-openapi:
	$(OAPI_CODEGEN) -config api/oapi-codegen.yaml api/openapi.yaml
	$(OAPI_CODEGEN) -config api/oapi-codegen-server.yaml api/openapi.yaml

clean:
	rm -rf bin var/*.db var/*.sock
