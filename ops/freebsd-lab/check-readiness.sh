#!/bin/sh
# Verifica pre-requisitos do laboratorio FreeBSD sem alterar o host.
set -eu

if [ "${1:-}" = "--help" ]; then
  printf '%s\n' 'Uso: check-readiness.sh'
  exit 0
fi
if [ "$#" -ne 0 ]; then
  printf '%s\n' 'erro: este comando nao aceita argumentos' >&2
  exit 64
fi

status=0
check_binary() {
  binary=$1
  if command -v "$binary" >/dev/null 2>&1; then
    printf 'ok    %s: %s\n' "$binary" "$(command -v "$binary")"
  else
    printf 'aviso %s: nao encontrado\n' "$binary" >&2
    status=1
  fi
}

os_name=$(uname -s 2>/dev/null || printf desconhecido)
os_release=$(uname -r 2>/dev/null || printf desconhecido)
printf 'sistema: %s %s\n' "$os_name" "$os_release"

if [ "$os_name" != FreeBSD ]; then
  printf '%s\n' 'erro: FreeBSD e obrigatorio para a rodada de homologacao' >&2
  exit 2
fi

case "$os_release" in
  15.1-RELEASE*|15.1-STABLE*) printf '%s\n' 'ok    release: familia baseline aprovada' ;;
  *) printf '%s\n' 'aviso release: registre aprovacao explicita para esta versao' >&2; status=1 ;;
esac

for binary in pkg go sqlite3 uname sysctl mount df ifconfig service testparm smbstatus wbinfo lpstat getfacl; do
  check_binary "$binary"
done

if [ -r /etc/fstab ]; then
  printf '%s\n' 'ok    /etc/fstab: legivel para inventario sanitizado'
else
  printf '%s\n' 'aviso /etc/fstab: indisponivel para o coletor' >&2
  status=1
fi

if [ -d /var/db/samba-admin ]; then
  printf '%s\n' 'info  diretorio de dados ja existe; nao o reutilize para fixtures'
fi

if [ "$status" -ne 0 ]; then
  printf '%s\n' 'resultado: ha lacunas; nao marque capabilities como suportadas' >&2
else
  printf '%s\n' 'resultado: pre-requisitos encontrados; prossiga apenas com coleta somente leitura'
fi
exit "$status"
