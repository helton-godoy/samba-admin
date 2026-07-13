#!/bin/sh

set -eu
umask 077

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
GO=${GO:-go}
FIXTURE_PATH=${SAMBA_ADMIN_E2E_FIXTURE_PATH:-"$BACKEND_DIR/testdata/fixtures/freebsd-15.1-samba423-sanitized.json"}
FRONTEND_PORT=${SAMBA_ADMIN_E2E_FRONTEND_PORT:-4174}
API_PORT=${SAMBA_ADMIN_E2E_API_PORT:-18080}
UNAVAILABLE_PORT=${SAMBA_ADMIN_E2E_UNAVAILABLE_PORT:-18081}
EXPIRY_PORT=${SAMBA_ADMIN_E2E_EXPIRY_PORT:-18082}
MFA_PORT=${SAMBA_ADMIN_E2E_MFA_PORT:-18083}
BACKEND_ONLY=${SAMBA_ADMIN_E2E_BACKEND_ONLY:-false}
FRONTEND_DIST=${SAMBA_ADMIN_E2E_FRONTEND_DIST:-}

: "${SAMBA_ADMIN_E2E_PASSWORD:?Defina SAMBA_ADMIN_E2E_PASSWORD para a suíte integrada.}"
: "${SAMBA_ADMIN_E2E_TOTP_SECRET:?Defina SAMBA_ADMIN_E2E_TOTP_SECRET em base32 para a suíte integrada.}"

if [ "${#SAMBA_ADMIN_E2E_PASSWORD}" -lt 14 ]; then
	printf '%s\n' 'erro: a senha E2E precisa ter ao menos 14 caracteres' >&2
	exit 2
fi
if [ ! -f "$FIXTURE_PATH" ]; then
	printf '%s\n' 'erro: fixture sanitizada não encontrada' >&2
	exit 2
fi

RUNTIME_DIR=$(mktemp -d "${TMPDIR:-/tmp}/samba-admin-e2e.XXXXXX")
BIN_DIR="$RUNTIME_DIR/bin"
LOG_DIR="$RUNTIME_DIR/logs"
mkdir -p "$BIN_DIR" "$LOG_DIR"
PIDS=''

cleanup()
{
	trap - EXIT INT TERM HUP
	for pid in $PIDS; do
		kill "$pid" 2>/dev/null || true
	done
	for pid in $PIDS; do
		wait "$pid" 2>/dev/null || true
	done
	if [ "${SAMBA_ADMIN_E2E_KEEP_RUNTIME:-false}" = 'true' ]; then
		printf '%s\n' "runtime E2E preservado em $RUNTIME_DIR" >&2
	else
		rm -rf "$RUNTIME_DIR"
	fi
}
trap cleanup EXIT INT TERM HUP

random_base64()
{
	dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 | tr -d '\n'
}

AGENT_KEY=$(random_base64)
TOTP_KEY=$(random_base64)
AGENT_SOCKET="$RUNTIME_DIR/agent.sock"

(cd "$BACKEND_DIR" && env GOTOOLCHAIN=local "$GO" build -o "$BIN_DIR/samba-admin-api" ./cmd/samba-admin-api)
(cd "$BACKEND_DIR" && env GOTOOLCHAIN=local "$GO" build -o "$BIN_DIR/samba-admin-agent" ./cmd/samba-admin-agent)
(cd "$BACKEND_DIR" && env GOTOOLCHAIN=local "$GO" build -o "$BIN_DIR/samba-admin-e2e-mfa-seed" ./cmd/samba-admin-e2e-mfa-seed)
if [ -n "$FRONTEND_DIST" ]; then
	(cd "$BACKEND_DIR" && env GOTOOLCHAIN=local "$GO" build -o "$BIN_DIR/samba-admin-e2e-web" ./cmd/samba-admin-e2e-web)
fi

env \
	SAMBA_ADMIN_DATA_DIR="$RUNTIME_DIR/agent-data" \
	SAMBA_ADMIN_AGENT_SOCKET="$AGENT_SOCKET" \
	SAMBA_ADMIN_AGENT_SOCKET_MODE=0600 \
	SAMBA_ADMIN_AGENT_KEY="$AGENT_KEY" \
	SAMBA_ADMIN_AGENT_NONCE_STORE="$RUNTIME_DIR/agent-nonces.json" \
	SAMBA_ADMIN_AGENT_EXECUTOR=fixture \
	SAMBA_ADMIN_ADAPTER_MODE=agent \
	SAMBA_ADMIN_FIXTURE_PATH="$FIXTURE_PATH" \
	SAMBA_ADMIN_AUTH_MODE=local \
	SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS=false \
	"$BIN_DIR/samba-admin-agent" >"$LOG_DIR/agent.log" 2>&1 &
PIDS="$PIDS $!"

i=0
while [ ! -S "$AGENT_SOCKET" ]; do
	i=$((i + 1))
	if [ "$i" -ge 100 ]; then
		printf '%s\n' 'erro: agente E2E não criou o socket UDS' >&2
		exit 1
	fi
	sleep 0.1
done

start_api()
{
	name=$1
	port=$2
	adapter=$3
	socket=$4
	database=$5
	session_ttl=$6
	executor=$7
	mfa_roles=$8
	env \
		SAMBA_ADMIN_HTTP_ADDR="127.0.0.1:$port" \
		SAMBA_ADMIN_DATA_DIR="$RUNTIME_DIR/$name-data" \
		SAMBA_ADMIN_DATABASE="$database" \
		SAMBA_ADMIN_AGENT_SOCKET="$socket" \
		SAMBA_ADMIN_AGENT_KEY="$AGENT_KEY" \
		SAMBA_ADMIN_AGENT_EXECUTOR="$executor" \
		SAMBA_ADMIN_ADAPTER_MODE="$adapter" \
		SAMBA_ADMIN_FIXTURE_PATH="$FIXTURE_PATH" \
		SAMBA_ADMIN_AUTH_MODE=local \
		SAMBA_ADMIN_BOOTSTRAP_USER=admin-e2e \
		SAMBA_ADMIN_BOOTSTRAP_PASSWORD="$SAMBA_ADMIN_E2E_PASSWORD" \
		SAMBA_ADMIN_TOTP_ENCRYPTION_KEY="$TOTP_KEY" \
		SAMBA_ADMIN_MFA_REQUIRED_ROLES="$mfa_roles" \
		SAMBA_ADMIN_ALLOWED_ORIGINS="http://localhost:$FRONTEND_PORT" \
		SAMBA_ADMIN_SESSION_TTL="$session_ttl" \
		SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS=false \
		"$BIN_DIR/samba-admin-api" >"$LOG_DIR/$name.log" 2>&1 &
	pid=$!
	PIDS="$PIDS $pid"
}

wait_http()
{
	url=$1
	i=0
	while ! curl --fail --silent --output /dev/null --max-time 2 "$url"; do
		i=$((i + 1))
		if [ "$i" -ge 150 ]; then
			printf '%s\n' 'erro: serviço E2E não ficou disponível' >&2
			exit 1
		fi
		sleep 0.2
	done
}

start_api primary "$API_PORT" agent "$AGENT_SOCKET" "$RUNTIME_DIR/primary.db" 15m fixture ''
start_api unavailable "$UNAVAILABLE_PORT" agent "$RUNTIME_DIR/agent-ausente.sock" "$RUNTIME_DIR/unavailable.db" 15m mock ''
start_api expiry "$EXPIRY_PORT" fixture "$RUNTIME_DIR/unused-expiry.sock" "$RUNTIME_DIR/expiry.db" 1s mock ''

env \
	SAMBA_ADMIN_DATA_DIR="$RUNTIME_DIR/mfa-data" \
	SAMBA_ADMIN_DATABASE="$RUNTIME_DIR/mfa.db" \
	SAMBA_ADMIN_AGENT_KEY="$AGENT_KEY" \
	SAMBA_ADMIN_AGENT_EXECUTOR=mock \
	SAMBA_ADMIN_ADAPTER_MODE=fixture \
	SAMBA_ADMIN_FIXTURE_PATH="$FIXTURE_PATH" \
	SAMBA_ADMIN_AUTH_MODE=local \
	SAMBA_ADMIN_BOOTSTRAP_USER=admin-e2e \
	SAMBA_ADMIN_BOOTSTRAP_PASSWORD="$SAMBA_ADMIN_E2E_PASSWORD" \
	SAMBA_ADMIN_TOTP_ENCRYPTION_KEY="$TOTP_KEY" \
	SAMBA_ADMIN_E2E_TOTP_SECRET="$SAMBA_ADMIN_E2E_TOTP_SECRET" \
	SAMBA_ADMIN_E2E_SEED_CONFIRM=sim \
	SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS=false \
	"$BIN_DIR/samba-admin-e2e-mfa-seed" >"$LOG_DIR/mfa-seed.log" 2>&1

start_api mfa "$MFA_PORT" fixture "$RUNTIME_DIR/unused-mfa.sock" "$RUNTIME_DIR/mfa.db" 15m mock 'administrador do sistema'

wait_http "http://127.0.0.1:$API_PORT/readyz"
wait_http "http://127.0.0.1:$UNAVAILABLE_PORT/healthz"
wait_http "http://127.0.0.1:$EXPIRY_PORT/readyz"
wait_http "http://127.0.0.1:$MFA_PORT/readyz"

if [ -n "$FRONTEND_DIST" ]; then
	env \
		SAMBA_ADMIN_E2E_FRONTEND_DIST="$FRONTEND_DIST" \
		SAMBA_ADMIN_E2E_FRONTEND_ADDR="127.0.0.1:$FRONTEND_PORT" \
		SAMBA_ADMIN_E2E_API_URL="http://127.0.0.1:$API_PORT" \
		"$BIN_DIR/samba-admin-e2e-web" >"$LOG_DIR/frontend.log" 2>&1 &
	PIDS="$PIDS $!"
	wait_http "http://127.0.0.1:$FRONTEND_PORT"
	BACKEND_ONLY=true
fi

if [ "$BACKEND_ONLY" = true ]; then
	printf '%s\n' 'backend E2E real iniciado em loopback com mutações desabilitadas'
	while :; do
		for pid in $PIDS; do
			if ! kill -0 "$pid" 2>/dev/null; then
				printf '%s\n' 'erro: um processo do backend E2E encerrou inesperadamente' >&2
				exit 1
			fi
		done
		sleep 1
	done
fi

env \
	VITE_USE_MSW=false \
	SAMBA_ADMIN_E2E_API_URL="http://127.0.0.1:$API_PORT" \
	npm --prefix "$FRONTEND_DIR" run dev -- --config vite.e2e-real.config.ts --host 127.0.0.1 --port "$FRONTEND_PORT" >"$LOG_DIR/frontend.log" 2>&1 &
PIDS="$PIDS $!"
wait_http "http://127.0.0.1:$FRONTEND_PORT"

printf '%s\n' 'ambiente E2E real iniciado em loopback com mutações desabilitadas'

while :; do
	for pid in $PIDS; do
		if ! kill -0 "$pid" 2>/dev/null; then
			printf '%s\n' 'erro: um processo do ambiente E2E encerrou inesperadamente' >&2
			exit 1
		fi
	done
	sleep 1
done
