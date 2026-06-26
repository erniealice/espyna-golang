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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	loanpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/loan"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Loan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver loan repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerLoanRepository(dbOps, tableName), nil
	})
}

// SQLServerLoanRepository implements loan CRUD operations using SQL Server.
type SQLServerLoanRepository struct {
	loanpb.UnimplementedLoanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerLoanRepository creates a new SQL Server loan repository.
func NewSQLServerLoanRepository(dbOps interfaces.DatabaseOperation, tableName string) loanpb.LoanDomainServiceServer {
	if tableName == "" {
		tableName = "loan"
	}

	var db *sql.DB
	if ep, ok := dbOps.(executorProvider); ok {
		if rawDB, ok2 := ep.GetExecutor(context.Background()).(*sql.DB); ok2 {
			db = rawDB
		}
	}

	return &SQLServerLoanRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateLoan creates a new loan record.
func (r *SQLServerLoanRepository) CreateLoan(ctx context.Context, req *loanpb.CreateLoanRequest) (*loanpb.CreateLoanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("loan data is required")
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
		return nil, fmt.Errorf("failed to create loan: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	loan := &loanpb.Loan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, loan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &loanpb.CreateLoanResponse{
		Success: true,
		Data:    []*loanpb.Loan{loan},
	}, nil
}

// ReadLoan retrieves a loan record by ID.
func (r *SQLServerLoanRepository) ReadLoan(ctx context.Context, req *loanpb.ReadLoanRequest) (*loanpb.ReadLoanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("loan ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read loan: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	loan := &loanpb.Loan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, loan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &loanpb.ReadLoanResponse{
		Success: true,
		Data:    []*loanpb.Loan{loan},
	}, nil
}

// UpdateLoan updates a loan record.
func (r *SQLServerLoanRepository) UpdateLoan(ctx context.Context, req *loanpb.UpdateLoanRequest) (*loanpb.UpdateLoanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("loan ID is required")
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
		return nil, fmt.Errorf("failed to update loan: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	loan := &loanpb.Loan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, loan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &loanpb.UpdateLoanResponse{
		Success: true,
		Data:    []*loanpb.Loan{loan},
	}, nil
}

// DeleteLoan soft-deletes a loan record.
func (r *SQLServerLoanRepository) DeleteLoan(ctx context.Context, req *loanpb.DeleteLoanRequest) (*loanpb.DeleteLoanResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("loan ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete loan: %w", err)
	}

	return &loanpb.DeleteLoanResponse{Success: true}, nil
}

// ListLoans lists loan records with optional filters.
func (r *SQLServerLoanRepository) ListLoans(ctx context.Context, req *loanpb.ListLoansRequest) (*loanpb.ListLoansResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list loans: %w", err)
	}

	var loans []*loanpb.Loan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal loan row: %v", err)
			continue
		}

		loan := &loanpb.Loan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, loan); err != nil {
			log.Printf("WARN: protojson unmarshal loan: %v", err)
			continue
		}
		loans = append(loans, loan)
	}

	return &loanpb.ListLoansResponse{
		Success: true,
		Data:    loans,
	}, nil
}

// GetLoanListPageData retrieves loans with pagination, filtering, sorting, and search.
// SQL Server translation: @pN placeholders, LIKE, active = 1, OFFSET/FETCH pagination.
func (r *SQLServerLoanRepository) GetLoanListPageData(
	ctx context.Context,
	req *loanpb.GetLoanListPageDataRequest,
) (*loanpb.GetLoanListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get loan list page data request is required")
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

	sortColKey := "l.date_created"
	sortDir := commonpb.SortDirection_DESC
	if req.Sort != nil && len(req.Sort.Fields) > 0 && req.Sort.Fields[0].Field != "" {
		sortColKey = req.Sort.Fields[0].Field
		sortDir = req.Sort.Fields[0].Direction
	}

	loanSortableCols := []string{
		"l.date_created", "l.date_modified", "l.loan_number", "l.lender_name",
		"l.principal_amount", "l.remaining_balance", "l.status", "l.start_date", "l.maturity_date",
	}

	orderByClause, err := sqlserverCore.BuildOrderBy(
		loanSortableCols,
		&commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: sortColKey, Direction: sortDir}}},
		"l.date_created DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("invalid sort column for loan: %w", err)
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				l.id,
				l.date_created,
				l.date_modified,
				l.active,
				l.loan_number,
				l.description,
				l.loan_type,
				l.lender_name,
				l.principal_amount,
				l.interest_rate,
				l.term_months,
				l.start_date,
				l.maturity_date,
				l.status,
				l.remaining_balance,
				l.account_id
			FROM %s l
			WHERE l.active = 1
			  AND (@p1 = '' OR
			       l.loan_number LIKE @p1 OR
			       l.lender_name LIKE @p1 OR
			       l.description LIKE @p1)
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s OFFSET @p2 ROWS FETCH NEXT @p3 ROWS ONLY;
	`, r.tableName, orderByClause)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, searchPattern, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query loan list page data: %w", err)
	}
	defer rows.Close()

	var loans []*loanpb.Loan
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			dateCreated      int64
			dateModified     int64
			active           bool
			loanNumber       string
			description      *string
			loanType         *string
			lenderName       string
			principalAmount  int64
			interestRate     float64
			termMonths       int32
			startDate        *string
			maturityDate     *string
			status           *string
			remainingBalance int64
			accountID        *string
			total            int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&loanNumber,
			&description,
			&loanType,
			&lenderName,
			&principalAmount,
			&interestRate,
			&termMonths,
			&startDate,
			&maturityDate,
			&status,
			&remainingBalance,
			&accountID,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan loan row: %w", err)
		}

		totalCount = total

		loan := &loanpb.Loan{
			Id:               id,
			Active:           active,
			LoanNumber:       loanNumber,
			LenderName:       lenderName,
			PrincipalAmount:  principalAmount,
			InterestRate:     interestRate,
			TermMonths:       termMonths,
			RemainingBalance: remainingBalance,
		}

		if description != nil {
			loan.Description = description
		}
		if accountID != nil {
			loan.AccountId = accountID
		}
		if loanType != nil {
			if val, ok := loanpb.LoanType_value[*loanType]; ok {
				loan.LoanType = loanpb.LoanType(val)
			}
		}
		if status != nil {
			if val, ok := loanpb.LoanStatus_value[*status]; ok {
				loan.Status = loanpb.LoanStatus(val)
			}
		}
		if startDate != nil {
			loan.StartDate = *startDate
		}
		if maturityDate != nil {
			loan.MaturityDate = *maturityDate
		}
		if dateCreated > 0 {
			loan.DateCreated = &dateCreated
		}
		if dateModified > 0 {
			loan.DateModified = &dateModified
		}

		loans = append(loans, loan)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating loan rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &loanpb.GetLoanListPageDataResponse{
		LoanList: loans,
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

// GetLoanItemPageData retrieves a single loan with enriched data.
func (r *SQLServerLoanRepository) GetLoanItemPageData(
	ctx context.Context,
	req *loanpb.GetLoanItemPageDataRequest,
) (*loanpb.GetLoanItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get loan item page data request is required")
	}
	if req.LoanId == "" {
		return nil, fmt.Errorf("loan ID is required")
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				l.id,
				l.date_created,
				l.date_modified,
				l.active,
				l.loan_number,
				l.description,
				l.loan_type,
				l.lender_name,
				l.principal_amount,
				l.interest_rate,
				l.term_months,
				l.start_date,
				l.maturity_date,
				l.status,
				l.remaining_balance,
				l.account_id
			FROM %s l
			WHERE l.id = @p1 AND l.active = 1
		)
		SELECT TOP 1 * FROM enriched;
	`, r.tableName)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.LoanId)

	var (
		id               string
		dateCreated      int64
		dateModified     int64
		active           bool
		loanNumber       string
		description      *string
		loanType         *string
		lenderName       string
		principalAmount  int64
		interestRate     float64
		termMonths       int32
		startDate        *string
		maturityDate     *string
		status           *string
		remainingBalance int64
		accountID        *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&loanNumber,
		&description,
		&loanType,
		&lenderName,
		&principalAmount,
		&interestRate,
		&termMonths,
		&startDate,
		&maturityDate,
		&status,
		&remainingBalance,
		&accountID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("loan with ID '%s' not found", req.LoanId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query loan item page data: %w", err)
	}

	loan := &loanpb.Loan{
		Id:               id,
		Active:           active,
		LoanNumber:       loanNumber,
		LenderName:       lenderName,
		PrincipalAmount:  principalAmount,
		InterestRate:     interestRate,
		TermMonths:       termMonths,
		RemainingBalance: remainingBalance,
	}

	if description != nil {
		loan.Description = description
	}
	if accountID != nil {
		loan.AccountId = accountID
	}
	if loanType != nil {
		if val, ok := loanpb.LoanType_value[*loanType]; ok {
			loan.LoanType = loanpb.LoanType(val)
		}
	}
	if status != nil {
		if val, ok := loanpb.LoanStatus_value[*status]; ok {
			loan.Status = loanpb.LoanStatus(val)
		}
	}
	if startDate != nil {
		loan.StartDate = *startDate
	}
	if maturityDate != nil {
		loan.MaturityDate = *maturityDate
	}
	if dateCreated > 0 {
		loan.DateCreated = &dateCreated
	}
	if dateModified > 0 {
		loan.DateModified = &dateModified
	}

	return &loanpb.GetLoanItemPageDataResponse{
		Loan:    loan,
		Success: true,
	}, nil
}

// NewLoanRepository creates a new SQL Server loan repository (old-style constructor).
func NewLoanRepository(db *sql.DB, tableName string) loanpb.LoanDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerLoanRepository(dbOps, tableName)
}
