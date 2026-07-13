#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
dist_dir=${FRONTEND_DIST_DIR:-"$repo_root/frontend/dist"}
artifact_dir=${FRONTEND_ARTIFACT_DIR:-"$repo_root/artifacts"}
archive_name=${FRONTEND_ARCHIVE_NAME:-frontend-dist.tar.gz}
source_date_epoch=${SOURCE_DATE_EPOCH:-0}

case "$source_date_epoch" in
  ''|*[!0-9]*)
    echo "SOURCE_DATE_EPOCH deve ser um inteiro não negativo." >&2
    exit 2
    ;;
esac

if [ ! -f "$dist_dir/index.html" ]; then
  echo "frontend-dist ausente; execute o build do front-end primeiro." >&2
  exit 1
fi

forbidden=$(find "$dist_dir" -type f \( \
  -name 'mockServiceWorker.js' -o \
  -name '*.map' -o \
  -name '*.trace' -o \
  -name '*.webm' -o \
  -name '*.zip' -o \
  -name '.env*' \
\) -print)
if [ -n "$forbidden" ]; then
  echo "frontend-dist contém arquivos proibidos de desenvolvimento ou teste." >&2
  printf '%s\n' "$forbidden" >&2
  exit 1
fi

if find "$dist_dir" -type l -print | grep -q .; then
  echo "frontend-dist não pode conter links simbólicos." >&2
  exit 1
fi

mkdir -p "$artifact_dir"
archive="$artifact_dir/$archive_name"

tar --sort=name \
  --mtime="@$source_date_epoch" \
  --owner=0 \
  --group=0 \
  --numeric-owner \
  -cf - \
  -C "$dist_dir" . | gzip -n > "$archive"

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$artifact_dir" && sha256sum "$archive_name" > frontend-dist.SHA256)
elif command -v sha256 >/dev/null 2>&1; then
  checksum=$(sha256 -q "$archive")
  printf '%s  %s\n' "$checksum" "$archive_name" > "$artifact_dir/frontend-dist.SHA256"
else
  echo "sha256sum ou sha256 é obrigatório para gerar o manifesto." >&2
  exit 1
fi

echo "Artefato frontend-dist criado sem conteúdo de desenvolvimento ou teste."
