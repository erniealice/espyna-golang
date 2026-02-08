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
	// Build grouped parameters for attribute use cases
	attributeRepositories := attributeUseCases.AttributeRepositories{
		Attribute: attributeRepo,
	}
	attributeServices := attributeUseCases.AttributeServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: translationService,
		IDService:          idService,
	}

	// Build grouped parameters for category use cases
	categoryRepositories := categoryUseCases.CategoryRepositories{
		Category: categoryRepo,
	}
	categoryServices := categoryUseCases.CategoryServices{
		TransactionService: ports.NewNoOpTransactionService(),
		TranslationService: translationService,
		IDService:          idService,
	}

	return &CommonUseCases{
		Attribute: attributeUseCases.NewUseCases(attributeRepositories, attributeServices),
		Category:  categoryUseCases.NewUseCases(categoryRepositories, categoryServices),
	}
}
