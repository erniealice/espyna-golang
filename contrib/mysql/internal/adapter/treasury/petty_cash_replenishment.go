//go:build mysql

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pettycashreplenishmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/petty_cash_replenishment"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.PettyCashReplenishment, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql petty_cash_replenishment repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLPettyCashReplenishmentRepository(dbOps, tableName), nil
	})
}

// MySQLPettyCashReplenishmentRepository implements petty_cash_replenishment CRUD using MySQL.
type MySQLPettyCashReplenishmentRepository struct {
	pettycashreplenishmentpb.UnimplementedPettyCashReplenishmentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLPettyCashReplenishmentRepository creates a new MySQL petty_cash_replenishment repository.
func NewMySQLPettyCashReplenishmentRepository(dbOps interfaces.DatabaseOperation, tableName string) pettycashreplenishmentpb.PettyCashReplenishmentDomainServiceServer {
	if tableName == "" {
		tableName = "petty_cash_replenishment"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &MySQLPettyCashReplenishmentRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePettyCashReplenishment creates a new petty_cash_replenishment record.
func (r *MySQLPettyCashReplenishmentRepository) CreatePettyCashReplenishment(ctx context.Context, req *pettycashreplenishmentpb.CreatePettyCashReplenishmentRequest) (*pettycashreplenishmentpb.CreatePettyCashReplenishmentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("petty_cash_replenishment data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "replenishmentDate", "replenishment_date")
	convertMillisToTime(data, "dateCreated", "date_created")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create petty_cash_replenishment: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pettyCashReplenishment := &pettycashreplenishmentpb.PettyCashReplenishment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashReplenishment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pettycashreplenishmentpb.CreatePettyCashReplenishmentResponse{
		Success: true,
		Data:    []*pettycashreplenishmentpb.PettyCashReplenishment{pettyCashReplenishment},
	}, nil
}

// ReadPettyCashReplenishment retrieves a petty_cash_replenishment record by ID.
func (r *MySQLPettyCashReplenishmentRepository) ReadPettyCashReplenishment(ctx context.Context, req *pettycashreplenishmentpb.ReadPettyCashReplenishmentRequest) (*pettycashreplenishmentpb.ReadPettyCashReplenishmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_replenishment ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read petty_cash_replenishment: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pettyCashReplenishment := &pettycashreplenishmentpb.PettyCashReplenishment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashReplenishment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pettycashreplenishmentpb.ReadPettyCashReplenishmentResponse{
		Success: true,
		Data:    []*pettycashreplenishmentpb.PettyCashReplenishment{pettyCashReplenishment},
	}, nil
}

// UpdatePettyCashReplenishment updates a petty_cash_replenishment record.
func (r *MySQLPettyCashReplenishmentRepository) UpdatePettyCashReplenishment(ctx context.Context, req *pettycashreplenishmentpb.UpdatePettyCashReplenishmentRequest) (*pettycashreplenishmentpb.UpdatePettyCashReplenishmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_replenishment ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	convertMillisToTime(data, "replenishmentDate", "replenishment_date")
	convertMillisToTime(data, "dateCreated", "date_created")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update petty_cash_replenishment: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pettyCashReplenishment := &pettycashreplenishmentpb.PettyCashReplenishment{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pettyCashReplenishment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &pettycashreplenishmentpb.UpdatePettyCashReplenishmentResponse{
		Success: true,
		Data:    []*pettycashreplenishmentpb.PettyCashReplenishment{pettyCashReplenishment},
	}, nil
}

// DeletePettyCashReplenishment deletes a petty_cash_replenishment record.
func (r *MySQLPettyCashReplenishmentRepository) DeletePettyCashReplenishment(ctx context.Context, req *pettycashreplenishmentpb.DeletePettyCashReplenishmentRequest) (*pettycashreplenishmentpb.DeletePettyCashReplenishmentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("petty_cash_replenishment ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete petty_cash_replenishment: %w", err)
	}
	return &pettycashreplenishmentpb.DeletePettyCashReplenishmentResponse{Success: true}, nil
}

// ListPettyCashReplenishments lists petty_cash_replenishment records with optional filters.
func (r *MySQLPettyCashReplenishmentRepository) ListPettyCashReplenishments(ctx context.Context, req *pettycashreplenishmentpb.ListPettyCashReplenishmentsRequest) (*pettycashreplenishmentpb.ListPettyCashReplenishmentsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list petty_cash_replenishments: %w", err)
	}
	var items []*pettycashreplenishmentpb.PettyCashReplenishment
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal petty_cash_replenishment row: %v", err)
			continue
		}
		item := &pettycashreplenishmentpb.PettyCashReplenishment{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, item); err != nil {
			log.Printf("WARN: protojson unmarshal petty_cash_replenishment: %v", err)
			continue
		}
		items = append(items, item)
	}
	return &pettycashreplenishmentpb.ListPettyCashReplenishmentsResponse{
		Success: true,
		Data:    items,
	}, nil
}

// GetPettyCashReplenishmentListPageData retrieves petty_cash_replenishments with pagination and search.
//
// Dialect changes: $N → ?; $1::text IS NULL → ? IS NULL; ILIKE → LIKE.
func (r *MySQLPettyCashReplenishmentRepository) GetPettyCashReplenishmentListPageData(
	ctx context.Context,
	req *pettycashreplenishmentpb.GetPettyCashReplenishmentListPageDataRequest,
) (*pettycashreplenishmentpb.GetPettyCashReplenishmentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get petty_cash_replenishment list page data request is required")
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

	orderBy := "pcr.date_created DESC"
	if req.Sort != nil {
		ob, err := mysqlCore.BuildOrderBy(
			[]string{"pcr.date_created", "pcr.replenishment_date", "pcr.amount", "pcr.replenishment_number"},
			req.Sort,
			"pcr.date_created DESC",
		)
		if err == nil {
			orderBy = ob
		}
	}

	// Args: searchPattern x2, limit, offset
	query := `
		WITH enriched AS (
			SELECT
				pcr.id,
				pcr.date_created,
				pcr.fund_id,
				pcr.replenishment_number,
				pcr.amount,
				pcr.replenishment_date,
				pcr.posted_by,
				pcr.notes
			FROM petty_cash_replenishment pcr
			WHERE (? IS NULL OR ? = '' OR
			       pcr.replenishment_number LIKE ? OR
			       pcr.notes LIKE ?)
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT e.*, c.total
		FROM enriched e, counted c
		ORDER BY ` + orderBy + `
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query petty_cash_replenishment list page data: %w", err)
	}
	defer rows.Close()

	var items []*pettycashreplenishmentpb.PettyCashReplenishment
	var totalCount int64

	for rows.Next() {
		var (
			id                  string
			dateCreated         time.Time
			fundID              string
			replenishmentNumber string
			amount              int64
			replenishmentDate   *time.Time
			postedBy            *string
			notes               *string
			total               int64
		)
		if err := rows.Scan(
			&id, &dateCreated, &fundID, &replenishmentNumber,
			&amount, &replenishmentDate, &postedBy, &notes, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan petty_cash_replenishment row: %w", err)
		}
		totalCount = total

		item := &pettycashreplenishmentpb.PettyCashReplenishment{
			Id:                  id,
			FundId:              fundID,
			ReplenishmentNumber: replenishmentNumber,
			Amount:              amount,
		}
		if postedBy != nil {
			item.PostedBy = postedBy
		}
		if notes != nil {
			item.Notes = notes
		}
		if replenishmentDate != nil && !replenishmentDate.IsZero() {
			item.ReplenishmentDate = replenishmentDate.UnixMilli()
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			item.DateCreated = &ts
			s := dateCreated.Format(time.RFC3339)
			item.DateCreatedString = &s
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating petty_cash_replenishment rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &pettycashreplenishmentpb.GetPettyCashReplenishmentListPageDataResponse{
		PettyCashReplenishmentList: items,
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

// GetPettyCashReplenishmentItemPageData retrieves a single petty_cash_replenishment by ID.
func (r *MySQLPettyCashReplenishmentRepository) GetPettyCashReplenishmentItemPageData(
	ctx context.Context,
	req *pettycashreplenishmentpb.GetPettyCashReplenishmentItemPageDataRequest,
) (*pettycashreplenishmentpb.GetPettyCashReplenishmentItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get petty_cash_replenishment item page data request is required")
	}
	if req.PettyCashReplenishmentId == "" {
		return nil, fmt.Errorf("petty_cash_replenishment ID is required")
	}

	const query = `
		SELECT
			pcr.id,
			pcr.date_created,
			pcr.fund_id,
			pcr.replenishment_number,
			pcr.amount,
			pcr.replenishment_date,
			pcr.posted_by,
			pcr.notes
		FROM petty_cash_replenishment pcr
		WHERE pcr.id = ?
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, req.PettyCashReplenishmentId)

	var (
		id                  string
		dateCreated         time.Time
		fundID              string
		replenishmentNumber string
		amount              int64
		replenishmentDate   *time.Time
		postedBy            *string
		notes               *string
	)

	err := row.Scan(
		&id, &dateCreated, &fundID, &replenishmentNumber,
		&amount, &replenishmentDate, &postedBy, &notes,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("petty_cash_replenishment with ID '%s' not found", req.PettyCashReplenishmentId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query petty_cash_replenishment item page data: %w", err)
	}

	item := &pettycashreplenishmentpb.PettyCashReplenishment{
		Id:                  id,
		FundId:              fundID,
		ReplenishmentNumber: replenishmentNumber,
		Amount:              amount,
	}
	if postedBy != nil {
		item.PostedBy = postedBy
	}
	if notes != nil {
		item.Notes = notes
	}
	if replenishmentDate != nil && !replenishmentDate.IsZero() {
		item.ReplenishmentDate = replenishmentDate.UnixMilli()
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		item.DateCreated = &ts
		s := dateCreated.Format(time.RFC3339)
		item.DateCreatedString = &s
	}

	return &pettycashreplenishmentpb.GetPettyCashReplenishmentItemPageDataResponse{
		PettyCashReplenishment: item,
		Success:                true,
	}, nil
}
