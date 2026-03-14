// Package registry re-exports internal registry types and functions for use by contrib sub-modules.
package registry

import (
	internal "github.com/erniealice/espyna-golang/internal/infrastructure/registry"
)

// =============================================================================
// Generic Factory Registry
// =============================================================================

type FactoryRegistry[T any, C any] = internal.FactoryRegistry[T, C]

// NewFactoryRegistry wraps the generic constructor (generic funcs cannot be
// assigned to package-level vars in Go).
func NewFactoryRegistry[T any, C any](providerType string) *FactoryRegistry[T, C] {
	return internal.NewFactoryRegistry[T, C](providerType)
}

// =============================================================================
// Database Provider Registry
// =============================================================================

type DatabaseConfigTransformer = internal.DatabaseConfigTransformer

var (
	RegisterDatabaseProvider        = internal.RegisterDatabaseProvider
	RegisterDatabaseProviderFactory = internal.RegisterDatabaseProviderFactory
	GetDatabaseProviderFactory      = internal.GetDatabaseProviderFactory

	RegisterDatabaseConfigTransformer = internal.RegisterDatabaseConfigTransformer
	GetDatabaseConfigTransformer      = internal.GetDatabaseConfigTransformer
	TransformDatabaseConfig           = internal.TransformDatabaseConfig

	RegisterDatabaseBuildFromEnv      = internal.RegisterDatabaseBuildFromEnv
	GetDatabaseBuildFromEnv           = internal.GetDatabaseBuildFromEnv
	BuildDatabaseProviderFromEnv      = internal.BuildDatabaseProviderFromEnv
	ListAvailableDatabaseBuildFromEnv = internal.ListAvailableDatabaseBuildFromEnv

	ListAvailableDatabaseProviderFactories = internal.ListAvailableDatabaseProviderFactories
)

// =============================================================================
// Database Table Config Registry
// =============================================================================

type DatabaseTableConfig = internal.DatabaseTableConfig
type TableConfigBuilder = internal.TableConfigBuilder

var (
	RegisterDatabaseTableConfigBuilder = internal.RegisterDatabaseTableConfigBuilder
	GetDatabaseTableConfigBuilder      = internal.GetDatabaseTableConfigBuilder
	BuildDatabaseTableConfig           = internal.BuildDatabaseTableConfig
	DefaultDatabaseTableConfig         = internal.DefaultDatabaseTableConfig
)

// =============================================================================
// Repository Factory Registry
// =============================================================================

type RepositoryFactory = internal.RepositoryFactory

var (
	RegisterRepositoryFactory  = internal.RegisterRepositoryFactory
	GetRepositoryFactory       = internal.GetRepositoryFactory
	CreateRepository           = internal.CreateRepository
	ListRepositoryFactories    = internal.ListRepositoryFactories
	ListAllRepositoryFactories = internal.ListAllRepositoryFactories
)

// =============================================================================
// Database Operations Factory Registry
// =============================================================================

type DatabaseOperationsFactory = internal.DatabaseOperationsFactory

var (
	RegisterDatabaseOperationsFactory = internal.RegisterDatabaseOperationsFactory
	GetDatabaseOperationsFactory      = internal.GetDatabaseOperationsFactory
	CreateDatabaseOperations          = internal.CreateDatabaseOperations
	ListDatabaseOperationsFactories   = internal.ListDatabaseOperationsFactories
)

// =============================================================================
// Storage Provider Registry
// =============================================================================

type StorageConfigTransformer = internal.StorageConfigTransformer

var (
	RegisterStorageProvider        = internal.RegisterStorageProvider
	RegisterStorageProviderFactory = internal.RegisterStorageProviderFactory
	GetStorageProviderFactory      = internal.GetStorageProviderFactory

	RegisterStorageConfigTransformer = internal.RegisterStorageConfigTransformer
	GetStorageConfigTransformer      = internal.GetStorageConfigTransformer
	TransformStorageConfig           = internal.TransformStorageConfig

	RegisterStorageBuildFromEnv      = internal.RegisterStorageBuildFromEnv
	GetStorageBuildFromEnv           = internal.GetStorageBuildFromEnv
	BuildStorageProviderFromEnv      = internal.BuildStorageProviderFromEnv
	ListAvailableStorageBuildFromEnv = internal.ListAvailableStorageBuildFromEnv

	ListAvailableStorageProviderFactories = internal.ListAvailableStorageProviderFactories
)

// =============================================================================
// Auth Provider Registry
// =============================================================================

type AuthConfigTransformer = internal.AuthConfigTransformer

var (
	RegisterAuthProvider        = internal.RegisterAuthProvider
	RegisterAuthProviderFactory = internal.RegisterAuthProviderFactory
	GetAuthProviderFactory      = internal.GetAuthProviderFactory

	RegisterAuthConfigTransformer = internal.RegisterAuthConfigTransformer
	GetAuthConfigTransformer      = internal.GetAuthConfigTransformer
	TransformAuthConfig           = internal.TransformAuthConfig

	RegisterAuthBuildFromEnv      = internal.RegisterAuthBuildFromEnv
	GetAuthBuildFromEnv           = internal.GetAuthBuildFromEnv
	BuildAuthProviderFromEnv      = internal.BuildAuthProviderFromEnv
	ListAvailableAuthBuildFromEnv = internal.ListAvailableAuthBuildFromEnv

	ListAvailableAuthProviderFactories = internal.ListAvailableAuthProviderFactories
)

// =============================================================================
// Email Provider Registry
// =============================================================================

type EmailConfigTransformer = internal.EmailConfigTransformer

var (
	RegisterEmailProvider        = internal.RegisterEmailProvider
	RegisterEmailProviderFactory = internal.RegisterEmailProviderFactory
	GetEmailProviderFactory      = internal.GetEmailProviderFactory

	RegisterEmailConfigTransformer = internal.RegisterEmailConfigTransformer
	GetEmailConfigTransformer      = internal.GetEmailConfigTransformer
	TransformEmailConfig           = internal.TransformEmailConfig

	RegisterEmailBuildFromEnv      = internal.RegisterEmailBuildFromEnv
	GetEmailBuildFromEnv           = internal.GetEmailBuildFromEnv
	BuildEmailProviderFromEnv      = internal.BuildEmailProviderFromEnv
	ListAvailableEmailBuildFromEnv = internal.ListAvailableEmailBuildFromEnv

	ListAvailableEmailProviderFactories = internal.ListAvailableEmailProviderFactories
)

// =============================================================================
// Tabular Provider Registry
// =============================================================================

type TabularConfigTransformer = internal.TabularConfigTransformer
type TabularBuildFromEnv = internal.TabularBuildFromEnv

var (
	RegisterTabularProvider        = internal.RegisterTabularProvider
	RegisterTabularProviderFactory = internal.RegisterTabularProviderFactory
	GetTabularProviderFactory      = internal.GetTabularProviderFactory

	RegisterTabularConfigTransformer = internal.RegisterTabularConfigTransformer
	GetTabularConfigTransformer      = internal.GetTabularConfigTransformer
	TransformTabularConfig           = internal.TransformTabularConfig

	RegisterTabularBuildFromEnv      = internal.RegisterTabularBuildFromEnv
	GetTabularBuildFromEnv           = internal.GetTabularBuildFromEnv
	BuildTabularProviderFromEnv      = internal.BuildTabularProviderFromEnv
	ListAvailableTabularBuildFromEnv = internal.ListAvailableTabularBuildFromEnv

	ListAvailableTabularProviderFactories = internal.ListAvailableTabularProviderFactories
)

// =============================================================================
// Server Provider Registry
// =============================================================================

var (
	RegisterServerProvider        = internal.RegisterServerProvider
	RegisterServerProviderFactory = internal.RegisterServerProviderFactory
	GetServerProviderFactory      = internal.GetServerProviderFactory

	RegisterServerBuildFromEnv      = internal.RegisterServerBuildFromEnv
	GetServerBuildFromEnv           = internal.GetServerBuildFromEnv
	BuildServerProviderFromEnv      = internal.BuildServerProviderFromEnv
	ListAvailableServerBuildFromEnv = internal.ListAvailableServerBuildFromEnv

	ListAvailableServerProviderFactories = internal.ListAvailableServerProviderFactories
)

// =============================================================================
// Ledger Reporting Factory Registry
// =============================================================================

var (
	RegisterLedgerReportingFactory = internal.RegisterLedgerReportingFactory
	GetLedgerReportingFactory      = internal.GetLedgerReportingFactory
)
