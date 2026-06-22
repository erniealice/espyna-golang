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
	Admin                  = "admin"
	Client                 = "client"
	ClientAttribute        = "client_attribute"
	ClientCategory         = "client_category"
	ClientPortalGrant      = "client_portal_grant"
	Delegate               = "delegate"
	DelegateAttribute      = "delegate_attribute"
	DelegateClient         = "delegate_client"
	DelegateSupplier       = "delegate_supplier"
	Group                  = "group"
	GroupAttribute         = "group_attribute"
	Location               = "location"
	LocationArea           = "location_area"
	LocationAttribute      = "location_attribute"
	Permission             = "permission"
	Role                   = "role"
	RolePermission         = "role_permission"
	Staff                  = "staff"
	StaffAttribute         = "staff_attribute"
	PaymentTerm            = "payment_term"
	Session                = "session"
	Supplier               = "supplier"
	SupplierAttribute      = "supplier_attribute"
	SupplierCategory       = "supplier_category"
	SupplierDependent      = "supplier_dependent"
	SupplierLifecycleEvent = "supplier_lifecycle_event"
	SupplierPortalGrant    = "supplier_portal_grant"
	User                   = "user"
	UserPreference         = "user_preference"
	Workspace              = "workspace"
	WorkspaceUser          = "workspace_user"
	WorkspaceUserRole      = "workspace_user_role"
	// Outsourcing-vertical client account-team membership (entity domain)
	ClientWorkspaceUser = "client_workspace_user"
)

// Event domain
const (
	Event              = "event"
	EventAttendee      = "event_attendee"
	EventAttribute     = "event_attribute"
	EventClient        = "event_client"
	EventOccurrence    = "event_occurrence"
	EventProduct       = "event_product"
	EventRecurrence    = "event_recurrence"
	EventResource      = "event_resource"
	EventTag           = "event_tag"
	EventTagAssignment = "event_tag_assignment"
)

// Communication domain
const (
	Conversation            = "conversation"
	ConversationPost        = "conversation_post"
	ConversationReadReceipt = "conversation_read_receipt"
	ConversationParticipant = "conversation_participant"
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
	// Education-grading R5 (2026-06-22): the plan-group taxonomy pair + the
	// staff-eligibility edge + the pre-existing line servicing grant (its proto
	// existed but the entity-id const was missing — added so it scaffolds).
	PlanGroup         = "plan_group"
	PlanGroupPlan     = "plan_group_plan"
	ProductPlanStaff  = "product_plan_staff"
	LineWorkspaceUser = "line_workspace_user"
)

// Revenue domain
const (
	Revenue           = "revenue"
	RevenueAttribute  = "revenue_attribute"
	RevenueCategory   = "revenue_category"
	RevenueLineItem   = "revenue_line_item"
	RevenueRun        = "revenue_run"
	RevenueRunAttempt = "revenue_run_attempt"
	RevenueTaxLine    = "revenue_tax_line"
	RevenuePayment    = "revenue_payment"
)

// Expenditure domain
const (
	Expenditure            = "expenditure"
	ExpenditureAttribute   = "expenditure_attribute"
	ExpenditureCategory    = "expenditure_category"
	ExpenditureLineItem    = "expenditure_line_item"
	SupplierContract       = "supplier_contract"
	SupplierContractLine   = "supplier_contract_line"
	ProcurementRequest     = "procurement_request"
	ProcurementRequestLine = "procurement_request_line"
	// Advance-cash-events (20260517) — buying-side mirror of BillingEvent.
	SupplierBillingEvent = "supplier_billing_event"
	// Expense Run (20260517) — recurrence-engine parent + per-attempt rows.
	ExpenseRecognitionRun        = "expense_recognition_run"
	ExpenseRecognitionRunAttempt = "expense_recognition_run_attempt"
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
	BillingEvent          = "billing_event"
	Invoice               = "invoice"
	InvoiceAttribute      = "invoice_attribute"
	License               = "license"
	LicenseHistory        = "license_history"
	Plan                  = "plan"
	PlanAttribute         = "plan_attribute"
	PlanSettings          = "plan_settings"
	PricePlan             = "price_plan"
	PriceSchedule         = "price_schedule"
	ProductPricePlan      = "product_price_plan"
	Subscription          = "subscription"
	SubscriptionAttribute = "subscription_attribute"
	// Outsourcing-vertical seat + servicing membership (subscription domain)
	SubscriptionSeat          = "subscription_seat"
	SubscriptionWorkspaceUser = "subscription_workspace_user"
	// Education-grading R5 (2026-06-22): the section cohort triplet + the class
	// edge (teacher-of-record) + the year-coordinator grant (5th *_workspace_user).
	SubscriptionGroup                 = "subscription_group"
	SubscriptionGroupMember           = "subscription_group_member"
	SubscriptionGroupWorkspaceUser    = "subscription_group_workspace_user"
	SubscriptionGroupProductPlanStaff = "subscription_group_product_plan_staff"
	PriceScheduleWorkspaceUser        = "price_schedule_workspace_user"
)

// Treasury domain
const (
	TreasuryCollection     = "treasury_collection"
	TreasuryDisbursement   = "treasury_disbursement"
	WithholdingCertificate = "withholding_certificate"
	// Advance-cash-events (20260517) — MILESTONE junction tables linking a
	// TreasuryCollection / TreasuryDisbursement to one or more BillingEvent /
	// SupplierBillingEvent rows. Tranche amounts are tracked per junction row;
	// SUM(tranche_amount) <= advance.amount is application-layer-enforced.
	CollectionBillingEvent           = "collection_billing_event"
	DisbursementSupplierBillingEvent = "disbursement_supplier_billing_event"
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
	JobTemplate         = "job_template"
	JobTemplatePhase    = "job_template_phase"
	JobTemplateTask     = "job_template_task"
	JobTemplateRelation = "job_template_relation"
	Job                 = "job"
	JobPhase            = "job_phase"
	JobTask             = "job_task"
	JobActivity         = "job_activity"
	ActivityLabor       = "activity_labor"
	ActivityMaterial    = "activity_material"
	ActivityExpense     = "activity_expense"
	JobSettlement       = "job_settlement"
	InventoryMovement   = "inventory_movement"
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

// Operation domain — Grading scoring-primitives (education-grading R5 — 2026-06-22)
//
// The composite-grading config (scheme + components + the component↔criteria
// junction + the transmutation scale/bands), the per-subject transcript line on
// a report card, and the data-driven reporting-checkpoint catalog (interim/final).
const (
	ScoringScheme            = "scoring_scheme"
	ScoringComponent         = "scoring_component"
	ScoringComponentCriteria = "scoring_component_criteria"
	ScoreScale               = "score_scale"
	ScoreScaleBand           = "score_scale_band"
	JobOutcomeLine           = "job_outcome_line"
	ReportingCheckpoint      = "reporting_checkpoint"
)

// Operation domain — Performance Evaluation (20260604-performance-evaluation v1)
const (
	Evaluation             = "evaluation"
	EvaluationResponse     = "evaluation_response"
	EvaluationTemplate     = "evaluation_template"
	EvaluationTemplateItem = "evaluation_template_item"
	EvaluationCycle        = "evaluation_cycle"
	EvaluationCycleMember  = "evaluation_cycle_member"
)

// Operation domain — Work Requests (20260604-requests-workflow v1)
const (
	WorkRequest     = "work_request"
	WorkRequestType = "work_request_type"
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

// Treasury domain — Collection Methods (extends existing Treasury)
const (
	CollectionMethod = "collection_method"
)

// Expenditure domain — Prepayments (extends existing Expenditure)
const (
	Prepayment = "prepayment"
)

// Expenditure domain — Purchase Orders (extends existing Expenditure)
const (
	PurchaseOrder         = "purchase_order"
	PurchaseOrderLineItem = "purchase_order_line_item"
)

// Expenditure domain — Supplier Pricing Symmetry (SPS Wave 2: 2026-04-30)
//
// SupplierContractPriceSchedule + Line model date-windowed pricing under a
// supplier contract (mirrors the sales-side PriceSchedule). ExpenseRecognition
// + Line model accrual-basis recognized expense (mirrors Revenue). AccruedExpense
// + Settlement model accrual-side commitments (no sales-side counterpart).
const (
	SupplierContractPriceSchedule     = "supplier_contract_price_schedule"
	SupplierContractPriceScheduleLine = "supplier_contract_price_schedule_line"
	ExpenseRecognition                = "expense_recognition"
	ExpenseRecognitionLine            = "expense_recognition_line"
	AccruedExpense                    = "accrued_expense"
	AccruedExpenseSettlement          = "accrued_expense_settlement"
)

// Revenue domain — Deferred Revenue (extends existing Revenue)
const (
	DeferredRevenue = "deferred_revenue"
)

// Payroll domain (NEW)
const (
	PayrollRun        = "payroll_run"
	PayrollRemittance = "payroll_remittance"
	PayCycle          = "pay_cycle"
	RateTable         = "rate_table"
	RateBand          = "rate_band"
	LeaveType         = "leave_type"
	LeaveBalance      = "leave_balance"
	LeaveRequest      = "leave_request"
)

// ---------------------------------------------------------------------------
// Domain-level slices
// ---------------------------------------------------------------------------

// CommonEntities lists all entity IDs in the Common domain.
var CommonEntities = []string{Attribute, AttributeValue, Category}

// EntityEntities lists all entity IDs in the Entity domain.
var EntityEntities = []string{
	Admin, Client, ClientAttribute, ClientCategory, ClientPortalGrant,
	Delegate, DelegateAttribute, DelegateClient, DelegateSupplier,
	Group, GroupAttribute,
	Location, LocationArea, LocationAttribute,
	Permission,
	Role, RolePermission,
	Staff, StaffAttribute,
	PaymentTerm,
	Session,
	Supplier, SupplierAttribute, SupplierCategory,
	SupplierDependent, SupplierLifecycleEvent, SupplierPortalGrant,
	User, UserPreference,
	Workspace, WorkspaceUser, WorkspaceUserRole,
	ClientWorkspaceUser,
}

// EventEntities lists all entity IDs in the Event domain.
var EventEntities = []string{
	Event, EventAttendee, EventAttribute, EventClient,
	EventOccurrence, EventProduct, EventRecurrence, EventResource,
	EventTag, EventTagAssignment,
}

// CommunicationEntities lists all entity IDs in the Communication domain.
var CommunicationEntities = []string{
	Conversation, ConversationPost, ConversationReadReceipt, ConversationParticipant,
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
	// Education-grading R5 (2026-06-22)
	PlanGroup, PlanGroupPlan, ProductPlanStaff, LineWorkspaceUser,
}

// RevenueEntities lists all entity IDs in the Revenue domain.
var RevenueEntities = []string{Revenue, RevenueAttribute, RevenueCategory, RevenueLineItem, DeferredRevenue, RevenueRun, RevenueRunAttempt, RevenueTaxLine, RevenuePayment}

// ExpenditureEntities lists all entity IDs in the Expenditure domain.
var ExpenditureEntities = []string{
	Expenditure, ExpenditureAttribute, ExpenditureCategory, ExpenditureLineItem,
	Prepayment, PurchaseOrder, PurchaseOrderLineItem,
	SupplierContract, SupplierContractLine,
	ProcurementRequest, ProcurementRequestLine,
	// SPS Wave 2 (2026-04-30)
	SupplierContractPriceSchedule, SupplierContractPriceScheduleLine,
	ExpenseRecognition, ExpenseRecognitionLine,
	AccruedExpense, AccruedExpenseSettlement,
	// Advance-cash-events + Expense Run (20260517)
	SupplierBillingEvent,
	ExpenseRecognitionRun, ExpenseRecognitionRunAttempt,
}

// InventoryEntities lists all entity IDs in the Inventory domain.
var InventoryEntities = []string{
	InventoryAttribute, InventoryDepreciation, InventoryItem,
	InventorySerial, InventorySerialHistory, InventoryTransaction,
}

// SubscriptionEntities lists all entity IDs in the Subscription domain.
var SubscriptionEntities = []string{
	Balance, BalanceAttribute,
	BillingEvent,
	Invoice, InvoiceAttribute,
	License, LicenseHistory,
	Plan, PlanAttribute, PlanSettings,
	PricePlan, PriceSchedule, ProductPricePlan,
	Subscription, SubscriptionAttribute,
	SubscriptionSeat, SubscriptionWorkspaceUser,
	// Education-grading R5 (2026-06-22)
	SubscriptionGroup, SubscriptionGroupMember, SubscriptionGroupWorkspaceUser,
	SubscriptionGroupProductPlanStaff, PriceScheduleWorkspaceUser,
}

// TreasuryEntities lists all entity IDs in the Treasury domain.
var TreasuryEntities = []string{
	TreasuryCollection, TreasuryDisbursement,
	CollectionSchedule, DisbursementSchedule,
	Loan, LoanPayment, SecurityDeposit,
	PettyCashFund, PettyCashVoucher, PettyCashReplenishment,
	CollectionMethod,
	WithholdingCertificate,
	// Advance-cash-events (20260517) — MILESTONE junction tables
	CollectionBillingEvent,
	DisbursementSupplierBillingEvent,
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
	JobTemplate, JobTemplatePhase, JobTemplateTask, JobTemplateRelation,
	Job, JobPhase, JobTask, JobActivity,
	ActivityLabor, ActivityMaterial, ActivityExpense,
	JobSettlement, InventoryMovement,
	Evaluation, EvaluationResponse, EvaluationTemplate,
	EvaluationTemplateItem, EvaluationCycle, EvaluationCycleMember,
	WorkRequest, WorkRequestType,
}

// OperationOutcomeEntities lists all entity IDs in the Operation Layer 7 Outcome domain.
var OperationOutcomeEntities = []string{
	OutcomeCriteria, CriteriaThreshold, CriteriaOption,
	TemplateTaskCriteria,
	TaskOutcome, TaskOutcomeCheck,
	PhaseOutcomeSummary, JobOutcomeSummary,
	// Grading scoring-primitives (R5 — 2026-06-22)
	ScoringScheme, ScoringComponent, ScoringComponentCriteria,
	ScoreScale, ScoreScaleBand, JobOutcomeLine, ReportingCheckpoint,
}

// LedgerAccountingEntities lists all entity IDs in the Ledger accounting domain.
var LedgerAccountingEntities = []string{
	Account, AccountGroup, AccountTemplate,
	JournalEntry, JournalLine,
	FiscalPeriod, RecurringJournalTemplate,
	EquityAccount, EquityTransaction,
}

// PayrollEntities lists all entity IDs in the Payroll domain.
var PayrollEntities = []string{
	PayrollRun, PayrollRemittance,
	PayCycle, RateTable, RateBand,
	LeaveType, LeaveBalance, LeaveRequest,
}

// Asset domain
const (
	Asset                = "asset"
	AssetCategory        = "asset_category"
	AssetTransaction     = "asset_transaction"
	DepreciationSchedule = "depreciation_schedule"
	DepreciationRun      = "depreciation_run"
	AssetRevaluation     = "asset_revaluation"
)

// Procurement domain (Supplier Subscriptions — 2026-05-06)
//
// Buying-side mirror of the Subscription domain. Six entities model the
// procurement pricing graph and the resulting SupplierSubscription.
const (
	CostSchedule            = "cost_schedule"
	SupplierPlan            = "supplier_plan"
	CostPlan                = "cost_plan"
	SupplierProductPlan     = "supplier_product_plan"
	SupplierProductCostPlan = "supplier_product_cost_plan"
	SupplierSubscription    = "supplier_subscription"
)

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

// AssetEntities lists all entity IDs in the Asset domain.
var AssetEntities = []string{Asset, AssetCategory, AssetTransaction, DepreciationSchedule, DepreciationRun, AssetRevaluation}

// ProcurementEntities lists all entity IDs in the Procurement domain.
var ProcurementEntities = []string{
	CostSchedule, SupplierPlan, CostPlan,
	SupplierProductPlan, SupplierProductCostPlan,
	SupplierSubscription,
}

// Tax domain (Tax Integration v1 — 2026-05-09)
//
// Six tax-domain entities: four lookup-only (TaxAuthority, TaxRegistrationKind,
// TaxTreatment, TaxClass), one read-only-with-find_applicable (TaxRate), and one
// full-CRUD-via-supersession (TaxRegistration).
const (
	TaxAuthority        = "tax_authority"
	TaxRegistrationKind = "tax_registration_kind"
	TaxTreatment        = "tax_treatment"
	TaxClass            = "tax_class"
	TaxRate             = "tax_rate"
	TaxRegistration     = "tax_registration"
)

// TaxEntities lists all entity IDs in the Tax domain.
var TaxEntities = []string{
	TaxAuthority, TaxRegistrationKind, TaxTreatment, TaxClass, TaxRate, TaxRegistration,
}

// Finance domain (NEW — Forex Rate; Tax Integration v1 — 2026-05-09)
const (
	ForexRate = "forex_rate"
)

// FinanceEntities lists all entity IDs in the Finance domain.
var FinanceEntities = []string{ForexRate}

// Tenancy domain (NEW — Portal E2E Wave 3 — 2026-05-17)
//
// Three entities model the Ichizen platform subscription, payment method,
// and invoice for a workspace tenant. These are billing-side records owned
// by the Ichizen platform itself (not the workspace's customers/suppliers).
const (
	TenantSubscription  = "tenant_subscription"
	TenantPaymentMethod = "tenant_payment_method"
	TenantInvoice       = "tenant_invoice"
)

// TenancyEntities lists all entity IDs in the Tenancy domain.
var TenancyEntities = []string{TenantSubscription, TenantPaymentMethod, TenantInvoice}

// Funding domain (NEW — Shared Fund Sources FS-A/FS-B — 2026-05-17)
//
// Three entities model cross-workspace shared funding sources:
//   - Fund: global entity (no workspace_id); the physical/financial instrument.
//   - FundAllocation: workspace-scoped junction binding a Fund to a workspace.
//   - FundTransaction: append-only event log for all money movements on a Fund.
//     workspace_id is nullable — fund-global events (OPENING_BALANCE, LIMIT_*)
//     have no workspace attribution; workspace-originated events carry workspace_id.
const (
	Fund            = "fund"
	FundAllocation  = "fund_allocation"
	FundTransaction = "fund_transaction"
)

// FundingEntities lists all entity IDs in the Funding domain.
var FundingEntities = []string{Fund, FundAllocation, FundTransaction}

// ---------------------------------------------------------------------------
// Permission action vocabulary
// ---------------------------------------------------------------------------

// These are the canonical RBAC action verbs used in permission codes
// (e.g. entityid.Evaluation + ":" + entityid.ActionList = "evaluation:list").
// They live here — not in ports/security — because they are domain vocabulary:
// the verbs that name what a principal can do, independent of any authorization
// implementation.
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionList   = "list"
	ActionManage = "manage"
)

// EntityPermission builds a permission code from an entity name and action verb.
// Example: EntityPermission(Evaluation, ActionList) = "evaluation:list".
func EntityPermission(entity, action string) string {
	return entity + ":" + action
}

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
	all = append(all, AssetEntities...)
	all = append(all, ProcurementEntities...)
	all = append(all, TaxEntities...)
	all = append(all, FinanceEntities...)
	all = append(all, TenancyEntities...)
	all = append(all, FundingEntities...)
	return all
}
