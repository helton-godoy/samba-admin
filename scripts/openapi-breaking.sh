#!/bin/sh
set -eu

if [ -z "${OPENAPI_BASE_REF:-}" ]; then
  printf '%s\n' 'OPENAPI_BASE_REF não definido; comparação de breaking change ignorada fora de CI.'
  exit 0
fi

baseline="$(mktemp)"
trap 'rm -f "$baseline"' EXIT HUP INT TERM
git show "${OPENAPI_BASE_REF}:backend/api/openapi.yaml" > "$baseline"

# oasdiff exits non-zero when a breaking change is found. A release manager may
# explicitly acknowledge an approved compatibility exception in the CI job.
if [ "${OPENAPI_BREAKING_CHANGE_APPROVED:-false}" = "true" ]; then
  printf '%s\n' 'Breaking change aprovado explicitamente para esta release.'
  exit 0
fi

go run github.com/oasdiff/oasdiff@v1.11.7 breaking --fail-on ERR "$baseline" backend/api/openapi.yaml
