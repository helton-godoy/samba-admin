#!/bin/sh
# scripts/run-integrated.sh
#
# Verifica um ambiente FreeBSD já provisionado e inicia apenas o frontend local.
# O provisionamento, a compilação e a gestão de serviços são fluxos separados e
# exigem aprovação operacional; este script nunca altera o host remoto.

set -eu

VM_HOST="${SAMBA_ADMIN_FREEBSD_HOST:-root@192.168.122.84}"
API_URL="${SAMBA_ADMIN_API_URL:-http://127.0.0.1:8080}"
SSH_CONFIG="${SAMBA_ADMIN_SSH_CONFIG:-/dev/null}"

case "$VM_HOST" in
    -*|*[!A-Za-z0-9_.@:-]*)
        printf '%s\n' "Host FreeBSD inválido: $VM_HOST" >&2
        exit 2
        ;;
esac
case "$SSH_CONFIG" in
    /*) ;;
    *)
        printf '%s\n' "Arquivo de configuracao SSH deve ser absoluto: $SSH_CONFIG" >&2
        exit 2
        ;;
esac

ssh_vm()
{
    ssh -F "$SSH_CONFIG" "$VM_HOST" "$@"
}

echo "======================================================================"
echo " Samba Admin Suite — Verificação Integrada Somente Leitura"
echo "======================================================================"
echo

# 1. Verificar conectividade
echo "[-] Verificando conexão SSH com ${VM_HOST}..."
if ! ssh -F "$SSH_CONFIG" -o ConnectTimeout=3 "$VM_HOST" /usr/bin/uname -a; then
    echo "Erro: não foi possível consultar o FreeBSD por SSH." >&2
    exit 1
fi
echo

echo "[-] Consultando release e pacotes relevantes..."
ssh_vm /bin/freebsd-version -ku
ssh_vm /usr/local/sbin/pkg query '%n-%v' samba423 || true
ssh_vm /usr/local/sbin/pkg query '%n-%v' cups || true
echo

echo "[-] Verificando a API já provisionada em ${API_URL}..."
if ! curl --fail --silent --show-error --max-time 5 "${API_URL}/healthz"; then
    printf '%s\n' "Inicie antes o tunel: ssh -F ${SSH_CONFIG} -N -L 8080:127.0.0.1:8080 ${VM_HOST}" >&2
    exit 1
fi
echo

echo "======================================================================"
echo " Iniciando o Frontend Vite no Host Linux"
echo " O console estará disponível em: http://localhost:5173"
echo "======================================================================"
echo

export VITE_USE_MSW=false
npm --prefix frontend run dev -- --host 127.0.0.1
