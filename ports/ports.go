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
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
)

// =============================================================================
// INFRASTRUCTURE PORTS
// =============================================================================

// Database types
type (
	DatabaseProvider         = internal.DatabaseProvider
	PoolSizer                = internal.PoolSizer
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
	AuthCapability    = internal.AuthCapability
	AuthConfigAdapter = internal.AuthConfigAdapter
)

var NewAuthConfigAdapter = internal.NewAuthConfigAdapter

// Audit context types — re-exported so contrib HTTP adapters can populate
// the audit context without importing internal/.
type AuditContext = infrastructure.AuditContext

var (
	WithAuditContext = infrastructure.WithAuditContext
	GetAuditContext  = infrastructure.GetAuditContext
)

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
	StreamingStorageProvider  = internal.StreamingStorageProvider
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
type IDGenerator = internal.IDGenerator
type NoOpIDGenerator = internal.NoOpIDGenerator

var NewNoOpIDGenerator = internal.NewNoOpIDGenerator

// Transaction types
type Transactor = internal.Transactor
type NoOpTransactor = internal.NoOpTransactor

var NewNoOpTransactor = internal.NewNoOpTransactor

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
	WorkflowEngineService          = internal.WorkflowEngineService
	WorkflowAssigneeQueryService   = internal.WorkflowAssigneeQueryService
	ActivityExecutor               = internal.ActivityExecutor
	ExecutorRegistry               = internal.ExecutorRegistry
)

// Workflow request/response types
type (
	ListPendingActivitiesForAssigneeRequest  = internal.ListPendingActivitiesForAssigneeRequest
	ListPendingActivitiesForAssigneeResponse = internal.ListPendingActivitiesForAssigneeResponse
)

// Translation types
type Translator = internal.Translator

var NewNoOpTranslator = internal.NewNoOpTranslator

// Ledger types
type LedgerReportingService = internal.LedgerReportingService

// =============================================================================
// SECURITY PORTS
// =============================================================================

// Authorization types
type (
	Authorizer             = internal.Authorizer
	AuthorizationProvider  = internal.AuthorizationProvider
	AuthorizationError     = internal.AuthorizationError
	AuthorizationErrorCode = internal.AuthorizationErrorCode
)

var NewNoOpAuthorizer = internal.NewNoOpAuthorizer
var NewAuthorizationError = internal.NewAuthorizationError

// Permission utility functions
var (
	EntityPermission = entityid.EntityPermission
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
	ActionCreate = entityid.ActionCreate
	ActionRead   = entityid.ActionRead
	ActionUpdate = entityid.ActionUpdate
	ActionDelete = entityid.ActionDelete
	ActionList   = entityid.ActionList
	ActionManage = entityid.ActionManage
)

// Entity constants
const (
	// Entity Domain
	EntityAdmin             = entityid.Admin
	EntityClient            = entityid.Client
	EntityClientAttribute   = entityid.ClientAttribute
	EntityDelegate          = entityid.Delegate
	EntityDelegateAttribute = entityid.DelegateAttribute
	EntityDelegateClient    = entityid.DelegateClient
	EntityGroup             = entityid.Group
	EntityGroupAttribute    = entityid.GroupAttribute
	EntityLocation          = entityid.Location
	EntityLocationAttribute = entityid.LocationAttribute
	EntityRole              = entityid.Role
	EntityRolePermission    = entityid.RolePermission
	EntityStaff             = entityid.Staff
	EntityStaffAttribute    = entityid.StaffAttribute
	EntityUser              = entityid.User
	EntityWorkspace         = entityid.Workspace
	EntityWorkspaceUser     = entityid.WorkspaceUser
	EntityWorkspaceUserRole = entityid.WorkspaceUserRole

	// Event Domain
	EntityEvent          = entityid.Event
	EntityEventAttribute = entityid.EventAttribute
	EntityEventClient    = entityid.EventClient
	EntityEventProduct   = entityid.EventProduct

	// Framework Domain

	// Expenditure Domain
	EntityExpenditure          = entityid.Expenditure
	EntityExpenditureAttribute = entityid.ExpenditureAttribute
	EntityExpenditureLineItem  = entityid.ExpenditureLineItem
	EntityExpenditureCategory  = entityid.ExpenditureCategory

	// Inventory Domain
	EntityInventoryItem          = entityid.InventoryItem
	EntityInventorySerial        = entityid.InventorySerial
	EntityInventoryTransaction   = entityid.InventoryTransaction
	EntityInventoryAttribute     = entityid.InventoryAttribute
	EntityInventoryDepreciation  = entityid.InventoryDepreciation
	EntityInventorySerialHistory = entityid.InventorySerialHistory

	// Product Domain
	EntityCollection           = entityid.Collection
	EntityCollectionAttribute  = entityid.CollectionAttribute
	EntityCollectionPlan       = entityid.CollectionPlan
	EntityPriceList            = entityid.PriceList
	EntityPriceProduct         = entityid.PriceProduct
	EntityProduct              = entityid.Product
	EntityProductAttribute     = entityid.ProductAttribute
	EntityLine                 = entityid.Line
	EntityProductLine          = entityid.ProductLine
	EntityProductOption        = entityid.ProductOption
	EntityProductOptionValue   = entityid.ProductOptionValue
	EntityProductPlan          = entityid.ProductPlan
	EntityProductVariant       = entityid.ProductVariant
	EntityProductVariantImage  = entityid.ProductVariantImage
	EntityProductVariantOption = entityid.ProductVariantOption
	EntityResource             = entityid.Resource

	// Record Domain

	// Subscription Domain
	EntityBalance               = entityid.Balance
	EntityBalanceAttribute      = entityid.BalanceAttribute
	EntityInvoice               = entityid.Invoice
	EntityInvoiceAttribute      = entityid.InvoiceAttribute
	EntityLicense               = entityid.License
	EntityLicenseHistory        = entityid.LicenseHistory
	EntityPlan                  = entityid.Plan
	EntityPlanAttribute         = entityid.PlanAttribute
	EntityPlanSettings          = entityid.PlanSettings
	EntityPricePlan             = entityid.PricePlan
	EntitySubscription          = entityid.Subscription
	EntitySubscriptionAttribute = entityid.SubscriptionAttribute

	// Asset Domain
	EntityAsset                = entityid.Asset
	EntityAssetCategory        = entityid.AssetCategory
	EntityDepreciationSchedule = entityid.DepreciationSchedule
	EntityAssetTransaction     = entityid.AssetTransaction
	EntityAssetRevaluation     = entityid.AssetRevaluation
)

// Ensure sub-package imports are used (prevents "imported and not used" errors
// when only type aliases from internal are used, since internal re-exports from
// these packages via its own type aliases).
var (
	_ domain.Translator
	_ infrastructure.DatabaseProvider
	_ integration.EmailProvider
	_ security.Authorizer
)
