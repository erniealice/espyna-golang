//go:build sqlserver

package product

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sqlserverCore "github.com/erniealice/espyna-golang/contrib/sqlserver/internal/adapter/core"
	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/collection_attribute"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("sqlserver", entityid.CollectionAttribute, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("sqlserver collection_attribute repository requires *sql.DB, got %T", conn)
		}
		dbOps := sqlserverCore.NewWorkspaceAwareOperations(db)
		return NewSQLServerCollectionAttributeRepository(dbOps, tableName), nil
	})
}

// SQLServerCollectionAttributeRepository implements collection_attribute CRUD using SQL Server.
type SQLServerCollectionAttributeRepository struct {
	collectionattributepb.UnimplementedCollectionAttributeDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewSQLServerCollectionAttributeRepository creates a new SQL Server collection_attribute repository.
func NewSQLServerCollectionAttributeRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionattributepb.CollectionAttributeDomainServiceServer {
	if tableName == "" {
		tableName = "collection_attribute"
	}
	var db *sql.DB
	if ops, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = ops.GetDB()
	}
	return &SQLServerCollectionAttributeRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

func (r *SQLServerCollectionAttributeRepository) CreateCollectionAttribute(ctx context.Context, req *collectionattributepb.CreateCollectionAttributeRequest) (*collectionattributepb.CreateCollectionAttributeResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection attribute data is required")
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
		return nil, fmt.Errorf("failed to create collection attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ca := &collectionattributepb.CollectionAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ca); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &collectionattributepb.CreateCollectionAttributeResponse{Data: []*collectionattributepb.CollectionAttribute{ca}}, nil
}

func (r *SQLServerCollectionAttributeRepository) ReadCollectionAttribute(ctx context.Context, req *collectionattributepb.ReadCollectionAttributeRequest) (*collectionattributepb.ReadCollectionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection attribute ID is required")
	}
	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ca := &collectionattributepb.CollectionAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ca); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &collectionattributepb.ReadCollectionAttributeResponse{Data: []*collectionattributepb.CollectionAttribute{ca}}, nil
}

func (r *SQLServerCollectionAttributeRepository) UpdateCollectionAttribute(ctx context.Context, req *collectionattributepb.UpdateCollectionAttributeRequest) (*collectionattributepb.UpdateCollectionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection attribute ID is required")
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
		return nil, fmt.Errorf("failed to update collection attribute: %w", err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	ca := &collectionattributepb.CollectionAttribute{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ca); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}
	return &collectionattributepb.UpdateCollectionAttributeResponse{Data: []*collectionattributepb.CollectionAttribute{ca}}, nil
}

func (r *SQLServerCollectionAttributeRepository) DeleteCollectionAttribute(ctx context.Context, req *collectionattributepb.DeleteCollectionAttributeRequest) (*collectionattributepb.DeleteCollectionAttributeResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection attribute ID is required")
	}
	if err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id); err != nil {
		return nil, fmt.Errorf("failed to delete collection attribute: %w", err)
	}
	return &collectionattributepb.DeleteCollectionAttributeResponse{Success: true}, nil
}

func (r *SQLServerCollectionAttributeRepository) ListCollectionAttributes(ctx context.Context, req *collectionattributepb.ListCollectionAttributesRequest) (*collectionattributepb.ListCollectionAttributesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collection attributes: %w", err)
	}
	var cas []*collectionattributepb.CollectionAttribute
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			continue
		}
		ca := &collectionattributepb.CollectionAttribute{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, ca); err != nil {
			continue
		}
		cas = append(cas, ca)
	}
	return &collectionattributepb.ListCollectionAttributesResponse{Data: cas}, nil
}

// GetCollectionAttributeListPageData retrieves collection attributes with filtering,
// sorting, searching, and pagination.
//
// SQL Server: ILIKE → LIKE; LIMIT/OFFSET → OFFSET/FETCH; $N → @pN.
func (r *SQLServerCollectionAttributeRepository) GetCollectionAttributeListPageData(
	ctx context.Context,
	req *collectionattributepb.GetCollectionAttributeListPageDataRequest,
) (*collectionattributepb.GetCollectionAttributeListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request required")
	}

	searchPattern := ""
	if req.Search != nil && req.Search.Query != "" {
		searchPattern = "%" + req.Search.Query + "%"
	}

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

	sortField := "date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}
	allowedSortFields := map[string]bool{
		"date_created": true, "date_modified": true, "collection_id": true,
		"attribute_id": true, "value": true,
	}
	if !allowedSortFields[sortField] {
		sortField = "date_created"
	}

	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

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
			WHERE (@p1 = '' OR
			       ca.collection_id LIKE @p1 OR
			       ca.attribute_id LIKE @p1 OR
			       ca.value LIKE @p1)
		),
		counted AS (
			SELECT COUNT(*) AS total FROM enriched
		)
		SELECT
			e.*,
			c.total
		FROM enriched e, counted c
		ORDER BY e.` + sortField + ` ` + sortOrder + `
		OFFSET @p3 ROWS FETCH NEXT @p2 ROWS ONLY
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
			&id, &collectionId, &attributeId, &value, &dateCreated, &dateModified, &total,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		totalCount = total
		ca := &collectionattributepb.CollectionAttribute{
			Id:           id,
			CollectionId: collectionId,
			AttributeId:  attributeId,
			Value:        value,
		}
		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			ca.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			ca.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			ca.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			ca.DateModifiedString = &dmStr
		}
		collectionAttributes = append(collectionAttributes, ca)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

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

// GetCollectionAttributeItemPageData retrieves a single collection attribute.
func (r *SQLServerCollectionAttributeRepository) GetCollectionAttributeItemPageData(
	ctx context.Context,
	req *collectionattributepb.GetCollectionAttributeItemPageDataRequest,
) (*collectionattributepb.GetCollectionAttributeItemPageDataResponse, error) {
	if req == nil || req.CollectionAttributeId == "" {
		return nil, fmt.Errorf("collection attribute ID required")
	}
	if r.db == nil {
		return nil, fmt.Errorf("database connection is not available")
	}

	query := `
		SELECT TOP 1
			ca.id,
			ca.collection_id,
			ca.attribute_id,
			ca.value,
			ca.date_created,
			ca.date_modified
		FROM collection_attribute ca
		WHERE ca.id = @p1
	`

	var (
		id           string
		collectionId string
		attributeId  string
		value        string
		dateCreated  time.Time
		dateModified time.Time
	)
	row := r.db.QueryRowContext(ctx, query, req.CollectionAttributeId)
	if err := row.Scan(&id, &collectionId, &attributeId, &value, &dateCreated, &dateModified); err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection attribute not found")
	} else if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	ca := &collectionattributepb.CollectionAttribute{
		Id:           id,
		CollectionId: collectionId,
		AttributeId:  attributeId,
		Value:        value,
	}
	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		ca.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		ca.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		ca.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		ca.DateModifiedString = &dmStr
	}
	return &collectionattributepb.GetCollectionAttributeItemPageDataResponse{
		CollectionAttribute: ca,
		Success:             true,
	}, nil
}
