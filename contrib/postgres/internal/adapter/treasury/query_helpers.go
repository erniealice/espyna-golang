//go:build postgresql

package treasury

import (
	"context"
	"database/sql"
	"fmt"

	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

type directWorkspaceResolver interface {
	RequireDirectWorkspace(context.Context, string) (string, error)
}

func requireTreasuryWorkspace(ctx context.Context, dbOps interfaces.DatabaseOperation, tableName string) (string, error) {
	resolver, ok := dbOps.(directWorkspaceResolver)
	if !ok {
		return "", fmt.Errorf("treasury repository requires direct workspace capability")
	}
	return resolver.RequireDirectWorkspace(ctx, tableName)
}

func requireTreasuryRawDB(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("treasury repository requires a PostgreSQL connection")
	}
	return nil
}

// mapTreasurySortRequest copies a public sort request and replaces only fields
// present in an adapter-owned map. Unknown fields remain unchanged so the
// downstream BuildOrderBy allowlist rejects them fail-closed.
func mapTreasurySortRequest(in *commonpb.SortRequest, fieldMap map[string]string) *commonpb.SortRequest {
	if in == nil {
		return nil
	}
	out := &commonpb.SortRequest{Fields: make([]*commonpb.SortField, 0, len(in.Fields))}
	for _, field := range in.Fields {
		if field == nil {
			out.Fields = append(out.Fields, nil)
			continue
		}
		mapped := *field
		if sqlField, ok := fieldMap[field.GetField()]; ok {
			mapped.Field = sqlField
		}
		out.Fields = append(out.Fields, &mapped)
	}
	return out
}
