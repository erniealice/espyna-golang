package staff_attribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	staffpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff"
	staffattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/staff_attribute"
)

// UseCases contains all staff attribute-related use cases
type UseCases struct {
	CreateStaffAttribute          *CreateStaffAttributeUseCase
	ReadStaffAttribute            *ReadStaffAttributeUseCase
	UpdateStaffAttribute          *UpdateStaffAttributeUseCase
	DeleteStaffAttribute          *DeleteStaffAttributeUseCase
	ListStaffAttributes           *ListStaffAttributesUseCase
	GetStaffAttributeListPageData *GetStaffAttributeListPageDataUseCase
	GetStaffAttributeItemPageData *GetStaffAttributeItemPageDataUseCase
}

// StaffAttributeRepositories groups all repository dependencies for staff attribute use cases
type StaffAttributeRepositories struct {
	StaffAttribute staffattributepb.StaffAttributeDomainServiceServer // Primary entity repository
	Staff          staffpb.StaffDomainServiceServer                   // Entity reference validation
	Attribute      attributepb.AttributeDomainServiceServer           // Entity reference validation
}

// StaffAttributeServices groups all business service dependencies for staff attribute use cases
type StaffAttributeServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// NewUseCases creates a new collection of staff attribute use cases
func NewUseCases(
	repositories StaffAttributeRepositories,
	services StaffAttributeServices,
) *UseCases {
	// Build individual grouped parameters for each use case
	createRepos := CreateStaffAttributeRepositories(repositories)
	createServices := CreateStaffAttributeServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	readRepos := ReadStaffAttributeRepositories{
		StaffAttribute: repositories.StaffAttribute,
	}
	readServices := ReadStaffAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	updateRepos := UpdateStaffAttributeRepositories{
		StaffAttribute: repositories.StaffAttribute,
		Staff:          repositories.Staff,
		Attribute:      repositories.Attribute,
	}
	updateServices := UpdateStaffAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	deleteRepos := DeleteStaffAttributeRepositories{
		StaffAttribute: repositories.StaffAttribute,
	}
	deleteServices := DeleteStaffAttributeServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	listRepos := ListStaffAttributesRepositories{
		StaffAttribute: repositories.StaffAttribute,
	}
	listServices := ListStaffAttributesServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getListPageDataRepos := GetStaffAttributeListPageDataRepositories{
		StaffAttribute: repositories.StaffAttribute,
	}
	getListPageDataServices := GetStaffAttributeListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	getItemPageDataRepos := GetStaffAttributeItemPageDataRepositories{
		StaffAttribute: repositories.StaffAttribute,
	}
	getItemPageDataServices := GetStaffAttributeItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	return &UseCases{
		CreateStaffAttribute:          NewCreateStaffAttributeUseCase(createRepos, createServices),
		ReadStaffAttribute:            NewReadStaffAttributeUseCase(readRepos, readServices),
		UpdateStaffAttribute:          NewUpdateStaffAttributeUseCase(updateRepos, updateServices),
		DeleteStaffAttribute:          NewDeleteStaffAttributeUseCase(deleteRepos, deleteServices),
		ListStaffAttributes:           NewListStaffAttributesUseCase(listRepos, listServices),
		GetStaffAttributeListPageData: NewGetStaffAttributeListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetStaffAttributeItemPageData: NewGetStaffAttributeItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),
	}
}

// NewUseCasesUngrouped creates a new collection of staff attribute use cases with individual parameters
// Deprecated: Use NewUseCases with grouped parameters instead
func NewUseCasesUngrouped(
	staffAttributeRepo staffattributepb.StaffAttributeDomainServiceServer,
	staffRepo staffpb.StaffDomainServiceServer,
	attributeRepo attributepb.AttributeDomainServiceServer,
	authorizationService ports.AuthorizationService,
) *UseCases {
	// Build grouped parameters internally for backward compatibility
	repositories := StaffAttributeRepositories{
		StaffAttribute: staffAttributeRepo,
		Staff:          staffRepo,
		Attribute:      attributeRepo,
	}

	services := StaffAttributeServices{
		AuthorizationService: authorizationService,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewUseCases(repositories, services)
}
