package tabular

import (
	"context"
	"fmt"
	"log"

	"leapfor.xyz/espyna/internal/application/ports/integration"
	commonpb "leapfor.xyz/esqyma/golang/v1/domain/common"
	tabularpb "leapfor.xyz/esqyma/golang/v1/integration/tabular"
)

// BatchExecuteRepositories groups all repository dependencies
type BatchExecuteRepositories struct {
	// No repositories needed for external tabular provider integration
}

// BatchExecuteServices groups all service dependencies
type BatchExecuteServices struct {
	Provider integration.TabularSourceProvider
}

// BatchExecuteUseCase handles executing batch operations on a tabular source
type BatchExecuteUseCase struct {
	repositories BatchExecuteRepositories
	services     BatchExecuteServices
}

// NewBatchExecuteUseCase creates a new BatchExecuteUseCase
func NewBatchExecuteUseCase(
	repositories BatchExecuteRepositories,
	services BatchExecuteServices,
) *BatchExecuteUseCase {
	return &BatchExecuteUseCase{
		repositories: repositories,
		services:     services,
	}
}

// Execute performs batch operations on a tabular source using the configured provider
func (uc *BatchExecuteUseCase) Execute(ctx context.Context, req *tabularpb.BatchExecuteRequest) (*tabularpb.BatchExecuteResponse, error) {
	if uc.services.Provider == nil || !uc.services.Provider.IsEnabled() {
		return &tabularpb.BatchExecuteResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "PROVIDER_UNAVAILABLE",
				Message: "Tabular provider is not available",
			},
		}, nil
	}

	if req.Data == nil {
		return &tabularpb.BatchExecuteResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Request data is required",
			},
		}, nil
	}

	// Validate request
	if req.Data.SourceId == "" {
		return &tabularpb.BatchExecuteResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "Source ID is required",
			},
		}, nil
	}

	if len(req.Data.Operations) == 0 {
		return &tabularpb.BatchExecuteResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "INVALID_REQUEST",
				Message: "At least one operation is required",
			},
		}, nil
	}

	log.Printf("Executing batch with %d operations on source %s (fail_fast: %v, transactional: %v)",
		len(req.Data.Operations), req.Data.SourceId, req.Data.FailFast, req.Data.Transactional)

	// Execute via provider
	response, err := uc.services.Provider.BatchExecute(ctx, req)
	if err != nil {
		log.Printf("Failed to execute batch: %v", err)
		return &tabularpb.BatchExecuteResponse{
			Success: false,
			Error: &commonpb.Error{
				Code:    "BATCH_EXECUTE_FAILED",
				Message: fmt.Sprintf("Failed to execute batch: %v", err),
			},
		}, nil
	}

	if response.Success && len(response.Data) > 0 {
		log.Printf("Batch execution completed: %d succeeded, %d failed",
			response.Data[0].SuccessCount, response.Data[0].FailureCount)
	} else if !response.Success && response.Error != nil {
		log.Printf("Batch execute failed: %s", response.Error.Message)
	}

	return response, nil
}
