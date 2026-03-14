
package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TreasuryDisbursement, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres disbursement repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresDisbursementRepository(dbOps, tableName), nil
	})
}

// PostgresDisbursementRepository implements disbursement CRUD operations using PostgreSQL
type PostgresDisbursementRepository struct {
	disbursementpb.UnimplementedDisbursementDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresDisbursementRepository creates a new PostgreSQL disbursement repository
func NewPostgresDisbursementRepository(dbOps interfaces.DatabaseOperation, tableName string) disbursementpb.DisbursementDomainServiceServer {
	if tableName == "" {
		tableName = "treasury_disbursement"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresDisbursementRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateDisbursement creates a new disbursement record
func (r *PostgresDisbursementRepository) CreateDisbursement(ctx context.Context, req *disbursementpb.CreateDisbursementRequest) (*disbursementpb.CreateDisbursementResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("disbursement data is required")
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
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create disbursement: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	disbursement := &disbursementpb.Disbursement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, disbursement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &disbursementpb.CreateDisbursementResponse{
		Success: true,
		Data:    []*disbursementpb.Disbursement{disbursement},
	}, nil
}

// ReadDisbursement retrieves a disbursement record by ID
func (r *PostgresDisbursementRepository) ReadDisbursement(ctx context.Context, req *disbursementpb.ReadDisbursementRequest) (*disbursementpb.ReadDisbursementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read disbursement: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	disbursement := &disbursementpb.Disbursement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, disbursement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &disbursementpb.ReadDisbursementResponse{
		Success: true,
		Data:    []*disbursementpb.Disbursement{disbursement},
	}, nil
}

// UpdateDisbursement updates a disbursement record
func (r *PostgresDisbursementRepository) UpdateDisbursement(ctx context.Context, req *disbursementpb.UpdateDisbursementRequest) (*disbursementpb.UpdateDisbursementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement ID is required")
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
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update disbursement: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	disbursement := &disbursementpb.Disbursement{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, disbursement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &disbursementpb.UpdateDisbursementResponse{
		Success: true,
		Data:    []*disbursementpb.Disbursement{disbursement},
	}, nil
}

// DeleteDisbursement deletes a disbursement record (soft delete)
func (r *PostgresDisbursementRepository) DeleteDisbursement(ctx context.Context, req *disbursementpb.DeleteDisbursementRequest) (*disbursementpb.DeleteDisbursementResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("disbursement ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete disbursement: %w", err)
	}

	return &disbursementpb.DeleteDisbursementResponse{
		Success: true,
	}, nil
}

// ListDisbursements lists disbursement records with optional filters
func (r *PostgresDisbursementRepository) ListDisbursements(ctx context.Context, req *disbursementpb.ListDisbursementsRequest) (*disbursementpb.ListDisbursementsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list disbursements: %w", err)
	}

	var disbursements []*disbursementpb.Disbursement
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal disbursement row: %v", err)
			continue
		}

		disbursement := &disbursementpb.Disbursement{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, disbursement); err != nil {
			log.Printf("WARN: protojson unmarshal disbursement: %v", err)
			continue
		}
		disbursements = append(disbursements, disbursement)
	}

	return &disbursementpb.ListDisbursementsResponse{
		Success: true,
		Data:    disbursements,
	}, nil
}

// GetDisbursementListPageData retrieves disbursements with pagination, filtering, sorting, and search using CTE
func (r *PostgresDisbursementRepository) GetDisbursementListPageData(
	ctx context.Context,
	req *disbursementpb.GetDisbursementListPageDataRequest,
) (*disbursementpb.GetDisbursementListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get disbursement list page data request is required")
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

	sortField := "d.date_created"
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
				d.id,
				d.date_created,
				d.date_modified,
				d.active,
				d.name,
				d.subscription_id,
				d.amount,
				d.status,
				d.expenditure_id,
				d.disbursement_type,
				d.disbursement_method_id,
				d.currency,
				d.reference_number,
				d.payment_date,
				d.approved_by
			FROM treasury_disbursement d
			WHERE d.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       d.name ILIKE $1 OR
			       d.reference_number ILIKE $1 OR
			       d.status ILIKE $1 OR
			       d.disbursement_type ILIKE $1)
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
		return nil, fmt.Errorf("failed to query disbursement list page data: %w", err)
	}
	defer rows.Close()

	var disbursements []*disbursementpb.Disbursement
	var totalCount int64

	for rows.Next() {
		var (
			id                   string
			dateCreated          time.Time
			dateModified         time.Time
			active               bool
			name                 string
			subscriptionID       *string
			amount               float64
			status               *string
			expenditureID        *string
			disbursementType     *string
			disbursementMethodID *string
			currency             *string
			referenceNumber      *string
			paymentDate          *time.Time
			approvedBy           *string
			total                int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&subscriptionID,
			&amount,
			&status,
			&expenditureID,
			&disbursementType,
			&disbursementMethodID,
			&currency,
			&referenceNumber,
			&paymentDate,
			&approvedBy,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan disbursement row: %w", err)
		}

		totalCount = total

		disbursement := &disbursementpb.Disbursement{
			Id:     id,
			Active: active,
			Name:   name,
			Amount: amount,
		}

		if subscriptionID != nil {
			disbursement.SubscriptionId = *subscriptionID
		}
		if status != nil {
			disbursement.Status = *status
		}
		if expenditureID != nil {
			disbursement.ExpenditureId = *expenditureID
		}
		if disbursementType != nil {
			disbursement.DisbursementType = *disbursementType
		}
		if disbursementMethodID != nil {
			disbursement.DisbursementMethodId = *disbursementMethodID
		}
		if currency != nil {
			disbursement.Currency = *currency
		}
		if referenceNumber != nil {
			disbursement.ReferenceNumber = *referenceNumber
		}
		if approvedBy != nil {
			disbursement.ApprovedBy = *approvedBy
		}
		if paymentDate != nil && !paymentDate.IsZero() {
			disbursement.PaymentDate = paymentDate.UnixMilli()
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			disbursement.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			disbursement.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			disbursement.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			disbursement.DateModifiedString = &dmStr
		}

		disbursements = append(disbursements, disbursement)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating disbursement rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &disbursementpb.GetDisbursementListPageDataResponse{
		DisbursementList: disbursements,
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

// GetDisbursementItemPageData retrieves a single disbursement with enriched data using CTE
func (r *PostgresDisbursementRepository) GetDisbursementItemPageData(
	ctx context.Context,
	req *disbursementpb.GetDisbursementItemPageDataRequest,
) (*disbursementpb.GetDisbursementItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get disbursement item page data request is required")
	}
	if req.DisbursementId == "" {
		return nil, fmt.Errorf("disbursement ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				d.id,
				d.date_created,
				d.date_modified,
				d.active,
				d.name,
				d.subscription_id,
				d.amount,
				d.status,
				d.expenditure_id,
				d.disbursement_type,
				d.disbursement_method_id,
				d.currency,
				d.reference_number,
				d.payment_date,
				d.approved_by
			FROM treasury_disbursement d
			WHERE d.id = $1 AND d.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.DisbursementId)

	var (
		id                   string
		dateCreated          time.Time
		dateModified         time.Time
		active               bool
		name                 string
		subscriptionID       *string
		amount               float64
		status               *string
		expenditureID        *string
		disbursementType     *string
		disbursementMethodID *string
		currency             *string
		referenceNumber      *string
		paymentDate          *time.Time
		approvedBy           *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&subscriptionID,
		&amount,
		&status,
		&expenditureID,
		&disbursementType,
		&disbursementMethodID,
		&currency,
		&referenceNumber,
		&paymentDate,
		&approvedBy,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("disbursement with ID '%s' not found", req.DisbursementId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query disbursement item page data: %w", err)
	}

	disbursement := &disbursementpb.Disbursement{
		Id:     id,
		Active: active,
		Name:   name,
		Amount: amount,
	}

	if subscriptionID != nil {
		disbursement.SubscriptionId = *subscriptionID
	}
	if status != nil {
		disbursement.Status = *status
	}
	if expenditureID != nil {
		disbursement.ExpenditureId = *expenditureID
	}
	if disbursementType != nil {
		disbursement.DisbursementType = *disbursementType
	}
	if disbursementMethodID != nil {
		disbursement.DisbursementMethodId = *disbursementMethodID
	}
	if currency != nil {
		disbursement.Currency = *currency
	}
	if referenceNumber != nil {
		disbursement.ReferenceNumber = *referenceNumber
	}
	if approvedBy != nil {
		disbursement.ApprovedBy = *approvedBy
	}
	if paymentDate != nil && !paymentDate.IsZero() {
		disbursement.PaymentDate = paymentDate.UnixMilli()
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		disbursement.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		disbursement.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		disbursement.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		disbursement.DateModifiedString = &dmStr
	}

	return &disbursementpb.GetDisbursementItemPageDataResponse{
		Disbursement: disbursement,
		Success:      true,
	}, nil
}

// NewDisbursementRepository creates a new PostgreSQL disbursement repository (old-style constructor)
func NewDisbursementRepository(db *sql.DB, tableName string) disbursementpb.DisbursementDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresDisbursementRepository(dbOps, tableName)
}
