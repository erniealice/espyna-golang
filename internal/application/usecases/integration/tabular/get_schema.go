package tabular

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports/integration"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// GetSchemaRepositories groups all repository dependencies
type GetSchemaRepositories struct {
	// No repositories needed for external tabular provider integration
}

// GetSchemaServices groups all service dependencies
type GetSchemaServices struct {
	Provider integration.TabularSourceProvider
}

// GetSchemaUseCase handles retrieving schema from a tabular source
type GetSchemaUseCase struct {
	repositories GetSchemaRepositories
	services     GetSchemaServices
}

// NewGetSchemaUseCase creates a new GetSchemaUseCase
func NewGetSchemaUseCase(
	repositories GetSchemaRepositories,
	services GetSchemaServices,
) *GetSchemaUseCase {
	return &GetSchemaUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute retrieves schema from a tabular source using the configured provider
func (uc *GetSchemaUseCase) Execute(ctx context.Context, req *tabularpb.GetSchemaRequest) (*tabularpb.GetSchemaResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &tabularpb.GetSchemaResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &tabularpb.GetSchemaResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Validate request
	if req.Data.SourceId == "" {
		return &tabularpb.GetSchemaResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Source ID is required",
			},
		}, nil
	}

	log.Printf("Getting schema from source %s", req.Data.SourceId)
	if req.Data.Table != "" {
		log.Printf("Getting schema for table: %s", req.Data.Table)
	}

	// Execute via provider
	response, err := uc.services.Provider.GetSchema(ctx, req)
	if err != nil {
		log.Printf("Failed to get schema: %v", err)
		return &tabularpb.GetSchemaResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "GET_SCHEMA_FAILED",
				Message: fmt.Sprintf("Failed to get schema: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("Successfully retrieved schema")
	} else if !response.Success && response.Error != nil {
		log.Printf("Get schema failed: %s", response.Error.Message)
	}

	return response, nil
}
