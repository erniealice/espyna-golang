package domain

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/document"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/document/attachment"
	"github.com/erniealice/espyna-golang/internal/application/usecases/domain/document/template"
	repodomain "github.com/erniealice/espyna-golang/internal/composition/providers/domain"
)

// InitializeDocument wires the document domain use cases (attachment +
// template sub-aggregates) from the LedgerRepositories (which historically
// hosts the document protos at LedgerRepositories.{Attachment,
// DocumentTemplate}).
//
// Returns a non-nil *document.UseCases even when individual repos are
// unavailable — each sub-aggregate field may be nil for graceful
// degradation on non-postgres builds.
func InitializeDocument(
	ledgerRepos *repodomain.LedgerRepositories,
	authSvc ports.Authorizer,
	txSvc ports.Transactor,
	i18nSvc ports.Translator,
	idSvc ports.IDGenerator,
	actionGate *actiongate.ActionGatekeeper,
) (*document.UseCases, error) {
	uc := &document.UseCases{}
	if ledgerRepos == nil {
		return uc, nil
	}

	if ledgerRepos.Attachment != nil {
		uc.Attachment = attachment.NewUseCases(
			attachment.AttachmentRepositories{Attachment: ledgerRepos.Attachment},
			attachment.AttachmentServices{
				Authorizer:       authSvc,
				Transactor:       txSvc,
				Translator:       i18nSvc,
				IDGenerator:      idSvc,
				ActionGatekeeper: actionGate,
			},
		)
	}
	if ledgerRepos.DocumentTemplate != nil {
		uc.Template = template.NewUseCases(
			template.DocumentTemplateRepositories{DocumentTemplate: ledgerRepos.DocumentTemplate},
			template.DocumentTemplateServices{
				Authorizer:       authSvc,
				Transactor:       txSvc,
				Translator:       i18nSvc,
				IDGenerator:      idSvc,
				ActionGatekeeper: actionGate,
			},
		)
	}
	return uc, nil
}
