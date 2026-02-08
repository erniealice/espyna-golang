//go:build mock_db && mock_auth

// Package workflow provides table-driven tests for the workflow list page data use case.
//
// The tests cover various scenarios including basic listing, filtering, searching,
// sorting, pagination, and validation errors. Each test case validates the
// workflow listing functionality with comprehensive assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestGetWorkflowListPageDataUseCase_Execute
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-SUCCESS-v1.0: Basic list success
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-FILTER-ACTIVE-v1.0: Filter by active status
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-FILTER-STATUS-v1.0: Filter by status
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-FILTER-WORKSPACE-ID-v1.0: Filter by workspace ID
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-SEARCH-NAME-v1.0: Search by name
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-SEARCH-DESCRIPTION-v1.0: Search by description
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-SORT-NAME-v1.0: Sort by name
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-SORT-DATE-CREATED-v1.0: Sort by date created
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-SORT-STATUS-v1.0: Sort by status
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-PAGINATION-v1.0: Pagination test
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-EMPTY-LIST-v1.0: Empty list test
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-NIL-REQUEST-v1.0: Nil request validation
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-INVALID-PAGINATION-v1.0: Invalid pagination
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-INVALID-FILTER-v1.0: Invalid filter field
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-LIST-PAGE-DATA-INVALID-SORT-v1.0: Invalid sort field
//
// Data Sources:
//   - Mock data: packages/copya/data/{businessType}/workflow.json
//   - Workspace data: packages/copya/data/{businessType}/workspace.json
package workflow

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workflowpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow"
)

func TestGetWorkflowListPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with some test data
	mockRepo := workflow.NewMockWorkflowRepository("education")

	// Create some test workflows first
	workflow1, err := mockRepo.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
		Data: &workflowpb.Workflow{
			Name:        "Test Workflow 1",
			Description: stringPtrWorkflow("Educational workflow for elementary school"),
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow 1: %v", err)
	}

	_, err = mockRepo.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
		Data: &workflowpb.Workflow{
			Name:        "Test Workflow 2",
			Description: stringPtrWorkflow("Educational workflow for high school"),
			Active:      false,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow 2: %v", err)
	}

	// Setup repositories and services
	repos := GetWorkflowListPageDataRepositories{
		Workflow: mockRepo,
	}
	services := GetWorkflowListPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetWorkflowListPageDataUseCase(repos, services)

	// Test case 1: Basic list without filters
	t.Run("BasicList", func(t *testing.T) {
		req := &workflowpb.GetWorkflowListPageDataRequest{
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

		if len(resp.WorkflowList) < 2 {
			t.Errorf("Expected at least 2 workflows, got %d", len(resp.WorkflowList))
		}

		if resp.Pagination == nil {
			t.Error("Expected pagination response")
		}
	})

	// Test case 2: Filtering by active status
	t.Run("FilterByActive", func(t *testing.T) {
		req := &workflowpb.GetWorkflowListPageDataRequest{
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

		// Should only get active workflows
		for _, wf := range resp.WorkflowList {
			if !wf.Active {
				t.Errorf("Expected only active workflows, found inactive: %s", wf.Id)
			}
		}
	})

	// Test case 3: Filtering by status
	t.Run("FilterByStatus", func(t *testing.T) {
		req := &workflowpb.GetWorkflowListPageDataRequest{
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
						Field: "status",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{},
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
	})

	// Test case 4: Filtering by workspace ID
	t.Run("FilterByWorkspaceId", func(t *testing.T) {
		testWorkspaceId := "test-workspace-123"

		req := &workflowpb.GetWorkflowListPageDataRequest{
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
						Field: "workspaceId",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Value: testWorkspaceId,
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

		// Should only get workflows from specified workspace
		for _, wf := range resp.WorkflowList {
			if wf.WorkspaceId == nil || *wf.WorkspaceId != testWorkspaceId {
				t.Errorf("Expected workflow from workspace %s, got from %v", testWorkspaceId, wf.WorkspaceId)
			}
		}
	})

	// Test case 5: Search by name
	t.Run("SearchByName", func(t *testing.T) {
		req := &workflowpb.GetWorkflowListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Search: &commonpb.SearchRequest{
				Query: "Test Workflow 1",
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

		// Should find at least the first workflow
		found := false
		for _, wf := range resp.WorkflowList {
			if wf.Id == workflow1.Data[0].Id {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find workflow1 in search results")
		}
	})

	// Test case 6: Search by description
	t.Run("SearchByDescription", func(t *testing.T) {
		req := &workflowpb.GetWorkflowListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Search: &commonpb.SearchRequest{
				Query: "elementary",
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

		// Should find the workflow with "elementary" in description
		found := false
		for _, wf := range resp.WorkflowList {
			if wf.Id == workflow1.Data[0].Id {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find workflow1 in search results for 'elementary'")
		}
	})

	// Test case 7: Sorting by name
	t.Run("SortByName", func(t *testing.T) {
		req := &workflowpb.GetWorkflowListPageDataRequest{
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
		if len(resp.WorkflowList) >= 2 {
			for i := 1; i < len(resp.WorkflowList); i++ {
				if resp.WorkflowList[i-1].Name > resp.WorkflowList[i].Name {
					t.Errorf("Workflows not sorted properly by name: %s > %s",
						resp.WorkflowList[i-1].Name, resp.WorkflowList[i].Name)
				}
			}
		}
	})

	// Test case 8: Sorting by date created
	t.Run("SortByDateCreated", func(t *testing.T) {
		req := &workflowpb.GetWorkflowListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Sort: &commonpb.SortRequest{
				Fields: []*commonpb.SortField{
					{
						Field:     "dateCreated",
						Direction: commonpb.SortDirection_DESC,
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

		// Verify sorting - date created should be in descending order
		if len(resp.WorkflowList) >= 2 {
			for i := 1; i < len(resp.WorkflowList); i++ {
				prevCreated := resp.WorkflowList[i-1].DateCreated
				currCreated := resp.WorkflowList[i].DateCreated

				if prevCreated != nil && currCreated != nil && *prevCreated < *currCreated {
					t.Errorf("Workflows not sorted properly by date created (DESC): %d < %d",
						*prevCreated, *currCreated)
				}
			}
		}
	})

	// Test case 9: Sorting by status
	t.Run("SortByStatus", func(t *testing.T) {
		req := &workflowpb.GetWorkflowListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Sort: &commonpb.SortRequest{
				Fields: []*commonpb.SortField{
					{
						Field:     "status",
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

		// Verify sorting - status should be in ascending order
		if len(resp.WorkflowList) >= 2 {
			for i := 1; i < len(resp.WorkflowList); i++ {
				prevStatus := resp.WorkflowList[i-1].Status
				currStatus := resp.WorkflowList[i].Status

				if prevStatus > currStatus {
					t.Errorf("Workflows not sorted properly by status (ASC): %v > %v",
						prevStatus, currStatus)
				}
			}
		}
	})

	// Test case 10: Pagination
	t.Run("Pagination", func(t *testing.T) {
		req := &workflowpb.GetWorkflowListPageDataRequest{
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

		if len(resp.WorkflowList) != 1 {
			t.Errorf("Expected exactly 1 workflow with limit=1, got %d", len(resp.WorkflowList))
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

func TestGetWorkflowListPageDataUseCase_Execute_EmptyList(t *testing.T) {
	// Setup with empty repository
	ctx := context.Background()
	mockRepo := workflow.NewMockWorkflowRepository("education")

	repos := GetWorkflowListPageDataRepositories{
		Workflow: mockRepo,
	}
	services := GetWorkflowListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetWorkflowListPageDataUseCase(repos, services)

	req := &workflowpb.GetWorkflowListPageDataRequest{
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

	if len(resp.WorkflowList) != 0 {
		t.Errorf("Expected 0 workflows for empty list, got %d", len(resp.WorkflowList))
	}

	if resp.Pagination == nil {
		t.Error("Expected pagination response")
	} else if resp.Pagination.TotalItems != 0 {
		t.Errorf("Expected total items = 0, got %d", resp.Pagination.TotalItems)
	}
}

func TestGetWorkflowListPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	mockRepo := workflow.NewMockWorkflowRepository("education")

	repos := GetWorkflowListPageDataRepositories{
		Workflow: mockRepo,
	}
	services := GetWorkflowListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetWorkflowListPageDataUseCase(repos, services)

	// Test nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
	})

	// Test invalid pagination limit
	t.Run("InvalidPaginationLimit", func(t *testing.T) {
		req := &workflowpb.GetWorkflowListPageDataRequest{
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
		req := &workflowpb.GetWorkflowListPageDataRequest{
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
		req := &workflowpb.GetWorkflowListPageDataRequest{
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
func stringPtrWorkflow(s string) *string {
	return &s
}
