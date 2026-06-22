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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/score_scale"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.ScoreScale, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres score_scale repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresScoreScaleRepository(dbOps, tableName), nil
	})
}

// PostgresScoreScaleRepository implements score_scale CRUD via PostgreSQL.
// ListByGroup and GetCurrentPublished are extra RPCs covered by the Unimplemented embedding.
type PostgresScoreScaleRepository struct {
	pb.UnimplementedScoreScaleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresScoreScaleRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.ScoreScaleDomainServiceServer {
	if tableName == "" {
		tableName = "score_scale"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresScoreScaleRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresScoreScaleRepository) CreateScoreScale(ctx context.Context, req *pb.CreateScoreScaleRequest) (*pb.CreateScoreScaleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("score scale data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create score scale: %w", err)
	}
	item, err := scoreScaleFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateScoreScaleResponse{Data: []*pb.ScoreScale{item}, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) ReadScoreScale(ctx context.Context, req *pb.ReadScoreScaleRequest) (*pb.ReadScoreScaleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read score scale: %w", err)
	}
	item, err := scoreScaleFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadScoreScaleResponse{Data: []*pb.ScoreScale{item}, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) UpdateScoreScale(ctx context.Context, req *pb.UpdateScoreScaleRequest) (*pb.UpdateScoreScaleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update score scale: %w", err)
	}
	item, err := scoreScaleFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateScoreScaleResponse{Data: []*pb.ScoreScale{item}, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) DeleteScoreScale(ctx context.Context, req *pb.DeleteScoreScaleRequest) (*pb.DeleteScoreScaleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("score scale ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete score scale: %w", err)
	}
	return &pb.DeleteScoreScaleResponse{Success: true}, nil
}

func (r *PostgresScoreScaleRepository) ListScoreScales(ctx context.Context, req *pb.ListScoreScalesRequest) (*pb.ListScoreScalesResponse, error) {
	items, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	return &pb.ListScoreScalesResponse{Data: items, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) GetScoreScaleListPageData(ctx context.Context, req *pb.GetScoreScaleListPageDataRequest) (*pb.GetScoreScaleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	all, err := r.listAll(ctx, req.GetFilters())
	if err != nil {
		return nil, err
	}
	page, items, pagination := paginateScoreScale(all, req.GetPagination())
	_ = page
	return &pb.GetScoreScaleListPageDataResponse{ScoreScaleList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) GetScoreScaleItemPageData(ctx context.Context, req *pb.GetScoreScaleItemPageDataRequest) (*pb.GetScoreScaleItemPageDataResponse, error) {
	if req == nil || req.ScoreScaleId == "" {
		return nil, fmt.Errorf("score scale ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.ScoreScaleId)
	if err != nil {
		return nil, fmt.Errorf("failed to read score scale: %w", err)
	}
	item, err := scoreScaleFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetScoreScaleItemPageDataResponse{ScoreScale: item, Success: true}, nil
}

func (r *PostgresScoreScaleRepository) listAll(ctx context.Context, filters *commonpb.FilterRequest) ([]*pb.ScoreScale, error) {
	var params *interfaces.ListParams
	if filters != nil {
		params = &interfaces.ListParams{Filters: filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list score scales: %w", err)
	}
	var items []*pb.ScoreScale
	for _, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			continue
		}
		item := &pb.ScoreScale{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func scoreScaleFromResult(result any) (*pb.ScoreScale, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.ScoreScale{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}

func paginateScoreScale(all []*pb.ScoreScale, p *commonpb.PaginationRequest) (int32, []*pb.ScoreScale, *commonpb.PaginationResponse) {
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
