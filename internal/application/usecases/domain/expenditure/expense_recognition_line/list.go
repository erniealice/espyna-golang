package expenserecognitionline

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

// ListExpenseRecognitionLinesRepositories groups repository dependencies.
type ListExpenseRecognitionLinesRepositories struct {
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
}

// ListExpenseRecognitionLinesServices groups service dependencies.
type ListExpenseRecognitionLinesServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
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
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenseRecognitionLine,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	return uc.repositories.ExpenseRecognitionLine.ListExpenseRecognitionLines(ctx, req)
}
