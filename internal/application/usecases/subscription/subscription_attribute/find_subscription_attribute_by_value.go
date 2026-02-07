package subscription_attribute

import (
	"context"
	"fmt"
	"log"

	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
	subscriptionattributepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription_attribute"
)

// FindSubscriptionAttributeByValue finds subscription attributes by attribute ID and value.
// Returns the first matching subscription_attribute record, or nil if not found.
func (uc *UseCases) FindSubscriptionAttributeByValue(ctx context.Context, attributeID, value string) (*subscriptionattributepb.SubscriptionAttribute, error) {
	if uc.ListSubscriptionAttributes == nil {
		return nil, fmt.Errorf("ListSubscriptionAttributes use case not initialized")
	}

	// List subscription attributes filtered by attribute_id and value
	listReq := &subscriptionattributepb.ListSubscriptionAttributesRequest{
		Filters: &attributepb.FilterRequest{
			Filters: []*attributepb.TypedFilter{
				{
					Field: "attribute_id",
					FilterType: &attributepb.TypedFilter_StringFilter{
						StringFilter: &attributepb.StringFilter{
							Value:    attributeID,
							Operator: attributepb.StringOperator_STRING_EQUALS,
						},
					},
				},
				{
					Field: "value",
					FilterType: &attributepb.TypedFilter_StringFilter{
						StringFilter: &attributepb.StringFilter{
							Value:    value,
							Operator: attributepb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	}

	resp, err := uc.ListSubscriptionAttributes.Execute(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription attributes: %w", err)
	}

	if resp == nil || len(resp.Data) == 0 {
		log.Printf("[SubscriptionAttribute.FindSubscriptionAttributeByValue] Not found: attribute=%s, value=%s", attributeID, value)
		return nil, nil // Not found is not an error
	}

	log.Printf("[SubscriptionAttribute.FindSubscriptionAttributeByValue] Found: attribute=%s, value=%s, subscription_id=%s",
		attributeID, value, resp.Data[0].SubscriptionId)
	return resp.Data[0], nil
}
