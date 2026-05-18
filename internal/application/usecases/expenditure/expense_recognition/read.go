package expenserecognition

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// ReadExpenseRecognitionRepositories groups repository dependencies.
type ReadExpenseRecognitionRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// ReadExpenseRecognitionServices groups service dependencies.
type ReadExpenseRecognitionServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// ReadExpenseRecognitionUseCase handles reading a recognition.
type ReadExpenseRecognitionUseCase struct {
	repositories ReadExpenseRecognitionRepositories
	services     ReadExpenseRecognitionServices
}

// NewReadExpenseRecognitionUseCase creates a use case with grouped dependencies.
func NewReadExpenseRecognitionUseCase(
	repositories ReadExpenseRecognitionRepositories,
	services ReadExpenseRecognitionServices,
) *ReadExpenseRecognitionUseCase {
	return &ReadExpenseRecognitionUseCase{repositories: repositories, services: services}
}

// Execute performs the read operation.
func (uc *ReadExpenseRecognitionUseCase) Execute(ctx context.Context, req *expenserecognitionpb.ReadExpenseRecognitionRequest) (*expenserecognitionpb.ReadExpenseRecognitionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognition, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition.validation.id_required", "Expense recognition ID is required [DEFAULT]"))
	}
	return uc.repositories.ExpenseRecognition.ReadExpenseRecognition(ctx, req)
}
