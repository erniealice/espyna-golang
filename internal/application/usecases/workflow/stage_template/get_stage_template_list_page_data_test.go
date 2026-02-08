//go:build mock_db && mock_auth

// Package stage_template provides table-driven tests for the stage template list page data use case.
//
// The tests cover various scenarios including basic listing, filtering, searching,
// sorting, pagination, and validation errors. Each test case validates the
// stage template listing functionality with comprehensive assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestGetStageTemplateListPageDataUseCase_Execute
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-SUCCESS-v1.0: Basic list success
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-FILTER-ACTIVE-v1.0: Filter by active status
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-FILTER-WORKFLOW-ID-v1.0: Filter by workflow ID
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-SEARCH-NAME-v1.0: Search by name
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-SEARCH-DESCRIPTION-v1.0: Search by description
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-SORT-NAME-v1.0: Sort by name
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-SORT-ORDER-INDEX-v1.0: Sort by order index
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-PAGINATION-v1.0: Pagination test
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-EMPTY-LIST-v1.0: Empty list test
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-NIL-REQUEST-v1.0: Nil request validation
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-INVALID-PAGINATION-v1.0: Invalid pagination
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-LIST-PAGE-DATA-INVALID-FILTER-v1.0: Invalid filter field
//
// Data Sources:
//   - Mock data: packages/copya/data/{businessType}/stage_template.json
//   - Workflow data: packages/copya/data/{businessType}/workflow.json
package stage_template

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	stageTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
	workflowTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

func TestGetStageTemplateListPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repositories
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository("education")
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository("education")

	// Create a test workflow first for foreign key relationship
	workflow1, err := mockWorkflowTemplateRepo.CreateWorkflowTemplate(ctx, &workflowTemplatepb.CreateWorkflowTemplateRequest{
		Data: &workflowTemplatepb.WorkflowTemplate{
			Name:        "Test Workflow 1",
			Description: stringPtr("Educational workflow for testing"),
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow 1: %v", err)
	}

	workflow2, err := mockWorkflowTemplateRepo.CreateWorkflowTemplate(ctx, &workflowTemplatepb.CreateWorkflowTemplateRequest{
		Data: &workflowTemplatepb.WorkflowTemplate{
			Name:        "Test Workflow 2",
			Description: stringPtr("Testing workflow 2"),
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow 2: %v", err)
	}

	// Create some test stage templates
	stageTemplate1, err := mockStageTemplateRepo.CreateStageTemplate(ctx, &stageTemplatepb.CreateStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Name:               "Test Stage Template 1",
			Description:        stringPtr("Initial stage template"),
			WorkflowTemplateId: workflow1.Data[0].Id,
			Active:             true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test stage template 1: %v", err)
	}

	_, err = mockStageTemplateRepo.CreateStageTemplate(ctx, &stageTemplatepb.CreateStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Name:               "Test Stage Template 2",
			Description:        stringPtr("Secondary stage template"),
			WorkflowTemplateId: workflow2.Data[0].Id,
			Active:             false,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test stage template 2: %v", err)
	}

	// Setup repositories and services
	repos := GetStageTemplateListPageDataRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo, // For foreign key validation
	}
	services := GetStageTemplateListPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetStageTemplateListPageDataUseCase(repos, services)

	// Test case 1: Basic list without filters
	t.Run("BasicList", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		if len(resp.StageTemplateList) < 2 {
			t.Errorf("Expected at least 2 stage templates, got %d", len(resp.StageTemplateList))
		}

		if resp.Pagination == nil {
			t.Error("Expected pagination response")
		}
	})

	// Test case 2: Filtering by active status
	t.Run("FilterByActive", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Filters: &commonpb.FilterRequest{
				Logic: commonpb.FilterLogic_AND,
				Filters: []*commonpb.TypedFilter{
					{
						Field: "active",
						FilterType: &commonpb.TypedFilter_BooleanFilter{
							BooleanFilter: &commonpb.BooleanFilter{
								Value: true,
							},
						},
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Should only get active stage templates
		for _, st := range resp.StageTemplateList {
			if !st.Active {
				t.Errorf("Expected only active stage templates, found inactive: %s", st.Id)
			}
		}
	})

	// Test case 3: Filtering by workflow ID
	t.Run("FilterByWorkflowTemplateId", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Filters: &commonpb.FilterRequest{
				Logic: commonpb.FilterLogic_AND,
				Filters: []*commonpb.TypedFilter{
					{
						Field: "workflowId",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Value: workflow1.Data[0].Id,
							},
						},
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Should only get stage templates from workflow1
		for _, st := range resp.StageTemplateList {
			if st.WorkflowTemplateId != workflow1.Data[0].Id {
				t.Errorf("Expected stage template from workflow %s, got from %s", workflow1.Data[0].Id, st.WorkflowTemplateId)
			}
		}
	})

	// Test case 4: Search by name
	t.Run("SearchByName", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Search: &commonpb.SearchRequest{
				Query: "Test Stage Template 1",
				Options: &commonpb.SearchOptions{
					SearchFields: []string{"name"},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Should find at least the first stage template
		found := false
		for _, st := range resp.StageTemplateList {
			if st.Id == stageTemplate1.Data[0].Id {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find stageTemplate1 in search results")
		}
	})

	// Test case 5: Search by description
	t.Run("SearchByDescription", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Search: &commonpb.SearchRequest{
				Query: "Initial",
				Options: &commonpb.SearchOptions{
					SearchFields: []string{"description"},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Should find the stage template with "Initial" in description
		found := false
		for _, st := range resp.StageTemplateList {
			if st.Id == stageTemplate1.Data[0].Id {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find stageTemplate1 in search results for 'Initial'")
		}
	})

	// Test case 6: Sorting by name
	t.Run("SortByName", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Sort: &commonpb.SortRequest{
				Fields: []*commonpb.SortField{
					{
						Field:     "name",
						Direction: commonpb.SortDirection_ASC,
						NullOrder: commonpb.NullOrder_NULLS_LAST,
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Verify sorting - names should be in ascending order
		if len(resp.StageTemplateList) >= 2 {
			for i := 1; i < len(resp.StageTemplateList); i++ {
				if resp.StageTemplateList[i-1].Name > resp.StageTemplateList[i].Name {
					t.Errorf("Stage templates not sorted properly: %s > %s",
						resp.StageTemplateList[i-1].Name, resp.StageTemplateList[i].Name)
				}
			}
		}
	})

	// Test case 7: Sorting by order index
	t.Run("SortByOrderIndex", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Sort: &commonpb.SortRequest{
				Fields: []*commonpb.SortField{
					{
						Field:     "orderIndex",
						Direction: commonpb.SortDirection_ASC,
						NullOrder: commonpb.NullOrder_NULLS_LAST,
					},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		// Verify sorting - order indices should be in ascending order
		if len(resp.StageTemplateList) >= 2 {
			for i := 1; i < len(resp.StageTemplateList); i++ {
				prevOrder := resp.StageTemplateList[i-1].OrderIndex
				currOrder := resp.StageTemplateList[i].OrderIndex

				if prevOrder != nil && currOrder != nil && *prevOrder > *currOrder {
					t.Errorf("Stage templates not sorted properly by order index: %d > %d",
						*prevOrder, *currOrder)
				}
			}
		}
	})

	// Test case 8: Pagination
	t.Run("Pagination", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 1, // Only get 1 item per page
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		if len(resp.StageTemplateList) != 1 {
			t.Errorf("Expected exactly 1 stage template with limit=1, got %d", len(resp.StageTemplateList))
		}

		if resp.Pagination == nil {
			t.Error("Expected pagination response")
		} else {
			if resp.Pagination.TotalItems < 2 {
				t.Errorf("Expected total items >= 2, got %d", resp.Pagination.TotalItems)
			}
		}
	})
}

func TestGetStageTemplateListPageDataUseCase_Execute_EmptyList(t *testing.T) {
	// Setup with empty repository
	ctx := context.Background()
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository("education")
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository("education")

	repos := GetStageTemplateListPageDataRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo,
	}
	services := GetStageTemplateListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetStageTemplateListPageDataUseCase(repos, services)

	req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
		Pagination: &commonpb.PaginationRequest{
			Limit: 10,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{Page: 1},
			},
		},
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if len(resp.StageTemplateList) != 0 {
		t.Errorf("Expected 0 stage templates for empty list, got %d", len(resp.StageTemplateList))
	}

	if resp.Pagination == nil {
		t.Error("Expected pagination response")
	} else if resp.Pagination.TotalItems != 0 {
		t.Errorf("Expected total items = 0, got %d", resp.Pagination.TotalItems)
	}
}

func TestGetStageTemplateListPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository("education")
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository("education")

	repos := GetStageTemplateListPageDataRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo,
	}
	services := GetStageTemplateListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetStageTemplateListPageDataUseCase(repos, services)

	// Test nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
	})

	// Test invalid pagination limit
	t.Run("InvalidPaginationLimit", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 200, // Invalid - too high
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid pagination limit")
		}
	})

	// Test invalid filter field
	t.Run("InvalidFilterField", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Filters: &commonpb.FilterRequest{
				Logic: commonpb.FilterLogic_AND,
				Filters: []*commonpb.TypedFilter{
					{
						Field: "invalid_field", // Invalid field
						FilterType: &commonpb.TypedFilter_BooleanFilter{
							BooleanFilter: &commonpb.BooleanFilter{
								Value: true,
							},
						},
					},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid filter field")
		}
	})

	// Test invalid sort field
	t.Run("InvalidSortField", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Sort: &commonpb.SortRequest{
				Fields: []*commonpb.SortField{
					{
						Field:     "invalid_field", // Invalid field
						Direction: commonpb.SortDirection_ASC,
						NullOrder: commonpb.NullOrder_NULLS_LAST,
					},
				},
			},
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid sort field")
		}
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
