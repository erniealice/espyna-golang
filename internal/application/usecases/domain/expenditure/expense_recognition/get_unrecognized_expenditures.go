package expenserecognition

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// GetUnrecognizedExpendituresRepositories groups repository dependencies.
type GetUnrecognizedExpendituresRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// GetUnrecognizedExpendituresServices groups service dependencies.
type GetUnrecognizedExpendituresServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// GetUnrecognizedExpendituresUseCase returns the IDs of expenditures lacking a
// POSTED ExpenseRecognition row. Surfaced in the AP team's review queue.
type GetUnrecognizedExpendituresUseCase struct {
	repositories GetUnrecognizedExpendituresRepositories
	services     GetUnrecognizedExpendituresServices
}

// NewGetUnrecognizedExpendituresUseCase creates a use case with grouped dependencies.
func NewGetUnrecognizedExpendituresUseCase(
	repositories GetUnrecognizedExpendituresRepositories,
	services GetUnrecognizedExpendituresServices,
) *GetUnrecognizedExpendituresUseCase {
	return &GetUnrecognizedExpendituresUseCase{repositories: repositories, services: services}
}

// Execute performs the get-unrecognized-expenditures operation.
func (uc *GetUnrecognizedExpendituresUseCase) Execute(ctx context.Context, req *expenserecognitionpb.GetUnrecognizedExpendituresRequest) (*expenserecognitionpb.GetUnrecognizedExpendituresResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognition, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.ExpenseRecognition.GetUnrecognizedExpenditures(ctx, req)
}
