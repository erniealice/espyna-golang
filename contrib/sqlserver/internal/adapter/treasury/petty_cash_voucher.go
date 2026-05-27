//go:build sqlserver

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pettycashvoucherpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_voucher"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.PettyCashVoucher, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver petty_cash_voucher repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerPettyCashVoucherRepository(dbOps, tableName), nil
	})
}

// SQLServerPettyCashVoucherRepository implements petty_cash_voucher CRUD operations using SQL Server.
type SQLServerPettyCashVoucherRepository struct {
	pettycashvoucherpb.UnimplementedPettyCashVoucherDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerPettyCashVoucherRepository creates a new SQL Server petty_cash_voucher repository.
func NewSQLServerPettyCashVoucherRepository(dbOps interfaces.DatabaseOperation, tableName string) pettycashvoucherpb.PettyCashVoucherDomainServiceServer {
	if tableName == "" {
		tableName = "petty_cash_voucher"
	}

	var db *sql.DB
	if ep, ok := dbOps.(executorProvider); ok {
		if rawDB, ok2 := ep.GetExecutor(context.Background()).(*sql.DB); ok2 {
			db = rawDB
		}
	}

	return &SQLServerPettyCashVoucherRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePettyCashVoucher creates a new petty_cash_voucher record.
func (r *SQLServerPettyCashVoucherRepository) CreatePettyCashVoucher(ctx context.Context, req *pettycashvoucherpb.CreatePettyCashVoucherRequest) (*pettycashvoucherpb.CreatePettyCashVoucherResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("petty_cash_voucher data is required")
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
		return nil, fmt.Errorf("failed to create petty_cash_voucher: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pettyCashVoucher := &pettycashvoucherpb.PettyCashVoucher{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashVoucher); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pettycashvoucherpb.CreatePettyCashVoucherResponse{
		Success: true,
		Data:    []*pettycashvoucherpb.PettyCashVoucher{pettyCashVoucher},
	}, nil
}

// ReadPettyCashVoucher retrieves a petty_cash_voucher record by ID.
func (r *SQLServerPettyCashVoucherRepository) ReadPettyCashVoucher(ctx context.Context, req *pettycashvoucherpb.ReadPettyCashVoucherRequest) (*pettycashvoucherpb.ReadPettyCashVoucherResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_voucher ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read petty_cash_voucher: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pettyCashVoucher := &pettycashvoucherpb.PettyCashVoucher{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashVoucher); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pettycashvoucherpb.ReadPettyCashVoucherResponse{
		Success: true,
		Data:    []*pettycashvoucherpb.PettyCashVoucher{pettyCashVoucher},
	}, nil
}

// UpdatePettyCashVoucher updates a petty_cash_voucher record.
func (r *SQLServerPettyCashVoucherRepository) UpdatePettyCashVoucher(ctx context.Context, req *pettycashvoucherpb.UpdatePettyCashVoucherRequest) (*pettycashvoucherpb.UpdatePettyCashVoucherResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_voucher ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update petty_cash_voucher: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	pettyCashVoucher := &pettycashvoucherpb.PettyCashVoucher{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashVoucher); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &pettycashvoucherpb.UpdatePettyCashVoucherResponse{
		Success: true,
		Data:    []*pettycashvoucherpb.PettyCashVoucher{pettyCashVoucher},
	}, nil
}

// DeletePettyCashVoucher soft-deletes a petty_cash_voucher record.
func (r *SQLServerPettyCashVoucherRepository) DeletePettyCashVoucher(ctx context.Context, req *pettycashvoucherpb.DeletePettyCashVoucherRequest) (*pettycashvoucherpb.DeletePettyCashVoucherResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_voucher ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete petty_cash_voucher: %w", err)
	}

	return &pettycashvoucherpb.DeletePettyCashVoucherResponse{Success: true}, nil
}

// ListPettyCashVouchers lists petty_cash_voucher records with optional filters.
func (r *SQLServerPettyCashVoucherRepository) ListPettyCashVouchers(ctx context.Context, req *pettycashvoucherpb.ListPettyCashVouchersRequest) (*pettycashvoucherpb.ListPettyCashVouchersResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list petty_cash_vouchers: %w", err)
	}

	var pettyCashVouchers []*pettycashvoucherpb.PettyCashVoucher
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal petty_cash_voucher row: %v", err)
			continue
		}

		pettyCashVoucher := &pettycashvoucherpb.PettyCashVoucher{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashVoucher); err != nil {
			log.Printf("WARN: protojson unmarshal petty_cash_voucher: %v", err)
			continue
		}
		pettyCashVouchers = append(pettyCashVouchers, pettyCashVoucher)
	}

	return &pettycashvoucherpb.ListPettyCashVouchersResponse{
		Success: true,
		Data:    pettyCashVouchers,
	}, nil
}

// GetPettyCashVoucherListPageData retrieves petty_cash_vouchers with pagination, filtering, sorting, and search.
// Note: petty_cash_voucher has no direct workspace_id column; workspace isolation is via fund_id.
// The WHERE clause mirrors the postgres gold standard.
//
// SQL Server differences:
//   - LIKE instead of ILIKE.
//   - @pN placeholders.
//   - OFFSET/FETCH pagination.
func (r *SQLServerPettyCashVoucherRepository) GetPettyCashVoucherListPageData(
	ctx context.Context,
	req *pettycashvoucherpb.GetPettyCashVoucherListPageDataRequest,
) (*pettycashvoucherpb.GetPettyCashVoucherListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get petty_cash_voucher list page data request is required")
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

	sortColKey := "pcv.date_created"
	if req.Sort != nil && len(req.Sort.Fields) > 0 && req.Sort.Fields[0].Field != "" {
		sortColKey = req.Sort.Fields[0].Field
	}

	sortDir := commonpb.SortDirection_DESC
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortDir = req.Sort.Fields[0].Direction
	}

	pettyCashVoucherSortableSQLCols := []string{
		"pcv.date_created", "pcv.voucher_number", "pcv.total_amount", "pcv.status",
	}

	orderByClause, err := sqlserverCore.BuildOrderBy(
		pettyCashVoucherSortableSQLCols,
		&commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: sortColKey, Direction: sortDir}}},
		"pcv.date_created DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("invalid sort column for petty_cash_voucher: %w", err)
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				pcv.id,
				pcv.date_created,
				pcv.fund_id,
				pcv.voucher_number,
				pcv.payee,
				pcv.description,
				pcv.total_amount,
				pcv.status,
				pcv.approved_by,
				pcv.approved_at
			FROM petty_cash_voucher pcv
			WHERE (@p1 = '' OR
			       pcv.voucher_number LIKE @p1 OR
			       pcv.description LIKE @p1 OR
			       pcv.payee LIKE @p1)
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s OFFSET @p2 ROWS FETCH NEXT @p3 ROWS ONLY;
	`, orderByClause)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, searchPattern, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query petty_cash_voucher list page data: %w", err)
	}
	defer rows.Close()

	var pettyCashVouchers []*pettycashvoucherpb.PettyCashVoucher
	var totalCount int64

	for rows.Next() {
		var (
			id            string
			dateCreated   int64
			fundID        string
			voucherNumber string
			payee         *string
			description   string
			totalAmount   int64
			status        *string
			approvedBy    *string
			approvedAt    *int64
			total         int64
		)

		if err := rows.Scan(
			&id,
			&dateCreated,
			&fundID,
			&voucherNumber,
			&payee,
			&description,
			&totalAmount,
			&status,
			&approvedBy,
			&approvedAt,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan petty_cash_voucher row: %w", err)
		}

		totalCount = total

		pettyCashVoucher := &pettycashvoucherpb.PettyCashVoucher{
			Id:            id,
			FundId:        fundID,
			VoucherNumber: voucherNumber,
			Description:   description,
			TotalAmount:   totalAmount,
		}

		if status != nil {
			if val, ok := pettycashvoucherpb.VoucherStatus_value[*status]; ok {
				pettyCashVoucher.Status = pettycashvoucherpb.VoucherStatus(val)
			}
		}
		if payee != nil {
			pettyCashVoucher.Payee = payee
		}
		if approvedBy != nil {
			pettyCashVoucher.ApprovedBy = approvedBy
		}
		if approvedAt != nil && *approvedAt > 0 {
			pettyCashVoucher.ApprovedAt = approvedAt
		}

		if dateCreated > 0 {
			pettyCashVoucher.DateCreated = &dateCreated
		}

		pettyCashVouchers = append(pettyCashVouchers, pettyCashVoucher)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating petty_cash_voucher rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &pettycashvoucherpb.GetPettyCashVoucherListPageDataResponse{
		PettyCashVoucherList: pettyCashVouchers,
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

// GetPettyCashVoucherItemPageData retrieves a single petty_cash_voucher.
//
// SQL Server: TOP 1 instead of LIMIT 1; @p1 instead of $1.
func (r *SQLServerPettyCashVoucherRepository) GetPettyCashVoucherItemPageData(
	ctx context.Context,
	req *pettycashvoucherpb.GetPettyCashVoucherItemPageDataRequest,
) (*pettycashvoucherpb.GetPettyCashVoucherItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get petty_cash_voucher item page data request is required")
	}
	if req.PettyCashVoucherId == "" {
		return nil, fmt.Errorf("petty_cash_voucher ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				pcv.id,
				pcv.date_created,
				pcv.fund_id,
				pcv.voucher_number,
				pcv.payee,
				pcv.description,
				pcv.total_amount,
				pcv.status,
				pcv.approved_by,
				pcv.approved_at
			FROM petty_cash_voucher pcv
			WHERE pcv.id = @p1
		)
		SELECT TOP 1 * FROM enriched;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.PettyCashVoucherId)

	var (
		id            string
		dateCreated   int64
		fundID        string
		voucherNumber string
		payee         *string
		description   string
		totalAmount   int64
		status        *string
		approvedBy    *string
		approvedAt    *int64
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&fundID,
		&voucherNumber,
		&payee,
		&description,
		&totalAmount,
		&status,
		&approvedBy,
		&approvedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("petty_cash_voucher with ID '%s' not found", req.PettyCashVoucherId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query petty_cash_voucher item page data: %w", err)
	}

	pettyCashVoucher := &pettycashvoucherpb.PettyCashVoucher{
		Id:            id,
		FundId:        fundID,
		VoucherNumber: voucherNumber,
		Description:   description,
		TotalAmount:   totalAmount,
	}

	if status != nil {
		if val, ok := pettycashvoucherpb.VoucherStatus_value[*status]; ok {
			pettyCashVoucher.Status = pettycashvoucherpb.VoucherStatus(val)
		}
	}
	if payee != nil {
		pettyCashVoucher.Payee = payee
	}
	if approvedBy != nil {
		pettyCashVoucher.ApprovedBy = approvedBy
	}
	if approvedAt != nil && *approvedAt > 0 {
		pettyCashVoucher.ApprovedAt = approvedAt
	}

	if dateCreated > 0 {
		pettyCashVoucher.DateCreated = &dateCreated
	}

	return &pettycashvoucherpb.GetPettyCashVoucherItemPageDataResponse{
		PettyCashVoucher: pettyCashVoucher,
		Success:          true,
	}, nil
}

// NewPettyCashVoucherRepository creates a new SQL Server petty_cash_voucher repository (old-style constructor).
func NewPettyCashVoucherRepository(db *sql.DB, tableName string) pettycashvoucherpb.PettyCashVoucherDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerPettyCashVoucherRepository(dbOps, tableName)
}
