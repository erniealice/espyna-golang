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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template_item"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EvaluationTemplateItem, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres evaluation_template_item repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEvaluationTemplateItemRepository(dbOps, tableName), nil
	})
}

// PostgresEvaluationTemplateItemRepository implements evaluation_template_item
// CRUD. It is a child of evaluation_template; workspace_id is copied from the
// parent at create + validated by the use case. No enum token columns.
type PostgresEvaluationTemplateItemRepository struct {
	pb.UnimplementedEvaluationTemplateItemDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresEvaluationTemplateItemRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.EvaluationTemplateItemDomainServiceServer {
	if tableName == "" {
		tableName = "evaluation_template_item"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresEvaluationTemplateItemRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func evaluationTemplateItemWriteMap(e *pb.EvaluationTemplateItem) (map[string]any, error) {
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

func evaluationTemplateItemFromResultJSON(resultJSON []byte) *pb.EvaluationTemplateItem {
	e := &pb.EvaluationTemplateItem{}
	_ = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, e)
	return e
}

func (r *PostgresEvaluationTemplateItemRepository) CreateEvaluationTemplateItem(ctx context.Context, req *pb.CreateEvaluationTemplateItemRequest) (*pb.CreateEvaluationTemplateItemResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("evaluation template item data is required")
	}
	data, err := evaluationTemplateItemWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluation template item: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.CreateEvaluationTemplateItemResponse{
		Data:    []*pb.EvaluationTemplateItem{evaluationTemplateItemFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationTemplateItemRepository) ReadEvaluationTemplateItem(ctx context.Context, req *pb.ReadEvaluationTemplateItemRequest) (*pb.ReadEvaluationTemplateItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation template item ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read evaluation template item: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.ReadEvaluationTemplateItemResponse{
		Data:    []*pb.EvaluationTemplateItem{evaluationTemplateItemFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationTemplateItemRepository) UpdateEvaluationTemplateItem(ctx context.Context, req *pb.UpdateEvaluationTemplateItemRequest) (*pb.UpdateEvaluationTemplateItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation template item ID is required")
	}
	data, err := evaluationTemplateItemWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update evaluation template item: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.UpdateEvaluationTemplateItemResponse{
		Data:    []*pb.EvaluationTemplateItem{evaluationTemplateItemFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationTemplateItemRepository) DeleteEvaluationTemplateItem(ctx context.Context, req *pb.DeleteEvaluationTemplateItemRequest) (*pb.DeleteEvaluationTemplateItemResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation template item ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete evaluation template item: %w", err)
	}
	return &pb.DeleteEvaluationTemplateItemResponse{Success: true}, nil
}

func (r *PostgresEvaluationTemplateItemRepository) ListEvaluationTemplateItems(ctx context.Context, req *pb.ListEvaluationTemplateItemsRequest) (*pb.ListEvaluationTemplateItemsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list evaluation template items: %w", err)
	}
	var items []*pb.EvaluationTemplateItem
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		items = append(items, evaluationTemplateItemFromResultJSON(resultJSON))
	}
	return &pb.ListEvaluationTemplateItemsResponse{Data: items, Success: true}, nil
}

var evaluationTemplateItemSortableSQLCols = []string{"sequence_order", "date_created"}

const evaluationTemplateItemSelectCols = `id, evaluation_template_id, workspace_id, outcome_criteria_id, sequence_order,
	question_label, question_prompt, required_override, weight_override, active, date_created`

func (r *PostgresEvaluationTemplateItemRepository) GetEvaluationTemplateItemListPageData(ctx context.Context, req *pb.GetEvaluationTemplateItemListPageDataRequest) (*pb.GetEvaluationTemplateItemListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	limit, offset, page := int32(50), int32(0), int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil && offsetPag.Page > 0 {
			page = offsetPag.Page
			offset = (page - 1) * limit
		}
	}
	orderBy, err := postgresCore.BuildOrderBy(evaluationTemplateItemSortableSQLCols, req.GetSort(), "sequence_order ASC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for evaluation template item list: %w", err)
	}
	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `SELECT ` + evaluationTemplateItemSelectCols + `
		FROM ` + r.tableName + `
		WHERE active = true AND ($3::text = '' OR workspace_id = $3::text) ` + orderBy + ` LIMIT $1 OFFSET $2;`
	rows, err := r.db.QueryContext(ctx, query, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*pb.EvaluationTemplateItem
	for rows.Next() {
		e, scanErr := scanEvaluationTemplateItemRow(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan failed: %w", scanErr)
		}
		items = append(items, e)
	}
	return &pb.GetEvaluationTemplateItemListPageDataResponse{
		EvaluationTemplateItemList: items,
		Pagination:                 &commonpb.PaginationResponse{CurrentPage: &page},
		Success:                    true,
	}, nil
}

func (r *PostgresEvaluationTemplateItemRepository) GetEvaluationTemplateItemItemPageData(ctx context.Context, req *pb.GetEvaluationTemplateItemItemPageDataRequest) (*pb.GetEvaluationTemplateItemItemPageDataResponse, error) {
	if req == nil || req.EvaluationTemplateItemId == "" {
		return nil, fmt.Errorf("evaluation template item ID required")
	}
	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	query := `SELECT ` + evaluationTemplateItemSelectCols + `
		FROM ` + r.tableName + `
		WHERE id = $1 AND active = true AND ($2::text = '' OR workspace_id = $2::text)`
	row := r.db.QueryRowContext(ctx, query, req.EvaluationTemplateItemId, wsID)
	e, err := scanEvaluationTemplateItemRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("evaluation template item not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &pb.GetEvaluationTemplateItemItemPageDataResponse{EvaluationTemplateItem: e, Success: true}, nil
}

func scanEvaluationTemplateItemRow(scan func(dest ...any) error) (*pb.EvaluationTemplateItem, error) {
	var id, evaluationTemplateID, workspaceID, outcomeCriteriaID string
	var questionLabel, questionPrompt sql.NullString
	var requiredOverride sql.NullBool
	var weightOverride sql.NullFloat64
	var sequenceOrder int32
	var active bool
	var dateCreated sql.NullTime
	if err := scan(&id, &evaluationTemplateID, &workspaceID, &outcomeCriteriaID, &sequenceOrder,
		&questionLabel, &questionPrompt, &requiredOverride, &weightOverride, &active, &dateCreated); err != nil {
		return nil, err
	}
	e := &pb.EvaluationTemplateItem{
		Id:                   id,
		EvaluationTemplateId: evaluationTemplateID,
		WorkspaceId:          workspaceID,
		OutcomeCriteriaId:    outcomeCriteriaID,
		SequenceOrder:        sequenceOrder,
		Active:               active,
	}
	setOptStr(&e.QuestionLabel, questionLabel)
	setOptStr(&e.QuestionPrompt, questionPrompt)
	if requiredOverride.Valid {
		v := requiredOverride.Bool
		e.RequiredOverride = &v
	}
	if weightOverride.Valid {
		v := weightOverride.Float64
		e.WeightOverride = &v
	}
	if dateCreated.Valid && !dateCreated.Time.IsZero() {
		ts := dateCreated.Time.UnixMilli()
		e.DateCreated = &ts
		s := dateCreated.Time.Format("2006-01-02T15:04:05Z07:00")
		e.DateCreatedString = &s
	}
	return e, nil
}

func NewEvaluationTemplateItemRepository(db *sql.DB, tableName string) pb.EvaluationTemplateItemDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEvaluationTemplateItemRepository(dbOps, tableName)
}
