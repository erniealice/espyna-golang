package expenserecognition

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
)

// ExpenseRecognitionRepositories groups all repository dependencies.
type ExpenseRecognitionRepositories struct {
	ExpenseRecognition expenserecognitionpb.ExpenseRecognitionDomainServiceServer
}

// ExpenseRecognitionServices groups all service dependencies.
type ExpenseRecognitionServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all expense recognition use cases.
type UseCases struct {
	CreateExpenseRecognition    *CreateExpenseRecognitionUseCase
	ReadExpenseRecognition      *ReadExpenseRecognitionUseCase
	UpdateExpenseRecognition    *UpdateExpenseRecognitionUseCase
	DeleteExpenseRecognition    *DeleteExpenseRecognitionUseCase
	ListExpenseRecognitions     *ListExpenseRecognitionsUseCase
	RecognizeFromExpenditure    *RecognizeFromExpenditureUseCase
	RecognizeFromContract       *RecognizeFromContractUseCase
	ReverseExpenseRecognition   *ReverseExpenseRecognitionUseCase
	GetUnrecognizedExpenditures *GetUnrecognizedExpendituresUseCase
}

// NewUseCases creates a new collection of expense recognition use cases.
func NewUseCases(
	repositories ExpenseRecognitionRepositories,
	services ExpenseRecognitionServices,
) *UseCases {
	return &UseCases{
		CreateExpenseRecognition: NewCreateExpenseRecognitionUseCase(
			CreateExpenseRecognitionRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			CreateExpenseRecognitionServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadExpenseRecognition: NewReadExpenseRecognitionUseCase(
			ReadExpenseRecognitionRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			ReadExpenseRecognitionServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateExpenseRecognition: NewUpdateExpenseRecognitionUseCase(
			UpdateExpenseRecognitionRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			UpdateExpenseRecognitionServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteExpenseRecognition: NewDeleteExpenseRecognitionUseCase(
			DeleteExpenseRecognitionRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			DeleteExpenseRecognitionServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListExpenseRecognitions: NewListExpenseRecognitionsUseCase(
			ListExpenseRecognitionsRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			ListExpenseRecognitionsServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		RecognizeFromExpenditure: NewRecognizeFromExpenditureUseCase(
			RecognizeFromExpenditureRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			RecognizeFromExpenditureServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		RecognizeFromContract: NewRecognizeFromContractUseCase(
			RecognizeFromContractRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			RecognizeFromContractServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReverseExpenseRecognition: NewReverseExpenseRecognitionUseCase(
			ReverseExpenseRecognitionRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			ReverseExpenseRecognitionServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		GetUnrecognizedExpenditures: NewGetUnrecognizedExpendituresUseCase(
			GetUnrecognizedExpendituresRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			GetUnrecognizedExpendituresServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
