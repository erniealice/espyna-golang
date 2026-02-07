package attribute

import (
	"context"
	"fmt"
	"log"

	attributepb "leapfor.xyz/esqyma/golang/v1/domain/common"
)

// ReadAttributeByCode looks up an Attribute by its code field and returns the attribute ID.
// This is a helper method that uses the ListAttributes use case with a code filter.
// Returns empty string and error if not found.
func (uc *UseCases) ReadAttributeByCode(ctx context.Context, code string) (string, error) {
	if uc.ListAttributes == nil {
		return "", fmt.Errorf("ListAttributes use case not initialized")
	}

	// List attributes filtered by code
	listReq := &attributepb.ListAttributesRequest{
		Filters: &attributepb.FilterRequest{
			Filters: []*attributepb.TypedFilter{
				{
					Field: "code",
					FilterType: &attributepb.TypedFilter_StringFilter{
						StringFilter: &attributepb.StringFilter{
							Value:    code,
							Operator: attributepb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	}

	resp, err := uc.ListAttributes.Execute(ctx, listReq)
	if err != nil {
		return "", fmt.Errorf("failed to list attributes by code '%s': %w", code, err)
	}

	if resp == nil || len(resp.Data) == 0 {
		return "", fmt.Errorf("attribute with code '%s' not found", code)
	}

	attributeID := resp.Data[0].Id
	log.Printf("[Attribute.ReadAttributeByCode] Found attribute: code=%s, id=%s", code, attributeID)
	return attributeID, nil
}
