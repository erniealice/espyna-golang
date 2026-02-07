package delegate_attribute

import (
	"leapfor.xyz/espyna/internal/application/ports"
	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	delegatepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate"
	delegateattributepb "leapfor.xyz/esqyma/golang/v1/domain/entity/delegate_attribute"
)

// UseCases contains all delegate attribute-related use cases
type UseCases struct {
	CreateDelegateAttribute          *CreateDelegateAttributeUseCase
	ReadDelegateAttribute            *ReadDelegateAttributeUseCase
	UpdateDelegateAttribute          *UpdateDelegateAttributeUseCase
	DeleteDelegateAttribute          *DeleteDelegateAttributeUseCase
	ListDelegateAttributes           *ListDelegateAttributesUseCase
	GetDelegateAttributeListPageData *GetDelegateAttributeListPageDataUseCase
	GetDelegateAttributeItemPageData *GetDelegateAttributeItemPageDataUseCase
}

// DelegateAttributeRepositories groups all repository dependencies for delegate attribute use cases
type DelegateAttributeRepositories struct {
	DelegateAttribute delegateattributepb.DelegateAttributeDomainServiceServer // Primary entity repository
	Delegate          delegatepb.DelegateDomainServiceServer                   // Entity reference validation
	Attribute         attributepb.AttributeDomainServiceServer                 // Entity reference validation
}

// DelegateAttributeServices groups all business service dependencies for delegate attribute use cases
type DelegateAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of delegate attribute use cases
func NewUseCases(
	repositories DelegateAttributeRepositories,
	services DelegateAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateDelegateAttributeRepositories(repositories)
	createServices := CreateDelegateAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	readRepos := ReadDelegateAttributeRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
	}
	readServices := ReadDelegateAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateDelegateAttributeRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
		Delegate:          repositories.Delegate,
		Attribute:         repositories.Attribute,
	}
	updateServices := UpdateDelegateAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteDelegateAttributeRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
	}
	deleteServices := DeleteDelegateAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListDelegateAttributesRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
	}
	listServices := ListDelegateAttributesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getListPageDataRepos := GetDelegateAttributeListPageDataRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
	}
	getListPageDataServices := GetDelegateAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetDelegateAttributeItemPageDataRepositories{
		DelegateAttribute: repositories.DelegateAttribute,
	}
	getItemPageDataServices := GetDelegateAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateDelegateAttribute:          NewCreateDelegateAttributeUseCase(createRepos, createServices),
		ReadDelegateAttribute:            NewReadDelegateAttributeUseCase(readRepos, readServices),
		UpdateDelegateAttribute:          NewUpdateDelegateAttributeUseCase(updateRepos, updateServices),
		DeleteDelegateAttribute:          NewDeleteDelegateAttributeUseCase(deleteRepos, deleteServices),
		ListDelegateAttributes:           NewListDelegateAttributesUseCase(listRepos, listServices),
		GetDelegateAttributeListPageData: NewGetDelegateAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetDelegateAttributeItemPageData: NewGetDelegateAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of delegate attribute use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	delegateAttributeRepo delegateattributepb.DelegateAttributeDomainServiceServer,
	delegateRepo delegatepb.DelegateDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := DelegateAttributeRepositories{
		DelegateAttribute: delegateAttributeRepo,
		Delegate:          delegateRepo,
		Attribute:         attributeRepo,
	}

	services := DelegateAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
