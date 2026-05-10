package withholding_certificate

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

const entityWithholdingCertificate = "withholding_certificate"

// WithholdingCertificateRepositories groups all repository dependencies for withholding_certificate use cases.
type WithholdingCertificateRepositories struct {
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// WithholdingCertificateServices groups all business service dependencies.
type WithholdingCertificateServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all withholding_certificate use cases.
type UseCases struct {
	CreateWithholdingCertificate *CreateWithholdingCertificateUseCase
	ReadWithholdingCertificate   *ReadWithholdingCertificateUseCase
	UpdateWithholdingCertificate *UpdateWithholdingCertificateUseCase
	DeleteWithholdingCertificate *DeleteWithholdingCertificateUseCase
	ListWithholdingCertificates  *ListWithholdingCertificatesUseCase
}

// NewUseCases creates a new collection of withholding_certificate use cases.
func NewUseCases(repositories WithholdingCertificateRepositories, services WithholdingCertificateServices) *UseCases {
	return &UseCases{
		CreateWithholdingCertificate: NewCreateWithholdingCertificateUseCase(
			CreateWithholdingCertificateRepositories{WithholdingCertificate: repositories.WithholdingCertificate},
			CreateWithholdingCertificateServices{
				AuthorizationService: services.AuthorizationService,
				TransactionService:   services.TransactionService,
				TranslationService:   services.TranslationService,
				IDService:            services.IDService,
			},
		),
		ReadWithholdingCertificate: NewReadWithholdingCertificateUseCase(
			ReadWithholdingCertificateRepositories{WithholdingCertificate: repositories.WithholdingCertificate},
			ReadWithholdingCertificateServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		UpdateWithholdingCertificate: NewUpdateWithholdingCertificateUseCase(
			UpdateWithholdingCertificateRepositories{WithholdingCertificate: repositories.WithholdingCertificate},
			UpdateWithholdingCertificateServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		DeleteWithholdingCertificate: NewDeleteWithholdingCertificateUseCase(
			DeleteWithholdingCertificateRepositories{WithholdingCertificate: repositories.WithholdingCertificate},
			DeleteWithholdingCertificateServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
		ListWithholdingCertificates: NewListWithholdingCertificatesUseCase(
			ListWithholdingCertificatesRepositories{WithholdingCertificate: repositories.WithholdingCertificate},
			ListWithholdingCertificatesServices{
				AuthorizationService: services.AuthorizationService,
				TranslationService:   services.TranslationService,
			},
		),
	}
}
