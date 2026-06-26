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
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	suppliercontractlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/supplier_contract_line"
)

// supplierContractLineSortableSQLCols is the fail-closed sort whitelist for
// GetSupplierContractLineListPageData (A2). Mirrors the enriched CTE projection.
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
	registry.RegisterRepositoryFactory("postgresql", entityid.SupplierContractLine, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres supplier_contract_line repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSupplierContractLineRepository(dbOps, tableName), nil
	})
}

// PostgresSupplierContractLineRepository implements supplier contract line CRUD using PostgreSQL.
type PostgresSupplierContractLineRepository struct {
	suppliercontractlinepb.UnimplementedSupplierContractLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresSupplierContractLineRepository creates a new PostgreSQL supplier contract line repository.
func NewPostgresSupplierContractLineRepository(dbOps interfaces.DatabaseOperation, tableName string) suppliercontractlinepb.SupplierContractLineDomainServiceServer {
	if tableName == "" {
		tableName = "supplier_contract_line"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSupplierContractLineRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateSupplierContractLine creates a new supplier contract line record.
func (r *PostgresSupplierContractLineRepository) CreateSupplierContractLine(ctx context.Context, req *suppliercontractlinepb.CreateSupplierContractLineRequest) (*suppliercontractlinepb.CreateSupplierContractLineResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("supplier contract line data is required")
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
		return nil, fmt.Errorf("failed to create supplier_contract_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &suppliercontractlinepb.SupplierContractLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &suppliercontractlinepb.CreateSupplierContractLineResponse{Success: true, Data: []*suppliercontractlinepb.SupplierContractLine{line}}, nil
}

// ReadSupplierContractLine retrieves a supplier contract line by ID.
func (r *PostgresSupplierContractLineRepository) ReadSupplierContractLine(ctx context.Context, req *suppliercontractlinepb.ReadSupplierContractLineRequest) (*suppliercontractlinepb.ReadSupplierContractLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &suppliercontractlinepb.SupplierContractLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &suppliercontractlinepb.ReadSupplierContractLineResponse{Success: true, Data: []*suppliercontractlinepb.SupplierContractLine{line}}, nil
}

// UpdateSupplierContractLine updates a supplier contract line record.
func (r *PostgresSupplierContractLineRepository) UpdateSupplierContractLine(ctx context.Context, req *suppliercontractlinepb.UpdateSupplierContractLineRequest) (*suppliercontractlinepb.UpdateSupplierContractLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract line ID is required")
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
		return nil, fmt.Errorf("failed to update supplier_contract_line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &suppliercontractlinepb.SupplierContractLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &suppliercontractlinepb.UpdateSupplierContractLineResponse{Success: true, Data: []*suppliercontractlinepb.SupplierContractLine{line}}, nil
}

// DeleteSupplierContractLine soft-deletes a supplier contract line.
func (r *PostgresSupplierContractLineRepository) DeleteSupplierContractLine(ctx context.Context, req *suppliercontractlinepb.DeleteSupplierContractLineRequest) (*suppliercontractlinepb.DeleteSupplierContractLineResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("supplier contract line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete supplier_contract_line: %w", err)
	}
	return &suppliercontractlinepb.DeleteSupplierContractLineResponse{Success: true}, nil
}

// ListSupplierContractLines lists supplier contract line records with optional filters.
func (r *PostgresSupplierContractLineRepository) ListSupplierContractLines(ctx context.Context, req *suppliercontractlinepb.ListSupplierContractLinesRequest) (*suppliercontractlinepb.ListSupplierContractLinesResponse, error) {
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
		resultJSON, err := json.Marshal(result)
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
func (r *PostgresSupplierContractLineRepository) GetSupplierContractLineListPageData(
	ctx context.Context,
	req *suppliercontractlinepb.GetSupplierContractLineListPageDataRequest,
) (*suppliercontractlinepb.GetSupplierContractLineListPageDataResponse, error) {
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

	orderBy, err := postgresCore.BuildOrderBy(supplierContractLineSortableSQLCols, req.GetSort(), "line_number ASC")
	if err != nil {
		return nil, err
	}

	// Filter by supplier_contract_id if supplied via Filters (TypedFilter with field="supplier_contract_id").
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
				scl.id,
				scl.supplier_contract_id,
				scl.description,
				scl.line_number,
				scl.treatment,
				scl.quantity,
				scl.unit_price,
				scl.total_amount,
				scl.active,
				scl.date_created,
				scl.date_modified,
				COUNT(*) OVER() AS total
			FROM supplier_contract_line scl
			WHERE scl.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR scl.supplier_contract_id = $1)
		)
		SELECT * FROM enriched
		` + orderBy + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, supplierContractID, limit, offset)
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
func (r *PostgresSupplierContractLineRepository) GetSupplierContractLineItemPageData(
	ctx context.Context,
	req *suppliercontractlinepb.GetSupplierContractLineItemPageDataRequest,
) (*suppliercontractlinepb.GetSupplierContractLineItemPageDataResponse, error) {
	if req == nil || req.GetSupplierContractLineId() == "" {
		return nil, fmt.Errorf("supplier contract line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.GetSupplierContractLineId())
	if err != nil {
		return nil, fmt.Errorf("failed to read supplier_contract_line item: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &suppliercontractlinepb.SupplierContractLine{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &suppliercontractlinepb.GetSupplierContractLineItemPageDataResponse{
		SupplierContractLine: line,
		Success:              true,
	}, nil
}
