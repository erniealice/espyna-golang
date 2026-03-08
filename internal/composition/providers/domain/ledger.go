package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"

	// Protobuf domain services - Ledger domain
	documenttemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/document_template"
)

// LedgerRepositories contains all ledger domain repositories
type LedgerRepositories struct {
	DocumentTemplate documenttemplatepb.DocumentTemplateDomainServiceServer
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

	documentTemplateRepo, err := repoCreator.CreateRepository("document_template", conn, dbTableConfig.DocumentTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create document template repository: %w", err)
	}

	return &LedgerRepositories{
		DocumentTemplate: documentTemplateRepo.(documenttemplatepb.DocumentTemplateDomainServiceServer),
	}, nil
}
