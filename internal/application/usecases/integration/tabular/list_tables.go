package tabular

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports/integration"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// ListTablesRepositories groups all repository dependencies
type ListTablesRepositories struct {
	// No repositories needed for external tabular provider integration
}

// ListTablesServices groups all service dependencies
type ListTablesServices struct {
	Provider integration.TabularSourceProvider
}

// ListTablesUseCase handles listing tables in a tabular source
type ListTablesUseCase struct {
	repositories ListTablesRepositories
	services     ListTablesServices
}

// NewListTablesUseCase creates a new ListTablesUseCase
func NewListTablesUseCase(
	repositories ListTablesRepositories,
	services ListTablesServices,
) *ListTablesUseCase {
	return &ListTablesUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute lists tables in a tabular source using the configured provider
func (uc *ListTablesUseCase) Execute(ctx context.Context, req *tabularpb.ListTablesRequest) (*tabularpb.ListTablesResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &tabularpb.ListTablesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &tabularpb.ListTablesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Validate request
	if req.Data.SourceId == "" {
		return &tabularpb.ListTablesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Source ID is required",
			},
		}, nil
	}

	log.Printf("Listing tables in source %s", req.Data.SourceId)

	// Execute via provider
	response, err := uc.services.Provider.ListTables(ctx, req)
	if err != nil {
		log.Printf("Failed to list tables: %v", err)
		return &tabularpb.ListTablesResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "LIST_TABLES_FAILED",
				Message: fmt.Sprintf("Failed to list tables: %v", err),
			},
		}, nil
	}

	if response.Success {
		log.Printf("Successfully listed %d tables", len(response.Data))
	} else if response.Error != nil {
		log.Printf("List tables failed: %s", response.Error.Message)
	}

	return response, nil
}
