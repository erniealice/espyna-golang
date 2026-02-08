//go:build google && googlesheets

package googlesheets

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/sheets/v4"

	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/common/google"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	tabularpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/tabular"
)

// =============================================================================
// Self-Registration - Adapter registers itself with the factory
// =============================================================================

func init() {
	registry.RegisterTabularProvider(
		"googlesheets",
		func() integration.TabularSourceProvider {
			return NewGoogleSheetsProvider()
		},
		transformConfig,
	)
	registry.RegisterTabularBuildFromEnv("googlesheets", buildFromEnv)
}

// buildFromEnv creates and initializes a Google Sheets provider from environment variables
func buildFromEnv() (integration.TabularSourceProvider, error) {
	delegateEmail := os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_DELEGATE_EMAIL")
	serviceAccountKeyPath := os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_SERVICE_ACCOUNT_KEY_PATH")
	projectID := os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_PROJECT_ID")
	secretManagerPath := os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_SECRET_MANAGER_PATH")
	useSecretManager := os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_USE_SECRET_MANAGER") == "true"

	timeoutStr := os.Getenv("LEAPFOR_INTEGRATION_TABULAR_GOOGLESHEETS_TIMEOUT")
	timeout := 30
	if timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil {
			timeout = t
		}
	}

	// Build GoogleSheetsAuth
	auth := &tabularpb.GoogleSheetsAuth{
		DelegatedEmail:    delegateEmail,
		ServiceAccountKey: serviceAccountKeyPath,
		ProjectId:         projectID,
		UseSecretManager:  useSecretManager,
		SecretManagerPath: secretManagerPath,
	}

	config := &tabularpb.TabularProviderConfig{
		ProviderId:     "googlesheets",
		ProviderType:   tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_GOOGLE_SHEETS,
		Enabled:        true,
		TimeoutSeconds: int32(timeout),
		Auth: &tabularpb.TabularProviderConfig_GoogleSheetsAuth{
			GoogleSheetsAuth: auth,
		},
	}

	p := NewGoogleSheetsProvider()
	if err := p.Initialize(config); err != nil {
		return nil, fmt.Errorf("googlesheets: failed to initialize: %w", err)
	}
	return p, nil
}

// transformConfig transforms raw config map to TabularProviderConfig
func transformConfig(rawConfig map[string]any) (*tabularpb.TabularProviderConfig, error) {
	config := &tabularpb.TabularProviderConfig{
		ProviderId:   "googlesheets",
		ProviderType: tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_GOOGLE_SHEETS,
		Enabled:      true,
	}

	// Extract auth settings
	auth := &tabularpb.GoogleSheetsAuth{}

	if delegateEmail, ok := rawConfig["delegate_email"].(string); ok {
		auth.DelegatedEmail = delegateEmail
	}
	if serviceAccountKey, ok := rawConfig["service_account_key"].(string); ok {
		auth.ServiceAccountKey = serviceAccountKey
	}
	if projectID, ok := rawConfig["project_id"].(string); ok {
		auth.ProjectId = projectID
	}
	if useSecretManager, ok := rawConfig["use_secret_manager"].(bool); ok {
		auth.UseSecretManager = useSecretManager
	}
	if secretManagerPath, ok := rawConfig["secret_manager_path"].(string); ok {
		auth.SecretManagerPath = secretManagerPath
	}

	config.Auth = &tabularpb.TabularProviderConfig_GoogleSheetsAuth{
		GoogleSheetsAuth: auth,
	}

	// Extract timeout
	if timeout, ok := rawConfig["timeout_seconds"].(int); ok {
		config.TimeoutSeconds = int32(timeout)
	} else if timeout, ok := rawConfig["timeout_seconds"].(float64); ok {
		config.TimeoutSeconds = int32(timeout)
	} else {
		config.TimeoutSeconds = 30
	}

	return config, nil
}

// =============================================================================
// Google Sheets Provider Implementation
// =============================================================================

// GoogleSheetsProvider provides Google Sheets as a tabular data source
type GoogleSheetsProvider struct {
	mu            sync.RWMutex
	enabled       bool
	config        *tabularpb.TabularProviderConfig
	clientManager *google.SheetsClientManager
	timeout       time.Duration
	logger        *slog.Logger
}

// NewGoogleSheetsProvider creates a new Google Sheets tabular provider
func NewGoogleSheetsProvider() *GoogleSheetsProvider {
	return &GoogleSheetsProvider{
		timeout: 30 * time.Second,
		logger:  slog.Default().With("provider", "googlesheets"),
	}
}

// =============================================================================
// Lifecycle Methods
// =============================================================================

// Name returns the unique identifier of this provider
func (p *GoogleSheetsProvider) Name() string {
	return "googlesheets"
}

// Initialize sets up the Google Sheets provider with the given configuration
func (p *GoogleSheetsProvider) Initialize(config *tabularpb.TabularProviderConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config

	// Extract Google Sheets auth config
	gsAuth := config.GetGoogleSheetsAuth()
	if gsAuth == nil {
		return fmt.Errorf("googlesheets: google_sheets_auth configuration is required")
	}

	// Set timeout
	if config.TimeoutSeconds > 0 {
		p.timeout = time.Duration(config.TimeoutSeconds) * time.Second
	}

	// Create SheetsConfig for the client manager
	sheetsConfig := &google.SheetsConfig{
		ProjectID:             gsAuth.ProjectId,
		DelegateEmail:         gsAuth.DelegatedEmail,
		ServiceAccountKeyPath: gsAuth.ServiceAccountKey,
		SecretManagerPath:     gsAuth.SecretManagerPath,
		UseSecretManager:      gsAuth.UseSecretManager,
		Timeout:               p.timeout,
	}

	// Initialize the client manager
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	clientManager, err := google.NewSheetsClientManager(ctx, sheetsConfig)
	if err != nil {
		return fmt.Errorf("googlesheets: failed to create client manager: %w", err)
	}

	p.clientManager = clientManager
	p.enabled = config.Enabled

	p.logger.Info("Google Sheets tabular provider initialized",
		"project_id", gsAuth.ProjectId,
		"delegate_email", gsAuth.DelegatedEmail,
	)

	return nil
}

// IsEnabled returns whether this provider is currently enabled
func (p *GoogleSheetsProvider) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// IsHealthy checks if the Google Sheets provider is available
func (p *GoogleSheetsProvider) IsHealthy(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.enabled {
		return fmt.Errorf("googlesheets: provider is not enabled")
	}

	if p.clientManager == nil {
		return fmt.Errorf("googlesheets: client manager is not initialized")
	}

	return nil
}

// Close cleans up Google Sheets provider resources
func (p *GoogleSheetsProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.enabled = false

	if p.clientManager != nil {
		if err := p.clientManager.Close(); err != nil {
			p.logger.Error("Failed to close client manager", "error", err)
		}
		p.clientManager = nil
	}

	p.logger.Info("Google Sheets tabular provider closed")
	return nil
}

// =============================================================================
// Metadata Methods
// =============================================================================

// GetCapabilities returns the list of capabilities supported by this provider
func (p *GoogleSheetsProvider) GetCapabilities() []tabularpb.TabularCapability {
	return []tabularpb.TabularCapability{
		tabularpb.TabularCapability_TABULAR_CAPABILITY_READ,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_WRITE,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_UPDATE,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_DELETE,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_SEARCH,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_SCHEMA,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_BATCH_OPERATIONS,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_MULTIPLE_TABLES,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_FORMULAS,
		tabularpb.TabularCapability_TABULAR_CAPABILITY_CELL_LEVEL_ACCESS,
	}
}

// GetProviderType returns the type of this provider
func (p *GoogleSheetsProvider) GetProviderType() tabularpb.TabularProviderType {
	return tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_GOOGLE_SHEETS
}

// =============================================================================
// Core CRUD Operations
// =============================================================================

// ReadRecords reads records from a Google Sheets spreadsheet
func (p *GoogleSheetsProvider) ReadRecords(ctx context.Context, req *tabularpb.ReadRecordsRequest) (*tabularpb.ReadRecordsResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.ReadRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Sheets tabular provider is not initialized",
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
	service := p.clientManager.GetService()
	p.mu.RUnlock()

	// Build A1 notation from selection
	a1Range := selectionToA1Notation(data.Selection)

	// Read from Google Sheets
	resp, err := service.Spreadsheets.Values.Get(data.SourceId, a1Range).
		ValueRenderOption("FORMATTED_VALUE").
		DateTimeRenderOption("FORMATTED_STRING").
		Context(ctx).
		Do()
	if err != nil {
		p.logger.Error("Failed to read from Google Sheets", "error", err, "source_id", data.SourceId, "range", a1Range)
		return &tabularpb.ReadRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "READ_FAILED",
				Message: fmt.Sprintf("Failed to read from Google Sheets: %v", err),
			},
		}, nil
	}

	// Convert ValueRange to records
	records := valueRangeToRecords(resp)

	// Apply sorting if requested
	if len(data.SortBy) > 0 {
		records = applySort(records, data.SortBy)
	}

	// Apply pagination from selection
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
	if data.IncludeSchema {
		schema, err := p.fetchSchema(ctx, service, data.SourceId, data.Selection.GetTable())
		if err == nil {
			result.Schema = schema
		}
	}

	p.logger.Info("Read records from Google Sheets",
		"source_id", data.SourceId,
		"range", a1Range,
		"count", len(paginatedRecords),
	)

	return &tabularpb.ReadRecordsResponse{
		Success: true,
		Data:    []*tabularpb.ReadRecordsResult{result},
	}, nil
}

// WriteRecords writes new records to a Google Sheets spreadsheet
func (p *GoogleSheetsProvider) WriteRecords(ctx context.Context, req *tabularpb.WriteRecordsRequest) (*tabularpb.WriteRecordsResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.WriteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Sheets tabular provider is not initialized",
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

	p.mu.RLock()
	service := p.clientManager.GetService()
	p.mu.RUnlock()

	// Build range for write
	tableName := data.Table
	if tableName == "" {
		tableName = "Sheet1"
	}

	// Convert records to ValueRange
	valueRange := recordsToValueRange(data.Records)

	// Determine value input option
	valueInputOption := "USER_ENTERED"
	if data.Options != nil && data.Options.ValueInputOption != "" {
		valueInputOption = data.Options.ValueInputOption
	}

	var writeResult *sheets.AppendValuesResponse
	var err error

	if data.InsertAt < 0 {
		// Append to end
		writeResult, err = service.Spreadsheets.Values.Append(data.SourceId, tableName, valueRange).
			ValueInputOption(valueInputOption).
			InsertDataOption("INSERT_ROWS").
			Context(ctx).
			Do()
	} else {
		// Update at specific position
		a1Range := fmt.Sprintf("%s!A%d", tableName, data.InsertAt+1) // Convert to 1-based
		_, err = service.Spreadsheets.Values.Update(data.SourceId, a1Range, valueRange).
			ValueInputOption(valueInputOption).
			Context(ctx).
			Do()
	}

	if err != nil {
		p.logger.Error("Failed to write to Google Sheets", "error", err, "source_id", data.SourceId)
		return &tabularpb.WriteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "WRITE_FAILED",
				Message: fmt.Sprintf("Failed to write to Google Sheets: %v", err),
			},
		}, nil
	}

	result := &tabularpb.WriteRecordsResult{
		RecordsWritten: int32(len(data.Records)),
	}

	if writeResult != nil {
		result.Location = writeResult.TableRange
	}

	// Return written records if requested
	if data.Options != nil && data.Options.ReturnRecords {
		result.WrittenRecords = data.Records
	}

	p.logger.Info("Wrote records to Google Sheets",
		"source_id", data.SourceId,
		"table", tableName,
		"count", len(data.Records),
	)

	return &tabularpb.WriteRecordsResponse{
		Success: true,
		Data:    []*tabularpb.WriteRecordsResult{result},
	}, nil
}

// UpdateRecords updates existing records in a Google Sheets spreadsheet
func (p *GoogleSheetsProvider) UpdateRecords(ctx context.Context, req *tabularpb.UpdateRecordsRequest) (*tabularpb.UpdateRecordsResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.UpdateRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Sheets tabular provider is not initialized",
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

	p.mu.RLock()
	service := p.clientManager.GetService()
	p.mu.RUnlock()

	// Build A1 notation from selection
	a1Range := selectionToA1Notation(data.Selection)

	// First read the existing data
	readResp, err := service.Spreadsheets.Values.Get(data.SourceId, a1Range).
		ValueRenderOption("FORMATTED_VALUE").
		Context(ctx).
		Do()
	if err != nil {
		p.logger.Error("Failed to read for update", "error", err, "source_id", data.SourceId)
		return &tabularpb.UpdateRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "READ_FAILED",
				Message: fmt.Sprintf("Failed to read records for update: %v", err),
			},
		}, nil
	}

	records := valueRangeToRecords(readResp)

	// Find matching records based on selection
	matchingIndices := findMatchingIndices(records, data.Selection)
	recordsMatched := int32(len(matchingIndices))
	recordsUpdated := int32(0)

	// Apply updates to matching records
	for _, idx := range matchingIndices {
		if idx >= 0 && idx < len(records) {
			record := records[idx]

			// Apply field updates
			for _, update := range data.Updates {
				if update.Value != nil {
					switch field := update.Field.(type) {
					case *tabularpb.FieldUpdate_FieldIndex:
						// Ensure values slice is large enough
						for len(record.Values) <= int(field.FieldIndex) {
							record.Values = append(record.Values, &tabularpb.FieldValue{})
						}
						record.Values[field.FieldIndex] = update.Value
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

	// Write back updated records
	if recordsUpdated > 0 {
		valueRange := recordsToValueRange(records)
		_, err = service.Spreadsheets.Values.Update(data.SourceId, a1Range, valueRange).
			ValueInputOption("USER_ENTERED").
			Context(ctx).
			Do()
		if err != nil {
			p.logger.Error("Failed to update records", "error", err, "source_id", data.SourceId)
			return &tabularpb.UpdateRecordsResponse{
				Success: false,
				Error: &commonpb.Error{
					Code:    "UPDATE_FAILED",
					Message: fmt.Sprintf("Failed to update records: %v", err),
				},
			}, nil
		}
	}

	p.logger.Info("Updated records in Google Sheets",
		"source_id", data.SourceId,
		"matched", recordsMatched,
		"updated", recordsUpdated,
	)

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

// DeleteRecords deletes records from a Google Sheets spreadsheet
func (p *GoogleSheetsProvider) DeleteRecords(ctx context.Context, req *tabularpb.DeleteRecordsRequest) (*tabularpb.DeleteRecordsResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Sheets tabular provider is not initialized",
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

	p.mu.RLock()
	service := p.clientManager.GetService()
	p.mu.RUnlock()

	// Get sheet ID for the table
	spreadsheet, err := service.Spreadsheets.Get(data.SourceId).Context(ctx).Do()
	if err != nil {
		p.logger.Error("Failed to get spreadsheet", "error", err, "source_id", data.SourceId)
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "SPREADSHEET_NOT_FOUND",
				Message: fmt.Sprintf("Failed to get spreadsheet: %v", err),
			},
		}, nil
	}

	// Find the sheet
	tableName := data.Selection.GetTable()
	if tableName == "" {
		tableName = "Sheet1"
	}

	var sheetID int64 = -1
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == tableName {
			sheetID = sheet.Properties.SheetId
			break
		}
	}

	if sheetID == -1 {
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "SHEET_NOT_FOUND",
				Message: fmt.Sprintf("Sheet '%s' not found", tableName),
			},
		}, nil
	}

	// First read the data to find matching records
	a1Range := selectionToA1Notation(data.Selection)
	readResp, err := service.Spreadsheets.Values.Get(data.SourceId, a1Range).
		ValueRenderOption("FORMATTED_VALUE").
		Context(ctx).
		Do()
	if err != nil {
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "READ_FAILED",
				Message: fmt.Sprintf("Failed to read records for delete: %v", err),
			},
		}, nil
	}

	records := valueRangeToRecords(readResp)
	matchingIndices := findMatchingIndices(records, data.Selection)

	// Sort in reverse order to delete from bottom to top
	sort.Sort(sort.Reverse(sort.IntSlice(matchingIndices)))

	recordsDeleted := int32(0)

	// Build batch update request for deletions
	var requests []*sheets.Request
	for _, idx := range matchingIndices {
		requests = append(requests, &sheets.Request{
			DeleteDimension: &sheets.DeleteDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "ROWS",
					StartIndex: int64(idx),
					EndIndex:   int64(idx + 1),
				},
			},
		})
		recordsDeleted++
	}

	if len(requests) > 0 {
		batchReq := &sheets.BatchUpdateSpreadsheetRequest{
			Requests: requests,
		}
		_, err = service.Spreadsheets.BatchUpdate(data.SourceId, batchReq).Context(ctx).Do()
		if err != nil {
			p.logger.Error("Failed to delete records", "error", err, "source_id", data.SourceId)
			return &tabularpb.DeleteRecordsResponse{
				Success: false,
				Error: &commonpb.Error{
					Code:    "DELETE_FAILED",
					Message: fmt.Sprintf("Failed to delete records: %v", err),
				},
			}, nil
		}
	}

	p.logger.Info("Deleted records from Google Sheets",
		"source_id", data.SourceId,
		"table", tableName,
		"count", recordsDeleted,
	)

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
func (p *GoogleSheetsProvider) SearchRecords(ctx context.Context, req *tabularpb.SearchRecordsRequest) (*tabularpb.SearchRecordsResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.SearchRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Sheets tabular provider is not initialized",
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
	service := p.clientManager.GetService()
	p.mu.RUnlock()

	// Build range - read all data from the table
	tableName := data.Table
	if tableName == "" {
		tableName = "Sheet1"
	}

	// Read all records from the table
	resp, err := service.Spreadsheets.Values.Get(data.SourceId, tableName).
		ValueRenderOption("FORMATTED_VALUE").
		Context(ctx).
		Do()
	if err != nil {
		p.logger.Error("Failed to read for search", "error", err, "source_id", data.SourceId)
		return &tabularpb.SearchRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "READ_FAILED",
				Message: fmt.Sprintf("Failed to read records for search: %v", err),
			},
		}, nil
	}

	records := valueRangeToRecords(resp)

	// Apply filter in-memory
	var filteredRecords []*tabularpb.Record
	if data.Filter != nil {
		for _, record := range records {
			if matchesFilter(record, data.Filter) {
				filteredRecords = append(filteredRecords, record)
			}
		}
	} else {
		filteredRecords = records
	}

	// Apply sorting
	if len(data.SortBy) > 0 {
		filteredRecords = applySort(filteredRecords, data.SortBy)
	}

	// Apply pagination
	totalCount := int64(len(filteredRecords))
	start := int(data.Offset)
	end := len(filteredRecords)
	if data.Limit > 0 {
		end = start + int(data.Limit)
	}
	if start > len(filteredRecords) {
		start = len(filteredRecords)
	}
	if end > len(filteredRecords) {
		end = len(filteredRecords)
	}
	paginatedRecords := filteredRecords[start:end]
	hasMore := end < len(filteredRecords)

	p.logger.Info("Searched records in Google Sheets",
		"source_id", data.SourceId,
		"table", tableName,
		"found", len(paginatedRecords),
	)

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
func (p *GoogleSheetsProvider) GetSchema(ctx context.Context, req *tabularpb.GetSchemaRequest) (*tabularpb.GetSchemaResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.GetSchemaResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Sheets tabular provider is not initialized",
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
	service := p.clientManager.GetService()
	p.mu.RUnlock()

	// Get spreadsheet metadata
	spreadsheet, err := service.Spreadsheets.Get(data.SourceId).Context(ctx).Do()
	if err != nil {
		p.logger.Error("Failed to get spreadsheet", "error", err, "source_id", data.SourceId)
		return &tabularpb.GetSchemaResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "SPREADSHEET_NOT_FOUND",
				Message: fmt.Sprintf("Failed to get spreadsheet: %v", err),
			},
		}, nil
	}

	result := &tabularpb.GetSchemaResult{
		Source: &tabularpb.Source{
			Id:           spreadsheet.SpreadsheetId,
			Name:         spreadsheet.Properties.Title,
			Url:          spreadsheet.SpreadsheetUrl,
			ProviderType: tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_GOOGLE_SHEETS,
		},
	}

	// Get specific table schema if table name is provided
	if data.Table != "" {
		schema, err := p.fetchSchema(ctx, service, data.SourceId, data.Table)
		if err != nil {
			return &tabularpb.GetSchemaResponse{
				Success: false,
				Error: &commonpb.Error{
					Code:    "SCHEMA_FETCH_FAILED",
					Message: fmt.Sprintf("Failed to fetch schema: %v", err),
				},
			}, nil
		}
		result.TableSchema = schema
	}

	p.logger.Info("Got schema from Google Sheets",
		"source_id", data.SourceId,
		"table", data.Table,
	)

	return &tabularpb.GetSchemaResponse{
		Success: true,
		Data:    []*tabularpb.GetSchemaResult{result},
	}, nil
}

// GetSource retrieves metadata about the data source
func (p *GoogleSheetsProvider) GetSource(ctx context.Context, req *tabularpb.GetSourceRequest) (*tabularpb.GetSourceResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.GetSourceResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Sheets tabular provider is not initialized",
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
	service := p.clientManager.GetService()
	p.mu.RUnlock()

	// Get spreadsheet metadata
	spreadsheet, err := service.Spreadsheets.Get(data.SourceId).Context(ctx).Do()
	if err != nil {
		p.logger.Error("Failed to get spreadsheet", "error", err, "source_id", data.SourceId)
		return &tabularpb.GetSourceResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "SPREADSHEET_NOT_FOUND",
				Message: fmt.Sprintf("Failed to get spreadsheet: %v", err),
			},
		}, nil
	}

	source := &tabularpb.Source{
		Id:           spreadsheet.SpreadsheetId,
		Name:         spreadsheet.Properties.Title,
		Url:          spreadsheet.SpreadsheetUrl,
		ProviderType: tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_GOOGLE_SHEETS,
	}

	// Include tables if requested
	if data.IncludeTables {
		for i, sheet := range spreadsheet.Sheets {
			source.Tables = append(source.Tables, &tabularpb.Table{
				Id:       fmt.Sprintf("%d", sheet.Properties.SheetId),
				Name:     sheet.Properties.Title,
				Position: int32(i),
				Hidden:   sheet.Properties.Hidden,
			})
		}
	}

	p.logger.Info("Got source from Google Sheets", "source_id", data.SourceId)

	return &tabularpb.GetSourceResponse{
		Success: true,
		Data:    []*tabularpb.Source{source},
	}, nil
}

// ListTables lists all available tables/sheets in the data source
func (p *GoogleSheetsProvider) ListTables(ctx context.Context, req *tabularpb.ListTablesRequest) (*tabularpb.ListTablesResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.ListTablesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Sheets tabular provider is not initialized",
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
	service := p.clientManager.GetService()
	p.mu.RUnlock()

	// Get spreadsheet metadata
	spreadsheet, err := service.Spreadsheets.Get(data.SourceId).Context(ctx).Do()
	if err != nil {
		p.logger.Error("Failed to get spreadsheet", "error", err, "source_id", data.SourceId)
		return &tabularpb.ListTablesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "SPREADSHEET_NOT_FOUND",
				Message: fmt.Sprintf("Failed to get spreadsheet: %v", err),
			},
		}, nil
	}

	var tables []*tabularpb.Table
	for i, sheet := range spreadsheet.Sheets {
		tables = append(tables, &tabularpb.Table{
			Id:       fmt.Sprintf("%d", sheet.Properties.SheetId),
			Name:     sheet.Properties.Title,
			Position: int32(i),
			Hidden:   sheet.Properties.Hidden,
		})
	}

	p.logger.Info("Listed tables from Google Sheets",
		"source_id", data.SourceId,
		"count", len(tables),
	)

	return &tabularpb.ListTablesResponse{
		Success: true,
		Data:    tables,
	}, nil
}

// =============================================================================
// Batch Operations
// =============================================================================

// BatchExecute executes multiple operations in a single request
func (p *GoogleSheetsProvider) BatchExecute(ctx context.Context, req *tabularpb.BatchExecuteRequest) (*tabularpb.BatchExecuteResponse, error) {
	if !p.IsEnabled() {
		return &tabularpb.BatchExecuteResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "NOT_INITIALIZED",
				Message: "Google Sheets tabular provider is not initialized",
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

	p.logger.Info("Batch executed operations",
		"source_id", data.SourceId,
		"total", len(data.Operations),
		"success", successCount,
		"failures", failureCount,
	)

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
// Health & Capabilities
// =============================================================================

// CheckHealth performs a detailed health check
func (p *GoogleSheetsProvider) CheckHealth(ctx context.Context, req *tabularpb.CheckHealthRequest) (*tabularpb.CheckHealthResponse, error) {
	err := p.IsHealthy(ctx)
	if err != nil {
		return &tabularpb.CheckHealthResponse{
			Success: true,
			Data: []*tabularpb.HealthStatus{
				{
					IsHealthy: false,
					Message:   err.Error(),
					Details: map[string]string{
						"provider": "googlesheets",
						"status":   "error",
					},
				},
			},
		}, nil
	}

	return &tabularpb.CheckHealthResponse{
		Success: true,
		Data: []*tabularpb.HealthStatus{
			{
				IsHealthy: true,
				Message:   "Google Sheets tabular provider is healthy",
				Details: map[string]string{
					"provider": "googlesheets",
					"status":   "operational",
				},
			},
		},
	}, nil
}

// GetCapabilitiesInfo returns detailed capability information
func (p *GoogleSheetsProvider) GetCapabilitiesInfo(ctx context.Context, req *tabularpb.GetCapabilitiesRequest) (*tabularpb.GetCapabilitiesResponse, error) {
	capabilities := p.GetCapabilities()

	return &tabularpb.GetCapabilitiesResponse{
		Success: true,
		Data: []*tabularpb.ProviderCapabilities{
			{
				ProviderId:           "googlesheets",
				ProviderType:         tabularpb.TabularProviderType_TABULAR_PROVIDER_TYPE_GOOGLE_SHEETS,
				Capabilities:         capabilities,
				MaxRecordsPerRequest: 10000,                   // Google Sheets API limit
				MaxFieldsPerRecord:   18278,                   // Max columns in Google Sheets (ZZZ)
				MaxSourceSizeBytes:   10 * 1024 * 1024 * 1024, // 10GB per spreadsheet
			},
		},
	}, nil
}

// =============================================================================
// Helper Methods
// =============================================================================

// fetchSchema reads the first row as headers and infers schema
func (p *GoogleSheetsProvider) fetchSchema(ctx context.Context, service *sheets.Service, sourceID, tableName string) (*tabularpb.TableSchema, error) {
	if tableName == "" {
		tableName = "Sheet1"
	}

	// Read first row for headers
	a1Range := fmt.Sprintf("%s!1:1", tableName)
	resp, err := service.Spreadsheets.Values.Get(sourceID, a1Range).
		ValueRenderOption("FORMATTED_VALUE").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read schema: %w", err)
	}

	schema := &tabularpb.TableSchema{
		Id:   tableName,
		Name: tableName,
	}

	if len(resp.Values) > 0 {
		for i, val := range resp.Values[0] {
			fieldName := ""
			if s, ok := val.(string); ok {
				fieldName = s
			} else {
				fieldName = fmt.Sprintf("Column%d", i+1)
			}

			schema.Fields = append(schema.Fields, &tabularpb.Field{
				Index:     int32(i),
				Name:      fieldName,
				FieldType: tabularpb.FieldType_FIELD_TYPE_STRING, // Default to string
			})
		}
	}

	return schema, nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// selectionToA1Notation converts a Selection to Google Sheets A1 notation
func selectionToA1Notation(selection *tabularpb.Selection) string {
	if selection == nil {
		return "Sheet1"
	}

	tableName := selection.Table
	if tableName == "" {
		tableName = "Sheet1"
	}

	// If no record selection, return entire sheet
	if selection.Records == nil {
		return tableName
	}

	// Build A1 range from selection
	startRow := int64(1) // 1-based
	endRow := int64(-1)  // -1 means open-ended

	if selection.Records.IndexRange != nil {
		startRow = selection.Records.IndexRange.Start + 1 // Convert to 1-based
		if selection.Records.IndexRange.End > 0 {
			endRow = selection.Records.IndexRange.End + 1
		}
	}

	// Build column range
	startCol := "A"
	endCol := ""

	if selection.Fields != nil && len(selection.Fields.Indices) > 0 {
		// Sort indices to get range
		indices := make([]int, len(selection.Fields.Indices))
		for i, idx := range selection.Fields.Indices {
			indices[i] = int(idx)
		}
		sort.Ints(indices)
		startCol = columnIndexToLetter(indices[0])
		endCol = columnIndexToLetter(indices[len(indices)-1])
	}

	// Build A1 notation
	if endRow < 0 {
		if endCol == "" {
			return fmt.Sprintf("%s!%s%d:%s", tableName, startCol, startRow, startCol)
		}
		return fmt.Sprintf("%s!%s%d:%s", tableName, startCol, startRow, endCol)
	}

	if endCol == "" {
		return fmt.Sprintf("%s!%s%d:%s%d", tableName, startCol, startRow, startCol, endRow)
	}
	return fmt.Sprintf("%s!%s%d:%s%d", tableName, startCol, startRow, endCol, endRow)
}

// columnIndexToLetter converts a 0-based column index to letter(s)
func columnIndexToLetter(index int) string {
	result := ""
	for index >= 0 {
		result = string(rune('A'+index%26)) + result
		index = index/26 - 1
	}
	return result
}

// recordsToValueRange converts protobuf records to Google Sheets ValueRange
// Handles both Values (indexed) and NamedValues (map) formats
func recordsToValueRange(records []*tabularpb.Record) *sheets.ValueRange {
	var values [][]interface{}

	for _, record := range records {
		var row []interface{}

		// First try indexed Values
		if len(record.Values) > 0 {
			for _, fv := range record.Values {
				row = append(row, fieldValueToInterface(fv))
			}
		} else if len(record.NamedValues) > 0 {
			// Fall back to NamedValues (sorted by key for consistent column order)
			// Extract keys and sort them
			keys := make([]string, 0, len(record.NamedValues))
			for k := range record.NamedValues {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			// Build row with values in sorted key order
			for _, key := range keys {
				row = append(row, fieldValueToInterface(record.NamedValues[key]))
			}
		}

		values = append(values, row)
	}

	return &sheets.ValueRange{
		Values: values,
	}
}

// valueRangeToRecords converts Google Sheets ValueRange to protobuf records
func valueRangeToRecords(vr *sheets.ValueRange) []*tabularpb.Record {
	var records []*tabularpb.Record

	for i, row := range vr.Values {
		record := &tabularpb.Record{
			Index: int64(i),
			Id:    fmt.Sprintf("row_%d", i),
		}

		for _, val := range row {
			record.Values = append(record.Values, interfaceToFieldValue(val))
		}

		records = append(records, record)
	}

	return records
}

// fieldValueToInterface converts a FieldValue to interface{} for Google Sheets
func fieldValueToInterface(fv *tabularpb.FieldValue) interface{} {
	if fv == nil {
		return ""
	}

	switch v := fv.Value.(type) {
	case *tabularpb.FieldValue_StringValue:
		return v.StringValue
	case *tabularpb.FieldValue_IntegerValue:
		return v.IntegerValue
	case *tabularpb.FieldValue_FloatValue:
		return v.FloatValue
	case *tabularpb.FieldValue_BooleanValue:
		return v.BooleanValue
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

// interfaceToFieldValue converts an interface{} from Google Sheets to FieldValue
func interfaceToFieldValue(val interface{}) *tabularpb.FieldValue {
	if val == nil {
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_NULL,
		}
	}

	switch v := val.(type) {
	case string:
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_STRING,
			Value:     &tabularpb.FieldValue_StringValue{StringValue: v},
			RawValue:  v,
		}
	case float64:
		// Check if it's actually an integer
		if v == float64(int64(v)) {
			return &tabularpb.FieldValue{
				FieldType: tabularpb.FieldType_FIELD_TYPE_INTEGER,
				Value:     &tabularpb.FieldValue_IntegerValue{IntegerValue: int64(v)},
				RawValue:  fmt.Sprintf("%d", int64(v)),
			}
		}
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_FLOAT,
			Value:     &tabularpb.FieldValue_FloatValue{FloatValue: v},
			RawValue:  fmt.Sprintf("%f", v),
		}
	case bool:
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_BOOLEAN,
			Value:     &tabularpb.FieldValue_BooleanValue{BooleanValue: v},
			RawValue:  fmt.Sprintf("%t", v),
		}
	default:
		str := fmt.Sprintf("%v", v)
		return &tabularpb.FieldValue{
			FieldType: tabularpb.FieldType_FIELD_TYPE_STRING,
			Value:     &tabularpb.FieldValue_StringValue{StringValue: str},
			RawValue:  str,
		}
	}
}

// findMatchingIndices finds indices of records matching selection criteria
func findMatchingIndices(records []*tabularpb.Record, selection *tabularpb.Selection) []int {
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
			if matchesFilter(record, selection.Records.Filter) {
				indices = append(indices, i)
			}
		}
		return indices
	}

	return indices
}

// matchesFilter checks if a record matches filter conditions
func matchesFilter(record *tabularpb.Record, filter *tabularpb.FilterGroup) bool {
	if filter == nil {
		return true
	}

	if len(filter.Filters) == 0 && len(filter.Groups) == 0 {
		return true
	}

	// Check individual filters
	filterResults := make([]bool, 0, len(filter.Filters)+len(filter.Groups))

	for _, f := range filter.Filters {
		filterResults = append(filterResults, matchesSingleFilter(record, f))
	}

	// Check nested groups
	for _, group := range filter.Groups {
		filterResults = append(filterResults, matchesFilter(record, group))
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
func matchesSingleFilter(record *tabularpb.Record, filter *tabularpb.Filter) bool {
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
func applySort(records []*tabularpb.Record, sortSpecs []*tabularpb.SortSpec) []*tabularpb.Record {
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
