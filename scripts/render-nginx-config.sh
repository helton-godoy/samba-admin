#!/bin/sh
set -eu

usage() {
	printf '%s\n' 'Uso: render-nginx-config.sh CAMINHO_DE_SAIDA' >&2
}

if [ "$#" -ne 1 ]; then
	usage
	exit 64
fi

: "${SAMBA_ADMIN_PUBLIC_HOST:?SAMBA_ADMIN_PUBLIC_HOST é obrigatório}"
: "${SAMBA_ADMIN_TLS_CERTIFICATE:?SAMBA_ADMIN_TLS_CERTIFICATE é obrigatório}"
: "${SAMBA_ADMIN_TLS_CERTIFICATE_KEY:?SAMBA_ADMIN_TLS_CERTIFICATE_KEY é obrigatório}"
: "${SAMBA_ADMIN_FRONTEND_ROOT:?SAMBA_ADMIN_FRONTEND_ROOT é obrigatório}"

case "$SAMBA_ADMIN_PUBLIC_HOST" in
	*[!A-Za-z0-9.-]*|.*|*..*|*.)
		printf '%s\n' 'SAMBA_ADMIN_PUBLIC_HOST inválido.' >&2
		exit 64
		;;
esac
for path in "$SAMBA_ADMIN_TLS_CERTIFICATE" "$SAMBA_ADMIN_TLS_CERTIFICATE_KEY" "$SAMBA_ADMIN_FRONTEND_ROOT"; do
	case "$path" in
		/*) ;;
		*) printf '%s\n' 'Caminhos de certificado e frontend devem ser absolutos.' >&2; exit 64 ;;
	esac
	case "$path" in
		*[!A-Za-z0-9_./+-]*) printf '%s\n' 'Caminho contém caracteres não permitidos no template.' >&2; exit 64 ;;
	esac
done

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
template="$repo_root/ops/reverse-proxy/nginx-samba-admin.conf.template"
output=$1
output_dir=$(dirname -- "$output")
if [ ! -d "$output_dir" ] || [ -L "$output_dir" ]; then
	printf '%s\n' 'Diretório de saída ausente ou inválido.' >&2
	exit 66
fi
if [ -e "$output" ] || [ -L "$output" ]; then
	printf '%s\n' 'Arquivo de saída já existe; revise-o antes de substituir.' >&2
	exit 73
fi

temporary=$(mktemp "${output}.tmp.XXXXXX")
trap 'rm -f "$temporary"' EXIT HUP INT TERM
sed \
	-e "s|@PUBLIC_HOST@|$SAMBA_ADMIN_PUBLIC_HOST|g" \
	-e "s|@TLS_CERTIFICATE_PATH@|$SAMBA_ADMIN_TLS_CERTIFICATE|g" \
	-e "s|@TLS_CERTIFICATE_KEY_PATH@|$SAMBA_ADMIN_TLS_CERTIFICATE_KEY|g" \
	-e "s|@FRONTEND_ROOT@|$SAMBA_ADMIN_FRONTEND_ROOT|g" \
	"$template" > "$temporary"

if grep -Eq '@[A-Z0-9_]+@' "$temporary"; then
	printf '%s\n' 'Template contém parâmetros não substituídos.' >&2
	exit 1
fi
chmod 0644 "$temporary"
mv "$temporary" "$output"
trap - EXIT HUP INT TERM
printf 'Configuração renderizada em %s; valide-a com nginx -t antes de ativar.\n' "$output"
