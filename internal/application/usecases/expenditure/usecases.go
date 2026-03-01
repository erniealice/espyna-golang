package expenditure

import (
	// Expenditure use cases
	expenditureUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure"
	expenditureAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure_attribute"
	expenditureCategoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure_category"
	expenditureLineItemUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/expenditure/expenditure_line_item"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for expenditure repositories
	expenditurepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure"
	expenditureattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
	expenditurecategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_category"
	expenditurelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_line_item"
)

// ExpenditureRepositories contains all expenditure domain repositories
type ExpenditureRepositories struct {
	Expenditure          expenditurepb.ExpenditureDomainServiceServer
	ExpenditureLineItem  expenditurelineitempb.ExpenditureLineItemDomainServiceServer
	ExpenditureCategory  expenditurecategorypb.ExpenditureCategoryDomainServiceServer
	ExpenditureAttribute expenditureattributepb.ExpenditureAttributeDomainServiceServer
}

// ExpenditureUseCases contains all expenditure-related use cases
type ExpenditureUseCases struct {
	Expenditure          *expenditureUseCases.UseCases
	ExpenditureLineItem  *expenditureLineItemUseCases.UseCases
	ExpenditureCategory  *expenditureCategoryUseCases.UseCases
	ExpenditureAttribute *expenditureAttributeUseCases.UseCases
}

// NewUseCases creates all expenditure use cases with proper constructor injection
func NewUseCases(
	repos ExpenditureRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *ExpenditureUseCases {
	expenditureUC := expenditureUseCases.NewUseCases(
		expenditureUseCases.ExpenditureRepositories{
			Expenditure: repos.Expenditure,
		},
		expenditureUseCases.ExpenditureServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	expenditureLineItemUC := expenditureLineItemUseCases.NewUseCases(
		expenditureLineItemUseCases.ExpenditureLineItemRepositories{
			ExpenditureLineItem: repos.ExpenditureLineItem,
		},
		expenditureLineItemUseCases.ExpenditureLineItemServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	expenditureCategoryUC := expenditureCategoryUseCases.NewUseCases(
		expenditureCategoryUseCases.ExpenditureCategoryRepositories{
			ExpenditureCategory: repos.ExpenditureCategory,
		},
		expenditureCategoryUseCases.ExpenditureCategoryServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	expenditureAttributeUC := expenditureAttributeUseCases.NewUseCases(
		expenditureAttributeUseCases.ExpenditureAttributeRepositories{
			ExpenditureAttribute: repos.ExpenditureAttribute,
		},
		expenditureAttributeUseCases.ExpenditureAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	return &ExpenditureUseCases{
		Expenditure:          expenditureUC,
		ExpenditureLineItem:  expenditureLineItemUC,
		ExpenditureCategory:  expenditureCategoryUC,
		ExpenditureAttribute: expenditureAttributeUC,
	}
}
