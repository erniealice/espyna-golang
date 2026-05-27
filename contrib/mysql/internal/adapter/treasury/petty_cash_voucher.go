//go:build mysql

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pettycashvoucherpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_voucher"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.PettyCashVoucher, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql petty_cash_voucher repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLPettyCashVoucherRepository(dbOps, tableName), nil
	})
}

// MySQLPettyCashVoucherRepository implements petty_cash_voucher CRUD using MySQL.
type MySQLPettyCashVoucherRepository struct {
	pettycashvoucherpb.UnimplementedPettyCashVoucherDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLPettyCashVoucherRepository creates a new MySQL petty_cash_voucher repository.
func NewMySQLPettyCashVoucherRepository(dbOps interfaces.DatabaseOperation, tableName string) pettycashvoucherpb.PettyCashVoucherDomainServiceServer {
	if tableName == "" {
		tableName = "petty_cash_voucher"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &MySQLPettyCashVoucherRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePettyCashVoucher creates a new petty_cash_voucher record.
func (r *MySQLPettyCashVoucherRepository) CreatePettyCashVoucher(ctx context.Context, req *pettycashvoucherpb.CreatePettyCashVoucherRequest) (*pettycashvoucherpb.CreatePettyCashVoucherResponse, error) {
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
	convertMillisToTime(data, "approvedAt", "approved_at")
	convertMillisToTime(data, "dateCreated", "date_created")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create petty_cash_voucher: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	v := &pettycashvoucherpb.PettyCashVoucher{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pettycashvoucherpb.CreatePettyCashVoucherResponse{
		Success: true,
		Data:    []*pettycashvoucherpb.PettyCashVoucher{v},
	}, nil
}

// ReadPettyCashVoucher retrieves a petty_cash_voucher record by ID.
func (r *MySQLPettyCashVoucherRepository) ReadPettyCashVoucher(ctx context.Context, req *pettycashvoucherpb.ReadPettyCashVoucherRequest) (*pettycashvoucherpb.ReadPettyCashVoucherResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_voucher ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read petty_cash_voucher: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	v := &pettycashvoucherpb.PettyCashVoucher{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pettycashvoucherpb.ReadPettyCashVoucherResponse{
		Success: true,
		Data:    []*pettycashvoucherpb.PettyCashVoucher{v},
	}, nil
}

// UpdatePettyCashVoucher updates a petty_cash_voucher record.
func (r *MySQLPettyCashVoucherRepository) UpdatePettyCashVoucher(ctx context.Context, req *pettycashvoucherpb.UpdatePettyCashVoucherRequest) (*pettycashvoucherpb.UpdatePettyCashVoucherResponse, error) {
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
	convertMillisToTime(data, "approvedAt", "approved_at")
	convertMillisToTime(data, "dateCreated", "date_created")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update petty_cash_voucher: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	v := &pettycashvoucherpb.PettyCashVoucher{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pettycashvoucherpb.UpdatePettyCashVoucherResponse{
		Success: true,
		Data:    []*pettycashvoucherpb.PettyCashVoucher{v},
	}, nil
}

// DeletePettyCashVoucher deletes a petty_cash_voucher record.
func (r *MySQLPettyCashVoucherRepository) DeletePettyCashVoucher(ctx context.Context, req *pettycashvoucherpb.DeletePettyCashVoucherRequest) (*pettycashvoucherpb.DeletePettyCashVoucherResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_voucher ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete petty_cash_voucher: %w", err)
	}
	return &pettycashvoucherpb.DeletePettyCashVoucherResponse{Success: true}, nil
}

// ListPettyCashVouchers lists petty_cash_voucher records with optional filters.
func (r *MySQLPettyCashVoucherRepository) ListPettyCashVouchers(ctx context.Context, req *pettycashvoucherpb.ListPettyCashVouchersRequest) (*pettycashvoucherpb.ListPettyCashVouchersResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list petty_cash_vouchers: %w", err)
	}
	var items []*pettycashvoucherpb.PettyCashVoucher
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal petty_cash_voucher row: %v", err)
			continue
		}
		item := &pettycashvoucherpb.PettyCashVoucher{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
			log.Printf("WARN: protojson unmarshal petty_cash_voucher: %v", err)
			continue
		}
		items = append(items, item)
	}
	return &pettycashvoucherpb.ListPettyCashVouchersResponse{
		Success: true,
		Data:    items,
	}, nil
}

// GetPettyCashVoucherListPageData retrieves petty_cash_vouchers with pagination and search.
//
// Dialect changes: $N → ?; ILIKE → LIKE; $1::text IS NULL → ? IS NULL.
func (r *MySQLPettyCashVoucherRepository) GetPettyCashVoucherListPageData(
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

	orderBy := "pcv.date_created DESC"
	if req.Sort != nil {
		ob, err := mysqlCore.BuildOrderBy(
			[]string{"pcv.date_created", "pcv.voucher_number", "pcv.total_amount", "pcv.status"},
			req.Sort,
			"pcv.date_created DESC",
		)
		if err == nil {
			orderBy = ob
		}
	}

	// Args: searchPattern x3, limit, offset
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
			WHERE (? IS NULL OR ? = '' OR
			       pcv.voucher_number LIKE ? OR
			       pcv.description LIKE ? OR
			       pcv.payee LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT e.*, c.total
		FROM enriched e, counted c
		ORDER BY ` + orderBy + `
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query,
		searchPattern, searchPattern, searchPattern, searchPattern, searchPattern,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query petty_cash_voucher list page data: %w", err)
	}
	defer rows.Close()

	var items []*pettycashvoucherpb.PettyCashVoucher
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
			&id, &dateCreated, &fundID, &voucherNumber,
			&payee, &description, &totalAmount, &status,
			&approvedBy, &approvedAt, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan petty_cash_voucher row: %w", err)
		}
		totalCount = total

		item := &pettycashvoucherpb.PettyCashVoucher{
			Id:            id,
			FundId:        fundID,
			VoucherNumber: voucherNumber,
			Description:   description,
			TotalAmount:   totalAmount,
		}
		if status != nil {
			if val, ok := pettycashvoucherpb.VoucherStatus_value[*status]; ok {
				item.Status = pettycashvoucherpb.VoucherStatus(val)
			}
		}
		if payee != nil {
			item.Payee = payee
		}
		if approvedBy != nil {
			item.ApprovedBy = approvedBy
		}
		if approvedAt != nil && *approvedAt > 0 {
			item.ApprovedAt = approvedAt
		}
		if dateCreated > 0 {
			item.DateCreated = &dateCreated
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating petty_cash_voucher rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &pettycashvoucherpb.GetPettyCashVoucherListPageDataResponse{
		PettyCashVoucherList: items,
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

// GetPettyCashVoucherItemPageData retrieves a single petty_cash_voucher by ID.
func (r *MySQLPettyCashVoucherRepository) GetPettyCashVoucherItemPageData(
	ctx context.Context,
	req *pettycashvoucherpb.GetPettyCashVoucherItemPageDataRequest,
) (*pettycashvoucherpb.GetPettyCashVoucherItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get petty_cash_voucher item page data request is required")
	}
	if req.PettyCashVoucherId == "" {
		return nil, fmt.Errorf("petty_cash_voucher ID is required")
	}

	const query = `
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
		WHERE pcv.id = ?
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, req.PettyCashVoucherId)

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
		&id, &dateCreated, &fundID, &voucherNumber,
		&payee, &description, &totalAmount, &status,
		&approvedBy, &approvedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("petty_cash_voucher with ID '%s' not found", req.PettyCashVoucherId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query petty_cash_voucher item page data: %w", err)
	}

	item := &pettycashvoucherpb.PettyCashVoucher{
		Id:            id,
		FundId:        fundID,
		VoucherNumber: voucherNumber,
		Description:   description,
		TotalAmount:   totalAmount,
	}
	if status != nil {
		if val, ok := pettycashvoucherpb.VoucherStatus_value[*status]; ok {
			item.Status = pettycashvoucherpb.VoucherStatus(val)
		}
	}
	if payee != nil {
		item.Payee = payee
	}
	if approvedBy != nil {
		item.ApprovedBy = approvedBy
	}
	if approvedAt != nil && *approvedAt > 0 {
		item.ApprovedAt = approvedAt
	}
	if dateCreated > 0 {
		item.DateCreated = &dateCreated
	}

	return &pettycashvoucherpb.GetPettyCashVoucherItemPageDataResponse{
		PettyCashVoucher: item,
		Success:          true,
	}, nil
}
