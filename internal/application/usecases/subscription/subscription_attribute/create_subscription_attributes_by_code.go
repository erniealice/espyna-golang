package subscription_attribute

import (
	"context"
	"fmt"
	"log"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// CreateSubscriptionAttributesByCodeRepositories groups repository dependencies
type CreateSubscriptionAttributesByCodeRepositories struct {
	SubscriptionAttribute subscriptionattributepb.SubscriptionAttributeDomainServiceServer
	Attribute             commonpb.AttributeDomainServiceServer
}

// CreateSubscriptionAttributesByCodeUseCase creates subscription attributes using attribute codes.
// Internally resolves each code to an attribute ID before creating.
type CreateSubscriptionAttributesByCodeUseCase struct {
	repos         CreateSubscriptionAttributesByCodeRepositories
	createUseCase *CreateSubscriptionAttributeUseCase
}

// NewCreateSubscriptionAttributesByCodeUseCase creates the use case with required dependencies.
func NewCreateSubscriptionAttributesByCodeUseCase(
	repos CreateSubscriptionAttributesByCodeRepositories,
	createUseCase *CreateSubscriptionAttributeUseCase,
) *CreateSubscriptionAttributesByCodeUseCase {
	return &CreateSubscriptionAttributesByCodeUseCase{
		repos:         repos,
		createUseCase: createUseCase,
	}
}

// Execute creates subscription attributes by resolving attribute codes to IDs.
// Skips attributes with empty values or codes that cannot be resolved.
func (uc *CreateSubscriptionAttributesByCodeUseCase) Execute(
	ctx context.Context,
	req *subscriptionattributepb.CreateSubscriptionAttributesByCodeRequest,
) (*subscriptionattributepb.CreateSubscriptionAttributesByCodeResponse, error) {
	// Access nested data field (standard pattern for workflow engine compatibility)
	if req == nil || req.Data == nil || len(req.Data.AttributesMap) == 0 {
		return &subscriptionattributepb.CreateSubscriptionAttributesByCodeResponse{
			Success: true,
			Data:    []*subscriptionattributepb.SubscriptionAttribute{},
		}, nil
	}

	data := req.Data
	var created []*subscriptionattributepb.SubscriptionAttribute

	for code, value := range data.AttributesMap {
		if code == "" || value == "" {
			continue
		}
		attributeID, err := uc.readAttributeByCode(ctx, code)
		if err != nil {
			log.Printf("[CreateSubscriptionAttributesByCode] attribute '%s' not found: %v", code, err)
			continue
		}

		// Create single attribute using the create use case
		singleReq := &subscriptionattributepb.CreateSubscriptionAttributeRequest{
			Data: &subscriptionattributepb.SubscriptionAttribute{
				SubscriptionId: data.SubscriptionId,
				AttributeId:    attributeID,
				Value:          value,
			},
		}

		resp, err := uc.createUseCase.Execute(ctx, singleReq)
		if err != nil {
			log.Printf("[CreateSubscriptionAttributesByCode] failed to create attribute_id=%s: %v", attributeID, err)
			continue
		}
		if resp != nil && len(resp.Data) > 0 {
			created = append(created, resp.Data[0])
		}
	}

	return &subscriptionattributepb.CreateSubscriptionAttributesByCodeResponse{
		Success: true,
		Data:    created,
	}, nil
}

// readAttributeByCode looks up an Attribute by its code field and returns the attribute ID.
func (uc *CreateSubscriptionAttributesByCodeUseCase) readAttributeByCode(ctx context.Context, code string) (string, error) {
	listReq := &commonpb.ListAttributesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "code",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    code,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	}

	resp, err := uc.repos.Attribute.ListAttributes(ctx, listReq)
	if err != nil {
		return "", fmt.Errorf("failed to list attributes by code '%s': %w", code, err)
	}

	if resp == nil || len(resp.Data) == 0 {
		return "", fmt.Errorf("attribute with code '%s' not found", code)
	}

	return resp.Data[0].Id, nil
}
