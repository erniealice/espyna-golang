package expenserecognitionline

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

// ReadExpenseRecognitionLineRepositories groups repository dependencies.
type ReadExpenseRecognitionLineRepositories struct {
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
}

// ReadExpenseRecognitionLineServices groups service dependencies.
type ReadExpenseRecognitionLineServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
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
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognitionLine, ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition_line.validation.id_required", "Recognition line ID is required [DEFAULT]"))
	}
	return uc.repositories.ExpenseRecognitionLine.ReadExpenseRecognitionLine(ctx, req)
}
