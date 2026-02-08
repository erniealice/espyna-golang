package workspace

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspacepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace"
)

// MockWorkspaceServiceForListPageData implements WorkspaceDomainServiceServer for testing list page data
type MockWorkspaceServiceForListPageData struct {
	workspacepb.UnimplementedWorkspaceDomainServiceServer
	ListWorkspacesFunc func(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error)
}

func (m *MockWorkspaceServiceForListPageData) CreateWorkspace(ctx context.Context, req *workspacepb.CreateWorkspaceRequest) (*workspacepb.CreateWorkspaceResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *MockWorkspaceServiceForListPageData) ReadWorkspace(ctx context.Context, req *workspacepb.ReadWorkspaceRequest) (*workspacepb.ReadWorkspaceResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *MockWorkspaceServiceForListPageData) UpdateWorkspace(ctx context.Context, req *workspacepb.UpdateWorkspaceRequest) (*workspacepb.UpdateWorkspaceResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *MockWorkspaceServiceForListPageData) DeleteWorkspace(ctx context.Context, req *workspacepb.DeleteWorkspaceRequest) (*workspacepb.DeleteWorkspaceResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *MockWorkspaceServiceForListPageData) ListWorkspaces(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
	if m.ListWorkspacesFunc != nil {
		return m.ListWorkspacesFunc(ctx, req)
	}
	return &workspacepb.ListWorkspacesResponse{
		Data: []*workspacepb.Workspace{
			{
				Id:          "workspace-1",
				Name:        "Test Workspace 1",
				Description: "A test workspace",
				Private:     false,
				Active:      true,
			},
			{
				Id:          "workspace-2",
				Name:        "Test Workspace 2",
				Description: "Another test workspace",
				Private:     true,
				Active:      true,
			},
		},
		Success: true,
	}, nil
}

func (m *MockWorkspaceServiceForListPageData) GetWorkspaceListPageData(ctx context.Context, req *workspacepb.GetWorkspaceListPageDataRequest) (*workspacepb.GetWorkspaceListPageDataResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *MockWorkspaceServiceForListPageData) GetWorkspaceItemPageData(ctx context.Context, req *workspacepb.GetWorkspaceItemPageDataRequest) (*workspacepb.GetWorkspaceItemPageDataResponse, error) {
	return nil, errors.New("not implemented in mock")
}

func TestGetWorkspaceListPageDataUseCase_Execute_Success(t *testing.T) {
	// Setup
	mockRepo := &MockWorkspaceServiceForListPageData{}
	repositories := GetWorkspaceListPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceListPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceListPageDataRequest{
		Pagination: &commonpb.PaginationRequest{
			Limit: 10,
			Method: &commonpb.PaginationRequest_Offset{
				Offset: &commonpb.OffsetPagination{
					Page: 1,
				},
			},
		},
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

	if len(resp.WorkspaceList) != 2 {
		t.Errorf("Expected 2 workspaces, got %d", len(resp.WorkspaceList))
	}

	if resp.Pagination == nil {
		t.Error("Expected pagination metadata")
	}
}

func TestGetWorkspaceListPageDataUseCase_Execute_EmptyResult(t *testing.T) {
	// Setup
	mockRepo := &MockWorkspaceServiceForListPageData{
		ListWorkspacesFunc: func(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
			return &workspacepb.ListWorkspacesResponse{
				Data:    []*workspacepb.Workspace{},
				Success: true,
			}, nil
		},
	}
	repositories := GetWorkspaceListPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceListPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceListPageDataRequest{}

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

	if len(resp.WorkspaceList) != 0 {
		t.Errorf("Expected 0 workspaces, got %d", len(resp.WorkspaceList))
	}
}

func TestGetWorkspaceListPageDataUseCase_Execute_RepositoryError(t *testing.T) {
	// Setup
	mockRepo := &MockWorkspaceServiceForListPageData{
		ListWorkspacesFunc: func(ctx context.Context, req *workspacepb.ListWorkspacesRequest) (*workspacepb.ListWorkspacesResponse, error) {
			return nil, errors.New("repository error")
		},
	}
	repositories := GetWorkspaceListPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceListPageDataUseCase(repositories, services)

	// Test input
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceListPageDataRequest{}

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

func TestGetWorkspaceListPageDataUseCase_Execute_WithSearch(t *testing.T) {
	// Setup
	mockRepo := &MockWorkspaceServiceForListPageData{}
	repositories := GetWorkspaceListPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceListPageDataUseCase(repositories, services)

	// Test input with search
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceListPageDataRequest{
		Search: &commonpb.SearchRequest{
			Query: "test",
			Options: &commonpb.SearchOptions{
				SearchFields: []string{"name", "description"},
				MaxResults:   100,
			},
		},
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

	// Should return some workspaces (filtered by search)
	if len(resp.WorkspaceList) == 0 {
		t.Error("Expected at least some workspaces in search results")
	}
}

func TestGetWorkspaceListPageDataUseCase_Execute_WithFilters(t *testing.T) {
	// Setup
	mockRepo := &MockWorkspaceServiceForListPageData{}
	repositories := GetWorkspaceListPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceListPageDataUseCase(repositories, services)

	// Test input with filters
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceListPageDataRequest{
		Filters: &commonpb.FilterRequest{
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

func TestGetWorkspaceListPageDataUseCase_Execute_WithSort(t *testing.T) {
	// Setup
	mockRepo := &MockWorkspaceServiceForListPageData{}
	repositories := GetWorkspaceListPageDataRepositories{
		Workspace: mockRepo,
	}
	services := GetWorkspaceListPageDataServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	useCase := NewGetWorkspaceListPageDataUseCase(repositories, services)

	// Test input with sort
	ctx := context.Background()
	req := &workspacepb.GetWorkspaceListPageDataRequest{
		Sort: &commonpb.SortRequest{
			Fields: []*commonpb.SortField{
				{
					Field:     "name",
					Direction: commonpb.SortDirection_ASC,
				},
			},
		},
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

	// Verify sorting (workspaces should be sorted by name)
	if len(resp.WorkspaceList) >= 2 {
		first := resp.WorkspaceList[0]
		second := resp.WorkspaceList[1]
		if first.Name > second.Name {
			t.Error("Workspaces should be sorted by name in ascending order")
		}
	}
}

func TestGetWorkspaceListPageDataUseCase_validateInput_NilRequest(t *testing.T) {
	// Setup
	repositories := GetWorkspaceListPageDataRepositories{}
	services := GetWorkspaceListPageDataServices{
		TranslationService: ports.NewNoOpTranslationService(),
	}
	useCase := NewGetWorkspaceListPageDataUseCase(repositories, services)

	// Test
	ctx := context.Background()
	err := useCase.validateInput(ctx, nil)

	// Assertions
	if err == nil {
		t.Error("Expected validation error for nil request")
	}
}

func TestGetWorkspaceListPageDataUseCase_validatePagination_InvalidLimit(t *testing.T) {
	// Setup
	repositories := GetWorkspaceListPageDataRepositories{}
	services := GetWorkspaceListPageDataServices{
		TranslationService: ports.NewNoOpTranslationService(),
	}
	useCase := NewGetWorkspaceListPageDataUseCase(repositories, services)

	// Test with invalid limit
	ctx := context.Background()
	pagination := &commonpb.PaginationRequest{
		Limit: 200, // Too high
	}

	err := useCase.validatePagination(ctx, pagination)

	// Assertions
	if err == nil {
		t.Error("Expected validation error for invalid limit")
	}
}

func TestGetWorkspaceListPageDataUseCase_isValidWorkspaceField(t *testing.T) {
	// Setup
	repositories := GetWorkspaceListPageDataRepositories{}
	services := GetWorkspaceListPageDataServices{}
	useCase := NewGetWorkspaceListPageDataUseCase(repositories, services)

	// Test valid fields
	validFields := []string{"id", "name", "description", "private", "active", "date_created"}
	for _, field := range validFields {
		if !useCase.isValidWorkspaceField(field) {
			t.Errorf("Field %s should be valid", field)
		}
	}

	// Test invalid fields
	invalidFields := []string{"invalid_field", "random", "not_a_field"}
	for _, field := range invalidFields {
		if useCase.isValidWorkspaceField(field) {
			t.Errorf("Field %s should be invalid", field)
		}
	}
}
