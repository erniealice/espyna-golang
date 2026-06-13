package depreciation_run

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	assettxpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_transaction"
	depschpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation"
	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
)

// DepreciationRunRepositories groups all repository dependencies for the
// depreciation-run sub-domain. Used by NewUseCases.
type DepreciationRunRepositories struct {
	Asset                assetpb.AssetDomainServiceServer
	AssetCategory        assetcategorypb.AssetCategoryDomainServiceServer
	AssetTransaction     assettxpb.AssetTransactionDomainServiceServer
	DepreciationSchedule depschpb.DepreciationDomainServiceServer
	DepreciationRun      deprunpb.DepreciationRunDomainServiceServer
}

// DepreciationRunServices groups all service dependencies.
type DepreciationRunServices struct {
	Authorizer  ports.Authorizer
	Transactor  ports.Transactor
	Translator  ports.Translator
	ActionGatekeeper *actiongate.ActionGatekeeper
	IDGenerator ports.IDGenerator
}

// UseCases contains all depreciation-run-related use cases.
type UseCases struct {
	GenerateDepreciationRun    *GenerateDepreciationRunUseCase
	ListDepreciationCandidates *ListDepreciationCandidatesUseCase
	ListDepreciationRuns       *ListDepreciationRunsUseCase
	ReadDepreciationRun        *ReadDepreciationRunUseCase
	ListDepreciationRunEntries *ListDepreciationRunEntriesUseCase
}

// NewUseCases creates a new collection of depreciation-run use cases.
func NewUseCases(
	repositories DepreciationRunRepositories,
	services DepreciationRunServices,
) *UseCases {
	generateRepos := GenerateDepreciationRunRepositories{
		Asset:                repositories.Asset,
		AssetCategory:        repositories.AssetCategory,
		AssetTransaction:     repositories.AssetTransaction,
		DepreciationSchedule: repositories.DepreciationSchedule,
		DepreciationRun:      repositories.DepreciationRun,
	}
	generateServices := GenerateDepreciationRunServices{
		Authorizer:  services.Authorizer,
		Transactor:  services.Transactor,
		Translator:  services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
		IDGenerator: services.IDGenerator,
	}

	candidatesRepos := ListDepreciationCandidatesRepositories{
		Asset:                repositories.Asset,
		AssetCategory:        repositories.AssetCategory,
		DepreciationSchedule: repositories.DepreciationSchedule,
	}
	candidatesServices := ListDepreciationCandidatesServices{
		Authorizer: services.Authorizer,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listRunsRepos := ListDepreciationRunsRepositories{
		DepreciationRun: repositories.DepreciationRun,
	}
	listRunsServices := ListDepreciationRunsServices{
		Authorizer: services.Authorizer,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	readRunRepos := ReadDepreciationRunRepositories{
		DepreciationRun: repositories.DepreciationRun,
	}
	readRunServices := ReadDepreciationRunServices{
		Authorizer: services.Authorizer,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	listEntriesRepos := ListDepreciationRunEntriesRepositories{
		DepreciationSchedule: repositories.DepreciationSchedule,
	}
	listEntriesServices := ListDepreciationRunEntriesServices{
		Authorizer: services.Authorizer,
		Translator: services.Translator,
		ActionGatekeeper: services.ActionGatekeeper,
	}

	return &UseCases{
		GenerateDepreciationRun:    NewGenerateDepreciationRunUseCase(generateRepos, generateServices),
		ListDepreciationCandidates: NewListDepreciationCandidatesUseCase(candidatesRepos, candidatesServices),
		ListDepreciationRuns:       NewListDepreciationRunsUseCase(listRunsRepos, listRunsServices),
		ReadDepreciationRun:        NewReadDepreciationRunUseCase(readRunRepos, readRunServices),
		ListDepreciationRunEntries: NewListDepreciationRunEntriesUseCase(listEntriesRepos, listEntriesServices),
	}
}
