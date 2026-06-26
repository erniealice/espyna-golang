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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_component_criteria"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ScoringComponentCriteria, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres scoring_component_criteria repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresScoringComponentCriteriaRepository(dbOps, tableName), nil
	})
}

// PostgresScoringComponentCriteriaRepository implements scoring_component_criteria CRUD via PostgreSQL.
type PostgresScoringComponentCriteriaRepository struct {
	pb.UnimplementedScoringComponentCriteriaDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresScoringComponentCriteriaRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ScoringComponentCriteriaDomainServiceServer {
	if tableName == "" {
		tableName = "scoring_component_criteria"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresScoringComponentCriteriaRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresScoringComponentCriteriaRepository) CreateScoringComponentCriteria(ctx context.Context, req *pb.CreateScoringComponentCriteriaRequest) (*pb.CreateScoringComponentCriteriaResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("scoring component criteria data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create scoring component criteria: %w", err)
	}
	item, err := scoringComponentCriteriaFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateScoringComponentCriteriaResponse{Data: []*pb.ScoringComponentCriteria{item}, Success: true}, nil
}

func (r *PostgresScoringComponentCriteriaRepository) ReadScoringComponentCriteria(ctx context.Context, req *pb.ReadScoringComponentCriteriaRequest) (*pb.ReadScoringComponentCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("scoring component criteria ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read scoring component criteria: %w", err)
	}
	item, err := scoringComponentCriteriaFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadScoringComponentCriteriaResponse{Data: []*pb.ScoringComponentCriteria{item}, Success: true}, nil
}

func (r *PostgresScoringComponentCriteriaRepository) UpdateScoringComponentCriteria(ctx context.Context, req *pb.UpdateScoringComponentCriteriaRequest) (*pb.UpdateScoringComponentCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("scoring component criteria ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update scoring component criteria: %w", err)
	}
	item, err := scoringComponentCriteriaFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateScoringComponentCriteriaResponse{Data: []*pb.ScoringComponentCriteria{item}, Success: true}, nil
}

func (r *PostgresScoringComponentCriteriaRepository) DeleteScoringComponentCriteria(ctx context.Context, req *pb.DeleteScoringComponentCriteriaRequest) (*pb.DeleteScoringComponentCriteriaResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("scoring component criteria ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete scoring component criteria: %w", err)
	}
	return &pb.DeleteScoringComponentCriteriaResponse{Success: true}, nil
}

func (r *PostgresScoringComponentCriteriaRepository) ListScoringComponentCriterias(ctx context.Context, req *pb.ListScoringComponentCriteriasRequest) (*pb.ListScoringComponentCriteriasResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListScoringComponentCriteriasResponse{Data: items, Success: true}, nil
}

func (r *PostgresScoringComponentCriteriaRepository) GetScoringComponentCriteriaListPageData(ctx context.Context, req *pb.GetScoringComponentCriteriaListPageDataRequest) (*pb.GetScoringComponentCriteriaListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	page, items, pagination := paginateScoringComponentCriteria(all, req.GetPagination())
	_ = page
	return &pb.GetScoringComponentCriteriaListPageDataResponse{ScoringComponentCriteriaList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresScoringComponentCriteriaRepository) GetScoringComponentCriteriaItemPageData(ctx context.Context, req *pb.GetScoringComponentCriteriaItemPageDataRequest) (*pb.GetScoringComponentCriteriaItemPageDataResponse, error) {
	if req == nil || req.ScoringComponentCriteriaId == "" {
		return nil, fmt.Errorf("scoring component criteria ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.ScoringComponentCriteriaId)
	if err != nil {
		return nil, fmt.Errorf("failed to read scoring component criteria: %w", err)
	}
	item, err := scoringComponentCriteriaFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetScoringComponentCriteriaItemPageDataResponse{ScoringComponentCriteria: item, Success: true}, nil
}

func (r *PostgresScoringComponentCriteriaRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.ScoringComponentCriteria, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list scoring component criterias: %w", err)
	}
	var items []*pb.ScoringComponentCriteria
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.ScoringComponentCriteria{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func scoringComponentCriteriaFromResult(result any) (*pb.ScoringComponentCriteria, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.ScoringComponentCriteria{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateScoringComponentCriteria(all []*pb.ScoringComponentCriteria, p *commonpb.PaginationRequest) (int32, []*pb.ScoringComponentCriteria, *commonpb.PaginationResponse) {
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
