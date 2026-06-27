package withholding_certificate

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	withholdingcertificatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/treasury/withholding_certificate"
)

const entityWithholdingCertificate = "withholding_certificate"

// WithholdingCertificateRepositories groups all repository dependencies for withholding_certificate use cases.
type WithholdingCertificateRepositories struct {
	WithholdingCertificate withholdingcertificatepb.WithholdingCertificateDomainServiceServer
}

// WithholdingCertificateServices groups all business service dependencies.
type WithholdingCertificateServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
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
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer:  services.Authorizer,
				Transactor:  services.Transactor,
				Translator:  services.Translator,
				IDGenerator: services.IDGenerator,
			},
		),
		ReadWithholdingCertificate: NewReadWithholdingCertificateUseCase(
			ReadWithholdingCertificateRepositories{WithholdingCertificate: repositories.WithholdingCertificate},
			ReadWithholdingCertificateServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		UpdateWithholdingCertificate: NewUpdateWithholdingCertificateUseCase(
			UpdateWithholdingCertificateRepositories{WithholdingCertificate: repositories.WithholdingCertificate},
			UpdateWithholdingCertificateServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		DeleteWithholdingCertificate: NewDeleteWithholdingCertificateUseCase(
			DeleteWithholdingCertificateRepositories{WithholdingCertificate: repositories.WithholdingCertificate},
			DeleteWithholdingCertificateServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
		ListWithholdingCertificates: NewListWithholdingCertificatesUseCase(
			ListWithholdingCertificatesRepositories{WithholdingCertificate: repositories.WithholdingCertificate},
			ListWithholdingCertificatesServices{
				ActionGatekeeper: services.ActionGatekeeper,
				Authorizer: services.Authorizer,
				Translator: services.Translator,
			},
		),
	}
}
