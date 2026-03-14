
package treasury

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	interfaces "github.com/erniealice/espyna-golang/database/interfaces"
	postgresCore "github.com/erniealice/espyna-golang/contrib/postgres/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
)

func init() {
	registry.RegisterRepositoryFactory("postgresql", entityid.TreasuryCollection, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("postgres collection repository requires *sql.DB, got %T", conn)
		}
		dbOps := postgresCore.NewPostgresOperations(db)
		return NewPostgresCollectionRepository(dbOps, tableName), nil
	})
}

// PostgresCollectionRepository implements collection CRUD operations using PostgreSQL
type PostgresCollectionRepository struct {
	collectionpb.UnimplementedCollectionDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	db        *sql.DB
	tableName string
}

// NewPostgresCollectionRepository creates a new PostgreSQL collection repository
func NewPostgresCollectionRepository(dbOps interfaces.DatabaseOperation, tableName string) collectionpb.CollectionDomainServiceServer {
	if tableName == "" {
		tableName = "treasury_collection"
	}

	var db *sql.DB
	if pgOps, ok := dbOps.(interface{ GetDB() *sql.DB }); ok {
		db = pgOps.GetDB()
	}

	return &PostgresCollectionRepository{
		dbOps:     dbOps,
		db:        db,
		tableName: tableName,
	}
}

// CreateCollection creates a new collection record
func (r *PostgresCollectionRepository) CreateCollection(ctx context.Context, req *collectionpb.CreateCollectionRequest) (*collectionpb.CreateCollectionResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("collection data is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "paymentDate", "payment_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Create(ctx, r.tableName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.CreateCollectionResponse{
		Success: true,
		Data:    []*collectionpb.Collection{collection},
	}, nil
}

// ReadCollection retrieves a collection record by ID
func (r *PostgresCollectionRepository) ReadCollection(ctx context.Context, req *collectionpb.ReadCollectionRequest) (*collectionpb.ReadCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	result, err := r.dbOps.Read(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to read collection: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.ReadCollectionResponse{
		Success: true,
		Data:    []*collectionpb.Collection{collection},
	}, nil
}

// UpdateCollection updates a collection record
func (r *PostgresCollectionRepository) UpdateCollection(ctx context.Context, req *collectionpb.UpdateCollectionRequest) (*collectionpb.UpdateCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	jsonData, err := protojson.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}

	// Convert millis timestamps to time.Time for postgres timestamp columns
	convertMillisToTime(data, "paymentDate", "payment_date")
	convertMillisToTime(data, "dateCreated", "date_created")
	convertMillisToTime(data, "dateModified", "date_modified")

	result, err := r.dbOps.Update(ctx, r.tableName, req.Data.Id, data)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	collection := &collectionpb.Collection{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf: %w", err)
	}

	return &collectionpb.UpdateCollectionResponse{
		Success: true,
		Data:    []*collectionpb.Collection{collection},
	}, nil
}

// DeleteCollection deletes a collection record (soft delete)
func (r *PostgresCollectionRepository) DeleteCollection(ctx context.Context, req *collectionpb.DeleteCollectionRequest) (*collectionpb.DeleteCollectionResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	err := r.dbOps.Delete(ctx, r.tableName, req.Data.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete collection: %w", err)
	}

	return &collectionpb.DeleteCollectionResponse{
		Success: true,
	}, nil
}

// ListCollections lists collection records with optional filters
func (r *PostgresCollectionRepository) ListCollections(ctx context.Context, req *collectionpb.ListCollectionsRequest) (*collectionpb.ListCollectionsResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	listResult, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var collections []*collectionpb.Collection
	for _, result := range listResult.Data {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("WARN: json.Marshal collection row: %v", err)
			continue
		}

		collection := &collectionpb.Collection{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(resultJSON, collection); err != nil {
			log.Printf("WARN: protojson unmarshal collection: %v", err)
			continue
		}
		collections = append(collections, collection)
	}

	return &collectionpb.ListCollectionsResponse{
		Success: true,
		Data:    collections,
	}, nil
}

// GetCollectionListPageData retrieves collections with pagination, filtering, sorting, and search using CTE
func (r *PostgresCollectionRepository) GetCollectionListPageData(
	ctx context.Context,
	req *collectionpb.GetCollectionListPageDataRequest,
) (*collectionpb.GetCollectionListPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection list page data request is required")
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

	sortField := "tc.date_created"
	sortOrder := "DESC"
	if req.Sort != nil && len(req.Sort.Fields) > 0 {
		sortField = req.Sort.Fields[0].Field
		if req.Sort.Fields[0].Direction == commonpb.SortDirection_ASC {
			sortOrder = "ASC"
		}
	}

	query := `
		WITH enriched AS (
			SELECT
				tc.id,
				tc.date_created,
				tc.date_modified,
				tc.active,
				tc.name,
				tc.subscription_id,
				tc.amount,
				tc.status,
				tc.revenue_id,
				tc.collection_method_id,
				tc.currency,
				tc.reference_number,
				tc.payment_date,
				tc.received_by,
				tc.received_role,
				tc.collection_type
			FROM treasury_collection tc
			WHERE tc.active = true
			  AND ($1::text IS NULL OR $1::text = '' OR
			       tc.name ILIKE $1 OR
			       tc.reference_number ILIKE $1 OR
			       tc.status ILIKE $1 OR
			       tc.collection_type ILIKE $1)
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
		return nil, fmt.Errorf("failed to query collection list page data: %w", err)
	}
	defer rows.Close()

	var collections []*collectionpb.Collection
	var totalCount int64

	for rows.Next() {
		var (
			id                 string
			dateCreated        time.Time
			dateModified       time.Time
			active             bool
			name               string
			subscriptionID     *string
			amount             float64
			status             *string
			revenueID          *string
			collectionMethodID *string
			currency           *string
			referenceNumber    *string
			paymentDate        *time.Time
			receivedBy         *string
			receivedRole       *string
			collectionType     *string
			total              int64
		)

		err := rows.Scan(
			&id,
			&dateCreated,
			&dateModified,
			&active,
			&name,
			&subscriptionID,
			&amount,
			&status,
			&revenueID,
			&collectionMethodID,
			&currency,
			&referenceNumber,
			&paymentDate,
			&receivedBy,
			&receivedRole,
			&collectionType,
			&total,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan collection row: %w", err)
		}

		totalCount = total

		collection := &collectionpb.Collection{
			Id:     id,
			Active: active,
			Name:   name,
			Amount: amount,
		}

		if subscriptionID != nil {
			collection.SubscriptionId = *subscriptionID
		}
		if status != nil {
			collection.Status = *status
		}
		if revenueID != nil {
			collection.RevenueId = *revenueID
		}
		if collectionMethodID != nil {
			collection.CollectionMethodId = *collectionMethodID
		}
		if currency != nil {
			collection.Currency = *currency
		}
		if referenceNumber != nil {
			collection.ReferenceNumber = *referenceNumber
		}
		if receivedBy != nil {
			collection.ReceivedBy = *receivedBy
		}
		if receivedRole != nil {
			collection.ReceivedRole = *receivedRole
		}
		if collectionType != nil {
			collection.CollectionType = *collectionType
		}
		if paymentDate != nil && !paymentDate.IsZero() {
			ts := paymentDate.UnixMilli()
			collection.PaymentDate = ts
		}

		if !dateCreated.IsZero() {
			ts := dateCreated.UnixMilli()
			collection.DateCreated = &ts
			dcStr := dateCreated.Format(time.RFC3339)
			collection.DateCreatedString = &dcStr
		}
		if !dateModified.IsZero() {
			ts := dateModified.UnixMilli()
			collection.DateModified = &ts
			dmStr := dateModified.Format(time.RFC3339)
			collection.DateModifiedString = &dmStr
		}

		collections = append(collections, collection)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating collection rows: %w", err)
	}

	totalPages := int32(0)
	if limit > 0 {
		totalPages = int32((totalCount + int64(limit) - 1) / int64(limit))
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return &collectionpb.GetCollectionListPageDataResponse{
		CollectionList: collections,
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

// GetCollectionItemPageData retrieves a single collection with enriched data using CTE
func (r *PostgresCollectionRepository) GetCollectionItemPageData(
	ctx context.Context,
	req *collectionpb.GetCollectionItemPageDataRequest,
) (*collectionpb.GetCollectionItemPageDataResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get collection item page data request is required")
	}
	if req.CollectionId == "" {
		return nil, fmt.Errorf("collection ID is required")
	}

	query := `
		WITH enriched AS (
			SELECT
				tc.id,
				tc.date_created,
				tc.date_modified,
				tc.active,
				tc.name,
				tc.subscription_id,
				tc.amount,
				tc.status,
				tc.revenue_id,
				tc.collection_method_id,
				tc.currency,
				tc.reference_number,
				tc.payment_date,
				tc.received_by,
				tc.received_role,
				tc.collection_type
			FROM treasury_collection tc
			WHERE tc.id = $1 AND tc.active = true
		)
		SELECT * FROM enriched LIMIT 1;
	`

	row := r.db.QueryRowContext(ctx, query, req.CollectionId)

	var (
		id                 string
		dateCreated        time.Time
		dateModified       time.Time
		active             bool
		name               string
		subscriptionID     *string
		amount             float64
		status             *string
		revenueID          *string
		collectionMethodID *string
		currency           *string
		referenceNumber    *string
		paymentDate        *time.Time
		receivedBy         *string
		receivedRole       *string
		collectionType     *string
	)

	err := row.Scan(
		&id,
		&dateCreated,
		&dateModified,
		&active,
		&name,
		&subscriptionID,
		&amount,
		&status,
		&revenueID,
		&collectionMethodID,
		&currency,
		&referenceNumber,
		&paymentDate,
		&receivedBy,
		&receivedRole,
		&collectionType,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection with ID '%s' not found", req.CollectionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query collection item page data: %w", err)
	}

	collection := &collectionpb.Collection{
		Id:     id,
		Active: active,
		Name:   name,
		Amount: amount,
	}

	if subscriptionID != nil {
		collection.SubscriptionId = *subscriptionID
	}
	if status != nil {
		collection.Status = *status
	}
	if revenueID != nil {
		collection.RevenueId = *revenueID
	}
	if collectionMethodID != nil {
		collection.CollectionMethodId = *collectionMethodID
	}
	if currency != nil {
		collection.Currency = *currency
	}
	if referenceNumber != nil {
		collection.ReferenceNumber = *referenceNumber
	}
	if receivedBy != nil {
		collection.ReceivedBy = *receivedBy
	}
	if receivedRole != nil {
		collection.ReceivedRole = *receivedRole
	}
	if collectionType != nil {
		collection.CollectionType = *collectionType
	}
	if paymentDate != nil && !paymentDate.IsZero() {
		ts := paymentDate.UnixMilli()
		collection.PaymentDate = ts
	}

	if !dateCreated.IsZero() {
		ts := dateCreated.UnixMilli()
		collection.DateCreated = &ts
		dcStr := dateCreated.Format(time.RFC3339)
		collection.DateCreatedString = &dcStr
	}
	if !dateModified.IsZero() {
		ts := dateModified.UnixMilli()
		collection.DateModified = &ts
		dmStr := dateModified.Format(time.RFC3339)
		collection.DateModifiedString = &dmStr
	}

	return &collectionpb.GetCollectionItemPageDataResponse{
		Collection: collection,
		Success:    true,
	}, nil
}

// NewCollectionRepository creates a new PostgreSQL collection repository (old-style constructor)
func NewCollectionRepository(db *sql.DB, tableName string) collectionpb.CollectionDomainServiceServer {
	dbOps := postgresCore.NewPostgresOperations(db)
	return NewPostgresCollectionRepository(dbOps, tableName)
}

// convertMillisToTime converts a millis-epoch value in a JSON map to time.Time.
// Protobuf int64 fields serialize to JSON strings via protojson (e.g. "1771886746000").
// Postgres timestamp columns need time.Time, not raw millis.
func convertMillisToTime(data map[string]any, jsonKey, _ string) {
	v, ok := data[jsonKey]
	if !ok {
		return
	}
	switch val := v.(type) {
	case string:
		// protojson serializes int64 as string
		var millis int64
		if _, err := fmt.Sscanf(val, "%d", &millis); err == nil && millis > 1e12 {
			data[jsonKey] = time.UnixMilli(millis)
		}
	case float64:
		if val > 1e12 {
			data[jsonKey] = time.UnixMilli(int64(val))
		}
	}
}
