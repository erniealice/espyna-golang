package expenserecognitionline

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

// UpdateExpenseRecognitionLineRepositories groups repository dependencies.
type UpdateExpenseRecognitionLineRepositories struct {
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
}

// UpdateExpenseRecognitionLineServices groups service dependencies.
type UpdateExpenseRecognitionLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UpdateExpenseRecognitionLineUseCase handles updating a recognition-line.
type UpdateExpenseRecognitionLineUseCase struct {
	repositories UpdateExpenseRecognitionLineRepositories
	services     UpdateExpenseRecognitionLineServices
}

// NewUpdateExpenseRecognitionLineUseCase creates a use case with grouped dependencies.
func NewUpdateExpenseRecognitionLineUseCase(
	repositories UpdateExpenseRecognitionLineRepositories,
	services UpdateExpenseRecognitionLineServices,
) *UpdateExpenseRecognitionLineUseCase {
	return &UpdateExpenseRecognitionLineUseCase{repositories: repositories, services: services}
}

// Execute performs the update operation.
func (uc *UpdateExpenseRecognitionLineUseCase) Execute(ctx context.Context, req *expenserecognitionlinepb.UpdateExpenseRecognitionLineRequest) (*expenserecognitionlinepb.UpdateExpenseRecognitionLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognitionLine, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition_line.validation.id_required", "Recognition line ID is required [DEFAULT]"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	return uc.repositories.ExpenseRecognitionLine.UpdateExpenseRecognitionLine(ctx, req)
}
