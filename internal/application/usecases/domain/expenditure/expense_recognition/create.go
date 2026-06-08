package expenserecognition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	"github.com/erniealice/espyna-golang/registry/entityid"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

const entityExpenseRecognition = "expense_recognition"

// CreateExpenseRecognitionRepositories groups repository dependencies.
type CreateExpenseRecognitionRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// CreateExpenseRecognitionServices groups service dependencies.
type CreateExpenseRecognitionServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	IDGenerator ports.IDGenerator
}

// CreateExpenseRecognitionUseCase handles creating a new recognition row.
//
// Idempotency: callers should populate idempotency_key BEFORE calling this use case
// (the recurrence engine and manual paths derive the key per HIGH-2 amendment).
// The DB-side unique index on idempotency_key is the final guard.
type CreateExpenseRecognitionUseCase struct {
	repositories CreateExpenseRecognitionRepositories
	services     CreateExpenseRecognitionServices
}

// NewCreateExpenseRecognitionUseCase creates a use case with grouped dependencies.
func NewCreateExpenseRecognitionUseCase(
	repositories CreateExpenseRecognitionRepositories,
	services CreateExpenseRecognitionServices,
) *CreateExpenseRecognitionUseCase {
	return &CreateExpenseRecognitionUseCase{repositories: repositories, services: services}
}

// Execute performs the create operation.
func (uc *CreateExpenseRecognitionUseCase) Execute(ctx context.Context, req *expenserecognitionpb.CreateExpenseRecognitionRequest) (*expenserecognitionpb.CreateExpenseRecognitionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.Authorizer, uc.services.Translator,
		entityExpenseRecognition, entityid.ActionCreate); err != nil {
		return nil, err
	}

	if uc.services.Transactor != nil && uc.services.Transactor.SupportsTransactions() {
		var result *expenserecognitionpb.CreateExpenseRecognitionResponse
		err := uc.services.Transactor.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, req)
			if err != nil {
				return fmt.Errorf("expense recognition creation failed: %w", err)
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return uc.executeCore(ctx, req)
}

func (uc *CreateExpenseRecognitionUseCase) executeCore(ctx context.Context, req *expenserecognitionpb.CreateExpenseRecognitionRequest) (*expenserecognitionpb.CreateExpenseRecognitionResponse, error) {
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator,
			"expense_recognition.validation.data_required", "Expense recognition data is required [DEFAULT]"))
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

	if req.Data.Status == expenserecognitionpb.ExpenseRecognitionStatus_EXPENSE_RECOGNITION_STATUS_UNSPECIFIED {
		req.Data.Status = expenserecognitionpb.ExpenseRecognitionStatus_EXPENSE_RECOGNITION_STATUS_DRAFT
	}

	return uc.repositories.ExpenseRecognition.CreateExpenseRecognition(ctx, req)
}
