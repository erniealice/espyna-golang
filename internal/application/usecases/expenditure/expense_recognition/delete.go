package expenserecognition

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// DeleteExpenseRecognitionRepositories groups repository dependencies.
type DeleteExpenseRecognitionRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// DeleteExpenseRecognitionServices groups service dependencies.
type DeleteExpenseRecognitionServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// DeleteExpenseRecognitionUseCase handles deleting a recognition.
type DeleteExpenseRecognitionUseCase struct {
	repositories DeleteExpenseRecognitionRepositories
	services     DeleteExpenseRecognitionServices
}

// NewDeleteExpenseRecognitionUseCase creates a use case with grouped dependencies.
func NewDeleteExpenseRecognitionUseCase(
	repositories DeleteExpenseRecognitionRepositories,
	services DeleteExpenseRecognitionServices,
) *DeleteExpenseRecognitionUseCase {
	return &DeleteExpenseRecognitionUseCase{repositories: repositories, services: services}
}

// Execute performs the delete operation.
func (uc *DeleteExpenseRecognitionUseCase) Execute(ctx context.Context, req *expenserecognitionpb.DeleteExpenseRecognitionRequest) (*expenserecognitionpb.DeleteExpenseRecognitionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognition, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition.validation.id_required", "Expense recognition ID is required [DEFAULT]"))
	}
	return uc.repositories.ExpenseRecognition.DeleteExpenseRecognition(ctx, req)
}
