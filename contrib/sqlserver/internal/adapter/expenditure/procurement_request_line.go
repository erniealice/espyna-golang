//go:build sqlserver

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	procurementrequestlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/procurement_request_line"
)

// procurementRequestLineSortableSQLCols is the fail-closed sort whitelist (A2).
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
	registry.RegisterRepositoryFactory("sqlserver", entityid.ProcurementRequestLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver procurement_request_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerProcurementRequestLineRepository(db, dbOps, tableName), nil
	})
}

// SQLServerProcurementRequestLineRepository implements procurement request line CRUD using SQL Server.
type SQLServerProcurementRequestLineRepository struct {
	procurementrequestlinepb.UnimplementedProcurementRequestLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerProcurementRequestLineRepository creates a new SQL Server procurement request line repository.
func NewSQLServerProcurementRequestLineRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) procurementrequestlinepb.ProcurementRequestLineDomainServiceServer {
	if tableName == "" {
		tableName = "procurement_request_line"
	}
	return &SQLServerProcurementRequestLineRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *SQLServerProcurementRequestLineRepository) CreateProcurementRequestLine(ctx context.Context, req *procurementrequestlinepb.CreateProcurementRequestLineRequest) (*procurementrequestlinepb.CreateProcurementRequestLineResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("procurement request line data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create procurement_request_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	line := &procurementrequestlinepb.ProcurementRequestLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &procurementrequestlinepb.CreateProcurementRequestLineResponse{Success: true, Data: []*procurementrequestlinepb.ProcurementRequestLine{line}}, nil
}

func (r *SQLServerProcurementRequestLineRepository) ReadProcurementRequestLine(ctx context.Context, req *procurementrequestlinepb.ReadProcurementRequestLineRequest) (*procurementrequestlinepb.ReadProcurementRequestLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read procurement_request_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	line := &procurementrequestlinepb.ProcurementRequestLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &procurementrequestlinepb.ReadProcurementRequestLineResponse{Success: true, Data: []*procurementrequestlinepb.ProcurementRequestLine{line}}, nil
}

func (r *SQLServerProcurementRequestLineRepository) UpdateProcurementRequestLine(ctx context.Context, req *procurementrequestlinepb.UpdateProcurementRequestLineRequest) (*procurementrequestlinepb.UpdateProcurementRequestLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request line ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update procurement_request_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	line := &procurementrequestlinepb.ProcurementRequestLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &procurementrequestlinepb.UpdateProcurementRequestLineResponse{Success: true, Data: []*procurementrequestlinepb.ProcurementRequestLine{line}}, nil
}

func (r *SQLServerProcurementRequestLineRepository) DeleteProcurementRequestLine(ctx context.Context, req *procurementrequestlinepb.DeleteProcurementRequestLineRequest) (*procurementrequestlinepb.DeleteProcurementRequestLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("procurement request line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete procurement_request_line: %w", err)
	}
	return &procurementrequestlinepb.DeleteProcurementRequestLineResponse{Success: true}, nil
}

func (r *SQLServerProcurementRequestLineRepository) ListProcurementRequestLines(ctx context.Context, req *procurementrequestlinepb.ListProcurementRequestLinesRequest) (*procurementrequestlinepb.ListProcurementRequestLinesResponse, error) {
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
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
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
func (r *SQLServerProcurementRequestLineRepository) GetProcurementRequestLineListPageData(ctx context.Context, req *procurementrequestlinepb.GetProcurementRequestLineListPageDataRequest) (*procurementrequestlinepb.GetProcurementRequestLineListPageDataResponse, error) {
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

	orderBy, err := sqlserverCore.BuildOrderBy(procurementRequestLineSortableSQLCols, req.GetSort(), "line_number ASC")
	if err != nil {
		return nil, err
	}

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
				[prl].[id],
				[prl].[procurement_request_id],
				[prl].[description],
				[prl].[line_number],
				[prl].[quantity],
				[prl].[estimated_unit_price],
				[prl].[estimated_total_price],
				[prl].[active],
				[prl].[date_created],
				[prl].[date_modified],
				COUNT(*) OVER() AS [total]
			FROM [procurement_request_line] [prl]
			WHERE [prl].[active] = 1
			  AND (@p1 IS NULL OR @p1 = '' OR [prl].[procurement_request_id] = @p1)
		)
		SELECT * FROM enriched
		` + orderBy + `
		OFFSET @p2 ROWS FETCH NEXT @p3 ROWS ONLY
	`

	rows, err := r.db.QueryContext(ctx, query, procurementRequestID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query procurement_request_line list page data: %w", err)
	}
	defer rows.Close()

	var lines []*procurementrequestlinepb.ProcurementRequestLine
	var totalCount int64

	for rows.Next() {
		var (
			id                  string
			procReqID           string
			description         string
			lineNumber          int32
			quantity            float64
			estimatedUnitPrice  int64
			estimatedTotalPrice int64
			active              bool
			dateCreated         time.Time
			dateModified        time.Time
			total               int64
		)
		if err := rows.Scan(
			&id, &procReqID, &description, &lineNumber,
			&quantity, &estimatedUnitPrice, &estimatedTotalPrice, &active,
			&dateCreated, &dateModified, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan procurement_request_line row: %w", err)
		}
		totalCount = total
		line := &procurementrequestlinepb.ProcurementRequestLine{
			Id:                   id,
			ProcurementRequestId: procReqID,
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
func (r *SQLServerProcurementRequestLineRepository) GetProcurementRequestLineItemPageData(ctx context.Context, req *procurementrequestlinepb.GetProcurementRequestLineItemPageDataRequest) (*procurementrequestlinepb.GetProcurementRequestLineItemPageDataResponse, error) {
	if req == nil || req.GetProcurementRequestLineId() == "" {
		return nil, fmt.Errorf("procurement request line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetProcurementRequestLineId())
	if err != nil {
		return nil, fmt.Errorf("failed to read procurement_request_line item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	line := &procurementrequestlinepb.ProcurementRequestLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &procurementrequestlinepb.GetProcurementRequestLineItemPageDataResponse{
		ProcurementRequestLine: line,
		Success:                true,
	}, nil
}
