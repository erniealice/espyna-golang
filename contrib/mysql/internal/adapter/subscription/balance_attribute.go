//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	balanceattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/balance_attribute"
	"google.golang.org/protobuf/encoding/protojson"
)

// MySQLBalanceAttributeRepository implements balance_attribute CRUD using MySQL 8.0+.
type MySQLBalanceAttributeRepository struct {
	balanceattributepb.UnimplementedBalanceAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.BalanceAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql balance_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLBalanceAttributeRepository(dbOps, tableName), nil
	})
}

// NewMySQLBalanceAttributeRepository creates a new MySQL balance_attribute repository.
func NewMySQLBalanceAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) balanceattributepb.BalanceAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "balance_attribute"
	}
	return &MySQLBalanceAttributeRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLBalanceAttributeRepository) CreateBalanceAttribute(ctx context.Context, req *balanceattributepb.CreateBalanceAttributeRequest) (*balanceattributepb.CreateBalanceAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("balance_attribute data is required")
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
		return nil, fmt.Errorf("failed to create balance_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ba := &balanceattributepb.BalanceAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ba); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &balanceattributepb.CreateBalanceAttributeResponse{Data: []*balanceattributepb.BalanceAttribute{ba}}, nil
}

func (r *MySQLBalanceAttributeRepository) ReadBalanceAttribute(ctx context.Context, req *balanceattributepb.ReadBalanceAttributeRequest) (*balanceattributepb.ReadBalanceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance_attribute ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read balance_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ba := &balanceattributepb.BalanceAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ba); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &balanceattributepb.ReadBalanceAttributeResponse{Data: []*balanceattributepb.BalanceAttribute{ba}}, nil
}

func (r *MySQLBalanceAttributeRepository) UpdateBalanceAttribute(ctx context.Context, req *balanceattributepb.UpdateBalanceAttributeRequest) (*balanceattributepb.UpdateBalanceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance_attribute ID is required")
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
		return nil, fmt.Errorf("failed to update balance_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ba := &balanceattributepb.BalanceAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ba); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &balanceattributepb.UpdateBalanceAttributeResponse{Data: []*balanceattributepb.BalanceAttribute{ba}}, nil
}

func (r *MySQLBalanceAttributeRepository) DeleteBalanceAttribute(ctx context.Context, req *balanceattributepb.DeleteBalanceAttributeRequest) (*balanceattributepb.DeleteBalanceAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("balance_attribute ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete balance_attribute: %w", err)
	}
	return &balanceattributepb.DeleteBalanceAttributeResponse{Success: true}, nil
}

func (r *MySQLBalanceAttributeRepository) ListBalanceAttributes(ctx context.Context, req *balanceattributepb.ListBalanceAttributesRequest) (*balanceattributepb.ListBalanceAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list balance_attributes: %w", err)
	}
	var bas []*balanceattributepb.BalanceAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		ba := &balanceattributepb.BalanceAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ba); err != nil {
			continue
		}
		bas = append(bas, ba)
	}
	return &balanceattributepb.ListBalanceAttributesResponse{Data: bas, Success: true}, nil
}
