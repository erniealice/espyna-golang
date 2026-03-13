package ledger

import (
	// Document template use cases
	documentTemplateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/document_template"

	// Attachment use cases
	attachmentUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/document/attachment"

	// Application ports
	"github.com/erniealice/espyna-golang/internal/application/ports"

	// Reporting use cases
	grossprofit "github.com/erniealice/espyna-golang/internal/application/usecases/ledger/reporting/gross_profit"

	// Protobuf domain services for ledger repositories
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/document_template"
)

// LedgerRepositories groups all repository dependencies for ledger use cases.
type LedgerRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
	Attachment       attachmentpb.AttachmentDomainServiceServer
	ReportingService ports.LedgerReportingService
}

// LedgerUseCases contains all ledger-related use cases.
type LedgerUseCases struct {
	DocumentTemplate     *documentTemplateUseCases.UseCases
	Attachment           *attachmentUseCases.UseCases
	GetGrossProfitReport *grossprofit.GetGrossProfitReportUseCase
}

// NewUseCases creates all ledger use cases with proper constructor injection.
func NewUseCases(
	repos LedgerRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idService ports.IDService,
) *LedgerUseCases {
	documentTemplateUC := documentTemplateUseCases.NewUseCases(
		documentTemplateUseCases.DocumentTemplateRepositories{
			DocumentTemplate: repos.DocumentTemplate,
		},
		documentTemplateUseCases.DocumentTemplateServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	attachmentUC := attachmentUseCases.NewUseCases(
		attachmentUseCases.AttachmentRepositories{
			Attachment: repos.Attachment,
		},
		attachmentUseCases.AttachmentServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idService,
		},
	)

	return &LedgerUseCases{
		DocumentTemplate:     documentTemplateUC,
		Attachment:           attachmentUC,
		GetGrossProfitReport: grossprofit.NewGetGrossProfitReportUseCase(repos.ReportingService),
	}
}
