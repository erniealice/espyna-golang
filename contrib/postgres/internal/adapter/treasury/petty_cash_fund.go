//go:build postgresql

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pettycashfundpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_fund"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PettyCashFund, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres petty_cash_fund repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPettyCashFundRepository(dbOps, tableName), nil
	})
}

// PostgresPettyCashFundRepository implements petty_cash_fund CRUD operations using PostgreSQL
type PostgresPettyCashFundRepository struct {
	pettycashfundpb.UnimplementedPettyCashFundDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresPettyCashFundRepository creates a new PostgreSQL petty_cash_fund repository
func NewPostgresPettyCashFundRepository(dbOps interfaces.DatabaseOperation, tableName string) pettycashfundpb.PettyCashFundDomainServiceServer {
	if tableName == "" {
		tableName = "petty_cash_fund"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPettyCashFundRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePettyCashFund creates a new petty_cash_fund record
func (r *PostgresPettyCashFundRepository) CreatePettyCashFund(ctx context.Context, req *pettycashfundpb.CreatePettyCashFundRequest) (*pettycashfundpb.CreatePettyCashFundResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("petty_cash_fund data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create petty_cash_fund: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pettyCashFund := &pettycashfundpb.PettyCashFund{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashFund); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pettycashfundpb.CreatePettyCashFundResponse{
		Success: true,
		Data:    []*pettycashfundpb.PettyCashFund{pettyCashFund},
	}, nil
}

// ReadPettyCashFund retrieves a petty_cash_fund record by ID
func (r *PostgresPettyCashFundRepository) ReadPettyCashFund(ctx context.Context, req *pettycashfundpb.ReadPettyCashFundRequest) (*pettycashfundpb.ReadPettyCashFundResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_fund ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read petty_cash_fund: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pettyCashFund := &pettycashfundpb.PettyCashFund{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashFund); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pettycashfundpb.ReadPettyCashFundResponse{
		Success: true,
		Data:    []*pettycashfundpb.PettyCashFund{pettyCashFund},
	}, nil
}

// UpdatePettyCashFund updates a petty_cash_fund record
func (r *PostgresPettyCashFundRepository) UpdatePettyCashFund(ctx context.Context, req *pettycashfundpb.UpdatePettyCashFundRequest) (*pettycashfundpb.UpdatePettyCashFundResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_fund ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update petty_cash_fund: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pettyCashFund := &pettycashfundpb.PettyCashFund{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashFund); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pettycashfundpb.UpdatePettyCashFundResponse{
		Success: true,
		Data:    []*pettycashfundpb.PettyCashFund{pettyCashFund},
	}, nil
}

// DeletePettyCashFund deletes a petty_cash_fund record (soft delete)
func (r *PostgresPettyCashFundRepository) DeletePettyCashFund(ctx context.Context, req *pettycashfundpb.DeletePettyCashFundRequest) (*pettycashfundpb.DeletePettyCashFundResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_fund ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete petty_cash_fund: %w", err)
	}

	return &pettycashfundpb.DeletePettyCashFundResponse{
		Success: true,
	}, nil
}

// ListPettyCashFunds lists petty_cash_fund records with optional filters
func (r *PostgresPettyCashFundRepository) ListPettyCashFunds(ctx context.Context, req *pettycashfundpb.ListPettyCashFundsRequest) (*pettycashfundpb.ListPettyCashFundsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list petty_cash_funds: %w", err)
	}

	var pettyCashFunds []*pettycashfundpb.PettyCashFund
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal petty_cash_fund row: %v", err)
			continue
		}

		pettyCashFund := &pettycashfundpb.PettyCashFund{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashFund); err != nil {
			log.Printf("WARN: protojson unmarshal petty_cash_fund: %v", err)
			continue
		}
		pettyCashFunds = append(pettyCashFunds, pettyCashFund)
	}

	return &pettycashfundpb.ListPettyCashFundsResponse{
		Success: true,
		Data:    pettyCashFunds,
	}, nil
}

// pettyCashFundSortableSQLCols whitelists sortable columns. authorized_amount
// and current_balance are centavo integers — integer sort is correct.
var pettyCashFundSortableSQLCols = []string{
	"id", "date_created", "date_modified", "active", "name",
	"authorized_amount", "current_balance", "custodian_id", "location_id",
}

// GetPettyCashFundListPageData retrieves petty_cash_funds with pagination, filtering, sorting, and search using CTE
func (r *PostgresPettyCashFundRepository) GetPettyCashFundListPageData(
	ctx context.Context,
	req *pettycashfundpb.GetPettyCashFundListPageDataRequest,
) (*pettycashfundpb.GetPettyCashFundListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get petty_cash_fund list page data request is required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Sort — fail-closed against the per-entity whitelist (A2 guard). The ORDER BY
	// runs against the outer `enriched e` projection (unprefixed cols), so the
	// whitelist + fallback are unprefixed.
	orderByClause, err := postgresCore.BuildOrderBy(pettyCashFundSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, err
	}

	query := `
		WITH enriched AS (
			SELECT
				pcf.id,
				pcf.date_created,
				pcf.date_modified,
				pcf.active,
				pcf.name,
				pcf.authorized_amount,
				pcf.current_balance,
				pcf.custodian_id,
				pcf.location_id
			FROM petty_cash_fund pcf
			WHERE pcf.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       pcf.name ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		` + orderByClause + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query petty_cash_fund list page data: %w", err)
	}
	defer rows.Close()

	var pettyCashFunds []*pettycashfundpb.PettyCashFund
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			dateCreated      time.Time
			dateModified     time.Time
			active           bool
			name             string
			authorizedAmount int64
			currentBalance   int64
			custodianID      *string
			locationID       *string
			total            int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&authorizedAmount,
			&currentBalance,
			&custodianID,
			&locationID,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan petty_cash_fund row: %w", err)
		}

		totalCount = total

		pettyCashFund := &pettycashfundpb.PettyCashFund{
			Id:               id,
			Active:           active,
			Name:             name,
			AuthorizedAmount: authorizedAmount,
			CurrentBalance:   currentBalance,
		}

		if custodianID != nil {
			pettyCashFund.CustodianId = custodianID
		}
		if locationID != nil {
			pettyCashFund.LocationId = locationID
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			pettyCashFund.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			pettyCashFund.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			pettyCashFund.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			pettyCashFund.DateModifiedString = &dmStr
		}

		pettyCashFunds = append(pettyCashFunds, pettyCashFund)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating petty_cash_fund rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pettycashfundpb.GetPettyCashFundListPageDataResponse{
		PettyCashFundList: pettyCashFunds,
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

// GetPettyCashFundItemPageData retrieves a single petty_cash_fund with enriched data using CTE
func (r *PostgresPettyCashFundRepository) GetPettyCashFundItemPageData(
	ctx context.Context,
	req *pettycashfundpb.GetPettyCashFundItemPageDataRequest,
) (*pettycashfundpb.GetPettyCashFundItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get petty_cash_fund item page data request is required")
	}
	if req.PettyCashFundId == "" {
		return nil, fmt.Errorf("petty_cash_fund ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				pcf.id,
				pcf.date_created,
				pcf.date_modified,
				pcf.active,
				pcf.name,
				pcf.authorized_amount,
				pcf.current_balance,
				pcf.custodian_id,
				pcf.location_id
			FROM petty_cash_fund pcf
			WHERE pcf.id = $1 AND pcf.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.PettyCashFundId)

	var (
		id               string
		dateCreated      time.Time
		dateModified     time.Time
		active           bool
		name             string
		authorizedAmount int64
		currentBalance   int64
		custodianID      *string
		locationID       *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&authorizedAmount,
		&currentBalance,
		&custodianID,
		&locationID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("petty_cash_fund with ID '%s' not found", req.PettyCashFundId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query petty_cash_fund item page data: %w", err)
	}

	pettyCashFund := &pettycashfundpb.PettyCashFund{
		Id:               id,
		Active:           active,
		Name:             name,
		AuthorizedAmount: authorizedAmount,
		CurrentBalance:   currentBalance,
	}

	if custodianID != nil {
		pettyCashFund.CustodianId = custodianID
	}
	if locationID != nil {
		pettyCashFund.LocationId = locationID
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		pettyCashFund.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		pettyCashFund.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		pettyCashFund.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		pettyCashFund.DateModifiedString = &dmStr
	}

	return &pettycashfundpb.GetPettyCashFundItemPageDataResponse{
		PettyCashFund: pettyCashFund,
		Success:       true,
	}, nil
}

// NewPettyCashFundRepository creates a new PostgreSQL petty_cash_fund repository (old-style constructor)
func NewPettyCashFundRepository(db *sql.DB, tableName string) pettycashfundpb.PettyCashFundDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresPettyCashFundRepository(dbOps, tableName)
}
