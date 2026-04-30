package expenserecognitionline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

// DeleteExpenseRecognitionLineRepositories groups repository dependencies.
type DeleteExpenseRecognitionLineRepositories struct {
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
}

// DeleteExpenseRecognitionLineServices groups service dependencies.
type DeleteExpenseRecognitionLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// DeleteExpenseRecognitionLineUseCase handles deleting a recognition-line.
type DeleteExpenseRecognitionLineUseCase struct {
	repositories DeleteExpenseRecognitionLineRepositories
	services     DeleteExpenseRecognitionLineServices
}

// NewDeleteExpenseRecognitionLineUseCase creates a use case with grouped dependencies.
func NewDeleteExpenseRecognitionLineUseCase(
	repositories DeleteExpenseRecognitionLineRepositories,
	services DeleteExpenseRecognitionLineServices,
) *DeleteExpenseRecognitionLineUseCase {
	return &DeleteExpenseRecognitionLineUseCase{repositories: repositories, services: services}
}

// Execute performs the delete operation.
func (uc *DeleteExpenseRecognitionLineUseCase) Execute(ctx context.Context, req *expenserecognitionlinepb.DeleteExpenseRecognitionLineRequest) (*expenserecognitionlinepb.DeleteExpenseRecognitionLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognitionLine, ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition_line.validation.id_required", "Recognition line ID is required [DEFAULT]"))
	}
	return uc.repositories.ExpenseRecognitionLine.DeleteExpenseRecognitionLine(ctx, req)
}
