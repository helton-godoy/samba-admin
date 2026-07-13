#!/bin/sh
# Invoca exclusivamente o coletor Go, que possui allowlist e sanitizacao testadas.
set -eu

script_dir=$(CDPATH= cd -P "$(dirname "$0")" && pwd)
repository_root=$(CDPATH= cd -P "$script_dir/../.." && pwd)
collector="$repository_root/backend/bin/samba-admin-fixture-collector"

if [ "${1:-}" = "--help" ]; then
  printf '%s\n' 'Uso: collect-fixtures.sh -output ARQUIVO.json [opcoes do coletor]'
  printf '%s\n' 'O arquivo deve ser novo; o coletor recusa execucao fora do FreeBSD.'
  exit 0
fi

if [ ! -x "$collector" ]; then
  printf '%s\n' 'erro: coletor ausente; execute make -C backend build-tools no FreeBSD' >&2
  exit 69
fi

exec "$collector" "$@"
