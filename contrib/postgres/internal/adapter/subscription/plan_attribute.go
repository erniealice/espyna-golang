
package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	planattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.PlanAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres plan_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresPlanAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresPlanAttributeRepository implements plan attribute CRUD operations using PostgreSQL
type PostgresPlanAttributeRepository struct {
	planattributepb.UnimplementedPlanAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresPlanAttributeRepository creates a new PostgreSQL plan attribute repository
func NewPostgresPlanAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) planattributepb.PlanAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "plan_attribute"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresPlanAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreatePlanAttribute creates a new plan attribute using common PostgreSQL operations
func (r *PostgresPlanAttributeRepository) CreatePlanAttribute(ctx context.Context, req *planattributepb.CreatePlanAttributeRequest) (*planattributepb.CreatePlanAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("plan attribute data is required")
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
		return nil, fmt.Errorf("failed to create plan attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	planAttribute := &planattributepb.PlanAttribute{}
	if err := protojson.Unmarshal(resultJSON, planAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &planattributepb.CreatePlanAttributeResponse{
		Data: []*planattributepb.PlanAttribute{planAttribute},
	}, nil
}

// ReadPlanAttribute retrieves a plan attribute using common PostgreSQL operations
func (r *PostgresPlanAttributeRepository) ReadPlanAttribute(ctx context.Context, req *planattributepb.ReadPlanAttributeRequest) (*planattributepb.ReadPlanAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan attribute ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	planAttribute := &planattributepb.PlanAttribute{}
	if err := protojson.Unmarshal(resultJSON, planAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &planattributepb.ReadPlanAttributeResponse{
		Data: []*planattributepb.PlanAttribute{planAttribute},
	}, nil
}

// UpdatePlanAttribute updates a plan attribute using common PostgreSQL operations
func (r *PostgresPlanAttributeRepository) UpdatePlanAttribute(ctx context.Context, req *planattributepb.UpdatePlanAttributeRequest) (*planattributepb.UpdatePlanAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan attribute ID is required")
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
		return nil, fmt.Errorf("failed to update plan attribute: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	planAttribute := &planattributepb.PlanAttribute{}
	if err := protojson.Unmarshal(resultJSON, planAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &planattributepb.UpdatePlanAttributeResponse{
		Data: []*planattributepb.PlanAttribute{planAttribute},
	}, nil
}

// DeletePlanAttribute deletes a plan attribute using common PostgreSQL operations
func (r *PostgresPlanAttributeRepository) DeletePlanAttribute(ctx context.Context, req *planattributepb.DeletePlanAttributeRequest) (*planattributepb.DeletePlanAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("plan attribute ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete plan attribute: %w", err)
	}

	return &planattributepb.DeletePlanAttributeResponse{
		Success: true,
	}, nil
}

// ListPlanAttributes lists plan attributes using common PostgreSQL operations
func (r *PostgresPlanAttributeRepository) ListPlanAttributes(ctx context.Context, req *planattributepb.ListPlanAttributesRequest) (*planattributepb.ListPlanAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list plan attributes: %w", err)
	}

	var planAttributes []*planattributepb.PlanAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}

		planAttribute := &planattributepb.PlanAttribute{}
		if err := protojson.Unmarshal(resultJSON, planAttribute); err != nil {
			continue
		}
		planAttributes = append(planAttributes, planAttribute)
	}

	return &planattributepb.ListPlanAttributesResponse{
		Data: planAttributes,
	}, nil
}

// GetPlanAttributeListPageData retrieves paginated plan attribute list data with CTE
func (r *PostgresPlanAttributeRepository) GetPlanAttributeListPageData(ctx context.Context, req *planattributepb.GetPlanAttributeListPageDataRequest) (*planattributepb.GetPlanAttributeListPageDataResponse, error) {
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
	sortField, sortOrder := "date_created", "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `WITH enriched AS (SELECT id, plan_id, attribute_id, value, active, date_created, date_modified FROM plan_attribute WHERE active = true AND ($1::text IS NULL OR $1::text = '' OR value ILIKE $1)), counted AS (SELECT COUNT(*) as total FROM enriched) SELECT e.*, c.total FROM enriched e, counted c ORDER BY ` + sortField + ` ` + sortOrder + ` LIMIT $2 OFFSET $3;`
	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()
	var planAttributes []*planattributepb.PlanAttribute
	var totalCount int64
	for rows.Next() {
		var id, planId, attributeId, attributeValue string
		var active bool
		var dateCreated, dateModified time.Time
		var total int64
		if err := rows.Scan(&id, &planId, &attributeId, &attributeValue, &active, &dateCreated, &dateModified, &total); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total

		rawData := map[string]interface{}{
			"id":          id,
			"planId":      planId,
			"attributeId": attributeId,
			"value":       attributeValue,
			"active":      active,
		}

		if !dateCreated.IsZero() {
			rawData["dateCreated"] = dateCreated.UnixMilli()
			rawData["dateCreatedString"] = dateCreated.Format(time.RFC3339)
		}
		if !dateModified.IsZero() {
			rawData["dateModified"] = dateModified.UnixMilli()
			rawData["dateModifiedString"] = dateModified.Format(time.RFC3339)
		}

		dataJSON, _ := json.Marshal(rawData)
		planAttribute := &planattributepb.PlanAttribute{}
		if err := protojson.Unmarshal(dataJSON, planAttribute); err == nil {
			planAttributes = append(planAttributes, planAttribute)
		}
	}
	totalPages := int32((totalCount + int64(limit) - 1) / int64(limit))
	return &planattributepb.GetPlanAttributeListPageDataResponse{PlanAttributeList: planAttributes, Pagination: &commonpb.PaginationResponse{TotalItems: int32(totalCount), CurrentPage: &page, TotalPages: &totalPages, HasNext: page < totalPages, HasPrev: page > 1}, Success: true}, nil
}

// GetPlanAttributeItemPageData retrieves plan attribute item page data
func (r *PostgresPlanAttributeRepository) GetPlanAttributeItemPageData(ctx context.Context, req *planattributepb.GetPlanAttributeItemPageDataRequest) (*planattributepb.GetPlanAttributeItemPageDataResponse, error) {
	if req == nil || req.PlanAttributeId == "" {
		return nil, fmt.Errorf("plan attribute ID required")
	}
	query := `SELECT id, plan_id, attribute_id, value, active, date_created, date_modified FROM plan_attribute WHERE id = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, req.PlanAttributeId)
	var id, planId, attributeId, attributeValue string
	var active bool
	var dateCreated, dateModified time.Time
	if err := row.Scan(&id, &planId, &attributeId, &attributeValue, &active, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("plan attribute not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	rawData := map[string]interface{}{
		"id":          id,
		"planId":      planId,
		"attributeId": attributeId,
		"value":       attributeValue,
		"active":      active,
	}

	if !dateCreated.IsZero() {
		rawData["dateCreated"] = dateCreated.UnixMilli()
		rawData["dateCreatedString"] = dateCreated.Format(time.RFC3339)
	}
	if !dateModified.IsZero() {
		rawData["dateModified"] = dateModified.UnixMilli()
		rawData["dateModifiedString"] = dateModified.Format(time.RFC3339)
	}

	dataJSON, _ := json.Marshal(rawData)
	planAttribute := &planattributepb.PlanAttribute{}
	if err := protojson.Unmarshal(dataJSON, planAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	return &planattributepb.GetPlanAttributeItemPageDataResponse{PlanAttribute: planAttribute, Success: true}, nil
}

// NewPlanAttributeRepository creates a new PostgreSQL plan_attribute repository (old-style constructor)
func NewPlanAttributeRepository(db *sql.DB, tableName string) planattributepb.PlanAttributeDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresPlanAttributeRepository(dbOps, tableName)
}
