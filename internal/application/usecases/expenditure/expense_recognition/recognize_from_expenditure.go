package expenserecognition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// RecognizeFromExpenditureRepositories groups repository dependencies.
type RecognizeFromExpenditureRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// RecognizeFromExpenditureServices groups service dependencies.
type RecognizeFromExpenditureServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// RecognizeFromExpenditureUseCase converts a posted Expenditure into one or more
// ExpenseRecognition rows. Routine pattern: derive idempotency_key, build the
// recognition row, persist via the underlying CRUD adapter. Multi-period
// amortization (e.g. annual prepayment recognized monthly) is driven by the
// caller emitting multiple calls with distinct recognition_period values.
type RecognizeFromExpenditureUseCase struct {
	repositories RecognizeFromExpenditureRepositories
	services     RecognizeFromExpenditureServices
}

// NewRecognizeFromExpenditureUseCase creates a use case with grouped dependencies.
func NewRecognizeFromExpenditureUseCase(
	repositories RecognizeFromExpenditureRepositories,
	services RecognizeFromExpenditureServices,
) *RecognizeFromExpenditureUseCase {
	return &RecognizeFromExpenditureUseCase{repositories: repositories, services: services}
}

// Execute performs the recognize-from-expenditure operation.
func (uc *RecognizeFromExpenditureUseCase) Execute(ctx context.Context, req *expenserecognitionpb.RecognizeFromExpenditureRequest) (*expenserecognitionpb.RecognizeFromExpenditureResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		entityExpenseRecognition, ports.ActionCreate); err != nil {
		return nil, err
	}
	if req == nil || req.GetExpenditureId() == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService,
			"expense_recognition.validation.expenditure_id_required", "Expenditure ID is required [DEFAULT]"))
	}

	period := req.GetRecognitionPeriod()
	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}

	// Derive idempotency_key per HIGH-2 amendment when the caller hasn't provided one.
	idempotencyKey := req.GetIdempotencyKey()
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("EXPENDITURE:%s:%s", req.GetExpenditureId(), period)
	}

	now := time.Now()
	id := uc.services.IDService.GenerateID()
	expenditureID := req.GetExpenditureId()
	createReq := &expenserecognitionpb.CreateExpenseRecognitionRequest{
		Data: &expenserecognitionpb.ExpenseRecognition{
			Id:                 id,
			DateCreated:        &[]int64{now.UnixMilli()}[0],
			DateCreatedString:  &[]string{now.Format(time.RFC3339)}[0],
			DateModified:       &[]int64{now.UnixMilli()}[0],
			DateModifiedString: &[]string{now.Format(time.RFC3339)}[0],
			Active:             true,
			Status:             expenserecognitionpb.ExpenseRecognitionStatus_EXPENSE_RECOGNITION_STATUS_DRAFT,
			ExpenditureId:      &expenditureID,
			IdempotencyKey:     idempotencyKey,
		},
	}
	createResp, err := uc.repositories.ExpenseRecognition.CreateExpenseRecognition(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create recognition from expenditure: %w", err)
	}
	var data *expenserecognitionpb.ExpenseRecognition
	if len(createResp.Data) > 0 {
		data = createResp.Data[0]
	}
	return &expenserecognitionpb.RecognizeFromExpenditureResponse{Success: true, Data: data}, nil
}
