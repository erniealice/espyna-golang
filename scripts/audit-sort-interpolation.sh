#!/usr/bin/env bash
# packages/espyna-golang/scripts/audit-sort-interpolation.sh
#
# Lint gate for the SQL-injection dialect-parity invariant (W4 of
# docs/plan/20260728-sql-injection-dialect-parity/plan.md §6.1).
#
# WHAT IT CATCHES
# ---------------
# A request-supplied SQL *identifier* (a sort column) reaching SQL *text*
# without passing a recognised guard. This is the F-1 defect class: the
# postgres adapter tree routes every caller-supplied sort column through
# core.BuildOrderBy + a per-entity <entity>SortableSQLCols whitelist; the
# mysql and sqlserver ports dropped that guard in 20 places (measured, see
# BASELINE below). Neither existing gate can see this class:
# apps/service-admin/scripts/audit-direct-sql.sh checks *layer placement*,
# scripts/audit-table-names.sh checks *table identifiers*.
#
# THE RULE (function-scoped — all three limbs must hold in ONE function body)
# ---------------------------------------------------------------------------
#   (i)   ASSIGNMENT   a request sort/filter field is read into a local:
#                        =<sp>req.Sort... | =<sp>req.GetSort... | =<sp>filter.Field
#   (ii)  INTERPOLATION that value can reach query TEXT. Three limbs:
#           ii-a  string concatenation of a sort-ish var:  + sortField +
#                 / + sortOrder + / + orderByClause + ...
#           ii-b  a %s verb immediately after ORDER BY, in any dialect's
#                 quoting:  ORDER BY %s  |  ORDER BY [%s]  |  ORDER BY `%s`
#           ii-c  a %s verb on a CASE WHEN line — the "parameterised-looking"
#                 sort ladder that mysql actually *interpolates*
#                 (CASE WHEN '%s' = 'name' ...), unlike the postgres/sqlserver
#                 twins which bind it as $4 / @p4.
#   (iii) NO GUARD     none of the recognised guards appears in that same
#                      function body (see GUARD_RE below).
#
# A function matching (i) AND (ii) AND (iii) is a HIT. Exit 1, file:line printed.
#
# WHY ii-c EXISTS (do not delete it as redundant)
# -----------------------------------------------
# plan.md §6.1 asserts the CASE WHEN sort ladders "bind the sort column as a
# query parameter and never interpolate it, so neither limb of (ii) fires".
# That is true for postgres ($4) and for sqlserver (@p4) — and FALSE for mysql:
# contrib/mysql/internal/adapter/subscription/{subscription,license,
# license_history}.go build the identical ladder with fmt.Sprintf and
# CASE WHEN '%s' = 'name', i.e. the request field lands inside a single-quoted
# SQL string literal. Those three are fail-closed by a
# slices.Contains(<entity>SortableSQLCols) whitelist, so they are not
# injectable — but a gate blind to the whole *shape* would rubber-stamp the
# next port that copies the ladder and forgets the whitelist. ii-c makes the
# gate see them; the whitelist guard is what clears them.
#
# BASELINE (DERIVED BY RUNNING THIS CLASSIFIER — not copied from any doc)
# -----------------------------------------------------------------------
# Reproduce with:  bash scripts/audit-sort-interpolation.sh --rev 756a27e3
#
#   756a27e3 is c5a2573f^ — the last espyna commit before the W0-W3 guard port
#   landed (c5a2573f, "Port postgres SQL-injection guard set to mysql and
#   sqlserver trees"). 923df036 (the older candidate) yields byte-identical
#   numbers: the only contrib/ change between the two commits is
#   contrib/postgres operation/{outcome_matrix_query,task_outcome}.go, which
#   carries no sort interpolation. 756a27e3 is used because it is the tightest
#   pre-W0 point.
#
#                       postgres   mysql   sqlserver   total
#   pre-fix @756a27e3       0        14        6         20     -> exit 1
#   pre-fix @923df036       0        14        6         20     -> exit 1
#   post W0-W3.5 (HEAD+wt)  0         0        0          0     -> exit 0
#
# The "21" in the audit report (14 mysql + 7 sqlserver) is NOT this number and
# must not be used to calibrate: its sqlserver row double-counted
# product/{collection_attribute,resource}.go (already map-clamped, so
# parity migrations rather than live holes) and omitted entity/location.go
# (a live unguarded site). Measured composition of the 6:
# entity/location.go, expenditure/expenditure.go, and
# revenue/{deferred_revenue,revenue_attribute,revenue_category,
# revenue_line_item}.go — which is exactly plan.md §6.1's corrected table.
#
# NEGATIVE CONTROLS (run 2026-07-31 against a scratch copy of the clean tree;
# each was expected to fire, and did):
#   1. restore 756a27e3:mysql/treasury/loan.go into the fixed tree -> caught
#      (limb ii-a/ii-b: the plain "+ sortField +" / "ORDER BY %s" regression)
#   2. delete only the slices.Contains clamp from
#      mysql/subscription/subscription.go -> caught (limb ii-c: proves the
#      CASE WHEN '%s' ladder is visible to the gate, not silently exempt)
#   3. restore 756a27e3:sqlserver/entity/location.go, which DOES contain a
#      ValidateSortColumns call — in the sibling ListLocations method -> caught
#      (proves rule (iii) is function-scoped, not file-scoped; a file-scoped
#      classifier is exactly what hid this site from the original audit)
#
# EXEMPTIONS
# ----------
# There is no allowlist file and there must never be one. The only way a
# function is cleared is by containing a real guard (GUARD_RE). GUARD_RE was
# derived empirically (see the block above its definition): start from the
# three canonical guards, run, and add only what a real function needs.
#
# LIMITATIONS (know these before trusting a green run)
# ----------------------------------------------------
# - Function boundaries are gofmt-shaped: a top-level `func ` at column 0,
#   closed by `}` at column 0. Verified against every scanned file — no raw
#   SQL string in contrib/*/internal/adapter puts `func ` or a bare `}` at
#   column 0 — but a hand-formatted file could fool it.
# - Whole-line `//` and SQL `--` comments are skipped in BOTH directions, so a
#   comment can neither trip the gate nor CLEAR a function. That is deliberate:
#   plan §3.3 / docs/wiki/articles/code-comment-provenance-rule.md — a false
#   safety comment must not be able to launder an unguarded function.
# - The gate proves a guard is *present in the function*, not that it is
#   *correct*. It cannot tell a fail-closed clamp from a fail-open one
#   (e.g. userSortAllowlist[] silently keeps the default on an unknown column,
#   while slices.Contains(...SortableSQLCols) errors). Both clear the gate.
# - Only sort/filter IDENTIFIER interpolation is in scope. Bound VALUES,
#   LIKE-wildcard widening (INFO-1), and table names (audit-table-names.sh)
#   are other gates' business.
# - Depends on bash + find + grep + awk + (for --rev only) git and tar.
#   Portable to the macOS BWK awk that ships with the OS; no gawk required.
#
# USAGE
#   bash packages/espyna-golang/scripts/audit-sort-interpolation.sh
#   bash packages/espyna-golang/scripts/audit-sort-interpolation.sh --rev <sha>
#   bash packages/espyna-golang/scripts/audit-sort-interpolation.sh --root <dir>
#   bash packages/espyna-golang/scripts/audit-sort-interpolation.sh --verbose
#
# Exit 0 = clean. Exit 1 = unguarded hit(s). Exit 2 = usage/environment error.
#
# NOT wired into CI by deliberate decision (W4, 2026-07-31) — run it manually
# beside scripts/audit-table-names.sh.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ESPYNA_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

SCAN_ROOT="$ESPYNA_ROOT"
REV=""
VERBOSE=0

while [ $# -gt 0 ]; do
  case "$1" in
    --rev)  REV="${2:-}"; [ -n "$REV" ] || { echo "--rev needs a commit-ish" >&2; exit 2; }; shift 2 ;;
    --root) SCAN_ROOT="${2:-}"; [ -d "$SCAN_ROOT" ] || { echo "--root: no such dir" >&2; exit 2; }; shift 2 ;;
    --verbose|-v) VERBOSE=1; shift ;;
    --help|-h)
      awk 'NR>1 && /^#/ {sub(/^# ?/, ""); print; next} NR>1 {exit}' "${BASH_SOURCE[0]}"
      exit 0 ;;
    *) echo "unknown arg: $1 (try --help)" >&2; exit 2 ;;
  esac
done

if [ -t 1 ]; then
  C_RED=$'\033[31m'; C_GREEN=$'\033[32m'; C_YEL=$'\033[33m'; C_CYAN=$'\033[36m'; C_DIM=$'\033[2m'; C_OFF=$'\033[0m'
else
  C_RED=''; C_GREEN=''; C_YEL=''; C_CYAN=''; C_DIM=''; C_OFF=''
fi

TMPD="$(mktemp -d)"
trap 'rm -rf "$TMPD"' EXIT

# --rev: materialise the adapter trees from history so the gate can be
# calibrated against a known-bad state instead of trusting a number in a doc.
if [ -n "$REV" ]; then
  cd "$ESPYNA_ROOT"
  git rev-parse --verify "$REV^{commit}" >/dev/null 2>&1 || { echo "no such commit: $REV" >&2; exit 2; }
  mkdir -p "$TMPD/tree"
  git archive "$REV" contrib | tar -x -C "$TMPD/tree"
  SCAN_ROOT="$TMPD/tree"
  printf "%s=== scanning %s @ %s ===%s\n" "$C_CYAN" "contrib" "$(git rev-parse --short "$REV")" "$C_OFF"
else
  printf "%s=== scanning %s (working tree) ===%s\n" "$C_CYAN" "$SCAN_ROOT/contrib" "$C_OFF"
fi

cd "$SCAN_ROOT"
[ -d contrib ] || { echo "no contrib/ under $SCAN_ROOT" >&2; exit 2; }

# ---------------------------------------------------------------------------
# GUARD_RE — rule (iii). MEASURED, not asserted: this list was derived by
# starting from {BuildOrderBy, ValidateSortColumns, ValidateSQLIdent} and
# adding only the shapes that a run against the post-W3.5 working tree proved
# are still load-bearing (2026-07-31). Every alternative below clears at least
# one real function today; drop any one of them and the gate false-positives.
#
#   BuildOrderBy(          the canonical guard (whitelist + per-component
#                          quoting + id tiebreaker). All three trees.
#   ValidateSortColumns(   espynahttp adapter-layer backstop (plan §3.4).
#   ValidateSQLIdent(      core identifier regex — the filter-builder guard.
#   slices.Contains(<X>SortableSQLCols
#                          fail-closed whitelist clamp (errors on unknown).
#                          Clears mysql/subscription/{subscription,license,
#                          license_history}.go and
#                          sqlserver/subscription/{subscription,license}.go —
#                          the D-7/T-3 postgres-parity CASE WHEN skips.
#   allowedSortFields[     map clamp. Clears sqlserver/product/collection.go
#                          (bound-@p4 twin of postgres product/collection.go).
#   <X>SortAllowlist[      map clamp. Clears postgres/entity/user.go and
#                          {postgres,mysql,sqlserver}/entity/workspace_user.go.
#                          plan §6.1 claims this spelling "appears nowhere in
#                          the tree" — that is WRONG: userSortAllowlist[ and
#                          workspaceUserSortAllowlist[ are live in 4 files.
#   switch sort<X> {       closed switch with a default arm. Clears
#                          {mysql,sqlserver}/subscription/invoice.go.
#   range <X>SortableSQLCols
#                          manual allowlist loop. Clears
#                          sqlserver/subscription/{price_plan,price_schedule}.go.
#
# Deliberately ABSENT — measured to be unnecessary, so it stays out:
#   <entity>ViewToSQLColMap[   a view-key REMAP, not a guard. Alone it fails
#                              OPEN (an unmapped key passes straight through).
#                              In the tree it is always paired with a
#                              slices.Contains clamp, which is what actually
#                              clears those functions. Adding it would widen
#                              the gate for nothing.
#
# There is no per-file allowlist and there must never be one: the only way to
# clear a function is to contain a guard.
#
# NOTE: every metacharacter is written as a bracket expression ([(] , [.] , [[])
# rather than a backslash escape — awk -v processes escape sequences in the
# assignment, so a backslash would be eaten before the regex engine sees it
# (macOS/BWK awk fails with "nonterminated character class").
# ---------------------------------------------------------------------------
GUARD_RE='BuildOrderBy[(]'
GUARD_RE="$GUARD_RE"'|ValidateSortColumns[(]'
GUARD_RE="$GUARD_RE"'|ValidateSQLIdent[(]'
GUARD_RE="$GUARD_RE"'|slices[.]Contains[(][A-Za-z_]*[Ss]ortableSQLCols'
GUARD_RE="$GUARD_RE"'|allowedSortFields[[]'
GUARD_RE="$GUARD_RE"'|[A-Za-z_]*SortAllowlist[[]'
GUARD_RE="$GUARD_RE"'|switch[ \t]+sort[A-Za-z]*[ \t]*[{]'
GUARD_RE="$GUARD_RE"'|range[ \t]+[A-Za-z_]*SortableSQLCols'

HITS="$TMPD/hits.txt"; : > "$HITS"
SAFE="$TMPD/safe.txt"; : > "$SAFE"

# Scan every contrib/*/internal/adapter tree (self-updating: a future dialect
# port is covered the day it lands), excluding _test.go.
while IFS= read -r -d '' f; do
  awk -v FILE="$f" -v GUARD_RE="$GUARD_RE" -v SAFE="$SAFE" '
  function report(   tag) {
    if (fname == "" ) return
    if (a_line > 0 && i_line > 0) {
      if (g_line > 0) {
        printf "%s:%d: %s [guarded @%d]\n", FILE, i_line, fname, g_line >> SAFE
      } else {
        printf "%s:%d: %s  (assign@%d, interpolation@%d, no guard in function)\n", \
               FILE, i_line, fname, a_line, i_line
      }
    }
  }
  BEGIN {
    # (i) assignment from a request sort / filter field
    RE_ASSIGN = "=[ \t]*req[.](Get)?Sort|=[ \t]*filter[.]Field"
    # (ii-a) concatenation of a sort-ish variable into query text
    RE_INT_A  = "[+][ \t]*(sort|order)[A-Za-z]*[ \t]*[+]"
    # (ii-b) %s verb straight after ORDER BY, any dialect quoting
    RE_INT_B  = "ORDER BY[ \t]*[\"[`]?%s"
    # (ii-c) %s verb inside a CASE WHEN sort ladder
    RE_INT_C  = "CASE WHEN.*%s"
    infunc = 0; fname = ""; a_line = 0; i_line = 0; g_line = 0
  }
  {
    line = $0
    # Function boundaries: gofmt guarantees a top-level func decl starts at
    # column 0 and its body closes with a "}" at column 0.
    if (line ~ /^func /) {
      report()
      infunc = 1; a_line = 0; i_line = 0; g_line = 0
      fname = line
      sub(/^func +/, "", fname); sub(/[({].*$/, "", fname)
      if (fname == "") { fname = line }
      # a method receiver leaves "(r *repo) Name" — keep the tail
      gsub(/^[^)]*\) */, "", fname)
      sub(/ *$/, "", fname)
      if (fname == "") fname = "(anon)"
      next
    }
    if (infunc && line ~ /^}/) { report(); infunc = 0; fname = ""; next }
    if (!infunc) next

    # Skip whole-line comments in BOTH directions. A comment must never be
    # able to *clear* a function (plan §3.3: a false safety comment turns a
    # reviewer 2 due diligence into a rubber stamp), and must never be able to
    # trip the gate either.
    t = line; sub(/^[ \t]+/, "", t)
    if (substr(t, 1, 2) == "//") next
    if (substr(t, 1, 2) == "--") next   # SQL comment inside a raw string

    if (a_line == 0 && line ~ RE_ASSIGN)  a_line = FNR
    if (i_line == 0 && (line ~ RE_INT_A || line ~ RE_INT_B || line ~ RE_INT_C)) i_line = FNR
    if (g_line == 0 && line ~ GUARD_RE)   g_line = FNR
  }
  END { report() }
  ' "$f" >> "$HITS"
done < <(find contrib -type d -path '*/internal/adapter' -print0 2>/dev/null | while IFS= read -r -d '' d; do
    find "$d" -type f -name '*.go' ! -name '*_test.go' -print0
  done)

# Per-tree tallies, so the output is directly comparable to the plan table.
tally() {
  local tree="$1"
  grep -c "^contrib/$tree/" "$HITS" 2>/dev/null || true
}

printf "\n%s%-12s %8s %8s%s\n" "$C_CYAN" "tree" "unguarded" "guarded" "$C_OFF"
total=0
for d in $(find contrib -maxdepth 1 -mindepth 1 -type d | sed 's#contrib/##' | sort); do
  [ -d "contrib/$d/internal/adapter" ] || continue
  n=$(grep -c "^contrib/$d/" "$HITS" 2>/dev/null || true); n=${n:-0}
  s=$(grep -c "^contrib/$d/" "$SAFE" 2>/dev/null || true); s=${s:-0}
  [ "$n" -eq 0 ] && [ "$s" -eq 0 ] && continue
  if [ "$n" -gt 0 ]; then col="$C_RED"; else col="$C_GREEN"; fi
  printf "%-12s %s%8s%s %s%8s%s\n" "$d" "$col" "$n" "$C_OFF" "$C_DIM" "$s" "$C_OFF"
  total=$((total + n))
done
printf "%-12s %8s\n" "TOTAL" "$total"

if [ "$VERBOSE" -eq 1 ] && [ -s "$SAFE" ]; then
  printf "\n%sGuarded (cleared by rule iii):%s\n" "$C_DIM" "$C_OFF"
  sort "$SAFE" | while IFS= read -r l; do printf "  %s%s%s\n" "$C_DIM" "$l" "$C_OFF"; done
fi

if [ "$total" -eq 0 ]; then
  printf "\n%sclean — zero unguarded sort-interpolation sites%s\n" "$C_GREEN" "$C_OFF"
  exit 0
fi

printf "\n%sUnguarded sort interpolation (file:line):%s\n" "$C_YEL" "$C_OFF"
sort -t: -k1,1 -k2,2n "$HITS" | while IFS= read -r l; do
  printf "  %s%s%s\n" "$C_RED" "$l" "$C_OFF"
done

printf "\n%s%s unguarded site(s). Route the caller-supplied sort column through core.BuildOrderBy(<entity>SortableSQLCols, req.GetSort(), \"<fallback>\") — see contrib/postgres/internal/adapter/entity/location.go for the reference shape.%s\n" \
  "$C_RED" "$total" "$C_OFF" >&2
exit 1
