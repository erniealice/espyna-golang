//go:build postgresql

package payroll

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/consumer"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	ratebandpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payroll/rate_band"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.RateBand, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres rate_band repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresRateBandRepository(dbOps, tableName), nil
	})
}

// PostgresRateBandRepository implements rate band CRUD operations using PostgreSQL.
type PostgresRateBandRepository struct {
	ratebandpb.UnimplementedRateBandDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresRateBandRepository creates a new PostgreSQL rate band repository.
func NewPostgresRateBandRepository(dbOps interfaces.DatabaseOperation, tableName string) ratebandpb.RateBandDomainServiceServer {
	if tableName == "" {
		tableName = "rate_band"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresRateBandRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateRateBand creates a new rate band record.
func (r *PostgresRateBandRepository) CreateRateBand(ctx context.Context, req *ratebandpb.CreateRateBandRequest) (*ratebandpb.CreateRateBandResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("rate band data is required")
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
		return nil, fmt.Errorf("failed to create rate_band: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rb := &ratebandpb.RateBand{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratebandpb.CreateRateBandResponse{Success: true, Data: []*ratebandpb.RateBand{rb}}, nil
}

// ReadRateBand retrieves a rate band by ID.
func (r *PostgresRateBandRepository) ReadRateBand(ctx context.Context, req *ratebandpb.ReadRateBandRequest) (*ratebandpb.ReadRateBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate band ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read rate_band: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rb := &ratebandpb.RateBand{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratebandpb.ReadRateBandResponse{Success: true, Data: []*ratebandpb.RateBand{rb}}, nil
}

// UpdateRateBand updates a rate band record.
func (r *PostgresRateBandRepository) UpdateRateBand(ctx context.Context, req *ratebandpb.UpdateRateBandRequest) (*ratebandpb.UpdateRateBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate band ID is required")
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
		return nil, fmt.Errorf("failed to update rate_band: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rb := &ratebandpb.RateBand{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratebandpb.UpdateRateBandResponse{Success: true, Data: []*ratebandpb.RateBand{rb}}, nil
}

// DeleteRateBand soft-deletes a rate band.
func (r *PostgresRateBandRepository) DeleteRateBand(ctx context.Context, req *ratebandpb.DeleteRateBandRequest) (*ratebandpb.DeleteRateBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("rate band ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete rate_band: %w", err)
	}
	return &ratebandpb.DeleteRateBandResponse{Success: true}, nil
}

// ListRateBands lists rate band records with optional filters.
func (r *PostgresRateBandRepository) ListRateBands(ctx context.Context, req *ratebandpb.ListRateBandsRequest) (*ratebandpb.ListRateBandsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list rate_bands: %w", err)
	}
	var items []*ratebandpb.RateBand
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal rate_band row: %v", err)
			continue
		}
		rb := &ratebandpb.RateBand{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
			log.Printf("WARN: protojson unmarshal rate_band: %v", err)
			continue
		}
		items = append(items, rb)
	}
	return &ratebandpb.ListRateBandsResponse{Success: true, Data: items}, nil
}

// rateBandSortableSQLCols is the A2 sort whitelist for rate_band list pages.
var rateBandSortableSQLCols = []string{
	"rb.id", "rb.rate_table_id", "rb.lower_bound_centavos", "rb.upper_bound_centavos",
	"rb.rate_type", "rb.rate_basis_points", "rb.fixed_amount_centavos",
	"rb.ordinal", "rb.active", "rb.date_created", "rb.date_modified",
}

// GetRateBandListPageData retrieves rate bands with pagination, filtering, sorting, and search.
// A1: scoped via LEFT JOIN rate_table — shows bands belonging to workspace-specific or global rate_tables.
// A2: sort column whitelisted via core.BuildOrderBy.
// A3: COUNT(*) OVER() for accurate total without a second query.
func (r *PostgresRateBandRepository) GetRateBandListPageData(
	ctx context.Context,
	req *ratebandpb.GetRateBandListPageDataRequest,
) (*ratebandpb.GetRateBandListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get rate band list page data request is required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("GetRateBandListPageData requires raw *sql.DB")
	}

	// A1: tenant guard via rate_table join.
	workspaceID := consumer.GetWorkspaceIDFromContext(ctx)

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
	orderByClause, err := postgresCore.BuildOrderBy(rateBandSortableSQLCols, req.GetSort(), "rb.ordinal ASC")
	if err != nil {
		return nil, err
	}

	// A3: COUNT(*) OVER() — accurate total in one pass.
	// rate_band has no workspace_id; tenant scoping is via the parent rate_table.workspace_id
	// (NULL = global, available to all tenants).
	query := fmt.Sprintf(`
		SELECT
			rb.id,
			rb.rate_table_id,
			rb.lower_bound_centavos,
			rb.upper_bound_centavos,
			rb.rate_type,
			rb.rate_basis_points,
			rb.fixed_amount_centavos,
			rb.formula_expression,
			rb.ordinal,
			rb.metadata,
			rb.active,
			rb.date_created,
			rb.date_modified,
			COUNT(*) OVER() AS total
		FROM %s rb
		LEFT JOIN rate_table rt ON rt.id = rb.rate_table_id
		WHERE (rt.workspace_id = $1 OR rt.workspace_id IS NULL)
		%s
		LIMIT $2 OFFSET $3;
	`, r.tableName, orderByClause)

	rows, err := r.db.QueryContext(ctx, query, workspaceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query rate_band list page data: %w", err)
	}
	defer rows.Close()

	var items []*ratebandpb.RateBand
	var totalCount int64

	for rows.Next() {
		var (
			id                  string
			rateTableID         string
			lowerBound          int64
			upperBound          *int64
			rateType            string
			rateBasisPoints     int32
			fixedAmountCentavos int64
			formulaExpression   *string
			ordinal             int32
			metadata            *string
			active              bool
			dateCreated         *int64
			dateModified        *int64
			total               int64
		)
		if scanErr := rows.Scan(
			&id, &rateTableID, &lowerBound, &upperBound,
			&rateType, &rateBasisPoints, &fixedAmountCentavos,
			&formulaExpression, &ordinal, &metadata,
			&active, &dateCreated, &dateModified,
			&total,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan rate_band row: %w", scanErr)
		}
		totalCount = total

		rb := &ratebandpb.RateBand{
			Id:                  id,
			RateTableId:         rateTableID,
			LowerBoundCentavos:  lowerBound,
			UpperBoundCentavos:  upperBound,
			RateType:            rateType,
			RateBasisPoints:     rateBasisPoints,
			FixedAmountCentavos: fixedAmountCentavos,
			FormulaExpression:   formulaExpression,
			Ordinal:             ordinal,
			Metadata:            metadata,
			Active:              active,
			DateCreated:         dateCreated,
			DateModified:        dateModified,
		}
		items = append(items, rb)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rate_band rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &ratebandpb.GetRateBandListPageDataResponse{
		RateBandList: items,
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

// GetRateBandItemPageData retrieves a single rate band.
func (r *PostgresRateBandRepository) GetRateBandItemPageData(
	ctx context.Context,
	req *ratebandpb.GetRateBandItemPageDataRequest,
) (*ratebandpb.GetRateBandItemPageDataResponse, error) {
	if req == nil || req.GetRateBandId() == "" {
		return nil, fmt.Errorf("rate band ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetRateBandId())
	if err != nil {
		return nil, fmt.Errorf("failed to read rate_band item: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	rb := &ratebandpb.RateBand{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rb); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &ratebandpb.GetRateBandItemPageDataResponse{
		RateBand: rb,
		Success:  true,
	}, nil
}
