//go:build sqlserver

package expenditure

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
	expenditurecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ExpenditureCategory, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver expenditure_category repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerExpenditureCategoryRepository(dbOps, tableName), nil
	})
}

// SQLServerExpenditureCategoryRepository implements expenditure_category CRUD using SQL Server.
type SQLServerExpenditureCategoryRepository struct {
	expenditurecategorypb.UnimplementedExpenditureCategoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

// NewSQLServerExpenditureCategoryRepository creates a new SQL Server expenditure category repository.
func NewSQLServerExpenditureCategoryRepository(dbOps interfaces.DatabaseOperation, tableName string) expenditurecategorypb.ExpenditureCategoryDomainServiceServer {
	if tableName == "" {
		tableName = "expenditure_category"
	}
	return &SQLServerExpenditureCategoryRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerExpenditureCategoryRepository) CreateExpenditureCategory(ctx context.Context, req *expenditurecategorypb.CreateExpenditureCategoryRequest) (*expenditurecategorypb.CreateExpenditureCategoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("expenditure category data is required")
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
		return nil, fmt.Errorf("failed to create expenditure category: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	category := &expenditurecategorypb.ExpenditureCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &expenditurecategorypb.CreateExpenditureCategoryResponse{
		Success: true,
		Data:    []*expenditurecategorypb.ExpenditureCategory{category},
	}, nil
}

func (r *SQLServerExpenditureCategoryRepository) ReadExpenditureCategory(ctx context.Context, req *expenditurecategorypb.ReadExpenditureCategoryRequest) (*expenditurecategorypb.ReadExpenditureCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure category ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expenditure category: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	category := &expenditurecategorypb.ExpenditureCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &expenditurecategorypb.ReadExpenditureCategoryResponse{Data: []*expenditurecategorypb.ExpenditureCategory{category}}, nil
}

func (r *SQLServerExpenditureCategoryRepository) UpdateExpenditureCategory(ctx context.Context, req *expenditurecategorypb.UpdateExpenditureCategoryRequest) (*expenditurecategorypb.UpdateExpenditureCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure category ID is required")
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
		return nil, fmt.Errorf("failed to update expenditure category: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	category := &expenditurecategorypb.ExpenditureCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &expenditurecategorypb.UpdateExpenditureCategoryResponse{Data: []*expenditurecategorypb.ExpenditureCategory{category}}, nil
}

func (r *SQLServerExpenditureCategoryRepository) DeleteExpenditureCategory(ctx context.Context, req *expenditurecategorypb.DeleteExpenditureCategoryRequest) (*expenditurecategorypb.DeleteExpenditureCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure category ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expenditure category: %w", err)
	}
	return &expenditurecategorypb.DeleteExpenditureCategoryResponse{Success: true}, nil
}

func (r *SQLServerExpenditureCategoryRepository) ListExpenditureCategories(ctx context.Context, req *expenditurecategorypb.ListExpenditureCategoriesRequest) (*expenditurecategorypb.ListExpenditureCategoriesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expenditure categories: %w", err)
	}
	var categories []*expenditurecategorypb.ExpenditureCategory
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal expenditure_category row: %v", err)
			continue
		}
		category := &expenditurecategorypb.ExpenditureCategory{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
			log.Printf("WARN: protojson unmarshal expenditure_category: %v", err)
			continue
		}
		categories = append(categories, category)
	}
	return &expenditurecategorypb.ListExpenditureCategoriesResponse{Data: categories}, nil
}
