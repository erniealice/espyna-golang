package mock

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	tabularpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/tabular"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterTabularProvider(
		"mock_tabular",
		func() integration.TabularSourceProvider {
			return NewMockTabularProvider()
		},
		nil, // No config transformer needed for mock
	)
	registry.RegisterTabularBuildFromEnv("mock_tabular", buildFromEnv)
}

// buildFromEnv creates and initializes a mock tabular provider from environment
func buildFromEnv() (integration.TabularSourceProvider, error) {
	p := NewMockTabularProvider()
	config := &tabularpb.TabularProviderConfig{
		ProviderId:   "mock_tabular",
		ProviderType: tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_MOCK,
		Enabled:      true,
	}
	if err := p.Initialize(config); err != nil {
		return nil, fmt.Errorf("mock_tabular: failed to initialize: %w", err)
	}
	return p, nil
}

// =============================================================================
// Mock Implementation
// =============================================================================

// mockSource represents an in-memory data source
type mockSource struct {
	id     string
	name   string
	tables map[string]*mockTable
}

// mockTable represents an in-memory table
type mockTable struct {
	id      string
	name    string
	schema  *tabularpb.TableSchema
	records []*tabularpb.Record
}

// MockTabularProvider provides an in-memory tabular implementation for testing
type MockTabularProvider struct {
	mu      sync.RWMutex
	enabled bool
	config  *tabularpb.TabularProviderConfig
	sources map[string]*mockSource
}

// NewMockTabularProvider creates a new mock tabular provider
func NewMockTabularProvider() *MockTabularProvider {
	return &MockTabularProvider{
		sources: make(map[string]*mockSource),
	}
}

// =============================================================================
// Lifecycle Methods
// =============================================================================

// Name returns the unique identifier of this provider
func (p *MockTabularProvider) Name() string {
	return "mock_tabular"
}

// Initialize sets up the mock tabular provider with the given configuration
func (p *MockTabularProvider) Initialize(config *tabularpb.TabularProviderConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config
	p.enabled = config.Enabled
	log.Printf("Mock tabular provider initialized")
	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (p *MockTabularProvider) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// IsHealthy checks if the mock provider is available
func (p *MockTabularProvider) IsHealthy(ctx context.Context) error {
	return nil // Mock provider is always healthy
}

// Close cleans up mock provider resources
func (p *MockTabularProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enabled = false
	p.sources = make(map[string]*mockSource)
	log.Printf("Mock tabular provider closed")
	return nil
}

// =============================================================================
// Metadata Methods
// =============================================================================

// GetCapabilities returns the list of capabilities supported by this provider
func (p *MockTabularProvider) GetCapabilities() []tabularpb.TabularCapability {
	return []tabularpb.TabularCapability{
		tabularpb.TabularCapability_TABULAR_CAPABILITY_READ,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_WRITE,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_UPDATE,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_DELETE,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_SEARCH,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_SCHEMA,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_BATCH_OPERATIONS,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_MULTIPLE_TABLES,
	}
}

// GetProviderType returns the type of this provider
func (p *MockTabularProvider) GetProviderType() tabularpb.TabularProviderType {
	return tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_MOCK
}

// =============================================================================
// Core CRUD Operations
// =============================================================================

// ReadRecords reads records from a tabular data source
func (p *MockTabularProvider) ReadRecords(ctx context.Context, req *tabularpb.ReadRecordsRequest) (*tabularpb.ReadRecordsResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.ReadRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Mock tabular provider is not initialized",
			},
		}, nil
	}

	data := req.GetData()
	if data == nil {
		return &tabularpb.ReadRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Get the table
	table, err := p.getTable(data.SourceId, data.Selection.GetTable())
	if err != nil {
		return &tabularpb.ReadRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "TABLE_NOT_FOUND",
				Message: err.Error(),
			},
		}, nil
	}

	// Apply filters and get records
	records := p.filterRecords(table.records, data.Selection)

	// Apply sorting
	if len(data.SortBy) > 0 {
		records = p.applySort(records, data.SortBy)
	}

	// Apply pagination
	totalCount := int64(len(records))
	offset := int32(0)
	limit := int32(len(records))

	if data.Selection != nil && data.Selection.Records != nil {
		if data.Selection.Records.Offset > 0 {
			offset = data.Selection.Records.Offset
		}
		if data.Selection.Records.Limit > 0 {
			limit = data.Selection.Records.Limit
		}
	}

	// Slice records for pagination
	start := int(offset)
	end := int(offset) + int(limit)
	if start > len(records) {
		start = len(records)
	}
	if end > len(records) {
		end = len(records)
	}
	paginatedRecords := records[start:end]
	hasMore := end < len(records)

	result := &tabularpb.ReadRecordsResult{
		Records:    paginatedRecords,
		TotalCount: totalCount,
		HasMore:    hasMore,
		NextOffset: int32(end),
	}

	// Include schema if requested
	if data.IncludeSchema && table.schema != nil {
		result.Schema = table.schema
	}

	log.Printf("Mock: Read %d records from source %s table %s", len(paginatedRecords), data.SourceId, data.Selection.GetTable())

	return &tabularpb.ReadRecordsResponse{
		Success: true,
		Data:    []*tabularpb.ReadRecordsResult{result},
	}, nil
}

// WriteRecords writes new records to a tabular data source
func (p *MockTabularProvider) WriteRecords(ctx context.Context, req *tabularpb.WriteRecordsRequest) (*tabularpb.WriteRecordsResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.WriteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Mock tabular provider is not initialized",
			},
		}, nil
	}

	data := req.GetData()
	if data == nil {
		return &tabularpb.WriteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Get or create source and table
	source := p.getOrCreateSource(data.SourceId)
	table := p.getOrCreateTable(source, data.Table)

	// Determine insert position
	insertAt := len(table.records)
	if data.InsertAt >= 0 && int(data.InsertAt) < len(table.records) {
		insertAt = int(data.InsertAt)
	}

	// Assign indices to new records
	for i, record := range data.Records {
		record.Index = int64(insertAt + i)
		if record.Id == "" {
			record.Id = fmt.Sprintf("rec_%d", insertAt+i)
		}
	}

	// Insert records
	if insertAt == len(table.records) {
		// Append at end
		table.records = append(table.records, data.Records...)
	} else {
		// Insert at position
		newRecords := make([]*tabularpb.Record, 0, len(table.records)+len(data.Records))
		newRecords = append(newRecords, table.records[:insertAt]...)
		newRecords = append(newRecords, data.Records...)
		newRecords = append(newRecords, table.records[insertAt:]...)
		table.records = newRecords

		// Update indices for shifted records
		for i := insertAt + len(data.Records); i < len(table.records); i++ {
			table.records[i].Index = int64(i)
		}
	}

	log.Printf("Mock: Wrote %d records to source %s table %s", len(data.Records), data.SourceId, data.Table)

	result := &tabularpb.WriteRecordsResult{
		RecordsWritten: int32(len(data.Records)),
		Location:       fmt.Sprintf("%s/%s", data.SourceId, data.Table),
	}

	// Return written records if requested
	if data.Options != nil && data.Options.ReturnRecords {
		result.WrittenRecords = data.Records
	}

	return &tabularpb.WriteRecordsResponse{
		Success: true,
		Data:    []*tabularpb.WriteRecordsResult{result},
	}, nil
}

// UpdateRecords updates existing records in a tabular data source
func (p *MockTabularProvider) UpdateRecords(ctx context.Context, req *tabularpb.UpdateRecordsRequest) (*tabularpb.UpdateRecordsResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.UpdateRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Mock tabular provider is not initialized",
			},
		}, nil
	}

	data := req.GetData()
	if data == nil {
		return &tabularpb.UpdateRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Get the table
	table, err := p.getTable(data.SourceId, data.Selection.GetTable())
	if err != nil {
		return &tabularpb.UpdateRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "TABLE_NOT_FOUND",
				Message: err.Error(),
			},
		}, nil
	}

	// Find matching records
	matchingIndices := p.findMatchingIndices(table.records, data.Selection)
	recordsMatched := int32(len(matchingIndices))
	recordsUpdated := int32(0)

	// Apply updates
	for _, idx := range matchingIndices {
		if idx >= 0 && idx < len(table.records) {
			record := table.records[idx]

			// Apply field updates
			for _, update := range data.Updates {
				if update.Value != nil {
					switch field := update.Field.(type) {
					case *tabularpb.FieldUpdate_FieldIndex:
						if int(field.FieldIndex) < len(record.Values) {
							record.Values[field.FieldIndex] = update.Value
						}
					case *tabularpb.FieldUpdate_FieldName:
						if record.NamedValues == nil {
							record.NamedValues = make(map[string]*tabularpb.FieldValue)
						}
						record.NamedValues[field.FieldName] = update.Value
					}
				}
			}
			recordsUpdated++
		}
	}

	log.Printf("Mock: Updated %d records in source %s table %s", recordsUpdated, data.SourceId, data.Selection.GetTable())

	return &tabularpb.UpdateRecordsResponse{
		Success: true,
		Data: []*tabularpb.UpdateRecordsResult{
			{
				RecordsUpdated: recordsUpdated,
				RecordsMatched: recordsMatched,
			},
		},
	}, nil
}

// DeleteRecords deletes records from a tabular data source
func (p *MockTabularProvider) DeleteRecords(ctx context.Context, req *tabularpb.DeleteRecordsRequest) (*tabularpb.DeleteRecordsResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Mock tabular provider is not initialized",
			},
		}, nil
	}

	data := req.GetData()
	if data == nil {
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Get the table
	table, err := p.getTable(data.SourceId, data.Selection.GetTable())
	if err != nil {
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "TABLE_NOT_FOUND",
				Message: err.Error(),
			},
		}, nil
	}

	// Find matching records to delete
	matchingIndices := p.findMatchingIndices(table.records, data.Selection)
	recordsDeleted := int32(0)

	// Delete records in reverse order to maintain index integrity
	sort.Sort(sort.Reverse(sort.IntSlice(matchingIndices)))
	for _, idx := range matchingIndices {
		if idx >= 0 && idx < len(table.records) {
			table.records = append(table.records[:idx], table.records[idx+1:]...)
			recordsDeleted++
		}
	}

	// Reindex remaining records if shift_remaining is true
	if data.ShiftRemaining {
		for i := range table.records {
			table.records[i].Index = int64(i)
		}
	}

	log.Printf("Mock: Deleted %d records from source %s table %s", recordsDeleted, data.SourceId, data.Selection.GetTable())

	return &tabularpb.DeleteRecordsResponse{
		Success: true,
		Data: []*tabularpb.DeleteRecordsResult{
			{
				RecordsDeleted: recordsDeleted,
			},
		},
	}, nil
}

// SearchRecords searches for records matching specified criteria
func (p *MockTabularProvider) SearchRecords(ctx context.Context, req *tabularpb.SearchRecordsRequest) (*tabularpb.SearchRecordsResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.SearchRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Mock tabular provider is not initialized",
			},
		}, nil
	}

	data := req.GetData()
	if data == nil {
		return &tabularpb.SearchRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Get the table
	table, err := p.getTable(data.SourceId, data.Table)
	if err != nil {
		return &tabularpb.SearchRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "TABLE_NOT_FOUND",
				Message: err.Error(),
			},
		}, nil
	}

	// Apply filter
	var records []*tabularpb.Record
	if data.Filter != nil {
		for _, record := range table.records {
			if p.matchesFilter(record, data.Filter) {
				records = append(records, record)
			}
		}
	} else {
		records = table.records
	}

	// Apply sorting
	if len(data.SortBy) > 0 {
		records = p.applySort(records, data.SortBy)
	}

	// Apply pagination
	totalCount := int64(len(records))
	start := int(data.Offset)
	end := len(records)
	if data.Limit > 0 {
		end = start + int(data.Limit)
	}
	if start > len(records) {
		start = len(records)
	}
	if end > len(records) {
		end = len(records)
	}
	paginatedRecords := records[start:end]
	hasMore := end < len(records)

	log.Printf("Mock: Search found %d records in source %s table %s", len(paginatedRecords), data.SourceId, data.Table)

	return &tabularpb.SearchRecordsResponse{
		Success: true,
		Data: []*tabularpb.SearchRecordsResult{
			{
				Records:    paginatedRecords,
				TotalCount: totalCount,
				HasMore:    hasMore,
			},
		},
	}, nil
}

// =============================================================================
// Schema Operations
// =============================================================================

// GetSchema retrieves the schema for a table
func (p *MockTabularProvider) GetSchema(ctx context.Context, req *tabularpb.GetSchemaRequest) (*tabularpb.GetSchemaResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.GetSchemaResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Mock tabular provider is not initialized",
			},
		}, nil
	}

	data := req.GetData()
	if data == nil {
		return &tabularpb.GetSchemaResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	source, exists := p.sources[data.SourceId]
	if !exists {
		return &tabularpb.GetSchemaResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "SOURCE_NOT_FOUND",
				Message: fmt.Sprintf("Source %s not found", data.SourceId),
			},
		}, nil
	}

	result := &tabularpb.GetSchemaResult{
		Source: &tabularpb.Source{
			Id:           source.id,
			Name:         source.name,
			ProviderType: tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_MOCK,
		},
	}

	// Get specific table schema if table name is provided
	if data.Table != "" {
		table, exists := source.tables[data.Table]
		if !exists {
			return &tabularpb.GetSchemaResponse{
				Success: false,
				Error: &commonpb.Error{
					Code:    "TABLE_NOT_FOUND",
					Message: fmt.Sprintf("Table %s not found in source %s", data.Table, data.SourceId),
				},
			}, nil
		}
		result.TableSchema = table.schema
	}

	log.Printf("Mock: Got schema for source %s table %s", data.SourceId, data.Table)

	return &tabularpb.GetSchemaResponse{
		Success: true,
		Data:    []*tabularpb.GetSchemaResult{result},
	}, nil
}

// GetSource retrieves metadata about the data source
func (p *MockTabularProvider) GetSource(ctx context.Context, req *tabularpb.GetSourceRequest) (*tabularpb.GetSourceResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.GetSourceResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Mock tabular provider is not initialized",
			},
		}, nil
	}

	data := req.GetData()
	if data == nil {
		return &tabularpb.GetSourceResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	source, exists := p.sources[data.SourceId]
	if !exists {
		// Return a mock source if it doesn't exist
		source = &mockSource{
			id:     data.SourceId,
			name:   "Mock Source",
			tables: make(map[string]*mockTable),
		}
	}

	result := &tabularpb.Source{
		Id:           source.id,
		Name:         source.name,
		ProviderType: tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_MOCK,
	}

	// Include tables if requested
	if data.IncludeTables {
		for _, table := range source.tables {
			result.Tables = append(result.Tables, &tabularpb.Table{
				Id:     table.id,
				Name:   table.name,
				Schema: table.schema,
			})
		}
	}

	log.Printf("Mock: Got source %s", data.SourceId)

	return &tabularpb.GetSourceResponse{
		Success: true,
		Data:    []*tabularpb.Source{result},
	}, nil
}

// ListTables lists all available tables in the data source
func (p *MockTabularProvider) ListTables(ctx context.Context, req *tabularpb.ListTablesRequest) (*tabularpb.ListTablesResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.ListTablesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Mock tabular provider is not initialized",
			},
		}, nil
	}

	data := req.GetData()
	if data == nil {
		return &tabularpb.ListTablesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	source, exists := p.sources[data.SourceId]
	if !exists {
		return &tabularpb.ListTablesResponse{
			Success: true,
			Data:    []*tabularpb.Table{},
		}, nil
	}

	var tables []*tabularpb.Table
	for _, table := range source.tables {
		tables = append(tables, &tabularpb.Table{
			Id:     table.id,
			Name:   table.name,
			Schema: table.schema,
		})
	}

	log.Printf("Mock: Listed %d tables in source %s", len(tables), data.SourceId)

	return &tabularpb.ListTablesResponse{
		Success: true,
		Data:    tables,
	}, nil
}

// =============================================================================
// Batch Operations
// =============================================================================

// BatchExecute executes multiple operations in a single request
func (p *MockTabularProvider) BatchExecute(ctx context.Context, req *tabularpb.BatchExecuteRequest) (*tabularpb.BatchExecuteResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.BatchExecuteResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Mock tabular provider is not initialized",
			},
		}, nil
	}

	data := req.GetData()
	if data == nil {
		return &tabularpb.BatchExecuteResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	var results []*tabularpb.BatchOperationResult
	successCount := int32(0)
	failureCount := int32(0)

	for _, op := range data.Operations {
		opResult := &tabularpb.BatchOperationResult{
			OperationId: op.OperationId,
			Success:     true,
		}

		var opErr error

		switch opData := op.Operation.(type) {
		case *tabularpb.BatchOperation_Write:
			_, opErr = p.WriteRecords(ctx, &tabularpb.WriteRecordsRequest{Data: opData.Write})
		case *tabularpb.BatchOperation_Update:
			_, opErr = p.UpdateRecords(ctx, &tabularpb.UpdateRecordsRequest{Data: opData.Update})
		case *tabularpb.BatchOperation_Delete:
			_, opErr = p.DeleteRecords(ctx, &tabularpb.DeleteRecordsRequest{Data: opData.Delete})
		default:
			opErr = fmt.Errorf("unknown operation type")
		}

		if opErr != nil {
			opResult.Success = false
			opResult.Error = &commonpb.Error{
				Code:    "OPERATION_FAILED",
				Message: opErr.Error(),
			}
			failureCount++

			if data.FailFast {
				results = append(results, opResult)
				break
			}
		} else {
			successCount++
		}

		results = append(results, opResult)
	}

	log.Printf("Mock: Batch executed %d operations (%d success, %d failures)", len(data.Operations), successCount, failureCount)

	return &tabularpb.BatchExecuteResponse{
		Success: failureCount == 0,
		Data: []*tabularpb.BatchExecuteResult{
			{
				SuccessCount: successCount,
				FailureCount: failureCount,
				Results:      results,
			},
		},
	}, nil
}

// =============================================================================
// Health & Capabilities (Request/Response Wrappers)
// =============================================================================

// CheckHealth performs a detailed health check
func (p *MockTabularProvider) CheckHealth(ctx context.Context, req *tabularpb.CheckHealthRequest) (*tabularpb.CheckHealthResponse, error) {
	return &tabularpb.CheckHealthResponse{
		Success: true,
		Data: []*tabularpb.HealthStatus{
			{
				IsHealthy: true,
				Message:   "Mock tabular provider is healthy",
				Details: map[string]string{
					"provider": "mock_tabular",
					"status":   "operational",
				},
			},
		},
	}, nil
}

// GetCapabilitiesInfo returns detailed capability information
func (p *MockTabularProvider) GetCapabilitiesInfo(ctx context.Context, req *tabularpb.GetCapabilitiesRequest) (*tabularpb.GetCapabilitiesResponse, error) {
	capabilities := p.GetCapabilities()

	return &tabularpb.GetCapabilitiesResponse{
		Success: true,
		Data: []*tabularpb.ProviderCapabilities{
			{
				ProviderId:           "mock_tabular",
				ProviderType:         tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_MOCK,
				Capabilities:         capabilities,
				MaxRecordsPerRequest: 10000,
				MaxFieldsPerRecord:   1000,
				MaxSourceSizeBytes:   100 * 1024 * 1024, // 100MB
			},
		},
	}, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// getOrCreateSource gets an existing source or creates a new one
func (p *MockTabularProvider) getOrCreateSource(sourceId string) *mockSource {
	source, exists := p.sources[sourceId]
	if !exists {
		source = &mockSource{
			id:     sourceId,
			name:   "Mock Source " + sourceId,
			tables: make(map[string]*mockTable),
		}
		p.sources[sourceId] = source
	}
	return source
}

// getOrCreateTable gets an existing table or creates a new one
func (p *MockTabularProvider) getOrCreateTable(source *mockSource, tableName string) *mockTable {
	if tableName == "" {
		tableName = "default"
	}
	table, exists := source.tables[tableName]
	if !exists {
		table = &mockTable{
			id:      tableName,
			name:    tableName,
			schema:  nil,
			records: []*tabularpb.Record{},
		}
		source.tables[tableName] = table
	}
	return table
}

// getTable retrieves a table from a source
func (p *MockTabularProvider) getTable(sourceId, tableName string) (*mockTable, error) {
	source, exists := p.sources[sourceId]
	if !exists {
		return nil, fmt.Errorf("source %s not found", sourceId)
	}

	if tableName == "" {
		tableName = "default"
	}

	table, exists := source.tables[tableName]
	if !exists {
		// Auto-create table for convenience in mock
		table = &mockTable{
			id:      tableName,
			name:    tableName,
			schema:  nil,
			records: []*tabularpb.Record{},
		}
		source.tables[tableName] = table
	}
	return table, nil
}

// filterRecords filters records based on selection criteria
func (p *MockTabularProvider) filterRecords(records []*tabularpb.Record, selection *tabularpb.Selection) []*tabularpb.Record {
	if selection == nil || selection.Records == nil {
		return records
	}

	var result []*tabularpb.Record

	// Filter by index range
	if selection.Records.IndexRange != nil {
		start := selection.Records.IndexRange.Start
		end := selection.Records.IndexRange.End
		if end == -1 {
			end = int64(len(records))
		}
		for i, record := range records {
			if int64(i) >= start && int64(i) < end {
				result = append(result, record)
			}
		}
		return result
	}

	// Filter by record IDs
	if len(selection.Records.RecordIds) > 0 {
		idSet := make(map[string]bool)
		for _, id := range selection.Records.RecordIds {
			idSet[id] = true
		}
		for _, record := range records {
			if idSet[record.Id] {
				result = append(result, record)
			}
		}
		return result
	}

	// Filter by filter conditions
	if selection.Records.Filter != nil {
		for _, record := range records {
			if p.matchesFilter(record, selection.Records.Filter) {
				result = append(result, record)
			}
		}
		return result
	}

	return records
}

// findMatchingIndices finds indices of records matching selection criteria
func (p *MockTabularProvider) findMatchingIndices(records []*tabularpb.Record, selection *tabularpb.Selection) []int {
	var indices []int

	if selection == nil || selection.Records == nil {
		for i := range records {
			indices = append(indices, i)
		}
		return indices
	}

	// Match by index range
	if selection.Records.IndexRange != nil {
		start := int(selection.Records.IndexRange.Start)
		end := int(selection.Records.IndexRange.End)
		if end == -1 {
			end = len(records)
		}
		for i := start; i < end && i < len(records); i++ {
			indices = append(indices, i)
		}
		return indices
	}

	// Match by record IDs
	if len(selection.Records.RecordIds) > 0 {
		idSet := make(map[string]bool)
		for _, id := range selection.Records.RecordIds {
			idSet[id] = true
		}
		for i, record := range records {
			if idSet[record.Id] {
				indices = append(indices, i)
			}
		}
		return indices
	}

	// Match by filter conditions
	if selection.Records.Filter != nil {
		for i, record := range records {
			if p.matchesFilter(record, selection.Records.Filter) {
				indices = append(indices, i)
			}
		}
		return indices
	}

	return indices
}

// matchesFilter checks if a record matches filter conditions
func (p *MockTabularProvider) matchesFilter(record *tabularpb.Record, filter *tabularpb.FilterGroup) bool {
	if filter == nil {
		return true
	}

	if len(filter.Filters) == 0 && len(filter.Groups) == 0 {
		return true
	}

	// Check individual filters
	filterResults := make([]bool, 0, len(filter.Filters)+len(filter.Groups))

	for _, f := range filter.Filters {
		filterResults = append(filterResults, p.matchesSingleFilter(record, f))
	}

	// Check nested groups
	for _, group := range filter.Groups {
		filterResults = append(filterResults, p.matchesFilter(record, group))
	}

	// Combine results based on logical operator
	if len(filterResults) == 0 {
		return true
	}

	switch filter.Operator {
	case tabularpb.LogicalOperator_LOGICAL_OPERATOR_OR:
		for _, result := range filterResults {
			if result {
				return true
			}
		}
		return false
	default: // AND is default
		for _, result := range filterResults {
			if !result {
				return false
			}
		}
		return true
	}
}

// matchesSingleFilter checks if a record matches a single filter condition
func (p *MockTabularProvider) matchesSingleFilter(record *tabularpb.Record, filter *tabularpb.Filter) bool {
	var fieldValue *tabularpb.FieldValue

	// Get field value
	switch field := filter.Field.(type) {
	case *tabularpb.Filter_FieldIndex:
		if int(field.FieldIndex) < len(record.Values) {
			fieldValue = record.Values[field.FieldIndex]
		}
	case *tabularpb.Filter_FieldName:
		if record.NamedValues != nil {
			fieldValue = record.NamedValues[field.FieldName]
		}
	}

	// Handle null checks
	switch filter.Operator {
	case tabularpb.FilterOperator_FILTER_OPERATOR_IS_NULL:
		return fieldValue == nil
	case tabularpb.FilterOperator_FILTER_OPERATOR_IS_NOT_NULL:
		return fieldValue != nil
	}

	if fieldValue == nil || filter.Value == nil {
		return false
	}

	// Get string representations for comparison
	recordStr := getStringValue(fieldValue)
	filterStr := getStringValue(filter.Value)

	// Apply operator
	switch filter.Operator {
	case tabularpb.FilterOperator_FILTER_OPERATOR_EQUALS:
		if filter.CaseSensitive {
			return recordStr == filterStr
		}
		return strings.EqualFold(recordStr, filterStr)
	case tabularpb.FilterOperator_FILTER_OPERATOR_NOT_EQUALS:
		if filter.CaseSensitive {
			return recordStr != filterStr
		}
		return !strings.EqualFold(recordStr, filterStr)
	case tabularpb.FilterOperator_FILTER_OPERATOR_CONTAINS:
		if filter.CaseSensitive {
			return strings.Contains(recordStr, filterStr)
		}
		return strings.Contains(strings.ToLower(recordStr), strings.ToLower(filterStr))
	case tabularpb.FilterOperator_FILTER_OPERATOR_NOT_CONTAINS:
		if filter.CaseSensitive {
			return !strings.Contains(recordStr, filterStr)
		}
		return !strings.Contains(strings.ToLower(recordStr), strings.ToLower(filterStr))
	case tabularpb.FilterOperator_FILTER_OPERATOR_STARTS_WITH:
		if filter.CaseSensitive {
			return strings.HasPrefix(recordStr, filterStr)
		}
		return strings.HasPrefix(strings.ToLower(recordStr), strings.ToLower(filterStr))
	case tabularpb.FilterOperator_FILTER_OPERATOR_ENDS_WITH:
		if filter.CaseSensitive {
			return strings.HasSuffix(recordStr, filterStr)
		}
		return strings.HasSuffix(strings.ToLower(recordStr), strings.ToLower(filterStr))
	case tabularpb.FilterOperator_FILTER_OPERATOR_IS_EMPTY:
		return recordStr == ""
	case tabularpb.FilterOperator_FILTER_OPERATOR_IS_NOT_EMPTY:
		return recordStr != ""
	case tabularpb.FilterOperator_FILTER_OPERATOR_IN:
		for _, v := range filter.Values {
			if strings.EqualFold(recordStr, getStringValue(v)) {
				return true
			}
		}
		return false
	case tabularpb.FilterOperator_FILTER_OPERATOR_NOT_IN:
		for _, v := range filter.Values {
			if strings.EqualFold(recordStr, getStringValue(v)) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

// applySort sorts records based on sort specifications
func (p *MockTabularProvider) applySort(records []*tabularpb.Record, sortSpecs []*tabularpb.SortSpec) []*tabularpb.Record {
	if len(sortSpecs) == 0 {
		return records
	}

	// Make a copy to avoid modifying original
	result := make([]*tabularpb.Record, len(records))
	copy(result, records)

	sort.SliceStable(result, func(i, j int) bool {
		for _, spec := range sortSpecs {
			var valI, valJ string

			switch field := spec.Field.(type) {
			case *tabularpb.SortSpec_FieldIndex:
				if int(field.FieldIndex) < len(result[i].Values) {
					valI = getStringValue(result[i].Values[field.FieldIndex])
				}
				if int(field.FieldIndex) < len(result[j].Values) {
					valJ = getStringValue(result[j].Values[field.FieldIndex])
				}
			case *tabularpb.SortSpec_FieldName:
				if result[i].NamedValues != nil {
					valI = getStringValue(result[i].NamedValues[field.FieldName])
				}
				if result[j].NamedValues != nil {
					valJ = getStringValue(result[j].NamedValues[field.FieldName])
				}
			}

			if valI != valJ {
				if spec.Direction == tabularpb.SortDirection_SORT_DIRECTION_DESCENDING {
					return valI > valJ
				}
				return valI < valJ
			}
		}
		return false
	})

	return result
}

// getStringValue extracts a string value from a FieldValue
func getStringValue(fv *tabularpb.FieldValue) string {
	if fv == nil {
		return ""
	}

	switch v := fv.Value.(type) {
	case *tabularpb.FieldValue_StringValue:
		return v.StringValue
	case *tabularpb.FieldValue_IntegerValue:
		return fmt.Sprintf("%d", v.IntegerValue)
	case *tabularpb.FieldValue_FloatValue:
		return fmt.Sprintf("%f", v.FloatValue)
	case *tabularpb.FieldValue_BooleanValue:
		return fmt.Sprintf("%t", v.BooleanValue)
	case *tabularpb.FieldValue_DateValue:
		return v.DateValue
	case *tabularpb.FieldValue_DatetimeValue:
		return v.DatetimeValue
	case *tabularpb.FieldValue_FormulaValue:
		return v.FormulaValue
	case *tabularpb.FieldValue_ErrorValue:
		return v.ErrorValue
	default:
		if fv.DisplayValue != "" {
			return fv.DisplayValue
		}
		if fv.RawValue != "" {
			return fv.RawValue
		}
		return ""
	}
}
