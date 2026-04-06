package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	prepaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/prepayment"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Prepayment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres prepayment repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPrepaymentRepository(dbOps, tableName), nil
	})
}

// PostgresPrepaymentRepository implements prepayment CRUD operations using PostgreSQL
type PostgresPrepaymentRepository struct {
	prepaymentpb.UnimplementedPrepaymentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresPrepaymentRepository creates a new PostgreSQL prepayment repository
func NewPostgresPrepaymentRepository(dbOps interfaces.DatabaseOperation, tableName string) prepaymentpb.PrepaymentDomainServiceServer {
	if tableName == "" {
		tableName = "prepayment"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPrepaymentRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePrepayment creates a new prepayment record
func (r *PostgresPrepaymentRepository) CreatePrepayment(ctx context.Context, req *prepaymentpb.CreatePrepaymentRequest) (*prepaymentpb.CreatePrepaymentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("prepayment data is required")
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
	convertMillisToTime(data, "startDate", "start_date")
	convertMillisToTime(data, "endDate", "end_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create prepayment: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	prepayment := &prepaymentpb.Prepayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prepayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &prepaymentpb.CreatePrepaymentResponse{
		Success: true,
		Data:    []*prepaymentpb.Prepayment{prepayment},
	}, nil
}

// ReadPrepayment retrieves a prepayment record by ID
func (r *PostgresPrepaymentRepository) ReadPrepayment(ctx context.Context, req *prepaymentpb.ReadPrepaymentRequest) (*prepaymentpb.ReadPrepaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("prepayment ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read prepayment: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	prepayment := &prepaymentpb.Prepayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prepayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &prepaymentpb.ReadPrepaymentResponse{
		Success: true,
		Data:    []*prepaymentpb.Prepayment{prepayment},
	}, nil
}

// UpdatePrepayment updates a prepayment record
func (r *PostgresPrepaymentRepository) UpdatePrepayment(ctx context.Context, req *prepaymentpb.UpdatePrepaymentRequest) (*prepaymentpb.UpdatePrepaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("prepayment ID is required")
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
	convertMillisToTime(data, "startDate", "start_date")
	convertMillisToTime(data, "endDate", "end_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update prepayment: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	prepayment := &prepaymentpb.Prepayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prepayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &prepaymentpb.UpdatePrepaymentResponse{
		Success: true,
		Data:    []*prepaymentpb.Prepayment{prepayment},
	}, nil
}

// DeletePrepayment deletes a prepayment record (soft delete)
func (r *PostgresPrepaymentRepository) DeletePrepayment(ctx context.Context, req *prepaymentpb.DeletePrepaymentRequest) (*prepaymentpb.DeletePrepaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("prepayment ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete prepayment: %w", err)
	}

	return &prepaymentpb.DeletePrepaymentResponse{
		Success: true,
	}, nil
}

// ListPrepayments lists prepayment records with optional filters
func (r *PostgresPrepaymentRepository) ListPrepayments(ctx context.Context, req *prepaymentpb.ListPrepaymentsRequest) (*prepaymentpb.ListPrepaymentsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list prepayments: %w", err)
	}

	var prepayments []*prepaymentpb.Prepayment
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToDateStr(result, "start_date", "end_date")
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal prepayment row: %v", err)
			continue
		}

		prepayment := &prepaymentpb.Prepayment{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prepayment); err != nil {
			log.Printf("WARN: protojson unmarshal prepayment: %v", err)
			continue
		}
		prepayments = append(prepayments, prepayment)
	}

	return &prepaymentpb.ListPrepaymentsResponse{
		Success: true,
		Data:    prepayments,
	}, nil
}

// GetPrepaymentListPageData retrieves prepayments with pagination, filtering, sorting, and search using CTE
// TODO: Add enriched joins with account table for GL account names once CoA is in place
func (r *PostgresPrepaymentRepository) GetPrepaymentListPageData(
	ctx context.Context,
	req *prepaymentpb.GetPrepaymentListPageDataRequest,
) (*prepaymentpb.GetPrepaymentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get prepayment list page data request is required")
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

	sortField := "p.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				p.id,
				p.date_created,
				p.date_modified,
				p.active,
				p.description,
				p.vendor_name,
				p.total_amount,
				p.remaining_amount,
				p.start_date,
				p.end_date,
				p.amortization_months,
				p.status,
				p.account_id,
				p.expense_account_id
			FROM %s p
			WHERE p.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       p.description ILIKE $1 OR
			       p.vendor_name ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY %s %s
		LIMIT $2 OFFSET $3;
	`, r.tableName, sortField, sortOrder)

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query prepayment list page data: %w", err)
	}
	defer rows.Close()

	var prepayments []*prepaymentpb.Prepayment
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			dateCreated        int64
			dateModified       int64
			active             bool
			description        string
			vendorName         *string
			totalAmount        int64
			remainingAmount    int64
			startDate          *string
			endDate            *string
			amortizationMonths int32
			statusStr          string
			accountID          *string
			expenseAccountID   *string
			total              int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&description,
			&vendorName,
			&totalAmount,
			&remainingAmount,
			&startDate,
			&endDate,
			&amortizationMonths,
			&statusStr,
			&accountID,
			&expenseAccountID,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan prepayment row: %w", err)
		}

		totalCount = total

		prepayment := &prepaymentpb.Prepayment{
			Id:                 id,
			Active:             active,
			Description:        description,
			VendorName:         vendorName,
			TotalAmount:        totalAmount,
			RemainingAmount:    remainingAmount,
			AmortizationMonths: amortizationMonths,
			AccountId:          accountID,
			ExpenseAccountId:   expenseAccountID,
		}

		if val, ok := prepaymentpb.PrepaymentStatus_value[statusStr]; ok {
			prepayment.Status = prepaymentpb.PrepaymentStatus(val)
		}

		if startDate != nil {
			prepayment.StartDate = *startDate
		}
		if endDate != nil {
			prepayment.EndDate = *endDate
		}
		if dateCreated > 0 {
			prepayment.DateCreated = &dateCreated
		}
		if dateModified > 0 {
			prepayment.DateModified = &dateModified
		}

		prepayments = append(prepayments, prepayment)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating prepayment rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &prepaymentpb.GetPrepaymentListPageDataResponse{
		PrepaymentList: prepayments,
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

// GetPrepaymentItemPageData retrieves a single prepayment with enriched data
// TODO: Add CTE query with joined account details once CoA is in place
func (r *PostgresPrepaymentRepository) GetPrepaymentItemPageData(
	ctx context.Context,
	req *prepaymentpb.GetPrepaymentItemPageDataRequest,
) (*prepaymentpb.GetPrepaymentItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get prepayment item page data request is required")
	}
	if req.PrepaymentId == "" {
		return nil, fmt.Errorf("prepayment ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.PrepaymentId)
	if err != nil {
		return nil, fmt.Errorf("failed to read prepayment '%s': %w", req.PrepaymentId, err)
	}

	postgresCore.ConvertMillisToDateStr(result, "start_date", "end_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	prepayment := &prepaymentpb.Prepayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, prepayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &prepaymentpb.GetPrepaymentItemPageDataResponse{
		Prepayment: prepayment,
		Success:    true,
	}, nil
}

// NewPrepaymentRepository creates a new PostgreSQL prepayment repository (old-style constructor)
func NewPrepaymentRepository(db *sql.DB, tableName string) prepaymentpb.PrepaymentDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresPrepaymentRepository(dbOps, tableName)
}
