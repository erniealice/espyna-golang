//go:build mysql

// Dialect translation from postgres gold standard:
//   - $1,$2,... → ? (MySQL positional placeholders)
//   - active = true → active = 1
//   - convertMillisToTime uses single-arg form (jsonKey only)
package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	expenditurecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.ExpenditureCategory, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql expenditure_category repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLExpenditureCategoryRepository(dbOps, tableName), nil
	})
}

// MySQLExpenditureCategoryRepository implements expenditure category CRUD using MySQL 8.0+.
type MySQLExpenditureCategoryRepository struct {
	expenditurecategorypb.UnimplementedExpenditureCategoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewMySQLExpenditureCategoryRepository creates a new MySQL expenditure category repository.
func NewMySQLExpenditureCategoryRepository(dbOps interfaces.DatabaseOperation, tableName string) expenditurecategorypb.ExpenditureCategoryDomainServiceServer {
	if tableName == "" {
		tableName = "expenditure_category"
	}
	return &MySQLExpenditureCategoryRepository{
		dbOps:     dbOps,
		db:        getDB(dbOps),
		tableName: tableName,
	}
}

// CreateExpenditureCategory creates a new expenditure category record.
func (r *MySQLExpenditureCategoryRepository) CreateExpenditureCategory(ctx context.Context, req *expenditurecategorypb.CreateExpenditureCategoryRequest) (*expenditurecategorypb.CreateExpenditureCategoryResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("expenditure category data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expenditure category: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	category := &expenditurecategorypb.ExpenditureCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &expenditurecategorypb.CreateExpenditureCategoryResponse{
		Success: true,
		Data:    []*expenditurecategorypb.ExpenditureCategory{category},
	}, nil
}

// ReadExpenditureCategory retrieves an expenditure category record by ID.
func (r *MySQLExpenditureCategoryRepository) ReadExpenditureCategory(ctx context.Context, req *expenditurecategorypb.ReadExpenditureCategoryRequest) (*expenditurecategorypb.ReadExpenditureCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure category ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expenditure category: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	category := &expenditurecategorypb.ExpenditureCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &expenditurecategorypb.ReadExpenditureCategoryResponse{
		Success: true,
		Data:    []*expenditurecategorypb.ExpenditureCategory{category},
	}, nil
}

// UpdateExpenditureCategory updates an expenditure category record.
func (r *MySQLExpenditureCategoryRepository) UpdateExpenditureCategory(ctx context.Context, req *expenditurecategorypb.UpdateExpenditureCategoryRequest) (*expenditurecategorypb.UpdateExpenditureCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure category ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	convertMillisToTime(data, "dateCreated")
	convertMillisToTime(data, "dateModified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update expenditure category: %w", err)
	}

	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	category := &expenditurecategorypb.ExpenditureCategory{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &expenditurecategorypb.UpdateExpenditureCategoryResponse{
		Success: true,
		Data:    []*expenditurecategorypb.ExpenditureCategory{category},
	}, nil
}

// DeleteExpenditureCategory soft-deletes an expenditure category record.
func (r *MySQLExpenditureCategoryRepository) DeleteExpenditureCategory(ctx context.Context, req *expenditurecategorypb.DeleteExpenditureCategoryRequest) (*expenditurecategorypb.DeleteExpenditureCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure category ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expenditure category: %w", err)
	}

	return &expenditurecategorypb.DeleteExpenditureCategoryResponse{Success: true}, nil
}

// ListExpenditureCategories lists expenditure category records with optional filters.
func (r *MySQLExpenditureCategoryRepository) ListExpenditureCategories(ctx context.Context, req *expenditurecategorypb.ListExpenditureCategoriesRequest) (*expenditurecategorypb.ListExpenditureCategoriesResponse, error) {
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
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal expenditure category row: %v", err)
			continue
		}
		category := &expenditurecategorypb.ExpenditureCategory{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, category); err != nil {
			log.Printf("WARN: protojson unmarshal expenditure category: %v", err)
			continue
		}
		categories = append(categories, category)
	}

	return &expenditurecategorypb.ListExpenditureCategoriesResponse{
		Success: true,
		Data:    categories,
	}, nil
}
