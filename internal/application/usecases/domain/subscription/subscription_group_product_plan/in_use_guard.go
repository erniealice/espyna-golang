package subscription_group_product_plan

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	espynaports "github.com/erniealice/espyna-golang/ports"
)

// guardNotInUse fail-closes Delete and the ACTIVE->EXCLUDED Update transition
// while an active subscription_group_product_plan_staff row references the
// class (plan.md §2 "Delete/Exclude guards: refuse when active assignments
// exist or a live realized course rides the class"; see checker.go's doc
// comment for the "live realized course" leg's current scope). A nil checker
// (composition gap) degrades to un-guarded rather than bricking every
// delete/exclude — mirrors plan.UpdatePlanUseCase's ReferenceChecker tolerance.
func guardNotInUse(ctx context.Context, checker espynaports.Checker, tr ports.Translator, id string) error {
	if checker == nil || id == "" {
		return nil
	}
	inUse, err := checker.GetSubscriptionGroupProductPlanInUseIDs(ctx, []string{id})
	if err != nil {
		return err
	}
	if inUse[id] {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, tr,
			"subscription_group_product_plan.validation.delete_blocked_in_use",
			"Cannot remove this class while it still has active assignments. Remove them first. [DEFAULT]"))
	}
	return nil
}
