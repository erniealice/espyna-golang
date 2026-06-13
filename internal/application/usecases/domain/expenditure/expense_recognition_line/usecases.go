package expenserecognitionline

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
)

// ExpenseRecognitionLineRepositories groups all repository dependencies.
type ExpenseRecognitionLineRepositories struct {
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
}

// ExpenseRecognitionLineServices groups all service dependencies.
type ExpenseRecognitionLineServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
				Authorizer:  services.Authorizer,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadExpenseRecognitionLine: NewReadExpenseRecognitionLineUseCase(
			ReadExpenseRecognitionLineRepositories{ExpenseRecognitionLine: repositories.ExpenseRecognitionLine},
			ReadExpenseRecognitionLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		UpdateExpenseRecognitionLine: NewUpdateExpenseRecognitionLineUseCase(
			UpdateExpenseRecognitionLineRepositories{ExpenseRecognitionLine: repositories.ExpenseRecognitionLine},
			UpdateExpenseRecognitionLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		DeleteExpenseRecognitionLine: NewDeleteExpenseRecognitionLineUseCase(
			DeleteExpenseRecognitionLineRepositories{ExpenseRecognitionLine: repositories.ExpenseRecognitionLine},
			DeleteExpenseRecognitionLineServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ListExpenseRecognitionLines: NewListExpenseRecognitionLinesUseCase(
			ListExpenseRecognitionLinesRepositories{ExpenseRecognitionLine: repositories.ExpenseRecognitionLine},
			ListExpenseRecognitionLinesServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
	}
}
