#!/bin/sh

set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
REPORT_DIR="$ROOT_DIR/frontend/playwright-report-real"
RESULT_DIR="$ROOT_DIR/frontend/test-results-real"
SAFE=true

contains_secret()
{
	secret=$1
	[ -z "$secret" ] && return 1
	for path in "$REPORT_DIR" "$RESULT_DIR"; do
		if [ -e "$path" ] && grep -aR -F -q -- "$secret" "$path"; then
			return 0
		fi
	done
	return 1
}

if contains_secret "${SAMBA_ADMIN_E2E_PASSWORD:-}" || contains_secret "${SAMBA_ADMIN_E2E_TOTP_SECRET:-}"; then
	SAFE=false
	printf '%s\n' 'erro: artefato E2E continha material sensível e foi removido' >&2
fi

for path in "$REPORT_DIR" "$RESULT_DIR"; do
	if [ -d "$path" ] && find "$path" -type f \( -name 'trace.zip' -o -name '*.png' -o -name '*.webm' \) | grep -q .; then
		SAFE=false
		printf '%s\n' 'erro: trace, screenshot ou vídeo inesperado foi removido' >&2
	fi
done

if [ "$SAFE" != true ]; then
	rm -rf "$REPORT_DIR" "$RESULT_DIR"
	if [ -n "${GITHUB_OUTPUT:-}" ]; then
		printf '%s\n' 'safe=false' >> "$GITHUB_OUTPUT"
	fi
	exit 1
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
	printf '%s\n' 'safe=true' >> "$GITHUB_OUTPUT"
fi
printf '%s\n' 'artefatos E2E verificados sem senha, segredo TOTP, trace, screenshot ou vídeo'
