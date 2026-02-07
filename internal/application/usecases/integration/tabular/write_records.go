package tabular

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports/integration"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// WriteRecordsRepositories groups all repository dependencies
type WriteRecordsRepositories struct {
	// No repositories needed for external tabular provider integration
}

// WriteRecordsServices groups all service dependencies
type WriteRecordsServices struct {
	Provider integration.TabularSourceProvider
}

// WriteRecordsUseCase handles writing records to a tabular source
type WriteRecordsUseCase struct {
	repositories WriteRecordsRepositories
	services     WriteRecordsServices
}

// NewWriteRecordsUseCase creates a new WriteRecordsUseCase
func NewWriteRecordsUseCase(
	repositories WriteRecordsRepositories,
	services WriteRecordsServices,
) *WriteRecordsUseCase {
	return &WriteRecordsUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute writes records to a tabular source using the configured provider
func (uc *WriteRecordsUseCase) Execute(ctx context.Context, req *tabularpb.WriteRecordsRequest) (*tabularpb.WriteRecordsResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &tabularpb.WriteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &tabularpb.WriteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Validate request
	if req.Data.SourceId == "" {
		return &tabularpb.WriteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Source ID is required",
			},
		}, nil
	}

	if len(req.Data.Records) == 0 {
		return &tabularpb.WriteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "At least one record is required",
			},
		}, nil
	}

	log.Printf("Writing %d records to source %s", len(req.Data.Records), req.Data.SourceId)

	// Execute via provider
	response, err := uc.services.Provider.WriteRecords(ctx, req)
	if err != nil {
		log.Printf("Failed to write records: %v", err)
		return &tabularpb.WriteRecordsResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "WRITE_RECORDS_FAILED",
				Message: fmt.Sprintf("Failed to write records: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("Successfully wrote %d records to %s", response.Data[0].RecordsWritten, response.Data[0].Location)
	} else if !response.Success && response.Error != nil {
		log.Printf("Write records failed: %s", response.Error.Message)
	}

	return response, nil
}
