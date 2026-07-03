#!/usr/bin/env bash
# packages/espyna-golang/scripts/audit-table-names.sh — grep gate for the
# table-name single-source invariant (Q-TABLE-NAMES, 20260703
# table-name-single-source). Fails if any contrib/*/internal/adapter/**.go
# file (excluding _test.go) still contains a bare quoted-table-literal SQL
# keyword site (FROM/JOIN/INSERT INTO/UPDATE/DELETE FROM) for a table that
# registry/entityid/entityid.go already has a constant for. Those sites must
# use the entityid constant instead (see contrib/postgres/internal/adapter/
# operation/outcome_matrix_query.go as the exemplar).
#
# The known-table list is GENERATED from registry/entityid/entityid.go on
# every run — never hand-maintained — so the gate self-updates as entityid
# grows.
#
# Usage (run from repo root or from packages/espyna-golang):
#   bash packages/espyna-golang/scripts/audit-table-names.sh
#
# Exit 0 = clean (zero hits). Exit 1 = hits found (printed to stdout, file:line:text).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESPYNA_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ESPYNA_ROOT"

if [ -t 1 ]; then
  C_RED=$'\033[31m'; C_GREEN=$'\033[32m'; C_YEL=$'\033[33m'; C_CYAN=$'\033[36m'; C_OFF=$'\033[0m'
else
  C_RED=''; C_GREEN=''; C_YEL=''; C_CYAN=''; C_OFF=''
fi

ENTITYID_FILE="registry/entityid/entityid.go"
if [ ! -f "$ENTITYID_FILE" ]; then
  printf "%sFATAL: %s not found%s\n" "$C_RED" "$ENTITYID_FILE" "$C_OFF" >&2
  exit 2
fi

# ---------------------------------------------------------------------------
# Generate the known-table list from registry/entityid/entityid.go.
# Matches lines like:  Name = "table_name"
# ---------------------------------------------------------------------------
TMPDIR_AUDIT="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_AUDIT"' EXIT

TABLES_FILE="$TMPDIR_AUDIT/tables.txt"
grep -oE '^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*"[a-z_][a-z0-9_]*"' "$ENTITYID_FILE" \
  | sed -E 's/^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*"([a-z_][a-z0-9_]*)"/\1/' \
  | sort -u > "$TABLES_FILE"

table_count=$(wc -l < "$TABLES_FILE" | tr -d ' ')
printf "%s=== audit-table-names: %s known tables from %s ===%s\n" "$C_CYAN" "$table_count" "$ENTITYID_FILE" "$C_OFF"

# Build a single alternation regex: (FROM|JOIN|INTO|UPDATE)\s+(tbl1|tbl2|...)\b
# and a DELETE FROM variant. Piping the table list into the regex keeps this
# script table-count-agnostic (currently 240+, will grow).
ALT="$(paste -sd'|' "$TABLES_FILE")"

HITS_FILE="$TMPDIR_AUDIT/hits.txt"
: > "$HITS_FILE"

# Scan every contrib/*/internal/adapter/**.go file, excluding _test.go.
while IFS= read -r -d '' f; do
  # Skip comment-only lines cheaply (a real hit could still follow inline
  # code on a code+comment line — grep below re-checks per match, not per line).
  grep -nE "(FROM|JOIN|INTO|UPDATE)[[:space:]]+($ALT)\\b" "$f" 2>/dev/null \
    | grep -vE '^[0-9]+:[[:space:]]*//' \
    | sed "s#^#$f:#" >> "$HITS_FILE" || true
  grep -nE "DELETE[[:space:]]+FROM[[:space:]]+($ALT)\\b" "$f" 2>/dev/null \
    | grep -vE '^[0-9]+:[[:space:]]*//' \
    | sed "s#^#$f:#" >> "$HITS_FILE" || true
done < <(find contrib -type d -name adapter -path '*/internal/adapter' -print0 2>/dev/null | while IFS= read -r -d '' d; do
    find "$d" -type f -name '*.go' ! -name '*_test.go' -print0
  done)

sort -u -t: -k1,1 -k2,2n "$HITS_FILE" -o "$HITS_FILE" 2>/dev/null || true

if [ ! -s "$HITS_FILE" ]; then
  printf "\n%sclean — zero bare table-literal hits in contrib/*/internal/adapter/**.go%s\n" "$C_GREEN" "$C_OFF"
  exit 0
fi

hit_count=$(wc -l < "$HITS_FILE" | tr -d ' ')
printf "\n%sHits (file:line:text):%s\n" "$C_YEL" "$C_OFF"
while IFS= read -r line; do
  printf "  %s%s%s\n" "$C_RED" "$line" "$C_OFF"
done < "$HITS_FILE"

printf "\n%s%s bare table-literal hit(s) found. Replace with the registry/entityid constant (see contrib/postgres/internal/adapter/operation/outcome_matrix_query.go).%s\n" \
  "$C_RED" "$hit_count" "$C_OFF" >&2
exit 1
