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
	expenditureattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "expenditure_attribute", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres expenditure_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresExpenditureAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresExpenditureAttributeRepository implements expenditure attribute CRUD operations using PostgreSQL
type PostgresExpenditureAttributeRepository struct {
	expenditureattributepb.UnimplementedExpenditureAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresExpenditureAttributeRepository creates a new PostgreSQL expenditure attribute repository
func NewPostgresExpenditureAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) expenditureattributepb.ExpenditureAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "expenditure_attribute"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresExpenditureAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateExpenditureAttribute creates a new expenditure attribute record
func (r *PostgresExpenditureAttributeRepository) CreateExpenditureAttribute(ctx context.Context, req *expenditureattributepb.CreateExpenditureAttributeRequest) (*expenditureattributepb.CreateExpenditureAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("expenditure attribute data is required")
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
		return nil, fmt.Errorf("failed to create expenditure attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attribute := &expenditureattributepb.ExpenditureAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &expenditureattributepb.CreateExpenditureAttributeResponse{
		Success: true,
		Data:    []*expenditureattributepb.ExpenditureAttribute{attribute},
	}, nil
}

// ReadExpenditureAttribute retrieves an expenditure attribute record by ID
func (r *PostgresExpenditureAttributeRepository) ReadExpenditureAttribute(ctx context.Context, req *expenditureattributepb.ReadExpenditureAttributeRequest) (*expenditureattributepb.ReadExpenditureAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure attribute ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expenditure attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attribute := &expenditureattributepb.ExpenditureAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &expenditureattributepb.ReadExpenditureAttributeResponse{
		Success: true,
		Data:    []*expenditureattributepb.ExpenditureAttribute{attribute},
	}, nil
}

// UpdateExpenditureAttribute updates an expenditure attribute record
func (r *PostgresExpenditureAttributeRepository) UpdateExpenditureAttribute(ctx context.Context, req *expenditureattributepb.UpdateExpenditureAttributeRequest) (*expenditureattributepb.UpdateExpenditureAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure attribute ID is required")
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
		return nil, fmt.Errorf("failed to update expenditure attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	attribute := &expenditureattributepb.ExpenditureAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &expenditureattributepb.UpdateExpenditureAttributeResponse{
		Success: true,
		Data:    []*expenditureattributepb.ExpenditureAttribute{attribute},
	}, nil
}

// DeleteExpenditureAttribute deletes an expenditure attribute record (soft delete)
func (r *PostgresExpenditureAttributeRepository) DeleteExpenditureAttribute(ctx context.Context, req *expenditureattributepb.DeleteExpenditureAttributeRequest) (*expenditureattributepb.DeleteExpenditureAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure attribute ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete expenditure attribute: %w", err)
	}

	return &expenditureattributepb.DeleteExpenditureAttributeResponse{
		Success: true,
	}, nil
}

// ListExpenditureAttributes lists expenditure attribute records with optional filters
func (r *PostgresExpenditureAttributeRepository) ListExpenditureAttributes(ctx context.Context, req *expenditureattributepb.ListExpenditureAttributesRequest) (*expenditureattributepb.ListExpenditureAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expenditure attributes: %w", err)
	}

	var attributes []*expenditureattributepb.ExpenditureAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal expenditure attribute row: %v", err)
			continue
		}

		attribute := &expenditureattributepb.ExpenditureAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attribute); err != nil {
			log.Printf("WARN: protojson unmarshal expenditure attribute: %v", err)
			continue
		}
		attributes = append(attributes, attribute)
	}

	return &expenditureattributepb.ListExpenditureAttributesResponse{
		Success: true,
		Data:    attributes,
	}, nil
}
