package treasury

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
	securitydepositpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/security_deposit"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SecurityDeposit, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres security_deposit repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresSecurityDepositRepository(dbOps, tableName), nil
	})
}

// PostgresSecurityDepositRepository implements security_deposit CRUD operations using PostgreSQL
type PostgresSecurityDepositRepository struct {
	securitydepositpb.UnimplementedSecurityDepositDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSecurityDepositRepository creates a new PostgreSQL security_deposit repository
func NewPostgresSecurityDepositRepository(dbOps interfaces.DatabaseOperation, tableName string) securitydepositpb.SecurityDepositDomainServiceServer {
	if tableName == "" {
		tableName = "security_deposit"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresSecurityDepositRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSecurityDeposit creates a new security_deposit record
func (r *PostgresSecurityDepositRepository) CreateSecurityDeposit(ctx context.Context, req *securitydepositpb.CreateSecurityDepositRequest) (*securitydepositpb.CreateSecurityDepositResponse, error) {
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

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "depositDate", "deposit_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create security_deposit: %w", err)
	}

	resultJSON, err := json.Marshal(result)
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

// ReadSecurityDeposit retrieves a security_deposit record by ID
func (r *PostgresSecurityDepositRepository) ReadSecurityDeposit(ctx context.Context, req *securitydepositpb.ReadSecurityDepositRequest) (*securitydepositpb.ReadSecurityDepositResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("security_deposit ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read security_deposit: %w", err)
	}

	resultJSON, err := json.Marshal(result)
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

// UpdateSecurityDeposit updates a security_deposit record
func (r *PostgresSecurityDepositRepository) UpdateSecurityDeposit(ctx context.Context, req *securitydepositpb.UpdateSecurityDepositRequest) (*securitydepositpb.UpdateSecurityDepositResponse, error) {
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

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "depositDate", "deposit_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update security_deposit: %w", err)
	}

	resultJSON, err := json.Marshal(result)
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

// DeleteSecurityDeposit deletes a security_deposit record (soft delete)
func (r *PostgresSecurityDepositRepository) DeleteSecurityDeposit(ctx context.Context, req *securitydepositpb.DeleteSecurityDepositRequest) (*securitydepositpb.DeleteSecurityDepositResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("security_deposit ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete security_deposit: %w", err)
	}

	return &securitydepositpb.DeleteSecurityDepositResponse{
		Success: true,
	}, nil
}

// ListSecurityDeposits lists security_deposit records with optional filters
func (r *PostgresSecurityDepositRepository) ListSecurityDeposits(ctx context.Context, req *securitydepositpb.ListSecurityDepositsRequest) (*securitydepositpb.ListSecurityDepositsResponse, error) {
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
		resultJSON, err := json.Marshal(result)
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

// GetSecurityDepositListPageData retrieves security_deposits with pagination, filtering, sorting, and search using CTE
func (r *PostgresSecurityDepositRepository) GetSecurityDepositListPageData(
	ctx context.Context,
	req *securitydepositpb.GetSecurityDepositListPageDataRequest,
) (*securitydepositpb.GetSecurityDepositListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get security_deposit list page data request is required")
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

	sortField := "sd.date_created"
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
			WHERE sd.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       sd.counterparty_name ILIKE $1 OR
			       sd.notes ILIKE $1)
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

		err := rows.Scan(
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
		)
		if err != nil {
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

// GetSecurityDepositItemPageData retrieves a single security_deposit with enriched data using CTE
func (r *PostgresSecurityDepositRepository) GetSecurityDepositItemPageData(
	ctx context.Context,
	req *securitydepositpb.GetSecurityDepositItemPageDataRequest,
) (*securitydepositpb.GetSecurityDepositItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get security_deposit item page data request is required")
	}
	if req.SecurityDepositId == "" {
		return nil, fmt.Errorf("security_deposit ID is required")
	}

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
			WHERE sd.id = $1 AND sd.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.SecurityDepositId)

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

// NewSecurityDepositRepository creates a new PostgreSQL security_deposit repository (old-style constructor)
func NewSecurityDepositRepository(db *sql.DB, tableName string) securitydepositpb.SecurityDepositDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresSecurityDepositRepository(dbOps, tableName)
}
