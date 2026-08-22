//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"log"
	"sort"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/database/interfaces"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/plan_job_template"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PlanJobTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres plan_job_template repository requires *sql.DB, got %T", conn)
		}
		return NewPostgresPlanJobTemplateRepository(postgresCore.NewWorkspaceAwareOperations(db), tableName), nil
	})
}

type PostgresPlanJobTemplateRepository struct {
	pb.UnimplementedPlanJobTemplateDomainServiceServer
	dbOps interfaces.DatabaseOperation
	table string
}

func NewPostgresPlanJobTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.PlanJobTemplateDomainServiceServer {
	if tableName == "" {
		tableName = entityid.PlanJobTemplate
	}
	return &PostgresPlanJobTemplateRepository{dbOps: dbOps, table: tableName}
}

func (r *PostgresPlanJobTemplateRepository) CreatePlanJobTemplate(ctx context.Context, req *pb.CreatePlanJobTemplateRequest) (*pb.CreatePlanJobTemplateResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("plan_job_template data is required")
	}
	data, err := marshalPlanJobTemplate(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.table, data)
	if err != nil {
		return nil, fmt.Errorf("create plan_job_template: %w", err)
	}
	row, err := decodePostgresPlanJobTemplate(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreatePlanJobTemplateResponse{Success: true, Data: []*pb.PlanJobTemplate{row}}, nil
}

func (r *PostgresPlanJobTemplateRepository) ReadPlanJobTemplate(ctx context.Context, req *pb.ReadPlanJobTemplateRequest) (*pb.ReadPlanJobTemplateResponse, error) {
	if req == nil || req.Data == nil || req.Data.GetId() == "" {
		return nil, fmt.Errorf("plan_job_template ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.table, req.Data.GetId())
	if err != nil {
		return nil, fmt.Errorf("read plan_job_template: %w", err)
	}
	row, err := decodePostgresPlanJobTemplate(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadPlanJobTemplateResponse{Success: true, Data: []*pb.PlanJobTemplate{row}}, nil
}

func (r *PostgresPlanJobTemplateRepository) UpdatePlanJobTemplate(ctx context.Context, req *pb.UpdatePlanJobTemplateRequest) (*pb.UpdatePlanJobTemplateResponse, error) {
	if req == nil || req.Data == nil || req.Data.GetId() == "" {
		return nil, fmt.Errorf("plan_job_template ID is required")
	}
	data, err := marshalPlanJobTemplate(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.table, req.Data.GetId(), data)
	if err != nil {
		return nil, fmt.Errorf("update plan_job_template: %w", err)
	}
	row, err := decodePostgresPlanJobTemplate(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdatePlanJobTemplateResponse{Success: true, Data: []*pb.PlanJobTemplate{row}}, nil
}

func (r *PostgresPlanJobTemplateRepository) DeletePlanJobTemplate(ctx context.Context, req *pb.DeletePlanJobTemplateRequest) (*pb.DeletePlanJobTemplateResponse, error) {
	if req == nil || req.Data == nil || req.Data.GetId() == "" {
		return nil, fmt.Errorf("plan_job_template ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.table, req.Data.GetId()); err != nil {
		return nil, fmt.Errorf("delete plan_job_template: %w", err)
	}
	return &pb.DeletePlanJobTemplateResponse{Success: true}, nil
}

func (r *PostgresPlanJobTemplateRepository) ListPlanJobTemplates(ctx context.Context, req *pb.ListPlanJobTemplatesRequest) (*pb.ListPlanJobTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	result, err := r.dbOps.List(ctx, r.table, params)
	if err != nil {
		return nil, fmt.Errorf("list plan_job_templates: %w", err)
	}
	rows := make([]*pb.PlanJobTemplate, 0, len(result.Data))
	for _, raw := range result.Data {
		row, err := decodePostgresPlanJobTemplate(raw)
		if err != nil {
			log.Printf("WARN: decode plan_job_template row: %v", err)
			continue
		}
		rows = append(rows, row)
	}
	return &pb.ListPlanJobTemplatesResponse{Success: true, Data: rows}, nil
}

func (r *PostgresPlanJobTemplateRepository) ListPlanJobTemplatesByPlan(ctx context.Context, req *pb.ListPlanJobTemplatesByPlanRequest) (*pb.ListPlanJobTemplatesByPlanResponse, error) {
	if req == nil || req.GetPlanId() == "" {
		return nil, fmt.Errorf("plan_id is required")
	}
	all, err := r.ListPlanJobTemplates(ctx, &pb.ListPlanJobTemplatesRequest{})
	if err != nil {
		return nil, err
	}
	rows := filterAndSortPlanJobTemplates(all.GetData(), req.GetPlanId())
	return &pb.ListPlanJobTemplatesByPlanResponse{Success: true, PlanJobTemplates: rows}, nil
}

func marshalPlanJobTemplate(row *pb.PlanJobTemplate) (map[string]any, error) {
	b, err := (protojson.MarshalOptions{UseEnumNumbers: true}).Marshal(row)
	if err != nil {
		return nil, fmt.Errorf("marshal plan_job_template: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("decode plan_job_template payload: %w", err)
	}
	return data, nil
}

func decodePostgresPlanJobTemplate(raw map[string]any) (*pb.PlanJobTemplate, error) {
	b, err := json.Marshal(postgresCore.DenormalizeKeys(raw))
	if err != nil {
		return nil, fmt.Errorf("marshal plan_job_template row: %w", err)
	}
	row := &pb.PlanJobTemplate{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, row); err != nil {
		return nil, fmt.Errorf("unmarshal plan_job_template row: %w", err)
	}
	return row, nil
}

func filterAndSortPlanJobTemplates(rows []*pb.PlanJobTemplate, planID string) []*pb.PlanJobTemplate {
	out := make([]*pb.PlanJobTemplate, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.GetActive() && row.GetPlanId() == planID {
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].GetSequenceOrder() != out[j].GetSequenceOrder() {
			return out[i].GetSequenceOrder() < out[j].GetSequenceOrder()
		}
		return out[i].GetId() < out[j].GetId()
	})
	return out
}
