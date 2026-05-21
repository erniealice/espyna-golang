// Package tabular provides use cases for tabular data integration (Google Sheets, etc.)
//
// # Adding New Use Cases
//
// When adding a new use case to this package, remember to update:
//
//  1. UseCases struct - Add the new use case field
//  2. NewUseCases() - Initialize the new use case
//  3. Routing config - packages/espyna/internal/composition/routing/config/integration/tabular.go
//  4. Workflow registry - packages/espyna/internal/orchestration/workflow/integration/tabular.go
//
// # Available Use Cases
//
//   - WriteRecords: Full record write with complex Record structure and FieldValue types
//   - WriteRecordSimple: Simplified single record write using google.protobuf.Struct for flat fields
//
// All use cases are proto-based for consistency and maintainability.
package tabular

import (
	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
)

// TabularRepositories groups all repository dependencies for tabular use cases
type TabularRepositories struct {
	// No repositories needed - provider handles all data access
}

// TabularServices groups all service dependencies for tabular use cases
type TabularServices struct {
	Provider integration.TabularSourceProvider
}

// UseCases contains all tabular integration use cases
type UseCases struct {
	ReadRecords       *ReadRecordsUseCase
	WriteRecords      *WriteRecordsUseCase
	WriteRecordSimple *WriteRecordSimpleUseCase // Workflow-friendly flat input format
	UpdateRecords     *UpdateRecordsUseCase
	DeleteRecords     *DeleteRecordsUseCase
	SearchRecords     *SearchRecordsUseCase
	GetSchema         *GetSchemaUseCase
	GetSource         *GetSourceUseCase
	ListTables        *ListTablesUseCase
	BatchExecute      *BatchExecuteUseCase
	CheckHealth       *CheckHealthUseCase
	GetCapabilities   *GetCapabilitiesUseCase
}

// NewUseCases creates a new collection of tabular integration use cases
func NewUseCases(
	repositories TabularRepositories,
	services TabularServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	readRecordsRepos := ReadRecordsRepositories{}
	readRecordsServices := ReadRecordsServices{
		Provider: services.Provider,
	}

	writeRecordsRepos := WriteRecordsRepositories{}
	writeRecordsServices := WriteRecordsServices{
		Provider: services.Provider,
	}

	updateRecordsRepos := UpdateRecordsRepositories{}
	updateRecordsServices := UpdateRecordsServices{
		Provider: services.Provider,
	}

	deleteRecordsRepos := DeleteRecordsRepositories{}
	deleteRecordsServices := DeleteRecordsServices{
		Provider: services.Provider,
	}

	searchRecordsRepos := SearchRecordsRepositories{}
	searchRecordsServices := SearchRecordsServices{
		Provider: services.Provider,
	}

	getSchemaRepos := GetSchemaRepositories{}
	getSchemaServices := GetSchemaServices{
		Provider: services.Provider,
	}

	getSourceRepos := GetSourceRepositories{}
	getSourceServices := GetSourceServices{
		Provider: services.Provider,
	}

	listTablesRepos := ListTablesRepositories{}
	listTablesServices := ListTablesServices{
		Provider: services.Provider,
	}

	batchExecuteRepos := BatchExecuteRepositories{}
	batchExecuteServices := BatchExecuteServices{
		Provider: services.Provider,
	}

	checkHealthRepos := CheckHealthRepositories{}
	checkHealthServices := CheckHealthServices{
		Provider: services.Provider,
	}

	getCapabilitiesRepos := GetCapabilitiesRepositories{}
	getCapabilitiesServices := GetCapabilitiesServices{
		Provider: services.Provider,
	}

	writeRecordSimpleRepos := WriteRecordSimpleRepositories{}
	writeRecordSimpleServices := WriteRecordSimpleServices{
		Provider: services.Provider,
	}

	return &UseCases{
		ReadRecords:       NewReadRecordsUseCase(readRecordsRepos, readRecordsServices),
		WriteRecords:      NewWriteRecordsUseCase(writeRecordsRepos, writeRecordsServices),
		WriteRecordSimple: NewWriteRecordSimpleUseCase(writeRecordSimpleRepos, writeRecordSimpleServices),
		UpdateRecords:     NewUpdateRecordsUseCase(updateRecordsRepos, updateRecordsServices),
		DeleteRecords:     NewDeleteRecordsUseCase(deleteRecordsRepos, deleteRecordsServices),
		SearchRecords:     NewSearchRecordsUseCase(searchRecordsRepos, searchRecordsServices),
		GetSchema:         NewGetSchemaUseCase(getSchemaRepos, getSchemaServices),
		GetSource:         NewGetSourceUseCase(getSourceRepos, getSourceServices),
		ListTables:        NewListTablesUseCase(listTablesRepos, listTablesServices),
		BatchExecute:      NewBatchExecuteUseCase(batchExecuteRepos, batchExecuteServices),
		CheckHealth:       NewCheckHealthUseCase(checkHealthRepos, checkHealthServices),
		GetCapabilities:   NewGetCapabilitiesUseCase(getCapabilitiesRepos, getCapabilitiesServices),
	}
}

// NewUseCasesFromProvider creates use cases directly from a tabular source provider
// This is a convenience function for simple setups
func NewUseCasesFromProvider(provider integration.TabularSourceProvider) *UseCases {
	if provider == nil {
		return nil
	}

	repositories := TabularRepositories{}
	services := TabularServices{
		Provider: provider,
	}

	return NewUseCases(repositories, services)
}
