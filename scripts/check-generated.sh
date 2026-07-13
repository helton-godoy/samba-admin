#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd -P)
tmp_root=$(mktemp -d "${TMPDIR:-/tmp}/samba-admin-generated.XXXXXX")

cleanup()
{
    rm -f -- "$tmp_root/models.gen.go" "$tmp_root/server.gen.go" "$tmp_root/openapi.ts" \
        "$tmp_root/models.yaml" "$tmp_root/server.yaml"
    rmdir "$tmp_root"
}
trap cleanup EXIT HUP INT TERM

run_oapi_codegen()
{
    if [ -n "${OAPI_CODEGEN:-}" ]; then
        "$OAPI_CODEGEN" "$@"
        return
    fi
    go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.7.0 "$@"
}

cd "$repo_root/backend"
sed "s|^output:.*|output: $tmp_root/models.gen.go|" \
    api/oapi-codegen.yaml > "$tmp_root/models.yaml"
sed "s|^output:.*|output: $tmp_root/server.gen.go|" \
    api/oapi-codegen-server.yaml > "$tmp_root/server.yaml"
run_oapi_codegen -config "$tmp_root/models.yaml" api/openapi.yaml
run_oapi_codegen -config "$tmp_root/server.yaml" api/openapi.yaml

"$repo_root/frontend/node_modules/.bin/openapi-typescript" \
    "$repo_root/backend/api/openapi.yaml" -o "$tmp_root/openapi.ts"

status=0
for generated in models.gen.go server.gen.go; do
    if ! cmp -s "$repo_root/backend/api/generated/$generated" "$tmp_root/$generated"; then
        printf '%s\n' "Contrato Go desatualizado: backend/api/generated/$generated" >&2
        status=1
    fi
done
if ! cmp -s "$repo_root/frontend/src/api/generated/openapi.ts" "$tmp_root/openapi.ts"; then
    printf '%s\n' 'Contrato TypeScript desatualizado: frontend/src/api/generated/openapi.ts' >&2
    status=1
fi

exit "$status"
