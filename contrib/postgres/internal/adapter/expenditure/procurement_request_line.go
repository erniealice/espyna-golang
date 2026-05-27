//go:build postgresql

package expenditure

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
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// procurementRequestLineSortableSQLCols is the fail-closed sort whitelist for
// GetProcurementRequestLineListPageData (A2). Mirrors the enriched CTE projection.
var procurementRequestLineSortableSQLCols = []string{
	"procurement_request_id",
	"description",
	"line_number",
	"quantity",
	"estimated_unit_price",
	"estimated_total_price",
	"date_created",
	"date_modified",
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ProcurementRequestLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres procurement_request_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresProcurementRequestLineRepository(dbOps, tableName), nil
	})
}

// PostgresProcurementRequestLineRepository implements procurement request line CRUD using PostgreSQL.
type PostgresProcurementRequestLineRepository struct {
	procurementrequestlinepb.UnimplementedProcurementRequestLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresProcurementRequestLineRepository creates a new PostgreSQL procurement request line repository.
func NewPostgresProcurementRequestLineRepository(dbOps interfaces.DatabaseOperation, tableName string) procurementrequestlinepb.ProcurementRequestLineDomainServiceServer {
	if tableName == "" {
		tableName = "procurement_request_line"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresProcurementRequestLineRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateProcurementRequestLine creates a new procurement request line record.
func (r *PostgresProcurementRequestLineRepository) CreateProcurementRequestLine(ctx context.Context, req *procurementrequestlinepb.CreateProcurementRequestLineRequest) (*procurementrequestlinepb.CreateProcurementRequestLineResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("procurement request line data is required")
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
		return nil, fmt.Errorf("failed to create procurement_request_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &procurementrequestlinepb.ProcurementRequestLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &procurementrequestlinepb.CreateProcurementRequestLineResponse{Success: true, Data: []*procurementrequestlinepb.ProcurementRequestLine{line}}, nil
}

// ReadProcurementRequestLine retrieves a procurement request line by ID.
func (r *PostgresProcurementRequestLineRepository) ReadProcurementRequestLine(ctx context.Context, req *procurementrequestlinepb.ReadProcurementRequestLineRequest) (*procurementrequestlinepb.ReadProcurementRequestLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read procurement_request_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &procurementrequestlinepb.ProcurementRequestLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &procurementrequestlinepb.ReadProcurementRequestLineResponse{Success: true, Data: []*procurementrequestlinepb.ProcurementRequestLine{line}}, nil
}

// UpdateProcurementRequestLine updates a procurement request line record.
func (r *PostgresProcurementRequestLineRepository) UpdateProcurementRequestLine(ctx context.Context, req *procurementrequestlinepb.UpdateProcurementRequestLineRequest) (*procurementrequestlinepb.UpdateProcurementRequestLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request line ID is required")
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
		return nil, fmt.Errorf("failed to update procurement_request_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &procurementrequestlinepb.ProcurementRequestLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &procurementrequestlinepb.UpdateProcurementRequestLineResponse{Success: true, Data: []*procurementrequestlinepb.ProcurementRequestLine{line}}, nil
}

// DeleteProcurementRequestLine soft-deletes a procurement request line.
func (r *PostgresProcurementRequestLineRepository) DeleteProcurementRequestLine(ctx context.Context, req *procurementrequestlinepb.DeleteProcurementRequestLineRequest) (*procurementrequestlinepb.DeleteProcurementRequestLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete procurement_request_line: %w", err)
	}
	return &procurementrequestlinepb.DeleteProcurementRequestLineResponse{Success: true}, nil
}

// ListProcurementRequestLines lists procurement request line records with optional filters.
func (r *PostgresProcurementRequestLineRepository) ListProcurementRequestLines(ctx context.Context, req *procurementrequestlinepb.ListProcurementRequestLinesRequest) (*procurementrequestlinepb.ListProcurementRequestLinesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list procurement_request_lines: %w", err)
	}
	var lines []*procurementrequestlinepb.ProcurementRequestLine
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal procurement_request_line row: %v", err)
			continue
		}
		line := &procurementrequestlinepb.ProcurementRequestLine{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
			log.Printf("WARN: protojson unmarshal procurement_request_line: %v", err)
			continue
		}
		lines = append(lines, line)
	}
	return &procurementrequestlinepb.ListProcurementRequestLinesResponse{Success: true, Data: lines}, nil
}

// GetProcurementRequestLineListPageData retrieves request lines with pagination.
func (r *PostgresProcurementRequestLineRepository) GetProcurementRequestLineListPageData(
	ctx context.Context,
	req *procurementrequestlinepb.GetProcurementRequestLineListPageDataRequest,
) (*procurementrequestlinepb.GetProcurementRequestLineListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get procurement request line list page data request is required")
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

	orderBy, err := postgresCore.BuildOrderBy(procurementRequestLineSortableSQLCols, req.GetSort(), "line_number ASC")
	if err != nil {
		return nil, err
	}

	// Filter by procurement_request_id if provided (TypedFilter with field="procurement_request_id").
	procurementRequestID := ""
	if req.Filters != nil {
		for _, f := range req.Filters.GetFilters() {
			if f.GetField() == "procurement_request_id" && f.GetStringFilter() != nil {
				procurementRequestID = f.GetStringFilter().GetValue()
			}
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				prl.id,
				prl.procurement_request_id,
				prl.description,
				prl.line_number,
				prl.quantity,
				prl.estimated_unit_price,
				prl.estimated_total_price,
				prl.active,
				prl.date_created,
				prl.date_modified,
				COUNT(*) OVER() AS total
			FROM procurement_request_line prl
			WHERE prl.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR prl.procurement_request_id = $1)
		)
		SELECT * FROM enriched
		` + orderBy + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, procurementRequestID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query procurement_request_line list page data: %w", err)
	}
	defer rows.Close()

	var lines []*procurementrequestlinepb.ProcurementRequestLine
	var totalCount int64

	for rows.Next() {
		var (
			id                   string
			procurementRequestID string
			description          string
			lineNumber           int32
			quantity             float64
			estimatedUnitPrice   int64
			estimatedTotalPrice  int64
			active               bool
			dateCreated          time.Time
			dateModified         time.Time
			total                int64
		)
		if err := rows.Scan(
			&id, &procurementRequestID, &description, &lineNumber,
			&quantity, &estimatedUnitPrice, &estimatedTotalPrice, &active,
			&dateCreated, &dateModified, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan procurement_request_line row: %w", err)
		}
		totalCount = total
		line := &procurementrequestlinepb.ProcurementRequestLine{
			Id:                   id,
			ProcurementRequestId: procurementRequestID,
			Description:          description,
			LineNumber:           lineNumber,
			Quantity:             quantity,
			EstimatedUnitPrice:   estimatedUnitPrice,
			EstimatedTotalPrice:  estimatedTotalPrice,
			Active:               active,
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			line.DateCreated = &ts
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			line.DateModified = &ts
		}
		lines = append(lines, line)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating procurement_request_line rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &procurementrequestlinepb.GetProcurementRequestLineListPageDataResponse{
		ProcurementRequestLineList: lines,
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

// GetProcurementRequestLineItemPageData retrieves a single procurement request line.
func (r *PostgresProcurementRequestLineRepository) GetProcurementRequestLineItemPageData(
	ctx context.Context,
	req *procurementrequestlinepb.GetProcurementRequestLineItemPageDataRequest,
) (*procurementrequestlinepb.GetProcurementRequestLineItemPageDataResponse, error) {
	if req == nil || req.GetProcurementRequestLineId() == "" {
		return nil, fmt.Errorf("procurement request line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetProcurementRequestLineId())
	if err != nil {
		return nil, fmt.Errorf("failed to read procurement_request_line item: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &procurementrequestlinepb.ProcurementRequestLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &procurementrequestlinepb.GetProcurementRequestLineItemPageDataResponse{
		ProcurementRequestLine: line,
		Success:                true,
	}, nil
}
