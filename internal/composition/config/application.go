package config

// DatabaseTableConfig holds database table/collection names for all database types
// This provides consistent naming across PostgreSQL, Firestore, and mock implementations
type DatabaseTableConfig struct {
	// Common tables - Shared across domains
	Attribute string

	// Entity tables - User management and workspace hierarchy
	Client            string
	ClientAttribute   string
	Admin             string
	Manager           string
	Staff             string
	StaffAttribute    string
	Delegate          string
	DelegateAttribute string
	DelegateClient    string
	Group             string
	GroupAttribute    string
	Location          string
	LocationAttribute string
	Permission        string
	Role              string
	RolePermission    string
	User              string
	Workspace         string
	WorkspaceClient   string
	WorkspaceUser     string
	WorkspaceUserRole string

	// Event tables - Scheduling and appointment management
	Event          string
	EventAttribute string
	EventClient    string
	EventProduct   string
	EventSettings  string

	// Framework tables - Task and objective management
	Framework string
	Objective string
	Task      string

	// Payment tables - Payment processing and method management
	Payment                     string
	PaymentAttribute            string
	PaymentMethod               string
	PaymentProfile              string
	PaymentProfilePaymentMethod string

	// Product tables - Product catalog and resource management
	Product             string
	Collection          string
	CollectionAttribute string
	CollectionParent    string
	CollectionPlan      string
	PriceProduct        string
	ProductAttribute    string
	ProductCollection   string
	ProductPlan         string
	Resource            string

	// Record tables - Document and record management
	Record string

	// Workflow tables - Workflow, stage, and activity management
	Workflow         string
	WorkflowTemplate string
	Stage            string
	Activity         string
	StageTemplate    string
	ActivityTemplate string

	// Subscription tables - Billing and subscription management
	Plan                  string
	PlanLocation          string
	PlanSettings          string
	Balance               string
	Invoice               string
	PricePlan             string
	Subscription          string
	BalanceAttribute      string
	InvoiceAttribute      string
	PlanAttribute         string
	SubscriptionAttribute string
}

// DefaultDatabaseTableConfig returns the default table/collection names used across all database types
// This ensures consistent naming conventions across PostgreSQL, Firestore, and mock implementations
func DefaultDatabaseTableConfig() DatabaseTableConfig {
	return DatabaseTableConfig{
		// Common tables
		Attribute: "attribute",

		// Entity tables
		Client:            "client",
		ClientAttribute:   "client_attribute",
		Admin:             "admin",
		Manager:           "manager",
		Staff:             "staff",
		StaffAttribute:    "staff_attribute",
		Delegate:          "delegate",
		DelegateAttribute: "delegate_attribute",
		DelegateClient:    "delegate_client",
		Group:             "group",
		GroupAttribute:    "group_attribute",
		Location:          "location",
		LocationAttribute: "location_attribute",
		Permission:        "permission",
		Role:              "role",
		RolePermission:    "role_permission",
		User:              "user",
		Workspace:         "workspace",
		WorkspaceClient:   "workspace_client",
		WorkspaceUser:     "workspace_user",
		WorkspaceUserRole: "workspace_user_role",

		// Event tables
		Event:          "event",
		EventAttribute: "event_attribute",
		EventClient:    "event_client",
		EventProduct:   "event_product",
		EventSettings:  "event_settings",

		// Framework tables
		Framework: "framework",
		Objective: "objective",
		Task:      "task",

		// Payment tables
		Payment:                     "payment",
		PaymentAttribute:            "payment_attribute",
		PaymentMethod:               "payment_method",
		PaymentProfile:              "payment_profile",
		PaymentProfilePaymentMethod: "payment_profile_payment_method",

		// Product tables
		Product:             "product",
		Collection:          "collection",
		CollectionAttribute: "collection_attribute",
		CollectionParent:    "collection_parent",
		CollectionPlan:      "collection_plan",
		PriceProduct:        "price_product",
		ProductAttribute:    "product_attribute",
		ProductCollection:   "product_collection",
		ProductPlan:         "product_plan",
		Resource:            "resource",

		// Record tables
		Record: "record",

		// Workflow tables
		Workflow:         "workflow",
		WorkflowTemplate: "workflow_template",
		Stage:            "stage",
		Activity:         "activity",
		StageTemplate:    "stage_template",
		ActivityTemplate: "activity_template",

		// Subscription tables
		Plan:                  "plan",
		PlanLocation:          "plan_location",
		PlanSettings:          "plan_settings",
		Balance:               "balance",
		Invoice:               "invoice",
		PricePlan:             "price_plan",
		Subscription:          "subscription",
		BalanceAttribute:      "balance_attribute",
		InvoiceAttribute:      "invoice_attribute",
		PlanAttribute:         "plan_attribute",
		SubscriptionAttribute: "subscription_attribute",
	}
}
