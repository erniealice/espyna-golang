package ports

// This file provides backward compatibility by re-exporting types from sub-packages.
// Existing code that imports "leapfor.xyz/espyna/internal/application/ports" continues to work.
// New code can import specific sub-packages for cleaner dependencies:
//   - leapfor.xyz/espyna/internal/application/ports/infrastructure
//   - leapfor.xyz/espyna/internal/application/ports/integration
//   - leapfor.xyz/espyna/internal/application/ports/domain
//   - leapfor.xyz/espyna/internal/application/ports/security

import (
	"leapfor.xyz/espyna/internal/application/ports/domain"
	"leapfor.xyz/espyna/internal/application/ports/infrastructure"
	"leapfor.xyz/espyna/internal/application/ports/integration"
	"leapfor.xyz/espyna/internal/application/ports/security"
)

// =============================================================================
// INFRASTRUCTURE PORTS (Database, Auth, Storage, ID, Transaction, Migration)
// =============================================================================

// Database types
type (
	DatabaseProvider         = infrastructure.DatabaseProvider
	RepositoryProvider       = infrastructure.RepositoryProvider
	RepositoryConfig         = infrastructure.RepositoryConfig
	ConcreteRepositoryConfig = infrastructure.ConcreteRepositoryConfig
	DatabaseConfigAdapter    = infrastructure.DatabaseConfigAdapter
)

// NewDatabaseConfigAdapter creates a new database config adapter
var NewDatabaseConfigAdapter = infrastructure.NewDatabaseConfigAdapter

// Auth types
type (
	AuthProvider      = infrastructure.AuthProvider
	AuthService       = infrastructure.AuthService
	AuthConfigAdapter = infrastructure.AuthConfigAdapter
)

// NewAuthConfigAdapter creates a new auth config adapter
var NewAuthConfigAdapter = infrastructure.NewAuthConfigAdapter

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
	StorageProvider           = infrastructure.StorageProvider
	StorageCapability         = infrastructure.StorageCapability
	StorageCapabilityProvider = infrastructure.StorageCapabilityProvider
	StorageError              = infrastructure.StorageError
	StorageConfigAdapter      = infrastructure.StorageConfigAdapter
)

// NewStorageConfigAdapter creates a new storage config adapter
var NewStorageConfigAdapter = infrastructure.NewStorageConfigAdapter

// NewStorageError creates a new storage error
var NewStorageError = infrastructure.NewStorageError

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
type IDService = infrastructure.IDService

// NoOpIDService provides fallback functionality
type NoOpIDService = infrastructure.NoOpIDService

// NewNoOpIDService creates a fallback ID service
var NewNoOpIDService = infrastructure.NewNoOpIDService

// Transaction types
type TransactionService = infrastructure.TransactionService

// NoOpTransactionService does nothing - used as fallback
type NoOpTransactionService = infrastructure.NoOpTransactionService

// NewNoOpTransactionService creates a no-operation transaction service
var NewNoOpTransactionService = infrastructure.NewNoOpTransactionService

// Migration types
type (
	MigrationService = infrastructure.MigrationService
	MigrationStatus  = infrastructure.MigrationStatus
	AppliedMigration = infrastructure.AppliedMigration
	PendingMigration = infrastructure.PendingMigration
	MigrationError   = infrastructure.MigrationError
)

// Server types
type ServerProvider = infrastructure.ServerProvider

// NewMigrationError creates a new migration error
var NewMigrationError = infrastructure.NewMigrationError

// Migration error codes
const (
	MigrationErrCodeDirtyDatabase    = infrastructure.MigrationErrCodeDirtyDatabase
	MigrationErrCodeVersionNotFound  = infrastructure.MigrationErrCodeVersionNotFound
	MigrationErrCodeMigrationFailed  = infrastructure.MigrationErrCodeMigrationFailed
	MigrationErrCodeInvalidVersion   = infrastructure.MigrationErrCodeInvalidVersion
	MigrationErrCodeConnectionFailed = infrastructure.MigrationErrCodeConnectionFailed
)

// =============================================================================
// INTEGRATION PORTS (Email, Payment, Scheduler)
// =============================================================================

// Email types
type (
	EmailProvider   = integration.EmailProvider
	EmailMessage    = integration.EmailMessage
	EmailAttachment = integration.EmailAttachment
	InboxOptions    = integration.InboxOptions
)

// FromProtoMessage converts protobuf EmailMessage to EmailMessage
var FromProtoMessage = integration.FromProtoMessage

// Payment types
type (
	PaymentProvider       = integration.PaymentProvider
	PaymentWebhookResult  = integration.PaymentWebhookResult
	CheckoutSessionParams = integration.CheckoutSessionParams
)

// Scheduler types
type (
	SchedulerProvider       = integration.SchedulerProvider
	ScheduleWebhookResult   = integration.ScheduleWebhookResult
	CreateScheduleParams    = integration.CreateScheduleParams
	CheckAvailabilityParams = integration.CheckAvailabilityParams
)

// Tabular types
type (
	TabularSourceProvider = integration.TabularSourceProvider
	TabularOptions        = integration.TabularOptions
	TabularRecord         = integration.TabularRecord
	TabularSelection      = integration.TabularSelection
)

// =============================================================================
// DOMAIN PORTS (Workflow, Translation)
// =============================================================================

// Workflow types
type (
	WorkflowEngineService = domain.WorkflowEngineService
	ActivityExecutor      = domain.ActivityExecutor
	ExecutorRegistry      = domain.ExecutorRegistry
)

// Translation types
type TranslationService = domain.TranslationService

// NewNoOpTranslationService creates a non-operational fallback
var NewNoOpTranslationService = domain.NewNoOpTranslationService

// =============================================================================
// SECURITY PORTS (Authorization)
// =============================================================================

// Authorization types
type (
	AuthorizationService   = security.AuthorizationService
	AuthorizationProvider  = security.AuthorizationProvider
	AuthorizationError     = security.AuthorizationError
	AuthorizationErrorCode = security.AuthorizationErrorCode
)

// NewNoOpAuthorizationService creates a non-operational fallback
var NewNoOpAuthorizationService = security.NewNoOpAuthorizationService

// NewAuthorizationError creates a new authorization error
var NewAuthorizationError = security.NewAuthorizationError

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

// Entity constants (all 40+ entities)
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

	// Payment Domain
	EntityPayment                     = security.EntityPayment
	EntityPaymentAttribute            = security.EntityPaymentAttribute
	EntityPaymentMethod               = security.EntityPaymentMethod
	EntityPaymentProfile              = security.EntityPaymentProfile
	EntityPaymentProfilePaymentMethod = security.EntityPaymentProfilePaymentMethod

	// Product Domain
	EntityCollection          = security.EntityCollection
	EntityCollectionAttribute = security.EntityCollectionAttribute
	EntityCollectionPlan      = security.EntityCollectionPlan
	EntityPriceProduct        = security.EntityPriceProduct
	EntityProduct             = security.EntityProduct
	EntityProductAttribute    = security.EntityProductAttribute
	EntityProductCollection   = security.EntityProductCollection
	EntityProductPlan         = security.EntityProductPlan
	EntityResource            = security.EntityResource

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
)
