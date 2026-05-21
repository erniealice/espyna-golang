package job_template

import (
	"context"
	"fmt"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
)

// ResolveBillRateRepositories groups all repository dependencies for this use case.
type ResolveBillRateRepositories struct {
	Resource     resourcepb.ResourceDomainServiceServer
	Product      productpb.ProductDomainServiceServer
	PriceList    pricelistpb.PriceListDomainServiceServer
	PriceProduct priceproductpb.PriceProductDomainServiceServer
}

// ResolveBillRateResult carries the resolved billing rate for a resource.
type ResolveBillRateResult struct {
	BillRate    int64 // centavos
	CostRate    int64 // centavos (from Product.price)
	Currency    string
	ProductID   string
	ProductName string
	Source      string // "price_product" or "product_default"
}

// ResolveBillRateUseCase resolves the bill rate for a resource by following the
// chain: resource_id → Resource.product_id → Product → FindApplicablePriceList →
// PriceProduct.amount (fallback: Product.price).
type ResolveBillRateUseCase struct {
	repositories ResolveBillRateRepositories
}

// NewResolveBillRateUseCase creates a new ResolveBillRateUseCase.
func NewResolveBillRateUseCase(repos ResolveBillRateRepositories) *ResolveBillRateUseCase {
	return &ResolveBillRateUseCase{repositories: repos}
}

// ResolveBillRate resolves the bill rate for the given resource, location, and date.
// entryDate must be an ISO 8601 date string (YYYY-MM-DD).
func (uc *ResolveBillRateUseCase) ResolveBillRate(
	ctx context.Context,
	resourceID string,
	locationID string,
	entryDate string,
) (*ResolveBillRateResult, error) {
	// 1. Read Resource by ID → get product_id.
	resourceResp, err := uc.repositories.Resource.ReadResource(ctx, &resourcepb.ReadResourceRequest{
		Data: &resourcepb.Resource{Id: resourceID},
	})
	if err != nil {
		return nil, fmt.Errorf("ResolveBillRate: read resource %s: %w", resourceID, err)
	}
	if len(resourceResp.GetData()) == 0 {
		return nil, fmt.Errorf("ResolveBillRate: resource %s not found", resourceID)
	}
	resource := resourceResp.GetData()[0]
	productID := resource.GetProductId()

	// 2. Read Product → get price (cost rate) and currency.
	productResp, err := uc.repositories.Product.ReadProduct(ctx, &productpb.ReadProductRequest{
		Data: &productpb.Product{Id: productID},
	})
	if err != nil {
		return nil, fmt.Errorf("ResolveBillRate: read product %s: %w", productID, err)
	}
	if len(productResp.GetData()) == 0 {
		return nil, fmt.Errorf("ResolveBillRate: product %s not found", productID)
	}
	product := productResp.GetData()[0]

	result := &ResolveBillRateResult{
		CostRate:    product.GetPrice(),
		Currency:    product.GetCurrency(),
		ProductID:   productID,
		ProductName: product.GetName(),
	}

	// 3. Find the applicable price list for this location and date.
	priceListResp, err := uc.repositories.PriceList.FindApplicablePriceList(ctx, &pricelistpb.FindApplicablePriceListRequest{
		LocationId: locationID,
		Date:       entryDate,
	})
	if err != nil {
		// Non-fatal: fall back to product default.
		result.BillRate = product.GetPrice()
		result.Source = "product_default"
		return result, nil
	}

	if !priceListResp.GetFound() || priceListResp.GetPriceList() == nil {
		// No applicable price list — fall back to product default.
		result.BillRate = product.GetPrice()
		result.Source = "product_default"
		return result, nil
	}

	priceListID := priceListResp.GetPriceList().GetId()

	// 4. Search PriceProduct records by product_id + price_list_id, then pick the
	// most recent one where date_start <= entry_date.
	ppResp, err := uc.repositories.PriceProduct.ListPriceProducts(ctx, &priceproductpb.ListPriceProductsRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "product_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    productID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
				{
					Field: "price_list_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    priceListID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
		Sort: &commonpb.SortRequest{
			Fields: []*commonpb.SortField{
				{
					Field:     "date_start",
					Direction: commonpb.SortDirection_DESC,
				},
			},
		},
	})
	if err != nil {
		// Non-fatal: fall back to product default.
		result.BillRate = product.GetPrice()
		result.Source = "product_default"
		return result, nil
	}

	// 5. Find the most recent PriceProduct where date_start <= entry_date.
	var bestPP *priceproductpb.PriceProduct
	for _, pp := range ppResp.GetData() {
		if pp.GetDateStart() <= entryDate {
			bestPP = pp
			break // list is sorted DESC by date_start, so first match is the best
		}
	}

	// 6. Apply result.
	if bestPP != nil {
		result.BillRate = bestPP.GetAmount()
		result.Source = "price_product"
	} else {
		result.BillRate = product.GetPrice()
		result.Source = "product_default"
	}

	return result, nil
}
