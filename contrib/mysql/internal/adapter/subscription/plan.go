//go:build mysql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	espynahttp "github.com/erniealice/espyna-golang/contrib/http"
	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	locationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/location"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	planlocationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_location"
	"google.golang.org/protobuf/encoding/protojson"
)

var planSortableSQLCols = []string{
	"id", "active", "name", "description", "client_id",
	"billing_kind", "date_created", "date_modified",
}

var planSortSpec = espynahttp.SortSpec{AllowedCols: planSortableSQLCols}

// MySQLPlanRepository implements plan CRUD operations using MySQL 8.0+.
type MySQLPlanRepository struct {
	planpb.UnimplementedPlanDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.Plan, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql plan repository requires *sql.DB, got %T", conn)
		}
		dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
		return NewMySQLPlanRepository(dbOps, tableName), nil
	})
}

// NewMySQLPlanRepository creates a new MySQL plan repository.
func NewMySQLPlanRepository(dbOps interfaces.DatabaseOperation, tableName string) planpb.PlanDomainServiceServer {
	if tableName == "" {
		tableName = "plan"
	}
	return &MySQLPlanRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLPlanRepository) CreatePlan(ctx context.Context, req *planpb.CreatePlanRequest) (*planpb.CreatePlanResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan data is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	if v, ok := data["client_id"].(string); ok && v == "" {
		data["client_id"] = nil
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	plan := &planpb.Plan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &planpb.CreatePlanResponse{Data: []*planpb.Plan{plan}}, nil
}

func (r *MySQLPlanRepository) ReadPlan(ctx context.Context, req *planpb.ReadPlanRequest) (*planpb.ReadPlanResponse, error) {
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, *req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	plan := &planpb.Plan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &planpb.ReadPlanResponse{Data: []*planpb.Plan{plan}}, nil
}

func (r *MySQLPlanRepository) UpdatePlan(ctx context.Context, req *planpb.UpdatePlanRequest) (*planpb.UpdatePlanResponse, error) {
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
	}
	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	data["active"] = req.Data.GetActive()
	if cid := req.Data.GetClientId(); cid == "" {
		data["client_id"] = nil
	} else {
		data["client_id"] = cid
	}
	result, err := r.dbOps.Update(ctx, r.tableName, *req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	plan := &planpb.Plan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &planpb.UpdatePlanResponse{Data: []*planpb.Plan{plan}}, nil
}

func (r *MySQLPlanRepository) DeletePlan(ctx context.Context, req *planpb.DeletePlanRequest) (*planpb.DeletePlanResponse, error) {
	if req.Data == nil || req.Data.Id == nil || *req.Data.Id == "" {
		return nil, fmt.Errorf("plan ID is required")
	}
	if err := r.dbOps.HardDelete(ctx, r.tableName, *req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete plan: %w", err)
	}
	return &planpb.DeletePlanResponse{Success: true}, nil
}

func (r *MySQLPlanRepository) ListPlans(ctx context.Context, req *planpb.ListPlansRequest) (*planpb.ListPlansResponse, error) {
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
	var plans []*planpb.Plan
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(result))
		if err != nil {
			continue
		}
		plan := &planpb.Plan{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, plan); err != nil {
			continue
		}
		plans = append(plans, plan)
	}
	return &planpb.ListPlansResponse{Data: plans}, nil
}

func (r *MySQLPlanRepository) GetPlanListPageData(ctx context.Context, req *planpb.GetPlanListPageDataRequest) (*planpb.GetPlanListPageDataResponse, error) {
	if req == nil {
		req = &planpb.GetPlanListPageDataRequest{}
	}
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

func (r *MySQLPlanRepository) GetPlanItemPageData(ctx context.Context, req *planpb.GetPlanItemPageDataRequest) (*planpb.GetPlanItemPageDataResponse, error) {
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
	byPlan, err := r.loadPlanLocationsByPlanIDs(ctx, []string{req.PlanId})
	if err != nil {
		return nil, fmt.Errorf("failed to load plan_locations: %w", err)
	}
	if locs, ok := byPlan[req.PlanId]; ok {
		plan.PlanLocations = locs
	}
	return &planpb.GetPlanItemPageDataResponse{Success: true, Plan: plan}, nil
}

// planFromMap converts a plan row to a Plan proto (dialect-agnostic).
func planFromMap(row map[string]any) (*planpb.Plan, error) {
	resultJSON, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
	if err != nil {
		return nil, err
	}
	plan := &planpb.Plan{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

// mergeActiveFilter returns a FilterRequest with active=<value> enforced.
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

func (r *MySQLPlanRepository) loadPlanLocationsByPlanIDs(ctx context.Context, planIDs []string) (map[string][]*planlocationpb.PlanLocation, error) {
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
		js, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
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
				js, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
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
		loc, ok := locByID[pl.GetLocationId()]
		if !ok {
			continue
		}
		pl.Location = loc
		out[pl.GetPlanId()] = append(out[pl.GetPlanId()], pl)
	}
	for k := range out {
		sort.SliceStable(out[k], func(i, j int) bool {
			return strings.ToLower(out[k][i].GetLocation().GetName()) <
				strings.ToLower(out[k][j].GetLocation().GetName())
		})
	}
	return out, nil
}

// SearchPlansByName searches active plans by name using LIKE.
//
// Dialect: $1/$2 → ?, ILIKE → LIKE, WHERE workspace_id = ? from WorkspaceAwareOperations.
func (r *MySQLPlanRepository) SearchPlansByName(ctx context.Context, req *planpb.SearchPlansByNameRequest) (*planpb.SearchPlansByNameResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("search plans by name request is required")
	}
	limit := int32(20)
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}
	pattern := ""
	if req.Query != "" {
		pattern = "%" + req.Query + "%"
	}
	// Dialect: ILIKE → LIKE, $1/$2 → ?, active = true → active = 1.
	query := `
		SELECT id, name
		FROM plan
		WHERE active = 1
			AND (? = '' OR name LIKE ?)
		ORDER BY name ASC
		LIMIT ?
	`
	exec := r.dbOps.(executorProvider).GetExecutor(ctx)
	rows, err := exec.QueryContext(ctx, query, pattern, pattern, limit)
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
		results = append(results, &planpb.SearchPlanResult{Id: id, Label: name})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search plan rows: %w", err)
	}
	return &planpb.SearchPlansByNameResponse{Results: results, Success: true}, nil
}

// NewPlanRepository creates a new MySQL plan repository (old-style constructor).
func NewPlanRepository(db *sql.DB, tableName string) planpb.PlanDomainServiceServer {
	dbOps := mysqlCore.NewWorkspaceAwareOperations(db)
	return NewMySQLPlanRepository(dbOps, tableName)
}
