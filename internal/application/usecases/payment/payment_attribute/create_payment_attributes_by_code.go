package payment_attribute

import (
	"context"
	"fmt"
	"log"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_attribute"
)

// CreatePaymentAttributesByCodeRepositories groups repository dependencies
type CreatePaymentAttributesByCodeRepositories struct {
	PaymentAttribute paymentattributepb.PaymentAttributeDomainServiceServer
	Attribute        commonpb.AttributeDomainServiceServer
}

// CreatePaymentAttributesByCodeUseCase creates payment attributes using attribute codes.
// Internally resolves each code to an attribute ID before creating.
type CreatePaymentAttributesByCodeUseCase struct {
	repos         CreatePaymentAttributesByCodeRepositories
	createUseCase *CreatePaymentAttributeUseCase
}

// NewCreatePaymentAttributesByCodeUseCase creates the use case with required dependencies.
func NewCreatePaymentAttributesByCodeUseCase(
	repos CreatePaymentAttributesByCodeRepositories,
	createUseCase *CreatePaymentAttributeUseCase,
) *CreatePaymentAttributesByCodeUseCase {
	return &CreatePaymentAttributesByCodeUseCase{
		repos:         repos,
		createUseCase: createUseCase,
	}
}

// Execute creates payment attributes by resolving attribute codes to IDs.
// Skips attributes with empty values or codes that cannot be resolved.
func (uc *CreatePaymentAttributesByCodeUseCase) Execute(
	ctx context.Context,
	req *paymentattributepb.CreatePaymentAttributesByCodeRequest,
) (*paymentattributepb.CreatePaymentAttributesByCodeResponse, error) {
	// Access nested data field (standard pattern for workflow engine compatibility)
	if req == nil || req.Data == nil || len(req.Data.AttributesMap) == 0 {
		return &paymentattributepb.CreatePaymentAttributesByCodeResponse{
			Success: true,
			Data:    []*paymentattributepb.PaymentAttribute{},
		}, nil
	}

	data := req.Data
	var created []*paymentattributepb.PaymentAttribute

	for code, value := range data.AttributesMap {
		if code == "" || value == "" {
			continue
		}
		attributeID, err := uc.readAttributeByCode(ctx, code)
		if err != nil {
			log.Printf("[CreatePaymentAttributesByCode] attribute '%s' not found: %v", code, err)
			continue
		}

		// Create single attribute using the create use case
		singleReq := &paymentattributepb.CreatePaymentAttributeRequest{
			Data: &paymentattributepb.PaymentAttribute{
				PaymentId:   data.PaymentId,
				AttributeId: attributeID,
				Value:       value,
			},
		}

		resp, err := uc.createUseCase.Execute(ctx, singleReq)
		if err != nil {
			log.Printf("[CreatePaymentAttributesByCode] failed to create attribute_id=%s: %v", attributeID, err)
			continue
		}
		if resp != nil && len(resp.Data) > 0 {
			created = append(created, resp.Data[0])
		}
	}

	return &paymentattributepb.CreatePaymentAttributesByCodeResponse{
		Success: true,
		Data:    created,
	}, nil
}

// readAttributeByCode looks up an Attribute by its code field and returns the attribute ID.
func (uc *CreatePaymentAttributesByCodeUseCase) readAttributeByCode(ctx context.Context, code string) (string, error) {
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
