#!/bin/sh
# Este script imprime somente um plano. Alterar o host esta fora do escopo.
set -eu

usage() {
  printf '%s\n' 'Uso: prepare-lab.sh [--platform proxmox|virtualbox|bhyve|manual] [--help]'
}

platform=proxmox
while [ "$#" -gt 0 ]; do
  case "$1" in
    --platform)
      shift
      if [ "$#" -eq 0 ]; then
        printf '%s\n' 'erro: --platform exige um valor' >&2
        exit 64
      fi
      platform=$1
      ;;
    --apply)
      printf '%s\n' 'erro: --apply e deliberadamente indisponivel; este repositorio nao altera hosts' >&2
      exit 64
      ;;
    --help)
      usage
      exit 0
      ;;
    *)
      printf 'erro: argumento nao suportado: %s\n' "$1" >&2
      usage >&2
      exit 64
      ;;
  esac
  shift
done

case "$platform" in
  proxmox|virtualbox|bhyve|manual) ;;
  *)
    printf 'erro: plataforma nao suportada: %s\n' "$platform" >&2
    exit 64
    ;;
esac

printf 'Plano do laboratorio FreeBSD (%s)\n' "$platform"
printf '%s\n' '1. Verifique checksum/assinatura do instalador FreeBSD e registre a versao.'
printf '%s\n' '2. Crie rede administrativa isolada, sem rota para o AD de producao.'
printf '%s\n' '3. Provisione discos separados para sistema e UFS2 descartavel de teste.'
printf '%s\n' '4. Crie o snapshot baseline-clean antes de instalar qualquer pacote.'
printf '%s\n' '5. Instale pacotes aprovados sob controle de mudanca; mantenha o agente desabilitado.'
printf '%s\n' '6. Execute check-readiness.sh e collect-fixtures.sh antes de qualquer teste mutavel.'
printf '%s\n' '7. Registre snapshots, pacotes, fixtures e testes no ticket de mudanca.'
printf '%s\n' '8. Restaure o checkpoint apos teste destrutivo ou inconclusivo.'
