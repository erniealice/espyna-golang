package depreciation_run

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"

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
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	candidatesRepos := ListDepreciationCandidatesRepositories{
		Asset:                repositories.Asset,
		AssetCategory:        repositories.AssetCategory,
		DepreciationSchedule: repositories.DepreciationSchedule,
	}
	candidatesServices := ListDepreciationCandidatesServices{
		AuthorizationService: services.AuthorizationService,
		TranslationService:   services.TranslationService,
	}

	listRunsRepos := ListDepreciationRunsRepositories{
		DepreciationRun: repositories.DepreciationRun,
	}
	listRunsServices := ListDepreciationRunsServices{
		AuthorizationService: services.AuthorizationService,
		TranslationService:   services.TranslationService,
	}

	readRunRepos := ReadDepreciationRunRepositories{
		DepreciationRun: repositories.DepreciationRun,
	}
	readRunServices := ReadDepreciationRunServices{
		AuthorizationService: services.AuthorizationService,
		TranslationService:   services.TranslationService,
	}

	listEntriesRepos := ListDepreciationRunEntriesRepositories{
		DepreciationSchedule: repositories.DepreciationSchedule,
	}
	listEntriesServices := ListDepreciationRunEntriesServices{
		AuthorizationService: services.AuthorizationService,
		TranslationService:   services.TranslationService,
	}

	return &UseCases{
		GenerateDepreciationRun:    NewGenerateDepreciationRunUseCase(generateRepos, generateServices),
		ListDepreciationCandidates: NewListDepreciationCandidatesUseCase(candidatesRepos, candidatesServices),
		ListDepreciationRuns:       NewListDepreciationRunsUseCase(listRunsRepos, listRunsServices),
		ReadDepreciationRun:        NewReadDepreciationRunUseCase(readRunRepos, readRunServices),
		ListDepreciationRunEntries: NewListDepreciationRunEntriesUseCase(listEntriesRepos, listEntriesServices),
	}
}
