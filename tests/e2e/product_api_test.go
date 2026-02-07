//go:build mock_auth
package e2e

import (
	"testing"

	"leapfor.xyz/espyna/tests/e2e/helper"
)

// TestProductDomainCRUDOperations validates create, read, and list operations for Product domain
func TestProductDomainCRUDOperations(t *testing.T) {
	env := helper.SetupTestEnvironment(t)

	// Test Product create/read/list operations
	t.Run("ProductOperations", func(t *testing.T) {
		entityPath := "/api/product/product"

		t.Run("CreateProduct", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product")
			updateData := helper.GetUpdateDataForEntity("product")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListProducts", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Collection create/read/list operations
	t.Run("CollectionOperations", func(t *testing.T) {
		entityPath := "/api/product/collection"

		t.Run("CreateCollection", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("collection")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("collection")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("collection")
			updateData := helper.GetUpdateDataForEntity("collection")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListCollections", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test CollectionPlan create/read/list operations
	t.Run("CollectionPlanOperations", func(t *testing.T) {
		entityPath := "/api/product/collection-plan"

		t.Run("CreateCollectionPlan", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("collection-plan")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("collection-plan")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("collection-plan")
			updateData := helper.GetUpdateDataForEntity("collection-plan")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListCollectionPlans", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test PriceProduct create/read/list operations
	t.Run("PriceProductOperations", func(t *testing.T) {
		entityPath := "/api/product/price-product"

		t.Run("CreatePriceProduct", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("price-product")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("price-product")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("price-product")
			updateData := helper.GetUpdateDataForEntity("price-product")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListPriceProducts", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test ProductAttribute create/read/list operations
	t.Run("ProductAttributeOperations", func(t *testing.T) {
		entityPath := "/api/product/product-attribute"

		t.Run("CreateProductAttribute", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product-attribute")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product-attribute")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product-attribute")
			updateData := helper.GetUpdateDataForEntity("product-attribute")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListProductAttributes", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test ProductCollection create/read/list operations
	t.Run("ProductCollectionOperations", func(t *testing.T) {
		entityPath := "/api/product/product-collection"

		t.Run("CreateProductCollection", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product-collection")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product-collection")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product-collection")
			updateData := helper.GetUpdateDataForEntity("product-collection")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListProductCollections", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test ProductPlan create/read/list operations
	t.Run("ProductPlanOperations", func(t *testing.T) {
		entityPath := "/api/product/product-plan"

		t.Run("CreateProductPlan", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product-plan")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product-plan")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("product-plan")
			updateData := helper.GetUpdateDataForEntity("product-plan")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListProductPlans", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})

	// Test Resource create/read/list operations
	t.Run("ResourceOperations", func(t *testing.T) {
		entityPath := "/api/product/resource"

		t.Run("CreateResource", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("resource")
			helper.TestCreateOperation(t, env, entityPath, createData)
		})

		t.Run("CreateReadFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("resource")
			helper.TestCreateReadFlow(t, env, entityPath, createData)
		})

		t.Run("CreateUpdateFlow", func(t *testing.T) {
			createData := helper.GetTestDataForEntity("resource")
			updateData := helper.GetUpdateDataForEntity("resource")
			helper.TestCreateUpdateFlow(t, env, entityPath, createData, updateData)
		})

		t.Run("ListResources", func(t *testing.T) {
			helper.TestListOperation(t, env, entityPath)
		})
	})
}
