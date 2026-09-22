//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/template_task_criteria_rating_description"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TemplateTaskCriteriaRatingDescription, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres template_task_criteria_rating_description repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresTemplateTaskCriteriaRatingDescriptionRepository(dbOps, tableName), nil
	})
}

// PostgresTemplateTaskCriteriaRatingDescriptionRepository persists the
// binding-specific wording. Generic CRUD remains workspace-aware; the matrix
// projection uses an explicit scale/band join so it cannot inherit wording
// from another binding or workspace.
type PostgresTemplateTaskCriteriaRatingDescriptionRepository struct {
	pb.UnimplementedTemplateTaskCriteriaRatingDescriptionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewPostgresTemplateTaskCriteriaRatingDescriptionRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.TemplateTaskCriteriaRatingDescriptionDomainServiceServer {
	if tableName == "" {
		tableName = entityid.TemplateTaskCriteriaRatingDescription
	}
	return &PostgresTemplateTaskCriteriaRatingDescriptionRepository{dbOps: dbOps, tableName: tableName}
}

func (r *PostgresTemplateTaskCriteriaRatingDescriptionRepository) CreateTemplateTaskCriteriaRatingDescription(ctx context.Context, req *pb.CreateTemplateTaskCriteriaRatingDescriptionRequest) (*pb.CreateTemplateTaskCriteriaRatingDescriptionResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("template task criteria rating description data is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create template task criteria rating description: %w", err)
	}
	item, err := templateTaskCriteriaRatingDescriptionFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateTemplateTaskCriteriaRatingDescriptionResponse{Data: []*pb.TemplateTaskCriteriaRatingDescription{item}, Success: true}, nil
}

func (r *PostgresTemplateTaskCriteriaRatingDescriptionRepository) ReadTemplateTaskCriteriaRatingDescription(ctx context.Context, req *pb.ReadTemplateTaskCriteriaRatingDescriptionRequest) (*pb.ReadTemplateTaskCriteriaRatingDescriptionResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("template task criteria rating description ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read template task criteria rating description: %w", err)
	}
	item, err := templateTaskCriteriaRatingDescriptionFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadTemplateTaskCriteriaRatingDescriptionResponse{Data: []*pb.TemplateTaskCriteriaRatingDescription{item}, Success: true}, nil
}

func (r *PostgresTemplateTaskCriteriaRatingDescriptionRepository) UpdateTemplateTaskCriteriaRatingDescription(ctx context.Context, req *pb.UpdateTemplateTaskCriteriaRatingDescriptionRequest) (*pb.UpdateTemplateTaskCriteriaRatingDescriptionResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("template task criteria rating description ID is required")
	}
	data, err := protoToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update template task criteria rating description: %w", err)
	}
	item, err := templateTaskCriteriaRatingDescriptionFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateTemplateTaskCriteriaRatingDescriptionResponse{Data: []*pb.TemplateTaskCriteriaRatingDescription{item}, Success: true}, nil
}

func (r *PostgresTemplateTaskCriteriaRatingDescriptionRepository) DeleteTemplateTaskCriteriaRatingDescription(ctx context.Context, req *pb.DeleteTemplateTaskCriteriaRatingDescriptionRequest) (*pb.DeleteTemplateTaskCriteriaRatingDescriptionResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("template task criteria rating description ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete template task criteria rating description: %w", err)
	}
	return &pb.DeleteTemplateTaskCriteriaRatingDescriptionResponse{Success: true}, nil
}

func (r *PostgresTemplateTaskCriteriaRatingDescriptionRepository) ListTemplateTaskCriteriaRatingDescriptions(ctx context.Context, req *pb.ListTemplateTaskCriteriaRatingDescriptionsRequest) (*pb.ListTemplateTaskCriteriaRatingDescriptionsResponse, error) {
	items, err := r.list(ctx, req.GetFilters(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	return &pb.ListTemplateTaskCriteriaRatingDescriptionsResponse{Data: items, Success: true}, nil
}

func (r *PostgresTemplateTaskCriteriaRatingDescriptionRepository) GetTemplateTaskCriteriaRatingDescriptionListPageData(ctx context.Context, req *pb.GetTemplateTaskCriteriaRatingDescriptionListPageDataRequest) (*pb.GetTemplateTaskCriteriaRatingDescriptionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	items, err := r.list(ctx, req.GetFilters(), req.GetPagination())
	if err != nil {
		return nil, err
	}
	return &pb.GetTemplateTaskCriteriaRatingDescriptionListPageDataResponse{
		TemplateTaskCriteriaRatingDescriptionList: items,
		Success: true,
	}, nil
}

func (r *PostgresTemplateTaskCriteriaRatingDescriptionRepository) GetTemplateTaskCriteriaRatingDescriptionItemPageData(ctx context.Context, req *pb.GetTemplateTaskCriteriaRatingDescriptionItemPageDataRequest) (*pb.GetTemplateTaskCriteriaRatingDescriptionItemPageDataResponse, error) {
	if req == nil || req.TemplateTaskCriteriaRatingDescriptionId == "" {
		return nil, fmt.Errorf("template task criteria rating description ID is required")
	}
	resp, err := r.ReadTemplateTaskCriteriaRatingDescription(ctx, &pb.ReadTemplateTaskCriteriaRatingDescriptionRequest{
		Data: &pb.TemplateTaskCriteriaRatingDescription{Id: req.TemplateTaskCriteriaRatingDescriptionId},
	})
	if err != nil {
		return nil, err
	}
	var item *pb.TemplateTaskCriteriaRatingDescription
	if len(resp.GetData()) > 0 {
		item = resp.GetData()[0]
	}
	return &pb.GetTemplateTaskCriteriaRatingDescriptionItemPageDataResponse{TemplateTaskCriteriaRatingDescription: item, Success: true}, nil
}

func (r *PostgresTemplateTaskCriteriaRatingDescriptionRepository) ListByTemplateTaskCriteria(ctx context.Context, req *pb.ListTemplateTaskCriteriaRatingDescriptionsByTemplateTaskCriteriaRequest) (*pb.ListTemplateTaskCriteriaRatingDescriptionsByTemplateTaskCriteriaResponse, error) {
	if req == nil || req.TemplateTaskCriteriaId == "" {
		return nil, fmt.Errorf("template task criteria ID is required")
	}
	items, err := r.list(ctx, &commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{{
		Field: "template_task_criteria_id",
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value:    req.TemplateTaskCriteriaId,
			Operator: commonpb.StringOperator_STRING_EQUALS,
		}},
	}}}, nil)
	if err != nil {
		return nil, err
	}
	return &pb.ListTemplateTaskCriteriaRatingDescriptionsByTemplateTaskCriteriaResponse{
		TemplateTaskCriteriaRatingDescriptions: items,
		Success:                                true,
	}, nil
}

func (r *PostgresTemplateTaskCriteriaRatingDescriptionRepository) list(ctx context.Context, filters *commonpb.FilterRequest, pagination *commonpb.PaginationRequest) ([]*pb.TemplateTaskCriteriaRatingDescription, error) {
	var params *interfaces.ListParams
	if filters != nil || pagination != nil {
		params = &interfaces.ListParams{Filters: filters, Pagination: pagination}
	}
	result, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list template task criteria rating descriptions: %w", err)
	}
	items := make([]*pb.TemplateTaskCriteriaRatingDescription, 0, len(result.Data))
	for _, row := range result.Data {
		item, err := templateTaskCriteriaRatingDescriptionFromResult(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func templateTaskCriteriaRatingDescriptionFromResult(result any) (*pb.TemplateTaskCriteriaRatingDescription, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template task criteria rating description result: %w", err)
	}
	item := &pb.TemplateTaskCriteriaRatingDescription{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template task criteria rating description result: %w", err)
	}
	return item, nil
}
