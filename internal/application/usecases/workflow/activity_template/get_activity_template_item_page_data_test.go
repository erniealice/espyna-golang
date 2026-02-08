//go:build mock_db && mock_auth

package activity_template

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	activityTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
)

func TestGetActivityTemplateItemPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with test data
	mockRepo := workflow.NewMockActivityTemplateRepository("education")

	// Create a test activity template first
	createResp, err := mockRepo.CreateActivityTemplate(ctx, &activityTemplatepb.CreateActivityTemplateRequest{
		Data: &activityTemplatepb.ActivityTemplate{
			Name:        "Test Activity Template",
			Description: stringPtr("A comprehensive assignment activity for educational institutions"),
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test activity template: %v", err)
	}

	activityTemplateId := createResp.Data[0].Id

	// Setup repositories and services
	repos := GetActivityTemplateItemPageDataRepositories{
		ActivityTemplate: mockRepo,
	}
	services := GetActivityTemplateItemPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetActivityTemplateItemPageDataUseCase(repos, services)

	// Test successful retrieval
	req := &activityTemplatepb.GetActivityTemplateItemPageDataRequest{
		ActivityTemplateId: activityTemplateId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.ActivityTemplate == nil {
		t.Fatal("Expected activity template data")
	}

	if resp.ActivityTemplate.Id != activityTemplateId {
		t.Errorf("Expected activity template ID %s, got %s", activityTemplateId, resp.ActivityTemplate.Id)
	}

	if resp.ActivityTemplate.Name != "Test Activity Template" {
		t.Errorf("Expected activity template name 'Test Activity Template', got %s", resp.ActivityTemplate.Name)
	}

	if resp.ActivityTemplate.Description == nil || *resp.ActivityTemplate.Description != "A comprehensive assignment activity for educational institutions" {
		t.Errorf("Expected activity template description to match, got %v", resp.ActivityTemplate.Description)
	}

	if !resp.ActivityTemplate.Active {
		t.Error("Expected activity template to be active")
	}
}

func TestGetActivityTemplateItemPageDataUseCase_Execute_NotFound(t *testing.T) {
	// Setup with empty repository
	ctx := context.Background()
	mockRepo := workflow.NewMockActivityTemplateRepository("education")

	repos := GetActivityTemplateItemPageDataRepositories{
		ActivityTemplate: mockRepo,
	}
	services := GetActivityTemplateItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetActivityTemplateItemPageDataUseCase(repos, services)

	// Test with non-existent activity template ID
	req := &activityTemplatepb.GetActivityTemplateItemPageDataRequest{
		ActivityTemplateId: "non-existent-id",
	}

	_, err := useCase.Execute(ctx, req)
	if err == nil {
		t.Error("Expected error for non-existent activity template")
	}
}

func TestGetActivityTemplateItemPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := workflow.NewMockActivityTemplateRepository("education")

	repos := GetActivityTemplateItemPageDataRepositories{
		ActivityTemplate: mockRepo,
	}
	services := GetActivityTemplateItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetActivityTemplateItemPageDataUseCase(repos, services)

	// Test case 1: Nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
	})

	// Test case 2: Empty activity template ID
	t.Run("EmptyActivityTemplateId", func(t *testing.T) {
		req := &activityTemplatepb.GetActivityTemplateItemPageDataRequest{
			ActivityTemplateId: "",
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for empty activity template ID")
		}
	})

	// Test case 3: Short activity template ID
	t.Run("ShortActivityTemplateId", func(t *testing.T) {
		req := &activityTemplatepb.GetActivityTemplateItemPageDataRequest{
			ActivityTemplateId: "ab", // Too short
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for too short activity template ID")
		}
	})
}

func TestGetActivityTemplateItemPageDataUseCase_Execute_InactiveActivityTemplate(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := workflow.NewMockActivityTemplateRepository("education")

	// Create an inactive activity template
	createResp, err := mockRepo.CreateActivityTemplate(ctx, &activityTemplatepb.CreateActivityTemplateRequest{
		Data: &activityTemplatepb.ActivityTemplate{
			Name:        "Inactive Activity Template",
			Description: stringPtr("A activity template that has been disabled"),
			Active:      false,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test activity template: %v", err)
	}

	activityTemplateId := createResp.Data[0].Id

	repos := GetActivityTemplateItemPageDataRepositories{
		ActivityTemplate: mockRepo,
	}
	services := GetActivityTemplateItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetActivityTemplateItemPageDataUseCase(repos, services)

	// Test retrieval of inactive activity template (should still work)
	req := &activityTemplatepb.GetActivityTemplateItemPageDataRequest{
		ActivityTemplateId: activityTemplateId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.ActivityTemplate == nil {
		t.Fatal("Expected activity template data")
	}

	if resp.ActivityTemplate.Active {
		t.Error("Expected activity template to be inactive")
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
