//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ScoringComponent, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres scoring_component repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresScoringComponentRepository(dbOps, tableName), nil
	})
}

// PostgresScoringComponentRepository implements scoring_component CRUD via PostgreSQL.
type PostgresScoringComponentRepository struct {
	pb.UnimplementedScoringComponentDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresScoringComponentRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ScoringComponentDomainServiceServer {
	if tableName == "" {
		tableName = "scoring_component"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresScoringComponentRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresScoringComponentRepository) CreateScoringComponent(ctx context.Context, req *pb.CreateScoringComponentRequest) (*pb.CreateScoringComponentResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("scoring component data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create scoring component: %w", err)
	}
	item, err := scoringComponentFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateScoringComponentResponse{Data: []*pb.ScoringComponent{item}, Success: true}, nil
}

func (r *PostgresScoringComponentRepository) ReadScoringComponent(ctx context.Context, req *pb.ReadScoringComponentRequest) (*pb.ReadScoringComponentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("scoring component ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read scoring component: %w", err)
	}
	item, err := scoringComponentFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadScoringComponentResponse{Data: []*pb.ScoringComponent{item}, Success: true}, nil
}

func (r *PostgresScoringComponentRepository) UpdateScoringComponent(ctx context.Context, req *pb.UpdateScoringComponentRequest) (*pb.UpdateScoringComponentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("scoring component ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update scoring component: %w", err)
	}
	item, err := scoringComponentFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateScoringComponentResponse{Data: []*pb.ScoringComponent{item}, Success: true}, nil
}

func (r *PostgresScoringComponentRepository) DeleteScoringComponent(ctx context.Context, req *pb.DeleteScoringComponentRequest) (*pb.DeleteScoringComponentResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("scoring component ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete scoring component: %w", err)
	}
	return &pb.DeleteScoringComponentResponse{Success: true}, nil
}

func (r *PostgresScoringComponentRepository) ListScoringComponents(ctx context.Context, req *pb.ListScoringComponentsRequest) (*pb.ListScoringComponentsResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListScoringComponentsResponse{Data: items, Success: true}, nil
}

func (r *PostgresScoringComponentRepository) GetScoringComponentListPageData(ctx context.Context, req *pb.GetScoringComponentListPageDataRequest) (*pb.GetScoringComponentListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	page, items, pagination := paginateScoringComponent(all, req.GetPagination())
	_ = page
	return &pb.GetScoringComponentListPageDataResponse{ScoringComponentList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresScoringComponentRepository) GetScoringComponentItemPageData(ctx context.Context, req *pb.GetScoringComponentItemPageDataRequest) (*pb.GetScoringComponentItemPageDataResponse, error) {
	if req == nil || req.ScoringComponentId == "" {
		return nil, fmt.Errorf("scoring component ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.ScoringComponentId)
	if err != nil {
		return nil, fmt.Errorf("failed to read scoring component: %w", err)
	}
	item, err := scoringComponentFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetScoringComponentItemPageDataResponse{ScoringComponent: item, Success: true}, nil
}

func (r *PostgresScoringComponentRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.ScoringComponent, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list scoring components: %w", err)
	}
	var items []*pb.ScoringComponent
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.ScoringComponent{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func scoringComponentFromResult(result any) (*pb.ScoringComponent, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.ScoringComponent{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateScoringComponent(all []*pb.ScoringComponent, p *commonpb.PaginationRequest) (int32, []*pb.ScoringComponent, *commonpb.PaginationResponse) {
	limit, page := int32(50), int32(1)
	if p != nil {
		if p.Limit > 0 {
			limit = p.Limit
		}
		if off := p.GetOffset(); off != nil && off.Page > 0 {
			page = off.Page
		}
	}
	total := int32(len(all))
	start := (page - 1) * limit
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	totalPages := int32(0)
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return page, all[start:end], &commonpb.PaginationResponse{
		TotalItems:  total,
		CurrentPage: &page,
		TotalPages:  &totalPages,
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
	}
}
