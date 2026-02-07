package tabular

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports/integration"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// ReadRecordsRepositories groups all repository dependencies
type ReadRecordsRepositories struct {
	// No repositories needed for external tabular provider integration
}

// ReadRecordsServices groups all service dependencies
type ReadRecordsServices struct {
	Provider integration.TabularSourceProvider
}

// ReadRecordsUseCase handles reading records from a tabular source
type ReadRecordsUseCase struct {
	repositories ReadRecordsRepositories
	services     ReadRecordsServices
}

// NewReadRecordsUseCase creates a new ReadRecordsUseCase
func NewReadRecordsUseCase(
	repositories ReadRecordsRepositories,
	services ReadRecordsServices,
) *ReadRecordsUseCase {
	return &ReadRecordsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute reads records from a tabular source using the configured provider
func (uc *ReadRecordsUseCase) Execute(ctx context.Context, req *tabularpb.ReadRecordsRequest) (*tabularpb.ReadRecordsResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &tabularpb.ReadRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &tabularpb.ReadRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Validate request
	if req.Data.SourceId == "" {
		return &tabularpb.ReadRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Source ID is required",
			},
		}, nil
	}

	log.Printf("Reading records from source %s", req.Data.SourceId)

	// Execute via provider
	response, err := uc.services.Provider.ReadRecords(ctx, req)
	if err != nil {
		log.Printf("Failed to read records: %v", err)
		return &tabularpb.ReadRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "READ_RECORDS_FAILED",
				Message: fmt.Sprintf("Failed to read records: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("Successfully read %d records", len(response.Data[0].Records))
	} else if !response.Success && response.Error != nil {
		log.Printf("Read records failed: %s", response.Error.Message)
	}

	return response, nil
}
