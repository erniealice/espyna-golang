package tabular

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports/integration"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// GetSourceRepositories groups all repository dependencies
type GetSourceRepositories struct {
	// No repositories needed for external tabular provider integration
}

// GetSourceServices groups all service dependencies
type GetSourceServices struct {
	Provider integration.TabularSourceProvider
}

// GetSourceUseCase handles retrieving source metadata from a tabular source
type GetSourceUseCase struct {
	repositories GetSourceRepositories
	services     GetSourceServices
}

// NewGetSourceUseCase creates a new GetSourceUseCase
func NewGetSourceUseCase(
	repositories GetSourceRepositories,
	services GetSourceServices,
) *GetSourceUseCase {
	return &GetSourceUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute retrieves source metadata from a tabular source using the configured provider
func (uc *GetSourceUseCase) Execute(ctx context.Context, req *tabularpb.GetSourceRequest) (*tabularpb.GetSourceResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &tabularpb.GetSourceResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &tabularpb.GetSourceResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Validate request
	if req.Data.SourceId == "" {
		return &tabularpb.GetSourceResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Source ID is required",
			},
		}, nil
	}

	log.Printf("Getting source metadata for source %s (include_tables: %v)", req.Data.SourceId, req.Data.IncludeTables)

	// Execute via provider
	response, err := uc.services.Provider.GetSource(ctx, req)
	if err != nil {
		log.Printf("Failed to get source: %v", err)
		return &tabularpb.GetSourceResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "GET_SOURCE_FAILED",
				Message: fmt.Sprintf("Failed to get source: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("Successfully retrieved source metadata: %s", response.Data[0].Name)
	} else if !response.Success && response.Error != nil {
		log.Printf("Get source failed: %s", response.Error.Message)
	}

	return response, nil
}
