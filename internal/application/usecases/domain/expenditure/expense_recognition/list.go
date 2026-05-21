package expenserecognition

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// ListExpenseRecognitionsRepositories groups repository dependencies.
type ListExpenseRecognitionsRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// ListExpenseRecognitionsServices groups service dependencies.
type ListExpenseRecognitionsServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
}

// ListExpenseRecognitionsUseCase handles listing recognitions.
type ListExpenseRecognitionsUseCase struct {
	repositories ListExpenseRecognitionsRepositories
	services     ListExpenseRecognitionsServices
}

// NewListExpenseRecognitionsUseCase creates a use case with grouped dependencies.
func NewListExpenseRecognitionsUseCase(
	repositories ListExpenseRecognitionsRepositories,
	services ListExpenseRecognitionsServices,
) *ListExpenseRecognitionsUseCase {
	return &ListExpenseRecognitionsUseCase{repositories: repositories, services: services}
}

// Execute performs the list operation.
func (uc *ListExpenseRecognitionsUseCase) Execute(ctx context.Context, req *expenserecognitionpb.ListExpenseRecognitionsRequest) (*expenserecognitionpb.ListExpenseRecognitionsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenseRecognition, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.ExpenseRecognition.ListExpenseRecognitions(ctx, req)
}
