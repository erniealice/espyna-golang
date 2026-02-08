//go:build mock_db && mock_auth

package activity_template

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	activityTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
)

func TestGetActivityTemplateListPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with some test data
	mockRepo := workflow.NewMockActivityTemplateRepository("education")

	// Create some test activity templates first
	activityTemplate1, err := mockRepo.CreateActivityTemplate(ctx, &activityTemplatepb.CreateActivityTemplateRequest{
		Data: &activityTemplatepb.ActivityTemplate{
			Name:        "Test Activity Template 1",
			Description: &[]string{"Assignment activity for elementary school"}[0],
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test activity template 1: %v", err)
	}

	_, err = mockRepo.CreateActivityTemplate(ctx, &activityTemplatepb.CreateActivityTemplateRequest{
		Data: &activityTemplatepb.ActivityTemplate{
			Name:        "Test Activity Template 2",
			Description: &[]string{"Quiz activity for high school"}[0],
			Active:      false,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test activity template 2: %v", err)
	}

	// Setup repositories and services
	repos := GetActivityTemplateListPageDataRepositories{
		ActivityTemplate: mockRepo,
	}
	services := GetActivityTemplateListPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetActivityTemplateListPageDataUseCase(repos, services)

	// Test case 1: Basic list without filters
	t.Run("BasicList", func(t *testing.T) {
		req := &activityTemplatepb.GetActivityTemplateListPageDataRequest{
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

		if len(resp.ActivityTemplateList) < 2 {
			t.Errorf("Expected at least 2 activity templates, got %d", len(resp.ActivityTemplateList))
		}

		if resp.Pagination == nil {
			t.Error("Expected pagination response")
		}
	})

	// Test case 2: Filtering by active status
	t.Run("FilterByActive", func(t *testing.T) {
		req := &activityTemplatepb.GetActivityTemplateListPageDataRequest{
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

		// Should only get active activity templates
		for _, at := range resp.ActivityTemplateList {
			if !at.Active {
				t.Errorf("Expected only active activity templates, found inactive: %s", at.Id)
			}
		}
	})

	// Test case 3: Search by name
	t.Run("SearchByName", func(t *testing.T) {
		req := &activityTemplatepb.GetActivityTemplateListPageDataRequest{
			Pagination: &commonpb.PaginationRequest{
				Limit: 10,
				Method: &commonpb.PaginationRequest_Offset{
					Offset: &commonpb.OffsetPagination{Page: 1},
				},
			},
			Search: &commonpb.SearchRequest{
				Query: "Test Activity Template 1",
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

		// Should find at least the first activity template
		found := false
		for _, at := range resp.ActivityTemplateList {
			if at.Id == activityTemplate1.Data[0].Id {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find activityTemplate1 in search results")
		}
	})

	// Test case 4: Search by description
	t.Run("SearchByDescription", func(t *testing.T) {
		req := &activityTemplatepb.GetActivityTemplateListPageDataRequest{
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

		// Should find the activity template with "elementary" in description
		found := false
		for _, at := range resp.ActivityTemplateList {
			if at.Id == activityTemplate1.Data[0].Id {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find activityTemplate1 in search results for 'elementary'")
		}
	})

	// Test case 5: Sorting by name
	t.Run("SortByName", func(t *testing.T) {
		req := &activityTemplatepb.GetActivityTemplateListPageDataRequest{
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
		if len(resp.ActivityTemplateList) >= 2 {
			for i := 1; i < len(resp.ActivityTemplateList); i++ {
				if resp.ActivityTemplateList[i-1].Name > resp.ActivityTemplateList[i].Name {
					t.Errorf("Activity templates not sorted properly: %s > %s",
						resp.ActivityTemplateList[i-1].Name, resp.ActivityTemplateList[i].Name)
				}
			}
		}
	})

	// Test case 6: Pagination
	t.Run("Pagination", func(t *testing.T) {
		req := &activityTemplatepb.GetActivityTemplateListPageDataRequest{
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

		if len(resp.ActivityTemplateList) != 1 {
			t.Errorf("Expected exactly 1 activity template with limit=1, got %d", len(resp.ActivityTemplateList))
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

func TestGetActivityTemplateListPageDataUseCase_Execute_EmptyList(t *testing.T) {
	// Setup with empty repository
	ctx := context.Background()
	mockRepo := workflow.NewMockActivityTemplateRepository("education")

	repos := GetActivityTemplateListPageDataRepositories{
		ActivityTemplate: mockRepo,
	}
	services := GetActivityTemplateListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetActivityTemplateListPageDataUseCase(repos, services)

	req := &activityTemplatepb.GetActivityTemplateListPageDataRequest{
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

	if len(resp.ActivityTemplateList) != 0 {
		t.Errorf("Expected 0 activity templates for empty list, got %d", len(resp.ActivityTemplateList))
	}

	if resp.Pagination == nil {
		t.Error("Expected pagination response")
	} else if resp.Pagination.TotalItems != 0 {
		t.Errorf("Expected total items = 0, got %d", resp.Pagination.TotalItems)
	}
}

func TestGetActivityTemplateListPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	mockRepo := workflow.NewMockActivityTemplateRepository("education")

	repos := GetActivityTemplateListPageDataRepositories{
		ActivityTemplate: mockRepo,
	}
	services := GetActivityTemplateListPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetActivityTemplateListPageDataUseCase(repos, services)

	// Test nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
	})

	// Test invalid pagination limit
	t.Run("InvalidPaginationLimit", func(t *testing.T) {
		req := &activityTemplatepb.GetActivityTemplateListPageDataRequest{
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
		req := &activityTemplatepb.GetActivityTemplateListPageDataRequest{
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
}
