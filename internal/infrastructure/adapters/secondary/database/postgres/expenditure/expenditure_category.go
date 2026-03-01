//go:build postgresql

package expenditure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	expenditurecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "expenditure_category", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres expenditure_category repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresExpenditureCategoryRepository(dbOps, tableName), nil
	})
}

// PostgresExpenditureCategoryRepository implements expenditure category CRUD operations using PostgreSQL
type PostgresExpenditureCategoryRepository struct {
	expenditurecategorypb.UnimplementedExpenditureCategoryDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresExpenditureCategoryRepository creates a new PostgreSQL expenditure category repository
func NewPostgresExpenditureCategoryRepository(dbOps interfaces.DatabaseOperation, tableName string) expenditurecategorypb.ExpenditureCategoryDomainServiceServer {
	if tableName == "" {
		tableName = "expenditure_category"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresExpenditureCategoryRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateExpenditureCategory creates a new expenditure category record
func (r *PostgresExpenditureCategoryRepository) CreateExpenditureCategory(ctx context.Context, req *expenditurecategorypb.CreateExpenditureCategoryRequest) (*expenditurecategorypb.CreateExpenditureCategoryResponse, error) {
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

	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create expenditure category: %w", err)
	}

	resultJSON, err := json.Marshal(result)
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

// ReadExpenditureCategory retrieves an expenditure category record by ID
func (r *PostgresExpenditureCategoryRepository) ReadExpenditureCategory(ctx context.Context, req *expenditurecategorypb.ReadExpenditureCategoryRequest) (*expenditurecategorypb.ReadExpenditureCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure category ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expenditure category: %w", err)
	}

	resultJSON, err := json.Marshal(result)
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

// UpdateExpenditureCategory updates an expenditure category record
func (r *PostgresExpenditureCategoryRepository) UpdateExpenditureCategory(ctx context.Context, req *expenditurecategorypb.UpdateExpenditureCategoryRequest) (*expenditurecategorypb.UpdateExpenditureCategoryResponse, error) {
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

	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update expenditure category: %w", err)
	}

	resultJSON, err := json.Marshal(result)
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

// DeleteExpenditureCategory deletes an expenditure category record (soft delete)
func (r *PostgresExpenditureCategoryRepository) DeleteExpenditureCategory(ctx context.Context, req *expenditurecategorypb.DeleteExpenditureCategoryRequest) (*expenditurecategorypb.DeleteExpenditureCategoryResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure category ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete expenditure category: %w", err)
	}

	return &expenditurecategorypb.DeleteExpenditureCategoryResponse{
		Success: true,
	}, nil
}

// ListExpenditureCategories lists expenditure category records with optional filters
func (r *PostgresExpenditureCategoryRepository) ListExpenditureCategories(ctx context.Context, req *expenditurecategorypb.ListExpenditureCategoriesRequest) (*expenditurecategorypb.ListExpenditureCategoriesResponse, error) {
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
		resultJSON, err := json.Marshal(result)
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
