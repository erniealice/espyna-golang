//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/erniealice/espyna-golang/schema"
)

// schema_validator.go is the postgresql boot-shot schema validator (Plan 2,
// docs/plan/20260530-reflectionless-crud/). It is the ONE retained
// information_schema read — repurposed from the per-call source (operations.go's
// getTableColumns / getTableColumnTypes) into a single startup reconcile.
//
// It reads the live public-schema column set ONCE and reconciles it against the
// dialect-neutral descriptor registry (schema.Global), failing fast on drift
// (Q-DD3=A boot-shot + Q-DD5=A strict). Per-dialect by design: it lives in
// contrib/postgres so the dialect-neutral schema package stays SQL-free and
// mysql/sqlserver get a ~20-line sibling later.
//
// SHADOW MODE (this wave): the validator runs and fails-fast on drift, but it does
// NOT flip operations.go's unknown-column LOG->ERROR. That flip is Phase 4 (a
// runtime soak gate), out of scope here.

// descriptorOutOfScope is the Q-DD5 out-of-scope allowlist: live public-schema
// tables that legitimately have NO table-annotated proto message and therefore are
// NOT in the descriptor registry. A live table absent from BOTH the registry and
// this allowlist is real drift and fails the boot.
//
// Seeded with:
//   - payment_method, revenue_payment — no proto message at all (phase0 §c GAP-B,
//     "no proto message" sub-gap). Design-named allowlist entries.
//   - integration_payment — no proto; raw-SQL writer (phase0 §b adapter/integration/payment.go).
//   - audit_entry, audit_field_change — proto messages exist but carry NO table=true
//     (audit infrastructure, written via raw SQL; phase0 §b adapter/audit/audit_adapter.go).
//     The live partitions live in the audit_trail schema (excluded by the public-schema
//     scan), but the plain names are allowlisted defensively in case a public view exists.
//   - session — service-shaped (proto/v1/service/auth/session.proto, no table=true);
//     written via raw SQL (phase0 §b adapter/entity/workspace.go, session_switch_principal.go).
//
// As the Phase 1 annotation sprint adds table=true to former GAP-B tables, those
// tables leave this list automatically (they become registry-covered). Keep this
// list minimal — every entry is an acknowledged reflectionless-write gap.
var descriptorOutOfScope = map[string]bool{
	"payment_method":      true,
	"revenue_payment":     true,
	"integration_payment": true,
	"audit_entry":         true,
	"audit_field_change":  true,
	"session":             true,
}

// ValidateSchema is the registered postgresql SchemaValidator. It reads the live
// public-schema columns once and reconciles them against schema.Global:
//
//  1. Every live table NOT in the registry must be in descriptorOutOfScope, else
//     ERROR (Q-DD5=A strict boot-shot).
//  2. Every registry table with no live table is SKIP-or-WARN (GAP-A:
//     deferred_revenue / prepayment / security_deposit still carry table=true but
//     were dropped 20260517). Never errors.
//  3. Every registry column absent from its live table is ERROR (descriptor claims
//     a column the DB lacks — a write would fail at runtime).
//
// schema.Build() must have run before this is called (the container wirePoint
// guarantees ordering). ValidateSchema defensively triggers Build() too.
func ValidateSchema(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("schema validator: nil *sql.DB")
	}
	if err := schema.Build(); err != nil {
		return fmt.Errorf("schema validator: registry build failed: %w", err)
	}

	live, err := liveColumns(ctx, db)
	if err != nil {
		return fmt.Errorf("schema validator: reading information_schema: %w", err)
	}

	drift, warnings := reconcile(live, descriptorOutOfScope)

	if len(warnings) > 0 {
		for _, w := range warnings {
			log.Printf("⚠️ schema validator: %s", w)
		}
	}

	if len(drift) > 0 {
		return fmt.Errorf("schema validator: %d drift issue(s) detected (shadow boot-shot, fail-fast):\n  - %s",
			len(drift), strings.Join(drift, "\n  - "))
	}

	log.Printf("✅ schema validator: %d descriptor tables reconciled against the live schema (no drift)",
		len(schema.Tables()))
	return nil
}

// reconcile is the pure (DB-free) core of the boot-shot: it compares the live
// column map against schema.Global and the allowlist, returning sorted drift
// errors (fail-fast) and warnings (skip). Extracted so the three Q-DD5 branches are
// unit-testable without a live database.
//
//	live[table][column] = true   — the public-schema columns read once at boot.
//
// Returns (drift, warnings); a non-empty drift slice means the boot must fail.
func reconcile(live map[string]map[string]bool, allowlist map[string]bool) (drift, warnings []string) {
	// (1) live tables not covered by the registry must be allowlisted.
	for table := range live {
		if _, ok := schema.ColsFor(table); ok {
			continue
		}
		if allowlist[table] {
			continue
		}
		drift = append(drift, fmt.Sprintf(
			"live table %q has no descriptor and is not in descriptorOutOfScope allowlist", table))
	}

	// (2) registry tables with no live table -> SKIP/WARN (dropped GAP-A tables).
	// (3) registry columns absent from the live table -> drift.
	for _, table := range schema.Tables() {
		liveCols, ok := live[table]
		if !ok {
			warnings = append(warnings, fmt.Sprintf(
				"descriptor table %q has no live table (skipped — dropped/deferred?)", table))
			continue
		}
		cols, _ := schema.ColsFor(table)
		for _, c := range cols {
			if !liveCols[c.Name] {
				drift = append(drift, fmt.Sprintf(
					"descriptor table %q claims column %q absent from the live schema", table, c.Name))
			}
		}
	}

	sort.Strings(drift)
	sort.Strings(warnings)
	return drift, warnings
}

// liveColumns reads every public-schema column in a SINGLE query and groups them
// by table name. Reuses the exact SELECT shape from operations.go's
// getTableColumns/getTableColumnTypes (information_schema.columns), but for the
// whole public schema at once rather than per-table — this is the one boot-time
// information_schema read that replaces the per-call round-trips.
func liveColumns(ctx context.Context, db *sql.DB) (map[string]map[string]bool, error) {
	const query = `
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]map[string]bool)
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			return nil, err
		}
		cols, ok := out[table]
		if !ok {
			cols = make(map[string]bool)
			out[table] = cols
		}
		cols[column] = true
	}
	return out, rows.Err()
}
