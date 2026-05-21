package tabular

import (
	"context"
	"fmt"
	"log"

	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	tabularpb "github.com/erniealice/esqyma/pkg/schema/v1/integration/tabular"
)

// UpdateRecordsRepositories groups all repository dependencies
type UpdateRecordsRepositories struct {
	// No repositories needed for external tabular provider integration
}

// UpdateRecordsServices groups all service dependencies
type UpdateRecordsServices struct {
	Provider integration.TabularSourceProvider
}

// UpdateRecordsUseCase handles updating records in a tabular source
type UpdateRecordsUseCase struct {
	repositories UpdateRecordsRepositories
	services     UpdateRecordsServices
}

// NewUpdateRecordsUseCase creates a new UpdateRecordsUseCase
func NewUpdateRecordsUseCase(
	repositories UpdateRecordsRepositories,
	services UpdateRecordsServices,
) *UpdateRecordsUseCase {
	return &UpdateRecordsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute updates records in a tabular source using the configured provider
func (uc *UpdateRecordsUseCase) Execute(ctx context.Context, req *tabularpb.UpdateRecordsRequest) (*tabularpb.UpdateRecordsResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &tabularpb.UpdateRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &tabularpb.UpdateRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Validate request
	if req.Data.SourceId == "" {
		return &tabularpb.UpdateRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Source ID is required",
			},
		}, nil
	}

	// Must have either updates or replacement records
	if len(req.Data.Updates) == 0 && len(req.Data.ReplacementRecords) == 0 {
		return &tabularpb.UpdateRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Either updates or replacement records are required",
			},
		}, nil
	}

	log.Printf("Updating records in source %s", req.Data.SourceId)

	// Execute via provider
	response, err := uc.services.Provider.UpdateRecords(ctx, req)
	if err != nil {
		log.Printf("Failed to update records: %v", err)
		return &tabularpb.UpdateRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "UPDATE_RECORDS_FAILED",
				Message: fmt.Sprintf("Failed to update records: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("Successfully updated %d records (matched: %d)", response.Data[0].RecordsUpdated, response.Data[0].RecordsMatched)
	} else if !response.Success && response.Error != nil {
		log.Printf("Update records failed: %s", response.Error.Message)
	}

	return response, nil
}
