package treasury

import (
	// Collection use cases
	collectionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/collection"
	// Disbursement use cases
	disbursementUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/treasury/disbursement"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Protobuf domain services for treasury repositories
	collectionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/collection"
	disbursementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/disbursement"
)

// TreasuryRepositories contains all treasury domain repositories
type TreasuryRepositories struct {
	Collection   collectionpb.CollectionDomainServiceServer
	Disbursement disbursementpb.DisbursementDomainServiceServer
}

// TreasuryUseCases contains all treasury-related use cases
type TreasuryUseCases struct {
	Collection   *collectionUseCases.UseCases
	Disbursement *disbursementUseCases.UseCases
}

// NewUseCases creates all treasury use cases with proper constructor injection
func NewUseCases(
	repos TreasuryRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *TreasuryUseCases {
	collectionUC := collectionUseCases.NewUseCases(
		collectionUseCases.CollectionRepositories{
			Collection: repos.Collection,
		},
		collectionUseCases.CollectionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	disbursementUC := disbursementUseCases.NewUseCases(
		disbursementUseCases.DisbursementRepositories{
			Disbursement: repos.Disbursement,
		},
		disbursementUseCases.DisbursementServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	return &TreasuryUseCases{
		Collection:   collectionUC,
		Disbursement: disbursementUC,
	}
}
