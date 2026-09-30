//go:build postgresql

package core

import (
	"encoding/csv"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Snapshot from the checked-in esqyma migration/live-column inventory in
// docs/plan/20260927-espyna-provider-parity/scope-registry.tsv.
// A factory is never inferred direct merely because it has a similarly named parent.
var factoryWorkspaceColumn = map[string]bool{
	"account":                               true,
	"accrued_expense":                       true,
	"accrued_expense_settlement":            true,
	"asset":                                 true,
	"asset_category":                        true,
	"asset_transaction":                     true,
	"attachment":                            true,
	"category":                              true,
	"charge_policy":                         true,
	"charge_policy_component":               true,
	"charge_policy_posting":                 true,
	"charge_policy_version":                 true,
	"charge_policy_version_editor":          true,
	"cost_source_component":                 true,
	"allocation_batch":                      true,
	"allocation_share":                      true,
	"agreement_line_term":                   true,
	"billable_charge":                       true,
	"charge_component":                      true,
	"document_series":                       true,
	"recovery_document":                     true,
	"recovery_document_line":                true,
	"collection_application":                true,
	"charge_effect":                         true,
	"client":                                true,
	"client_portal_grant":                   true,
	"client_workspace_user":                 true,
	"collection_billing_event":              true,
	"conversation":                          true,
	"conversation_participant":              true,
	"conversation_post":                     true,
	"conversation_read_receipt":             true,
	"cost_plan":                             true,
	"cost_schedule":                         true,
	"criteria_option":                       true,
	"criteria_threshold":                    true,
	"delegate_client":                       true,
	"delegate_supplier":                     true,
	"depreciation_run":                      true,
	"depreciation_schedule":                 true,
	"disbursement_supplier_billing_event":   true,
	"document_template":                     true,
	"evaluation":                            true,
	"evaluation_cycle":                      true,
	"evaluation_cycle_member":               true,
	"evaluation_response":                   true,
	"evaluation_template":                   true,
	"evaluation_template_item":              true,
	"event":                                 true,
	"event_attendee":                        true,
	"event_occurrence":                      true,
	"event_recurrence":                      true,
	"event_resource":                        true,
	"event_tag":                             true,
	"event_tag_assignment":                  true,
	"expenditure":                           true,
	"expense_recognition":                   true,
	"expense_recognition_line":              true,
	"expense_recognition_run":               true,
	"forex_rate":                            true,
	"fulfillment":                           true,
	"fund_allocation":                       true,
	"fund_transaction":                      true,
	"inventory_attribute":                   true,
	"inventory_depreciation":                true,
	"inventory_item":                        true,
	"inventory_movement":                    true,
	"inventory_serial":                      true,
	"inventory_serial_history":              true,
	"job":                                   true,
	"job_activity":                          true,
	"job_category":                          true,
	"job_outcome_line":                      true,
	"job_outcome_summary":                   true,
	"job_outcome_summary_document_template": true,
	"job_phase":                             true,
	"job_settlement":                        true,
	"job_task":                              true,
	"job_template":                          true,
	"job_template_document_template":        true,
	"job_template_phase":                    true,
	"job_template_relation":                 true,
	"job_template_task":                     true,
	"journal_entry":                         true,
	"leave_balance":                         true,
	"leave_request":                         true,
	"leave_type":                            true,
	"line":                                  true,
	"line_workspace_user":                   true,
	"location":                              true,
	"location_area":                         true,
	"outcome_criteria":                      true,
	"pay_cycle":                             true,
	"payment_term":                          true,
	"payroll_run":                           true,
	"permission":                            true,
	"phase_outcome_summary":                 true,
	"plan":                                  true,
	"plan_group":                            true,
	"plan_group_plan":                       true,
	"plan_job_template":                     true,
	"price_schedule":                        true,
	"price_schedule_workspace_user":         true,
	"procurement_request":                   true,
	"product":                               true,
	"product_plan_staff":                    true,
	"rate_table":                            true,
	"rating_description_set":                true,
	"rating_description_set_entry":          true,
	"rating_description_set_product_plan":   true,
	"reporting_checkpoint":                  true,
	"revenue":                               true,
	"revenue_line_item":                     true,
	"revenue_run":                           true,
	"revenue_tax_line":                      true,
	"role":                                  true,
	"score_scale":                           true,
	"score_scale_band":                      true,
	"scoring_component":                     true,
	"scoring_component_criteria":            true,
	"scoring_scheme":                        true,
	"session":                               true,
	"staff":                                 true,
	"subscription":                          true,
	"subscription_group":                    true,
	"subscription_group_document_template":  true,
	"subscription_group_member":             true,
	"subscription_group_product_plan":       true,
	"subscription_group_product_plan_staff": true,
	"subscription_group_workspace_user":     true,
	"subscription_seat":                     true,
	"subscription_workspace_user":           true,
	"supplier":                              true,
	"supplier_billing_event":                true,
	"supplier_contract":                     true,
	"supplier_contract_price_schedule":      true,
	"supplier_contract_price_schedule_line": true,
	"supplier_dependent":                    true,
	"supplier_lifecycle_event":              true,
	"supplier_plan":                         true,
	"supplier_portal_grant":                 true,
	"supplier_subscription":                 true,
	"task_outcome":                          true,
	"task_outcome_check":                    true,
	"tax_rate":                              true,
	"tax_registration":                      true,
	"template_task_criteria":                true,
	"template_task_criteria_rating_description": true,
	"tenant_invoice":          true,
	"tenant_payment_method":   true,
	"tenant_subscription":     true,
	"treasury_collection":     true,
	"treasury_disbursement":   true,
	"user_preference":         true,
	"withholding_certificate": true,
	"work_request":            true,
	"work_request_type":       true,
	"workflow":                true,
	"workflow_template":       true,
	"workspace_user":          true,
}

// Existing registrations outside the 61-row classification and the direct
// column snapshot. They remain explicit guard debt, not falsely marked global.
// A new entry is never added automatically. See plan tests.md.
var preexistingFactoryClassificationDebt = map[string]bool{
	"balance":                true,
	"balance_attribute":      true,
	"billing_event":          true,
	"deferred_revenue":       true,
	"delegate":               true,
	"event_attribute":        true,
	"expenditure_attribute":  true,
	"fund":                   true,
	"integration_payment":    true,
	"inventory_transaction":  true,
	"invoice_attribute":      true,
	"license":                true,
	"license_history":        true,
	"payroll_remittance":     true,
	"plan_attribute":         true,
	"plan_settings":          true,
	"prepayment":             true,
	"price_plan":             true,
	"product_plan":           true,
	"product_price_plan":     true,
	"rate_band":              true,
	"revenue_attribute":      true,
	"role_permission":        true,
	"subscription_attribute": true,
	"tax_authority":          true,
	"tax_class":              true,
	"tax_registration_kind":  true,
	"tax_treatment":          true,
	"user":                   true,
	"workspace":              true,
	"workspace_user_role":    true,
}

func factoryMechanism(table string) string {
	if factoryWorkspaceColumn[table] {
		return "own_column"
	}
	if columnLessTenantTables[table] {
		return "existing_parent"
	}
	if _, ok := columnlessChildChains[table]; ok {
		return "child_parent"
	}
	if pendingOwnColumn[table] {
		return "pending_own_column"
	}
	if _, ok := declaredGlobal[table]; ok {
		return "declared_global"
	}
	return ""
}

func TestColumnlessRegistryMatchesClassificationTSV(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		p := filepath.Join(dir, "docs/plan/20260927-columnless-tenant-scope/classification.tsv")
		if f, err := os.Open(p); err == nil {
			defer f.Close()
			reader := csv.NewReader(f)
			reader.Comma = '\t'
			reader.FieldsPerRecord = -1
			records, err := reader.ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			if len(records) == 0 {
				t.Fatal("empty classification")
			}
			head := map[string]int{}
			for i, v := range records[0] {
				head[v] = i
			}
			gotChildren := map[string]string{}
			gotRoots := map[string]bool{}
			for _, r := range records[1:] {
				switch r[head["proposed_class"]] {
				case "CHILD_PARENT_JOIN":
					gotChildren[r[head["table"]]] = r[head["parent_chain"]]
				case "ROOT_OWN_COLUMN":
					gotRoots[r[head["table"]]] = true
				default:
					t.Fatalf("unknown classification %q", r[head["proposed_class"]])
				}
			}
			if !reflect.DeepEqual(gotChildren, columnlessChildChains) {
				t.Fatalf("child registry differs from TSV: got %d want %d", len(columnlessChildChains), len(gotChildren))
			}
			if !reflect.DeepEqual(gotRoots, pendingOwnColumn) {
				t.Fatalf("pending roots differ from TSV: got %d want %d", len(pendingOwnColumn), len(gotRoots))
			}
			if len(gotChildren) != 37 || len(gotRoots) != 24 {
				t.Fatalf("classification sizes: child=%d root=%d", len(gotChildren), len(gotRoots))
			}
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("classification.tsv absent outside monorepo; registry is static at build time")
		}
		dir = parent
	}
}

func registeredPostgresFactoryTables(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	entityFile, err := parser.ParseFile(fset, filepath.Join("..", "..", "..", "..", "..", "registry", "entityid", "entityid.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	entityNames := map[string]string{}
	ast.Inspect(entityFile, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range spec.Names {
			if i >= len(spec.Values) {
				continue
			}
			lit, ok := spec.Values[i].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			value, err := strconv.Unquote(lit.Value)
			if err == nil {
				entityNames[name.Name] = value
			}
		}
		return true
	})
	tables := map[string]bool{}
	err = filepath.WalkDir("..", func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "RegisterRepositoryFactory" {
				return true
			}
			provider, ok := call.Args[0].(*ast.BasicLit)
			if !ok {
				t.Errorf("non-literal provider in %s cannot be enumerated", path)
				return true
			}
			if provider.Value != `"postgresql"` {
				return true
			}
			key, ok := call.Args[1].(*ast.SelectorExpr)
			if !ok {
				t.Errorf("non-entityid factory key in %s cannot be enumerated", path)
				return true
			}
			qualifier, ok := key.X.(*ast.Ident)
			if !ok || qualifier.Name != "entityid" {
				t.Errorf("non-entityid factory key in %s cannot be enumerated", path)
				return true
			}
			value, ok := entityNames[key.Sel.Name]
			if !ok {
				t.Errorf("unresolved factory entityid.%s", key.Sel.Name)
				return true
			}
			tables[value] = true
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(tables))
	for x := range tables {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func TestColumnlessFactoryMechanismGuard(t *testing.T) {
	tables := registeredPostgresFactoryTables(t)
	if len(tables) < 200 {
		t.Fatalf("factory inventory unexpectedly small: %d", len(tables))
	}
	newUnknown := newUnclassifiedFactoryTables(tables)
	var debt []string
	for _, table := range tables {
		if factoryMechanism(table) != "" {
			continue
		}
		if preexistingFactoryClassificationDebt[table] {
			debt = append(debt, table)
		}
	}
	if len(newUnknown) != 0 {
		t.Fatalf("new unclassified factory tables: %v", newUnknown)
	}
	if len(debt) != 31 {
		t.Fatalf("classification debt changed: %d entries, want 31", len(debt))
	}
	t.Logf("%d factories; %d pre-existing unclassified tables remain for owner classification", len(tables), len(debt))
}

func newUnclassifiedFactoryTables(tables []string) []string {
	var unknown []string
	for _, table := range tables {
		if factoryMechanism(table) == "" && !preexistingFactoryClassificationDebt[table] {
			unknown = append(unknown, table)
		}
	}
	return unknown
}

func TestColumnlessFactoryMechanismGuardRejectsSyntheticUnknown(t *testing.T) {
	const table = "synthetic_new_unclassified_table"
	if got := newUnclassifiedFactoryTables([]string{table}); !reflect.DeepEqual(got, []string{table}) {
		t.Fatalf("new unclassified table escaped guard: %v", got)
	}
}
