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
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	loanpaymentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan_payment"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.LoanPayment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres loan_payment repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresLoanPaymentRepository(dbOps, tableName), nil
	})
}

// PostgresLoanPaymentRepository implements loan_payment CRUD operations using PostgreSQL
type PostgresLoanPaymentRepository struct {
	loanpaymentpb.UnimplementedLoanPaymentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresLoanPaymentRepository creates a new PostgreSQL loan_payment repository
func NewPostgresLoanPaymentRepository(dbOps interfaces.DatabaseOperation, tableName string) loanpaymentpb.LoanPaymentDomainServiceServer {
	if tableName == "" {
		tableName = "loan_payment"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresLoanPaymentRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateLoanPayment creates a new loan_payment record
func (r *PostgresLoanPaymentRepository) CreateLoanPayment(ctx context.Context, req *loanpaymentpb.CreateLoanPaymentRequest) (*loanpaymentpb.CreateLoanPaymentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("loan_payment data is required")
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
	convertMillisToTime(data, "paymentDate", "payment_date")
	convertMillisToTime(data, "dateCreated", "date_created")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create loan_payment: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "payment_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	loanPayment := &loanpaymentpb.LoanPayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, loanPayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &loanpaymentpb.CreateLoanPaymentResponse{
		Success: true,
		Data:    []*loanpaymentpb.LoanPayment{loanPayment},
	}, nil
}

// ReadLoanPayment retrieves a loan_payment record by ID
func (r *PostgresLoanPaymentRepository) ReadLoanPayment(ctx context.Context, req *loanpaymentpb.ReadLoanPaymentRequest) (*loanpaymentpb.ReadLoanPaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("loan_payment ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read loan_payment: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "payment_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	loanPayment := &loanpaymentpb.LoanPayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, loanPayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &loanpaymentpb.ReadLoanPaymentResponse{
		Success: true,
		Data:    []*loanpaymentpb.LoanPayment{loanPayment},
	}, nil
}

// UpdateLoanPayment updates a loan_payment record
func (r *PostgresLoanPaymentRepository) UpdateLoanPayment(ctx context.Context, req *loanpaymentpb.UpdateLoanPaymentRequest) (*loanpaymentpb.UpdateLoanPaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("loan_payment ID is required")
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
	convertMillisToTime(data, "paymentDate", "payment_date")
	convertMillisToTime(data, "dateCreated", "date_created")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update loan_payment: %w", err)
	}

	postgresCore.ConvertMillisToDateStr(result, "payment_date")
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	loanPayment := &loanpaymentpb.LoanPayment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, loanPayment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &loanpaymentpb.UpdateLoanPaymentResponse{
		Success: true,
		Data:    []*loanpaymentpb.LoanPayment{loanPayment},
	}, nil
}

// DeleteLoanPayment deletes a loan_payment record
func (r *PostgresLoanPaymentRepository) DeleteLoanPayment(ctx context.Context, req *loanpaymentpb.DeleteLoanPaymentRequest) (*loanpaymentpb.DeleteLoanPaymentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("loan_payment ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete loan_payment: %w", err)
	}

	return &loanpaymentpb.DeleteLoanPaymentResponse{
		Success: true,
	}, nil
}

// ListLoanPayments lists loan_payment records with optional filters
func (r *PostgresLoanPaymentRepository) ListLoanPayments(ctx context.Context, req *loanpaymentpb.ListLoanPaymentsRequest) (*loanpaymentpb.ListLoanPaymentsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list loan_payments: %w", err)
	}

	var loanPayments []*loanpaymentpb.LoanPayment
	for _, result := range listResult.Data {
		postgresCore.ConvertMillisToDateStr(result, "payment_date")
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal loan_payment row: %v", err)
			continue
		}

		loanPayment := &loanpaymentpb.LoanPayment{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, loanPayment); err != nil {
			log.Printf("WARN: protojson unmarshal loan_payment: %v", err)
			continue
		}
		loanPayments = append(loanPayments, loanPayment)
	}

	return &loanpaymentpb.ListLoanPaymentsResponse{
		Success: true,
		Data:    loanPayments,
	}, nil
}

// GetLoanPaymentListPageData retrieves loan_payments with pagination, filtering, sorting, and search using CTE
func (r *PostgresLoanPaymentRepository) GetLoanPaymentListPageData(
	ctx context.Context,
	req *loanpaymentpb.GetLoanPaymentListPageDataRequest,
) (*loanpaymentpb.GetLoanPaymentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get loan_payment list page data request is required")
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

	sortField := "lp.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				lp.id,
				lp.date_created,
				lp.loan_id,
				lp.payment_number,
				lp.payment_date,
				lp.principal_amount,
				lp.interest_amount,
				lp.fee_amount,
				lp.total_amount,
				lp.remaining_balance,
				lp.notes
			FROM loan_payment lp
			WHERE ($1::text IS NULL OR $1::text = '' OR
			       lp.payment_number ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query loan_payment list page data: %w", err)
	}
	defer rows.Close()

	var loanPayments []*loanpaymentpb.LoanPayment
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			dateCreated      time.Time
			loanID           string
			paymentNumber    string
			paymentDate      *string
			principalAmount  int64
			interestAmount   int64
			feeAmount        int64
			totalAmount      int64
			remainingBalance int64
			notes            *string
			total            int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&loanID,
			&paymentNumber,
			&paymentDate,
			&principalAmount,
			&interestAmount,
			&feeAmount,
			&totalAmount,
			&remainingBalance,
			&notes,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan loan_payment row: %w", err)
		}

		totalCount = total

		loanPayment := &loanpaymentpb.LoanPayment{
			Id:               id,
			LoanId:           loanID,
			PaymentNumber:    paymentNumber,
			PrincipalAmount:  principalAmount,
			InterestAmount:   interestAmount,
			FeeAmount:        feeAmount,
			TotalAmount:      totalAmount,
			RemainingBalance: remainingBalance,
		}

		if notes != nil {
			loanPayment.Notes = notes
		}
		if paymentDate != nil {
			loanPayment.PaymentDate = *paymentDate
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			loanPayment.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			loanPayment.DateCreatedString = &dcStr
		}

		loanPayments = append(loanPayments, loanPayment)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating loan_payment rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &loanpaymentpb.GetLoanPaymentListPageDataResponse{
		LoanPaymentList: loanPayments,
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

// GetLoanPaymentItemPageData retrieves a single loan_payment with enriched data using CTE
func (r *PostgresLoanPaymentRepository) GetLoanPaymentItemPageData(
	ctx context.Context,
	req *loanpaymentpb.GetLoanPaymentItemPageDataRequest,
) (*loanpaymentpb.GetLoanPaymentItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get loan_payment item page data request is required")
	}
	if req.LoanPaymentId == "" {
		return nil, fmt.Errorf("loan_payment ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				lp.id,
				lp.date_created,
				lp.loan_id,
				lp.payment_number,
				lp.payment_date,
				lp.principal_amount,
				lp.interest_amount,
				lp.fee_amount,
				lp.total_amount,
				lp.remaining_balance,
				lp.notes
			FROM loan_payment lp
			WHERE lp.id = $1
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.LoanPaymentId)

	var (
		id               string
		dateCreated      time.Time
		loanID           string
		paymentNumber    string
		paymentDate      *string
		principalAmount  int64
		interestAmount   int64
		feeAmount        int64
		totalAmount      int64
		remainingBalance int64
		notes            *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&loanID,
		&paymentNumber,
		&paymentDate,
		&principalAmount,
		&interestAmount,
		&feeAmount,
		&totalAmount,
		&remainingBalance,
		&notes,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("loan_payment with ID '%s' not found", req.LoanPaymentId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query loan_payment item page data: %w", err)
	}

	loanPayment := &loanpaymentpb.LoanPayment{
		Id:               id,
		LoanId:           loanID,
		PaymentNumber:    paymentNumber,
		PrincipalAmount:  principalAmount,
		InterestAmount:   interestAmount,
		FeeAmount:        feeAmount,
		TotalAmount:      totalAmount,
		RemainingBalance: remainingBalance,
	}

	if notes != nil {
		loanPayment.Notes = notes
	}
	if paymentDate != nil {
		loanPayment.PaymentDate = *paymentDate
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		loanPayment.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		loanPayment.DateCreatedString = &dcStr
	}

	return &loanpaymentpb.GetLoanPaymentItemPageDataResponse{
		LoanPayment: loanPayment,
		Success:     true,
	}, nil
}

// NewLoanPaymentRepository creates a new PostgreSQL loan_payment repository (old-style constructor)
func NewLoanPaymentRepository(db *sql.DB, tableName string) loanpaymentpb.LoanPaymentDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresLoanPaymentRepository(dbOps, tableName)
}
