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

// GetClientByEmailRepositories groups all repository dependencies
type GetClientByEmailRepositories struct {
	Client clientpb.ClientDomainServiceServer // Primary entity repository
	User   userpb.UserDomainServiceServer     // User repository for email lookup
}

// GetClientByEmailServices groups all business service dependencies
type GetClientByEmailServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

// GetClientByEmailUseCase handles the business logic for finding a client by email address.
// This performs a two-step lookup:
// 1. Search for user by email_address using User.ListUsers
// 2. If user found, search for associated client by user_id using Client.ListClients
//
// Input: ListUsersRequest - uses filters to search by email_address
// Output: ListClientsResponse - returns the found client (does NOT create if not found)
type GetClientByEmailUseCase struct {
	repositories GetClientByEmailRepositories
	services     GetClientByEmailServices
}

// NewGetClientByEmailUseCase creates use case with grouped dependencies
func NewGetClientByEmailUseCase(
	repositories GetClientByEmailRepositories,
	services GetClientByEmailServices,
) *GetClientByEmailUseCase {
	return &GetClientByEmailUseCase{
		repositories: repositories,
		services:     services,
	}
}

// NewGetClientByEmailUseCaseUngrouped creates use case with individual parameters
// Deprecated: Use NewGetClientByEmailUseCase with grouped parameters instead
func NewGetClientByEmailUseCaseUngrouped(clientRepo clientpb.ClientDomainServiceServer, userRepo userpb.UserDomainServiceServer) *GetClientByEmailUseCase {
	repositories := GetClientByEmailRepositories{
		Client: clientRepo,
		User:   userRepo,
	}

	services := GetClientByEmailServices{
		AuthorizationService: nil,
		TransactionService:   ports.NewNoOpTransactionService(),
		TranslationService:   ports.NewNoOpTranslationService(),
	}

	return NewGetClientByEmailUseCase(repositories, services)
}

// Execute performs the get client by email operation
// Input: ListUsersRequest with email_address filter
// Output: ListClientsResponse with the found client
//
// The use case:
// 1. Extracts email_address from the ListUsersRequest filters
// 2. Searches for an existing user by email_address using User.ListUsers
// 3. If user found, searches for associated client by user_id using Client.ListClients
// 4. Returns the client if found, or an error if not found (does NOT create)
func (uc *GetClientByEmailUseCase) Execute(ctx context.Context, req *userpb.ListUsersRequest) (*clientpb.ListClientsResponse, error) {
	// Input validation
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.request_required", "Request is required [DEFAULT]"))
	}

	// Extract email from filters for validation
	email := uc.extractEmailFromFilters(req.Filters)
	if email == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.validation.email_filter_required", "Email address filter is required [DEFAULT]"))
	}

	// Step 1: Search for existing user by email using the provided request
	if uc.repositories.User == nil {
		return nil, errors.New("user repository not available")
	}

	userListResp, err := uc.repositories.User.ListUsers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search for user: %w", err)
	}

	if userListResp == nil || len(userListResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.errors.user_not_found", "User not found with the provided email [DEFAULT]"))
	}

	// Step 2: Search for associated client by user_id
	existingUser := userListResp.Data[0]

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

	if clientListResp == nil || len(clientListResp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "client.errors.client_not_found", "Client not found for the provided email [DEFAULT]"))
	}

	return clientListResp, nil
}

// extractEmailFromFilters extracts the email_address value from filters
func (uc *GetClientByEmailUseCase) extractEmailFromFilters(filters *commonpb.FilterRequest) string {
	if filters == nil || len(filters.Filters) == 0 {
		return ""
	}

	for _, filter := range filters.Filters {
		if filter.Field == "email_address" {
			if sf := filter.GetStringFilter(); sf != nil {
				return sf.Value
			}
		}
	}

	return ""
}
