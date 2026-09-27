//go:build postgresql

package core

// columnlessChildChains is the static shadow-only parent-chain proposal from
// classification.tsv. A suffix [P3 root required] means the current root lacks
// workspace_id; the probe logs unresolved until that migration is deployed.
// Existing columnLessTenantTables takes precedence and retains its behavior.
var columnlessChildChains = map[string]string{
	"activity_expense":           "activity_id:job_activity>workspace_id",
	"activity_labor":             "activity_id:job_activity>workspace_id",
	"activity_material":          "activity_id:job_activity>workspace_id",
	"activity_template":          "stage_template_id:stage_template>workflow_template_id:workflow_template>workspace_id",
	"asset_revaluation":          "asset_id:asset>workspace_id",
	"attribute_value":            "attribute_id:attribute>workspace_id[P3 root required]",
	"client_attribute":           "client_id:client>workspace_id",
	"client_category":            "client_id:client>workspace_id",
	"collection_attribute":       "collection_id:collection>workspace_id[P3 root required]",
	"collection_plan":            "plan_id:plan>workspace_id",
	"equity_transaction":         "equity_account_id:equity_account>workspace_id[P3 root required]",
	"event_client":               "event_id:event>workspace_id",
	"event_product":              "event_id:event>workspace_id",
	"fulfillment_item":           "fulfillment_id:fulfillment>workspace_id",
	"fulfillment_return":         "fulfillment_id:fulfillment>workspace_id",
	"fulfillment_return_item":    "fulfillment_return_id:fulfillment_return>fulfillment_id:fulfillment>workspace_id",
	"fulfillment_status_event":   "fulfillment_id:fulfillment>workspace_id",
	"journal_line":               "journal_entry_id:journal_entry>workspace_id",
	"loan_payment":               "loan_id:loan>account_id:account>workspace_id",
	"location_attribute":         "location_id:location>workspace_id",
	"petty_cash_replenishment":   "fund_id:petty_cash_fund>location_id:location>workspace_id",
	"petty_cash_voucher":         "fund_id:petty_cash_fund>location_id:location>workspace_id",
	"price_product":              "price_list_id:price_list>workspace_id[P3 root required]",
	"procurement_request_line":   "procurement_request_id:procurement_request>workspace_id",
	"product_attribute":          "product_id:product>workspace_id",
	"product_collection":         "product_id:product>workspace_id",
	"product_option":             "product_id:product>workspace_id",
	"product_option_value":       "product_option_id:product_option>product_id:product>workspace_id",
	"product_variant":            "product_id:product>workspace_id",
	"product_variant_image":      "product_variant_id:product_variant>product_id:product>workspace_id",
	"product_variant_option":     "product_variant_id:product_variant>product_id:product>workspace_id",
	"purchase_order_line_item":   "purchase_order_id:purchase_order>supplier_id:supplier>workspace_id",
	"stage":                      "workflow_id:workflow>workspace_id",
	"stage_template":             "workflow_template_id:workflow_template>workspace_id",
	"supplier_contract_line":     "supplier_contract_id:supplier_contract>workspace_id",
	"supplier_product_cost_plan": "supplier_product_plan_id:supplier_product_plan>supplier_plan_id:supplier_plan>workspace_id",
	"supplier_product_plan":      "supplier_plan_id:supplier_plan>workspace_id",
}

// pendingOwnColumn records the 24 proposed roots. No runtime mechanism is
// installed for them in this log-only wave.
var pendingOwnColumn = map[string]bool{
	"account_group":              true,
	"account_template":           true,
	"activity":                   true,
	"admin":                      true,
	"attribute":                  true,
	"collection":                 true,
	"collection_method":          true,
	"collection_schedule":        true,
	"disbursement_schedule":      true,
	"equity_account":             true,
	"expenditure_category":       true,
	"expenditure_line_item":      true,
	"fiscal_period":              true,
	"group":                      true,
	"loan":                       true,
	"petty_cash_fund":            true,
	"price_list":                 true,
	"product_line":               true,
	"purchase_order":             true,
	"recurring_journal_template": true,
	"resource":                   true,
	"revenue_category":           true,
	"revenue_payment":            true,
	"security_deposit":           true,
}

// declaredGlobal is intentionally empty: no table in the classification has
// a cited global ownership rule. Each future entry needs a written reason.
var declaredGlobal = map[string]string{}
