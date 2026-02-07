//go:build mock_db && mock_auth

// Package workflow provides table-driven tests for the workflow item page data use case.
//
// The tests cover various scenarios including successful retrieval, not found cases,
// validation errors, and inactive workflows. Each test case validates the
// workflow retrieval functionality with comprehensive assertions.
//
// Environment Variables:
//   - TEST_USER_ID: Sets user ID for test contexts (default: "test-user").
//   - TEST_BUSINESS_TYPE: Sets business type for test contexts (default: "education").
//
// Usage:
//   - Run all tests: go test -tags="mock_db,mock_auth" ./...
//   - Run specific tests: go test -tags="mock_db,mock_auth" -run TestGetWorkflowItemPageDataUseCase_Execute
//   - Set environment variables: TEST_USER_ID="admin" TEST_BUSINESS_TYPE="fitness_center" go test
//
// Test Codes:
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-SUCCESS-v1.0: Successful retrieval
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-NOT-FOUND-v1.0: Not found case
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-NIL-REQUEST-v1.0: Nil request validation
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-EMPTY-ID-v1.0: Empty ID validation
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-SHORT-ID-v1.0: Short ID validation
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-LONG-ID-v1.0: Long ID validation
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-INVALID-CHARS-ID-v1.0: Invalid characters in ID
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-INACTIVE-v1.0: Inactive workflow
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-WORKSPACE-RELATIONSHIP-v1.0: Workspace relationship validation
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-DRAFT-STATUS-v1.0: Draft workflow status
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-ARCHIVED-STATUS-v1.0: Archived workflow status
//   - ESPYNA-TEST-WORKFLOW-WORKFLOW-ITEM-PAGE-DATA-WITH-VERSION-v1.0: Workflow with version
//
// Data Sources:
//   - Mock data: packages/copya/data/{businessType}/workflow.json
//   - Workspace data: packages/copya/data/{businessType}/workspace.json
package workflow

import (
	"context"
	"testing"

	"leapfor.xyz/espyna/internal/infrastructure/adapters/secondary/database/mock/workflow"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow"
)

func TestGetWorkflowItemPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock repository with test data
	mockRepo := workflow.NewMockWorkflowRepository("education")

	// Create a test workflow first
	createResp, err := mockRepo.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
		Data: &workflowpb.Workflow{
			Name:        "Test Workflow",
			Description: stringPtrItem("A comprehensive educational workflow for testing"),
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	workflowId := createResp.Data[0].Id

	// Setup repositories and services
	repos := GetWorkflowItemPageDataRepositories{
		Workflow: mockRepo,
	}
	services := GetWorkflowItemPageDataServices{
		TransactionService: nil, // No transaction for this test
		TranslationService: nil, // No translation for this test
	}

	// Create use case
	useCase := NewGetWorkflowItemPageDataUseCase(repos, services)

	// Test successful retrieval
	req := &workflowpb.GetWorkflowItemPageDataRequest{
		WorkflowId: workflowId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.Workflow == nil {
		t.Fatal("Expected workflow data")
	}

	if resp.Workflow.Id != workflowId {
		t.Errorf("Expected workflow ID %s, got %s", workflowId, resp.Workflow.Id)
	}

	if resp.Workflow.Name != "Test Workflow" {
		t.Errorf("Expected workflow name 'Test Workflow', got %s", resp.Workflow.Name)
	}

	if resp.Workflow.Description == nil || *resp.Workflow.Description != "A comprehensive educational workflow for testing" {
		t.Errorf("Expected workflow description to match, got %v", resp.Workflow.Description)
	}

	if !resp.Workflow.Active {
		t.Error("Expected workflow to be active")
	}
}

func TestGetWorkflowItemPageDataUseCase_Execute_NotFound(t *testing.T) {
	// Setup with empty repository
	ctx := context.Background()
	mockRepo := workflow.NewMockWorkflowRepository("education")

	repos := GetWorkflowItemPageDataRepositories{
		Workflow: mockRepo,
	}
	services := GetWorkflowItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetWorkflowItemPageDataUseCase(repos, services)

	// Test with non-existent workflow ID
	req := &workflowpb.GetWorkflowItemPageDataRequest{
		WorkflowId: "non-existent-id",
	}

	_, err := useCase.Execute(ctx, req)
	if err == nil {
		t.Error("Expected error for non-existent workflow")
	}
}

func TestGetWorkflowItemPageDataUseCase_Execute_ValidationErrors(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := workflow.NewMockWorkflowRepository("education")

	repos := GetWorkflowItemPageDataRepositories{
		Workflow: mockRepo,
	}
	services := GetWorkflowItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetWorkflowItemPageDataUseCase(repos, services)

	// Test case 1: Nil request
	t.Run("NilRequest", func(t *testing.T) {
		_, err := useCase.Execute(ctx, nil)
		if err == nil {
			t.Error("Expected error for nil request")
		}
	})

	// Test case 2: Empty workflow ID
	t.Run("EmptyWorkflowId", func(t *testing.T) {
		req := &workflowpb.GetWorkflowItemPageDataRequest{
			WorkflowId: "",
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for empty workflow ID")
		}
	})

	// Test case 3: Short workflow ID
	t.Run("ShortWorkflowId", func(t *testing.T) {
		req := &workflowpb.GetWorkflowItemPageDataRequest{
			WorkflowId: "ab", // Too short
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for too short workflow ID")
		}
	})

	// Test case 4: Long workflow ID
	t.Run("LongWorkflowId", func(t *testing.T) {
		longId := ""
		for i := 0; i < 101; i++ {
			longId += "a"
		}

		req := &workflowpb.GetWorkflowItemPageDataRequest{
			WorkflowId: longId, // Too long
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for too long workflow ID")
		}
	})

	// Test case 5: Invalid characters in workflow ID
	t.Run("InvalidCharsWorkflowId", func(t *testing.T) {
		req := &workflowpb.GetWorkflowItemPageDataRequest{
			WorkflowId: "workflow@123#invalid", // Invalid characters
		}

		_, err := useCase.Execute(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid characters in workflow ID")
		}
	})
}

func TestGetWorkflowItemPageDataUseCase_Execute_InactiveWorkflow(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := workflow.NewMockWorkflowRepository("education")

	// Create an inactive workflow
	createResp, err := mockRepo.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
		Data: &workflowpb.Workflow{
			Name:        "Inactive Workflow",
			Description: stringPtrItem("A workflow that has been disabled"),
			Active:      false,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	workflowId := createResp.Data[0].Id

	repos := GetWorkflowItemPageDataRepositories{
		Workflow: mockRepo,
	}
	services := GetWorkflowItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetWorkflowItemPageDataUseCase(repos, services)

	// Test retrieval of inactive workflow (should still work)
	req := &workflowpb.GetWorkflowItemPageDataRequest{
		WorkflowId: workflowId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.Workflow == nil {
		t.Fatal("Expected workflow data")
	}

	if resp.Workflow.Active {
		t.Error("Expected workflow to be inactive")
	}

}

func TestGetWorkflowItemPageDataUseCase_Execute_WorkflowStatuses(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := workflow.NewMockWorkflowRepository("education")

	// Test case 1: Draft workflow
	t.Run("DraftStatus", func(t *testing.T) {
		createResp, err := mockRepo.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
			Data: &workflowpb.Workflow{
				Name:        "Draft Workflow",
				Description: stringPtrItem("A workflow in draft status"),
				Active:      true,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create test workflow: %v", err)
		}

		workflowId := createResp.Data[0].Id

		repos := GetWorkflowItemPageDataRepositories{
			Workflow: mockRepo,
		}
		services := GetWorkflowItemPageDataServices{
			TransactionService: nil,
			TranslationService: nil,
		}

		useCase := NewGetWorkflowItemPageDataUseCase(repos, services)

		req := &workflowpb.GetWorkflowItemPageDataRequest{
			WorkflowId: workflowId,
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}
	})

	// Test case 2: Archived workflow
	t.Run("ArchivedStatus", func(t *testing.T) {
		createResp, err := mockRepo.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
			Data: &workflowpb.Workflow{
				Name:        "Archived Workflow",
				Description: stringPtrItem("A workflow that has been archived"),
				Active:      false,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create test workflow: %v", err)
		}

		workflowId := createResp.Data[0].Id

		repos := GetWorkflowItemPageDataRepositories{
			Workflow: mockRepo,
		}
		services := GetWorkflowItemPageDataServices{
			TransactionService: nil,
			TranslationService: nil,
		}

		useCase := NewGetWorkflowItemPageDataUseCase(repos, services)

		req := &workflowpb.GetWorkflowItemPageDataRequest{
			WorkflowId: workflowId,
		}

		resp, err := useCase.Execute(ctx, req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}

		if resp.Workflow.Active {
			t.Error("Expected archived workflow to be inactive")
		}
	})
}

func TestGetWorkflowItemPageDataUseCase_Execute_WorkflowWithVersion(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := workflow.NewMockWorkflowRepository("education")

	// Create a workflow with version
	version := int32(3)
	createResp, err := mockRepo.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
		Data: &workflowpb.Workflow{
			Name:        "Versioned Workflow",
			Description: stringPtrItem("A workflow with version control"),
			Version:     &version,
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	workflowId := createResp.Data[0].Id

	repos := GetWorkflowItemPageDataRepositories{
		Workflow: mockRepo,
	}
	services := GetWorkflowItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetWorkflowItemPageDataUseCase(repos, services)

	// Test retrieval with version
	req := &workflowpb.GetWorkflowItemPageDataRequest{
		WorkflowId: workflowId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.Workflow == nil {
		t.Fatal("Expected workflow data")
	}

	if resp.Workflow.Version == nil || *resp.Workflow.Version != 3 {
		t.Errorf("Expected workflow version 3, got %v", resp.Workflow.Version)
	}
}

func TestGetWorkflowItemPageDataUseCase_Execute_WorkflowWithWorkspace(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := workflow.NewMockWorkflowRepository("education")

	// Create a workflow with workspace ID
	workspaceId := "test-workspace-123"
	createResp, err := mockRepo.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
		Data: &workflowpb.Workflow{
			Name:        "Workspace Workflow",
			Description: stringPtrItem("A workflow associated with a workspace"),
			WorkspaceId: &workspaceId,
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	workflowId := createResp.Data[0].Id

	repos := GetWorkflowItemPageDataRepositories{
		Workflow: mockRepo,
	}
	services := GetWorkflowItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetWorkflowItemPageDataUseCase(repos, services)

	// Test retrieval with workspace relationship
	req := &workflowpb.GetWorkflowItemPageDataRequest{
		WorkflowId: workflowId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.Workflow == nil {
		t.Fatal("Expected workflow data")
	}

	if resp.Workflow.WorkspaceId == nil || *resp.Workflow.WorkspaceId != workspaceId {
		t.Errorf("Expected workspace ID %s, got %v", workspaceId, resp.Workflow.WorkspaceId)
	}
}

func TestGetWorkflowItemPageDataUseCase_Execute_CompleteWorkflowData(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockRepo := workflow.NewMockWorkflowRepository("education")

	// Create a comprehensive workflow with all properties
	version := int32(5)
	workspaceId := "comprehensive-workspace-456"
	createResp, err := mockRepo.CreateWorkflow(ctx, &workflowpb.CreateWorkflowRequest{
		Data: &workflowpb.Workflow{
			Name:        "Comprehensive Workflow",
			Description: stringPtrItem("A complete workflow with all properties for testing"),
			Version:     &version,
			WorkspaceId: &workspaceId,
			Active:      true,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	workflowId := createResp.Data[0].Id

	repos := GetWorkflowItemPageDataRepositories{
		Workflow: mockRepo,
	}
	services := GetWorkflowItemPageDataServices{
		TransactionService: nil,
		TranslationService: nil,
	}

	useCase := NewGetWorkflowItemPageDataUseCase(repos, services)

	// Test comprehensive retrieval
	req := &workflowpb.GetWorkflowItemPageDataRequest{
		WorkflowId: workflowId,
	}

	resp, err := useCase.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success=true")
	}

	if resp.Workflow == nil {
		t.Fatal("Expected workflow data")
	}

	// Verify all workflow properties
	if resp.Workflow.Id != workflowId {
		t.Errorf("Expected workflow ID %s, got %s", workflowId, resp.Workflow.Id)
	}

	if resp.Workflow.Name != "Comprehensive Workflow" {
		t.Errorf("Expected workflow name 'Comprehensive Workflow', got %s", resp.Workflow.Name)
	}

	if resp.Workflow.Description == nil || *resp.Workflow.Description != "A complete workflow with all properties for testing" {
		t.Errorf("Expected workflow description to match, got %v", resp.Workflow.Description)
	}

	if resp.Workflow.Version == nil || *resp.Workflow.Version != 5 {
		t.Errorf("Expected workflow version 5, got %v", resp.Workflow.Version)
	}

	if resp.Workflow.WorkspaceId == nil || *resp.Workflow.WorkspaceId != workspaceId {
		t.Errorf("Expected workspace ID %s, got %v", workspaceId, resp.Workflow.WorkspaceId)
	}

	if !resp.Workflow.Active {
		t.Error("Expected workflow to be active")
	}

	// Verify audit fields
	if resp.Workflow.DateCreated == nil {
		t.Error("Expected DateCreated to be set")
	}

	if resp.Workflow.DateCreatedString == nil {
		t.Error("Expected DateCreatedString to be set")
	}

	if resp.Workflow.DateModified == nil {
		t.Error("Expected DateModified to be set")
	}

	if resp.Workflow.DateModifiedString == nil {
		t.Error("Expected DateModifiedString to be set")
	}
}

// Helper function to create string pointers
func stringPtrItem(s string) *string {
	return &s
}
