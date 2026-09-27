//go:build postgresql

package core

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
)

// Only the first 64 returned IDs are inspected. One SQL query covers that
// sample, so a List page adds at most one round trip regardless of page size.
const columnlessListSampleLimit = 64

var columnlessNoMechanismLogged sync.Map

func shadowColumnlessLog(table, op, reason string) {
	log.Printf("AUTHZ_WS_COLUMNLESS_SHADOW_DENY table=%s op=%s reason=%s", table, op, reason)
}

func shadowNoMechanism(table, op string) {
	if _, loaded := columnlessNoMechanismLogged.LoadOrStore(table, struct{}{}); !loaded {
		shadowColumnlessLog(table, op, "no_mechanism")
	}
}

func (w *WorkspaceAwareOperations) shadowColumnlessByID(ctx context.Context, op, table, id, workspace string) {
	if _, global := declaredGlobal[table]; global {
		return
	}
	chain, registered := columnlessChildChains[table]
	if !registered {
		shadowNoMechanism(table, op)
		return
	}
	resolved, err := w.shadowParentWorkspaces(ctx, table, chain, []string{id})
	if err != nil {
		shadowColumnlessLog(table, op, "unresolved")
		return
	}
	owner, found := resolved[id]
	if !found || owner == "" {
		shadowColumnlessLog(table, op, "unresolved")
	} else if owner != workspace {
		shadowColumnlessLog(table, op, "mismatch")
	}
}

func (w *WorkspaceAwareOperations) shadowColumnlessList(ctx context.Context, table, workspace string, result *interfaces.ListResult) {
	if _, global := declaredGlobal[table]; global {
		return
	}
	chain, registered := columnlessChildChains[table]
	if !registered {
		shadowNoMechanism(table, "list")
		return
	}
	if result == nil || len(result.Data) == 0 {
		return
	}
	ids := make([]string, 0, columnlessListSampleLimit)
	unresolved := false
	for i, row := range result.Data {
		if i == columnlessListSampleLimit {
			break
		}
		if id, ok := shadowRowID(row["id"]); ok {
			ids = append(ids, id)
		} else {
			unresolved = true
		}
	}
	if len(ids) == 0 {
		if unresolved {
			shadowColumnlessLog(table, "list", "unresolved")
		}
		return
	}
	owners, err := w.shadowParentWorkspaces(ctx, table, chain, ids)
	if err != nil {
		shadowColumnlessLog(table, "list", "unresolved")
		return
	}
	mismatch := false
	for _, id := range ids {
		owner, found := owners[id]
		if !found || owner == "" {
			unresolved = true
		} else if owner != workspace {
			mismatch = true
		}
	}
	if mismatch {
		shadowColumnlessLog(table, "list", "mismatch")
	}
	if unresolved {
		shadowColumnlessLog(table, "list", "unresolved")
	}
}

func shadowRowID(value any) (string, bool) {
	switch id := value.(type) {
	case string:
		return id, id != ""
	case []byte:
		return string(id), len(id) != 0
	default:
		return "", false
	}
}

// All SQL identifiers come from the static registry, never request data.
// Values are bound. A missing P3 root column or FK returns an error that the
// caller records as unresolved while preserving the original operation result.
func (w *WorkspaceAwareOperations) shadowParentWorkspaces(ctx context.Context, table, chain string, ids []string) (map[string]string, error) {
	if w.db == nil {
		return nil, fmt.Errorf("database handle unavailable")
	}
	parts := strings.Split(strings.TrimSuffix(chain, "[P3 root required]"), ">")
	if len(parts) < 2 || parts[len(parts)-1] != "workspace_id" {
		return nil, fmt.Errorf("invalid parent chain")
	}
	if !columnlessSQLIdentifier(table) {
		return nil, fmt.Errorf("invalid child table")
	}
	var query strings.Builder
	fmt.Fprintf(&query, "SELECT c.id, p%d.workspace_id FROM %q c", len(parts)-2, table)
	previous := "c"
	for i, step := range parts[:len(parts)-1] {
		fk, parent, ok := strings.Cut(step, ":")
		if !ok || !columnlessSQLIdentifier(fk) || !columnlessSQLIdentifier(parent) {
			return nil, fmt.Errorf("invalid parent chain step")
		}
		alias := fmt.Sprintf("p%d", i)
		fmt.Fprintf(&query, " LEFT JOIN %q %s ON %s.%q = %s.id", parent, alias, previous, fk, alias)
		previous = alias
	}
	args := make([]any, len(ids))
	query.WriteString(" WHERE c.id IN (")
	for i, id := range ids {
		if i > 0 {
			query.WriteByte(',')
		}
		fmt.Fprintf(&query, "$%d", i+1)
		args[i] = id
	}
	query.WriteByte(')')
	rows, err := w.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	owners := make(map[string]string, len(ids))
	for rows.Next() {
		var id string
		var owner sql.NullString
		if err := rows.Scan(&id, &owner); err != nil {
			return nil, err
		}
		if owner.Valid {
			owners[id] = owner.String
		} else {
			owners[id] = ""
		}
	}
	return owners, rows.Err()
}

func columnlessSQLIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != '_' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}
