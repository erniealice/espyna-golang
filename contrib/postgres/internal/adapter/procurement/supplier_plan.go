//go:build postgresql

package procurement

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	sqlexec "github.com/erniealice/espyna-golang/shared/database/sqlexec"
	supplierplanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_plan"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresSupplierPlanRepository implements supplier_plan CRUD operations using PostgreSQL.
type PostgresSupplierPlanRepository struct {
	supplierplanpb.UnimplementedSupplierPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

type supplierPlanDirectWorkspaceExecutor interface {
	RequireDirectWorkspace(context.Context, string) (string, error)
	GetExecutor(context.Context) sqlexec.DBExecutor
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierPlan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierPlanRepository(dbOps, tableName), nil
	})
}

// NewPostgresSupplierPlanRepository creates a new PostgreSQL supplier plan repository.
func NewPostgresSupplierPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) supplierplanpb.SupplierPlanDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_plan"
	}
	return &PostgresSupplierPlanRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *PostgresSupplierPlanRepository) CreateSupplierPlan(ctx context.Context, req *supplierplanpb.CreateSupplierPlanRequest) (*supplierplanpb.CreateSupplierPlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier plan data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sp := &supplierplanpb.SupplierPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierplanpb.CreateSupplierPlanResponse{Data: []*supplierplanpb.SupplierPlan{sp}}, nil
}

func (r *PostgresSupplierPlanRepository) ReadSupplierPlan(ctx context.Context, req *supplierplanpb.ReadSupplierPlanRequest) (*supplierplanpb.ReadSupplierPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier plan ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sp := &supplierplanpb.SupplierPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierplanpb.ReadSupplierPlanResponse{Data: []*supplierplanpb.SupplierPlan{sp}}, nil
}

func (r *PostgresSupplierPlanRepository) UpdateSupplierPlan(ctx context.Context, req *supplierplanpb.UpdateSupplierPlanRequest) (*supplierplanpb.UpdateSupplierPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier plan ID is required")
	}
	jsonData, err := (protojson.MarshalOptions{EmitDefaultValues: true}).Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update supplier plan: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	sp := &supplierplanpb.SupplierPlan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &supplierplanpb.UpdateSupplierPlanResponse{Data: []*supplierplanpb.SupplierPlan{sp}}, nil
}

func (r *PostgresSupplierPlanRepository) DeleteSupplierPlan(ctx context.Context, req *supplierplanpb.DeleteSupplierPlanRequest) (*supplierplanpb.DeleteSupplierPlanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier plan ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier plan: %w", err)
	}
	return &supplierplanpb.DeleteSupplierPlanResponse{Success: true}, nil
}

func (r *PostgresSupplierPlanRepository) ListSupplierPlans(ctx context.Context, req *supplierplanpb.ListSupplierPlansRequest) (*supplierplanpb.ListSupplierPlansResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier plans: %w", err)
	}
	var items []*supplierplanpb.SupplierPlan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		sp := &supplierplanpb.SupplierPlan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, sp); err != nil {
			continue
		}
		items = append(items, sp)
	}
	return &supplierplanpb.ListSupplierPlansResponse{Data: items}, nil
}

var supplierPlanSortableSQLCols = []string{
	"id", "name", "description", "active", "supplier_id",
	"date_created", "date_modified",
}

func supplierPlanListPageSQL(orderByClause string) string {
	return `SELECT id, name, description, active, supplier_id, date_created, date_modified
	          FROM ` + entityid.SupplierPlan + `
	          WHERE active = true
	            AND workspace_id = $4
	            AND ($1::text IS NULL OR $1::text = '' OR
	                 name ILIKE $1 ESCAPE '\' OR
	                 description ILIKE $1 ESCAPE '\')
	          ` + orderByClause + ` LIMIT $2 OFFSET $3`
}

const supplierPlanItemPageSQL = `SELECT id, name, description, active, supplier_id, date_created, date_modified
	          FROM ` + entityid.SupplierPlan + `
	          WHERE id = $1
	            AND workspace_id = $2`

func (r *PostgresSupplierPlanRepository) GetSupplierPlanListPageData(ctx context.Context, req *supplierplanpb.GetSupplierPlanListPageDataRequest) (*supplierplanpb.GetSupplierPlanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern, err := postgresCore.BoundedContainsSearchPattern(req.GetSearch())
	if err != nil {
		return nil, err
	}
	limit, offset, _, err := postgresCore.BoundedOffsetPagination(req.GetPagination(), 50)
	if err != nil {
		return nil, err
	}
	// Sort — fail-closed against the per-entity whitelist (A2 guard). Route the
	// caller-supplied sort column through core.BuildOrderBy so an unknown column
	// errors instead of being interpolated verbatim into ORDER BY.
	orderByClause, err := postgresCore.BuildOrderBy(supplierPlanSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}
	directOps, ok := r.dbOps.(supplierPlanDirectWorkspaceExecutor)
	if !ok {
		return nil, fmt.Errorf("supplier_plan repository requires direct workspace capability")
	}
	workspaceID, err := directOps.RequireDirectWorkspace(ctx, entityid.SupplierPlan)
	if err != nil {
		return nil, fmt.Errorf("require supplier plan workspace: %w", err)
	}
	rows, err := directOps.GetExecutor(ctx).QueryContext(ctx, supplierPlanListPageSQL(orderByClause), searchPattern, limit, offset, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*supplierplanpb.SupplierPlan
	for rows.Next() {
		var id, name, supplierID string
		var description sql.NullString
		var active bool
		var dateCreated, dateModified time.Time
		if err := rows.Scan(&id, &name, &description, &active, &supplierID, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		sp := &supplierplanpb.SupplierPlan{Id: id, Name: name, Active: active, SupplierId: supplierID}
		if description.Valid {
			sp.Description = &description.String
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			sp.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			sp.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			sp.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			sp.DateModifiedString = &dmStr
		}
		items = append(items, sp)
	}
	return &supplierplanpb.GetSupplierPlanListPageDataResponse{SupplierPlanList: items, Success: true}, nil
}

func (r *PostgresSupplierPlanRepository) GetSupplierPlanItemPageData(ctx context.Context, req *supplierplanpb.GetSupplierPlanItemPageDataRequest) (*supplierplanpb.GetSupplierPlanItemPageDataResponse, error) {
	if req == nil || req.SupplierPlanId == "" {
		return nil, fmt.Errorf("supplier plan ID required")
	}
	directOps, ok := r.dbOps.(supplierPlanDirectWorkspaceExecutor)
	if !ok {
		return nil, fmt.Errorf("supplier_plan repository requires direct workspace capability")
	}
	workspaceID, err := directOps.RequireDirectWorkspace(ctx, entityid.SupplierPlan)
	if err != nil {
		return nil, fmt.Errorf("require supplier plan workspace: %w", err)
	}
	row := directOps.GetExecutor(ctx).QueryRowContext(ctx, supplierPlanItemPageSQL, req.SupplierPlanId, workspaceID)
	var id, name, supplierID string
	var description sql.NullString
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &name, &description, &active, &supplierID, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("supplier plan not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	sp := &supplierplanpb.SupplierPlan{Id: id, Name: name, Active: active, SupplierId: supplierID}
	if description.Valid {
		sp.Description = &description.String
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		sp.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		sp.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		sp.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		sp.DateModifiedString = &dmStr
	}
	return &supplierplanpb.GetSupplierPlanItemPageDataResponse{SupplierPlan: sp, Success: true}, nil
}

// SearchSupplierPlansByName searches active supplier plans by name using ILIKE.
func (r *PostgresSupplierPlanRepository) SearchSupplierPlansByName(ctx context.Context, req *supplierplanpb.SearchSupplierPlansByNameRequest) (*supplierplanpb.SearchSupplierPlansByNameResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("search supplier plans by name request is required")
	}
	pattern, err := postgresCore.BoundedContainsPattern(req.GetQuery())
	if err != nil {
		return nil, fmt.Errorf("bounded supplier plan name search: %w", err)
	}
	limit, err := postgresCore.BoundedQueryLimit(req.GetLimit(), 20, 100)
	if err != nil {
		return nil, fmt.Errorf("bounded supplier plan name limit: %w", err)
	}
	directOps, ok := r.dbOps.(supplierPlanDirectWorkspaceExecutor)
	if !ok {
		return nil, fmt.Errorf("supplier_plan repository requires direct workspace capability")
	}
	workspaceID, err := directOps.RequireDirectWorkspace(ctx, entityid.SupplierPlan)
	if err != nil {
		return nil, fmt.Errorf("require supplier plan workspace: %w", err)
	}
	query := `
		SELECT id, name
		FROM ` + entityid.SupplierPlan + `
		WHERE active = true
			AND workspace_id = $3
			AND ($1::text = '' OR name ILIKE $1 ESCAPE '\')
		ORDER BY name ASC
		LIMIT $2
	`
	rows, err := directOps.GetExecutor(ctx).QueryContext(ctx, query, pattern, limit, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to search supplier plans by name: %w", err)
	}
	defer rows.Close()
	var results []*supplierplanpb.SearchSupplierPlanResult
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("failed to scan search supplier plan row: %w", err)
		}
		results = append(results, &supplierplanpb.SearchSupplierPlanResult{Id: id, Label: name})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search supplier plan rows: %w", err)
	}
	return &supplierplanpb.SearchSupplierPlansByNameResponse{Results: results, Success: true}, nil
}
