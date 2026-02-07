package common

import (
	"leapfor.xyz/espyna/internal/application/ports"
	attributeUseCases "leapfor.xyz/espyna/internal/application/usecases/common/attribute"
	categoryUseCases "leapfor.xyz/espyna/internal/application/usecases/common/category"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	categorypb "leapfor.xyz/esqyma/golang/v1/domain/common"
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
