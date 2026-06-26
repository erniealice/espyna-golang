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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/scoring_scheme"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ScoringScheme, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres scoring_scheme repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresScoringSchemeRepository(dbOps, tableName), nil
	})
}

// PostgresScoringSchemeRepository implements scoring_scheme CRUD via PostgreSQL.
// PageData reads delegate to the workspace-aware dbOps decorator, which scopes
// every list/read by the request context's workspace_id (the table carries its
// own workspace_id column → the decorator's direct-column path applies).
type PostgresScoringSchemeRepository struct {
	pb.UnimplementedScoringSchemeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresScoringSchemeRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ScoringSchemeDomainServiceServer {
	if tableName == "" {
		tableName = "scoring_scheme"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresScoringSchemeRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresScoringSchemeRepository) CreateScoringScheme(ctx context.Context, req *pb.CreateScoringSchemeRequest) (*pb.CreateScoringSchemeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("scoring scheme data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create scoring scheme: %w", err)
	}
	item, err := scoringSchemeFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateScoringSchemeResponse{Data: []*pb.ScoringScheme{item}, Success: true}, nil
}

func (r *PostgresScoringSchemeRepository) ReadScoringScheme(ctx context.Context, req *pb.ReadScoringSchemeRequest) (*pb.ReadScoringSchemeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("scoring scheme ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read scoring scheme: %w", err)
	}
	item, err := scoringSchemeFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadScoringSchemeResponse{Data: []*pb.ScoringScheme{item}, Success: true}, nil
}

func (r *PostgresScoringSchemeRepository) UpdateScoringScheme(ctx context.Context, req *pb.UpdateScoringSchemeRequest) (*pb.UpdateScoringSchemeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("scoring scheme ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update scoring scheme: %w", err)
	}
	item, err := scoringSchemeFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateScoringSchemeResponse{Data: []*pb.ScoringScheme{item}, Success: true}, nil
}

func (r *PostgresScoringSchemeRepository) DeleteScoringScheme(ctx context.Context, req *pb.DeleteScoringSchemeRequest) (*pb.DeleteScoringSchemeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("scoring scheme ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete scoring scheme: %w", err)
	}
	return &pb.DeleteScoringSchemeResponse{Success: true}, nil
}

func (r *PostgresScoringSchemeRepository) ListScoringSchemes(ctx context.Context, req *pb.ListScoringSchemesRequest) (*pb.ListScoringSchemesResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListScoringSchemesResponse{Data: items, Success: true}, nil
}

func (r *PostgresScoringSchemeRepository) GetScoringSchemeListPageData(ctx context.Context, req *pb.GetScoringSchemeListPageDataRequest) (*pb.GetScoringSchemeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	page, items, pagination := paginateScoringScheme(all, req.GetPagination())
	_ = page
	return &pb.GetScoringSchemeListPageDataResponse{ScoringSchemeList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresScoringSchemeRepository) GetScoringSchemeItemPageData(ctx context.Context, req *pb.GetScoringSchemeItemPageDataRequest) (*pb.GetScoringSchemeItemPageDataResponse, error) {
	if req == nil || req.ScoringSchemeId == "" {
		return nil, fmt.Errorf("scoring scheme ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.ScoringSchemeId)
	if err != nil {
		return nil, fmt.Errorf("failed to read scoring scheme: %w", err)
	}
	item, err := scoringSchemeFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetScoringSchemeItemPageDataResponse{ScoringScheme: item, Success: true}, nil
}

func (r *PostgresScoringSchemeRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.ScoringScheme, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list scoring schemes: %w", err)
	}
	var items []*pb.ScoringScheme
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.ScoringScheme{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func scoringSchemeFromResult(result any) (*pb.ScoringScheme, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.ScoringScheme{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateScoringScheme(all []*pb.ScoringScheme, p *commonpb.PaginationRequest) (int32, []*pb.ScoringScheme, *commonpb.PaginationResponse) {
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
