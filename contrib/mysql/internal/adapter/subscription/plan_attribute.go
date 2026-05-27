//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
	"google.golang.org/protobuf/encoding/protojson"
)

// MySQLPlanAttributeRepository implements plan_attribute CRUD using MySQL 8.0+.
type MySQLPlanAttributeRepository struct {
	planattributepb.UnimplementedPlanAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.PlanAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql plan_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLPlanAttributeRepository(dbOps, tableName), nil
	})
}

// NewMySQLPlanAttributeRepository creates a new MySQL plan_attribute repository.
func NewMySQLPlanAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) planattributepb.PlanAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "plan_attribute"
	}
	return &MySQLPlanAttributeRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLPlanAttributeRepository) CreatePlanAttribute(ctx context.Context, req *planattributepb.CreatePlanAttributeRequest) (*planattributepb.CreatePlanAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan_attribute data is required")
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
		return nil, fmt.Errorf("failed to create plan_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pa := &planattributepb.PlanAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pa); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &planattributepb.CreatePlanAttributeResponse{Data: []*planattributepb.PlanAttribute{pa}}, nil
}

func (r *MySQLPlanAttributeRepository) ReadPlanAttribute(ctx context.Context, req *planattributepb.ReadPlanAttributeRequest) (*planattributepb.ReadPlanAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan_attribute ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pa := &planattributepb.PlanAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pa); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &planattributepb.ReadPlanAttributeResponse{Data: []*planattributepb.PlanAttribute{pa}}, nil
}

func (r *MySQLPlanAttributeRepository) UpdatePlanAttribute(ctx context.Context, req *planattributepb.UpdatePlanAttributeRequest) (*planattributepb.UpdatePlanAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan_attribute ID is required")
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
		return nil, fmt.Errorf("failed to update plan_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pa := &planattributepb.PlanAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pa); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &planattributepb.UpdatePlanAttributeResponse{Data: []*planattributepb.PlanAttribute{pa}}, nil
}

func (r *MySQLPlanAttributeRepository) DeletePlanAttribute(ctx context.Context, req *planattributepb.DeletePlanAttributeRequest) (*planattributepb.DeletePlanAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan_attribute ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete plan_attribute: %w", err)
	}
	return &planattributepb.DeletePlanAttributeResponse{Success: true}, nil
}

func (r *MySQLPlanAttributeRepository) ListPlanAttributes(ctx context.Context, req *planattributepb.ListPlanAttributesRequest) (*planattributepb.ListPlanAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan_attributes: %w", err)
	}
	var pas []*planattributepb.PlanAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		pa := &planattributepb.PlanAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pa); err != nil {
			continue
		}
		pas = append(pas, pa)
	}
	return &planattributepb.ListPlanAttributesResponse{Data: pas, Success: true}, nil
}

// GetPlanAttributeListPageData delegates to the dbOps List path.
// Dialect: workspace_id isolation is handled by WorkspaceAwareOperations.
func (r *MySQLPlanAttributeRepository) GetPlanAttributeListPageData(ctx context.Context, req *planattributepb.GetPlanAttributeListPageDataRequest) (*planattributepb.GetPlanAttributeListPageDataResponse, error) {
	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.GetFilters()
		params.Search = req.GetSearch()
		params.Sort = req.GetSort()
		params.Pagination = req.GetPagination()
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan_attributes: %w", err)
	}
	var pas []*planattributepb.PlanAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		pa := &planattributepb.PlanAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pa); err != nil {
			continue
		}
		pas = append(pas, pa)
	}
	currentPage := int32(1)
	totalPages := int32(1)
	return &planattributepb.GetPlanAttributeListPageDataResponse{
		PlanAttributeList: pas,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(len(pas)),
			CurrentPage: &currentPage,
			TotalPages:  &totalPages,
		},
		Success: true,
	}, nil
}

// GetPlanAttributeItemPageData retrieves a single plan_attribute.
func (r *MySQLPlanAttributeRepository) GetPlanAttributeItemPageData(ctx context.Context, req *planattributepb.GetPlanAttributeItemPageDataRequest) (*planattributepb.GetPlanAttributeItemPageDataResponse, error) {
	if req == nil || req.PlanAttributeId == "" {
		return nil, fmt.Errorf("plan_attribute ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.PlanAttributeId)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan_attribute: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	pa := &planattributepb.PlanAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, pa); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &planattributepb.GetPlanAttributeItemPageDataResponse{PlanAttribute: pa, Success: true}, nil
}
