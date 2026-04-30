package expenserecognitionline

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

// ListExpenseRecognitionLinesRepositories groups repository dependencies.
type ListExpenseRecognitionLinesRepositories struct {
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
}

// ListExpenseRecognitionLinesServices groups service dependencies.
type ListExpenseRecognitionLinesServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ListExpenseRecognitionLinesUseCase handles listing recognition-lines.
type ListExpenseRecognitionLinesUseCase struct {
	repositories ListExpenseRecognitionLinesRepositories
	services     ListExpenseRecognitionLinesServices
}

// NewListExpenseRecognitionLinesUseCase creates a use case with grouped dependencies.
func NewListExpenseRecognitionLinesUseCase(
	repositories ListExpenseRecognitionLinesRepositories,
	services ListExpenseRecognitionLinesServices,
) *ListExpenseRecognitionLinesUseCase {
	return &ListExpenseRecognitionLinesUseCase{repositories: repositories, services: services}
}

// Execute performs the list operation.
func (uc *ListExpenseRecognitionLinesUseCase) Execute(ctx context.Context, req *expenserecognitionlinepb.ListExpenseRecognitionLinesRequest) (*expenserecognitionlinepb.ListExpenseRecognitionLinesResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognitionLine, ports.ActionList); err != nil {
		return nil, err
	}
	return uc.repositories.ExpenseRecognitionLine.ListExpenseRecognitionLines(ctx, req)
}
