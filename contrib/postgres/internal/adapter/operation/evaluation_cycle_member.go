//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/consumer"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_cycle_member"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EvaluationCycleMember, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres evaluation_cycle_member repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEvaluationCycleMemberRepository(dbOps, tableName), nil
	})
}

// PostgresEvaluationCycleMemberRepository implements evaluation_cycle_member CRUD
// (SR-1 frozen-denominator snapshot). It is a child of evaluation_cycle;
// workspace_id + client_id are denormalized scope anchors copied at create. No
// enum token columns. The domain-layer evaluation_cycle.OpenUseCase performs the
// idempotent INSERT … ON CONFLICT DO NOTHING via this adapter's Create.
type PostgresEvaluationCycleMemberRepository struct {
	pb.UnimplementedEvaluationCycleMemberDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresEvaluationCycleMemberRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.EvaluationCycleMemberDomainServiceServer {
	if tableName == "" {
		tableName = "evaluation_cycle_member"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresEvaluationCycleMemberRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func evaluationCycleMemberWriteMap(e *pb.EvaluationCycleMember) (map[string]any, error) {
	jsonData, err := protojson.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	return data, nil
}

func evaluationCycleMemberFromResultJSON(resultJSON []byte) *pb.EvaluationCycleMember {
	e := &pb.EvaluationCycleMember{}
	_ = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, e)
	return e
}

func (r *PostgresEvaluationCycleMemberRepository) CreateEvaluationCycleMember(ctx context.Context, req *pb.CreateEvaluationCycleMemberRequest) (*pb.CreateEvaluationCycleMemberResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("evaluation cycle member data is required")
	}
	data, err := evaluationCycleMemberWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluation cycle member: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.CreateEvaluationCycleMemberResponse{
		Data:    []*pb.EvaluationCycleMember{evaluationCycleMemberFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationCycleMemberRepository) ReadEvaluationCycleMember(ctx context.Context, req *pb.ReadEvaluationCycleMemberRequest) (*pb.ReadEvaluationCycleMemberResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation cycle member ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read evaluation cycle member: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.ReadEvaluationCycleMemberResponse{
		Data:    []*pb.EvaluationCycleMember{evaluationCycleMemberFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationCycleMemberRepository) UpdateEvaluationCycleMember(ctx context.Context, req *pb.UpdateEvaluationCycleMemberRequest) (*pb.UpdateEvaluationCycleMemberResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation cycle member ID is required")
	}
	data, err := evaluationCycleMemberWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update evaluation cycle member: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.UpdateEvaluationCycleMemberResponse{
		Data:    []*pb.EvaluationCycleMember{evaluationCycleMemberFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationCycleMemberRepository) DeleteEvaluationCycleMember(ctx context.Context, req *pb.DeleteEvaluationCycleMemberRequest) (*pb.DeleteEvaluationCycleMemberResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation cycle member ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete evaluation cycle member: %w", err)
	}
	return &pb.DeleteEvaluationCycleMemberResponse{Success: true}, nil
}

func (r *PostgresEvaluationCycleMemberRepository) ListEvaluationCycleMembers(ctx context.Context, req *pb.ListEvaluationCycleMembersRequest) (*pb.ListEvaluationCycleMembersResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list evaluation cycle members: %w", err)
	}
	var items []*pb.EvaluationCycleMember
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		items = append(items, evaluationCycleMemberFromResultJSON(resultJSON))
	}
	return &pb.ListEvaluationCycleMembersResponse{Data: items, Success: true}, nil
}

var evaluationCycleMemberSortableSQLCols = []string{"subject_staff_id", "date_added"}

const evaluationCycleMemberSelectCols = `id, workspace_id, evaluation_cycle_id, client_id, subject_staff_id,
	is_probation, active, date_added`

func (r *PostgresEvaluationCycleMemberRepository) GetEvaluationCycleMemberListPageData(ctx context.Context, req *pb.GetEvaluationCycleMemberListPageDataRequest) (*pb.GetEvaluationCycleMemberListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	limit, offset, page := int32(200), int32(0), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}
	orderBy, err := postgresCore.BuildOrderBy(evaluationCycleMemberSortableSQLCols, req.GetSort(), "date_added ASC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for evaluation cycle member list: %w", err)
	}
	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `SELECT ` + evaluationCycleMemberSelectCols + `
		FROM ` + r.tableName + `
		WHERE active = true AND ($3::text = '' OR workspace_id = $3::text) ` + orderBy + ` LIMIT $1 OFFSET $2;`
	rows, err := r.db.QueryContext(ctx, query, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*pb.EvaluationCycleMember
	for rows.Next() {
		e, scanErr := scanEvaluationCycleMemberRow(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan failed: %w", scanErr)
		}
		items = append(items, e)
	}
	return &pb.GetEvaluationCycleMemberListPageDataResponse{
		EvaluationCycleMemberList: items,
		Pagination:                &commonpb.PaginationResponse{CurrentPage: &page},
		Success:                   true,
	}, nil
}

func (r *PostgresEvaluationCycleMemberRepository) GetEvaluationCycleMemberItemPageData(ctx context.Context, req *pb.GetEvaluationCycleMemberItemPageDataRequest) (*pb.GetEvaluationCycleMemberItemPageDataResponse, error) {
	if req == nil || req.EvaluationCycleMemberId == "" {
		return nil, fmt.Errorf("evaluation cycle member ID required")
	}
	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `SELECT ` + evaluationCycleMemberSelectCols + `
		FROM ` + r.tableName + `
		WHERE id = $1 AND active = true AND ($2::text = '' OR workspace_id = $2::text)`
	row := r.db.QueryRowContext(ctx, query, req.EvaluationCycleMemberId, wsID)
	e, err := scanEvaluationCycleMemberRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("evaluation cycle member not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &pb.GetEvaluationCycleMemberItemPageDataResponse{EvaluationCycleMember: e, Success: true}, nil
}

func scanEvaluationCycleMemberRow(scan func(dest ...any) error) (*pb.EvaluationCycleMember, error) {
	var id, workspaceID, evaluationCycleID, clientID, subjectStaffID string
	var isProbation, active bool
	var dateAdded sql.NullInt64
	if err := scan(&id, &workspaceID, &evaluationCycleID, &clientID, &subjectStaffID,
		&isProbation, &active, &dateAdded); err != nil {
		return nil, err
	}
	e := &pb.EvaluationCycleMember{
		Id:                id,
		WorkspaceId:       workspaceID,
		EvaluationCycleId: evaluationCycleID,
		ClientId:          clientID,
		SubjectStaffId:    subjectStaffID,
		IsProbation:       isProbation,
		Active:            active,
	}
	if dateAdded.Valid {
		v := dateAdded.Int64
		e.DateAdded = &v
	}
	return e, nil
}

func NewEvaluationCycleMemberRepository(db *sql.DB, tableName string) pb.EvaluationCycleMemberDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEvaluationCycleMemberRepository(dbOps, tableName)
}
