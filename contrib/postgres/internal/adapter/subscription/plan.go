//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	planlocationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_location"
	"google.golang.org/protobuf/encoding/protojson"
)

// PostgresPlanRepository implements plan CRUD operations using PostgreSQL
type PostgresPlanRepository struct {
	planpb.UnimplementedPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.Plan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresPlanRepository(dbOps, tableName), nil
	})
}

// NewPostgresPlanRepository creates a new PostgreSQL plan repository
func NewPostgresPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) planpb.PlanDomainServiceServer {
	if tableName == "" {
		tableName = "plan" // default fallback
	}
	return &PostgresPlanRepository{
		dbOps:     dbOps,
		tableName: tableName,
	}
}

// CreatePlan creates a new plan using common PostgreSQL operations
func (r *PostgresPlanRepository) CreatePlan(ctx context.Context, req *planpb.CreatePlanRequest) (*planpb.CreatePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan data is required")
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

	// Empty optional FK ("" from a blank picker) must arrive at postgres as
	// SQL NULL, otherwise plan_client_id_fkey rejects the insert. Mirrors the
	// UpdatePlan normalisation below.
	if v, ok := data["client_id"].(string); ok && v == "" {
		data["client_id"] = nil
	}

	// Create document using common operations
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	plan := &planpb.Plan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &planpb.CreatePlanResponse{
		Data: []*planpb.Plan{plan},
	}, nil
}

// ReadPlan retrieves a plan using common PostgreSQL operations
func (r *PostgresPlanRepository) ReadPlan(ctx context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, *req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	plan := &planpb.Plan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &planpb.ReadPlanResponse{
		Data: []*planpb.Plan{plan},
	}, nil
}

// UpdatePlan updates a plan using common PostgreSQL operations
func (r *PostgresPlanRepository) UpdatePlan(ctx context.Context, req *planpb.UpdatePlanRequest) (*planpb.UpdatePlanResponse, error) {
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
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

	// Always include active flag — proto3 omits bool=false during JSON marshal,
	// which would silently skip deactivation via the form toggle.
	data["active"] = req.Data.GetActive()

	// Always include client_id so master ↔ client_id transitions actually
	// reach the column. protojson omits nil optional strings (would silently
	// keep the old FK), and serializes &"" as "" (would trip
	// plan_client_id_fkey since '' is not a valid FK). Force-write: empty
	// or nil → SQL NULL; non-empty → the value.
	if cid := req.Data.GetClientId(); cid == "" {
		data["client_id"] = nil
	} else {
		data["client_id"] = cid
	}

	// Update document using common operations
	result, err := r.dbOps.Update(ctx, r.tableName, *req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	plan := &planpb.Plan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &planpb.UpdatePlanResponse{
		Data: []*planpb.Plan{plan},
	}, nil
}

// DeletePlan deletes a plan using common PostgreSQL operations
func (r *PostgresPlanRepository) DeletePlan(ctx context.Context, req *planpb.DeletePlanRequest) (*planpb.DeletePlanResponse, error) {
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	// Hard delete — catalog entities rely on FK RESTRICT to block deletion
	// when historical references exist (subscription, product_plan, price_plan, etc.).
	err := r.dbOps.HardDelete(ctx, r.tableName, *req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete plan: %w", err)
	}

	return &planpb.DeletePlanResponse{
		Success: true,
	}, nil
}

var planSortableSQLCols = []string{
	"id", "active", "name", "description", "client_id",
	"billing_kind", "date_created", "date_modified",
}

var planSortSpec = espynahttp.SortSpec{AllowedCols: planSortableSQLCols}

// ListPlans lists plans using common PostgreSQL operations
func (r *PostgresPlanRepository) ListPlans(ctx context.Context, req *planpb.ListPlansRequest) (*planpb.ListPlansResponse, error) {
	if err := espynahttp.ValidateSortColumns(planSortSpec, req.GetSort(), "plan"); err != nil {
		return nil, err
	}

	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Search = req.Search
		params.Sort = req.Sort
		params.Pagination = req.Pagination
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var plans []*planpb.Plan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(result))
		if err != nil {
			// Log error and continue with next item
			continue
		}

		plan := &planpb.Plan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, plan); err != nil {
			// Log error and continue with next item
			continue
		}
		plans = append(plans, plan)
	}

	return &planpb.ListPlansResponse{
		Data: plans,
	}, nil
}

// GetPlanListPageData retrieves a paginated, filtered, sorted, and searchable
// list of plans with adjacent plan_location relationships. Canonicalized to
// delegate field-agnostic plan loading to the generic List path (dbOps.List +
// protojson DiscardUnknown round-trip), so new proto fields surface here
// automatically. Adjacent denorm: plan_locations are loaded in a single
// `plan_id IN (...)` lookup followed by one `location_id IN (...)` lookup —
// never per-row N+1.
func (r *PostgresPlanRepository) GetPlanListPageData(ctx context.Context, req *planpb.GetPlanListPageDataRequest) (*planpb.GetPlanListPageDataResponse, error) {
	if req == nil {
		req = &planpb.GetPlanListPageDataRequest{}
	}

	// Preserve the page-data caller intent: only active plans on the list page.
	filters := mergeActiveFilter(req.GetFilters(), true)
	params := &interfaces.ListParams{
		Search:     req.GetSearch(),
		Filters:    filters,
		Sort:       req.GetSort(),
		Pagination: req.GetPagination(),
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}

	// Field-agnostic round-trip: every column on `plan` arrives as a Plan field
	// via protojson DiscardUnknown — drift-proof for new proto fields.
	plans := make([]*planpb.Plan, 0, len(listResult.Data))
	planIDs := make([]string, 0, len(listResult.Data))
	for _, row := range listResult.Data {
		plan, err := planFromMap(row)
		if err != nil || plan == nil {
			continue
		}
		plans = append(plans, plan)
		if plan.GetId() != "" {
			planIDs = append(planIDs, plan.GetId())
		}
	}

	// Adjacent denorm: load plan_locations for the page in one shot, attach.
	if len(planIDs) > 0 {
		byPlan, err := r.loadPlanLocationsByPlanIDs(ctx, planIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to load plan_locations for page: %w", err)
		}
		for _, p := range plans {
			if locs, ok := byPlan[p.GetId()]; ok {
				p.PlanLocations = locs
			}
		}
	}

	return &planpb.GetPlanListPageDataResponse{
		Success:    true,
		PlanList:   plans,
		Pagination: listResult.Pagination,
	}, nil
}

// GetPlanItemPageData retrieves a single plan with adjacent plan_location
// relationships expanded. Canonicalized to delegate the plan load to ReadPlan
// (dbOps.Read + protojson DiscardUnknown), so every Plan proto field surfaces
// here automatically — drift-proof. Adjacent denorm: plan_locations + nested
// Location are loaded with two targeted lookups (plan_location filtered by
// plan_id, then location filtered by id IN (...)).
func (r *PostgresPlanRepository) GetPlanItemPageData(ctx context.Context, req *planpb.GetPlanItemPageDataRequest) (*planpb.GetPlanItemPageDataResponse, error) {
	if req == nil || req.PlanId == "" {
		return nil, fmt.Errorf("plan ID is required")
	}

	rr, err := r.ReadPlan(ctx, &planpb.ReadPlanRequest{Data: &planpb.Plan{Id: &req.PlanId}})
	if err != nil {
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}
	if len(rr.GetData()) == 0 {
		return nil, fmt.Errorf("plan not found with ID: %s", req.PlanId)
	}
	plan := rr.GetData()[0]

	// Adjacent denorm: plan_locations (active) with nested Location (active),
	// loaded in two targeted queries — no per-row reads.
	byPlan, err := r.loadPlanLocationsByPlanIDs(ctx, []string{req.PlanId})
	if err != nil {
		return nil, fmt.Errorf("failed to load plan_locations: %w", err)
	}
	if locs, ok := byPlan[req.PlanId]; ok {
		plan.PlanLocations = locs
	}

	return &planpb.GetPlanItemPageDataResponse{
		Success: true,
		Plan:    plan,
	}, nil
}

// planFromMap converts a `plan` row (column->value map) into a Plan proto via
// the canonical protojson DiscardUnknown round-trip. Field-agnostic — any new
// column with a matching proto field arrives automatically.
func planFromMap(row map[string]any) (*planpb.Plan, error) {
	resultJSON, err := json.Marshal(postgresCore.DenormalizeKeys(row))
	if err != nil {
		return nil, err
	}
	plan := &planpb.Plan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

// mergeActiveFilter returns a FilterRequest with `active = <value>` enforced.
// If the caller already supplied an explicit `active` filter, that wins
// (preserves caller intent — e.g. an admin toggle to show inactive rows).
func mergeActiveFilter(in *commonpb.FilterRequest, active bool) *commonpb.FilterRequest {
	out := &commonpb.FilterRequest{}
	if in != nil {
		out.Logic = in.GetLogic()
		out.Filters = append(out.Filters, in.GetFilters()...)
		for _, f := range in.GetFilters() {
			if f.GetField() == "active" && f.GetBooleanFilter() != nil {
				return out
			}
		}
	}
	out.Filters = append(out.Filters, &commonpb.TypedFilter{
		Field: "active",
		FilterType: &commonpb.TypedFilter_BooleanFilter{
			BooleanFilter: &commonpb.BooleanFilter{Value: active},
		},
	})
	return out
}

// loadPlanLocationsByPlanIDs returns a map keyed by plan_id of attached, active
// PlanLocation rows with their Location nested. Two queries total:
//
//  1. plan_location WHERE plan_id IN (ids) AND active = true
//  2. location WHERE id IN (location_ids) AND active = true
//
// Both pass through dbOps.List → protojson DiscardUnknown so the field set
// stays drift-proof. Within each plan_id, results are sorted by location.name
// ASC to match the prior CTE ordering.
func (r *PostgresPlanRepository) loadPlanLocationsByPlanIDs(ctx context.Context, planIDs []string) (map[string][]*planlocationpb.PlanLocation, error) {
	out := make(map[string][]*planlocationpb.PlanLocation)
	if len(planIDs) == 0 {
		return out, nil
	}

	planIDValues := make([]string, len(planIDs))
	copy(planIDValues, planIDs)

	plLinks, err := r.dbOps.List(ctx, "plan_location", &interfaces.ListParams{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "plan_id",
					FilterType: &commonpb.TypedFilter_ListFilter{
						ListFilter: &commonpb.ListFilter{
							Operator: commonpb.ListOperator_LIST_IN,
							Values:   planIDValues,
						},
					},
				},
				{
					Field: "active",
					FilterType: &commonpb.TypedFilter_BooleanFilter{
						BooleanFilter: &commonpb.BooleanFilter{Value: true},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list plan_location: %w", err)
	}
	if plLinks == nil || len(plLinks.Data) == 0 {
		return out, nil
	}

	links := make([]*planlocationpb.PlanLocation, 0, len(plLinks.Data))
	locIDSet := make(map[string]struct{})
	for _, row := range plLinks.Data {
		js, err := json.Marshal(postgresCore.DenormalizeKeys(row))
		if err != nil {
			continue
		}
		pl := &planlocationpb.PlanLocation{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, pl); err != nil {
			continue
		}
		links = append(links, pl)
		if pl.GetLocationId() != "" {
			locIDSet[pl.GetLocationId()] = struct{}{}
		}
	}

	locByID := make(map[string]*locationpb.Location, len(locIDSet))
	if len(locIDSet) > 0 {
		locIDs := make([]string, 0, len(locIDSet))
		for id := range locIDSet {
			locIDs = append(locIDs, id)
		}
		locResult, err := r.dbOps.List(ctx, "location", &interfaces.ListParams{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "id",
						FilterType: &commonpb.TypedFilter_ListFilter{
							ListFilter: &commonpb.ListFilter{
								Operator: commonpb.ListOperator_LIST_IN,
								Values:   locIDs,
							},
						},
					},
					{
						Field: "active",
						FilterType: &commonpb.TypedFilter_BooleanFilter{
							BooleanFilter: &commonpb.BooleanFilter{Value: true},
						},
					},
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list location: %w", err)
		}
		if locResult != nil {
			for _, row := range locResult.Data {
				js, err := json.Marshal(postgresCore.DenormalizeKeys(row))
				if err != nil {
					continue
				}
				loc := &locationpb.Location{}
				if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(js, loc); err != nil {
					continue
				}
				if loc.GetId() != "" {
					locByID[loc.GetId()] = loc
				}
			}
		}
	}

	for _, pl := range links {
		if loc, ok := locByID[pl.GetLocationId()]; ok {
			pl.Location = loc
		} else {
			// Skip links whose Location is missing or inactive (matches prior
			// `INNER JOIN ... AND l.active = true` semantics).
			continue
		}
		out[pl.GetPlanId()] = append(out[pl.GetPlanId()], pl)
	}

	// Stable order within each plan: location.name ASC, mirroring the prior
	// CTE's `ORDER BY l.name ASC`.
	for k := range out {
		sort.SliceStable(out[k], func(i, j int) bool {
			return strings.ToLower(out[k][i].GetLocation().GetName()) <
				strings.ToLower(out[k][j].GetLocation().GetName())
		})
	}
	return out, nil
}

// SearchPlansByName searches active plans by name using ILIKE
func (r *PostgresPlanRepository) SearchPlansByName(ctx context.Context, req *planpb.SearchPlansByNameRequest) (*planpb.SearchPlansByNameResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("search plans by name request is required")
	}

	limit := int32(20)
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}

	// A1: scope to the caller's workspace. This typeahead bypasses the
	// WorkspaceAwareOperations decorator (raw SQL via db.GetDB()) and enumerates
	// rows, so without this predicate it would leak other tenants' plan names. The
	// plan table carries its own workspace_id (verified against the baseline
	// schema), so scope directly. Empty wsID = service-to-service call → no scoping.
	wsID := identity.Must(ctx).WorkspaceID
	query := `
		SELECT id, name
		FROM plan
		WHERE active = true
			AND ($3::text = '' OR workspace_id = $3::text)
			AND ($1::text = '' OR name ILIKE $1)
		ORDER BY name ASC
		LIMIT $2
	`

	pattern := ""
	if req.Query != "" {
		pattern = "%" + req.Query + "%"
	}

	// Get DB connection from dbOps interface
	db, ok := r.dbOps.(interface{ GetDB() *sql.DB })
	if !ok {
		return nil, fmt.Errorf("database operations does not support raw SQL queries")
	}

	rows, err := db.GetDB().QueryContext(ctx, query, pattern, limit, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to search plans by name: %w", err)
	}
	defer rows.Close()

	var results []*planpb.SearchPlanResult
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("failed to scan search plan row: %w", err)
		}
		results = append(results, &planpb.SearchPlanResult{
			Id:    id,
			Label: name,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search plan rows: %w", err)
	}

	return &planpb.SearchPlansByNameResponse{
		Results: results,
		Success: true,
	}, nil
}

// NewPlanRepository creates a new PostgreSQL plan repository (old-style constructor)
func NewPlanRepository(db *sql.DB, tableName string) planpb.PlanDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresPlanRepository(dbOps, tableName)
}
