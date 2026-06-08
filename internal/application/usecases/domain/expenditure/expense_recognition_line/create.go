package expenserecognitionline

import (
	"context"
	"errors"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

const entityExpenseRecognitionLine = "expense_recognition_line"

// CreateExpenseRecognitionLineRepositories groups repository dependencies.
type CreateExpenseRecognitionLineRepositories struct {
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
}

// CreateExpenseRecognitionLineServices groups service dependencies.
type CreateExpenseRecognitionLineServices struct {
	Authorizer  ports.Authorizer
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateExpenseRecognitionLineUseCase handles creating a new recognition-line.
type CreateExpenseRecognitionLineUseCase struct {
	repositories CreateExpenseRecognitionLineRepositories
	services     CreateExpenseRecognitionLineServices
}

// NewCreateExpenseRecognitionLineUseCase creates a use case with grouped dependencies.
func NewCreateExpenseRecognitionLineUseCase(
	repositories CreateExpenseRecognitionLineRepositories,
	services CreateExpenseRecognitionLineServices,
) *CreateExpenseRecognitionLineUseCase {
	return &CreateExpenseRecognitionLineUseCase{repositories: repositories, services: services}
}

// Execute performs the create operation.
func (uc *CreateExpenseRecognitionLineUseCase) Execute(ctx context.Context, req *expenserecognitionlinepb.CreateExpenseRecognitionLineRequest) (*expenserecognitionlinepb.CreateExpenseRecognitionLineResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenseRecognitionLine, entityid.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"expense_recognition_line.validation.data_required", "Recognition line data is required [DEFAULT]"))
	}
	now := time.Now()
	if req.Data.Id == "" {
		req.Data.Id = uc.services.IDGenerator.GenerateID()
	}
	req.Data.DateCreated = &[]int64{now.UnixMilli()}[0]
	req.Data.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.DateModified = &[]int64{now.UnixMilli()}[0]
	req.Data.DateModifiedString = &[]string{now.Format(time.RFC3339)}[0]
	req.Data.Active = true
	return uc.repositories.ExpenseRecognitionLine.CreateExpenseRecognitionLine(ctx, req)
}
