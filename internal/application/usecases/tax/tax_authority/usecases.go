package tax_authority

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	taxauthoritypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/tax/tax_authority"
)

const entityTaxAuthority = "tax_authority"

// TaxAuthorityRepositories groups all repository dependencies for tax_authority use cases.
type TaxAuthorityRepositories struct {
	TaxAuthority taxauthoritypb.TaxAuthorityDomainServiceServer
}

// TaxAuthorityServices groups all business service dependencies.
type TaxAuthorityServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

// UseCases contains all tax_authority use cases.
type UseCases struct {
	ReadTaxAuthority  *ReadTaxAuthorityUseCase
	ListTaxAuthorities *ListTaxAuthoritiesUseCase
}

// NewUseCases creates a new collection of tax_authority use cases.
func NewUseCases(repositories TaxAuthorityRepositories, services TaxAuthorityServices) *UseCases {
	return &UseCases{
		ReadTaxAuthority: NewReadTaxAuthorityUseCase(
			ReadTaxAuthorityRepositories{TaxAuthority: repositories.TaxAuthority},
			ReadTaxAuthorityServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListTaxAuthorities: NewListTaxAuthoritiesUseCase(
			ListTaxAuthoritiesRepositories{TaxAuthority: repositories.TaxAuthority},
			ListTaxAuthoritiesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
