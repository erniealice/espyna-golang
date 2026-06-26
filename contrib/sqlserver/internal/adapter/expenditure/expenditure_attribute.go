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
	expenditureattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.ExpenditureAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver expenditure_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerExpenditureAttributeRepository(dbOps, tableName), nil
	})
}

// SQLServerExpenditureAttributeRepository implements expenditure_attribute CRUD using SQL Server.
type SQLServerExpenditureAttributeRepository struct {
	expenditureattributepb.UnimplementedExpenditureAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewSQLServerExpenditureAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) expenditureattributepb.ExpenditureAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "expenditure_attribute"
	}
	return &SQLServerExpenditureAttributeRepository{dbOps: dbOps, tableName: tableName}
}

func (r *SQLServerExpenditureAttributeRepository) CreateExpenditureAttribute(ctx context.Context, req *expenditureattributepb.CreateExpenditureAttributeRequest) (*expenditureattributepb.CreateExpenditureAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("expenditure attribute data is required")
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
		return nil, fmt.Errorf("failed to create expenditure attribute: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	attr := &expenditureattributepb.ExpenditureAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &expenditureattributepb.CreateExpenditureAttributeResponse{Data: []*expenditureattributepb.ExpenditureAttribute{attr}}, nil
}

func (r *SQLServerExpenditureAttributeRepository) ReadExpenditureAttribute(ctx context.Context, req *expenditureattributepb.ReadExpenditureAttributeRequest) (*expenditureattributepb.ReadExpenditureAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure attribute ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read expenditure attribute: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	attr := &expenditureattributepb.ExpenditureAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &expenditureattributepb.ReadExpenditureAttributeResponse{Data: []*expenditureattributepb.ExpenditureAttribute{attr}}, nil
}

func (r *SQLServerExpenditureAttributeRepository) UpdateExpenditureAttribute(ctx context.Context, req *expenditureattributepb.UpdateExpenditureAttributeRequest) (*expenditureattributepb.UpdateExpenditureAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure attribute ID is required")
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
		return nil, fmt.Errorf("failed to update expenditure attribute: %w", err)
	}
	resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	attr := &expenditureattributepb.ExpenditureAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &expenditureattributepb.UpdateExpenditureAttributeResponse{Data: []*expenditureattributepb.ExpenditureAttribute{attr}}, nil
}

func (r *SQLServerExpenditureAttributeRepository) DeleteExpenditureAttribute(ctx context.Context, req *expenditureattributepb.DeleteExpenditureAttributeRequest) (*expenditureattributepb.DeleteExpenditureAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("expenditure attribute ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete expenditure attribute: %w", err)
	}
	return &expenditureattributepb.DeleteExpenditureAttributeResponse{Success: true}, nil
}

func (r *SQLServerExpenditureAttributeRepository) ListExpenditureAttributes(ctx context.Context, req *expenditureattributepb.ListExpenditureAttributesRequest) (*expenditureattributepb.ListExpenditureAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list expenditure attributes: %w", err)
	}
	var attrs []*expenditureattributepb.ExpenditureAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(sqlserverCore.DenormalizeKeys(result))
		if err != nil {
			log.Printf("WARN: json.Marshal expenditure_attribute row: %v", err)
			continue
		}
		attr := &expenditureattributepb.ExpenditureAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, attr); err != nil {
			log.Printf("WARN: protojson unmarshal expenditure_attribute: %v", err)
			continue
		}
		attrs = append(attrs, attr)
	}
	return &expenditureattributepb.ListExpenditureAttributesResponse{Data: attrs}, nil
}
