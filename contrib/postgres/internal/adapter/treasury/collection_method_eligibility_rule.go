//go:build postgresql

package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"

	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	eligibilityrulepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection_method_eligibility_rule"
)

// Treasury-domain-rebuild Stage 2: the postgres collection_method_eligibility_rule
// repository. The table (migration 20260527010000) carries scalar / enum columns
// plus two JSONB array columns (applicable_product_ids, applicable_category_ids).
// Enum fields persist as their string name and repeated string fields as JSON
// arrays, so the generic protojson round-trip (protojson -> map -> dbOps; result
// -> protojson) carries every field faithfully with no per-column scan. Mirrors
// collection_method.go exactly.
func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.CollectionMethodEligibilityRule, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres collection_method_eligibility_rule repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewWorkspaceAwareOperations(db)
		return NewPostgresCollectionMethodEligibilityRuleRepository(dbOps, tableName), nil
	})
}

// PostgresCollectionMethodEligibilityRuleRepository implements collection_method_eligibility_rule CRUD using PostgreSQL.
type PostgresCollectionMethodEligibilityRuleRepository struct {
	eligibilityrulepb.UnimplementedCollectionMethodEligibilityRuleDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresCollectionMethodEligibilityRuleRepository creates a new PostgreSQL collection_method_eligibility_rule repository.
func NewPostgresCollectionMethodEligibilityRuleRepository(dbOps interfaces.DatabaseOperation, tableName string) eligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer {
	if tableName == "" {
		tableName = "collection_method_eligibility_rule"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCollectionMethodEligibilityRuleRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func collectionMethodEligibilityRuleToMap(rule *eligibilityrulepb.CollectionMethodEligibilityRule) (map[string]any, error) {
	jsonData, err := protojson.Marshal(rule)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_method_eligibility_rule protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collection_method_eligibility_rule JSON to map: %w", err)
	}
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")
	return data, nil
}

func mapToCollectionMethodEligibilityRule(result map[string]any) (*eligibilityrulepb.CollectionMethodEligibilityRule, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection_method_eligibility_rule result to JSON: %w", err)
	}
	rule := &eligibilityrulepb.CollectionMethodEligibilityRule{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, rule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to collection_method_eligibility_rule protobuf: %w", err)
	}
	return rule, nil
}

// CreateCollectionMethodEligibilityRule creates a new collection_method_eligibility_rule record.
func (r *PostgresCollectionMethodEligibilityRuleRepository) CreateCollectionMethodEligibilityRule(ctx context.Context, req *eligibilityrulepb.CreateCollectionMethodEligibilityRuleRequest) (*eligibilityrulepb.CreateCollectionMethodEligibilityRuleResponse, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("collection_method_eligibility_rule data is required")
	}

	data, err := collectionMethodEligibilityRuleToMap(req.Data)
	if err != nil {
		return nil, err
	}

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection_method_eligibility_rule: %w", err)
	}

	rule, err := mapToCollectionMethodEligibilityRule(result)
	if err != nil {
		return nil, err
	}

	return &eligibilityrulepb.CreateCollectionMethodEligibilityRuleResponse{
		Success: true,
		Data:    []*eligibilityrulepb.CollectionMethodEligibilityRule{rule},
	}, nil
}

// ReadCollectionMethodEligibilityRule retrieves a collection_method_eligibility_rule record by ID.
func (r *PostgresCollectionMethodEligibilityRuleRepository) ReadCollectionMethodEligibilityRule(ctx context.Context, req *eligibilityrulepb.ReadCollectionMethodEligibilityRuleRequest) (*eligibilityrulepb.ReadCollectionMethodEligibilityRuleResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method_eligibility_rule ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_method_eligibility_rule: %w", err)
	}

	rule, err := mapToCollectionMethodEligibilityRule(result)
	if err != nil {
		return nil, err
	}

	return &eligibilityrulepb.ReadCollectionMethodEligibilityRuleResponse{
		Success: true,
		Data:    []*eligibilityrulepb.CollectionMethodEligibilityRule{rule},
	}, nil
}

// UpdateCollectionMethodEligibilityRule updates a collection_method_eligibility_rule record.
func (r *PostgresCollectionMethodEligibilityRuleRepository) UpdateCollectionMethodEligibilityRule(ctx context.Context, req *eligibilityrulepb.UpdateCollectionMethodEligibilityRuleRequest) (*eligibilityrulepb.UpdateCollectionMethodEligibilityRuleResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method_eligibility_rule ID is required")
	}

	data, err := collectionMethodEligibilityRuleToMap(req.Data)
	if err != nil {
		return nil, err
	}

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection_method_eligibility_rule: %w", err)
	}

	rule, err := mapToCollectionMethodEligibilityRule(result)
	if err != nil {
		return nil, err
	}

	return &eligibilityrulepb.UpdateCollectionMethodEligibilityRuleResponse{
		Success: true,
		Data:    []*eligibilityrulepb.CollectionMethodEligibilityRule{rule},
	}, nil
}

// DeleteCollectionMethodEligibilityRule deletes a collection_method_eligibility_rule record (soft delete).
func (r *PostgresCollectionMethodEligibilityRuleRepository) DeleteCollectionMethodEligibilityRule(ctx context.Context, req *eligibilityrulepb.DeleteCollectionMethodEligibilityRuleRequest) (*eligibilityrulepb.DeleteCollectionMethodEligibilityRuleResponse, error) {
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection_method_eligibility_rule ID is required")
	}

	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete collection_method_eligibility_rule: %w", err)
	}

	return &eligibilityrulepb.DeleteCollectionMethodEligibilityRuleResponse{
		Success: true,
	}, nil
}

// ListCollectionMethodEligibilityRules lists collection_method_eligibility_rule records with optional filters.
func (r *PostgresCollectionMethodEligibilityRuleRepository) ListCollectionMethodEligibilityRules(ctx context.Context, req *eligibilityrulepb.ListCollectionMethodEligibilityRulesRequest) (*eligibilityrulepb.ListCollectionMethodEligibilityRulesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && (req.Filters != nil || req.Pagination != nil) {
		params = &interfaces.ListParams{Filters: req.GetFilters(), Pagination: req.GetPagination()}
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collection_method_eligibility_rules: %w", err)
	}

	rules := make([]*eligibilityrulepb.CollectionMethodEligibilityRule, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		rule, err := mapToCollectionMethodEligibilityRule(result)
		if err != nil {
			log.Printf("WARN: collection_method_eligibility_rule list row decode: %v", err)
			continue
		}
		rules = append(rules, rule)
	}

	return &eligibilityrulepb.ListCollectionMethodEligibilityRulesResponse{
		Success: true,
		Data:    rules,
	}, nil
}

// GetCollectionMethodEligibilityRuleListPageData lists collection_method_eligibility_rules with pagination metadata.
func (r *PostgresCollectionMethodEligibilityRuleRepository) GetCollectionMethodEligibilityRuleListPageData(ctx context.Context, req *eligibilityrulepb.GetCollectionMethodEligibilityRuleListPageDataRequest) (*eligibilityrulepb.GetCollectionMethodEligibilityRuleListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection_method_eligibility_rule list page data request is required")
	}

	params := &interfaces.ListParams{
		Filters:    req.GetFilters(),
		Pagination: req.GetPagination(),
	}

	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection_method_eligibility_rule list page data: %w", err)
	}

	rules := make([]*eligibilityrulepb.CollectionMethodEligibilityRule, 0, len(listResult.Data))
	for _, result := range listResult.Data {
		rule, err := mapToCollectionMethodEligibilityRule(result)
		if err != nil {
			log.Printf("WARN: collection_method_eligibility_rule page-data row decode: %v", err)
			continue
		}
		rules = append(rules, rule)
	}

	resp := &eligibilityrulepb.GetCollectionMethodEligibilityRuleListPageDataResponse{
		CollectionMethodEligibilityRuleList: rules,
		Success:                             true,
	}
	if listResult.Pagination != nil {
		resp.Pagination = listResult.Pagination
	} else {
		total := listResult.Total
		resp.Pagination = &commonpb.PaginationResponse{TotalItems: total}
	}
	return resp, nil
}

// GetCollectionMethodEligibilityRuleItemPageData retrieves a single collection_method_eligibility_rule by ID.
func (r *PostgresCollectionMethodEligibilityRuleRepository) GetCollectionMethodEligibilityRuleItemPageData(ctx context.Context, req *eligibilityrulepb.GetCollectionMethodEligibilityRuleItemPageDataRequest) (*eligibilityrulepb.GetCollectionMethodEligibilityRuleItemPageDataResponse, error) {
	if req == nil || req.CollectionMethodEligibilityRuleId == "" {
		return nil, fmt.Errorf("collection_method_eligibility_rule ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.CollectionMethodEligibilityRuleId)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection_method_eligibility_rule item page data: %w", err)
	}

	rule, err := mapToCollectionMethodEligibilityRule(result)
	if err != nil {
		return nil, err
	}

	return &eligibilityrulepb.GetCollectionMethodEligibilityRuleItemPageDataResponse{
		CollectionMethodEligibilityRule: rule,
		Success:                         true,
	}, nil
}

// NewCollectionMethodEligibilityRuleRepository creates a new PostgreSQL collection_method_eligibility_rule repository (old-style constructor).
func NewCollectionMethodEligibilityRuleRepository(db *sql.DB, tableName string) eligibilityrulepb.CollectionMethodEligibilityRuleDomainServiceServer {
	dbOps := postgresCore.NewWorkspaceAwareOperations(db)
	return NewPostgresCollectionMethodEligibilityRuleRepository(dbOps, tableName)
}
