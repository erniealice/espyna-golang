package integration

import (
	"context"
	"time"

	tabularpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/tabular"
	spreadsheetpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/tabular/extensions"
)

// TabularSourceProvider defines the contract for tabular data source providers.
// This interface abstracts tabular data sources such as Google Sheets, CSV files,
// databases, and other structured data providers following the hexagonal architecture pattern.
// Implementations should be stateless and thread-safe for concurrent access.
type TabularSourceProvider interface {
	// ==========================================================================
	// Lifecycle Methods
	// ==========================================================================

	// Name returns the unique identifier of the tabular provider (e.g., "googlesheets", "csv", "postgres")
	Name() string

	// Initialize sets up the tabular provider with the given configuration.
	// This should be called before any other operations.
	// Returns an error if initialization fails (e.g., invalid credentials, connection issues).
	Initialize(config *tabularpb.TabularProviderConfig) error

	// IsEnabled returns whether this provider is currently enabled and available for use.
	// A disabled provider should not be used for operations.
	IsEnabled() bool

	// IsHealthy performs a health check on the underlying data source.
	// Returns nil if the provider is healthy, or an error describing the issue.
	IsHealthy(ctx context.Context) error

	// Close cleans up provider resources and connections.
	// Should be called when the provider is no longer needed.
	Close() error

	// ==========================================================================
	// Metadata Methods
	// ==========================================================================

	// GetCapabilities returns the list of capabilities supported by this provider.
	// This allows clients to check feature availability before attempting operations.
	GetCapabilities() []tabularpb.TabularCapability

	// GetProviderType returns the type of this provider (e.g., SPREADSHEET, DATABASE, FILE).
	GetProviderType() tabularpb.TabularProviderType

	// ==========================================================================
	// Core CRUD Operations
	// ==========================================================================

	// ReadRecords reads records from a tabular data source.
	// Supports pagination, column selection, and filtering.
	ReadRecords(ctx context.Context, req *tabularpb.ReadRecordsRequest) (*tabularpb.ReadRecordsResponse, error)

	// WriteRecords writes new records to a tabular data source.
	// Returns information about the written records including any generated IDs.
	WriteRecords(ctx context.Context, req *tabularpb.WriteRecordsRequest) (*tabularpb.WriteRecordsResponse, error)

	// UpdateRecords updates existing records in a tabular data source.
	// Supports partial updates and bulk operations.
	UpdateRecords(ctx context.Context, req *tabularpb.UpdateRecordsRequest) (*tabularpb.UpdateRecordsResponse, error)

	// DeleteRecords deletes records from a tabular data source.
	// Supports deletion by ID, filter criteria, or selection.
	DeleteRecords(ctx context.Context, req *tabularpb.DeleteRecordsRequest) (*tabularpb.DeleteRecordsResponse, error)

	// SearchRecords searches for records matching specified criteria.
	// Supports full-text search, filtering, and sorting.
	SearchRecords(ctx context.Context, req *tabularpb.SearchRecordsRequest) (*tabularpb.SearchRecordsResponse, error)

	// ==========================================================================
	// Schema Operations
	// ==========================================================================

	// GetSchema retrieves the schema (column definitions) for a table.
	// Returns column names, types, constraints, and other metadata.
	GetSchema(ctx context.Context, req *tabularpb.GetSchemaRequest) (*tabularpb.GetSchemaResponse, error)

	// GetSource retrieves metadata about the data source (e.g., spreadsheet, database).
	// Returns information like name, owner, creation date, and available tables.
	GetSource(ctx context.Context, req *tabularpb.GetSourceRequest) (*tabularpb.GetSourceResponse, error)

	// ListTables lists all available tables/sheets in the data source.
	// Returns table names and basic metadata.
	ListTables(ctx context.Context, req *tabularpb.ListTablesRequest) (*tabularpb.ListTablesResponse, error)

	// ==========================================================================
	// Batch Operations
	// ==========================================================================

	// BatchExecute executes multiple operations in a single request.
	// Supports transactional semantics where the provider supports it.
	// Operations are executed in order; failure handling depends on configuration.
	BatchExecute(ctx context.Context, req *tabularpb.BatchExecuteRequest) (*tabularpb.BatchExecuteResponse, error)

	// ==========================================================================
	// Health & Capabilities (Request/Response Wrappers)
	// ==========================================================================

	// CheckHealth performs a detailed health check with structured request/response.
	// Provides more detailed health information than IsHealthy.
	CheckHealth(ctx context.Context, req *tabularpb.CheckHealthRequest) (*tabularpb.CheckHealthResponse, error)

	// GetCapabilitiesInfo returns detailed capability information with structured request/response.
	// Provides more detailed capability information than GetCapabilities.
	GetCapabilitiesInfo(ctx context.Context, req *tabularpb.GetCapabilitiesRequest) (*tabularpb.GetCapabilitiesResponse, error)
}

// SpreadsheetExtensions provides optional spreadsheet-specific operations.
// Providers that support spreadsheet functionality can implement this interface
// in addition to TabularSourceProvider for enhanced cell-level and sheet management operations.
// Use type assertion to check if a provider supports these extensions:
//
//	if ext, ok := provider.(SpreadsheetExtensions); ok {
//	    // Use spreadsheet-specific operations
//	}
type SpreadsheetExtensions interface {
	// ==========================================================================
	// Cell-Level Operations
	// ==========================================================================

	// ReadCells reads individual cells from a spreadsheet based on selection criteria.
	// Returns cell values along with formatting and metadata.
	ReadCells(ctx context.Context, selection *spreadsheetpb.SpreadsheetSelection) ([]*spreadsheetpb.SpreadsheetCell, error)

	// WriteCells writes values to individual cells in a spreadsheet.
	// Supports writing to non-contiguous cells in a single operation.
	WriteCells(ctx context.Context, cells []*spreadsheetpb.SpreadsheetCell) error

	// FormatCells applies formatting to cells within a selection.
	// Supports font, color, borders, alignment, and other formatting options.
	FormatCells(ctx context.Context, selection *spreadsheetpb.SpreadsheetSelection, format *spreadsheetpb.CellFormat) error

	// ==========================================================================
	// Sheet Management
	// ==========================================================================

	// CreateSheet creates a new sheet within a spreadsheet.
	// Returns the created table metadata including the assigned sheet ID.
	CreateSheet(ctx context.Context, sourceId string, name string, schema *tabularpb.TableSchema) (*tabularpb.Table, error)

	// DeleteSheet removes a sheet from a spreadsheet.
	// This operation is destructive and cannot be undone.
	DeleteSheet(ctx context.Context, sourceId string, name string) error

	// RenameSheet changes the name of an existing sheet.
	// The new name must be unique within the spreadsheet.
	RenameSheet(ctx context.Context, sourceId string, oldName string, newName string) error
}

// ==========================================================================
// Helper Types
// ==========================================================================

// TabularOptions contains common options for tabular operations.
// These options can be used to configure operation behavior across providers.
type TabularOptions struct {
	// Timeout specifies the maximum duration for the operation.
	// A zero value means no timeout (use provider default).
	Timeout time.Duration

	// MaxRetries specifies the maximum number of retry attempts for transient failures.
	// A zero value means no retries.
	MaxRetries int

	// IncludeSchema indicates whether to include schema information in read responses.
	// When true, column metadata is included with the data.
	IncludeSchema bool

	// ValueInputOption determines how input data should be interpreted.
	// Common values: "RAW" (literal values) or "USER_ENTERED" (parse formulas/dates).
	ValueInputOption string
}

// TabularRecord is a convenience type alias for the protobuf Record message.
// Represents a single record (row) of tabular data.
type TabularRecord = tabularpb.Record

// TabularSelection is a convenience type alias for the protobuf Selection message.
// Represents a selection of cells, rows, or columns in a tabular data source.
type TabularSelection = tabularpb.Selection

// TabularTable is a convenience type alias for the protobuf Table message.
// Represents metadata about a table/sheet in a data source.
type TabularTable = tabularpb.Table

// TabularSchema is a convenience type alias for the protobuf TableSchema message.
// Represents the schema definition for a table including column definitions.
type TabularSchema = tabularpb.TableSchema
