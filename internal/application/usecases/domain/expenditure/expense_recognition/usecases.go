package expenserecognition

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
	expenserecognitionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition"
	expenserecognitionlinepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expense_recognition_line"
	suppliersubscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/procurement/supplier_subscription"
)

// ExpenseRecognitionRepositories groups all repository dependencies.
type ExpenseRecognitionRepositories struct {
	ExpenseRecognition     expenserecognitionpb.ExpenseRecognitionDomainServiceServer
	ExpenseRecognitionLine expenserecognitionlinepb.ExpenseRecognitionLineDomainServiceServer
	Expenditure            expenditurepb.ExpenditureDomainServiceServer
	ExpenditureLineItem    expenditurelineitempb.ExpenditureLineItemDomainServiceServer
	// Optional: when set, supplier subscription workspace ownership is validated on
	// RecognizeFromExpenditure calls that carry a supplier_subscription_id.
	SupplierSubscription suppliersubscriptionpb.SupplierSubscriptionDomainServiceServer
}

// ExpenseRecognitionServices groups all service dependencies.
type ExpenseRecognitionServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all expense recognition use cases.
//
// 20260518-hexagonal-strict-adherence Phase 3 — RecognizeFromSupplierSubscription
// (formerly a flat field on ExpenditureUseCases) now nests here. The parent
// expenditure aggregator post-assigns it because the use case needs cross-domain
// (procurement) repositories.
type UseCases struct {
	CreateExpenseRecognition          *CreateExpenseRecognitionUseCase
	ReadExpenseRecognition            *ReadExpenseRecognitionUseCase
	UpdateExpenseRecognition          *UpdateExpenseRecognitionUseCase
	DeleteExpenseRecognition          *DeleteExpenseRecognitionUseCase
	ListExpenseRecognitions           *ListExpenseRecognitionsUseCase
	RecognizeFromExpenditure          *RecognizeFromExpenditureUseCase
	RecognizeFromContract             *RecognizeFromContractUseCase
	RecognizeFromSupplierSubscription *RecognizeExpenseFromSupplierSubscriptionUseCase
	ReverseExpenseRecognition         *ReverseExpenseRecognitionUseCase
	GetUnrecognizedExpenditures       *GetUnrecognizedExpendituresUseCase
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
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadExpenseRecognition: NewReadExpenseRecognitionUseCase(
			ReadExpenseRecognitionRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			ReadExpenseRecognitionServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		UpdateExpenseRecognition: NewUpdateExpenseRecognitionUseCase(
			UpdateExpenseRecognitionRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			UpdateExpenseRecognitionServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		DeleteExpenseRecognition: NewDeleteExpenseRecognitionUseCase(
			DeleteExpenseRecognitionRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			DeleteExpenseRecognitionServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		ListExpenseRecognitions: NewListExpenseRecognitionsUseCase(
			ListExpenseRecognitionsRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			ListExpenseRecognitionsServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
		RecognizeFromExpenditure: NewRecognizeFromExpenditureUseCase(
			RecognizeFromExpenditureRepositories{
				ExpenseRecognition:     repositories.ExpenseRecognition,
				ExpenseRecognitionLine: repositories.ExpenseRecognitionLine,
				Expenditure:            repositories.Expenditure,
				ExpenditureLineItem:    repositories.ExpenditureLineItem,
				SupplierSubscription:   repositories.SupplierSubscription,
			},
			RecognizeFromExpenditureServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
		),
		RecognizeFromContract: NewRecognizeFromContractUseCase(
			RecognizeFromContractRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			RecognizeFromContractServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
		),
		ReverseExpenseRecognition: NewReverseExpenseRecognitionUseCase(
			ReverseExpenseRecognitionRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			ReverseExpenseRecognitionServices{
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
				IDGenerator: services.IDGenerator,
			},
		),
		GetUnrecognizedExpenditures: NewGetUnrecognizedExpendituresUseCase(
			GetUnrecognizedExpendituresRepositories{ExpenseRecognition: repositories.ExpenseRecognition},
			GetUnrecognizedExpendituresServices{
				Authorizer: services.Authorizer,
				Translator: services.Translator,
				ActionGatekeeper: services.ActionGatekeeper,
			},
		),
	}
}
