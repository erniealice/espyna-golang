package reference

import "context"

// NewNoOp returns a Checker that reports nothing in use and never errors.
// Useful as a sane default in non-postgres providers and tests that don't
// care about reference checks.
func NewNoOp() Checker { return &noOp{} }

type noOp struct{}

func (n *noOp) GetLocationInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetRoleInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetCategoryInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetClientInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetProductInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetProductVariantInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetProductOptionValueInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetProductOptionInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetPlanInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetPriceListInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetPricePlanInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetPriceScheduleInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetPlanClientScopeLockedIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetActiveSubscriptionCountForPricePlan(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (n *noOp) GetAssetCategoryInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetPaymentTermInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetLineInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetLocationAreaInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetEventTagInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetSubscriptionInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (n *noOp) GetSupplierInUseIDs(_ context.Context, _ []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
