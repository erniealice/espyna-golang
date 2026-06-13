//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/shared/identity"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EvaluationTemplate, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres evaluation_template repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEvaluationTemplateRepository(dbOps, tableName), nil
	})
}

// PostgresEvaluationTemplateRepository implements evaluation_template CRUD.
//
// The status (EvaluationTemplateStatus), evaluation_type, relationship_type and
// visibility_type columns are DB CHECK-pinned lowercase tokens; this adapter
// translates proto-enum ↔ token in both directions (see evaluation_enums.go).
type PostgresEvaluationTemplateRepository struct {
	pb.UnimplementedEvaluationTemplateDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresEvaluationTemplateRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.EvaluationTemplateDomainServiceServer {
	if tableName == "" {
		tableName = "evaluation_template"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresEvaluationTemplateRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func evaluationTemplateWriteMap(e *pb.EvaluationTemplate) (map[string]any, error) {
	jsonData, err := protojson.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	setOrDeleteToken(data, "status", evaluationTemplateStatusTokenFromEnum(e.Status))
	setOrDeleteToken(data, "evaluationType", evaluationTypeTokenFromEnum(e.EvaluationType))
	setOrDeleteToken(data, "relationshipType", relationshipTypeTokenFromEnum(e.RelationshipType))
	setOrDeleteToken(data, "visibilityType", visibilityTypeTokenFromEnum(e.VisibilityType))
	return data, nil
}

func evaluationTemplateFromResultJSON(resultJSON []byte) *pb.EvaluationTemplate {
	var raw map[string]any
	_ = json.Unmarshal(resultJSON, &raw)
	statusTok, _ := raw["status"].(string)
	typeTok, _ := raw["evaluationType"].(string)
	relTok, _ := raw["relationshipType"].(string)
	visTok, _ := raw["visibilityType"].(string)
	delete(raw, "status")
	delete(raw, "evaluationType")
	delete(raw, "relationshipType")
	delete(raw, "visibilityType")
	cleaned, _ := json.Marshal(raw)

	e := &pb.EvaluationTemplate{}
	_ = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(cleaned, e)
	e.Status = evaluationTemplateStatusFromString(statusTok)
	e.EvaluationType = evaluationTypeFromString(typeTok)
	e.RelationshipType = relationshipTypeFromString(relTok)
	e.VisibilityType = visibilityTypeFromString(visTok)
	return e
}

func (r *PostgresEvaluationTemplateRepository) CreateEvaluationTemplate(ctx context.Context, req *pb.CreateEvaluationTemplateRequest) (*pb.CreateEvaluationTemplateResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("evaluation template data is required")
	}
	data, err := evaluationTemplateWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluation template: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.CreateEvaluationTemplateResponse{
		Data:    []*pb.EvaluationTemplate{evaluationTemplateFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationTemplateRepository) ReadEvaluationTemplate(ctx context.Context, req *pb.ReadEvaluationTemplateRequest) (*pb.ReadEvaluationTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation template ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read evaluation template: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.ReadEvaluationTemplateResponse{
		Data:    []*pb.EvaluationTemplate{evaluationTemplateFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationTemplateRepository) UpdateEvaluationTemplate(ctx context.Context, req *pb.UpdateEvaluationTemplateRequest) (*pb.UpdateEvaluationTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation template ID is required")
	}
	data, err := evaluationTemplateWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update evaluation template: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.UpdateEvaluationTemplateResponse{
		Data:    []*pb.EvaluationTemplate{evaluationTemplateFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationTemplateRepository) DeleteEvaluationTemplate(ctx context.Context, req *pb.DeleteEvaluationTemplateRequest) (*pb.DeleteEvaluationTemplateResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation template ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete evaluation template: %w", err)
	}
	return &pb.DeleteEvaluationTemplateResponse{Success: true}, nil
}

func (r *PostgresEvaluationTemplateRepository) ListEvaluationTemplates(ctx context.Context, req *pb.ListEvaluationTemplatesRequest) (*pb.ListEvaluationTemplatesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list evaluation templates: %w", err)
	}
	var items []*pb.EvaluationTemplate
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		items = append(items, evaluationTemplateFromResultJSON(resultJSON))
	}
	return &pb.ListEvaluationTemplatesResponse{Data: items, Success: true}, nil
}

var evaluationTemplateSortableSQLCols = []string{
	"name", "version", "status", "date_created", "date_modified",
}

const evaluationTemplateSelectCols = `id, workspace_id, name, description, evaluation_type, relationship_type,
	version, status, visibility_type, copied_from_id, active, date_created, date_modified`

// GetEvaluationTemplateListPageData retrieves paginated template data. Templates
// are workspace-scoped (NOT client-scoped).
func (r *PostgresEvaluationTemplateRepository) GetEvaluationTemplateListPageData(ctx context.Context, req *pb.GetEvaluationTemplateListPageDataRequest) (*pb.GetEvaluationTemplateListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
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
	orderBy, err := postgresCore.BuildOrderBy(evaluationTemplateSortableSQLCols, req.GetSort(), "name ASC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for evaluation template list: %w", err)
	}
	wsID := identity.Must(ctx).WorkspaceID
	query := `SELECT ` + evaluationTemplateSelectCols + `
		FROM ` + r.tableName + `
		WHERE active = true
			AND ($4::text = '' OR workspace_id = $4::text)
			AND ($1::text IS NULL OR $1::text = '' OR name ILIKE $1) ` + orderBy + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*pb.EvaluationTemplate
	for rows.Next() {
		e, scanErr := scanEvaluationTemplateRow(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan failed: %w", scanErr)
		}
		items = append(items, e)
	}
	return &pb.GetEvaluationTemplateListPageDataResponse{
		EvaluationTemplateList: items,
		Pagination:             &commonpb.PaginationResponse{CurrentPage: &page},
		Success:                true,
	}, nil
}

func (r *PostgresEvaluationTemplateRepository) GetEvaluationTemplateItemPageData(ctx context.Context, req *pb.GetEvaluationTemplateItemPageDataRequest) (*pb.GetEvaluationTemplateItemPageDataResponse, error) {
	if req == nil || req.EvaluationTemplateId == "" {
		return nil, fmt.Errorf("evaluation template ID required")
	}
	wsID := identity.Must(ctx).WorkspaceID
	query := `SELECT ` + evaluationTemplateSelectCols + `
		FROM ` + r.tableName + `
		WHERE id = $1 AND active = true AND ($2::text = '' OR workspace_id = $2::text)`
	row := r.db.QueryRowContext(ctx, query, req.EvaluationTemplateId, wsID)
	e, err := scanEvaluationTemplateRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("evaluation template not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &pb.GetEvaluationTemplateItemPageDataResponse{EvaluationTemplate: e, Success: true}, nil
}

func scanEvaluationTemplateRow(scan func(dest ...any) error) (*pb.EvaluationTemplate, error) {
	var id, workspaceID, name, typeTok, relTok, statusTok, visTok string
	var description, copiedFromID sql.NullString
	var version int32
	var active bool
	var dateCreated, dateModified sql.NullTime
	if err := scan(&id, &workspaceID, &name, &description, &typeTok, &relTok,
		&version, &statusTok, &visTok, &copiedFromID, &active, &dateCreated, &dateModified); err != nil {
		return nil, err
	}
	e := &pb.EvaluationTemplate{
		Id:               id,
		WorkspaceId:      workspaceID,
		Name:             name,
		EvaluationType:   evaluationTypeFromString(typeTok),
		RelationshipType: relationshipTypeFromString(relTok),
		Version:          version,
		Status:           evaluationTemplateStatusFromString(statusTok),
		VisibilityType:   visibilityTypeFromString(visTok),
		Active:           active,
	}
	setOptStr(&e.Description, description)
	setOptStr(&e.CopiedFromId, copiedFromID)
	if dateCreated.Valid && !dateCreated.Time.IsZero() {
		ts := dateCreated.Time.UnixMilli()
		e.DateCreated = &ts
		s := dateCreated.Time.Format("2006-01-02T15:04:05Z07:00")
		e.DateCreatedString = &s
	}
	if dateModified.Valid && !dateModified.Time.IsZero() {
		ts := dateModified.Time.UnixMilli()
		e.DateModified = &ts
		s := dateModified.Time.Format("2006-01-02T15:04:05Z07:00")
		e.DateModifiedString = &s
	}
	return e, nil
}

func NewEvaluationTemplateRepository(db *sql.DB, tableName string) pb.EvaluationTemplateDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEvaluationTemplateRepository(dbOps, tableName)
}
