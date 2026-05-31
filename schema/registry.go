package schema

import "sort"

// Registry is the dialect-neutral store of column truth (Q-DD2). It wraps a map
// keyed by RESOLVED table name (the snake_case message name, or the
// (options.v1.table).table_name override when present) to the classified column
// set for that table. It contains NO SQL.
//
// The package-level singleton (Global) is built once by Build() and consumed
// read-only by the postgres operations layer (column-knowledge source) and the
// per-dialect boot-shot validator (drift reconcile).
type Registry struct {
	tables map[string][]ColumnInfo
}

// NewRegistry returns an empty Registry ready to be populated by Build().
func NewRegistry() *Registry {
	return &Registry{tables: make(map[string][]ColumnInfo)}
}

// put stores the classified column set under a resolved table name. Used by
// Build() during the protoregistry walk.
func (r *Registry) put(table string, cols []ColumnInfo) {
	r.tables[table] = cols
}

// ColsFor returns the column set for a resolved table name and whether it is
// known to the registry.
func (r *Registry) ColsFor(table string) ([]ColumnInfo, bool) {
	cols, ok := r.tables[table]
	return cols, ok
}

// ColByName returns a single column descriptor by table + column name. This is the
// feed for autoTimestampValue's type decision in operations.go (bigint-millis vs
// Timestamp) without re-reading information_schema.
func (r *Registry) ColByName(table, col string) (ColumnInfo, bool) {
	cols, ok := r.tables[table]
	if !ok {
		return ColumnInfo{}, false
	}
	for _, c := range cols {
		if c.Name == col {
			return c, true
		}
	}
	return ColumnInfo{}, false
}

// Tables returns the sorted list of resolved table names known to the registry.
// The boot-shot reconcile iterates this to compare against the live schema.
func (r *Registry) Tables() []string {
	out := make([]string, 0, len(r.tables))
	for t := range r.tables {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// Len reports the number of registered tables. Used by Build()'s boot assertion.
func (r *Registry) Len() int {
	return len(r.tables)
}

// Global is the package-level singleton, populated by Build() (driven from the
// container's postgres-tagged init path). Access it only after Build() has run.
var Global = NewRegistry()

// ColsFor is the package-level convenience wrapper over Global.ColsFor.
func ColsFor(table string) ([]ColumnInfo, bool) { return Global.ColsFor(table) }

// Tables is the package-level convenience wrapper over Global.Tables.
func Tables() []string { return Global.Tables() }

// ColByName is the package-level convenience wrapper over Global.ColByName.
func ColByName(table, col string) (ColumnInfo, bool) { return Global.ColByName(table, col) }
