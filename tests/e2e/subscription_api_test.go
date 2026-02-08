//go:build mock_auth
package e2e

import (
	"testing"

	"github.com/erniealice/espyna-golang/tests/e2e/helper"
)

// TestSubscriptionDomainCRUDOperations validates create, read, and list operations for Subscription domain
func TestSubscriptionDomainCRUDOperations(t *testing.T) {
	env := helper.SetupTestEnvironment(t)

	// Test Subscription create/read/list operations
	t.Run("SubscriptionOperations", func(t *testing.T) {
		entityPath := "/api/subscription/subscription"

		t.Run("CreateSubscription", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("subscription")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("subscription")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("subscription")
			updateData := helper.GetUpdateDataForEntity("subscription")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListSubscriptions", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Balance create/read/list operations
	t.Run("BalanceOperations", func(t *testing.T) {
		entityPath := "/api/subscription/balance"

		t.Run("CreateBalance", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("balance")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("balance")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("balance")
			updateData := helper.GetUpdateDataForEntity("balance")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListBalances", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Invoice create/read/list operations
	t.Run("InvoiceOperations", func(t *testing.T) {
		entityPath := "/api/subscription/invoice"

		t.Run("CreateInvoice", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("invoice")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("invoice")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("invoice")
			updateData := helper.GetUpdateDataForEntity("invoice")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListInvoices", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Plan create/read/list operations
	t.Run("PlanOperations", func(t *testing.T) {
		entityPath := "/api/subscription/plan"

		t.Run("CreatePlan", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("plan")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("plan")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("plan")
			updateData := helper.GetUpdateDataForEntity("plan")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListPlans", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test PlanSettings create/read/list operations
	t.Run("PlanSettingsOperations", func(t *testing.T) {
		entityPath := "/api/subscription/plan-settings"

		t.Run("CreatePlanSettings", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("plan-settings")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("plan-settings")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("plan-settings")
			updateData := helper.GetUpdateDataForEntity("plan-settings")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListPlanSettings", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test PricePlan create/read/list operations
	t.Run("PricePlanOperations", func(t *testing.T) {
		entityPath := "/api/subscription/price-plan"

		t.Run("CreatePricePlan", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("price-plan")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("price-plan")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("price-plan")
			updateData := helper.GetUpdateDataForEntity("price-plan")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListPricePlans", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})
}
