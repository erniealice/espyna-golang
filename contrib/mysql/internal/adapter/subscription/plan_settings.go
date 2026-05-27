//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/database/operations"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
	"google.golang.org/protobuf/encoding/protojson"
)

// MySQLPlanSettingsRepository implements plan_settings CRUD using MySQL 8.0+.
type MySQLPlanSettingsRepository struct {
	plansettingspb.UnimplementedPlanSettingsDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.PlanSettings, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql plan_settings repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLPlanSettingsRepository(dbOps, tableName), nil
	})
}

// NewMySQLPlanSettingsRepository creates a new MySQL plan_settings repository.
func NewMySQLPlanSettingsRepository(dbOps interfaces.DatabaseOperation, tableName string) plansettingspb.PlanSettingsDomainServiceServer {
	if tableName == "" {
		tableName = "plan_settings"
	}
	return &MySQLPlanSettingsRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLPlanSettingsRepository) CreatePlanSettings(ctx context.Context, req *plansettingspb.CreatePlanSettingsRequest) (*plansettingspb.CreatePlanSettingsResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan_settings data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan_settings: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ps := &plansettingspb.PlanSettings{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &plansettingspb.CreatePlanSettingsResponse{Data: []*plansettingspb.PlanSettings{ps}}, nil
}

func (r *MySQLPlanSettingsRepository) ReadPlanSettings(ctx context.Context, req *plansettingspb.ReadPlanSettingsRequest) (*plansettingspb.ReadPlanSettingsResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan_settings ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan_settings: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ps := &plansettingspb.PlanSettings{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &plansettingspb.ReadPlanSettingsResponse{Data: []*plansettingspb.PlanSettings{ps}}, nil
}

func (r *MySQLPlanSettingsRepository) UpdatePlanSettings(ctx context.Context, req *plansettingspb.UpdatePlanSettingsRequest) (*plansettingspb.UpdatePlanSettingsResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan_settings ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan_settings: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ps := &plansettingspb.PlanSettings{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &plansettingspb.UpdatePlanSettingsResponse{Data: []*plansettingspb.PlanSettings{ps}}, nil
}

func (r *MySQLPlanSettingsRepository) DeletePlanSettings(ctx context.Context, req *plansettingspb.DeletePlanSettingsRequest) (*plansettingspb.DeletePlanSettingsResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan_settings ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete plan_settings: %w", err)
	}
	return &plansettingspb.DeletePlanSettingsResponse{Success: true}, nil
}

func (r *MySQLPlanSettingsRepository) ListPlanSettings(ctx context.Context, req *plansettingspb.ListPlanSettingsRequest) (*plansettingspb.ListPlanSettingsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan_settings: %w", err)
	}
	var planSettingsList []*plansettingspb.PlanSettings
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		ps := &plansettingspb.PlanSettings{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ps); err != nil {
			continue
		}
		planSettingsList = append(planSettingsList, ps)
	}
	return &plansettingspb.ListPlanSettingsResponse{Data: planSettingsList}, nil
}

// ListPlanSettingsByPlan lists plan settings by plan ID.
// Dialect: uses dbOps.Query with the same QueryBuilder as postgres.
func (r *MySQLPlanSettingsRepository) ListPlanSettingsByPlan(ctx context.Context, req *plansettingspb.ListPlanSettingsByPlanRequest) (*plansettingspb.ListPlanSettingsByPlanResponse, error) {
	if req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}
	query := operations.NewQueryBuilder().Where("plan_id", "==", req.PlanId)
	results, err := r.dbOps.Query(ctx, r.tableName, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan settings by plan: %w", err)
	}
	var planSettingsList []*plansettingspb.PlanSettings
	for _, result := range results {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		ps := &plansettingspb.PlanSettings{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ps); err != nil {
			continue
		}
		planSettingsList = append(planSettingsList, ps)
	}
	return &plansettingspb.ListPlanSettingsByPlanResponse{Data: planSettingsList}, nil
}

// GetPlanSettingsListPageData retrieves paginated plan_settings list data.
//
// Dialect: $N → ?, ILIKE → LIKE, active = true → active = 1,
// WHERE workspace_id = ? via WorkspaceAwareOperations.
func (r *MySQLPlanSettingsRepository) GetPlanSettingsListPageData(ctx context.Context, req *plansettingspb.ListPlanSettingsRequest) (*plansettingspb.ListPlanSettingsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern := ""
	limit, offset := int32(50), int32(0)

	// Dialect: ILIKE → LIKE, active = true → active = 1, $N → ?.
	// ORDER BY is author-controlled (no caller interpolation).
	query := `SELECT id, plan_id, name, description, active, date_created, date_modified FROM plan_settings WHERE active = 1 AND (? IS NULL OR ? = '' OR plan_id LIKE ?) ORDER BY date_created DESC LIMIT ? OFFSET ?`

	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, searchPattern, searchPattern, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var planSettingsList []*plansettingspb.PlanSettings
	for rows.Next() {
		var id, planId, name, description string
		var active bool
		var dateCreated, dateModified sql.NullInt64
		if err := rows.Scan(&id, &planId, &name, &description, &active, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("scan plan_settings row: %w", err)
		}
		ps := &plansettingspb.PlanSettings{
			Id:          id,
			PlanId:      planId,
			Name:        name,
			Description: description,
			Active:      active,
		}
		if dateCreated.Valid {
			ps.DateCreated = &dateCreated.Int64
		}
		if dateModified.Valid {
			ps.DateModified = &dateModified.Int64
		}
		planSettingsList = append(planSettingsList, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating plan_settings rows: %w", err)
	}
	return &plansettingspb.ListPlanSettingsResponse{Data: planSettingsList}, nil
}
