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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

// supplierContractLineSortableSQLCols is the fail-closed sort whitelist (A2).
var supplierContractLineSortableSQLCols = []string{
	"supplier_contract_id",
	"description",
	"line_number",
	"treatment",
	"quantity",
	"unit_price",
	"total_amount",
	"date_created",
	"date_modified",
}

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.SupplierContractLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver supplier_contract_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerSupplierContractLineRepository(db, dbOps, tableName), nil
	})
}

// SQLServerSupplierContractLineRepository implements supplier contract line CRUD using SQL Server.
type SQLServerSupplierContractLineRepository struct {
	suppliercontractlinepb.UnimplementedSupplierContractLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerSupplierContractLineRepository creates a new SQL Server supplier contract line repository.
func NewSQLServerSupplierContractLineRepository(db *sql.DB, dbOps interfaces.DatabaseOperation, tableName string) suppliercontractlinepb.SupplierContractLineDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_contract_line"
	}
	return &SQLServerSupplierContractLineRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *SQLServerSupplierContractLineRepository) CreateSupplierContractLine(ctx context.Context, req *suppliercontractlinepb.CreateSupplierContractLineRequest) (*suppliercontractlinepb.CreateSupplierContractLineResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier contract line data is required")
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
		return nil, fmt.Errorf("failed to create supplier_contract_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	line := &suppliercontractlinepb.SupplierContractLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &suppliercontractlinepb.CreateSupplierContractLineResponse{Success: true, Data: []*suppliercontractlinepb.SupplierContractLine{line}}, nil
}

func (r *SQLServerSupplierContractLineRepository) ReadSupplierContractLine(ctx context.Context, req *suppliercontractlinepb.ReadSupplierContractLineRequest) (*suppliercontractlinepb.ReadSupplierContractLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	line := &suppliercontractlinepb.SupplierContractLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &suppliercontractlinepb.ReadSupplierContractLineResponse{Success: true, Data: []*suppliercontractlinepb.SupplierContractLine{line}}, nil
}

func (r *SQLServerSupplierContractLineRepository) UpdateSupplierContractLine(ctx context.Context, req *suppliercontractlinepb.UpdateSupplierContractLineRequest) (*suppliercontractlinepb.UpdateSupplierContractLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract line ID is required")
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
		return nil, fmt.Errorf("failed to update supplier_contract_line: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	line := &suppliercontractlinepb.SupplierContractLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &suppliercontractlinepb.UpdateSupplierContractLineResponse{Success: true, Data: []*suppliercontractlinepb.SupplierContractLine{line}}, nil
}

func (r *SQLServerSupplierContractLineRepository) DeleteSupplierContractLine(ctx context.Context, req *suppliercontractlinepb.DeleteSupplierContractLineRequest) (*suppliercontractlinepb.DeleteSupplierContractLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_contract_line: %w", err)
	}
	return &suppliercontractlinepb.DeleteSupplierContractLineResponse{Success: true}, nil
}

func (r *SQLServerSupplierContractLineRepository) ListSupplierContractLines(ctx context.Context, req *suppliercontractlinepb.ListSupplierContractLinesRequest) (*suppliercontractlinepb.ListSupplierContractLinesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list supplier_contract_lines: %w", err)
	}
	var lines []*suppliercontractlinepb.SupplierContractLine
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal supplier_contract_line row: %v", err)
			continue
		}
		line := &suppliercontractlinepb.SupplierContractLine{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
			log.Printf("WARN: protojson unmarshal supplier_contract_line: %v", err)
			continue
		}
		lines = append(lines, line)
	}
	return &suppliercontractlinepb.ListSupplierContractLinesResponse{Success: true, Data: lines}, nil
}

// GetSupplierContractLineListPageData retrieves contract lines with pagination.
func (r *SQLServerSupplierContractLineRepository) GetSupplierContractLineListPageData(ctx context.Context, req *suppliercontractlinepb.GetSupplierContractLineListPageDataRequest) (*suppliercontractlinepb.GetSupplierContractLineListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get supplier contract line list page data request is required")
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

	orderBy, err := sqlserverCore.BuildOrderBy(supplierContractLineSortableSQLCols, req.GetSort(), "line_number ASC")
	if err != nil {
		return nil, err
	}

	supplierContractID := ""
	if req.Filters != nil {
		for _, f := range req.Filters.GetFilters() {
			if f.GetField() == "supplier_contract_id" && f.GetStringFilter() != nil {
				supplierContractID = f.GetStringFilter().GetValue()
			}
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				[scl].[id],
				[scl].[supplier_contract_id],
				[scl].[description],
				[scl].[line_number],
				[scl].[treatment],
				[scl].[quantity],
				[scl].[unit_price],
				[scl].[total_amount],
				[scl].[active],
				[scl].[date_created],
				[scl].[date_modified],
				COUNT(*) OVER() AS [total]
			FROM [supplier_contract_line] [scl]
			WHERE [scl].[active] = 1
			  AND (@p1 IS NULL OR @p1 = '' OR [scl].[supplier_contract_id] = @p1)
		)
		SELECT * FROM enriched
		` + orderBy + `
		OFFSET @p2 ROWS FETCH NEXT @p3 ROWS ONLY
	`

	rows, err := r.db.QueryContext(ctx, query, supplierContractID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query supplier_contract_line list page data: %w", err)
	}
	defer rows.Close()

	var lines []*suppliercontractlinepb.SupplierContractLine
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			supplierContractID string
			description        string
			lineNumber         int32
			treatment          int32
			quantity           float64
			unitPrice          int64
			totalAmount        int64
			active             bool
			dateCreated        time.Time
			dateModified       time.Time
			total              int64
		)
		if err := rows.Scan(
			&id, &supplierContractID, &description, &lineNumber, &treatment,
			&quantity, &unitPrice, &totalAmount, &active,
			&dateCreated, &dateModified, &total,
		); err != nil {
			return nil, fmt.Errorf("failed to scan supplier_contract_line row: %w", err)
		}
		totalCount = total
		line := &suppliercontractlinepb.SupplierContractLine{
			Id:                 id,
			SupplierContractId: supplierContractID,
			Description:        description,
			LineNumber:         lineNumber,
			Treatment:          suppliercontractlinepb.SupplierContractLineTreatment(treatment),
			Quantity:           quantity,
			UnitPrice:          unitPrice,
			TotalAmount:        totalAmount,
			Active:             active,
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
		return nil, fmt.Errorf("error iterating supplier_contract_line rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}
	hasNext := page < totalPages
	hasPrev := page > 1

	return &suppliercontractlinepb.GetSupplierContractLineListPageDataResponse{
		SupplierContractLineList: lines,
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

// GetSupplierContractLineItemPageData retrieves a single contract line.
func (r *SQLServerSupplierContractLineRepository) GetSupplierContractLineItemPageData(ctx context.Context, req *suppliercontractlinepb.GetSupplierContractLineItemPageDataRequest) (*suppliercontractlinepb.GetSupplierContractLineItemPageDataResponse, error) {
	if req == nil || req.GetSupplierContractLineId() == "" {
		return nil, fmt.Errorf("supplier contract line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetSupplierContractLineId())
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_line item: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	line := &suppliercontractlinepb.SupplierContractLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &suppliercontractlinepb.GetSupplierContractLineItemPageDataResponse{SupplierContractLine: line, Success: true}, nil
}
