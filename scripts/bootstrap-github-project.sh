#!/usr/bin/env bash
set -euo pipefail

PROJECT_TITLE="Samba Admin — Roadmap v1.0.0"
ROOT="$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
BACKLOG="${ROOT}/.github/project/backlog.tsv"
REPO="${GH_REPO:-helton-godoy/samba-admin}"
APPLY=false

usage() {
  cat <<'EOF'
Uso: ./scripts/bootstrap-github-project.sh [--apply] [--repo owner/repo]

Sem --apply, valida o manifesto e não altera o GitHub.
Com --apply, reconcilia labels, Milestones, Issues e Project v2.
EOF
}

while (($#)); do
  case "$1" in
    --apply) APPLY=true; shift ;;
    --repo) REPO="${2:?informe owner/repo}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) printf 'Argumento inválido: %s\n' "$1" >&2; exit 2 ;;
  esac
done

fail() { printf 'ERRO: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || fail "comando ausente: $1"; }

[[ -r "$BACKLOG" ]] || fail "backlog não encontrado: $BACKLOG"
[[ "$REPO" == *"/"* ]] || fail "repositório inválido: $REPO"
OWNER="${REPO%%/*}"
COUNT="$(awk -F '\t' '!/^#/ && $1 != "code" && NF >= 12 {n++} END {print n+0}' "$BACKLOG")"
[[ "$COUNT" -eq 114 ]] || fail "esperados 114 itens; encontrados $COUNT"
[[ "$(awk -F '\t' '!/^#/ && $1 != "code" {print $1}' "$BACKLOG" | sort | uniq -d | wc -l)" -eq 0 ]] ||
  fail "códigos duplicados"

printf 'Repositório: %s\nItens: %s\nProject: %s\n' "$REPO" "$COUNT" "$PROJECT_TITLE"
if [[ "$APPLY" != true ]]; then
  printf '%s\n' 'Plano válido; use --apply para criar/reconciliar 30 labels, 8 Milestones, 114 Issues e 1 Project.'
  exit 0
fi

need gh
need jq
gh auth status >/dev/null 2>&1 || fail "gh não autenticado"
gh repo view "$REPO" >/dev/null 2>&1 || fail "sem acesso a $REPO"
TMP="$(mktemp -d)"
trap 'rm -rf -- "$TMP"' EXIT

printf 'Labels...\n'
while IFS=$'\t' read -r name color description; do
  gh label create "$name" --repo "$REPO" --color "$color" --description "$description" --force >/dev/null
done <<'EOF'
type:feature	1F883D	Nova capacidade do produto
type:bug	D73A4A	Defeito ou regressão
type:security	B60205	Segurança, hardening ou threat model
type:test	5319E7	Testes, homologação ou evidência
type:docs	0075CA	Documentação e governança
type:refactor	FBCA04	Refatoração sem mudança funcional
type:ci	0E8A16	Integração contínua e qualidade
type:release	C2E0C6	Release engineering e distribuição
area:frontend	61DAFB	React, TypeScript, Vite e UX
area:backend	0052CC	API e serviços Go
area:agent	006B75	Agente privilegiado e UDS
area:freebsd	8B5CF6	FreeBSD, UFS2, rc.d e pacote
area:samba	4C1F45	Samba, SMB e compartilhamentos
area:ad	7057FF	AD, Kerberos, LDAP e Winbind
area:acl	D93F0B	ACL NFSv4 e identidades
area:security	B60205	Segurança e governança
area:observability	0E8A16	Logs, métricas, auditoria e SIEM
priority:P0	B60205	Bloqueador
priority:P1	D93F0B	Alta prioridade
priority:P2	FBCA04	Prioridade normal
priority:P3	0E8A16	Baixa prioridade
risk:low	0E8A16	Baixo risco
risk:medium	FBCA04	Risco moderado
risk:high	D93F0B	Alto risco
risk:critical	B60205	Risco crítico
status:blocked	6E7781	Bloqueado por gate
status:ready	0E8A16	Pronto para execução
status:in-progress	1D76DB	Em implementação
status:review	5319E7	Em revisão
status:homologation	FBCA04	Em homologação
EOF

printf 'Milestones...\n'
while IFS=$'\t' read -r title description; do
  if ! gh api --paginate "repos/$REPO/milestones?state=all&per_page=100" --jq '.[].title' |
    grep -Fqx -- "$title"; then
    gh api --method POST "repos/$REPO/milestones" -f "title=$title" -f "description=$description" >/dev/null
  fi
done <<'EOF'
v0.4.0 — Engineering Baseline	Governança, identidade e qualidade estática. Sprints 0–1.
v0.5.0 — Verified CI/CD	CI, supply chain e release engineering. Sprints 2–3.
v0.6.0 — Production Front-end	Front-end real e E2E. Sprint 4.
v0.7.0 — FreeBSD Read-only	Observabilidade e inventário sem mutação. Sprints 5–6.
v0.8.0 — AD Member Diagnostics	Diagnóstico AD sem ingresso. Sprint 7.
v0.9.0 — Safe Samba Mutations	Primeira mutação Samba segura. Sprint 8.
v0.9.5 — NFSv4 ACL and Domain Join	ACL NFSv4 e ingresso como membro. Sprints 9–10.
v1.0.0 — Stable	Segurança dinâmica, homologação e release. Sprints 11–12.
EOF

phase_objective() {
  case "$1" in
    GOV) echo 'Transformar o snapshot inicial em projeto auditável e orientado a Pull Requests.' ;;
    QLT) echo 'Tornar formatação e qualidade de código determinísticas.' ;;
    CICD) echo 'Executar e comprovar CI e segurança de supply chain.' ;;
    REL) echo 'Produzir releases reproduzíveis, atestadas e promovíveis.' ;;
    FE) echo 'Aprovar o console contra API e agente reais, sem depender de MSW.' ;;
    OBS) echo 'Tornar falhas detectáveis, diagnosticáveis e recuperáveis.' ;;
    BSD) echo 'Concluir o produto FreeBSD somente leitura antes da escrita.' ;;
    AD) echo 'Validar completamente o ambiente AD antes do ingresso.' ;;
    SMB) echo 'Entregar a primeira mutação real com aprovação e rollback.' ;;
    ACL) echo 'Administrar ACLs NFSv4 apenas em paths autorizados.' ;;
    JOIN) echo 'Ingressar como membro do AD, nunca como controlador de domínio.' ;;
    SEC) echo 'Executar segurança dinâmica e homologação completa.' ;;
    STB) echo 'Concluir homologação e promoção controlada da v1.0.0.' ;;
    *) fail "fase desconhecida: $1" ;;
  esac
}

phase_rules() {
  case "$1" in
    GOV) echo '- Manter rastreabilidade por Issue, PR e check; não introduzir funcionalidade do produto.' ;;
    QLT) echo '- Integrar ao Makefile e CI; justificar supressões; preservar OpenAPI e gerados.' ;;
    CICD) echo '- Separar checks rápidos/completos; publicar SARIF; bloquear Critical/High, segredos e Actions sem SHA.' ;;
    REL) echo '- Unificar versões; gerar SBOM, checksum e provenance; produção exige aprovação manual.' ;;
    FE) echo '- Usar cliente OpenAPI; testar API/agente real; WCAG 2.2 AA; artefatos sem segredos.' ;;
    OBS) echo '- Propagar correlation ID; redigir eventos; falhar fechado; testar restauração e SIEM.' ;;
    BSD) echo '- Percorrer API sem root → UDS → agente; comandos fixos de leitura; nenhuma mutação.' ;;
    AD) echo '- Diagnosticar DNS/SRV, horário, Kerberos, LDAP, SMB, Winbind e idmap; credencial só em memória.' ;;
    SMB) echo '- Exigir diff, lock, backup, testparm, health check, auditoria e rollback; não alterar ACL.' ;;
    ACL) echo '- Cobrir ACE, herança, ordem e identidades; impedir traversal/TOCTOU; exportar e restaurar ACL.' ;;
    JOIN) echo '- Member server apenas; credencial em memória; timeout/redaction; dupla aprovação e rollback.' ;;
    SEC) echo '- DAST ativo somente em ambiente autorizado; registrar achado, correção e reteste.' ;;
    STB) echo '- Manter não homologados bloqueados; exigir todos os gates e zero P0/P1 ou Critical/High.' ;;
  esac
}

create_body() {
  local out="$1" code="$2" sprint="$3" milestone="$4" sp="$5" priority="$6"
  local type="$7" areas="$8" risk="$9" status="${10}" deps="${11}" title="${12}"
  local prefix="${code%%-*}"
  cat >"$out" <<EOF
## Resultado esperado

${title}.

## Contexto necessário

O \`samba-admin\` é um monorepo React/TypeScript/Vite + Go. A API sem privilégios delega ao agente por UDS autenticado. O alvo é FreeBSD 15.1, UFS2 com ACL NFSv4, Samba standalone ou membro do AD e Windows 11.

O estado de referência é **no-go para produção** e FreeBSD somente leitura. Mutações reais permanecem bloqueadas até seus gates. Samba AD DC, conversão POSIX para NFSv4, DFS-R, mutações CUPS/quotas e editor genérico estão fora da v1.0.0.

Antes de implementar, leia \`AGENTS.md\`, \`README.md\`, \`docs/current-state.md\`, \`docs/github-project-memory.md\` e \`docs/v1-roadmap.md\`; recupere Issues, PRs e checks com \`gh\` e confronte a descrição com o código.

## Objetivo da fase

$(phase_objective "$prefix")

## Escopo

- Implementar e comprovar **${title}**.
- Inspecionar primeiro o estado atual e registrar divergências.
- Limitar a PR a \`${code}\`; abrir Issues separadas para achados não bloqueantes.
- Atualizar OpenAPI, gerados, testes, documentação, capabilities e feature flags quando aplicável.
- Preservar separação de privilégios, catálogo fechado, idempotência, locks, auditoria e fail-closed.
$(phase_rules "$prefix")

## Dependências planejadas

\`${deps}\`. Confirme-as por código com \`gh issue list --state all --search '<CODIGO> in:title'\`. Issue fechada não equivale a capability homologada.

## Critérios de aceite

- [ ] O resultado do título foi entregue sem ampliação silenciosa.
- [ ] Testes proporcionais ao risco cobrem sucesso, falha e fail-closed.
- [ ] OpenAPI e gerados permanecem consistentes quando afetados.
- [ ] Nenhum segredo ou dado institucional sensível aparece em logs ou artefatos.
- [ ] Documentação e \`docs/current-state.md\` foram atualizadas quando necessário.
- [ ] Risco residual, evidências e rollback estão anexados à PR.
- [ ] Mudança crítica possui dois revisores.

## Evidências obrigatórias

1. comandos e resultados sanitizados;
2. testes e justificativa de cobertura;
3. diff ou artefato relevante e checks da PR;
4. falha induzida e rollback quando aplicável;
5. limitações e instruções para o próximo agente.

## Metadados canônicos

| Campo | Valor |
|---|---|
| Código | ${code} |
| Sprint | ${sprint} |
| Milestone | ${milestone} |
| Story Points | ${sp} |
| Prioridade | ${priority} |
| Tipo | ${type} |
| Áreas | ${areas} |
| Risco | ${risk} |
| Estado inicial | ${status} |
| Dependências | ${deps} |

> Gerada de \`.github/project/backlog.tsv\`; o código é imutável.
EOF
}

printf 'Project v2...\n'
PROJECT_NUMBER="$(gh project list --owner "$OWNER" --limit 100 --format json \
  --jq ".projects[] | select(.title == \"$PROJECT_TITLE\") | .number" | head -n1)"
if [[ -z "$PROJECT_NUMBER" ]]; then
  PROJECT_NUMBER="$(gh project create --owner "$OWNER" --title "$PROJECT_TITLE" --format json --jq '.number')" ||
    fail "autorize os escopos project e read:project"
fi

ensure_field() {
  local name="$1" type="$2" options="${3:-}"
  if gh project field-list "$PROJECT_NUMBER" --owner "$OWNER" --format json --jq '.fields[].name' |
    grep -Fqx -- "$name"; then
    return
  fi
  if [[ "$type" == SINGLE_SELECT ]]; then
    gh project field-create "$PROJECT_NUMBER" --owner "$OWNER" --name "$name" \
      --data-type "$type" --single-select-options "$options" >/dev/null
  else
    gh project field-create "$PROJECT_NUMBER" --owner "$OWNER" --name "$name" --data-type "$type" >/dev/null
  fi
}

ensure_field Sprint SINGLE_SELECT 'Sprint 0,Sprint 1,Sprint 2,Sprint 3,Sprint 4,Sprint 5,Sprint 6,Sprint 7,Sprint 8,Sprint 9,Sprint 10,Sprint 11,Sprint 12'
ensure_field 'Story Points' NUMBER
ensure_field Prioridade SINGLE_SELECT 'P0,P1,P2,P3'
ensure_field Risco SINGLE_SELECT 'Low,Medium,High,Critical'
ensure_field Área SINGLE_SELECT 'Frontend,Backend,Agent,FreeBSD,Samba,AD,ACL,Security,Observability'

PROJECT_ID="$(gh project view "$PROJECT_NUMBER" --owner "$OWNER" --format json --jq '.id')"
gh project field-list "$PROJECT_NUMBER" --owner "$OWNER" --format json >"$TMP/fields.json"
gh project item-list "$PROJECT_NUMBER" --owner "$OWNER" --limit 1000 --format json >"$TMP/items.json"

field_id() {
  jq -r --arg name "$1" '.fields[] | select(.name == $name) | .id' "$TMP/fields.json" | head -n1
}

option_id() {
  jq -r --arg field "$1" --arg option "$2" \
    '.fields[] | select(.name == $field) | .options[]? | select(.name == $option) | .id' \
    "$TMP/fields.json" | head -n1
}

set_select() {
  local item="$1" field="$2" value="$3" fid oid
  fid="$(field_id "$field")"
  oid="$(option_id "$field" "$value")"
  [[ -n "$fid" && -n "$oid" ]] || fail "campo ou opção ausente: $field=$value"
  gh project item-edit --id "$item" --project-id "$PROJECT_ID" \
    --field-id "$fid" --single-select-option-id "$oid" >/dev/null
}

set_number() {
  local item="$1" field="$2" value="$3" fid
  fid="$(field_id "$field")"
  [[ -n "$fid" ]] || fail "campo ausente: $field"
  gh project item-edit --id "$item" --project-id "$PROJECT_ID" \
    --field-id "$fid" --number "$value" >/dev/null
}

display_value() {
  local value="${1#*:}"
  case "$value" in
    low) echo Low ;; medium) echo Medium ;; high) echo High ;; critical) echo Critical ;;
    frontend) echo Frontend ;; backend) echo Backend ;; agent) echo Agent ;;
    freebsd) echo FreeBSD ;; samba) echo Samba ;; ad) echo AD ;; acl) echo ACL ;;
    security) echo Security ;; observability) echo Observability ;;
    *) echo "$value" ;;
  esac
}

created=0
updated=0
printf 'Issues...\n'
while IFS=$'\t' read -r code sprint_no sprint milestone sp priority type areas risk status deps title; do
  [[ -n "$code" && "$code" != code && "$code" != \#* ]] || continue
  body="$TMP/$code.md"
  create_body "$body" "$code" "$sprint" "$milestone" "$sp" "$priority" "$type" "$areas" "$risk" "$status" "$deps" "$title"
  number="$(gh issue list --repo "$REPO" --state all --limit 1000 --json number,title \
    --jq ".[] | select(.title | startswith(\"[$code]\")) | .number" | head -n1)"
  IFS=',' read -r -a area_labels <<<"$areas"
  labels=("$type" "priority:$priority" "$risk" "$status" "${area_labels[@]}")
  if [[ -z "$number" ]]; then
    args=()
    for label in "${labels[@]}"; do args+=(--label "$label"); done
    url="$(gh issue create --repo "$REPO" --title "[$code] $title" --body-file "$body" \
      --milestone "$milestone" "${args[@]}")"
    number="${url##*'/'}"
    ((created+=1))
  else
    args=()
    for label in "${labels[@]}"; do args+=(--add-label "$label"); done
    gh issue edit "$number" --repo "$REPO" --title "[$code] $title" --body-file "$body" \
      --milestone "$milestone" "${args[@]}" >/dev/null
    url="$(gh issue view "$number" --repo "$REPO" --json url --jq '.url')"
    ((updated+=1))
  fi
  item_id="$(jq -r --arg url "$url" '.items[] | select(.content.url == $url) | .id' "$TMP/items.json" | head -n1)"
  if [[ -z "$item_id" ]]; then
    item_id="$(gh project item-add "$PROJECT_NUMBER" --owner "$OWNER" --url "$url" --format json --jq '.id')"
  fi
  set_select "$item_id" Sprint "Sprint $sprint_no"
  set_number "$item_id" 'Story Points' "$sp"
  set_select "$item_id" Prioridade "$priority"
  set_select "$item_id" Risco "$(display_value "$risk")"
  set_select "$item_id" Área "$(display_value "${areas%%,*}")"
  printf '  %-9s #%s\n' "$code" "$number"
done <"$BACKLOG"

printf 'Concluído: %s criadas; %s reconciliadas.\n' "$created" "$updated"
printf 'Project: https://github.com/users/%s/projects/%s\n' "$OWNER" "$PROJECT_NUMBER"
printf 'Configure as visualizações e valide campos e branch rules em GOV-001 e GOV-005.\n'
