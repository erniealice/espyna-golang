//go:build mock_db && mock_auth

// Package stage_template provides table-driven tests for the stage template item page data use case.
//
// The tests cover various scenarios including successful retrieval, not found cases,
// validation errors, and inactive stage templates. Each test case validates the
// stage template retrieval functionality with comprehensive assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestGetStageTemplateItemPageDataUseCase_Execute
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-ITEM-PAGE-DATA-SUCCESS-v1.0: Successful retrieval
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-ITEM-PAGE-DATA-NOT-FOUND-v1.0: Not found case
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-ITEM-PAGE-DATA-NIL-REQUEST-v1.0: Nil request validation
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-ITEM-PAGE-DATA-EMPTY-ID-v1.0: Empty ID validation
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-ITEM-PAGE-DATA-SHORT-ID-v1.0: Short ID validation
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-ITEM-PAGE-DATA-INACTIVE-v1.0: Inactive stage template
//   - ESPYNA-TEST-WORKFLOW-STAGE-TEMPLATE-ITEM-PAGE-DATA-WORKFLOW-RELATIONSHIP-v1.0: Workflow relationship validation
//
// Data Sources:
//   - Mock data: packages/copya/data/{businessType}/stage_template.json
//   - Workflow data: packages/copya/data/{businessType}/workflow.json
package stage_template

import (
	"context"
	"testing"

	"github.com/erniealice/espyna-golang/internal/infrastructure/adapters/secondary/database/mock/workflow"
	stageTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
	workflowTemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

func TestGetStageTemplateItemPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repositories
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository("education")
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository("education")

	// Create a test workflow first for foreign key relationship
	workflowResp, err := mockWorkflowTemplateRepo.CreateWorkflowTemplate(ctx, &workflowTemplatepb.CreateWorkflowTemplateRequest{
		Data: &workflowTemplatepb.WorkflowTemplate{
			Name:        "Test Workflow",
			Description: stringPtrItem("Educational workflow for testing stage templates"),
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	workflowId := workflowResp.Data[0].Id

	// Create a test stage template
	createResp, err := mockStageTemplateRepo.CreateStageTemplate(ctx, &stageTemplatepb.CreateStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Name:               "Test Stage Template",
			Description:        stringPtrItem("Initial stage template for workflow testing"),
			WorkflowTemplateId: workflowId,
			Active:             true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test stage template: %v", err)
	}

	stageTemplateId := createResp.Data[0].Id

	// Setup repositories and services
	repos := GetStageTemplateItemPageDataRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo, // For foreign key validation
	}
	services := GetStageTemplateItemPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetStageTemplateItemPageDataUseCase(repos, services)

	// Test successful retrieval
	req := &stageTemplatepb.GetStageTemplateItemPageDataRequest{
		StageTemplateId: stageTemplateId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.StageTemplate == nil {
		t.Fatal("Expected stage template data")
	}

	if resp.StageTemplate.Id != stageTemplateId {
		t.Errorf("Expected stage template ID %s, got %s", stageTemplateId, resp.StageTemplate.Id)
	}

	if resp.StageTemplate.Name != "Test Stage Template" {
		t.Errorf("Expected stage template name 'Test Stage Template', got %s", resp.StageTemplate.Name)
	}

	if resp.StageTemplate.Description == nil || *resp.StageTemplate.Description != "Initial stage template for workflow testing" {
		t.Errorf("Expected stage template description to match, got %v", resp.StageTemplate.Description)
	}

	if resp.StageTemplate.WorkflowTemplateId != workflowId {
		t.Errorf("Expected workflow ID %s, got %s", workflowId, resp.StageTemplate.WorkflowTemplateId)
	}

	if !resp.StageTemplate.Active {
		t.Error("Expected stage template to be active")
	}

	// Verify workflow relationship data is included in the stage template
	if resp.StageTemplate.WorkflowTemplateId != workflowId {
		t.Errorf("Expected workflow ID %s in stage template, got %s", workflowId, resp.StageTemplate.WorkflowTemplateId)
	}
}

func TestGetStageTemplateItemPageDataUseCase_Execute_NotFound(t *testing.T) {
	// Setup with empty repository
	ctx := context.Background()
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository("education")
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository("education")

	repos := GetStageTemplateItemPageDataRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo,
	}
	services := GetStageTemplateItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetStageTemplateItemPageDataUseCase(repos, services)

	// Test with non-existent stage template ID
	req := &stageTemplatepb.GetStageTemplateItemPageDataRequest{
		StageTemplateId: "non-existent-id",
	}

	_, err := useCase.Execute(ctx, req)
	if err == nil {
		t.Error("Expected error for non-existent stage template")
	}
}

func TestGetStageTemplateItemPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository("education")
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository("education")

	repos := GetStageTemplateItemPageDataRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo,
	}
	services := GetStageTemplateItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetStageTemplateItemPageDataUseCase(repos, services)

	// Test case 1: Nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
	})

	// Test case 2: Empty stage template ID
	t.Run("EmptyStageTemplateId", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateItemPageDataRequest{
			StageTemplateId: "",
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for empty stage template ID")
		}
	})

	// Test case 3: Short stage template ID
	t.Run("ShortStageTemplateId", func(t *testing.T) {
		req := &stageTemplatepb.GetStageTemplateItemPageDataRequest{
			StageTemplateId: "ab", // Too short
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for too short stage template ID")
		}
	})
}

func TestGetStageTemplateItemPageDataUseCase_Execute_InactiveStageTemplate(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository("education")
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository("education")

	// Create a test workflow first
	workflowResp, err := mockWorkflowTemplateRepo.CreateWorkflowTemplate(ctx, &workflowTemplatepb.CreateWorkflowTemplateRequest{
		Data: &workflowTemplatepb.WorkflowTemplate{
			Name:        "Test Workflow",
			Description: stringPtrItem("Workflow for inactive stage template test"),
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	workflowId := workflowResp.Data[0].Id

	// Create an inactive stage template
	createResp, err := mockStageTemplateRepo.CreateStageTemplate(ctx, &stageTemplatepb.CreateStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Name:               "Inactive Stage Template",
			Description:        stringPtrItem("A stage template that has been disabled"),
			WorkflowTemplateId: workflowId,
			Active:             false,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test stage template: %v", err)
	}

	stageTemplateId := createResp.Data[0].Id

	repos := GetStageTemplateItemPageDataRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo,
	}
	services := GetStageTemplateItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetStageTemplateItemPageDataUseCase(repos, services)

	// Test retrieval of inactive stage template (should still work)
	req := &stageTemplatepb.GetStageTemplateItemPageDataRequest{
		StageTemplateId: stageTemplateId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.StageTemplate == nil {
		t.Fatal("Expected stage template data")
	}

	if resp.StageTemplate.Active {
		t.Error("Expected stage template to be inactive")
	}
}

func TestGetStageTemplateItemPageDataUseCase_Execute_WorkflowRelationship(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository("education")
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository("education")

	// Create a test workflow
	workflowResp, err := mockWorkflowTemplateRepo.CreateWorkflowTemplate(ctx, &workflowTemplatepb.CreateWorkflowTemplateRequest{
		Data: &workflowTemplatepb.WorkflowTemplate{
			Name:        "Complex Workflow",
			Description: stringPtrItem("A complex workflow with multiple properties"),
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	workflowId := workflowResp.Data[0].Id

	// Create a stage template with all properties
	createResp, err := mockStageTemplateRepo.CreateStageTemplate(ctx, &stageTemplatepb.CreateStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Name:               "Complete Stage Template",
			Description:        stringPtrItem("A stage template with all properties for relationship testing"),
			WorkflowTemplateId: workflowId,
			Active:             true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test stage template: %v", err)
	}

	stageTemplateId := createResp.Data[0].Id

	repos := GetStageTemplateItemPageDataRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo,
	}
	services := GetStageTemplateItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetStageTemplateItemPageDataUseCase(repos, services)

	// Test retrieval with workflow relationship
	req := &stageTemplatepb.GetStageTemplateItemPageDataRequest{
		StageTemplateId: stageTemplateId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	// Verify stage template data
	if resp.StageTemplate == nil {
		t.Fatal("Expected stage template data")
	}

	// Verify workflow relationship data is included in the stage template
	if resp.StageTemplate.WorkflowTemplateId != workflowId {
		t.Errorf("Expected workflow ID %s in stage template, got %s", workflowId, resp.StageTemplate.WorkflowTemplateId)
	}
}

func TestGetStageTemplateItemPageDataUseCase_Execute_WorkflowNotFound(t *testing.T) {
	// Setup - stage template with non-existent workflow ID
	ctx := context.Background()
	mockStageTemplateRepo := workflow.NewMockStageTemplateRepository("education")
	mockWorkflowTemplateRepo := workflow.NewMockWorkflowTemplateRepository("education")

	// Create a stage template with a non-existent workflow ID
	createResp, err := mockStageTemplateRepo.CreateStageTemplate(ctx, &stageTemplatepb.CreateStageTemplateRequest{
		Data: &stageTemplatepb.StageTemplate{
			Name:               "Orphaned Stage Template",
			Description:        stringPtrItem("Stage template with missing workflow"),
			WorkflowTemplateId: "non-existent-workflow-id",
			Active:             true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test stage template: %v", err)
	}

	stageTemplateId := createResp.Data[0].Id

	repos := GetStageTemplateItemPageDataRepositories{
		StageTemplate:    mockStageTemplateRepo,
		WorkflowTemplate: mockWorkflowTemplateRepo,
	}
	services := GetStageTemplateItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetStageTemplateItemPageDataUseCase(repos, services)

	// Test retrieval when workflow doesn't exist - should still return stage template
	req := &stageTemplatepb.GetStageTemplateItemPageDataRequest{
		StageTemplateId: stageTemplateId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.StageTemplate == nil {
		t.Fatal("Expected stage template data")
	}

	// Stage template should exist but workflow ID should be the non-existent one
	if resp.StageTemplate.WorkflowTemplateId != "non-existent-workflow" {
		t.Errorf("Expected workflow ID 'non-existent-workflow', got %s", resp.StageTemplate.WorkflowTemplateId)
	}
}

// Helper function to create string pointers
func stringPtrItem(s string) *string {
	return &s
}
