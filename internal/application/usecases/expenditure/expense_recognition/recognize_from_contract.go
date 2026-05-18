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

// RecognizeFromContractRepositories groups repository dependencies.
type RecognizeFromContractRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// RecognizeFromContractServices groups service dependencies.
type RecognizeFromContractServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// RecognizeFromContractUseCase converts a recurring contract cycle into a
// recognition row. Companion to the recurrence engine: each cycle for an
// accrual-basis workspace also emits a recognition (resolved via the schedule).
type RecognizeFromContractUseCase struct {
	repositories RecognizeFromContractRepositories
	services     RecognizeFromContractServices
}

// NewRecognizeFromContractUseCase creates a use case with grouped dependencies.
func NewRecognizeFromContractUseCase(
	repositories RecognizeFromContractRepositories,
	services RecognizeFromContractServices,
) *RecognizeFromContractUseCase {
	return &RecognizeFromContractUseCase{repositories: repositories, services: services}
}

// Execute performs the recognize-from-contract operation.
func (uc *RecognizeFromContractUseCase) Execute(ctx context.Context, req *expenserecognitionpb.RecognizeFromContractRequest) (*expenserecognitionpb.RecognizeFromContractResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognition, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.GetSupplierContractId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition.validation.contract_id_required", "Supplier contract ID is required [DEFAULT]"))
	}
	if req.GetCycleDate() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition.validation.cycle_date_required", "Cycle date is required [DEFAULT]"))
	}

	idempotencyKey := req.GetIdempotencyKey()
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("RECURRENCE:%s:%s", req.GetSupplierContractId(), req.GetCycleDate())
	}

	now := time.Now()
	id := uc.services.IDService.GenerateID()
	contractID := req.GetSupplierContractId()
	cycleDate := req.GetCycleDate()
	createReq := &expenserecognitionpb.CreateExpenseRecognitionRequest{
		Data: &expenserecognitionpb.ExpenseRecognition{
			Id:                 id,
			DateCreated:        &[]int64{now.UnixMilli()}[0],
			DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
			DateModified:       &[]int64{now.UnixMilli()}[0],
			DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
			Active:             true,
			Status:             expenserecognitionpb.ExpenseRecognitionStatus_EXPENSE_RECOGNITION_STATUS_DRAFT,
			SupplierContractId: &contractID,
			CycleDate:          &cycleDate,
			IdempotencyKey:     idempotencyKey,
			TotalAmount:        req.GetAmount(),
		},
	}
	createResp, err := uc.repositories.ExpenseRecognition.CreateExpenseRecognition(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create recognition from contract: %w", err)
	}
	var data *expenserecognitionpb.ExpenseRecognition
	if len(createResp.Data) > 0 {
		data = createResp.Data[0]
	}
	return &expenserecognitionpb.RecognizeFromContractResponse{Success: true, Data: data}, nil
}
