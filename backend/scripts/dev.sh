#!/bin/sh
set -eu
mkdir -p ./var
export SAMBA_ADMIN_AUTH_MODE="${SAMBA_ADMIN_AUTH_MODE:-development-bypass}"
export SAMBA_ADMIN_ADAPTER_MODE="${SAMBA_ADMIN_ADAPTER_MODE:-mock}"
exec go run ./cmd/samba-admin-api
