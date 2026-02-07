package registry

import (
	"leapfor.xyz/espyna/internal/application/ports"
	storagepb "leapfor.xyz/esqyma/golang/v1/infrastructure/storage"
)

// =============================================================================
// Storage Factory Registry Instance
// =============================================================================

var storageRegistry = NewFactoryRegistry[ports.StorageProvider, *storagepb.StorageProviderConfig]("storage")

// =============================================================================
// Storage Provider Functions
// =============================================================================

func RegisterStorageProviderFactory(name string, factory func() ports.StorageProvider) {
	storageRegistry.RegisterFactory(name, factory)
}

func GetStorageProviderFactory(name string) (func() ports.StorageProvider, bool) {
	return storageRegistry.GetFactory(name)
}

func ListAvailableStorageProviderFactories() []string {
	return storageRegistry.ListFactories()
}

type StorageConfigTransformer func(rawConfig map[string]any) (*storagepb.StorageProviderConfig, error)

func RegisterStorageConfigTransformer(name string, transformer StorageConfigTransformer) {
	storageRegistry.RegisterConfigTransformer(name, transformer)
}

func GetStorageConfigTransformer(name string) (StorageConfigTransformer, bool) {
	return storageRegistry.GetConfigTransformer(name)
}

func TransformStorageConfig(name string, rawConfig map[string]any) (*storagepb.StorageProviderConfig, error) {
	return storageRegistry.TransformConfig(name, rawConfig)
}

func RegisterStorageBuildFromEnv(name string, builder func() (ports.StorageProvider, error)) {
	storageRegistry.RegisterBuildFromEnv(name, builder)
}

func GetStorageBuildFromEnv(name string) (func() (ports.StorageProvider, error), bool) {
	return storageRegistry.GetBuildFromEnv(name)
}

func BuildStorageProviderFromEnv(name string) (ports.StorageProvider, error) {
	return storageRegistry.BuildFromEnv(name)
}

func ListAvailableStorageBuildFromEnv() []string {
	return storageRegistry.ListBuildFromEnv()
}

func RegisterStorageProvider(name string, factory func() ports.StorageProvider, transformer StorageConfigTransformer) {
	RegisterStorageProviderFactory(name, factory)
	if transformer != nil {
		RegisterStorageConfigTransformer(name, transformer)
	}
}
