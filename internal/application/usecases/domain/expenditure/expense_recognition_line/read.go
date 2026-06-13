package expenserecognitionline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

// ReadExpenseRecognitionLineRepositories groups repository dependencies.
type ReadExpenseRecognitionLineRepositories struct {
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
}

// ReadExpenseRecognitionLineServices groups service dependencies.
type ReadExpenseRecognitionLineServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// ReadExpenseRecognitionLineUseCase handles reading a recognition-line.
type ReadExpenseRecognitionLineUseCase struct {
	repositories ReadExpenseRecognitionLineRepositories
	services     ReadExpenseRecognitionLineServices
}

// NewReadExpenseRecognitionLineUseCase creates a use case with grouped dependencies.
func NewReadExpenseRecognitionLineUseCase(
	repositories ReadExpenseRecognitionLineRepositories,
	services ReadExpenseRecognitionLineServices,
) *ReadExpenseRecognitionLineUseCase {
	return &ReadExpenseRecognitionLineUseCase{repositories: repositories, services: services}
}

// Execute performs the read operation.
func (uc *ReadExpenseRecognitionLineUseCase) Execute(ctx context.Context, req *expenserecognitionlinepb.ReadExpenseRecognitionLineRequest) (*expenserecognitionlinepb.ReadExpenseRecognitionLineResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenseRecognitionLine,
		Action: entityid.ActionRead,
	}); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"expense_recognition_line.validation.id_required", "Recognition line ID is required [DEFAULT]"))
	}
	return uc.repositories.ExpenseRecognitionLine.ReadExpenseRecognitionLine(ctx, req)
}
