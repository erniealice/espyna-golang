package expenserecognition

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// UpdateExpenseRecognitionRepositories groups repository dependencies.
type UpdateExpenseRecognitionRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// UpdateExpenseRecognitionServices groups service dependencies.
type UpdateExpenseRecognitionServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UpdateExpenseRecognitionUseCase handles updating a recognition.
type UpdateExpenseRecognitionUseCase struct {
	repositories UpdateExpenseRecognitionRepositories
	services     UpdateExpenseRecognitionServices
}

// NewUpdateExpenseRecognitionUseCase creates a use case with grouped dependencies.
func NewUpdateExpenseRecognitionUseCase(
	repositories UpdateExpenseRecognitionRepositories,
	services UpdateExpenseRecognitionServices,
) *UpdateExpenseRecognitionUseCase {
	return &UpdateExpenseRecognitionUseCase{repositories: repositories, services: services}
}

// Execute performs the update operation.
func (uc *UpdateExpenseRecognitionUseCase) Execute(ctx context.Context, req *expenserecognitionpb.UpdateExpenseRecognitionRequest) (*expenserecognitionpb.UpdateExpenseRecognitionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognition, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition.validation.id_required", "Expense recognition ID is required [DEFAULT]"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.ExpenseRecognition.UpdateExpenseRecognition(ctx, req)
}
