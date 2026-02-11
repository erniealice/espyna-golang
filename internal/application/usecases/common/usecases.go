package common

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/common/attribute"
	categoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/common/category"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	categorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// CommonUseCases contains all common domain use cases
type CommonUseCases struct {
	Attribute *attributeUseCases.UseCases
	Category  *categoryUseCases.UseCases
}

// NewCommonUseCases creates a new collection of common use cases
func NewCommonUseCases(
	attributeRepo attributepb.AttributeDomainServiceServer,
	categoryRepo categorypb.CategoryDomainServiceServer,
	translationService ports.TranslationService,
	idService ports.IDService,
) *CommonUseCases {
	uc := &CommonUseCases{}

	// Initialize attribute use cases only if repo is available
	if attributeRepo != nil {
		attributeRepositories := attributeUseCases.AttributeRepositories{
			Attribute: attributeRepo,
		}
		attributeServices := attributeUseCases.AttributeServices{
			TransactionService: ports.NewNoOpTransactionService(),
			TranslationService: translationService,
			IDService:          idService,
		}
		uc.Attribute = attributeUseCases.NewUseCases(attributeRepositories, attributeServices)
	}

	// Initialize category use cases only if repo is available
	if categoryRepo != nil {
		categoryRepositories := categoryUseCases.CategoryRepositories{
			Category: categoryRepo,
		}
		categoryServices := categoryUseCases.CategoryServices{
			TransactionService: ports.NewNoOpTransactionService(),
			TranslationService: translationService,
			IDService:          idService,
		}
		uc.Category = categoryUseCases.NewUseCases(categoryRepositories, categoryServices)
	}

	return uc
}
