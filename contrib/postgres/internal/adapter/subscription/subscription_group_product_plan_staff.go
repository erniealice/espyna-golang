//go:build postgresql

package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_group_product_plan_staff"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.SubscriptionGroupProductPlanStaff, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres subscription_group_product_plan_staff repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresSubscriptionGroupProductPlanStaffRepository(dbOps, tableName), nil
	})
}

type PostgresSubscriptionGroupProductPlanStaffRepository struct {
	pb.UnimplementedSubscriptionGroupProductPlanStaffDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

func NewPostgresSubscriptionGroupProductPlanStaffRepository(dbOps interfaces.DatabaseOperation, tableName string) pb.SubscriptionGroupProductPlanStaffDomainServiceServer {
	if tableName == "" {
		tableName = "subscription_group_product_plan_staff"
	}
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}
	return &PostgresSubscriptionGroupProductPlanStaffRepository{dbOps: dbOps, db: db, tableName: tableName}
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) CreateSubscriptionGroupProductPlanStaff(ctx context.Context, req *pb.CreateSubscriptionGroupProductPlanStaffRequest) (*pb.CreateSubscriptionGroupProductPlanStaffResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("subscription group product plan staff data is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription group product plan staff: %w", err)
	}
	item, err := subscriptionGroupProductPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.CreateSubscriptionGroupProductPlanStaffResponse{Data: []*pb.SubscriptionGroupProductPlanStaff{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) ReadSubscriptionGroupProductPlanStaff(ctx context.Context, req *pb.ReadSubscriptionGroupProductPlanStaffRequest) (*pb.ReadSubscriptionGroupProductPlanStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group product plan staff ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group product plan staff: %w", err)
	}
	item, err := subscriptionGroupProductPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.ReadSubscriptionGroupProductPlanStaffResponse{Data: []*pb.SubscriptionGroupProductPlanStaff{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) UpdateSubscriptionGroupProductPlanStaff(ctx context.Context, req *pb.UpdateSubscriptionGroupProductPlanStaffRequest) (*pb.UpdateSubscriptionGroupProductPlanStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group product plan staff ID is required")
	}
	data, err := protoGradingToMap(req.Data)
	if err != nil {
		return nil, err
	}
	// LIVE-FOUND (checkpoint C2, 2026-07-24): job_template_phase_id is a
	// proto3 `optional string` FK column (NULLable — plan.md §2.5 "NULL = all
	// phases"). protojson's explicit-presence rule means a caller that
	// deliberately CLEARS the phase (a non-nil pointer to "") marshals as the
	// JSON string `""`, never JSON `null` — proto3 has no wire/JSON
	// representation of "explicitly null" for a scalar field, only "present"
	// (any value, including "") or "absent" (omitted). PostgresOperations.
	// Update() then walks the map's PRESENT keys verbatim into the SET
	// clause, so an omitted key correctly leaves the persisted value
	// untouched (right for a partial update that never mentioned the field),
	// but a present `""` writes the LITERAL empty string — which the
	// fk_subscription_group_product_plan_staff_job_template_phase_id FK
	// constraint rejects (empty string is not NULL and matches no
	// job_template_phase row). Confirmed live: the S3 "clear a row's phase
	// back to All Phases" edit path (UpdateSubscriptionGroupProductPlanStaffUseCase.
	// effectiveEdge's firstSetPtr, and the create-side reactivate-on-legacy-
	// collision path in class_edge_v2.go / create_subscription_group_product_
	// plan_staff.go) both hit this identically — a pre-existing gap, not
	// introduced by either. The one place that CAN distinguish "explicitly
	// cleared" from "untouched" is here, where the caller's ORIGINAL pointer
	// is still in scope (unlike inside the generic map-only Update()): a
	// non-nil pointer to "" means "write NULL", so translate it to a real Go
	// nil in the map before handing off — the only representation
	// PostgresOperations.Update()'s serializeValue() will pass through as SQL
	// NULL instead of the literal empty string.
	//
	// ⚠ CORRECTED 2026-07-25. The first version of this fix wrote
	//     data["job_template_phase_id"] = nil
	// which was NON-DETERMINISTIC, not merely incomplete. protoGradingToMap
	// emits protojson's CAMEL spelling ("jobTemplatePhaseId": ""), so adding the
	// snake spelling left BOTH keys in the payload; PostgresOperations.Update's
	// normalizeKeys() then canonicalizes both to the same "job_template_phase_id"
	// and Go's randomized map iteration order decides which value survives. The
	// clear therefore wrote SQL NULL only on some runs and the literal "" on
	// others — the FK rejecting the "" case. Caught 2026-07-25 by
	// TestUpdateSubscriptionGroupProductPlanStaff_PhaseNullTranslation, which
	// asserts exactly one key canonicalizes to the column.
	//
	// The fix: remove EVERY spelling that canonicalizes to the column, then set
	// exactly one. postgresCore.CamelToSnake is the same algorithm normalizeKeys
	// uses, exported for precisely this purpose, so the two cannot drift.
	if req.Data.JobTemplatePhaseId != nil && req.Data.GetJobTemplatePhaseId() == "" {
		const phaseCol = "job_template_phase_id"
		for k := range data {
			if postgresCore.CamelToSnake(k) == phaseCol {
				delete(data, k)
			}
		}
		data[phaseCol] = nil
	}
	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription group product plan staff: %w", err)
	}
	item, err := subscriptionGroupProductPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateSubscriptionGroupProductPlanStaffResponse{Data: []*pb.SubscriptionGroupProductPlanStaff{item}, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) DeleteSubscriptionGroupProductPlanStaff(ctx context.Context, req *pb.DeleteSubscriptionGroupProductPlanStaffRequest) (*pb.DeleteSubscriptionGroupProductPlanStaffResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("subscription group product plan staff ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete subscription group product plan staff: %w", err)
	}
	return &pb.DeleteSubscriptionGroupProductPlanStaffResponse{Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) ListSubscriptionGroupProductPlanStaffs(ctx context.Context, req *pb.ListSubscriptionGroupProductPlanStaffsRequest) (*pb.ListSubscriptionGroupProductPlanStaffsResponse, error) {
	var params *interfaces.ListParams
	if req.GetFilters() != nil {
		params = &interfaces.ListParams{Filters: req.GetFilters()}
	}
	items, _, err := r.listAll(ctx, params)
	if err != nil {
		return nil, err
	}
	return &pb.ListSubscriptionGroupProductPlanStaffsResponse{Data: items, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) GetSubscriptionGroupProductPlanStaffListPageData(ctx context.Context, req *pb.GetSubscriptionGroupProductPlanStaffListPageDataRequest) (*pb.GetSubscriptionGroupProductPlanStaffListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}
	params := &interfaces.ListParams{
		Search:     req.GetSearch(),
		Filters:    req.GetFilters(),
		Sort:       req.GetSort(),
		Pagination: req.GetPagination(),
	}
	items, pagination, err := r.listAll(ctx, params)
	if err != nil {
		return nil, err
	}
	return &pb.GetSubscriptionGroupProductPlanStaffListPageDataResponse{SubscriptionGroupProductPlanStaffList: items, Pagination: pagination, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) GetSubscriptionGroupProductPlanStaffItemPageData(ctx context.Context, req *pb.GetSubscriptionGroupProductPlanStaffItemPageDataRequest) (*pb.GetSubscriptionGroupProductPlanStaffItemPageDataResponse, error) {
	if req == nil || req.SubscriptionGroupProductPlanStaffId == "" {
		return nil, fmt.Errorf("subscription group product plan staff ID required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.SubscriptionGroupProductPlanStaffId)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription group product plan staff: %w", err)
	}
	item, err := subscriptionGroupProductPlanStaffFromResult(result)
	if err != nil {
		return nil, err
	}
	return &pb.GetSubscriptionGroupProductPlanStaffItemPageDataResponse{SubscriptionGroupProductPlanStaff: item, Success: true}, nil
}

func (r *PostgresSubscriptionGroupProductPlanStaffRepository) listAll(ctx context.Context, params *interfaces.ListParams) ([]*pb.SubscriptionGroupProductPlanStaff, *commonpb.PaginationResponse, error) {
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list subscription group product plan staffs: %w", err)
	}
	var items []*pb.SubscriptionGroupProductPlanStaff
	for i, row := range listResult.Data {
		rj, err := json.Marshal(row)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal subscription_group_product_plan_staff row %d: %w", i, err)
		}
		item := &pb.SubscriptionGroupProductPlanStaff{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal subscription_group_product_plan_staff row %d to proto: %w", i, err)
		}
		items = append(items, item)
	}
	return items, listResult.Pagination, nil
}

func subscriptionGroupProductPlanStaffFromResult(result any) (*pb.SubscriptionGroupProductPlanStaff, error) {
	rj, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	item := &pb.SubscriptionGroupProductPlanStaff{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rj, item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to proto: %w", err)
	}
	return item, nil
}
