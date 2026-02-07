//go:build mock_db

package core

import (
	"context"
	"fmt"
	"time"

	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"

	interfaces "leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/interface"
	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/common/model"
	"leapfor.xyz/espyna/internal/infrastructure/registry"
)

func init() {
	// Register database operations factory for mock_db
	registry.RegisterDatabaseOperationsFactory("mock_db", func(conn any) (any, error) {
		// Mock operations accepts the data store as connection
		// If nil, create an empty data store
		if data, ok := conn.(map[string]map[string]map[string]any); ok {
			return NewMockOperations(data), nil
		}
		return NewMockOperations(nil), nil
	})
}

// MockOperations implements DatabaseOperation for mock data
type MockOperations struct {
	data map[string]map[string]map[string]any // businessType -> table -> id -> record
}

// NewMockOperations creates a new mock operations instance
func NewMockOperations(initialData map[string]map[string]map[string]any) interfaces.DatabaseOperation {
	if initialData == nil {
		initialData = make(map[string]map[string]map[string]any)
	}
	return &MockOperations{
		data: initialData,
	}
}

// Create creates a new record in the mock data store
func (m *MockOperations) Create(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	businessType := "default" // Mock operations are not business-type aware in this version
	if _, exists := m.data[businessType]; !exists {
		m.data[businessType] = make(map[string]map[string]any)
	}
	if _, exists := m.data[businessType][tableName]; !exists {
		m.data[businessType][tableName] = make(map[string]any)
	}

	id, ok := data["id"].(string)
	if !ok || id == "" {
		id = fmt.Sprintf("mock-%d", time.Now().UnixNano())
		data["id"] = id
	}

	m.data[businessType][tableName][id] = data
	return data, nil
}

// Read retrieves a record by ID from the mock data store
func (m *MockOperations) Read(ctx context.Context, tableName string, id string) (map[string]any, error) {
	businessType := "default"
	if table, exists := m.data[businessType][tableName]; exists {
		if record, exists := table[id]; exists {
			if recordMap, ok := record.(map[string]any); ok {
				return recordMap, nil
			}
			return nil, model.NewDatabaseError("invalid record format", "INVALID_RECORD_FORMAT", 500)
		}
	}
	return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
}

// Update updates an existing record in the mock data store
func (m *MockOperations) Update(ctx context.Context, tableName string, id string, data map[string]any) (map[string]any, error) {
	businessType := "default"
	if table, exists := m.data[businessType][tableName]; exists {
		if record, exists := table[id]; exists {
			if recordMap, ok := record.(map[string]any); ok {
				for k, v := range data {
					recordMap[k] = v
				}
				return recordMap, nil
			}
			return nil, model.NewDatabaseError("invalid record format", "INVALID_RECORD_FORMAT", 500)
		}
	}
	return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
}

// Delete removes a record from the mock data store
func (m *MockOperations) Delete(ctx context.Context, tableName string, id string) error {
	businessType := "default"
	if table, exists := m.data[businessType][tableName]; exists {
		if _, exists := table[id]; exists {
			delete(table, id)
			return nil
		}
	}
	return model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
}

// List retrieves all records from a table in the mock data store with pagination support
func (m *MockOperations) List(ctx context.Context, tableName string, params *interfaces.ListParams) (*interfaces.ListResult, error) {
	businessType := "default"
	var results []map[string]any

	// Collect all records from the table
	if table, exists := m.data[businessType][tableName]; exists {
		for _, record := range table {
			if recordMap, ok := record.(map[string]any); ok {
				// Basic filtering support (simplified for mock)
				// In a full implementation, we would parse the TypedFilter oneof fields
				// For mock purposes, we skip complex filtering and return all records
				if params != nil && params.Filters != nil && len(params.Filters.Filters) > 0 {
					// Simplified: just check if filters exist but don't apply them
					// This is acceptable for mock data where we control the test data
				}
				results = append(results, recordMap)
			}
		}
	}

	// Calculate total before pagination
	total := int32(len(results))

	// Build pagination response with defaults
	paginationResponse := &commonpb.PaginationResponse{
		TotalItems: total,
		HasNext:    false,
		HasPrev:    false,
	}

	// Apply pagination if provided
	if params != nil && params.Pagination != nil {
		limit := params.Pagination.Limit

		// Handle offset-based pagination
		if offsetPagination := params.Pagination.GetOffset(); offsetPagination != nil && limit > 0 {
			page := offsetPagination.Page
			if page < 1 {
				page = 1
			}

			// Calculate offset from page number (1-based)
			offset := (page - 1) * limit
			start := int(offset)
			end := int(offset + limit)

			if start > len(results) {
				results = []map[string]any{}
			} else {
				if end > len(results) {
					end = len(results)
				}
				results = results[start:end]
			}

			// Calculate pagination metadata
			totalPages := (total + limit - 1) / limit
			paginationResponse.CurrentPage = &page
			paginationResponse.TotalPages = &totalPages
			paginationResponse.HasNext = page < totalPages
			paginationResponse.HasPrev = page > 1
		} else if limit > 0 {
			// Simple limit without offset (treat as page 1)
			if limit < int32(len(results)) {
				results = results[:limit]
			}
			page := int32(1)
			totalPages := (total + limit - 1) / limit
			paginationResponse.CurrentPage = &page
			paginationResponse.TotalPages = &totalPages
			paginationResponse.HasNext = total > limit
		}
	}

	return &interfaces.ListResult{
		Data:       results,
		Pagination: paginationResponse,
		Total:      total,
	}, nil
}

// Query is a simplified implementation for mock operations
func (m *MockOperations) Query(ctx context.Context, tableName string, queryBuilder interfaces.QueryBuilder) ([]map[string]any, error) {
	// This is a simplified implementation and does not support complex queries
	result, err := m.List(ctx, tableName, nil)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// QueryOne is a simplified implementation for mock operations
func (m *MockOperations) QueryOne(ctx context.Context, tableName string, queryBuilder interfaces.QueryBuilder) (map[string]any, error) {
	// This is a simplified implementation and does not support complex queries
	result, err := m.List(ctx, tableName, nil)
	if err != nil {
		return nil, err
	}
	if len(result.Data) > 0 {
		return result.Data[0], nil
	}
	return nil, model.NewDatabaseError("record not found", "RECORD_NOT_FOUND", 404)
}
