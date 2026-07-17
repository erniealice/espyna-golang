package common

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	attributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/common/attribute"
	attributeValueUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/common/attribute_value"
	categoryUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/domain/common/category"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	categorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
)

// CommonUseCases contains all common domain use cases
type CommonUseCases struct {
	Attribute      *attributeUseCases.UseCases
	AttributeValue *attributeValueUseCases.UseCases
	Category       *categoryUseCases.UseCases
}

// NewCommonUseCases creates a new collection of common use cases
func NewCommonUseCases(
	attributeRepo attributepb.AttributeDomainServiceServer,
	attributeValueRepo attributepb.AttributeValueDomainServiceServer,
	categoryRepo categorypb.CategoryDomainServiceServer,
	translationService ports.Translator,
	idService ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) *CommonUseCases {
	uc := &CommonUseCases{}

	// Initialize attribute use cases only if repo is available
	if attributeRepo != nil {
		attributeRepositories := attributeUseCases.AttributeRepositories{
			Attribute: attributeRepo,
		}
		attributeServices := attributeUseCases.AttributeServices{
			Transactor:       ports.NewNoOpTransactor(),
			Translator:       translationService,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		}
		uc.Attribute = attributeUseCases.NewUseCases(attributeRepositories, attributeServices)
	}

	// Initialize attribute_value use cases only if repo is available.
	// Surfaces uc.Common.AttributeValue.ListAttributeValues.Execute — the generic
	// list path that preserves the av.label column for the client-attribute drawer.
	if attributeValueRepo != nil {
		attributeValueRepositories := attributeValueUseCases.AttributeValueRepositories{
			AttributeValue: attributeValueRepo,
		}
		attributeValueServices := attributeValueUseCases.AttributeValueServices{
			Transactor:       ports.NewNoOpTransactor(),
			Translator:       translationService,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		}
		uc.AttributeValue = attributeValueUseCases.NewUseCases(attributeValueRepositories, attributeValueServices)
	}

	// Initialize category use cases only if repo is available
	if categoryRepo != nil {
		categoryRepositories := categoryUseCases.CategoryRepositories{
			Category: categoryRepo,
		}
		categoryServices := categoryUseCases.CategoryServices{
			Transactor:       ports.NewNoOpTransactor(),
			Translator:       translationService,
			IDGenerator:      idService,
			ActionGatekeeper: actionGate,
		}
		uc.Category = categoryUseCases.NewUseCases(categoryRepositories, categoryServices)
	}

	return uc
}
