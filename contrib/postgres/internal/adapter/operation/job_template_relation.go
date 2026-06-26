//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_template_relation"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.JobTemplateRelation, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres job_template_relation repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresJobTemplateRelationRepository(dbOps, tableName), nil
	})
}

// PostgresJobTemplateRelationRepository implements job_template_relation
// CRUD + the ListByParent / ListByChild filters used by
// MaterializeJobsForSubscription per
// docs/plan/20260429-auto-spawn-jobs-from-subscription/plan.md §2.2 + §3.2.
type PostgresJobTemplateRelationRepository struct {
	pb.UnimplementedJobTemplateRelationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresJobTemplateRelationRepository builds the adapter.
func NewPostgresJobTemplateRelationRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.JobTemplateRelationDomainServiceServer {
	if tableName == "" {
		tableName = "job_template_relation"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresJobTemplateRelationRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateJobTemplateRelation inserts a new join-table row.
func (r *PostgresJobTemplateRelationRepository) CreateJobTemplateRelation(
	ctx context.Context, req *pb.CreateJobTemplateRelationRequest,
) (*pb.CreateJobTemplateRelationResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("job_template_relation data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal job_template_relation: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal job_template_relation: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("create job_template_relation: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	rel := &pb.JobTemplateRelation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rel); err != nil {
		return nil, fmt.Errorf("unmarshal job_template_relation: %w", err)
	}
	return &pb.CreateJobTemplateRelationResponse{Success: true, Data: []*pb.JobTemplateRelation{rel}}, nil
}

// ReadJobTemplateRelation fetches one row by ID.
func (r *PostgresJobTemplateRelationRepository) ReadJobTemplateRelation(
	ctx context.Context, req *pb.ReadJobTemplateRelationRequest,
) (*pb.ReadJobTemplateRelationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_relation ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("read job_template_relation: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	rel := &pb.JobTemplateRelation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rel); err != nil {
		return nil, fmt.Errorf("unmarshal job_template_relation: %w", err)
	}
	return &pb.ReadJobTemplateRelationResponse{Success: true, Data: []*pb.JobTemplateRelation{rel}}, nil
}

// UpdateJobTemplateRelation updates a row.
func (r *PostgresJobTemplateRelationRepository) UpdateJobTemplateRelation(
	ctx context.Context, req *pb.UpdateJobTemplateRelationRequest,
) (*pb.UpdateJobTemplateRelationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_relation ID is required")
	}
	jsonData, _ := protojson.Marshal(req.Data)
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal job_template_relation: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("update job_template_relation: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	rel := &pb.JobTemplateRelation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rel); err != nil {
		return nil, fmt.Errorf("unmarshal job_template_relation: %w", err)
	}
	return &pb.UpdateJobTemplateRelationResponse{Success: true, Data: []*pb.JobTemplateRelation{rel}}, nil
}

// DeleteJobTemplateRelation soft-deletes a row.
func (r *PostgresJobTemplateRelationRepository) DeleteJobTemplateRelation(
	ctx context.Context, req *pb.DeleteJobTemplateRelationRequest,
) (*pb.DeleteJobTemplateRelationResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job_template_relation ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("delete job_template_relation: %w", err)
	}
	return &pb.DeleteJobTemplateRelationResponse{Success: true}, nil
}

// ListJobTemplateRelations lists rows.
func (r *PostgresJobTemplateRelationRepository) ListJobTemplateRelations(
	ctx context.Context, req *pb.ListJobTemplateRelationsRequest,
) (*pb.ListJobTemplateRelationsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("list job_template_relations: %w", err)
	}
	var rels []*pb.JobTemplateRelation
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: marshal job_template_relation row: %v", err)
			continue
		}
		rel := &pb.JobTemplateRelation{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rel); err != nil {
			log.Printf("WARN: unmarshal job_template_relation: %v", err)
			continue
		}
		rels = append(rels, rel)
	}
	return &pb.ListJobTemplateRelationsResponse{Success: true, Data: rels}, nil
}

// GetJobTemplateRelationListPageData returns a paginated listing for the
// list page. Minimal implementation — list-page wiring lands with Phase D
// UI work.
func (r *PostgresJobTemplateRelationRepository) GetJobTemplateRelationListPageData(
	ctx context.Context, req *pb.GetJobTemplateRelationListPageDataRequest,
) (*pb.GetJobTemplateRelationListPageDataResponse, error) {
	listResp, err := r.ListJobTemplateRelations(ctx, &pb.ListJobTemplateRelationsRequest{})
	if err != nil {
		return nil, err
	}
	return &pb.GetJobTemplateRelationListPageDataResponse{
		JobTemplateRelationList: listResp.GetData(),
		Success:                 true,
	}, nil
}

// GetJobTemplateRelationItemPageData reads one row for the item page.
func (r *PostgresJobTemplateRelationRepository) GetJobTemplateRelationItemPageData(
	ctx context.Context, req *pb.GetJobTemplateRelationItemPageDataRequest,
) (*pb.GetJobTemplateRelationItemPageDataResponse, error) {
	if req == nil || req.JobTemplateRelationId == "" {
		return nil, fmt.Errorf("job_template_relation ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.JobTemplateRelationId)
	if err != nil {
		return nil, fmt.Errorf("read job_template_relation: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	rel := &pb.JobTemplateRelation{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rel); err != nil {
		return nil, fmt.Errorf("unmarshal job_template_relation: %w", err)
	}
	return &pb.GetJobTemplateRelationItemPageDataResponse{JobTemplateRelation: rel, Success: true}, nil
}

// ListByParent returns relations whose parent_template_id matches, ordered
// by sequence_order ASC. Hot path for MaterializeJobsForSubscription per
// plan §3.2.
func (r *PostgresJobTemplateRelationRepository) ListByParent(
	ctx context.Context, req *pb.ListJobTemplateRelationsByParentRequest,
) (*pb.ListJobTemplateRelationsByParentResponse, error) {
	if req == nil || req.ParentTemplateId == "" {
		return nil, fmt.Errorf("parent_template_id is required")
	}
	rels, err := r.listByColumn(ctx, "parent_template_id", req.ParentTemplateId)
	if err != nil {
		return nil, err
	}
	return &pb.ListJobTemplateRelationsByParentResponse{
		JobTemplateRelations: rels,
		Success:              true,
	}, nil
}

// ListByChild returns relations whose child_template_id matches.
func (r *PostgresJobTemplateRelationRepository) ListByChild(
	ctx context.Context, req *pb.ListJobTemplateRelationsByChildRequest,
) (*pb.ListJobTemplateRelationsByChildResponse, error) {
	if req == nil || req.ChildTemplateId == "" {
		return nil, fmt.Errorf("child_template_id is required")
	}
	rels, err := r.listByColumn(ctx, "child_template_id", req.ChildTemplateId)
	if err != nil {
		return nil, err
	}
	return &pb.ListJobTemplateRelationsByChildResponse{
		JobTemplateRelations: rels,
		Success:              true,
	}, nil
}

// listByColumn is the shared filter helper. Allowlists the column name to
// avoid SQL injection.
func (r *PostgresJobTemplateRelationRepository) listByColumn(
	ctx context.Context, column, value string,
) ([]*pb.JobTemplateRelation, error) {
	if r.db == nil {
		return nil, fmt.Errorf("job_template_relation repository missing *sql.DB")
	}
	switch column {
	case "parent_template_id", "child_template_id":
		// ok
	default:
		return nil, fmt.Errorf("unsupported list column: %s", column)
	}

	query := `SELECT * FROM ` + r.tableName +
		` WHERE ` + column + ` = $1 AND active = true` +
		` ORDER BY sequence_order ASC, date_created ASC`
	rows, err := r.db.QueryContext(ctx, query, value)
	if err != nil {
		return nil, fmt.Errorf("job_template_relation query (%s=%s): %w", column, value, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("rows.Columns: %w", err)
	}

	var out []*pb.JobTemplateRelation
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan job_template_relation row: %w", err)
		}
		raw := map[string]any{}
		for i, c := range cols {
			raw[c] = normalizeJTRScanValue(vals[i])
		}
		dataJSON, err := json.Marshal(postgresCore.DenormalizeKeys(raw))
		if err != nil {
			log.Printf("WARN: marshal job_template_relation row: %v", err)
			continue
		}
		rel := &pb.JobTemplateRelation{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(dataJSON, rel); err != nil {
			log.Printf("WARN: unmarshal job_template_relation row: %v", err)
			continue
		}
		out = append(out, rel)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("job_template_relation row iter: %w", err)
	}
	return out, nil
}

// normalizeJTRScanValue mirrors the small helper in billing_event.go for
// converting database/sql.Scan return types into proto-friendly values.
func normalizeJTRScanValue(v any) any {
	switch t := v.(type) {
	case []byte:
		return string(t)
	default:
		return t
	}
}
