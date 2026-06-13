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
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_cycle"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.EvaluationCycle, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres evaluation_cycle repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresEvaluationCycleRepository(dbOps, tableName), nil
	})
}

// PostgresEvaluationCycleRepository implements evaluation_cycle CRUD. The status
// (EvaluationCycleStatus) column is a DB CHECK-pinned lowercase token; this
// adapter translates proto-enum ↔ token in both directions.
type PostgresEvaluationCycleRepository struct {
	pb.UnimplementedEvaluationCycleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresEvaluationCycleRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.EvaluationCycleDomainServiceServer {
	if tableName == "" {
		tableName = "evaluation_cycle"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresEvaluationCycleRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func evaluationCycleWriteMap(e *pb.EvaluationCycle) (map[string]any, error) {
	jsonData, err := protojson.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	setOrDeleteToken(data, "status", evaluationCycleStatusTokenFromEnum(e.Status))
	return data, nil
}

func evaluationCycleFromResultJSON(resultJSON []byte) *pb.EvaluationCycle {
	var raw map[string]any
	_ = json.Unmarshal(resultJSON, &raw)
	statusTok, _ := raw["status"].(string)
	delete(raw, "status")
	cleaned, _ := json.Marshal(raw)

	e := &pb.EvaluationCycle{}
	_ = (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(cleaned, e)
	e.Status = evaluationCycleStatusFromString(statusTok)
	return e
}

func (r *PostgresEvaluationCycleRepository) CreateEvaluationCycle(ctx context.Context, req *pb.CreateEvaluationCycleRequest) (*pb.CreateEvaluationCycleResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("evaluation cycle data is required")
	}
	data, err := evaluationCycleWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluation cycle: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.CreateEvaluationCycleResponse{
		Data:    []*pb.EvaluationCycle{evaluationCycleFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationCycleRepository) ReadEvaluationCycle(ctx context.Context, req *pb.ReadEvaluationCycleRequest) (*pb.ReadEvaluationCycleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation cycle ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read evaluation cycle: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.ReadEvaluationCycleResponse{
		Data:    []*pb.EvaluationCycle{evaluationCycleFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationCycleRepository) UpdateEvaluationCycle(ctx context.Context, req *pb.UpdateEvaluationCycleRequest) (*pb.UpdateEvaluationCycleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation cycle ID is required")
	}
	data, err := evaluationCycleWriteMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update evaluation cycle: %w", err)
	}
	resultJSON, _ := json.Marshal(result)
	return &pb.UpdateEvaluationCycleResponse{
		Data:    []*pb.EvaluationCycle{evaluationCycleFromResultJSON(resultJSON)},
		Success: true,
	}, nil
}

func (r *PostgresEvaluationCycleRepository) DeleteEvaluationCycle(ctx context.Context, req *pb.DeleteEvaluationCycleRequest) (*pb.DeleteEvaluationCycleResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("evaluation cycle ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete evaluation cycle: %w", err)
	}
	return &pb.DeleteEvaluationCycleResponse{Success: true}, nil
}

func (r *PostgresEvaluationCycleRepository) ListEvaluationCycles(ctx context.Context, req *pb.ListEvaluationCyclesRequest) (*pb.ListEvaluationCyclesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list evaluation cycles: %w", err)
	}
	var items []*pb.EvaluationCycle
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		items = append(items, evaluationCycleFromResultJSON(resultJSON))
	}
	return &pb.ListEvaluationCyclesResponse{Data: items, Success: true}, nil
}

var evaluationCycleSortableSQLCols = []string{
	"name", "period_start", "period_end", "status", "close_date", "date_created", "date_modified",
}

const evaluationCycleSelectCols = `id, workspace_id, subscription_id, name, period_start, period_end,
	sign_off_due_date, close_date, status, active, date_created, date_modified`

func (r *PostgresEvaluationCycleRepository) GetEvaluationCycleListPageData(ctx context.Context, req *pb.GetEvaluationCycleListPageDataRequest) (*pb.GetEvaluationCycleListPageDataResponse, error) {
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
	orderBy, err := postgresCore.BuildOrderBy(evaluationCycleSortableSQLCols, req.GetSort(), "period_start DESC")
	if err != nil {
		return nil, fmt.Errorf("invalid sort for evaluation cycle list: %w", err)
	}
	wsID := identity.Must(ctx).WorkspaceID
	query := `SELECT ` + evaluationCycleSelectCols + `
		FROM ` + r.tableName + `
		WHERE active = true
			AND ($4::text = '' OR workspace_id = $4::text)
			AND ($1::text IS NULL OR $1::text = '' OR name ILIKE $1) ` + orderBy + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset, wsID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var items []*pb.EvaluationCycle
	for rows.Next() {
		e, scanErr := scanEvaluationCycleRow(rows.Scan)
		if scanErr != nil {
			return nil, fmt.Errorf("scan failed: %w", scanErr)
		}
		items = append(items, e)
	}
	return &pb.GetEvaluationCycleListPageDataResponse{
		EvaluationCycleList: items,
		Pagination:          &commonpb.PaginationResponse{CurrentPage: &page},
		Success:             true,
	}, nil
}

func (r *PostgresEvaluationCycleRepository) GetEvaluationCycleItemPageData(ctx context.Context, req *pb.GetEvaluationCycleItemPageDataRequest) (*pb.GetEvaluationCycleItemPageDataResponse, error) {
	if req == nil || req.EvaluationCycleId == "" {
		return nil, fmt.Errorf("evaluation cycle ID required")
	}
	wsID := identity.Must(ctx).WorkspaceID
	query := `SELECT ` + evaluationCycleSelectCols + `
		FROM ` + r.tableName + `
		WHERE id = $1 AND active = true AND ($2::text = '' OR workspace_id = $2::text)`
	row := r.db.QueryRowContext(ctx, query, req.EvaluationCycleId, wsID)
	e, err := scanEvaluationCycleRow(row.Scan)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("evaluation cycle not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return &pb.GetEvaluationCycleItemPageDataResponse{EvaluationCycle: e, Success: true}, nil
}

func scanEvaluationCycleRow(scan func(dest ...any) error) (*pb.EvaluationCycle, error) {
	var id, workspaceID, subscriptionID, name, periodStart, periodEnd, statusTok string
	var signOffDueDate, closeDate sql.NullString
	var active bool
	var dateCreated, dateModified sql.NullTime
	if err := scan(&id, &workspaceID, &subscriptionID, &name, &periodStart, &periodEnd,
		&signOffDueDate, &closeDate, &statusTok, &active, &dateCreated, &dateModified); err != nil {
		return nil, err
	}
	e := &pb.EvaluationCycle{
		Id:             id,
		WorkspaceId:    workspaceID,
		SubscriptionId: subscriptionID,
		Name:           name,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
		Status:         evaluationCycleStatusFromString(statusTok),
		Active:         active,
	}
	setOptStr(&e.SignOffDueDate, signOffDueDate)
	setOptStr(&e.CloseDate, closeDate)
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

func NewEvaluationCycleRepository(db *sql.DB, tableName string) pb.EvaluationCycleDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresEvaluationCycleRepository(dbOps, tableName)
}
