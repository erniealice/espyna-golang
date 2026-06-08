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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_response"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EvaluationResponse, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres evaluation_response repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEvaluationResponseRepository(dbOps, tableName), nil
	})
}

// PostgresEvaluationResponseRepository implements evaluation_response CRUD.
//
// It is a child of evaluation (CASCADE on evaluation_id); workspace_id is copied
// from the parent at create + validated by the use case. The snapshotted
// criteria_type column is a DB CHECK-pinned lowercase token; this adapter
// translates proto-enum ↔ token in both directions.
type PostgresEvaluationResponseRepository struct {
	pb.UnimplementedEvaluationResponseDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresEvaluationResponseRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.EvaluationResponseDomainServiceServer {
	if tableName == "" {
		tableName = "evaluation_response"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresEvaluationResponseRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func evaluationResponseWriteMap(e *pb.EvaluationResponse) (map[string]any, error) {
	jsonData, err := protojson.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	setOrDeleteToken(data, "criteriaType", criteriaTypeTokenFromEnum(e.CriteriaType))
	return data, nil
}

func evaluationResponseFromResultJSON(resultJSON []byte) *pb.EvaluationResponse {
	var raw map[string]any
	_ = json.Unmarshal(resultJSON, &raw)
	typeTok, _ := raw["criteriaType"].(string)
	delete(raw, "criteriaType")
	cleaned, _ := json.Marshal(raw)

	e := &pb.EvaluationResponse{}
	_ = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(cleaned, e)
	e.CriteriaType = criteriaTypeFromString(typeTok)
	return e
}

func (r *PostgresEvaluationResponseRepository) CreateEvaluationResponse(ctx context.Context, req *pb.CreateEvaluationResponseRequest) (*pb.CreateEvaluationResponseResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("evaluation response data is required")
	}
	data, err := evaluationResponseWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluation response: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.CreateEvaluationResponseResponse{
		Data:    []*pb.EvaluationResponse{evaluationResponseFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationResponseRepository) ReadEvaluationResponse(ctx context.Context, req *pb.ReadEvaluationResponseRequest) (*pb.ReadEvaluationResponseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation response ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read evaluation response: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.ReadEvaluationResponseResponse{
		Data:    []*pb.EvaluationResponse{evaluationResponseFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationResponseRepository) UpdateEvaluationResponse(ctx context.Context, req *pb.UpdateEvaluationResponseRequest) (*pb.UpdateEvaluationResponseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation response ID is required")
	}
	data, err := evaluationResponseWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update evaluation response: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.UpdateEvaluationResponseResponse{
		Data:    []*pb.EvaluationResponse{evaluationResponseFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationResponseRepository) DeleteEvaluationResponse(ctx context.Context, req *pb.DeleteEvaluationResponseRequest) (*pb.DeleteEvaluationResponseResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation response ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete evaluation response: %w", err)
	}
	return &pb.DeleteEvaluationResponseResponse{Success: true}, nil
}

func (r *PostgresEvaluationResponseRepository) ListEvaluationResponses(ctx context.Context, req *pb.ListEvaluationResponsesRequest) (*pb.ListEvaluationResponsesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list evaluation responses: %w", err)
	}
	var items []*pb.EvaluationResponse
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		items = append(items, evaluationResponseFromResultJSON(resultJSON))
	}
	return &pb.ListEvaluationResponsesResponse{Data: items, Success: true}, nil
}

var evaluationResponseSortableSQLCols = []string{"sequence_order", "date_created"}

const evaluationResponseSelectCols = `id, evaluation_id, workspace_id, outcome_criteria_id, criteria_version_id,
	criteria_label, criteria_weight, criteria_type, numeric_value, text_value, categorical_value, pass_fail_value,
	comment, sequence_order, active, date_created`

func (r *PostgresEvaluationResponseRepository) GetEvaluationResponseListPageData(ctx context.Context, req *pb.GetEvaluationResponseListPageDataRequest) (*pb.GetEvaluationResponseListPageDataResponse, error) {
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
	orderBy, err := postgresCore.BuildOrderBy(evaluationResponseSortableSQLCols, req.GetSort(), "sequence_order ASC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for evaluation response list: %w", err)
	}
	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `SELECT ` + evaluationResponseSelectCols + `
		FROM ` + r.tableName + `
		WHERE active = true AND ($3::text = '' OR workspace_id = $3::text) ` + orderBy + ` LIMIT $1 OFFSET $2;`
	rows, err := r.db.QueryContext(ctx, query, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*pb.EvaluationResponse
	for rows.Next() {
		e, scanErr := scanEvaluationResponseRow(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan failed: %w", scanErr)
		}
		items = append(items, e)
	}
	return &pb.GetEvaluationResponseListPageDataResponse{
		EvaluationResponseList: items,
		Pagination:             &commonpb.PaginationResponse{CurrentPage: &page},
		Success:                true,
	}, nil
}

func (r *PostgresEvaluationResponseRepository) GetEvaluationResponseItemPageData(ctx context.Context, req *pb.GetEvaluationResponseItemPageDataRequest) (*pb.GetEvaluationResponseItemPageDataResponse, error) {
	if req == nil || req.EvaluationResponseId == "" {
		return nil, fmt.Errorf("evaluation response ID required")
	}
	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `SELECT ` + evaluationResponseSelectCols + `
		FROM ` + r.tableName + `
		WHERE id = $1 AND active = true AND ($2::text = '' OR workspace_id = $2::text)`
	row := r.db.QueryRowContext(ctx, query, req.EvaluationResponseId, wsID)
	e, err := scanEvaluationResponseRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("evaluation response not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &pb.GetEvaluationResponseItemPageDataResponse{EvaluationResponse: e, Success: true}, nil
}

func scanEvaluationResponseRow(scan func(dest ...any) error) (*pb.EvaluationResponse, error) {
	var id, evaluationID, workspaceID, outcomeCriteriaID, criteriaLabel, typeTok string
	var criteriaVersionID, textValue, categoricalValue, comment sql.NullString
	var criteriaWeight, numericValue sql.NullFloat64
	var passFailValue sql.NullBool
	var sequenceOrder int32
	var active bool
	var dateCreated sql.NullTime
	if err := scan(&id, &evaluationID, &workspaceID, &outcomeCriteriaID, &criteriaVersionID,
		&criteriaLabel, &criteriaWeight, &typeTok, &numericValue, &textValue, &categoricalValue, &passFailValue,
		&comment, &sequenceOrder, &active, &dateCreated); err != nil {
		return nil, err
	}
	e := &pb.EvaluationResponse{
		Id:                id,
		EvaluationId:      evaluationID,
		WorkspaceId:       workspaceID,
		OutcomeCriteriaId: outcomeCriteriaID,
		CriteriaLabel:     criteriaLabel,
		CriteriaType:      criteriaTypeFromString(typeTok),
		SequenceOrder:     sequenceOrder,
		Active:            active,
	}
	setOptStr(&e.CriteriaVersionId, criteriaVersionID)
	setOptStr(&e.TextValue, textValue)
	setOptStr(&e.CategoricalValue, categoricalValue)
	setOptStr(&e.Comment, comment)
	if criteriaWeight.Valid {
		v := criteriaWeight.Float64
		e.CriteriaWeight = &v
	}
	if numericValue.Valid {
		v := numericValue.Float64
		e.NumericValue = &v
	}
	if passFailValue.Valid {
		v := passFailValue.Bool
		e.PassFailValue = &v
	}
	if dateCreated.Valid && !dateCreated.Time.IsZero() {
		ts := dateCreated.Time.UnixMilli()
		e.DateCreated = &ts
		s := dateCreated.Time.Format("2006-01-02T15:04:05Z07:00")
		e.DateCreatedString = &s
	}
	return e, nil
}

func NewEvaluationResponseRepository(db *sql.DB, tableName string) pb.EvaluationResponseDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEvaluationResponseRepository(dbOps, tableName)
}
