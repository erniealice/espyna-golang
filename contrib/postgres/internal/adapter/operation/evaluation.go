//go:build postgresql

package operation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/erniealice/espyna-golang/consumer"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Evaluation, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres evaluation repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEvaluationRepository(dbOps, tableName), nil
	})
}

// evaluationSortableSQLCols whitelists the sort columns for list page data
// (A2 fail-closed guard).
var evaluationSortableSQLCols = []string{
	"status", "period_start", "period_end", "overall_score",
	"submitted_at", "date_created", "date_modified",
}

// PostgresEvaluationRepository implements evaluation CRUD using PostgreSQL.
//
// IDOR: every list/read path scopes workspace_id AND (client path) client_id.
// The status/type/visibility columns are DB CHECK-pinned lowercase tokens; this
// adapter translates proto-enum ↔ token in both directions (see
// evaluation_enums.go), mirroring subscription_seat.
type PostgresEvaluationRepository struct {
	pb.UnimplementedEvaluationDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresEvaluationRepository creates a new PostgreSQL evaluation repository.
func NewPostgresEvaluationRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.EvaluationDomainServiceServer {
	if tableName == "" {
		tableName = "evaluation"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresEvaluationRepository{dbOps: dbOps, db: db, tableName: tableName}
}

// evaluationWriteMap serializes an Evaluation to the DB write map, translating
// every CHECK-pinned enum column to its lowercase token.
func evaluationWriteMap(e *pb.Evaluation) (map[string]any, error) {
	jsonData, err := protojson.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	setOrDeleteToken(data, "status", evaluationStatusTokenFromEnum(e.Status))
	setOrDeleteToken(data, "evaluationType", evaluationTypeTokenFromEnum(e.EvaluationType))
	setOrDeleteToken(data, "relationshipType", relationshipTypeTokenFromEnum(e.RelationshipType))
	setOrDeleteToken(data, "evaluatorType", evaluatorTypeTokenFromEnum(e.EvaluatorType))
	setOrDeleteToken(data, "subjectType", subjectTypeTokenFromEnum(e.SubjectType))
	setOrDeleteToken(data, "visibilityType", visibilityTypeTokenFromEnum(e.VisibilityType))
	return data, nil
}

// evaluationFromResultJSON unmarshals an adapter result row (DB token form) into
// a proto Evaluation. The enum token columns are stripped before
// protojson.Unmarshal (which rejects unknown enum values) and set from tokens.
func evaluationFromResultJSON(resultJSON []byte) *pb.Evaluation {
	var raw map[string]any
	_ = json.Unmarshal(resultJSON, &raw)
	statusTok, _ := raw["status"].(string)
	typeTok, _ := raw["evaluationType"].(string)
	relTok, _ := raw["relationshipType"].(string)
	evalrTok, _ := raw["evaluatorType"].(string)
	subjTok, _ := raw["subjectType"].(string)
	visTok, _ := raw["visibilityType"].(string)
	delete(raw, "status")
	delete(raw, "evaluationType")
	delete(raw, "relationshipType")
	delete(raw, "evaluatorType")
	delete(raw, "subjectType")
	delete(raw, "visibilityType")
	cleaned, _ := json.Marshal(raw)

	e := &pb.Evaluation{}
	_ = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(cleaned, e)
	e.Status = evaluationStatusFromString(statusTok)
	e.EvaluationType = evaluationTypeFromString(typeTok)
	e.RelationshipType = relationshipTypeFromString(relTok)
	e.EvaluatorType = evaluatorTypeFromString(evalrTok)
	e.SubjectType = subjectTypeFromString(subjTok)
	e.VisibilityType = visibilityTypeFromString(visTok)
	return e
}

func (r *PostgresEvaluationRepository) CreateEvaluation(ctx context.Context, req *pb.CreateEvaluationRequest) (*pb.CreateEvaluationResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("evaluation data is required")
	}
	data, err := evaluationWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluation: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	return &pb.CreateEvaluationResponse{
		Data:    []*pb.Evaluation{evaluationFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationRepository) ReadEvaluation(ctx context.Context, req *pb.ReadEvaluationRequest) (*pb.ReadEvaluationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read evaluation: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	return &pb.ReadEvaluationResponse{
		Data:    []*pb.Evaluation{evaluationFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationRepository) UpdateEvaluation(ctx context.Context, req *pb.UpdateEvaluationRequest) (*pb.UpdateEvaluationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation ID is required")
	}
	data, err := evaluationWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update evaluation: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	return &pb.UpdateEvaluationResponse{
		Data:    []*pb.Evaluation{evaluationFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationRepository) DeleteEvaluation(ctx context.Context, req *pb.DeleteEvaluationRequest) (*pb.DeleteEvaluationResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete evaluation: %w", err)
	}
	return &pb.DeleteEvaluationResponse{Success: true}, nil
}

func (r *PostgresEvaluationRepository) ListEvaluations(ctx context.Context, req *pb.ListEvaluationsRequest) (*pb.ListEvaluationsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list evaluations: %w", err)
	}
	var items []*pb.Evaluation
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		items = append(items, evaluationFromResultJSON(resultJSON))
	}
	return &pb.ListEvaluationsResponse{Data: items, Success: true}, nil
}

// evaluationSelectCols is the canonical projection used by the raw list/item
// queries; the scanEvaluationRow column order MUST match it.
const evaluationSelectCols = `id, workspace_id, client_id, subscription_id, subscription_seat_id, evaluation_template_id,
	evaluation_type, relationship_type, evaluator_type, evaluator_workspace_user_id, evaluator_client_portal_grant_id,
	subject_type, subject_staff_id, subject_client_id, period_start, period_end, status, visibility_type,
	overall_score, narrative, submitted_at, active, date_created, date_modified, evaluation_cycle_id,
	signed_off_by_workspace_user_id, signed_off_by_client_portal_grant_id, signed_off_at`

// GetEvaluationListPageData retrieves paginated evaluation list data.
//
// IDOR (Q-EVAL-IDOR-1): scopes workspace_id ALWAYS, and when the caller is acting
// as a client (acting_as_client_id set) it ALSO scopes client_id = acting_as AND
// enforces visibility_type <> 'internal_only' in the query predicate (raw
// handlers bypass the ViewAdapter, so this gate must live in the query, not only
// the view). A staff caller (empty acting_as) sees the workspace scope only.
func (r *PostgresEvaluationRepository) GetEvaluationListPageData(ctx context.Context, req *pb.GetEvaluationListPageDataRequest) (*pb.GetEvaluationListPageDataResponse, error) {
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
	orderBy, err := postgresCore.BuildOrderBy(evaluationSortableSQLCols, req.GetSort(), "date_created DESC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for evaluation list: %w", err)
	}

	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	actingClient := consumer.GetActingAsClientIDFromContext(ctx)

	// $5 = acting_as_client_id. When set, scope client_id AND fail-closed on
	// internal_only visibility. When empty (staff), no client/visibility gate.
	query := `SELECT ` + evaluationSelectCols + `
		FROM ` + r.tableName + `
		WHERE active = true
			AND ($4::text = '' OR workspace_id = $4::text)
			AND ($5::text = '' OR (client_id = $5::text AND visibility_type <> 'internal_only'))
			AND ($1::text IS NULL OR $1::text = '' OR COALESCE(narrative,'') ILIKE $1 OR status ILIKE $1) ` + orderBy + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, wsID, actingClient)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*pb.Evaluation
	for rows.Next() {
		e, scanErr := scanEvaluationRow(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan failed: %w", scanErr)
		}
		items = append(items, e)
	}
	return &pb.GetEvaluationListPageDataResponse{
		EvaluationList: items,
		Pagination:     &commonpb.PaginationResponse{CurrentPage: &page},
		Success:        true,
	}, nil
}

// GetEvaluationItemPageData retrieves a single evaluation.
//
// IDOR (Q-EVAL-IDOR-1): workspace_id scope ALWAYS; client path adds client_id +
// visibility_type <> 'internal_only' (raw-handler-safe gate).
func (r *PostgresEvaluationRepository) GetEvaluationItemPageData(ctx context.Context, req *pb.GetEvaluationItemPageDataRequest) (*pb.GetEvaluationItemPageDataResponse, error) {
	if req == nil || req.EvaluationId == "" {
		return nil, fmt.Errorf("evaluation ID required")
	}
	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	actingClient := consumer.GetActingAsClientIDFromContext(ctx)
	query := `SELECT ` + evaluationSelectCols + `
		FROM ` + r.tableName + `
		WHERE id = $1 AND active = true
			AND ($2::text = '' OR workspace_id = $2::text)
			AND ($3::text = '' OR (client_id = $3::text AND visibility_type <> 'internal_only'))`
	row := r.db.QueryRowContext(ctx, query, req.EvaluationId, wsID, actingClient)
	e, err := scanEvaluationRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("evaluation not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &pb.GetEvaluationItemPageDataResponse{Evaluation: e, Success: true}, nil
}

// GetLatestEvaluationScore returns the latest SUBMITTED/SIGNED_OFF overall_score
// per subject_staff_id (Q-RATING-XJOIN-1). Batched over staffIDs; tie-break is
// ORDER BY submitted_at DESC, id DESC. The returned map omits staff with no
// scored evaluation. This is a PURE workspace-scoped read; the servicing gate
// (Q-SERVICING-SCOPE-1 / CR-5) is applied by the calling service-layer UC/view,
// NOT here.
func (r *PostgresEvaluationRepository) GetLatestEvaluationScore(ctx context.Context, staffIDs []string) (map[string]*float64, error) {
	out := make(map[string]*float64)
	if len(staffIDs) == 0 {
		return out, nil
	}
	wsID := consumer.GetWorkspaceIDFromContext(ctx)
	// DISTINCT ON (subject_staff_id) keeps the first row per staff after the
	// deterministic ORDER BY — i.e. the latest scored evaluation.
	query := `SELECT DISTINCT ON (subject_staff_id) subject_staff_id, overall_score
		FROM ` + r.tableName + `
		WHERE subject_staff_id = ANY($1)
			AND active = true
			AND status IN ('submitted', 'signed_off')
			AND overall_score IS NOT NULL
			AND ($2::text = '' OR workspace_id = $2::text)
		ORDER BY subject_staff_id, submitted_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, sqlStringArray(staffIDs), wsID)
	if err != nil {
		return nil, fmt.Errorf("latest evaluation score query failed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var staffID string
		var score sql.NullFloat64
		if err := rows.Scan(&staffID, &score); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		if score.Valid {
			v := score.Float64
			out[staffID] = &v
		}
	}
	return out, nil
}

// scanEvaluationRow scans one evaluation row (column order MUST match
// evaluationSelectCols) into a proto Evaluation, mapping the token enum columns
// onto the proto enums and nullable columns onto the optional proto fields.
func scanEvaluationRow(scan func(dest ...any) error) (*pb.Evaluation, error) {
	var id, workspaceID, clientID, periodStart, periodEnd, statusTok, visTok, typeTok, relTok, evalrTok, subjTok string
	var subscriptionID, subscriptionSeatID, evaluationTemplateID sql.NullString
	var evaluatorWorkspaceUserID, evaluatorClientPortalGrantID sql.NullString
	var subjectStaffID, subjectClientID, narrative sql.NullString
	var signedOffByWorkspaceUserID, signedOffByClientPortalGrantID sql.NullString
	var evaluationCycleID sql.NullString
	var overallScore sql.NullFloat64
	var submittedAt, signedOffAt sql.NullInt64
	var active bool
	var dateCreated, dateModified time.Time

	if err := scan(
		&id, &workspaceID, &clientID, &subscriptionID, &subscriptionSeatID, &evaluationTemplateID,
		&typeTok, &relTok, &evalrTok, &evaluatorWorkspaceUserID, &evaluatorClientPortalGrantID,
		&subjTok, &subjectStaffID, &subjectClientID, &periodStart, &periodEnd, &statusTok, &visTok,
		&overallScore, &narrative, &submittedAt, &active, &dateCreated, &dateModified, &evaluationCycleID,
		&signedOffByWorkspaceUserID, &signedOffByClientPortalGrantID, &signedOffAt,
	); err != nil {
		return nil, err
	}
	e := &pb.Evaluation{
		Id:               id,
		WorkspaceId:      workspaceID,
		ClientId:         clientID,
		EvaluationType:   evaluationTypeFromString(typeTok),
		RelationshipType: relationshipTypeFromString(relTok),
		EvaluatorType:    evaluatorTypeFromString(evalrTok),
		SubjectType:      subjectTypeFromString(subjTok),
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		Status:           evaluationStatusFromString(statusTok),
		VisibilityType:   visibilityTypeFromString(visTok),
		Active:           active,
	}
	setOptStr(&e.SubscriptionId, subscriptionID)
	setOptStr(&e.SubscriptionSeatId, subscriptionSeatID)
	setOptStr(&e.EvaluationTemplateId, evaluationTemplateID)
	setOptStr(&e.EvaluatorWorkspaceUserId, evaluatorWorkspaceUserID)
	setOptStr(&e.EvaluatorClientPortalGrantId, evaluatorClientPortalGrantID)
	setOptStr(&e.SubjectStaffId, subjectStaffID)
	setOptStr(&e.SubjectClientId, subjectClientID)
	setOptStr(&e.Narrative, narrative)
	setOptStr(&e.EvaluationCycleId, evaluationCycleID)
	setOptStr(&e.SignedOffByWorkspaceUserId, signedOffByWorkspaceUserID)
	setOptStr(&e.SignedOffByClientPortalGrantId, signedOffByClientPortalGrantID)
	if overallScore.Valid {
		v := overallScore.Float64
		e.OverallScore = &v
	}
	if submittedAt.Valid {
		v := submittedAt.Int64
		e.SubmittedAt = &v
	}
	if signedOffAt.Valid {
		v := signedOffAt.Int64
		e.SignedOffAt = &v
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		e.DateCreated = &ts
		s := dateCreated.Format(time.RFC3339)
		e.DateCreatedString = &s
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		e.DateModified = &ts
		s := dateModified.Format(time.RFC3339)
		e.DateModifiedString = &s
	}
	return e, nil
}

// setOptStr copies a non-empty NullString into a *string proto optional field.
func setOptStr(dst **string, ns sql.NullString) {
	if ns.Valid && ns.String != "" {
		v := ns.String
		*dst = &v
	}
}

// sqlStringArray formats a Go []string as a Postgres text[] literal for ANY($1).
func sqlStringArray(ss []string) string {
	if len(ss) == 0 {
		return "{}"
	}
	escaped := make([]string, len(ss))
	for i, s := range ss {
		escaped[i] = `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
	}
	return "{" + strings.Join(escaped, ",") + "}"
}

// NewEvaluationRepository creates a new PostgreSQL evaluation repository (old-style).
func NewEvaluationRepository(db *sql.DB, tableName string) pb.EvaluationDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEvaluationRepository(dbOps, tableName)
}
