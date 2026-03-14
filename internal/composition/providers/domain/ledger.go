package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	// Protobuf domain services - Ledger domain
	attachmentpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/document/attachment"
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/document_template"
)

// LedgerRepositories contains all ledger domain repositories
type LedgerRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
	Attachment       attachmentpb.AttachmentDomainServiceServer
}

// NewLedgerRepositories creates and returns a new set of LedgerRepositories
func NewLedgerRepositories(dbProvider contracts.Provider, dbTableConfig *registry.DatabaseTableConfig) (*LedgerRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	documentTemplateRepo, err := repoCreator.CreateRepository(entityid.DocumentTemplate, conn, dbTableConfig.DocumentTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create document template repository: %w", err)
	}

	attachmentRepo, err := repoCreator.CreateRepository(entityid.Attachment, conn, dbTableConfig.Attachment)
	if err != nil {
		return nil, fmt.Errorf("failed to create attachment repository: %w", err)
	}

	return &LedgerRepositories{
		DocumentTemplate: documentTemplateRepo.(documenttemplatepb.DocumentTemplateDomainServiceServer),
		Attachment:       attachmentRepo.(attachmentpb.AttachmentDomainServiceServer),
	}, nil
}
