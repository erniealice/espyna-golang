//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/operations"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	plansettingspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_settings"
)

// PostgresPlanSettingsRepository implements plan_settings CRUD operations using PostgreSQL
//
// Performance Index Recommendations:
//   - CREATE INDEX idx_plan_settings_active ON plan_settings(active) WHERE active = true - Filter active records
//   - CREATE INDEX idx_plan_settings_plan_id ON plan_settings(plan_id) - Filter by plan
//   - CREATE INDEX idx_plan_settings_date_created ON plan_settings(date_created DESC) - Default sorting
type PostgresPlanSettingsRepository struct {
	plansettingspb.UnimplementedPlanSettingsDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", "plan_settings", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres plan_settings repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresPlanSettingsRepository(dbOps, tableName), nil
	})
}

// NewPostgresPlanSettingsRepository creates a new PostgreSQL plan settings repository
func NewPostgresPlanSettingsRepository(dbOps interfaces.DatabaseOperation, tableName string) plansettingspb.PlanSettingsDomainServiceServer {
	if tableName == "" {
		tableName = "plan_settings" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPlanSettingsRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePlanSettings creates a new plan settings using common PostgreSQL operations
func (r *PostgresPlanSettingsRepository) CreatePlanSettings(ctx context.Context, req *plansettingspb.CreatePlanSettingsRequest) (*plansettingspb.CreatePlanSettingsResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan settings data is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan settings: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	planSettings := &plansettingspb.PlanSettings{}
	if err := protojson.Unmarshal(resultJSON, planSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &plansettingspb.CreatePlanSettingsResponse{
		Data: []*plansettingspb.PlanSettings{planSettings},
	}, nil
}

// ReadPlanSettings retrieves a plan settings using common PostgreSQL operations
func (r *PostgresPlanSettingsRepository) ReadPlanSettings(ctx context.Context, req *plansettingspb.ReadPlanSettingsRequest) (*plansettingspb.ReadPlanSettingsResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan settings ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan settings: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	planSettings := &plansettingspb.PlanSettings{}
	if err := protojson.Unmarshal(resultJSON, planSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &plansettingspb.ReadPlanSettingsResponse{
		Data: []*plansettingspb.PlanSettings{planSettings},
	}, nil
}

// UpdatePlanSettings updates a plan settings using common PostgreSQL operations
func (r *PostgresPlanSettingsRepository) UpdatePlanSettings(ctx context.Context, req *plansettingspb.UpdatePlanSettingsRequest) (*plansettingspb.UpdatePlanSettingsResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan settings ID is required")
	}

	// Convert protobuf to map using protojson
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan settings: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	planSettings := &plansettingspb.PlanSettings{}
	if err := protojson.Unmarshal(resultJSON, planSettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &plansettingspb.UpdatePlanSettingsResponse{
		Data: []*plansettingspb.PlanSettings{planSettings},
	}, nil
}

// DeletePlanSettings deletes a plan settings using common PostgreSQL operations
func (r *PostgresPlanSettingsRepository) DeletePlanSettings(ctx context.Context, req *plansettingspb.DeletePlanSettingsRequest) (*plansettingspb.DeletePlanSettingsResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan settings ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete plan settings: %w", err)
	}

	return &plansettingspb.DeletePlanSettingsResponse{
		Success: true,
	}, nil
}

// ListPlanSettings lists plan settings using common PostgreSQL operations
func (r *PostgresPlanSettingsRepository) ListPlanSettings(ctx context.Context, req *plansettingspb.ListPlanSettingsRequest) (*plansettingspb.ListPlanSettingsResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan settings: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var planSettingsList []*plansettingspb.PlanSettings
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		planSettings := &plansettingspb.PlanSettings{}
		if err := protojson.Unmarshal(resultJSON, planSettings); err != nil {
			// Log error and continue with next item
			continue
		}
		planSettingsList = append(planSettingsList, planSettings)
	}

	return &plansettingspb.ListPlanSettingsResponse{
		Data: planSettingsList,
	}, nil
}

// ListPlanSettingsByPlan lists plan settings by plan using common PostgreSQL operations
func (r *PostgresPlanSettingsRepository) ListPlanSettingsByPlan(ctx context.Context, req *plansettingspb.ListPlanSettingsByPlanRequest) (*plansettingspb.ListPlanSettingsByPlanResponse, error) {
	if req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	// Note: Using Query method instead of List because we need to filter by plan_id
	// and ListPlanSettingsByPlanRequest doesn't have filter/sort/pagination fields
	query := operations.NewQueryBuilder().
		Where("plan_id", "==", req.PlanId)

	results, err := r.dbOps.Query(ctx, r.tableName, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan settings by plan: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var planSettingsList []*plansettingspb.PlanSettings
	for _, result := range results {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		planSettings := &plansettingspb.PlanSettings{}
		if err := protojson.Unmarshal(resultJSON, planSettings); err != nil {
			// Log error and continue with next item
			continue
		}
		planSettingsList = append(planSettingsList, planSettings)
	}

	return &plansettingspb.ListPlanSettingsByPlanResponse{
		Data: planSettingsList,
	}, nil
}

// GetPlanSettingsListPageData retrieves paginated plan settings list data with CTE
func (r *PostgresPlanSettingsRepository) GetPlanSettingsListPageData(ctx context.Context, req *plansettingspb.ListPlanSettingsRequest) (*plansettingspb.ListPlanSettingsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	searchPattern := ""
	limit, offset := int32(50), int32(0)
	sortField, sortOrder := "date_created", "DESC"

	query := `SELECT id, plan_id, name, description, active, date_created, date_modified FROM plan_settings WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR plan_id ILIKE $1) ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var planSettingsList []*plansettingspb.PlanSettings
	for rows.Next() {
		var id, planId, name, description string
		var active bool
		var dateCreated, dateModified time.Time
		if err := rows.Scan(&id, &planId, &name, &description, &active, &dateCreated, &dateModified); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		// totalCount assignment removed - not needed for current protobuf schema
planSettings := &plansettingspb.PlanSettings{Id: id, PlanId: planId, Name: name, Description: description, Active: active}
		if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		planSettings.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		planSettings.DateCreatedString = &dcStr
	}
		if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		planSettings.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		planSettings.DateModifiedString = &dmStr
	}
		planSettingsList = append(planSettingsList, planSettings)
	}
	// totalPages calculation removed - not needed for current protobuf schema
	return &plansettingspb.ListPlanSettingsResponse{Data: planSettingsList, Success: true}, nil
	// Note: Pagination removed - not available in current protobuf schema
}

// GetPlanSettingsItemPageData retrieves plan settings item page data
// func (r *PostgresPlanSettingsRepository) GetPlanSettingsItemPageData(ctx context.Context, req *plansettingspb.GetPlanSettingsItemPageDataRequest) (*plansettingspb.GetPlanSettingsItemPageDataResponse, error) {
// Note: Function commented out - GetPlanSettingsItemPageDataRequest/Response not defined in protobuf schema
// TODO: Implement when protobuf schema is updated with enhanced pagination functionality

// NewPlanSettingsRepository creates a new PostgreSQL plan_settings repository (old-style constructor)
func NewPlanSettingsRepository(db *sql.DB, tableName string) plansettingspb.PlanSettingsDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresPlanSettingsRepository(dbOps, tableName)
}
