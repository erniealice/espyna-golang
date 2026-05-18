package expenserecognition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/shared/authcheck"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// ReverseExpenseRecognitionRepositories groups repository dependencies.
type ReverseExpenseRecognitionRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// ReverseExpenseRecognitionServices groups service dependencies.
type ReverseExpenseRecognitionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// ReverseExpenseRecognitionUseCase creates a reversing recognition row that nets to zero
// against the original. The original stays POSTED for audit trail; the reversal is a
// new row with status=REVERSED and reversal_of_recognition_id pointing back.
type ReverseExpenseRecognitionUseCase struct {
	repositories ReverseExpenseRecognitionRepositories
	services     ReverseExpenseRecognitionServices
}

// NewReverseExpenseRecognitionUseCase creates a use case with grouped dependencies.
func NewReverseExpenseRecognitionUseCase(
	repositories ReverseExpenseRecognitionRepositories,
	services ReverseExpenseRecognitionServices,
) *ReverseExpenseRecognitionUseCase {
	return &ReverseExpenseRecognitionUseCase{repositories: repositories, services: services}
}

// Execute performs the reverse operation.
func (uc *ReverseExpenseRecognitionUseCase) Execute(ctx context.Context, req *expenserecognitionpb.ReverseExpenseRecognitionRequest) (*expenserecognitionpb.ReverseExpenseRecognitionResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognition, ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.GetExpenseRecognitionId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition.validation.id_required", "Expense recognition ID is required [DEFAULT]"))
	}

	// Read the original to compose the reversal row.
	readResp, err := uc.repositories.ExpenseRecognition.ReadExpenseRecognition(ctx, &expenserecognitionpb.ReadExpenseRecognitionRequest{
		Data: &expenserecognitionpb.ExpenseRecognition{Id: req.GetExpenseRecognitionId()},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read original recognition: %w", err)
	}
	if len(readResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition.errors.not_found", "[ERR-DEFAULT] Expense recognition not found"))
	}
	original := readResp.Data[0]

	now := time.Now()
	id := uc.services.IDService.GenerateID()
	originalID := original.Id
	idempotencyKey := fmt.Sprintf("REVERSAL:%s:%s", originalID, now.UTC().Format("2006-01-02T15:04:05Z"))

	createReq := &expenserecognitionpb.CreateExpenseRecognitionRequest{
		Data: &expenserecognitionpb.ExpenseRecognition{
			Id:                      id,
			DateCreated:             &[]int64{now.UnixMilli()}[0],
			DateCreatedString:       &[]string{now.Format(time.RFC3339)}[0],
			DateModified:            &[]int64{now.UnixMilli()}[0],
			DateModifiedString:      &[]string{now.Format(time.RFC3339)}[0],
			Active:                  true,
			Status:                  expenserecognitionpb.ExpenseRecognitionStatus_EXPENSE_RECOGNITION_STATUS_REVERSED,
			Currency:                original.GetCurrency(),
			TotalAmount:             -original.GetTotalAmount(),
			ReversalOfRecognitionId: &originalID,
			IdempotencyKey:          idempotencyKey,
		},
	}
	createResp, err := uc.repositories.ExpenseRecognition.CreateExpenseRecognition(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create reversal recognition: %w", err)
	}
	var data *expenserecognitionpb.ExpenseRecognition
	if len(createResp.Data) > 0 {
		data = createResp.Data[0]
	}
	return &expenserecognitionpb.ReverseExpenseRecognitionResponse{Success: true, Data: data}, nil
}
