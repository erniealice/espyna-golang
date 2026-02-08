package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// FindOrCreateClientRepositories groups all repository dependencies
type FindOrCreateClientRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
	User   userpb.UserDomainServiceServer     // User repository for email lookup
}

// FindOrCreateClientServices groups all business service dependencies
type FindOrCreateClientServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// FindOrCreateClientUseCase handles the business logic for finding an existing client
// by email address, or creating a new one if not found.
// This implements the "find or create" pattern commonly used in checkout flows.
//
// Input: ListUsersRequest - uses filters to search by email_address
// Output: ListClientsResponse - returns the found or created client
type FindOrCreateClientUseCase struct {
	repositories FindOrCreateClientRepositories
	services     FindOrCreateClientServices
}

// NewFindOrCreateClientUseCase creates use case with grouped dependencies
func NewFindOrCreateClientUseCase(
	repositories FindOrCreateClientRepositories,
	services FindOrCreateClientServices,
) *FindOrCreateClientUseCase {
	return &FindOrCreateClientUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewFindOrCreateClientUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewFindOrCreateClientUseCase with grouped parameters instead
func NewFindOrCreateClientUseCaseUngrouped(clientRepo clientpb.ClientDomainServiceServer, userRepo userpb.UserDomainServiceServer) *FindOrCreateClientUseCase {
	repositories := FindOrCreateClientRepositories{
		Client: clientRepo,
		User:   userRepo,
	}

	services := FindOrCreateClientServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
		IDService:            ports.NewNoOpIDService(),
	}

	return NewFindOrCreateClientUseCase(repositories, services)
}

// Execute performs the find or create client operation
// Input: CreateClientRequest with nested Client.User containing email_address, first_name, last_name
// Output: ListClientsResponse with the found or created client
//
// The use case:
// 1. Extracts email_address from req.Data.User.EmailAddress
// 2. Searches for an existing user by email_address using User.ListUsers
// 3. If user found, searches for associated client by user_id using Client.ListClients
// 4. If client found, returns it in ListClientsResponse
// 5. If no user found, delegates to CreateClient use case which handles User+Client creation
func (uc *FindOrCreateClientUseCase) Execute(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.ListClientsResponse, error) {
	// Input validation
	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.request_required", "Request is required [DEFAULT]"))
	}

	// Extract email from nested User
	if req.Data.User == nil || req.Data.User.EmailAddress == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.email_required", "Client user email address is required [DEFAULT]"))
	}

	email := req.Data.User.EmailAddress

	// Step 1: Search for existing user by email
	userListReq := &userpb.ListUsersRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "email_address",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    email,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	}

	userListResp, err := uc.repositories.User.ListUsers(ctx, userListReq)
	if err != nil {
		return nil, fmt.Errorf("failed to search for user: %w", err)
	}

	// Step 2: If user exists, search for associated client
	if userListResp != nil && len(userListResp.Data) > 0 {
		existingUser := userListResp.Data[0]

		// Search for client by user_id
		clientListResp, err := uc.repositories.Client.ListClients(ctx, &clientpb.ListClientsRequest{
			Filters: &commonpb.FilterRequest{
				Filters: []*commonpb.TypedFilter{
					{
						Field: "user_id",
						FilterType: &commonpb.TypedFilter_StringFilter{
							StringFilter: &commonpb.StringFilter{
								Value:    existingUser.Id,
								Operator: commonpb.StringOperator_STRING_EQUALS,
							},
						},
					},
				},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search for client: %w", err)
		}

		// If client exists, return it
		if clientListResp != nil && len(clientListResp.Data) > 0 {
			return clientListResp, nil
		}
	}

	// Step 3: No user found - delegate to CreateClient use case
	// This properly handles User+Client creation with the provided first_name, last_name, email_address
	createClientUC := NewCreateClientUseCase(
		CreateClientRepositories{
			Client: uc.repositories.Client,
			User:   uc.repositories.User,
		},
		CreateClientServices{
			AuthorizationService: uc.services.AuthorizationService,
			TransactionService:   uc.services.TransactionService,
			TranslationService:   uc.services.TranslationService,
			IDService:            uc.services.IDService,
		},
	)

	createResp, err := createClientUC.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	if createResp == nil || len(createResp.Data) == 0 {
		return nil, errors.New("client creation returned no data")
	}

	return &clientpb.ListClientsResponse{
		Data:    createResp.Data,
		Success: true,
	}, nil
}
