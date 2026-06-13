package expenserecognition

import (
	"context"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/registry/entityid"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// GetUnrecognizedExpendituresRepositories groups repository dependencies.
type GetUnrecognizedExpendituresRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// GetUnrecognizedExpendituresServices groups service dependencies.
type GetUnrecognizedExpendituresServices struct {
	Authorizer ports.Authorizer
	Translator ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

// GetUnrecognizedExpendituresUseCase returns the IDs of expenditures lacking a
// POSTED ExpenseRecognition row. Surfaced in the AP team's review queue.
type GetUnrecognizedExpendituresUseCase struct {
	repositories GetUnrecognizedExpendituresRepositories
	services     GetUnrecognizedExpendituresServices
}

// NewGetUnrecognizedExpendituresUseCase creates a use case with grouped dependencies.
func NewGetUnrecognizedExpendituresUseCase(
	repositories GetUnrecognizedExpendituresRepositories,
	services GetUnrecognizedExpendituresServices,
) *GetUnrecognizedExpendituresUseCase {
	return &GetUnrecognizedExpendituresUseCase{repositories: repositories, services: services}
}

// Execute performs the get-unrecognized-expenditures operation.
func (uc *GetUnrecognizedExpendituresUseCase) Execute(ctx context.Context, req *expenserecognitionpb.GetUnrecognizedExpendituresRequest) (*expenserecognitionpb.GetUnrecognizedExpendituresResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{
		Entity: entityExpenseRecognition,
		Action: entityid.ActionList,
	}); err != nil {
		return nil, err
	}
	return uc.repositories.ExpenseRecognition.GetUnrecognizedExpenditures(ctx, req)
}
