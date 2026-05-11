package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
	revaluation_pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_revaluation"
	assettxpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_transaction"
	depschpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation"
	deprunpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/depreciation_run"
)

// AssetRepositories contains all asset domain repositories.
type AssetRepositories struct {
	Asset                assetpb.AssetDomainServiceServer
	AssetCategory        assetcategorypb.AssetCategoryDomainServiceServer
	AssetTransaction     assettxpb.AssetTransactionDomainServiceServer
	DepreciationSchedule depschpb.DepreciationDomainServiceServer
	DepreciationRun      deprunpb.DepreciationRunDomainServiceServer
	AssetRevaluation     revaluation_pb.AssetRevaluationDomainServiceServer
}

// NewAssetRepositories creates and returns a new set of AssetRepositories.
func NewAssetRepositories(dbProvider contracts.Provider, tableConfig *registry.TableConfig) (*AssetRepositories, error) {
	if dbProvider == nil {
		return nil, fmt.Errorf("database provider not initialized")
	}

	repoCreator, ok := dbProvider.(contracts.RepositoryProvider)
	if !ok {
		return nil, fmt.Errorf("database provider doesn't implement contracts.RepositoryProvider interface")
	}

	conn := repoCreator.GetConnection()

	assetRepo, err := repoCreator.CreateRepository(entityid.Asset, conn, tableConfig.TableName(entityid.Asset))
	if err != nil {
		return nil, fmt.Errorf("failed to create asset repository: %w", err)
	}

	assetCategoryRepo, err := repoCreator.CreateRepository(entityid.AssetCategory, conn, tableConfig.TableName(entityid.AssetCategory))
	if err != nil {
		return nil, fmt.Errorf("failed to create asset_category repository: %w", err)
	}

	// New repos added for depreciation run + revaluation. Fall back gracefully if not registered.
	var assetTxRepo assettxpb.AssetTransactionDomainServiceServer
	if r, e := repoCreator.CreateRepository(entityid.AssetTransaction, conn, tableConfig.TableName(entityid.AssetTransaction)); e == nil && r != nil {
		if typed, ok := r.(assettxpb.AssetTransactionDomainServiceServer); ok {
			assetTxRepo = typed
		}
	}

	var depSchRepo depschpb.DepreciationDomainServiceServer
	if r, e := repoCreator.CreateRepository(entityid.DepreciationSchedule, conn, tableConfig.TableName(entityid.DepreciationSchedule)); e == nil && r != nil {
		if typed, ok := r.(depschpb.DepreciationDomainServiceServer); ok {
			depSchRepo = typed
		}
	}

	var depRunRepo deprunpb.DepreciationRunDomainServiceServer
	if r, e := repoCreator.CreateRepository(entityid.DepreciationRun, conn, tableConfig.TableName(entityid.DepreciationRun)); e == nil && r != nil {
		if typed, ok := r.(deprunpb.DepreciationRunDomainServiceServer); ok {
			depRunRepo = typed
		}
	}

	var assetRevRepo revaluation_pb.AssetRevaluationDomainServiceServer
	if r, e := repoCreator.CreateRepository(entityid.AssetRevaluation, conn, tableConfig.TableName(entityid.AssetRevaluation)); e == nil && r != nil {
		if typed, ok := r.(revaluation_pb.AssetRevaluationDomainServiceServer); ok {
			assetRevRepo = typed
		}
	}

	return &AssetRepositories{
		Asset:                assetRepo.(assetpb.AssetDomainServiceServer),
		AssetCategory:        assetCategoryRepo.(assetcategorypb.AssetCategoryDomainServiceServer),
		AssetTransaction:     assetTxRepo,
		DepreciationSchedule: depSchRepo,
		DepreciationRun:      depRunRepo,
		AssetRevaluation:     assetRevRepo,
	}, nil
}
