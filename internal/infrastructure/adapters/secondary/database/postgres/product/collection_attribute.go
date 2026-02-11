//go:build postgresql

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	postgresCore "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/postgres/core"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", "collection_attribute", func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres collection_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresCollectionAttributeRepository(dbOps, tableName), nil
	})
}

// PostgresCollectionAttributeRepository implements collection_attribute CRUD operations using PostgreSQL
type PostgresCollectionAttributeRepository struct {
	collectionattributepb.UnimplementedCollectionAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB // Direct database access for complex queries (CTEs)
	tableName string
}

// NewPostgresCollectionAttributeRepository creates a new PostgreSQL collection attribute repository
func NewPostgresCollectionAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionattributepb.CollectionAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "collection_attribute" // default fallback
	}

	// Extract the underlying database connection for complex queries (CTEs)
	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCollectionAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCollectionAttribute creates a new collection attribute using common PostgreSQL operations
func (r *PostgresCollectionAttributeRepository) CreateCollectionAttribute(ctx context.Context, req *collectionattributepb.CreateCollectionAttributeRequest) (*collectionattributepb.CreateCollectionAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection attribute data is required")
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
		return nil, fmt.Errorf("failed to create collection attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collectionAttribute := &collectionattributepb.CollectionAttribute{}
	if err := protojson.Unmarshal(resultJSON, collectionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionattributepb.CreateCollectionAttributeResponse{
		Data: []*collectionattributepb.CollectionAttribute{collectionAttribute},
	}, nil
}

// ReadCollectionAttribute retrieves a collection attribute using common PostgreSQL operations
func (r *PostgresCollectionAttributeRepository) ReadCollectionAttribute(ctx context.Context, req *collectionattributepb.ReadCollectionAttributeRequest) (*collectionattributepb.ReadCollectionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection attribute ID is required")
	}

	// Read document using common operations
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection attribute: %w", err)
	}

	// Convert result to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collectionAttribute := &collectionattributepb.CollectionAttribute{}
	if err := protojson.Unmarshal(resultJSON, collectionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionattributepb.ReadCollectionAttributeResponse{
		Data: []*collectionattributepb.CollectionAttribute{collectionAttribute},
	}, nil
}

// UpdateCollectionAttribute updates a collection attribute using common PostgreSQL operations
func (r *PostgresCollectionAttributeRepository) UpdateCollectionAttribute(ctx context.Context, req *collectionattributepb.UpdateCollectionAttributeRequest) (*collectionattributepb.UpdateCollectionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection attribute ID is required")
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
		return nil, fmt.Errorf("failed to update collection attribute: %w", err)
	}

	// Convert result back to protobuf using protojson
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collectionAttribute := &collectionattributepb.CollectionAttribute{}
	if err := protojson.Unmarshal(resultJSON, collectionAttribute); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionattributepb.UpdateCollectionAttributeResponse{
		Data: []*collectionattributepb.CollectionAttribute{collectionAttribute},
	}, nil
}

// DeleteCollectionAttribute deletes a collection attribute using common PostgreSQL operations
func (r *PostgresCollectionAttributeRepository) DeleteCollectionAttribute(ctx context.Context, req *collectionattributepb.DeleteCollectionAttributeRequest) (*collectionattributepb.DeleteCollectionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection attribute ID is required")
	}

	// Delete document using common operations (soft delete)
	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection attribute: %w", err)
	}

	return &collectionattributepb.DeleteCollectionAttributeResponse{
		Success: true,
	}, nil
}

// ListCollectionAttributes lists collection attributes using common PostgreSQL operations
func (r *PostgresCollectionAttributeRepository) ListCollectionAttributes(ctx context.Context, req *collectionattributepb.ListCollectionAttributesRequest) (*collectionattributepb.ListCollectionAttributesResponse, error) {
	// List documents using common operations
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collection attributes: %w", err)
	}

	// Convert results to protobuf slice using protojson
	var collectionAttributes []*collectionattributepb.CollectionAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			// Log error and continue with next item
			continue
		}

		collectionAttribute := &collectionattributepb.CollectionAttribute{}
		if err := protojson.Unmarshal(resultJSON, collectionAttribute); err != nil {
			// Log error and continue with next item
			continue
		}
		collectionAttributes = append(collectionAttributes, collectionAttribute)
	}

	return &collectionattributepb.ListCollectionAttributesResponse{
		Data: collectionAttributes,
	}, nil
}

// GetCollectionAttributeListPageData retrieves collection attributes with advanced filtering, sorting, searching, and pagination using CTE
func (r *PostgresCollectionAttributeRepository) GetCollectionAttributeListPageData(
	ctx context.Context,
	req *collectionattributepb.GetCollectionAttributeListPageDataRequest,
) (*collectionattributepb.GetCollectionAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	// Build search condition
	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

	// Default pagination values
	limit := int32(50)
	offset := int32(0)
	page := int32(1)
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if offsetPag := req.Pagination.GetOffset(); offsetPag != nil {
			if offsetPag.Page > 0 {
				page = offsetPag.Page
				offset = (page - 1) * limit
			}
		}
	}

	// Default sort
	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	// CTE Query - Attribute table pattern with collection_id/attribute_id/value search
	query := `
		WITH enriched AS (
			SELECT
				ca.id,
				ca.collection_id,
				ca.attribute_id,
				ca.value,
				ca.date_created,
				ca.date_modified
			FROM collection_attribute ca
			WHERE ($1::text IS NULL OR $1::text = '' OR
			       ca.collection_id ILIKE $1 OR
			       ca.attribute_id ILIKE $1 OR
			       ca.value ILIKE $1)
		),
		counted AS (
			SELECT COUNT(*) as total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY ` + sortField + ` ` + sortOrder + `
		LIMIT $2 OFFSET $3;
	`

	rows, err := r.db.QueryContext(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var collectionAttributes []*collectionattributepb.CollectionAttribute
	var totalCount int64

	for rows.Next() {
		var (
			id           string
			collectionId string
			attributeId  string
			value        string
			dateCreated  time.Time
			dateModified time.Time
			total        int64
		)

		if err := rows.Scan(
			&id,
			&collectionId,
			&attributeId,
			&value,
			&dateCreated,
			&dateModified,
			&total,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		totalCount = total

		collectionAttribute := &collectionattributepb.CollectionAttribute{
			Id:           id,
			CollectionId: collectionId,
			AttributeId:  attributeId,
			Value:        value,
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			collectionAttribute.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			collectionAttribute.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			collectionAttribute.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			collectionAttribute.DateModifiedString = &dmStr
		}

		collectionAttributes = append(collectionAttributes, collectionAttribute)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &collectionattributepb.GetCollectionAttributeListPageDataResponse{
		CollectionAttributeList: collectionAttributes,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  int32(totalCount),
			CurrentPage: &page,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
		Success: true,
	}, nil
}

// GetCollectionAttributeItemPageData retrieves a single collection attribute with enhanced item page data
func (r *PostgresCollectionAttributeRepository) GetCollectionAttributeItemPageData(
	ctx context.Context,
	req *collectionattributepb.GetCollectionAttributeItemPageDataRequest,
) (*collectionattributepb.GetCollectionAttributeItemPageDataResponse, error) {
	if req == nil || req.CollectionAttributeId == "" {
		return nil, fmt.Errorf("collection attribute ID required")
	}

	query := `
		SELECT
			ca.id,
			ca.collection_id,
			ca.attribute_id,
			ca.value,
			ca.date_created,
			ca.date_modified
		FROM collection_attribute ca
		WHERE ca.id = $1
		LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.CollectionAttributeId)

	var (
		id           string
		collectionId string
		attributeId  string
		value        string
		dateCreated  time.Time
		dateModified time.Time
	)

	if err := row.Scan(
		&id,
		&collectionId,
		&attributeId,
		&value,
		&dateCreated,
		&dateModified,
	); err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection attribute not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	collectionAttribute := &collectionattributepb.CollectionAttribute{
		Id:           id,
		CollectionId: collectionId,
		AttributeId:  attributeId,
		Value:        value,
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		collectionAttribute.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		collectionAttribute.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		collectionAttribute.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		collectionAttribute.DateModifiedString = &dmStr
	}

	return &collectionattributepb.GetCollectionAttributeItemPageDataResponse{
		CollectionAttribute: collectionAttribute,
		Success:             true,
	}, nil
}

// NewCollectionAttributeRepository creates a new PostgreSQL collection_attribute repository (old-style constructor)
func NewCollectionAttributeRepository(db *sql.DB, tableName string) collectionattributepb.CollectionAttributeDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresCollectionAttributeRepository(dbOps, tableName)
}
