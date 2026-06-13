package expenditureattribute

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/expenditure/expenditure_attribute"
)

// ExpenditureAttributeRepositories groups all repository dependencies
type ExpenditureAttributeRepositories struct {
	ExpenditureAttribute pb.ExpenditureAttributeDomainServiceServer
}

// ExpenditureAttributeServices groups all business service dependencies
type ExpenditureAttributeServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all expenditure attribute use cases
type UseCases struct {
	CreateExpenditureAttribute *CreateExpenditureAttributeUseCase
	ReadExpenditureAttribute   *ReadExpenditureAttributeUseCase
	UpdateExpenditureAttribute *UpdateExpenditureAttributeUseCase
	DeleteExpenditureAttribute *DeleteExpenditureAttributeUseCase
	ListExpenditureAttributes  *ListExpenditureAttributesUseCase
}

// NewUseCases creates a new collection of expenditure attribute use cases
func NewUseCases(
	repositories ExpenditureAttributeRepositories,
	services ExpenditureAttributeServices,
) *UseCases {
	createRepos := CreateExpenditureAttributeRepositories(repositories)
	createServices := CreateExpenditureAttributeServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	readRepos := ReadExpenditureAttributeRepositories(repositories)
	readServices := ReadExpenditureAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	updateRepos := UpdateExpenditureAttributeRepositories(repositories)
	updateServices := UpdateExpenditureAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	deleteRepos := DeleteExpenditureAttributeRepositories(repositories)
	deleteServices := DeleteExpenditureAttributeServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRepos := ListExpenditureAttributesRepositories(repositories)
	listServices := ListExpenditureAttributesServices{
		Authorizer: services.Authorizer,
		Transactor: services.Transactor,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		CreateExpenditureAttribute: NewCreateExpenditureAttributeUseCase(createRepos, createServices),
		ReadExpenditureAttribute:   NewReadExpenditureAttributeUseCase(readRepos, readServices),
		UpdateExpenditureAttribute: NewUpdateExpenditureAttributeUseCase(updateRepos, updateServices),
		DeleteExpenditureAttribute: NewDeleteExpenditureAttributeUseCase(deleteRepos, deleteServices),
		ListExpenditureAttributes:  NewListExpenditureAttributesUseCase(listRepos, listServices),
	}
}
