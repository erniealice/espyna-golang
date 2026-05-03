package domain

import (
	"fmt"

	"github.com/erniealice/espyna-golang/internal/composition/contracts"
	"github.com/erniealice/espyna-golang/internal/infrastructure/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"

	assetpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset"
	assetcategorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/asset/asset_category"
)

// AssetRepositories contains all asset domain repositories.
type AssetRepositories struct {
	Asset         assetpb.AssetDomainServiceServer
	AssetCategory assetcategorypb.AssetCategoryDomainServiceServer
}

// NewAssetRepositories creates and returns a new set of AssetRepositories.
// The function will return an error at runtime until a postgres adapter is
// registered for entityid.Asset (Phase 4 of the asset typed-stack buildout).
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

	return &AssetRepositories{
		Asset:         assetRepo.(assetpb.AssetDomainServiceServer),
		AssetCategory: assetCategoryRepo.(assetcategorypb.AssetCategoryDomainServiceServer),
	}, nil
}
