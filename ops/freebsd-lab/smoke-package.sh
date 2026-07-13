#!/bin/sh
# Homologa build/package/runtime somente leitura em uma VM FreeBSD descartável.
set -eu

usage() {
	printf '%s\n' 'Uso: smoke-package.sh [--keep-installed] CAMINHO_DO_PACKAGE'
}

keep_installed=false
if [ "${1:-}" = "--keep-installed" ]; then
	keep_installed=true
	shift
fi
if [ "$#" -ne 1 ]; then
	usage >&2
	exit 64
fi
if [ "$(/usr/bin/id -u)" -ne 0 ]; then
	printf '%s\n' 'erro: o smoke test deve executar como root na VM descartável' >&2
	exit 77
fi
if [ "$(/usr/bin/uname -s)" != FreeBSD ]; then
	printf '%s\n' 'erro: FreeBSD é obrigatório' >&2
	exit 2
fi

package_input=$1
if [ -L "$package_input" ] || [ ! -f "$package_input" ]; then
	printf '%s\n' 'erro: package ausente, irregular ou symlink' >&2
	exit 66
fi
package=$(/bin/realpath "$package_input")
if /usr/local/sbin/pkg info -e samba-admin; then
	printf '%s\n' 'erro: samba-admin já está instalado; use uma VM limpa ou remova-o de forma controlada' >&2
	exit 65
fi
for binary in /usr/local/sbin/pkg /usr/local/bin/curl /usr/bin/openssl /usr/bin/stat /usr/bin/sockstat /bin/ps /usr/sbin/service /usr/sbin/sysrc; do
	if [ ! -x "$binary" ]; then
		printf 'erro: pré-requisito ausente: %s\n' "$binary" >&2
		exit 69
	fi
done

timestamp=$(/bin/date -u '+%Y%m%dT%H%M%SZ')
evidence_root=${SAMBA_ADMIN_EVIDENCE_ROOT:-/root/samba-admin-evidence}
case "$evidence_root" in
	/*) ;;
	*) printf '%s\n' 'erro: diretorio de evidencias deve ser absoluto' >&2; exit 64 ;;
esac
if [ -L "$evidence_root" ]; then
	printf '%s\n' 'erro: diretorio de evidencias nao pode ser symlink' >&2
	exit 66
fi
/usr/bin/install -d -o root -g wheel -m 0700 "$evidence_root"
evidence_dir="${evidence_root}/package-smoke-${timestamp}"
/usr/bin/install -d -o root -g wheel -m 0700 "$evidence_dir"
/sbin/sha256 "$package" > "${evidence_dir}/package.sha256"
/usr/local/sbin/pkg info -F "$package" > "${evidence_dir}/package-info.txt"
/usr/local/sbin/pkg info -a > "${evidence_dir}/packages-before.txt"
/sbin/sha256 /etc/rc.conf > "${evidence_dir}/rc-conf-before.sha256"

service_state() {
	if /usr/sbin/service "$1" onestatus >/dev/null 2>&1; then
		printf '%s\n' running
	else
		printf '%s\n' stopped-or-unavailable
	fi
}

file_fingerprint() {
	path=$1
	if [ -f "$path" ]; then
		/sbin/sha256 -q "$path"
	else
		printf '%s\n' absent
	fi
}

samba_before=$(service_state samba_server)
cups_before=$(service_state cupsd)
samba_config_before=$(file_fingerprint /usr/local/etc/smb4.conf)
cups_config_before=$(file_fingerprint /usr/local/etc/cups/cupsd.conf)
product_started=false
rc_changed=false
config_created=false
cookie_jar=
cleanup_failure() {
	status=$?
	trap - EXIT HUP INT TERM
	if [ -n "$cookie_jar" ] && [ -f "$cookie_jar" ]; then
		/bin/rm -f "$cookie_jar"
	fi
	if [ "$status" -ne 0 ] && [ "$product_started" = true ]; then
		/usr/sbin/service samba_admin_api onestop >/dev/null 2>&1 || true
		/usr/sbin/service samba_admin_agent onestop >/dev/null 2>&1 || true
		printf 'falha: serviços do produto foram interrompidos; evidências em %s\n' "$evidence_dir" >&2
	fi
	if [ "$status" -ne 0 ] && [ "$rc_changed" = true ]; then
		/usr/sbin/sysrc -x samba_admin_api_enable >/dev/null 2>&1 || true
		/usr/sbin/sysrc -x samba_admin_agent_enable >/dev/null 2>&1 || true
	fi
	if [ "$status" -ne 0 ] && [ "$config_created" = true ]; then
		/bin/rm -f /usr/local/etc/samba-admin/shared.env /usr/local/etc/samba-admin/api.env /usr/local/etc/samba-admin/agent.env
	fi
	if [ "$status" -ne 0 ] && [ ! -f "${evidence_dir}/result.txt" ]; then
		printf 'resultado=falhou\nstatus=%s\n' "$status" > "${evidence_dir}/result.txt"
	fi
	exit "$status"
}
trap cleanup_failure EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

/usr/local/sbin/pkg add "$package"
/usr/local/sbin/pkg info samba-admin > "${evidence_dir}/installed-package.txt"

user_record=$(/usr/sbin/pw usershow sambaadmin)
user_group=$(printf '%s\n' "$user_record" | /usr/bin/cut -d: -f4)
expected_group=$(/usr/sbin/pw groupshow sambaadmin | /usr/bin/cut -d: -f3)
user_home=$(printf '%s\n' "$user_record" | /usr/bin/cut -d: -f9)
user_shell=$(printf '%s\n' "$user_record" | /usr/bin/cut -d: -f10)
user_groups=$(/usr/bin/id -Gn sambaadmin)
if [ "$user_group" != "$expected_group" ] || [ "$user_home" != /nonexistent ] || [ "$user_shell" != /usr/sbin/nologin ] || [ "$user_groups" != sambaadmin ]; then
	printf '%s\n' 'erro: identidade sambaadmin não atende ao perfil restrito' >&2
	exit 1
fi

assert_policy() {
	path=$1
	expected=$2
	actual=$(/usr/bin/stat -f '%Su:%Sg:%p' "$path")
	if [ "$actual" != "$expected" ]; then
		printf 'erro: política de %s é %s; esperado %s\n' "$path" "$actual" "$expected" >&2
		exit 1
	fi
}

wait_agent_socket() {
	socket_ready=false
	attempt=0
	while [ "$attempt" -lt 30 ]; do
		if [ -S /var/run/samba-admin/samba-admin-agent.sock ]; then
			socket_ready=true
			break
		fi
		if ! /usr/sbin/service samba_admin_agent onestatus >/dev/null 2>&1; then
			printf '%s\n' 'erro: agente encerrou antes de criar o socket' >&2
			exit 1
		fi
		attempt=$((attempt + 1))
		/bin/sleep 1
	done
	if [ "$socket_ready" != true ]; then
		printf '%s\n' 'erro: timeout aguardando socket do agente' >&2
		exit 1
	fi
}

assert_policy /var/db/samba-admin sambaadmin:sambaadmin:40750
assert_policy /var/db/samba-admin-agent root:wheel:40700
assert_policy /var/log/samba-admin root:sambaadmin:40750
assert_policy /var/run/samba-admin root:sambaadmin:42750
assert_policy /usr/local/etc/samba-admin root:sambaadmin:40750

if /usr/sbin/service samba_admin_api enabled >/dev/null 2>&1 || /usr/sbin/service samba_admin_agent enabled >/dev/null 2>&1; then
	printf '%s\n' 'erro: package habilitou serviço automaticamente' >&2
	exit 1
fi

hmac_key=$(/usr/bin/openssl rand -hex 32)
bootstrap_password=$(/usr/bin/openssl rand -hex 24)
totp_key=$(/usr/bin/openssl rand -base64 32 | /usr/bin/tr -d '\n')
api_uid=$(/usr/bin/id -u sambaadmin)

/usr/bin/install -o root -g sambaadmin -m 0640 /dev/null /usr/local/etc/samba-admin/shared.env
/usr/bin/install -o root -g sambaadmin -m 0640 /dev/null /usr/local/etc/samba-admin/api.env
/usr/bin/install -o root -g wheel -m 0600 /dev/null /usr/local/etc/samba-admin/agent.env
config_created=true

{
	printf 'SAMBA_ADMIN_AGENT_KEY=%s\n' "$hmac_key"
	printf '%s\n' 'SAMBA_ADMIN_AGENT_KEY_ID=rc0'
} > /usr/local/etc/samba-admin/shared.env
{
	printf '%s\n' 'SAMBA_ADMIN_HTTP_ADDR=127.0.0.1:8080'
	printf '%s\n' 'SAMBA_ADMIN_DATA_DIR=/var/db/samba-admin'
	printf 'SAMBA_ADMIN_DATABASE=/var/db/samba-admin/smoke-%s.db\n' "$timestamp"
	printf '%s\n' 'SAMBA_ADMIN_AGENT_SOCKET=/var/run/samba-admin/samba-admin-agent.sock'
	printf '%s\n' 'SAMBA_ADMIN_AUTH_MODE=local'
	printf '%s\n' 'SAMBA_ADMIN_BOOTSTRAP_USER=admin'
	printf 'SAMBA_ADMIN_BOOTSTRAP_PASSWORD=%s\n' "$bootstrap_password"
	printf 'SAMBA_ADMIN_TOTP_ENCRYPTION_KEY=%s\n' "$totp_key"
	printf '%s\n' 'SAMBA_ADMIN_ADAPTER_MODE=agent'
	printf '%s\n' 'SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS=false'
	printf '%s\n' 'SAMBA_ADMIN_ALLOWED_ORIGINS=http://localhost:8080'
	printf '%s\n' 'SAMBA_ADMIN_SESSION_TTL=1h'
	printf '%s\n' 'SAMBA_ADMIN_REQUEST_TIMEOUT=30s'
	printf '%s\n' 'SAMBA_ADMIN_MAX_BODY_BYTES=2097152'
} > /usr/local/etc/samba-admin/api.env
{
	printf '%s\n' 'SAMBA_ADMIN_DATA_DIR=/var/db/samba-admin-agent'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_SOCKET=/var/run/samba-admin/samba-admin-agent.sock'
	printf 'SAMBA_ADMIN_AGENT_NONCE_STORE=/var/db/samba-admin-agent/smoke-%s-nonces.json\n' "$timestamp"
	printf '%s\n' 'SAMBA_ADMIN_AGENT_NONCE_TTL=5m'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_NONCE_CACHE_LIMIT=4096'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_MAX_MESSAGE_BYTES=1048576'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_MAX_OUTPUT_BYTES=131072'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_MAX_CONCURRENT=4'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_MAX_CLOCK_SKEW=2m'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_SOCKET_MODE=0660'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_SOCKET_OWNER=root'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_SOCKET_GROUP=sambaadmin'
	printf 'SAMBA_ADMIN_AGENT_EXPECTED_PEER_UID=%s\n' "$api_uid"
	printf '%s\n' 'SAMBA_ADMIN_ADAPTER_MODE=agent'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_EXECUTOR=readonly-freebsd'
	printf '%s\n' 'SAMBA_ADMIN_AGENT_CAPABILITIES=system.inspect,capabilities.inspect,filesystem.inspect,samba.inspect,samba.testparm,domain.member.diagnose,cups.inspect'
	printf '%s\n' 'SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS=false'
} > /usr/local/etc/samba-admin/agent.env
/bin/chmod 0640 /usr/local/etc/samba-admin/shared.env /usr/local/etc/samba-admin/api.env
/bin/chmod 0600 /usr/local/etc/samba-admin/agent.env

/usr/sbin/sysrc samba_admin_agent_enable=YES >/dev/null
/usr/sbin/sysrc samba_admin_api_enable=YES >/dev/null
rc_changed=true
product_started=true
/usr/sbin/service samba_admin_agent start
wait_agent_socket
assert_policy /var/run/samba-admin root:sambaadmin:42750
assert_policy /var/run/samba-admin/samba-admin-agent.sock root:sambaadmin:140660
/usr/sbin/service samba_admin_api start
assert_policy /var/run/samba-admin root:sambaadmin:42750
assert_policy /var/log/samba-admin root:sambaadmin:40750

agent_pid=$(/bin/cat /var/run/samba_admin_agent.pid)
api_pid=$(/bin/cat /var/run/samba_admin_api.pid)
agent_uid=$(/bin/ps -o uid= -p "$agent_pid" | /usr/bin/tr -d ' ')
api_process_uid=$(/bin/ps -o uid= -p "$api_pid" | /usr/bin/tr -d ' ')
if [ "$agent_uid" != 0 ] || [ "$api_process_uid" != "$api_uid" ]; then
	printf 'erro: identidades de processo invalidas: agent=%s api=%s\n' "$agent_uid" "$api_process_uid" >&2
	exit 1
fi
ready=false
attempt=0
while [ "$attempt" -lt 30 ]; do
	if /usr/local/bin/curl --fail --silent --show-error --max-time 2 http://localhost:8080/healthz > "${evidence_dir}/health.json" 2>/dev/null; then
		ready=true
		break
	fi
	attempt=$((attempt + 1))
	/bin/sleep 1
done
if [ "$ready" != true ]; then
	printf '%s\n' 'erro: API não respondeu ao health check' >&2
	exit 1
fi
if ! /usr/bin/sockstat -4 -l | /usr/bin/grep -Fq '127.0.0.1:8080'; then
	printf '%s\n' 'erro: API nao esta vinculada ao loopback esperado' >&2
	exit 1
fi
/usr/local/bin/curl --fail --silent --show-error --max-time 5 http://localhost:8080/readyz > "${evidence_dir}/ready.json"

cookie_jar="${evidence_dir}/cookies.txt"
/usr/bin/touch "$cookie_jar"
/bin/chmod 0600 "$cookie_jar"
login_response=$(printf '{"username":"admin","password":"%s"}' "$bootstrap_password" | /usr/local/bin/curl --fail --silent --show-error --max-time 10 -c "$cookie_jar" -H 'Content-Type: application/json' --data-binary @- http://localhost:8080/api/v1/auth/login)
csrf=$(printf '%s\n' "$login_response" | /usr/bin/sed -n 's/.*"csrfToken":"\([^"]*\)".*/\1/p')
if [ -z "$csrf" ]; then
	printf '%s\n' 'erro: login não retornou token CSRF' >&2
	exit 1
fi

sanitized_api=$(/usr/bin/mktemp /usr/local/etc/samba-admin/api.env.XXXXXX)
/usr/bin/grep -v '^SAMBA_ADMIN_BOOTSTRAP_PASSWORD=' /usr/local/etc/samba-admin/api.env > "$sanitized_api"
/usr/bin/install -o root -g sambaadmin -m 0640 "$sanitized_api" /usr/local/etc/samba-admin/api.env
/bin/rm -f "$sanitized_api"

hostname=$(/bin/hostname)
/usr/local/bin/curl --fail --silent --show-error --max-time 30 -b "$cookie_jar" http://localhost:8080/api/v1/system > "${evidence_dir}/system.json"
/usr/local/bin/curl --fail --silent --show-error --max-time 30 -b "$cookie_jar" http://localhost:8080/api/v1/capabilities > "${evidence_dir}/capabilities.json"
/usr/local/bin/curl --fail --silent --show-error --max-time 30 -b "$cookie_jar" http://localhost:8080/api/v1/filesystems > "${evidence_dir}/filesystems.json"
/usr/local/bin/curl --fail --silent --show-error --max-time 30 -b "$cookie_jar" http://localhost:8080/api/v1/domain > "${evidence_dir}/domain.json"
/usr/local/bin/curl --fail --silent --show-error --max-time 30 -b "$cookie_jar" http://localhost:8080/api/v1/samba > "${evidence_dir}/samba.json"
/usr/local/bin/curl --fail --silent --show-error --max-time 30 -b "$cookie_jar" http://localhost:8080/api/v1/cups > "${evidence_dir}/cups.json"
/usr/local/bin/curl --fail --silent --show-error --max-time 30 -b "$cookie_jar" http://localhost:8080/api/v1/printers > "${evidence_dir}/printers.json"
/usr/bin/grep -Fq "\"hostname\":\"${hostname}\"" "${evidence_dir}/system.json"
/usr/bin/grep -Fq '"freebsdVersion":"15.1-' "${evidence_dir}/system.json"
/usr/bin/grep -Fq '"id":"system.inspect"' "${evidence_dir}/capabilities.json"
/usr/bin/grep -Fq '"mountPoint":"/"' "${evidence_dir}/filesystems.json"
/usr/bin/grep -Fq '"joined":false' "${evidence_dir}/domain.json"
/usr/bin/grep -Fq '"configurationPath":"/usr/local/etc/smb4.conf"' "${evidence_dir}/samba.json"
/usr/bin/grep -Fq '"printers":' "${evidence_dir}/cups.json"

/usr/sbin/service samba_admin_agent stop
agent_unavailable_status=$(/usr/local/bin/curl --silent --show-error --max-time 10 -o "${evidence_dir}/ready-agent-unavailable.json" -w '%{http_code}' http://localhost:8080/readyz)
if [ "$agent_unavailable_status" != 503 ]; then
	printf 'erro: readiness sem agente retornou HTTP %s; esperado 503\n' "$agent_unavailable_status" >&2
	exit 1
fi
/usr/bin/grep -Fq '"correlationId":' "${evidence_dir}/ready-agent-unavailable.json"
for endpoint in samba cups; do
	status=$(/usr/local/bin/curl --silent --show-error --max-time 10 -o "${evidence_dir}/${endpoint}-agent-unavailable.json" -w '%{http_code}' -b "$cookie_jar" "http://localhost:8080/api/v1/${endpoint}")
	if [ "$status" != 503 ]; then
		printf 'erro: %s sem agente retornou HTTP %s; esperado 503\n' "$endpoint" "$status" >&2
		exit 1
	fi
	/usr/bin/grep -Fq '"correlationId":' "${evidence_dir}/${endpoint}-agent-unavailable.json"
done
/usr/sbin/service samba_admin_agent start
wait_agent_socket
/usr/local/bin/curl --fail --silent --show-error --max-time 30 http://localhost:8080/readyz > "${evidence_dir}/ready-recovered.json"

mutation_payload='{"id":"pending","name":"SmokeBlocked","description":"Validação de bloqueio","path":"/srv/dados/smoke","enabled":true,"readOnly":false,"guestAccess":false,"allowedPrincipals":["root"],"deniedPrincipals":[],"encryption":"desired","signing":"mandatory","auditProfile":"security","recycleBin":false,"dfs":false,"vfsModules":[],"maxConnections":1,"aclModel":"nfsv4"}'
mutation_status=$(printf '%s' "$mutation_payload" | /usr/local/bin/curl --silent --show-error --max-time 10 -o "${evidence_dir}/mutation-blocked.json" -w '%{http_code}' -b "$cookie_jar" -H 'Content-Type: application/json' -H "X-CSRF-Token: ${csrf}" -H 'Idempotency-Key: freebsd-smoke-mutation-blocked' --data-binary @- http://localhost:8080/api/v1/shares)
if [ "$mutation_status" != 503 ]; then
	printf 'erro: mutação retornou HTTP %s; esperado 503\n' "$mutation_status" >&2
	exit 1
fi
/usr/bin/grep -Fq 'mutation_blocked' "${evidence_dir}/mutation-blocked.json"

for secret_value in "$hmac_key" "$bootstrap_password" "$totp_key"; do
	if /usr/bin/grep -Fq "$secret_value" /var/log/samba-admin/api.log /var/log/samba-admin/agent.log; then
		printf '%s\n' 'erro: segredo efemero encontrado nos logs' >&2
		exit 1
	fi
done

/bin/rm -f "$cookie_jar"
cookie_jar=
/usr/local/sbin/pkg delete -y samba-admin > "${evidence_dir}/deinstall-active.txt" 2>&1
product_started=false
if /usr/sbin/service samba_admin_api onestatus >/dev/null 2>&1 || /usr/sbin/service samba_admin_agent onestatus >/dev/null 2>&1; then
	printf '%s\n' 'erro: processo do produto persistiu apos desinstalacao' >&2
	exit 1
fi
assert_policy /var/run/samba-admin root:sambaadmin:42750
if [ -e /var/run/samba-admin/samba-admin-agent.sock ]; then
	printf '%s\n' 'erro: socket persistiu após parada do agente' >&2
	exit 1
fi
if [ -e /usr/local/sbin/samba-admin-api ] || [ -e /usr/local/sbin/samba-admin-agent ]; then
	printf '%s\n' 'erro: binarios persistiram apos desinstalacao' >&2
	exit 1
fi
if [ ! -d /var/db/samba-admin ] || [ ! -d /var/db/samba-admin-agent ]; then
	printf '%s\n' 'erro: dados auditaveis nao foram preservados apos desinstalacao' >&2
	exit 1
fi

if [ "$keep_installed" = false ]; then
	/usr/sbin/sysrc -x samba_admin_api_enable >/dev/null 2>&1 || true
	/usr/sbin/sysrc -x samba_admin_agent_enable >/dev/null 2>&1 || true
	rc_changed=false
	/bin/rm -f /usr/local/etc/samba-admin/shared.env /usr/local/etc/samba-admin/api.env /usr/local/etc/samba-admin/agent.env
	config_created=false
else
	/usr/local/sbin/pkg add "$package"
	product_started=true
	/usr/sbin/service samba_admin_agent start
	wait_agent_socket
	assert_policy /var/run/samba-admin/samba-admin-agent.sock root:sambaadmin:140660
	/usr/sbin/service samba_admin_api start
	assert_policy /var/run/samba-admin root:sambaadmin:42750
	ready=false
	attempt=0
	while [ "$attempt" -lt 30 ]; do
		if /usr/local/bin/curl --fail --silent --show-error --max-time 2 http://localhost:8080/healthz >/dev/null 2>&1; then
			ready=true
			break
		fi
		attempt=$((attempt + 1))
		/bin/sleep 1
	done
	if [ "$ready" != true ]; then
		printf '%s\n' 'erro: API nao retomou apos reinstalacao' >&2
		exit 1
	fi
	/usr/local/bin/curl --fail --silent --show-error --max-time 10 http://localhost:8080/readyz > "${evidence_dir}/ready-reinstalled.json"
	/usr/sbin/service samba_admin_api stop
	/usr/sbin/service samba_admin_agent stop
	product_started=false
	/usr/sbin/sysrc -x samba_admin_api_enable >/dev/null 2>&1 || true
	/usr/sbin/sysrc -x samba_admin_agent_enable >/dev/null 2>&1 || true
	/bin/rm -f /usr/local/etc/samba-admin/shared.env /usr/local/etc/samba-admin/api.env /usr/local/etc/samba-admin/agent.env
	config_created=false
	rc_changed=false
fi

samba_after=$(service_state samba_server)
cups_after=$(service_state cupsd)
samba_config_after=$(file_fingerprint /usr/local/etc/smb4.conf)
cups_config_after=$(file_fingerprint /usr/local/etc/cups/cupsd.conf)
if [ "$samba_after" != "$samba_before" ] || [ "$cups_after" != "$cups_before" ] || \
   [ "$samba_config_after" != "$samba_config_before" ] || [ "$cups_config_after" != "$cups_config_before" ]; then
	printf '%s\n' 'erro: estado de Samba ou CUPS foi alterado pelo smoke test' >&2
	exit 1
fi
/usr/local/sbin/pkg info -a > "${evidence_dir}/packages-after.txt"
{
	printf 'resultado=aprovado\n'
	printf 'package=%s\n' "$package"
	printf 'keep_installed=%s\n' "$keep_installed"
	printf 'samba_state=%s\n' "$samba_after"
	printf 'cups_state=%s\n' "$cups_after"
	printf 'credential_retained=false\n'
} > "${evidence_dir}/result.txt"

trap - EXIT HUP INT TERM
printf 'smoke test aprovado; evidências em %s\n' "$evidence_dir"
