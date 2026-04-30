package expenserecognitionline

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

// ExpenseRecognitionLineRepositories groups all repository dependencies.
type ExpenseRecognitionLineRepositories struct {
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
}

// ExpenseRecognitionLineServices groups all service dependencies.
type ExpenseRecognitionLineServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all expense recognition line use cases.
type UseCases struct {
	CreateExpenseRecognitionLine *CreateExpenseRecognitionLineUseCase
	ReadExpenseRecognitionLine   *ReadExpenseRecognitionLineUseCase
	UpdateExpenseRecognitionLine *UpdateExpenseRecognitionLineUseCase
	DeleteExpenseRecognitionLine *DeleteExpenseRecognitionLineUseCase
	ListExpenseRecognitionLines  *ListExpenseRecognitionLinesUseCase
}

// NewUseCases creates a new collection of expense recognition line use cases.
func NewUseCases(
	repositories ExpenseRecognitionLineRepositories,
	services ExpenseRecognitionLineServices,
) *UseCases {
	return &UseCases{
		CreateExpenseRecognitionLine: NewCreateExpenseRecognitionLineUseCase(
			CreateExpenseRecognitionLineRepositories{ExpenseRecognitionLine: repositories.ExpenseRecognitionLine},
			CreateExpenseRecognitionLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadExpenseRecognitionLine: NewReadExpenseRecognitionLineUseCase(
			ReadExpenseRecognitionLineRepositories{ExpenseRecognitionLine: repositories.ExpenseRecognitionLine},
			ReadExpenseRecognitionLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateExpenseRecognitionLine: NewUpdateExpenseRecognitionLineUseCase(
			UpdateExpenseRecognitionLineRepositories{ExpenseRecognitionLine: repositories.ExpenseRecognitionLine},
			UpdateExpenseRecognitionLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteExpenseRecognitionLine: NewDeleteExpenseRecognitionLineUseCase(
			DeleteExpenseRecognitionLineRepositories{ExpenseRecognitionLine: repositories.ExpenseRecognitionLine},
			DeleteExpenseRecognitionLineServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListExpenseRecognitionLines: NewListExpenseRecognitionLinesUseCase(
			ListExpenseRecognitionLinesRepositories{ExpenseRecognitionLine: repositories.ExpenseRecognitionLine},
			ListExpenseRecognitionLinesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
