package expenserecognition

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// ReadExpenseRecognitionRepositories groups repository dependencies.
type ReadExpenseRecognitionRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// ReadExpenseRecognitionServices groups service dependencies.
type ReadExpenseRecognitionServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenseRecognition,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"expense_recognition.validation.id_required", "Expense recognition ID is required [DEFAULT]"))
	}
	return uc.repositories.ExpenseRecognition.ReadExpenseRecognition(ctx, req)
}
