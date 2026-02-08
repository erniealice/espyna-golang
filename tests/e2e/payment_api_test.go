//go:build mock_auth
package e2e

import (
	"testing"

	"github.com/erniealice/espyna-golang/tests/e2e/helper"
)

// TestPaymentDomainCRUDOperations validates create, read, and list operations for Payment domain
func TestPaymentDomainCRUDOperations(t *testing.T) {
	env := helper.SetupTestEnvironment(t)

	// Test Payment create/read/list operations
	t.Run("PaymentOperations", func(t *testing.T) {
		entityPath := "/api/payment/payment"

		t.Run("CreatePayment", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("payment")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("payment")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("payment")
			updateData := helper.GetUpdateDataForEntity("payment")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListPayments", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test PaymentMethod create/read/list operations
	t.Run("PaymentMethodOperations", func(t *testing.T) {
		entityPath := "/api/payment/payment-method"

		t.Run("CreatePaymentMethod", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("payment-method")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("payment-method")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("payment-method")
			updateData := helper.GetUpdateDataForEntity("payment-method")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListPaymentMethods", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test PaymentProfile create/read/list operations
	t.Run("PaymentProfileOperations", func(t *testing.T) {
		entityPath := "/api/payment/payment-profile"

		t.Run("CreatePaymentProfile", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("payment-profile")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("payment-profile")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("payment-profile")
			updateData := helper.GetUpdateDataForEntity("payment-profile")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListPaymentProfiles", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})
}
