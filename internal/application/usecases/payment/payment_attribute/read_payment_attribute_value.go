package payment_attribute

import (
	"context"
	"fmt"
	"log"

	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	paymentattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/payment/payment_attribute"
)

// ReadPaymentAttributeValue retrieves the value of a PaymentAttribute by payment ID and attribute ID.
// Returns empty string if not found (not an error).
func (uc *UseCases) ReadPaymentAttributeValue(ctx context.Context, paymentID, attributeID string) (string, error) {
	if uc.ListPaymentAttributes == nil {
		return "", fmt.Errorf("ListPaymentAttributes use case not initialized")
	}

	// List payment attributes filtered by payment_id and attribute_id
	listReq := &paymentattributepb.ListPaymentAttributesRequest{
		Filters: &attributepb.FilterRequest{
			Filters: []*attributepb.TypedFilter{
				{
					Field: "payment_id",
					FilterType: &attributepb.TypedFilter_StringFilter{
						StringFilter: &attributepb.StringFilter{
							Value:    paymentID,
							Operator: attributepb.StringOperator_STRING_EQUALS,
						},
					},
				},
				{
					Field: "attribute_id",
					FilterType: &attributepb.TypedFilter_StringFilter{
						StringFilter: &attributepb.StringFilter{
							Value:    attributeID,
							Operator: attributepb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	}

	resp, err := uc.ListPaymentAttributes.Execute(ctx, listReq)
	if err != nil {
		return "", fmt.Errorf("failed to list payment attributes: %w", err)
	}

	if resp == nil || len(resp.Data) == 0 {
		log.Printf("[PaymentAttribute.ReadPaymentAttributeValue] Not found: payment=%s, attribute=%s", paymentID, attributeID)
		return "", nil // Not found is not an error
	}

	value := resp.Data[0].Value
	log.Printf("[PaymentAttribute.ReadPaymentAttributeValue] Found: payment=%s, attribute=%s, value=%s", paymentID, attributeID, value)
	return value, nil
}
