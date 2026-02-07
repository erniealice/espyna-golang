package tabular

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports/integration"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// DeleteRecordsRepositories groups all repository dependencies
type DeleteRecordsRepositories struct {
	// No repositories needed for external tabular provider integration
}

// DeleteRecordsServices groups all service dependencies
type DeleteRecordsServices struct {
	Provider integration.TabularSourceProvider
}

// DeleteRecordsUseCase handles deleting records from a tabular source
type DeleteRecordsUseCase struct {
	repositories DeleteRecordsRepositories
	services     DeleteRecordsServices
}

// NewDeleteRecordsUseCase creates a new DeleteRecordsUseCase
func NewDeleteRecordsUseCase(
	repositories DeleteRecordsRepositories,
	services DeleteRecordsServices,
) *DeleteRecordsUseCase {
	return &DeleteRecordsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute deletes records from a tabular source using the configured provider
func (uc *DeleteRecordsUseCase) Execute(ctx context.Context, req *tabularpb.DeleteRecordsRequest) (*tabularpb.DeleteRecordsResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Validate request
	if req.Data.SourceId == "" {
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Source ID is required",
			},
		}, nil
	}

	if req.Data.Selection == nil {
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Selection criteria is required",
			},
		}, nil
	}

	log.Printf("Deleting records from source %s", req.Data.SourceId)

	// Execute via provider
	response, err := uc.services.Provider.DeleteRecords(ctx, req)
	if err != nil {
		log.Printf("Failed to delete records: %v", err)
		return &tabularpb.DeleteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "DELETE_RECORDS_FAILED",
				Message: fmt.Sprintf("Failed to delete records: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("Successfully deleted %d records", response.Data[0].RecordsDeleted)
	} else if !response.Success && response.Error != nil {
		log.Printf("Delete records failed: %s", response.Error.Message)
	}

	return response, nil
}
