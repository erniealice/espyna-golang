package subscription_attribute

import (
	"context"
	"fmt"
	"log"

	attributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	subscriptionattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription_attribute"
)

// ReadSubscriptionAttributeValue retrieves the value of a SubscriptionAttribute by subscription ID and attribute ID.
// Returns empty string if not found (not an error).
func (uc *UseCases) ReadSubscriptionAttributeValue(ctx context.Context, subscriptionID, attributeID string) (string, error) {
	if uc.ListSubscriptionAttributes == nil {
		return "", fmt.Errorf("ListSubscriptionAttributes use case not initialized")
	}

	// List subscription attributes filtered by subscription_id and attribute_id
	listReq := &subscriptionattributepb.ListSubscriptionAttributesRequest{
		Filters: &attributepb.FilterRequest{
			Filters: []*attributepb.TypedFilter{
				{
					Field: "subscription_id",
					FilterType: &attributepb.TypedFilter_StringFilter{
						StringFilter: &attributepb.StringFilter{
							Value:    subscriptionID,
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

	resp, err := uc.ListSubscriptionAttributes.Execute(ctx, listReq)
	if err != nil {
		return "", fmt.Errorf("failed to list subscription attributes: %w", err)
	}

	if resp == nil || len(resp.Data) == 0 {
		log.Printf("[SubscriptionAttribute.ReadSubscriptionAttributeValue] Not found: subscription=%s, attribute=%s", subscriptionID, attributeID)
		return "", nil // Not found is not an error
	}

	value := resp.Data[0].Value
	log.Printf("[SubscriptionAttribute.ReadSubscriptionAttributeValue] Found: subscription=%s, attribute=%s, value=%s", subscriptionID, attributeID, value)
	return value, nil
}
