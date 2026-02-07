package tabular

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports/integration"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// SearchRecordsRepositories groups all repository dependencies
type SearchRecordsRepositories struct {
	// No repositories needed for external tabular provider integration
}

// SearchRecordsServices groups all service dependencies
type SearchRecordsServices struct {
	Provider integration.TabularSourceProvider
}

// SearchRecordsUseCase handles searching records in a tabular source
type SearchRecordsUseCase struct {
	repositories SearchRecordsRepositories
	services     SearchRecordsServices
}

// NewSearchRecordsUseCase creates a new SearchRecordsUseCase
func NewSearchRecordsUseCase(
	repositories SearchRecordsRepositories,
	services SearchRecordsServices,
) *SearchRecordsUseCase {
	return &SearchRecordsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute searches for records in a tabular source using the configured provider
func (uc *SearchRecordsUseCase) Execute(ctx context.Context, req *tabularpb.SearchRecordsRequest) (*tabularpb.SearchRecordsResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &tabularpb.SearchRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &tabularpb.SearchRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Validate request
	if req.Data.SourceId == "" {
		return &tabularpb.SearchRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Source ID is required",
			},
		}, nil
	}

	log.Printf("Searching records in source %s, table %s", req.Data.SourceId, req.Data.Table)

	// Execute via provider
	response, err := uc.services.Provider.SearchRecords(ctx, req)
	if err != nil {
		log.Printf("Failed to search records: %v", err)
		return &tabularpb.SearchRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "SEARCH_RECORDS_FAILED",
				Message: fmt.Sprintf("Failed to search records: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("Search found %d records (total: %d)", len(response.Data[0].Records), response.Data[0].TotalCount)
	} else if !response.Success && response.Error != nil {
		log.Printf("Search records failed: %s", response.Error.Message)
	}

	return response, nil
}
