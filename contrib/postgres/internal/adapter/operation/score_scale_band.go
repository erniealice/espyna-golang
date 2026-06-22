//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale_band"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ScoreScaleBand, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres score_scale_band repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresScoreScaleBandRepository(dbOps, tableName), nil
	})
}

// PostgresScoreScaleBandRepository implements score_scale_band CRUD via PostgreSQL.
type PostgresScoreScaleBandRepository struct {
	pb.UnimplementedScoreScaleBandDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresScoreScaleBandRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ScoreScaleBandDomainServiceServer {
	if tableName == "" {
		tableName = "score_scale_band"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresScoreScaleBandRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresScoreScaleBandRepository) CreateScoreScaleBand(ctx context.Context, req *pb.CreateScoreScaleBandRequest) (*pb.CreateScoreScaleBandResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("score scale band data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create score scale band: %w", err)
	}
	item, err := scoreScaleBandFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateScoreScaleBandResponse{Data: []*pb.ScoreScaleBand{item}, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) ReadScoreScaleBand(ctx context.Context, req *pb.ReadScoreScaleBandRequest) (*pb.ReadScoreScaleBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale band ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read score scale band: %w", err)
	}
	item, err := scoreScaleBandFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadScoreScaleBandResponse{Data: []*pb.ScoreScaleBand{item}, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) UpdateScoreScaleBand(ctx context.Context, req *pb.UpdateScoreScaleBandRequest) (*pb.UpdateScoreScaleBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale band ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update score scale band: %w", err)
	}
	item, err := scoreScaleBandFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateScoreScaleBandResponse{Data: []*pb.ScoreScaleBand{item}, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) DeleteScoreScaleBand(ctx context.Context, req *pb.DeleteScoreScaleBandRequest) (*pb.DeleteScoreScaleBandResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale band ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete score scale band: %w", err)
	}
	return &pb.DeleteScoreScaleBandResponse{Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) ListScoreScaleBands(ctx context.Context, req *pb.ListScoreScaleBandsRequest) (*pb.ListScoreScaleBandsResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListScoreScaleBandsResponse{Data: items, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) GetScoreScaleBandListPageData(ctx context.Context, req *pb.GetScoreScaleBandListPageDataRequest) (*pb.GetScoreScaleBandListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	page, items, pagination := paginateScoreScaleBand(all, req.GetPagination())
	_ = page
	return &pb.GetScoreScaleBandListPageDataResponse{ScoreScaleBandList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) GetScoreScaleBandItemPageData(ctx context.Context, req *pb.GetScoreScaleBandItemPageDataRequest) (*pb.GetScoreScaleBandItemPageDataResponse, error) {
	if req == nil || req.ScoreScaleBandId == "" {
		return nil, fmt.Errorf("score scale band ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.ScoreScaleBandId)
	if err != nil {
		return nil, fmt.Errorf("failed to read score scale band: %w", err)
	}
	item, err := scoreScaleBandFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetScoreScaleBandItemPageDataResponse{ScoreScaleBand: item, Success: true}, nil
}

func (r *PostgresScoreScaleBandRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.ScoreScaleBand, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list score scale bands: %w", err)
	}
	var items []*pb.ScoreScaleBand
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.ScoreScaleBand{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func scoreScaleBandFromResult(result any) (*pb.ScoreScaleBand, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.ScoreScaleBand{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateScoreScaleBand(all []*pb.ScoreScaleBand, p *commonpb.PaginationRequest) (int32, []*pb.ScoreScaleBand, *commonpb.PaginationResponse) {
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
