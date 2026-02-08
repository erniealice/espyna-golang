//go:build firestore

package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	interfaces "github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/interface"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/common/model"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

func init() {
	// Register database operations factory for firestore
	registry.RegisterDatabaseOperationsFactory("firestore", func(conn any) (any, error) {
		client, ok := conn.(*firestore.Client)
		if !ok {
			return nil, fmt.Errorf("firestore: expected *firestore.Client, got %T", conn)
		}
		return NewFirestoreOperations(client), nil
	})
}

// FirestoreOperations implements DatabaseOperation for Firestore
type FirestoreOperations struct {
	client *firestore.Client
}

// NewFirestoreOperations creates a new Firestore operations instance
func NewFirestoreOperations(client *firestore.Client) interfaces.DatabaseOperation {
	return &FirestoreOperations{
		client: client,
	}
}

// Create creates a new document in the specified collection
func (f *FirestoreOperations) Create(ctx context.Context, collectionName string, data map[string]any) (map[string]any, error) {
	if collectionName == "" {
		return nil, model.NewDatabaseError("collection name is required", "MISSING_COLLECTION_NAME", 400)
	}

	// Generate document ID if not provided
	var docRef *firestore.DocumentRef
	if id, exists := data["id"]; exists && id != "" {
		docRef = f.client.Collection(collectionName).Doc(id.(string))
	} else {
		docRef = f.client.Collection(collectionName).NewDoc()
		data["id"] = docRef.ID
	}

	// Set creation properties - store as int64 and string for protobuf compatibility
	now := time.Now().UTC()
	data["active"] = true
	data["date_created"] = now.UnixMilli() // Store as int64 for protobuf
	data["date_created_string"] = now.Format("2006-01-02T15:04:05.000Z")
	data["date_modified"] = now.UnixMilli() // Store as int64 for protobuf
	data["date_modified_string"] = now.Format("2006-01-02T15:04:05.000Z")

	// Create document
	_, err := docRef.Set(ctx, data)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to create document: %v", err),
			"FIRESTORE_CREATE_FAILED",
			500,
		)
	}

	return data, nil
}

// Read retrieves a document by ID from the specified collection
func (f *FirestoreOperations) Read(ctx context.Context, collectionName string, id string) (map[string]any, error) {
	if collectionName == "" {
		return nil, model.NewDatabaseError("collection name is required", "MISSING_COLLECTION_NAME", 400)
	}
	if id == "" {
		return nil, model.NewDatabaseError("document ID is required", "MISSING_DOCUMENT_ID", 400)
	}

	docSnap, err := f.client.Collection(collectionName).Doc(id).Get(ctx)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get document: %v", err),
			"FIRESTORE_READ_FAILED",
			500,
		)
	}

	if !docSnap.Exists() {
		return nil, model.NewDatabaseError("document not found", "DOCUMENT_NOT_FOUND", 404)
	}

	data := docSnap.Data()
	data["id"] = docSnap.Ref.ID

	return data, nil
}

// Update updates an existing document in the specified collection
func (f *FirestoreOperations) Update(ctx context.Context, collectionName string, id string, data map[string]any) (map[string]any, error) {
	if collectionName == "" {
		return nil, model.NewDatabaseError("collection name is required", "MISSING_COLLECTION_NAME", 400)
	}
	if id == "" {
		return nil, model.NewDatabaseError("document ID is required", "MISSING_DOCUMENT_ID", 400)
	}

	docRef := f.client.Collection(collectionName).Doc(id)

	// Check if document exists
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to get document: %v", err),
			"FIRESTORE_READ_FAILED",
			500,
		)
	}
	if !docSnap.Exists() {
		return nil, model.NewDatabaseError("document not found", "DOCUMENT_NOT_FOUND", 404)
	}

	// Set update properties - store as int64 and string for protobuf compatibility
	now := time.Now().UTC()
	data["date_modified"] = now.UnixMilli() // Store as int64 for protobuf
	data["date_modified_string"] = now.Format("2006-01-02T15:04:05.000Z")

	// Preserve original creation data
	originalData := docSnap.Data()
	if dateCreated, exists := originalData["date_created"]; exists {
		data["date_created"] = dateCreated
	}
	if dateCreatedString, exists := originalData["date_created_string"]; exists {
		data["date_created_string"] = dateCreatedString
	}

	// Update document using merge to preserve fields not being updated
	_, err = docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to update document: %v", err),
			"FIRESTORE_UPDATE_FAILED",
			500,
		)
	}

	// Return updated data
	data["id"] = id
	return data, nil
}

// Delete deletes a document from the specified collection (soft delete by default)
func (f *FirestoreOperations) Delete(ctx context.Context, collectionName string, id string) error {
	if collectionName == "" {
		return model.NewDatabaseError("collection name is required", "MISSING_COLLECTION_NAME", 400)
	}
	if id == "" {
		return model.NewDatabaseError("document ID is required", "MISSING_DOCUMENT_ID", 400)
	}

	docRef := f.client.Collection(collectionName).Doc(id)

	// Check if document exists
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get document: %v", err),
			"FIRESTORE_READ_FAILED",
			500,
		)
	}
	if !docSnap.Exists() {
		return model.NewDatabaseError("document not found", "DOCUMENT_NOT_FOUND", 404)
	}

	// Soft delete by setting active to false
	now := time.Now().UTC()
	updateData := map[string]any{
		"active":               false,
		"date_modified":        now.UnixMilli(), // Store as int64 for protobuf
		"date_modified_string": now.Format("2006-01-02T15:04:05.000Z"),
	}

	_, err = docRef.Set(ctx, updateData, firestore.MergeAll)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to delete document: %v", err),
			"FIRESTORE_DELETE_FAILED",
			500,
		)
	}

	return nil
}

// HardDelete permanently deletes a document from the specified collection
func (f *FirestoreOperations) HardDelete(ctx context.Context, collectionName string, id string) error {
	if collectionName == "" {
		return model.NewDatabaseError("collection name is required", "MISSING_COLLECTION_NAME", 400)
	}
	if id == "" {
		return model.NewDatabaseError("document ID is required", "MISSING_DOCUMENT_ID", 400)
	}

	docRef := f.client.Collection(collectionName).Doc(id)

	// Check if document exists
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to get document: %v", err),
			"FIRESTORE_READ_FAILED",
			500,
		)
	}
	if !docSnap.Exists() {
		return model.NewDatabaseError("document not found", "DOCUMENT_NOT_FOUND", 404)
	}

	// Permanently delete document
	_, err = docRef.Delete(ctx)
	if err != nil {
		return model.NewDatabaseError(
			fmt.Sprintf("failed to hard delete document: %v", err),
			"FIRESTORE_HARD_DELETE_FAILED",
			500,
		)
	}

	return nil
}

// List retrieves documents from the specified collection with standardized params
func (f *FirestoreOperations) List(ctx context.Context, collectionName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	if collectionName == "" {
		return nil, model.NewDatabaseError("collection name is required", "MISSING_COLLECTION_NAME", 400)
	}

	query := f.client.Collection(collectionName).Query

	// Apply default active filter
	query = query.Where("active", "==", true)

	// Apply filters from FilterRequest
	if params != nil && params.Filters != nil {
		for _, filter := range params.Filters.Filters {
			query = f.applyTypedFilter(query, filter)
		}
	}

	// Apply sorting from SortRequest
	if params != nil && params.Sort != nil {
		for _, sortField := range params.Sort.Fields {
			direction := firestore.Asc
			if sortField.Direction == commonpb.SortDirection_DESC {
				direction = firestore.Desc
			}
			query = query.OrderBy(sortField.Field, direction)
		}
	}

	// Get total count before pagination (for pagination response)
	countQuery := query
	allDocs, err := countQuery.Documents(ctx).GetAll()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to count documents: %v", err),
			"FIRESTORE_COUNT_FAILED",
			500,
		)
	}
	totalItems := int32(len(allDocs))

	// Apply pagination from PaginationRequest
	limit := int32(100) // Default limit
	offset := int32(0)
	if params != nil && params.Pagination != nil {
		if params.Pagination.Limit > 0 && params.Pagination.Limit <= 100 {
			limit = params.Pagination.Limit
		}
		// Handle offset pagination
		if offsetPagination := params.Pagination.GetOffset(); offsetPagination != nil {
			if offsetPagination.Page > 0 {
				offset = (offsetPagination.Page - 1) * limit
			}
		}
	}

	// Apply limit and offset
	query = query.Limit(int(limit))
	if offset > 0 {
		query = query.Offset(int(offset))
	}

	// Execute query
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to list documents from collection '%s': %v", collectionName, err),
			"FIRESTORE_LIST_FAILED",
			500,
		)
	}

	// Convert documents to map slice
	var results []map[string]any
	for _, doc := range docs {
		data := doc.Data()
		data["id"] = doc.Ref.ID
		results = append(results, data)
	}

	// Build pagination response
	currentPage := int32(1)
	if offset > 0 && limit > 0 {
		currentPage = (offset / limit) + 1
	}
	totalPages := (totalItems + limit - 1) / limit
	hasNext := currentPage < totalPages
	hasPrev := currentPage > 1

	return &interfaces.ListResult{
		Data:  results,
		Total: totalItems,
		Pagination: &commonpb.PaginationResponse{
			TotalItems:  totalItems,
			CurrentPage: &currentPage,
			TotalPages:  &totalPages,
			HasNext:     hasNext,
			HasPrev:     hasPrev,
		},
	}, nil
}

// applyTypedFilter applies a TypedFilter to a Firestore query
func (f *FirestoreOperations) applyTypedFilter(query firestore.Query, filter *commonpb.TypedFilter) firestore.Query {
	field := filter.Field

	switch ft := filter.FilterType.(type) {
	case *commonpb.TypedFilter_StringFilter:
		switch ft.StringFilter.Operator {
		case commonpb.StringOperator_STRING_EQUALS:
			query = query.Where(field, "==", ft.StringFilter.Value)
		case commonpb.StringOperator_STRING_NOT_EQUALS:
			query = query.Where(field, "!=", ft.StringFilter.Value)
		case commonpb.StringOperator_STRING_STARTS_WITH:
			// Firestore prefix query
			query = query.Where(field, ">=", ft.StringFilter.Value).
				Where(field, "<", ft.StringFilter.Value+"\uf8ff")
		}
	case *commonpb.TypedFilter_NumberFilter:
		switch ft.NumberFilter.Operator {
		case commonpb.NumberOperator_NUMBER_EQUALS:
			query = query.Where(field, "==", ft.NumberFilter.Value)
		case commonpb.NumberOperator_NUMBER_NOT_EQUALS:
			query = query.Where(field, "!=", ft.NumberFilter.Value)
		case commonpb.NumberOperator_NUMBER_GREATER_THAN:
			query = query.Where(field, ">", ft.NumberFilter.Value)
		case commonpb.NumberOperator_NUMBER_GREATER_THAN_OR_EQUAL:
			query = query.Where(field, ">=", ft.NumberFilter.Value)
		case commonpb.NumberOperator_NUMBER_LESS_THAN:
			query = query.Where(field, "<", ft.NumberFilter.Value)
		case commonpb.NumberOperator_NUMBER_LESS_THAN_OR_EQUAL:
			query = query.Where(field, "<=", ft.NumberFilter.Value)
		}
	case *commonpb.TypedFilter_BooleanFilter:
		query = query.Where(field, "==", ft.BooleanFilter.Value)
	case *commonpb.TypedFilter_ListFilter:
		switch ft.ListFilter.Operator {
		case commonpb.ListOperator_LIST_IN:
			query = query.Where(field, "in", ft.ListFilter.Values)
		case commonpb.ListOperator_LIST_NOT_IN:
			query = query.Where(field, "not-in", ft.ListFilter.Values)
		}
	case *commonpb.TypedFilter_RangeFilter:
		if ft.RangeFilter.IncludeMin {
			query = query.Where(field, ">=", ft.RangeFilter.Min)
		} else {
			query = query.Where(field, ">", ft.RangeFilter.Min)
		}
		if ft.RangeFilter.IncludeMax {
			query = query.Where(field, "<=", ft.RangeFilter.Max)
		} else {
			query = query.Where(field, "<", ft.RangeFilter.Max)
		}
	}

	return query
}

// applySearch applies search logic (basic implementation for Firestore)
func (f *FirestoreOperations) applySearch(query firestore.Query, search *commonpb.SearchRequest) firestore.Query {
	if search == nil || search.Query == "" {
		return query
	}

	// Firestore doesn't support full-text search natively
	// For basic search, we can do prefix matching on indexed fields
	if search.Options != nil && len(search.Options.SearchFields) > 0 {
		// Apply prefix search on the first search field
		field := search.Options.SearchFields[0]
		searchTerm := strings.ToLower(search.Query)
		query = query.Where(field, ">=", searchTerm).
			Where(field, "<", searchTerm+"\uf8ff")
	}

	return query
}

// ListWithQuery provides advanced querying capabilities
func (f *FirestoreOperations) ListWithQuery(ctx context.Context, collectionName string, queryBuilder func(firestore.Query) firestore.Query) ([]map[string]any, error) {
	if collectionName == "" {
		return nil, model.NewDatabaseError("collection name is required", "MISSING_COLLECTION_NAME", 400)
	}

	query := f.client.Collection(collectionName).Query

	// Apply custom query
	if queryBuilder != nil {
		query = queryBuilder(query)
	}

	// Execute query
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to execute query: %v", err),
			"FIRESTORE_QUERY_FAILED",
			500,
		)
	}

	// Convert documents to map slice
	var results []map[string]any
	for _, doc := range docs {
		data := doc.Data()
		data["id"] = doc.Ref.ID
		results = append(results, data)
	}

	return results, nil
}

// Query executes a structured query against the collection
func (f *FirestoreOperations) Query(ctx context.Context, collectionName string, queryBuilder interfaces.QueryBuilder) ([]map[string]any, error) {
	if collectionName == "" {
		return nil, model.NewDatabaseError("collection name is required", "MISSING_COLLECTION_NAME", 400)
	}

	if queryBuilder == nil {
		return nil, model.NewDatabaseError("query builder is required", "MISSING_QUERY_BUILDER", 400)
	}

	// Build the query filter
	filter, err := queryBuilder.Build()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to build query: %v", err),
			"QUERY_BUILD_FAILED",
			400,
		)
	}

	query := f.client.Collection(collectionName).Query

	// Apply conditions
	for _, condition := range filter.Conditions {
		query = query.Where(condition.Field, condition.Operator, condition.Value)
	}

	// Apply ordering
	for _, orderBy := range filter.OrderBy {
		direction := firestore.Asc
		if !orderBy.Ascending {
			direction = firestore.Desc
		}
		query = query.OrderBy(orderBy.Field, direction)
	}

	// Apply limit
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	// Execute query
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, model.NewDatabaseError(
			fmt.Sprintf("failed to execute query: %v", err),
			"FIRESTORE_QUERY_FAILED",
			500,
		)
	}

	// Convert documents to map slice
	var results []map[string]any
	for _, doc := range docs {
		data := doc.Data()
		data["id"] = doc.Ref.ID
		results = append(results, data)
	}

	return results, nil
}

// QueryOne executes a structured query and returns the first result
func (f *FirestoreOperations) QueryOne(ctx context.Context, collectionName string, queryBuilder interfaces.QueryBuilder) (map[string]any, error) {
	// Use Query with limit 1
	limitedBuilder := queryBuilder.Limit(1)
	results, err := f.Query(ctx, collectionName, limitedBuilder)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, model.NewDatabaseError("no results found", "NO_RESULTS_FOUND", 404)
	}

	return results[0], nil
}
