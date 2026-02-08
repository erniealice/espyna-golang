package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// MockWorkspaceServiceForItemPageData implements WorkspaceDomainServiceServer for testing item page data
type MockWorkspaceServiceForItemPageData struct {
	workspacepb.UnimplementedWorkspaceDomainServiceServer
	ReadWorkspaceFunc func(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error)
}

func (m *MockWorkspaceServiceForItemPageData) CreateWorkspace(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *MockWorkspaceServiceForItemPageData) ReadWorkspace(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	if m.ReadWorkspaceFunc != nil {
		return m.ReadWorkspaceFunc(ctx, req)
	}

	// Default implementation - return a workspace that matches the requested ID
	workspace := &workspacepb.Workspace{
		Id:          req.Data.Id,
		Name:        "Test Workspace",
		Description: "A test workspace",
		Private:     false,
		Active:      true,
	}

	return &workspacepb.ReadWorkspaceResponse{
		Data:    []*workspacepb.Workspace{workspace},
		Success: true,
	}, nil
}

func (m *MockWorkspaceServiceForItemPageData) UpdateWorkspace(ctx context.Context, req *workspacepb.UpdateWorkspaceRequest) (*workspacepb.UpdateWorkspaceResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *MockWorkspaceServiceForItemPageData) DeleteWorkspace(ctx context.Context, req *workspacepb.DeleteWorkspaceRequest) (*workspacepb.DeleteWorkspaceResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *MockWorkspaceServiceForItemPageData) ListWorkspaces(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *MockWorkspaceServiceForItemPageData) GetWorkspaceListPageData(ctx context.Context, req *workspacepb.GetWorkspaceListPageDataRequest) (*workspacepb.GetWorkspaceListPageDataResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *MockWorkspaceServiceForItemPageData) GetWorkspaceItemPageData(ctx context.Context, req *workspacepb.GetWorkspaceItemPageDataRequest) (*workspacepb.GetWorkspaceItemPageDataResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func TestGetWorkspaceItemPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	mockRepo := &MockWorkspaceServiceForItemPageData{}
	repositories := GetWorkspaceItemPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceItemPageDataRequest{
		WorkspaceId: "workspace-123",
	}

	// Execute
	resp, err := useCase.Execute(ctx, req)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}

	if resp.Workspace == nil {
		t.Fatal("Expected workspace data, got nil")
	}

	if resp.Workspace.Id != "workspace-123" {
		t.Errorf("Expected workspace ID 'workspace-123', got '%s'", resp.Workspace.Id)
	}
}

func TestGetWorkspaceItemPageDataUseCase_Execute_NotFound(t *testing.T) {
	// Setup
	mockRepo := &MockWorkspaceServiceForItemPageData{
		ReadWorkspaceFunc: func(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
			return &workspacepb.ReadWorkspaceResponse{
				Data:    []*workspacepb.Workspace{},
				Success: true,
			}, nil
		},
	}
	repositories := GetWorkspaceItemPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceItemPageDataRequest{
		WorkspaceId: "nonexistent-workspace",
	}

	// Execute
	resp, err := useCase.Execute(ctx, req)

	// Assertions
	if err == nil {
		t.Fatal("Expected error for not found workspace, got nil")
	}

	if resp != nil {
		t.Error("Expected no response for not found workspace, got one")
	}
}

func TestGetWorkspaceItemPageDataUseCase_Execute_RepositoryError(t *testing.T) {
	// Setup
	mockRepo := &MockWorkspaceServiceForItemPageData{
		ReadWorkspaceFunc: func(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
			return nil, errors.New("repository error")
		},
	}
	repositories := GetWorkspaceItemPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceItemPageDataRequest{
		WorkspaceId: "workspace-123",
	}

	// Execute
	resp, err := useCase.Execute(ctx, req)

	// Assertions
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if resp != nil {
		t.Error("Expected no response, got one")
	}
}

func TestGetWorkspaceItemPageDataUseCase_Execute_IdMismatch(t *testing.T) {
	// Setup
	mockRepo := &MockWorkspaceServiceForItemPageData{
		ReadWorkspaceFunc: func(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
			// Return a workspace with different ID than requested
			workspace := &workspacepb.Workspace{
				Id:          "different-id",
				Name:        "Test Workspace",
				Description: "A test workspace",
				Private:     false,
				Active:      true,
			}

			return &workspacepb.ReadWorkspaceResponse{
				Data:    []*workspacepb.Workspace{workspace},
				Success: true,
			}, nil
		},
	}
	repositories := GetWorkspaceItemPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceItemPageDataRequest{
		WorkspaceId: "workspace-123",
	}

	// Execute
	resp, err := useCase.Execute(ctx, req)

	// Assertions
	if err == nil {
		t.Fatal("Expected error for ID mismatch, got nil")
	}

	if resp != nil {
		t.Error("Expected no response for ID mismatch, got one")
	}
}

func TestGetWorkspaceItemPageDataUseCase_Execute_WithTransaction(t *testing.T) {
	// Setup with real transaction service
	mockRepo := &MockWorkspaceServiceForItemPageData{}
	repositories := GetWorkspaceItemPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceItemPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceItemPageDataRequest{
		WorkspaceId: "workspace-123",
	}

	// Execute
	resp, err := useCase.Execute(ctx, req)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}
}

func TestGetWorkspaceItemPageDataUseCase_validateInput_NilRequest(t *testing.T) {
	// Setup
	repositories := GetWorkspaceItemPageDataRepositories{}
	services := GetWorkspaceItemPageDataServices{
		TranslationService: ports.NewNoOpTranslationService(),
	}
	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test
	ctx := context.Background()
	err := useCase.validateInput(ctx, nil)

	// Assertions
	if err == nil {
		t.Error("Expected validation error for nil request")
	}
}

func TestGetWorkspaceItemPageDataUseCase_validateInput_EmptyWorkspaceId(t *testing.T) {
	// Setup
	repositories := GetWorkspaceItemPageDataRepositories{}
	services := GetWorkspaceItemPageDataServices{
		TranslationService: ports.NewNoOpTranslationService(),
	}
	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceItemPageDataRequest{
		WorkspaceId: "",
	}
	err := useCase.validateInput(ctx, req)

	// Assertions
	if err == nil {
		t.Error("Expected validation error for empty workspace ID")
	}
}

func TestGetWorkspaceItemPageDataUseCase_validateBusinessRules_ShortId(t *testing.T) {
	// Setup
	repositories := GetWorkspaceItemPageDataRepositories{}
	services := GetWorkspaceItemPageDataServices{
		TranslationService: ports.NewNoOpTranslationService(),
	}
	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test
	ctx := context.Background()
	err := useCase.validateBusinessRules(ctx, "xy") // Too short

	// Assertions
	if err == nil {
		t.Error("Expected validation error for short workspace ID")
	}
}

func TestGetWorkspaceItemPageDataUseCase_validateBusinessRules_ValidId(t *testing.T) {
	// Setup
	repositories := GetWorkspaceItemPageDataRepositories{}
	services := GetWorkspaceItemPageDataServices{
		TranslationService: ports.NewNoOpTranslationService(),
	}
	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test
	ctx := context.Background()
	err := useCase.validateBusinessRules(ctx, "workspace-123") // Valid length

	// Assertions
	if err != nil {
		t.Errorf("Expected no validation error for valid workspace ID, got: %v", err)
	}
}

func TestGetWorkspaceItemPageDataUseCase_processWorkspaceForUser(t *testing.T) {
	// Setup
	repositories := GetWorkspaceItemPageDataRepositories{}
	services := GetWorkspaceItemPageDataServices{
		TranslationService: ports.NewNoOpTranslationService(),
	}
	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	workspace := &workspacepb.Workspace{
		Id:          "workspace-123",
		Name:        "Test Workspace",
		Description: "A test workspace",
		Private:     false,
		Active:      true,
	}

	// Execute
	result, err := useCase.processWorkspaceForUser(ctx, workspace)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected processed workspace, got nil")
	}

	// Should return the same workspace (since we're not doing complex processing yet)
	if result.Id != workspace.Id {
		t.Errorf("Expected same workspace ID, got different one")
	}
}

func TestGetWorkspaceItemPageDataUseCase_checkAuthorizationPermissions_NoService(t *testing.T) {
	// Setup without authorization service
	repositories := GetWorkspaceItemPageDataRepositories{}
	services := GetWorkspaceItemPageDataServices{
		AuthorizationService: nil, // No service
		TranslationService:   ports.NewNoOpTranslationService(),
	}
	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test
	ctx := context.Background()
	err := useCase.checkAuthorizationPermissions(ctx, "workspace-123")

	// Assertions
	if err != nil {
		t.Errorf("Expected no error when no authorization service, got: %v", err)
	}
}

func TestGetWorkspaceItemPageDataUseCase_applyDataTransformation(t *testing.T) {
	// Setup
	repositories := GetWorkspaceItemPageDataRepositories{}
	services := GetWorkspaceItemPageDataServices{}
	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	workspace := &workspacepb.Workspace{
		Id:          "workspace-123",
		Name:        "Test Workspace",
		Description: "A test workspace",
		Private:     false,
		Active:      true,
	}

	// Execute
	result := useCase.applyDataTransformation(ctx, workspace)

	// Assertions
	if result == nil {
		t.Fatal("Expected transformed workspace, got nil")
	}

	// Should return the same workspace (since we're not doing transformations yet)
	if result.Id != workspace.Id {
		t.Errorf("Expected same workspace ID, got different one")
	}
}

func TestGetWorkspaceItemPageDataUseCase_applySecurityFiltering(t *testing.T) {
	// Setup
	repositories := GetWorkspaceItemPageDataRepositories{}
	services := GetWorkspaceItemPageDataServices{}
	useCase := NewGetWorkspaceItemPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	workspace := &workspacepb.Workspace{
		Id:          "workspace-123",
		Name:        "Test Workspace",
		Description: "A test workspace",
		Private:     false,
		Active:      true,
	}

	// Execute
	result, err := useCase.applySecurityFiltering(ctx, workspace)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected filtered workspace, got nil")
	}

	// Should return the same workspace (since we're not doing filtering yet)
	if result.Id != workspace.Id {
		t.Errorf("Expected same workspace ID, got different one")
	}
}
