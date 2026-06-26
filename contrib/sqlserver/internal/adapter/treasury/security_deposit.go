//go:build sqlserver

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.SecurityDeposit, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver security_deposit repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerSecurityDepositRepository(dbOps, tableName), nil
	})
}

// SQLServerSecurityDepositRepository implements security_deposit CRUD operations using SQL Server.
type SQLServerSecurityDepositRepository struct {
	securitydepositpb.UnimplementedSecurityDepositDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerSecurityDepositRepository creates a new SQL Server security_deposit repository.
func NewSQLServerSecurityDepositRepository(dbOps interfaces.DatabaseOperation, tableName string) securitydepositpb.SecurityDepositDomainServiceServer {
	if tableName == "" {
		tableName = "security_deposit"
	}

	var db *sql.DB
	if ep, ok := dbOps.(executorProvider); ok {
		if rawDB, ok2 := ep.GetExecutor(context.Background()).(*sql.DB); ok2 {
			db = rawDB
		}
	}

	return &SQLServerSecurityDepositRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSecurityDeposit creates a new security_deposit record.
func (r *SQLServerSecurityDepositRepository) CreateSecurityDeposit(ctx context.Context, req *securitydepositpb.CreateSecurityDepositRequest) (*securitydepositpb.CreateSecurityDepositResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("security_deposit data is required")
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
		return nil, fmt.Errorf("failed to create security_deposit: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	securityDeposit := &securitydepositpb.SecurityDeposit{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, securityDeposit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &securitydepositpb.CreateSecurityDepositResponse{
		Success: true,
		Data:    []*securitydepositpb.SecurityDeposit{securityDeposit},
	}, nil
}

// ReadSecurityDeposit retrieves a security_deposit record by ID.
func (r *SQLServerSecurityDepositRepository) ReadSecurityDeposit(ctx context.Context, req *securitydepositpb.ReadSecurityDepositRequest) (*securitydepositpb.ReadSecurityDepositResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("security_deposit ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read security_deposit: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	securityDeposit := &securitydepositpb.SecurityDeposit{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, securityDeposit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &securitydepositpb.ReadSecurityDepositResponse{
		Success: true,
		Data:    []*securitydepositpb.SecurityDeposit{securityDeposit},
	}, nil
}

// UpdateSecurityDeposit updates a security_deposit record.
func (r *SQLServerSecurityDepositRepository) UpdateSecurityDeposit(ctx context.Context, req *securitydepositpb.UpdateSecurityDepositRequest) (*securitydepositpb.UpdateSecurityDepositResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("security_deposit ID is required")
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
		return nil, fmt.Errorf("failed to update security_deposit: %w", err)
	}

	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	securityDeposit := &securitydepositpb.SecurityDeposit{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, securityDeposit); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &securitydepositpb.UpdateSecurityDepositResponse{
		Success: true,
		Data:    []*securitydepositpb.SecurityDeposit{securityDeposit},
	}, nil
}

// DeleteSecurityDeposit soft-deletes a security_deposit record.
func (r *SQLServerSecurityDepositRepository) DeleteSecurityDeposit(ctx context.Context, req *securitydepositpb.DeleteSecurityDepositRequest) (*securitydepositpb.DeleteSecurityDepositResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("security_deposit ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete security_deposit: %w", err)
	}

	return &securitydepositpb.DeleteSecurityDepositResponse{Success: true}, nil
}

// ListSecurityDeposits lists security_deposit records with optional filters.
func (r *SQLServerSecurityDepositRepository) ListSecurityDeposits(ctx context.Context, req *securitydepositpb.ListSecurityDepositsRequest) (*securitydepositpb.ListSecurityDepositsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list security_deposits: %w", err)
	}

	var securityDeposits []*securitydepositpb.SecurityDeposit
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal security_deposit row: %v", err)
			continue
		}

		securityDeposit := &securitydepositpb.SecurityDeposit{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, securityDeposit); err != nil {
			log.Printf("WARN: protojson unmarshal security_deposit: %v", err)
			continue
		}
		securityDeposits = append(securityDeposits, securityDeposit)
	}

	return &securitydepositpb.ListSecurityDepositsResponse{
		Success: true,
		Data:    securityDeposits,
	}, nil
}

// GetSecurityDepositListPageData retrieves security_deposits with pagination, filtering, sorting, and search.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server differences from the postgres gold standard:
//   - @p1,@p2,… placeholders.
//   - LIKE instead of ILIKE.
//   - active = 1 (BIT) instead of active = true.
//   - Pagination: ORDER BY … OFFSET @pM ROWS FETCH NEXT @pN ROWS ONLY.
func (r *SQLServerSecurityDepositRepository) GetSecurityDepositListPageData(
	ctx context.Context,
	req *securitydepositpb.GetSecurityDepositListPageDataRequest,
) (*securitydepositpb.GetSecurityDepositListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get security_deposit list page data request is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

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

	sortColKey := "sd.date_created"
	if req.Sort != nil && len(req.Sort.Fields) > 0 && req.Sort.Fields[0].Field != "" {
		sortColKey = req.Sort.Fields[0].Field
	}

	sortDir := commonpb.SortDirection_DESC
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortDir = req.Sort.Fields[0].Direction
	}

	securityDepositSortableSQLCols := []string{
		"sd.date_created", "sd.date_modified", "sd.counterparty_name",
		"sd.amount", "sd.deposit_date", "sd.status",
	}

	orderByClause, err := sqlserverCore.BuildOrderBy(
		securityDepositSortableSQLCols,
		&commonpb.SortRequest{Fields: []*commonpb.SortField{{Field: sortColKey, Direction: sortDir}}},
		"sd.date_created DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("invalid sort column for security_deposit: %w", err)
	}

	query := fmt.Sprintf(`
		WITH enriched AS (
			SELECT
				sd.id,
				sd.date_created,
				sd.date_modified,
				sd.active,
				sd.direction,
				sd.counterparty_name,
				sd.amount,
				sd.deposit_date,
				sd.status,
				sd.account_id,
				sd.notes
			FROM security_deposit sd
			WHERE sd.active = 1
			  AND sd.workspace_id = @p1
			  AND (@p2 = '' OR
			       sd.counterparty_name LIKE @p2 OR
			       sd.notes LIKE @p2)
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		%s OFFSET @p3 ROWS FETCH NEXT @p4 ROWS ONLY;
	`, orderByClause)

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, workspaceID, searchPattern, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query security_deposit list page data: %w", err)
	}
	defer rows.Close()

	var securityDeposits []*securitydepositpb.SecurityDeposit
	var totalCount int64

	for rows.Next() {
		var (
			id               string
			dateCreated      int64
			dateModified     int64
			active           bool
			direction        *string
			counterpartyName string
			amount           int64
			depositDate      *int64
			status           *string
			accountID        *string
			notes            *string
			total            int64
		)

		if err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&direction,
			&counterpartyName,
			&amount,
			&depositDate,
			&status,
			&accountID,
			&notes,
			&total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan security_deposit row: %w", err)
		}

		totalCount = total

		securityDeposit := &securitydepositpb.SecurityDeposit{
			Id:               id,
			Active:           active,
			CounterpartyName: counterpartyName,
			Amount:           amount,
		}

		if direction != nil {
			if val, ok := securitydepositpb.DepositDirection_value[*direction]; ok {
				securityDeposit.Direction = securitydepositpb.DepositDirection(val)
			}
		}
		if status != nil {
			if val, ok := securitydepositpb.DepositStatus_value[*status]; ok {
				securityDeposit.Status = securitydepositpb.DepositStatus(val)
			}
		}
		if accountID != nil {
			securityDeposit.AccountId = accountID
		}
		if notes != nil {
			securityDeposit.Notes = notes
		}
		if depositDate != nil && *depositDate > 0 {
			securityDeposit.DepositDate = *depositDate
		}

		if dateCreated > 0 {
			securityDeposit.DateCreated = &dateCreated
		}
		if dateModified > 0 {
			securityDeposit.DateModified = &dateModified
		}

		securityDeposits = append(securityDeposits, securityDeposit)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating security_deposit rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &securitydepositpb.GetSecurityDepositListPageDataResponse{
		SecurityDepositList: securityDeposits,
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

// GetSecurityDepositItemPageData retrieves a single security_deposit.
// CRITICAL: Always filters by workspace_id for multi-tenancy.
//
// SQL Server: TOP 1 instead of LIMIT 1; @p1/@p2 instead of $1/$2; active = 1.
func (r *SQLServerSecurityDepositRepository) GetSecurityDepositItemPageData(
	ctx context.Context,
	req *securitydepositpb.GetSecurityDepositItemPageDataRequest,
) (*securitydepositpb.GetSecurityDepositItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get security_deposit item page data request is required")
	}
	if req.SecurityDepositId == "" {
		return nil, fmt.Errorf("security_deposit ID is required")
	}

	workspaceID := identity.Must(ctx).WorkspaceID

	query := `
		WITH enriched AS (
			SELECT
				sd.id,
				sd.date_created,
				sd.date_modified,
				sd.active,
				sd.direction,
				sd.counterparty_name,
				sd.amount,
				sd.deposit_date,
				sd.status,
				sd.account_id,
				sd.notes
			FROM security_deposit sd
			WHERE sd.id = @p1 AND sd.workspace_id = @p2 AND sd.active = 1
		)
		SELECT TOP 1 * FROM enriched;
	`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	row := exec.QueryRowContext(ctx, query, req.SecurityDepositId, workspaceID)

	var (
		id               string
		dateCreated      int64
		dateModified     int64
		active           bool
		direction        *string
		counterpartyName string
		amount           int64
		depositDate      *int64
		status           *string
		accountID        *string
		notes            *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&direction,
		&counterpartyName,
		&amount,
		&depositDate,
		&status,
		&accountID,
		&notes,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("security_deposit with ID '%s' not found", req.SecurityDepositId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query security_deposit item page data: %w", err)
	}

	securityDeposit := &securitydepositpb.SecurityDeposit{
		Id:               id,
		Active:           active,
		CounterpartyName: counterpartyName,
		Amount:           amount,
	}

	if direction != nil {
		if val, ok := securitydepositpb.DepositDirection_value[*direction]; ok {
			securityDeposit.Direction = securitydepositpb.DepositDirection(val)
		}
	}
	if status != nil {
		if val, ok := securitydepositpb.DepositStatus_value[*status]; ok {
			securityDeposit.Status = securitydepositpb.DepositStatus(val)
		}
	}
	if accountID != nil {
		securityDeposit.AccountId = accountID
	}
	if notes != nil {
		securityDeposit.Notes = notes
	}
	if depositDate != nil && *depositDate > 0 {
		securityDeposit.DepositDate = *depositDate
	}

	if dateCreated > 0 {
		securityDeposit.DateCreated = &dateCreated
	}
	if dateModified > 0 {
		securityDeposit.DateModified = &dateModified
	}

	return &securitydepositpb.GetSecurityDepositItemPageDataResponse{
		SecurityDeposit: securityDeposit,
		Success:         true,
	}, nil
}

// NewSecurityDepositRepository creates a new SQL Server security_deposit repository (old-style constructor).
func NewSecurityDepositRepository(db *sql.DB, tableName string) securitydepositpb.SecurityDepositDomainServiceServer {
	dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
	return NewSQLServerSecurityDepositRepository(dbOps, tableName)
}
