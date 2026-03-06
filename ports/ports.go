// Package ports re-exports internal application port types for use by contrib sub-modules.
// Contrib packages (which are separate Go modules) cannot import internal/ directly,
// so this package provides stable public aliases.
package ports

import (
	internal "github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	"github.com/erniealice/espyna-golang/internal/application/ports/security"
)

// =============================================================================
// INFRASTRUCTURE PORTS
// =============================================================================

// Database types
type (
	DatabaseProvider         = internal.DatabaseProvider
	RepositoryProvider       = internal.RepositoryProvider
	RepositoryConfig         = internal.RepositoryConfig
	ConcreteRepositoryConfig = internal.ConcreteRepositoryConfig
	DatabaseConfigAdapter    = internal.DatabaseConfigAdapter
)

var NewDatabaseConfigAdapter = internal.NewDatabaseConfigAdapter

// Auth types
type (
	AuthProvider      = internal.AuthProvider
	AuthService       = internal.AuthService
	AuthConfigAdapter = internal.AuthConfigAdapter
)

var NewAuthConfigAdapter = internal.NewAuthConfigAdapter

// Auth error codes
const (
	ErrCodeMissingToken = infrastructure.ErrCodeMissingToken
	ErrCodeInvalidToken = infrastructure.ErrCodeInvalidToken
	ErrCodeExpiredToken = infrastructure.ErrCodeExpiredToken
	ErrCodeServiceDown  = infrastructure.ErrCodeServiceDown
	ErrCodeUnauthorized = infrastructure.ErrCodeUnauthorized
)

// Storage types
type (
	StorageProvider           = internal.StorageProvider
	StorageCapability         = internal.StorageCapability
	StorageCapabilityProvider = internal.StorageCapabilityProvider
	StorageError              = internal.StorageError
	StorageConfigAdapter      = internal.StorageConfigAdapter
)

var NewStorageConfigAdapter = internal.NewStorageConfigAdapter
var NewStorageError = internal.NewStorageError

// Storage capability constants
const (
	StorageCapabilityUpload          = infrastructure.StorageCapabilityUpload
	StorageCapabilityDownload        = infrastructure.StorageCapabilityDownload
	StorageCapabilityDelete          = infrastructure.StorageCapabilityDelete
	StorageCapabilityList            = infrastructure.StorageCapabilityList
	StorageCapabilityMetadata        = infrastructure.StorageCapabilityMetadata
	StorageCapabilityPresignedUrls   = infrastructure.StorageCapabilityPresignedUrls
	StorageCapabilityMultipartUpload = infrastructure.StorageCapabilityMultipartUpload
	StorageCapabilityVersioning      = infrastructure.StorageCapabilityVersioning
	StorageCapabilityEncryption      = infrastructure.StorageCapabilityEncryption
	StorageCapabilityAccessTiers     = infrastructure.StorageCapabilityAccessTiers
	StorageCapabilityObjectLock      = infrastructure.StorageCapabilityObjectLock
	StorageCapabilityLifecyclePolicy = infrastructure.StorageCapabilityLifecyclePolicy
	StorageCapabilityReplication     = infrastructure.StorageCapabilityReplication
	StorageCapabilityStreaming       = infrastructure.StorageCapabilityStreaming
)

// Storage error codes
const (
	StorageErrorCodeNotFound         = infrastructure.StorageErrorCodeNotFound
	StorageErrorCodeAlreadyExists    = infrastructure.StorageErrorCodeAlreadyExists
	StorageErrorCodeAccessDenied     = infrastructure.StorageErrorCodeAccessDenied
	StorageErrorCodeQuotaExceeded    = infrastructure.StorageErrorCodeQuotaExceeded
	StorageErrorCodeInvalidPath      = infrastructure.StorageErrorCodeInvalidPath
	StorageErrorCodeUploadFailed     = infrastructure.StorageErrorCodeUploadFailed
	StorageErrorCodeDownloadFailed   = infrastructure.StorageErrorCodeDownloadFailed
	StorageErrorCodeDeleteFailed     = infrastructure.StorageErrorCodeDeleteFailed
	StorageErrorCodeProviderError    = infrastructure.StorageErrorCodeProviderError
	StorageErrorCodeConfigError      = infrastructure.StorageErrorCodeConfigError
	StorageErrorCodeConnectionFailed = infrastructure.StorageErrorCodeConnectionFailed
)

// ID types
type IDService = internal.IDService
type NoOpIDService = internal.NoOpIDService

var NewNoOpIDService = internal.NewNoOpIDService

// Transaction types
type TransactionService = internal.TransactionService
type NoOpTransactionService = internal.NoOpTransactionService

var NewNoOpTransactionService = internal.NewNoOpTransactionService

// Migration types
type (
	MigrationService = internal.MigrationService
	MigrationStatus  = internal.MigrationStatus
	AppliedMigration = internal.AppliedMigration
	PendingMigration = internal.PendingMigration
	MigrationError   = internal.MigrationError
)

// Server types
type ServerProvider = internal.ServerProvider

var NewMigrationError = internal.NewMigrationError

// Migration error codes
const (
	MigrationErrCodeDirtyDatabase    = infrastructure.MigrationErrCodeDirtyDatabase
	MigrationErrCodeVersionNotFound  = infrastructure.MigrationErrCodeVersionNotFound
	MigrationErrCodeMigrationFailed  = infrastructure.MigrationErrCodeMigrationFailed
	MigrationErrCodeInvalidVersion   = infrastructure.MigrationErrCodeInvalidVersion
	MigrationErrCodeConnectionFailed = infrastructure.MigrationErrCodeConnectionFailed
)

// =============================================================================
// INTEGRATION PORTS
// =============================================================================

// Email types
type (
	EmailProvider   = internal.EmailProvider
	EmailMessage    = internal.EmailMessage
	EmailAttachment = internal.EmailAttachment
	InboxOptions    = internal.InboxOptions
)

var FromProtoMessage = internal.FromProtoMessage

// Payment types
type (
	PaymentProvider       = internal.PaymentProvider
	PaymentWebhookResult  = internal.PaymentWebhookResult
	CheckoutSessionParams = internal.CheckoutSessionParams
)

// Scheduler types
type (
	SchedulerProvider       = internal.SchedulerProvider
	ScheduleWebhookResult   = internal.ScheduleWebhookResult
	CreateScheduleParams    = internal.CreateScheduleParams
	CheckAvailabilityParams = internal.CheckAvailabilityParams
)

// Tabular types
type (
	TabularSourceProvider = internal.TabularSourceProvider
	TabularOptions        = internal.TabularOptions
	TabularRecord         = internal.TabularRecord
	TabularSelection      = internal.TabularSelection
)

// =============================================================================
// DOMAIN PORTS
// =============================================================================

// Workflow types
type (
	WorkflowEngineService = internal.WorkflowEngineService
	ActivityExecutor      = internal.ActivityExecutor
	ExecutorRegistry      = internal.ExecutorRegistry
)

// Translation types
type TranslationService = internal.TranslationService

var NewNoOpTranslationService = internal.NewNoOpTranslationService

// Ledger types
type LedgerReportingService = internal.LedgerReportingService

// =============================================================================
// SECURITY PORTS
// =============================================================================

// Authorization types
type (
	AuthorizationService   = internal.AuthorizationService
	AuthorizationProvider  = internal.AuthorizationProvider
	AuthorizationError     = internal.AuthorizationError
	AuthorizationErrorCode = internal.AuthorizationErrorCode
)

var NewNoOpAuthorizationService = internal.NewNoOpAuthorizationService
var NewAuthorizationError = internal.NewAuthorizationError

// Permission utility functions
var (
	EntityPermission    = security.EntityPermission
	WorkspacePermission = security.WorkspacePermission
)

// Authorization error constructors
var (
	ErrPermissionDenied      = security.ErrPermissionDenied
	ErrWorkspaceAccessDenied = security.ErrWorkspaceAccessDenied
	ErrUserNotAuthenticated  = security.ErrUserNotAuthenticated
	ErrInsufficientRole      = security.ErrInsufficientRole
	ErrProviderUnavailable   = security.ErrProviderUnavailable
	ErrServiceDisabled       = security.ErrServiceDisabled
	ErrAccessDenied          = security.ErrAccessDenied
)

// Authorization error codes
const (
	AuthErrCodePermissionDenied      = security.AuthErrCodePermissionDenied
	AuthErrCodeInsufficientRole      = security.AuthErrCodeInsufficientRole
	AuthErrCodeWorkspaceAccessDenied = security.AuthErrCodeWorkspaceAccessDenied
	AuthErrCodeUserNotFound          = security.AuthErrCodeUserNotFound
	AuthErrCodeUserNotAuthenticated  = security.AuthErrCodeUserNotAuthenticated
	AuthErrCodeInvalidUserID         = security.AuthErrCodeInvalidUserID
	AuthErrCodeWorkspaceNotFound     = security.AuthErrCodeWorkspaceNotFound
	AuthErrCodeInvalidWorkspaceID    = security.AuthErrCodeInvalidWorkspaceID
	AuthErrCodeInvalidPermission     = security.AuthErrCodeInvalidPermission
	AuthErrCodeRoleNotFound          = security.AuthErrCodeRoleNotFound
	AuthErrCodePermissionNotFound    = security.AuthErrCodePermissionNotFound
	AuthErrCodeProviderUnavailable   = security.AuthErrCodeProviderUnavailable
	AuthErrCodeProviderError         = security.AuthErrCodeProviderError
	AuthErrCodeConfigurationError    = security.AuthErrCodeConfigurationError
	AuthErrCodeServiceDisabled       = security.AuthErrCodeServiceDisabled
	AuthErrCodeInternalError         = security.AuthErrCodeInternalError
)

// Permission action constants
const (
	ActionCreate = security.ActionCreate
	ActionRead   = security.ActionRead
	ActionUpdate = security.ActionUpdate
	ActionDelete = security.ActionDelete
	ActionList   = security.ActionList
	ActionManage = security.ActionManage
)

// Entity constants
const (
	// Entity Domain
	EntityAdmin             = security.EntityAdmin
	EntityClient            = security.EntityClient
	EntityClientAttribute   = security.EntityClientAttribute
	EntityDelegate          = security.EntityDelegate
	EntityDelegateAttribute = security.EntityDelegateAttribute
	EntityDelegateClient    = security.EntityDelegateClient
	EntityGroup             = security.EntityGroup
	EntityGroupAttribute    = security.EntityGroupAttribute
	EntityLocation          = security.EntityLocation
	EntityLocationAttribute = security.EntityLocationAttribute
	EntityManager           = security.EntityManager
	EntityPermissions       = security.EntityPermissions
	EntityRole              = security.EntityRole
	EntityRolePermission    = security.EntityRolePermission
	EntityStaff             = security.EntityStaff
	EntityStaffAttribute    = security.EntityStaffAttribute
	EntityUser              = security.EntityUser
	EntityWorkspace         = security.EntityWorkspace
	EntityWorkspaceUser     = security.EntityWorkspaceUser
	EntityWorkspaceUserRole = security.EntityWorkspaceUserRole

	// Event Domain
	EntityEvent          = security.EntityEvent
	EntityEventAttribute = security.EntityEventAttribute
	EntityEventClient    = security.EntityEventClient
	EntityEventProduct   = security.EntityEventProduct

	// Framework Domain
	EntityFramework = security.EntityFramework
	EntityObjective = security.EntityObjective
	EntityTask      = security.EntityTask

	// Expenditure Domain
	EntityExpenditure          = security.EntityExpenditure
	EntityExpenditureAttribute = security.EntityExpenditureAttribute
	EntityExpenditureLineItem  = security.EntityExpenditureLineItem
	EntityExpenditureCategory  = security.EntityExpenditureCategory

	// Inventory Domain
	EntityInventoryItem          = security.EntityInventoryItem
	EntityInventorySerial        = security.EntityInventorySerial
	EntityInventoryTransaction   = security.EntityInventoryTransaction
	EntityInventoryAttribute     = security.EntityInventoryAttribute
	EntityInventoryDepreciation  = security.EntityInventoryDepreciation
	EntityInventorySerialHistory = security.EntityInventorySerialHistory

	// Product Domain
	EntityCollection           = security.EntityCollection
	EntityCollectionAttribute  = security.EntityCollectionAttribute
	EntityCollectionPlan       = security.EntityCollectionPlan
	EntityPriceList            = security.EntityPriceList
	EntityPriceProduct         = security.EntityPriceProduct
	EntityProduct              = security.EntityProduct
	EntityProductAttribute     = security.EntityProductAttribute
	EntityProductCollection    = security.EntityProductCollection
	EntityProductOption        = security.EntityProductOption
	EntityProductOptionValue   = security.EntityProductOptionValue
	EntityProductPlan          = security.EntityProductPlan
	EntityProductVariant       = security.EntityProductVariant
	EntityProductVariantImage  = security.EntityProductVariantImage
	EntityProductVariantOption = security.EntityProductVariantOption
	EntityResource             = security.EntityResource

	// Record Domain
	EntityRecord = security.EntityRecord

	// Subscription Domain
	EntityBalance               = security.EntityBalance
	EntityBalanceAttribute      = security.EntityBalanceAttribute
	EntityInvoice               = security.EntityInvoice
	EntityInvoiceAttribute      = security.EntityInvoiceAttribute
	EntityLicense               = security.EntityLicense
	EntityLicenseHistory        = security.EntityLicenseHistory
	EntityPlan                  = security.EntityPlan
	EntityPlanAttribute         = security.EntityPlanAttribute
	EntityPlanSettings          = security.EntityPlanSettings
	EntityPricePlan             = security.EntityPricePlan
	EntitySubscription          = security.EntitySubscription
	EntitySubscriptionAttribute = security.EntitySubscriptionAttribute

	// Asset Domain
	EntityAsset                = security.EntityAsset
	EntityAssetCategory        = security.EntityAssetCategory
	EntityAssetAttribute       = security.EntityAssetAttribute
	EntityAssetLocation        = security.EntityAssetLocation
	EntityDepreciationSchedule = security.EntityDepreciationSchedule
	EntityAssetTransaction     = security.EntityAssetTransaction
	EntityAssetDisposal        = security.EntityAssetDisposal
	EntityAssetRevaluation     = security.EntityAssetRevaluation
	EntityAssetMaintenance     = security.EntityAssetMaintenance
	EntityAssetComponent       = security.EntityAssetComponent
)

// Ensure sub-package imports are used (prevents "imported and not used" errors
// when only type aliases from internal are used, since internal re-exports from
// these packages via its own type aliases).
var (
	_ domain.TranslationService
	_ infrastructure.DatabaseProvider
	_ integration.EmailProvider
	_ security.AuthorizationService
)
