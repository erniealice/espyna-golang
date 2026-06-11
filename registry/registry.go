// Package registry is the public API surface for espyna's self-registration system.
// It re-exports types and functions from internal/infrastructure/registry so that
// consumers (apps, contrib adapters) import this package and never the internal one.
//
// Major registry subsystems exposed here:
//   - Database Provider: factory, config transformer, BuildFromEnv for each DB backend
//   - Table Config: map-based table/collection name resolution with env-var overrides
//   - Repository Factory: "provider:entity" keyed factories, self-registered via init()
//   - Database Operations Factory: provider-keyed raw CRUD operation factories
//   - Storage: provider factory, config transformer, BuildFromEnv
//   - Auth: provider factory, config transformer, BuildFromEnv
//   - Email: provider factory, config transformer, BuildFromEnv
//   - Tabular: provider factory, config transformer, BuildFromEnv
//   - Server: provider factory, BuildFromEnv
//   - Ledger Reporting: factory for ledger report generators
//
// Note: entityid constants live in registry/entityid/ (separate package, no dependency on this one).
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

type TableConfigBuilder = internal.TableConfigBuilder

var (
	RegisterDatabaseTableConfigBuilder = internal.RegisterDatabaseTableConfigBuilder
	GetDatabaseTableConfigBuilder      = internal.GetDatabaseTableConfigBuilder
	BuildDatabaseTableConfig           = internal.BuildDatabaseTableConfig
)

// =============================================================================
// Database Schema Validator Registry (Plan 2 — reflectionless CRUD boot-shot)
// =============================================================================

type SchemaValidator = internal.SchemaValidator

var (
	RegisterSchemaValidator = internal.RegisterSchemaValidator
	GetSchemaValidator      = internal.GetSchemaValidator
)

// =============================================================================
// Table Config (Map-Based)
// =============================================================================

type TableConfig = internal.TableConfig

var (
	NewTableConfig        = internal.NewTableConfig
	NewDefaultTableConfig = internal.NewDefaultTableConfig
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
// Scheduler Provider Registry
// =============================================================================
// (Integration provider. Re-exported here so contrib/ scheduler adapters —
// e.g. contrib/calendly — can self-register without importing internal/.
// Added 2026-06-10 per E5 contrib relocation: the public registry previously
// re-exported email + tabular but omitted scheduler/payment/fulfillment.
// registry/integration/ namespacing tracked as an N9 follow-up.)

type SchedulerConfigTransformer = internal.SchedulerConfigTransformer

var (
	RegisterSchedulerProvider        = internal.RegisterSchedulerProvider
	RegisterSchedulerProviderFactory = internal.RegisterSchedulerProviderFactory
	GetSchedulerProviderFactory      = internal.GetSchedulerProviderFactory

	RegisterSchedulerConfigTransformer = internal.RegisterSchedulerConfigTransformer
	GetSchedulerConfigTransformer      = internal.GetSchedulerConfigTransformer
	TransformSchedulerConfig           = internal.TransformSchedulerConfig

	RegisterSchedulerBuildFromEnv      = internal.RegisterSchedulerBuildFromEnv
	GetSchedulerBuildFromEnv           = internal.GetSchedulerBuildFromEnv
	BuildSchedulerProviderFromEnv      = internal.BuildSchedulerProviderFromEnv
	ListAvailableSchedulerBuildFromEnv = internal.ListAvailableSchedulerBuildFromEnv

	ListAvailableSchedulerProviderFactories = internal.ListAvailableSchedulerProviderFactories
)

// =============================================================================
// Payment Provider Registry
// =============================================================================
// (Integration provider. Re-exported here so contrib/ payment adapters —
// e.g. contrib/maya, contrib/asiapay, contrib/paypal — can self-register
// without importing internal/. Added 2026-06-11 per E5 P1 contrib relocation.)

type PaymentConfigTransformer = internal.PaymentConfigTransformer

var (
	RegisterPaymentProvider        = internal.RegisterPaymentProvider
	RegisterPaymentProviderFactory = internal.RegisterPaymentProviderFactory
	GetPaymentProviderFactory      = internal.GetPaymentProviderFactory

	RegisterPaymentConfigTransformer = internal.RegisterPaymentConfigTransformer
	GetPaymentConfigTransformer      = internal.GetPaymentConfigTransformer
	TransformPaymentConfig           = internal.TransformPaymentConfig

	RegisterPaymentBuildFromEnv      = internal.RegisterPaymentBuildFromEnv
	GetPaymentBuildFromEnv           = internal.GetPaymentBuildFromEnv
	BuildPaymentProviderFromEnv      = internal.BuildPaymentProviderFromEnv
	ListAvailablePaymentBuildFromEnv = internal.ListAvailablePaymentBuildFromEnv

	ListAvailablePaymentProviderFactories = internal.ListAvailablePaymentProviderFactories
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

// =============================================================================
// Audit Service Factory Registry
// =============================================================================

var (
	RegisterAuditServiceFactory           = internal.RegisterAuditServiceFactory
	GetAuditServiceFactory                = internal.GetAuditServiceFactory
	RegisterAuditEnabledOperationsFactory = internal.RegisterAuditEnabledOperationsFactory
	GetAuditEnabledOperationsFactory      = internal.GetAuditEnabledOperationsFactory
)
