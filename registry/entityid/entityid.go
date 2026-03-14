// Package entityid provides compile-time constants for entity registry keys
// used in RegisterRepositoryFactory and CreateRepository calls.
// These keys are provider-agnostic and shared across postgresql, firestore, and mock.
package entityid

// Common domain
const (
	Attribute      = "attribute"
	AttributeValue = "attribute_value"
	Category       = "category"
)

// Entity domain
const (
	Admin             = "admin"
	Client            = "client"
	ClientAttribute   = "client_attribute"
	ClientCategory    = "client_category"
	Delegate          = "delegate"
	DelegateAttribute = "delegate_attribute"
	DelegateClient    = "delegate_client"
	Group             = "group"
	GroupAttribute    = "group_attribute"
	Location          = "location"
	LocationAttribute = "location_attribute"
	Permission        = "permission"
	Role              = "role"
	RolePermission    = "role_permission"
	Staff             = "staff"
	StaffAttribute    = "staff_attribute"
	Supplier          = "supplier"
	SupplierAttribute = "supplier_attribute"
	SupplierCategory  = "supplier_category"
	User              = "user"
	Workspace         = "workspace"
	WorkspaceUser     = "workspace_user"
	WorkspaceUserRole = "workspace_user_role"
)

// Event domain
const (
	Event          = "event"
	EventAttribute = "event_attribute"
	EventClient    = "event_client"
)

// Product domain
const (
	Collection           = "collection"
	CollectionAttribute  = "collection_attribute"
	CollectionPlan       = "collection_plan"
	PriceList            = "price_list"
	PriceProduct         = "price_product"
	Product              = "product"
	ProductAttribute     = "product_attribute"
	ProductCollection    = "product_collection"
	ProductOption        = "product_option"
	ProductOptionValue   = "product_option_value"
	ProductPlan          = "product_plan"
	ProductVariant       = "product_variant"
	ProductVariantImage  = "product_variant_image"
	ProductVariantOption = "product_variant_option"
	Resource             = "resource"
)

// Revenue domain
const (
	Revenue          = "revenue"
	RevenueAttribute = "revenue_attribute"
	RevenueCategory  = "revenue_category"
	RevenueLineItem  = "revenue_line_item"
)

// Expenditure domain
const (
	Expenditure          = "expenditure"
	ExpenditureAttribute = "expenditure_attribute"
	ExpenditureCategory  = "expenditure_category"
	ExpenditureLineItem  = "expenditure_line_item"
)

// Inventory domain
const (
	InventoryAttribute     = "inventory_attribute"
	InventoryDepreciation  = "inventory_depreciation"
	InventoryItem          = "inventory_item"
	InventorySerial        = "inventory_serial"
	InventorySerialHistory = "inventory_serial_history"
	InventoryTransaction   = "inventory_transaction"
)

// Subscription domain
const (
	Balance               = "balance"
	BalanceAttribute      = "balance_attribute"
	Invoice               = "invoice"
	InvoiceAttribute      = "invoice_attribute"
	License               = "license"
	LicenseHistory        = "license_history"
	Plan                  = "plan"
	PlanAttribute         = "plan_attribute"
	PlanSettings          = "plan_settings"
	PricePlan             = "price_plan"
	Subscription          = "subscription"
	SubscriptionAttribute = "subscription_attribute"
)

// Treasury domain
const (
	TreasuryCollection   = "treasury_collection"
	TreasuryDisbursement = "treasury_disbursement"
)

// Ledger / Document domain
const (
	Attachment       = "attachment"
	DocumentTemplate = "document_template"
)

// Integration domain
const (
	IntegrationPayment = "integration_payment"
)

// Workflow domain
const (
	Workflow         = "workflow"
	WorkflowTemplate = "workflow_template"
	Stage            = "stage"
	StageTemplate    = "stage_template"
	Activity         = "activity"
	ActivityTemplate = "activity_template"
)

// Operation domain
const (
	JobTemplate       = "job_template"
	JobTemplatePhase  = "job_template_phase"
	JobTemplateTask   = "job_template_task"
	Job               = "job"
	JobPhase          = "job_phase"
	JobTask           = "job_task"
	JobActivity       = "job_activity"
	ActivityLabor     = "activity_labor"
	ActivityMaterial  = "activity_material"
	ActivityExpense   = "activity_expense"
	JobSettlement     = "job_settlement"
	InventoryMovement = "inventory_movement"
)
