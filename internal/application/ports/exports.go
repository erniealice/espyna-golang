package ports

// This file provides backward compatibility by re-exporting types from sub-packages.
// Existing code that imports "github.com/erniealice/espyna-golang/internal/application/ports" continues to work.
// New code can import specific sub-packages for cleaner dependencies:
//   - github.com/erniealice/espyna-golang/internal/application/ports/infrastructure
//   - github.com/erniealice/espyna-golang/internal/application/ports/integration
//   - github.com/erniealice/espyna-golang/internal/application/ports/domain
//   - github.com/erniealice/espyna-golang/internal/application/ports/security

import (
	"github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/ports/infrastructure"
	"github.com/erniealice/espyna-golang/internal/application/ports/integration"
	"github.com/erniealice/espyna-golang/internal/application/ports/security"
)

// =============================================================================
// INFRASTRUCTURE PORTS (Database, Auth, Storage, ID, Transaction, Migration)
// =============================================================================

// Database types
type (
	DatabaseProvider         = infrastructure.DatabaseProvider
	PoolSizer                = infrastructure.PoolSizer
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
	StreamingStorageProvider  = infrastructure.StreamingStorageProvider
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
type IDGenerator = infrastructure.IDGenerator

// NoOpIDGenerator provides fallback functionality
type NoOpIDGenerator = infrastructure.NoOpIDGenerator

// NewNoOpIDGenerator creates a fallback ID service
var NewNoOpIDGenerator = infrastructure.NewNoOpIDGenerator

// Transaction types
type Transactor = infrastructure.Transactor

// NoOpTransactor does nothing - used as fallback
type NoOpTransactor = infrastructure.NoOpTransactor

// NewNoOpTransactor creates a no-operation transaction service
var NewNoOpTransactor = infrastructure.NewNoOpTransactor

// Reference checker — application port over postgres reference.Checker.
type ReferenceChecker = infrastructure.ReferenceChecker

// NewNoOpReferenceChecker returns a stub checker (reports nothing in use).
var NewNoOpReferenceChecker = infrastructure.NewNoOpReferenceChecker

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

// Fulfillment types
type (
	FulfillmentProvider        = integration.FulfillmentProvider
	FulfillmentQuoteRequest    = integration.FulfillmentQuoteRequest
	FulfillmentQuoteResponse   = integration.FulfillmentQuoteResponse
	CreateDeliveryRequest      = integration.CreateDeliveryRequest
	CreateDeliveryResponse     = integration.CreateDeliveryResponse
	CancelDeliveryRequest      = integration.CancelDeliveryRequest
	CancelDeliveryResponse     = integration.CancelDeliveryResponse
	TrackDeliveryRequest       = integration.TrackDeliveryRequest
	TrackDeliveryResponse      = integration.TrackDeliveryResponse
	FulfillmentWebhookRequest  = integration.FulfillmentWebhookRequest
	FulfillmentWebhookResponse = integration.FulfillmentWebhookResponse
	FulfillmentAddress         = integration.Address
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
type Translator = domain.Translator

// NewNoOpTranslator creates a non-operational fallback
var NewNoOpTranslator = domain.NewNoOpTranslator

// Ledger types
type LedgerReportingService = domain.LedgerReportingService

// Evaluation types
type (
	OutcomeEvaluationService = domain.OutcomeEvaluationService
	EvaluationResult         = domain.EvaluationResult
)

// =============================================================================
// SECURITY PORTS (Authorization)
// =============================================================================

// Authorization types
type (
	Authorizer             = security.Authorizer
	AuthorizationProvider  = security.AuthorizationProvider
	AuthorizationError     = security.AuthorizationError
	AuthorizationErrorCode = security.AuthorizationErrorCode
)

// NewNoOpAuthorizer creates a non-operational fallback
var NewNoOpAuthorizer = security.NewNoOpAuthorizer

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
	EntityEvent              = security.EntityEvent
	EntityEventAttendee      = security.EntityEventAttendee
	EntityEventAttribute     = security.EntityEventAttribute
	EntityEventClient        = security.EntityEventClient
	EntityEventOccurrence    = security.EntityEventOccurrence
	EntityEventProduct       = security.EntityEventProduct
	EntityEventRecurrence    = security.EntityEventRecurrence
	EntityEventResource      = security.EntityEventResource
	EntityEventTag           = security.EntityEventTag
	EntityEventTagAssignment = security.EntityEventTagAssignment

	// Framework Domain
	EntityFramework = security.EntityFramework
	EntityObjective = security.EntityObjective
	EntityTask      = security.EntityTask

	// Payment Domain (legacy entities removed — superseded by Collection and Disbursement)

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
	EntityLine                 = security.EntityLine
	EntityProductLine          = security.EntityProductLine
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
	EntityPriceSchedule         = security.EntityPriceSchedule
	EntityProductPricePlan      = security.EntityProductPricePlan
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

	// Procurement Domain
	EntityCostSchedule            = security.EntityCostSchedule
	EntitySupplierPlan            = security.EntitySupplierPlan
	EntityCostPlan                = security.EntityCostPlan
	EntitySupplierProductPlan     = security.EntitySupplierProductPlan
	EntitySupplierProductCostPlan = security.EntitySupplierProductCostPlan
	EntitySupplierSubscription    = security.EntitySupplierSubscription

	// Operation Domain
	EntityJobTemplate       = security.EntityJobTemplate
	EntityJobTemplatePhase  = security.EntityJobTemplatePhase
	EntityJobTemplateTask   = security.EntityJobTemplateTask
	EntityJob               = security.EntityJob
	EntityJobPhase          = security.EntityJobPhase
	EntityJobTask           = security.EntityJobTask
	EntityJobActivity       = security.EntityJobActivity
	EntityActivityLabor     = security.EntityActivityLabor
	EntityActivityMaterial  = security.EntityActivityMaterial
	EntityActivityExpense   = security.EntityActivityExpense
	EntityJobSettlement     = security.EntityJobSettlement
	EntityInventoryMovement = security.EntityInventoryMovement

	// Operation Domain — Layer 7: Outcome
	EntityOutcomeCriteria      = security.EntityOutcomeCriteria
	EntityCriteriaThreshold    = security.EntityCriteriaThreshold
	EntityCriteriaOption       = security.EntityCriteriaOption
	EntityTemplateTaskCriteria = security.EntityTemplateTaskCriteria
	EntityTaskOutcome          = security.EntityTaskOutcome
	EntityTaskOutcomeCheck     = security.EntityTaskOutcomeCheck
	EntityPhaseOutcomeSummary  = security.EntityPhaseOutcomeSummary
	EntityJobOutcomeSummary    = security.EntityJobOutcomeSummary
)
