// Package entityid provides compile-time constants for entity registry keys
// used in RegisterRepositoryFactory and CreateRepository calls.
// These keys are provider-agnostic and shared across postgresql, firestore, and mock.
//
// The constant value IS the default table/collection name. No separate table
// config struct is needed — registry.TableConfig derives defaults from these
// values and only stores overrides (e.g., from POSTGRES_TABLE_* env vars).
//
// # Adding a new entity
//
//  1. Add the constant to the appropriate domain group below.
//  2. Add it to the corresponding domain slice (e.g., EntityEntities).
//     The All slice picks it up automatically via buildAll().
//
// That's it for table name resolution. The rest depends on what the entity needs:
//
// # Where entityid constants are consumed
//
//   - Adapter registration (init):
//     contrib/postgres/internal/adapter/{domain}/{entity}.go
//     contrib/google/internal/database/firestore/{domain}/{entity}.go
//     internal/infrastructure/adapters/secondary/database/mock/{domain}/{entity}.go
//     Each adapter calls: registry.RegisterRepositoryFactory("postgresql", entityid.X, factory)
//
//   - Repository composition:
//     internal/composition/providers/domain/{domain}.go
//     Each domain provider calls: repoCreator.CreateRepository(entityid.X, conn, tableConfig.TableName(entityid.X))
//
//   - Proto schema (if new entity needs a service interface):
//     esqyma: pkg/schema/v1/domain/{domain}/{entity}/
//
// # Adding an entirely new domain (not just a new entity in an existing domain)
//
//  1. Add const block + domain slice + wire into buildAll() in this file.
//  2. Create proto schema in esqyma: pkg/schema/v1/domain/{domain}/{entity}/
//  3. Create adapters for each provider (postgres, firestore, mock) with init() registration.
//  4. Create domain provider: internal/composition/providers/domain/{domain}.go
//     with New{Domain}Repositories(dbProvider, tableConfig) function.
//  5. Wire into composition: internal/composition/core/usecases.go
//     (call New{Domain}Repositories and create use cases).
//  6. Create use cases: internal/application/usecases/{domain}/
//  7. Create initializer: internal/composition/core/initializers/{domain}.go
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
	LocationArea      = "location_area"
	LocationAttribute = "location_attribute"
	Permission        = "permission"
	Role              = "role"
	RolePermission    = "role_permission"
	Staff             = "staff"
	StaffAttribute    = "staff_attribute"
	PaymentTerm       = "payment_term"
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
	Event           = "event"
	EventAttendee   = "event_attendee"
	EventAttribute  = "event_attribute"
	EventClient     = "event_client"
	EventOccurrence = "event_occurrence"
	EventProduct    = "event_product"
	EventRecurrence = "event_recurrence"
	EventResource   = "event_resource"
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
	Line                 = "line"
	ProductLine          = "product_line"
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
	ProductPricePlan      = "product_price_plan"
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

// Operation domain — Layer 7: Outcome
const (
	OutcomeCriteria      = "outcome_criteria"
	CriteriaThreshold    = "criteria_threshold"
	CriteriaOption       = "criteria_option"
	TemplateTaskCriteria = "template_task_criteria"
	TaskOutcome          = "task_outcome"
	TaskOutcomeCheck     = "task_outcome_check"
	PhaseOutcomeSummary  = "phase_outcome_summary"
	JobOutcomeSummary    = "job_outcome_summary"
)

// Ledger domain — Chart of Accounts
const (
	Account                  = "account"
	AccountGroup             = "account_group"
	AccountTemplate          = "account_template"
	JournalEntry             = "journal_entry"
	JournalLine              = "journal_line"
	FiscalPeriod             = "fiscal_period"
	RecurringJournalTemplate = "recurring_journal_template"
	EquityAccount            = "equity_account"
	EquityTransaction        = "equity_transaction"
)

// Treasury domain — Schedules (extends existing Treasury)
const (
	CollectionSchedule   = "collection_schedule"
	DisbursementSchedule = "disbursement_schedule"
)

// Treasury domain — Loans & Petty Cash (extends existing Treasury)
const (
	Loan                   = "loan"
	LoanPayment            = "loan_payment"
	SecurityDeposit        = "security_deposit"
	PettyCashFund          = "petty_cash_fund"
	PettyCashVoucher       = "petty_cash_voucher"
	PettyCashReplenishment = "petty_cash_replenishment"
)

// Expenditure domain — Prepayments (extends existing Expenditure)
const (
	Prepayment = "prepayment"
)

// Revenue domain — Deferred Revenue (extends existing Revenue)
const (
	DeferredRevenue = "deferred_revenue"
)

// Payroll domain (NEW)
const (
	PayrollRun        = "payroll_run"
	PayrollRemittance = "payroll_remittance"
)

// ---------------------------------------------------------------------------
// Domain-level slices
// ---------------------------------------------------------------------------

// CommonEntities lists all entity IDs in the Common domain.
var CommonEntities = []string{Attribute, AttributeValue, Category}

// EntityEntities lists all entity IDs in the Entity domain.
var EntityEntities = []string{
	Admin, Client, ClientAttribute, ClientCategory,
	Delegate, DelegateAttribute, DelegateClient,
	Group, GroupAttribute,
	Location, LocationArea, LocationAttribute,
	Permission,
	Role, RolePermission,
	Staff, StaffAttribute,
	PaymentTerm,
	Supplier, SupplierAttribute, SupplierCategory,
	User,
	Workspace, WorkspaceUser, WorkspaceUserRole,
}

// EventEntities lists all entity IDs in the Event domain.
var EventEntities = []string{
	Event, EventAttendee, EventAttribute, EventClient,
	EventOccurrence, EventProduct, EventRecurrence, EventResource,
}

// ProductEntities lists all entity IDs in the Product domain.
var ProductEntities = []string{
	Collection, CollectionAttribute, CollectionPlan,
	PriceList, PriceProduct,
	Product, ProductAttribute, Line,
	ProductLine,
	ProductOption, ProductOptionValue,
	ProductPlan, ProductVariant, ProductVariantImage, ProductVariantOption,
	Resource,
}

// RevenueEntities lists all entity IDs in the Revenue domain.
var RevenueEntities = []string{Revenue, RevenueAttribute, RevenueCategory, RevenueLineItem, DeferredRevenue}

// ExpenditureEntities lists all entity IDs in the Expenditure domain.
var ExpenditureEntities = []string{Expenditure, ExpenditureAttribute, ExpenditureCategory, ExpenditureLineItem, Prepayment}

// InventoryEntities lists all entity IDs in the Inventory domain.
var InventoryEntities = []string{
	InventoryAttribute, InventoryDepreciation, InventoryItem,
	InventorySerial, InventorySerialHistory, InventoryTransaction,
}

// SubscriptionEntities lists all entity IDs in the Subscription domain.
var SubscriptionEntities = []string{
	Balance, BalanceAttribute,
	Invoice, InvoiceAttribute,
	License, LicenseHistory,
	Plan, PlanAttribute, PlanSettings,
	PricePlan, ProductPricePlan,
	Subscription, SubscriptionAttribute,
}

// TreasuryEntities lists all entity IDs in the Treasury domain.
var TreasuryEntities = []string{
	TreasuryCollection, TreasuryDisbursement,
	CollectionSchedule, DisbursementSchedule,
	Loan, LoanPayment, SecurityDeposit,
	PettyCashFund, PettyCashVoucher, PettyCashReplenishment,
}

// LedgerDocumentEntities lists all entity IDs in the Ledger / Document domain.
var LedgerDocumentEntities = []string{Attachment, DocumentTemplate}

// IntegrationEntities lists all entity IDs in the Integration domain.
var IntegrationEntities = []string{IntegrationPayment}

// WorkflowEntities lists all entity IDs in the Workflow domain.
var WorkflowEntities = []string{
	Workflow, WorkflowTemplate,
	Stage, StageTemplate,
	Activity, ActivityTemplate,
}

// OperationEntities lists all entity IDs in the Operation domain.
var OperationEntities = []string{
	JobTemplate, JobTemplatePhase, JobTemplateTask,
	Job, JobPhase, JobTask, JobActivity,
	ActivityLabor, ActivityMaterial, ActivityExpense,
	JobSettlement, InventoryMovement,
}

// OperationOutcomeEntities lists all entity IDs in the Operation Layer 7 Outcome domain.
var OperationOutcomeEntities = []string{
	OutcomeCriteria, CriteriaThreshold, CriteriaOption,
	TemplateTaskCriteria,
	TaskOutcome, TaskOutcomeCheck,
	PhaseOutcomeSummary, JobOutcomeSummary,
}

// LedgerAccountingEntities lists all entity IDs in the Ledger accounting domain.
var LedgerAccountingEntities = []string{
	Account, AccountGroup, AccountTemplate,
	JournalEntry, JournalLine,
	FiscalPeriod, RecurringJournalTemplate,
	EquityAccount, EquityTransaction,
}

// PayrollEntities lists all entity IDs in the Payroll domain.
var PayrollEntities = []string{PayrollRun, PayrollRemittance}

// Fulfillment domain
const (
	Fulfillment            = "fulfillment"
	FulfillmentItem        = "fulfillment_item"
	FulfillmentStatusEvent = "fulfillment_status_event"
	FulfillmentReturn      = "fulfillment_return"
	FulfillmentReturnItem  = "fulfillment_return_item"
)

// FulfillmentEntities lists all entity IDs in the Fulfillment domain.
var FulfillmentEntities = []string{Fulfillment, FulfillmentItem, FulfillmentStatusEvent, FulfillmentReturn, FulfillmentReturnItem}

// ---------------------------------------------------------------------------
// Consolidated slice
// ---------------------------------------------------------------------------

// All contains every registered entity ID constant.
var All = buildAll()

func buildAll() []string {
	var all []string
	all = append(all, CommonEntities...)
	all = append(all, EntityEntities...)
	all = append(all, EventEntities...)
	all = append(all, ProductEntities...)
	all = append(all, RevenueEntities...)
	all = append(all, ExpenditureEntities...)
	all = append(all, InventoryEntities...)
	all = append(all, SubscriptionEntities...)
	all = append(all, TreasuryEntities...)
	all = append(all, LedgerDocumentEntities...)
	all = append(all, IntegrationEntities...)
	all = append(all, WorkflowEntities...)
	all = append(all, OperationEntities...)
	all = append(all, OperationOutcomeEntities...)
	all = append(all, LedgerAccountingEntities...)
	all = append(all, PayrollEntities...)
	all = append(all, FulfillmentEntities...)
	return all
}
