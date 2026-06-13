//go:build postgresql

package payroll

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	ratetablepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_table"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RateTable, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres rate_table repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresRateTableRepository(dbOps, tableName), nil
	})
}

// PostgresRateTableRepository implements rate table CRUD operations using PostgreSQL.
type PostgresRateTableRepository struct {
	ratetablepb.UnimplementedRateTableDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresRateTableRepository creates a new PostgreSQL rate table repository.
func NewPostgresRateTableRepository(dbOps interfaces.DatabaseOperation, tableName string) ratetablepb.RateTableDomainServiceServer {
	if tableName == "" {
		tableName = "rate_table"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresRateTableRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateRateTable creates a new rate table record.
func (r *PostgresRateTableRepository) CreateRateTable(ctx context.Context, req *ratetablepb.CreateRateTableRequest) (*ratetablepb.CreateRateTableResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("rate table data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate_table: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rt := &ratetablepb.RateTable{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratetablepb.CreateRateTableResponse{Success: true, Data: []*ratetablepb.RateTable{rt}}, nil
}

// ReadRateTable retrieves a rate table by ID.
func (r *PostgresRateTableRepository) ReadRateTable(ctx context.Context, req *ratetablepb.ReadRateTableRequest) (*ratetablepb.ReadRateTableResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate table ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read rate_table: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rt := &ratetablepb.RateTable{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratetablepb.ReadRateTableResponse{Success: true, Data: []*ratetablepb.RateTable{rt}}, nil
}

// UpdateRateTable updates a rate table record.
func (r *PostgresRateTableRepository) UpdateRateTable(ctx context.Context, req *ratetablepb.UpdateRateTableRequest) (*ratetablepb.UpdateRateTableResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate table ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update rate_table: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rt := &ratetablepb.RateTable{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratetablepb.UpdateRateTableResponse{Success: true, Data: []*ratetablepb.RateTable{rt}}, nil
}

// DeleteRateTable soft-deletes a rate table.
func (r *PostgresRateTableRepository) DeleteRateTable(ctx context.Context, req *ratetablepb.DeleteRateTableRequest) (*ratetablepb.DeleteRateTableResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate table ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete rate_table: %w", err)
	}
	return &ratetablepb.DeleteRateTableResponse{Success: true}, nil
}

// ListRateTables lists rate table records with optional filters.
func (r *PostgresRateTableRepository) ListRateTables(ctx context.Context, req *ratetablepb.ListRateTablesRequest) (*ratetablepb.ListRateTablesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list rate_tables: %w", err)
	}
	var items []*ratetablepb.RateTable
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal rate_table row: %v", err)
			continue
		}
		rt := &ratetablepb.RateTable{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
			log.Printf("WARN: protojson unmarshal rate_table: %v", err)
			continue
		}
		items = append(items, rt)
	}
	return &ratetablepb.ListRateTablesResponse{Success: true, Data: items}, nil
}

// rateTableSortableSQLCols is the A2 sort whitelist for rate_table list pages.
var rateTableSortableSQLCols = []string{
	"rt.id", "rt.workspace_id", "rt.compliance_region", "rt.kind",
	"rt.effective_from", "rt.effective_to", "rt.version_label",
	"rt.supersedes_id", "rt.source_citation",
	"rt.active", "rt.date_created", "rt.date_modified",
}

// GetRateTableListPageData retrieves rate tables with pagination, filtering, sorting, and search.
// A1: shows workspace-specific rows (workspace_id = $1) AND global rows (workspace_id IS NULL)
//
//	because rate_table.workspace_id is optional — NULL means a global default for all tenants.
//
// A2: sort column whitelisted via core.BuildOrderBy.
// A3: COUNT(*) OVER() for accurate total without a second query.
func (r *PostgresRateTableRepository) GetRateTableListPageData(
	ctx context.Context,
	req *ratetablepb.GetRateTableListPageDataRequest,
) (*ratetablepb.GetRateTableListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get rate table list page data request is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("GetRateTableListPageData requires raw *sql.DB")
	}

	// A1: show workspace-specific rows plus global (NULL workspace_id) rows.
	workspaceID := identity.Must(ctx).WorkspaceID

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}

	// A2: sort guard — fail-closed via core.BuildOrderBy whitelist.
	orderByClause, err := postgresCore.BuildOrderBy(rateTableSortableSQLCols, req.GetSort(), "rt.effective_from DESC")
	if err != nil {
		return nil, err
	}

	// A3: COUNT(*) OVER() — accurate total in one pass.
	query := fmt.Sprintf(`
		SELECT
			rt.id,
			rt.workspace_id,
			rt.compliance_region,
			rt.kind,
			rt.effective_from,
			rt.effective_to,
			rt.version_label,
			rt.supersedes_id,
			rt.source_citation,
			rt.active,
			rt.date_created,
			rt.date_modified,
			COUNT(*) OVER() AS total
		FROM %s rt
		WHERE (rt.workspace_id = $1 OR rt.workspace_id IS NULL)
		%s
		LIMIT $2 OFFSET $3;
	`, r.tableName, orderByClause)

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query rate_table list page data: %w", err)
	}
	defer rows.Close()

	var items []*ratetablepb.RateTable
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			wsID             *string
			complianceRegion string
			kind             string
			effectiveFrom    string
			effectiveTo      *string
			versionLabel     string
			supersedesID     *string
			sourceCitation   string
			active           bool
			dateCreated      *int64
			dateModified     *int64
			total            int64
		)
		if scanErr := rows.Scan(
			&id, &wsID, &complianceRegion, &kind,
			&effectiveFrom, &effectiveTo, &versionLabel,
			&supersedesID, &sourceCitation,
			&active, &dateCreated, &dateModified,
			&total,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan rate_table row: %w", scanErr)
		}
		totalCount = total

		rt := &ratetablepb.RateTable{
			Id:               id,
			WorkspaceId:      wsID,
			ComplianceRegion: complianceRegion,
			Kind:             kind,
			EffectiveFrom:    effectiveFrom,
			EffectiveTo:      effectiveTo,
			VersionLabel:     versionLabel,
			SupersedesId:     supersedesID,
			SourceCitation:   sourceCitation,
			Active:           active,
			DateCreated:      dateCreated,
			DateModified:     dateModified,
		}
		items = append(items, rt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rate_table rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &ratetablepb.GetRateTableListPageDataResponse{
		RateTableList: items,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetRateTableItemPageData retrieves a single rate table.
func (r *PostgresRateTableRepository) GetRateTableItemPageData(
	ctx context.Context,
	req *ratetablepb.GetRateTableItemPageDataRequest,
) (*ratetablepb.GetRateTableItemPageDataResponse, error) {
	if req == nil || req.GetRateTableId() == "" {
		return nil, fmt.Errorf("rate table ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetRateTableId())
	if err != nil {
		return nil, fmt.Errorf("failed to read rate_table item: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rt := &ratetablepb.RateTable{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratetablepb.GetRateTableItemPageDataResponse{
		RateTable: rt,
		Success:   true,
	}, nil
}
