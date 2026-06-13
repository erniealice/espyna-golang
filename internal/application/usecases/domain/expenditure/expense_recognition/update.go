package expenserecognition

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// UpdateExpenseRecognitionRepositories groups repository dependencies.
type UpdateExpenseRecognitionRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// UpdateExpenseRecognitionServices groups service dependencies.
type UpdateExpenseRecognitionServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenseRecognition,
		Action: entityid.ActionUpdate,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"expense_recognition.validation.id_required", "Expense recognition ID is required [DEFAULT]"))
	}
	now := time.Now()
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]

	return uc.repositories.ExpenseRecognition.UpdateExpenseRecognition(ctx, req)
}
