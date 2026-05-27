//go:build sqlserver

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	linepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/line"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.Line, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver line repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerLineRepository(dbOps, tableName), nil
	})
}

// SQLServerLineRepository implements line CRUD using SQL Server.
type SQLServerLineRepository struct {
	linepb.UnimplementedLineDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerLineRepository creates a new SQL Server line repository.
func NewSQLServerLineRepository(dbOps interfaces.DatabaseOperation, tableName string) linepb.LineDomainServiceServer {
	if tableName == "" {
		tableName = "line"
	}
	return &SQLServerLineRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

func (r *SQLServerLineRepository) CreateLine(ctx context.Context, req *linepb.CreateLineRequest) (*linepb.CreateLineResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("line data is required")
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
		return nil, fmt.Errorf("failed to create line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &linepb.Line{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &linepb.CreateLineResponse{Data: []*linepb.Line{line}}, nil
}

func (r *SQLServerLineRepository) ReadLine(ctx context.Context, req *linepb.ReadLineRequest) (*linepb.ReadLineResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &linepb.Line{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &linepb.ReadLineResponse{Data: []*linepb.Line{line}}, nil
}

func (r *SQLServerLineRepository) UpdateLine(ctx context.Context, req *linepb.UpdateLineRequest) (*linepb.UpdateLineResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("line ID is required")
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
		return nil, fmt.Errorf("failed to update line: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &linepb.Line{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &linepb.UpdateLineResponse{Data: []*linepb.Line{line}}, nil
}

func (r *SQLServerLineRepository) DeleteLine(ctx context.Context, req *linepb.DeleteLineRequest) (*linepb.DeleteLineResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("line ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete line: %w", err)
	}
	return &linepb.DeleteLineResponse{Success: true}, nil
}

func (r *SQLServerLineRepository) ListLines(ctx context.Context, req *linepb.ListLinesRequest) (*linepb.ListLinesResponse, error) {
	var params *interfaces.ListParams
	if req != nil {
		params = &interfaces.ListParams{
			Search:     req.Search,
			Filters:    req.Filters,
			Sort:       req.Sort,
			Pagination: req.Pagination,
		}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list lines: %w", err)
	}
	lines := make([]*linepb.Line, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		line := &linepb.Line{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
			continue
		}
		lines = append(lines, line)
	}
	return &linepb.ListLinesResponse{Data: lines}, nil
}

func (r *SQLServerLineRepository) GetLineListPageData(ctx context.Context, req *linepb.GetLineListPageDataRequest) (*linepb.GetLineListPageDataResponse, error) {
	var params *interfaces.ListParams
	if req != nil {
		params = &interfaces.ListParams{
			Search:     req.Search,
			Filters:    req.Filters,
			Sort:       req.Sort,
			Pagination: req.Pagination,
		}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get line list page data: %w", err)
	}
	lines := make([]*linepb.Line, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		line := &linepb.Line{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
			continue
		}
		lines = append(lines, line)
	}
	return &linepb.GetLineListPageDataResponse{
		LineList:      lines,
		Pagination:    listResult.Pagination,
		SearchResults: []*commonpb.SearchResult{},
		Success:       true,
	}, nil
}

func (r *SQLServerLineRepository) GetLineItemPageData(ctx context.Context, req *linepb.GetLineItemPageDataRequest) (*linepb.GetLineItemPageDataResponse, error) {
	if req == nil || req.LineId == "" {
		return nil, fmt.Errorf("line ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.LineId)
	if err != nil {
		return nil, fmt.Errorf("failed to get line item page data: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	line := &linepb.Line{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, line); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &linepb.GetLineItemPageDataResponse{
		Line:    line,
		Success: true,
	}, nil
}
