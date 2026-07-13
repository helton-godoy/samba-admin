#!/bin/sh
set -eu
mkdir -p ./var
: "${SAMBA_ADMIN_AGENT_KEY:=development-only-change-me-32-bytes}"
export SAMBA_ADMIN_AGENT_KEY
export SAMBA_ADMIN_AUTH_MODE="${SAMBA_ADMIN_AUTH_MODE:-development-bypass}"
export SAMBA_ADMIN_ADAPTER_MODE=agent
go run ./cmd/samba-admin-agent &
AGENT_PID=$!
trap 'kill $AGENT_PID 2>/dev/null || true' EXIT INT TERM
sleep 1
exec go run ./cmd/samba-admin-api
