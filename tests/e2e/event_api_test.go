//go:build mock_auth
package e2e

import (
	"testing"

	"github.com/erniealice/espyna-golang/tests/e2e/helper"
)

// TestEventDomainCRUDOperations validates create, read, and list operations for Event domain
func TestEventDomainCRUDOperations(t *testing.T) {
	env := helper.SetupTestEnvironment(t)

	// Test Event create/read/list operations
	t.Run("EventOperations", func(t *testing.T) {
		entityPath := "/api/event/event"

		t.Run("CreateEvent", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("event")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("event")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("event")
			updateData := helper.GetUpdateDataForEntity("event")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListEvents", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})
}
