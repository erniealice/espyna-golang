package job_template

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	jobactivitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
	jobsettlementpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_settlement"
	pricelistpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_list"
	priceproductpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/price_product"
	productpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/product"
	resourcepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/product/resource"
	revenuepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue"
	revenuelineitempb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/revenue_line_item"
)

// BillJobActivitiesRepositories groups all repository dependencies for this use case.
type BillJobActivitiesRepositories struct {
	JobActivity     jobactivitypb.JobActivityDomainServiceServer
	Revenue         revenuepb.RevenueDomainServiceServer
	RevenueLineItem revenuelineitempb.RevenueLineItemDomainServiceServer
	JobSettlement   jobsettlementpb.JobSettlementDomainServiceServer
	// For bill rate resolution:
	Resource     resourcepb.ResourceDomainServiceServer
	Product      productpb.ProductDomainServiceServer
	PriceList    pricelistpb.PriceListDomainServiceServer
	PriceProduct priceproductpb.PriceProductDomainServiceServer
}

// BillJobActivitiesServices groups all business service dependencies.
type BillJobActivitiesServices struct {
	TransactionService ports.TransactionService
	IDService          ports.IDService
}

// BillJobActivitiesRequest specifies which activities to bill and billing context.
type BillJobActivitiesRequest struct {
	ActivityIDs []string // specific activities to bill (selected by user)
	ClientID    string   // the client being billed
	LocationID  string   // for rate resolution
	Currency    string   // invoice currency
	Name        string   // optional — auto-generated if empty
}

// BillJobActivitiesResult carries the outcome of the billing operation.
type BillJobActivitiesResult struct {
	RevenueID string
}

// BillJobActivitiesUseCase generates a Revenue record and RevenueLineItems from
// unbilled, posted, billable job activities.
type BillJobActivitiesUseCase struct {
	repositories BillJobActivitiesRepositories
	services     BillJobActivitiesServices
}

// NewBillJobActivitiesUseCase creates a new BillJobActivitiesUseCase.
func NewBillJobActivitiesUseCase(
	repos BillJobActivitiesRepositories,
	services BillJobActivitiesServices,
) *BillJobActivitiesUseCase {
	return &BillJobActivitiesUseCase{repositories: repos, services: services}
}

// activityBillData is an internal struct that pairs a resolved activity with its
// computed bill rate and amount.
type activityBillData struct {
	activity   *jobactivitypb.JobActivity
	billRate   int64 // centavos
	billAmount int64 // centavos
	productID  string
}

// BillJobActivities validates, resolves rates, and writes the Revenue + line items
// + settlements in a single transaction.
func (uc *BillJobActivitiesUseCase) BillJobActivities(
	ctx context.Context,
	req *BillJobActivitiesRequest,
) (*BillJobActivitiesResult, error) {
	if req == nil {
		return nil, fmt.Errorf("BillJobActivities: request is required")
	}
	if len(req.ActivityIDs) == 0 {
		return nil, fmt.Errorf("BillJobActivities: at least one activity ID is required")
	}
	if req.ClientID == "" {
		return nil, fmt.Errorf("BillJobActivities: client_id is required")
	}

	// 1. Read all specified activities; validate they are posted + billable + not billed.
	activities := make([]*jobactivitypb.JobActivity, 0, len(req.ActivityIDs))
	for _, actID := range req.ActivityIDs {
		actResp, err := uc.repositories.JobActivity.ReadJobActivity(ctx, &jobactivitypb.ReadJobActivityRequest{
			Data: &jobactivitypb.JobActivity{Id: actID},
		})
		if err != nil {
			return nil, fmt.Errorf("BillJobActivities: read activity %s: %w", actID, err)
		}
		if len(actResp.GetData()) == 0 {
			return nil, fmt.Errorf("BillJobActivities: activity %s not found", actID)
		}
		act := actResp.GetData()[0]

		if act.GetPostingStatus() != jobactivitypb.ActivityPostingStatus_ACTIVITY_POSTING_STATUS_POSTED {
			return nil, fmt.Errorf("BillJobActivities: activity %s is not posted", actID)
		}
		if act.GetBillableStatus() != jobactivitypb.BillableStatus_BILLABLE_STATUS_BILLABLE {
			return nil, fmt.Errorf("BillJobActivities: activity %s is not billable", actID)
		}
		if act.GetBillAmount() != 0 {
			return nil, fmt.Errorf("BillJobActivities: activity %s has already been billed", actID)
		}

		activities = append(activities, act)
	}

	// 2. Collect unique job names for auto-generating the revenue name.
	jobNames := make(map[string]struct{})
	for _, act := range activities {
		if act.GetJob() != nil && act.GetJob().GetName() != "" {
			jobNames[act.GetJob().GetName()] = struct{}{}
		}
	}

	// 3. Resolve bill rates for each activity.
	rateResolver := &ResolveBillRateUseCase{
		repositories: ResolveBillRateRepositories{
			Resource:     uc.repositories.Resource,
			Product:      uc.repositories.Product,
			PriceList:    uc.repositories.PriceList,
			PriceProduct: uc.repositories.PriceProduct,
		},
	}

	billData := make([]*activityBillData, 0, len(activities))
	for _, act := range activities {
		bd := &activityBillData{activity: act}

		resourceID := act.GetResourceId()
		if resourceID != "" {
			// 3a. Activity has a resource — resolve via price list chain.
			entryDate := act.GetEntryDateString()
			rateResult, err := rateResolver.ResolveBillRate(ctx, resourceID, req.LocationID, entryDate)
			if err != nil {
				return nil, fmt.Errorf("BillJobActivities: resolve bill rate for activity %s: %w", act.GetId(), err)
			}
			bd.billRate = rateResult.BillRate
			bd.productID = rateResult.ProductID
		} else {
			// 3b. No resource — use unit_cost as pass-through.
			bd.billRate = act.GetUnitCost()
		}

		bd.billAmount = int64(float64(bd.billRate) * act.GetQuantity())
		billData = append(billData, bd)
	}

	// 4. Compute total amount.
	var totalAmount int64
	for _, bd := range billData {
		totalAmount += bd.billAmount
	}

	// 5. Auto-generate revenue name if not provided.
	revenueName := req.Name
	if revenueName == "" {
		names := make([]string, 0, len(jobNames))
		for name := range jobNames {
			names = append(names, name)
		}
		today := time.Now().Format("2006-01-02")
		if len(names) > 0 {
			revenueName = fmt.Sprintf("Invoice — %s (%s)", strings.Join(names, ", "), today)
		} else {
			revenueName = fmt.Sprintf("Invoice (%s)", today)
		}
	}

	// 6. Generate IDs.
	generateID := func() string {
		if uc.services.IDService != nil {
			return uc.services.IDService.GenerateID()
		}
		return fmt.Sprintf("id-%d", time.Now().UnixNano())
	}

	revenueID := generateID()
	result := &BillJobActivitiesResult{RevenueID: revenueID}

	// 7. Execute everything in a transaction.
	now := time.Now()
	dc := now.UnixMilli()
	dcs := now.Format(time.RFC3339)
	today := now.Format("2006-01-02")

	writeFunc := func(txCtx context.Context) error {
		// 7a. Stamp bill_rate and bill_amount on each activity.
		for _, bd := range billData {
			billRate := bd.billRate
			billAmount := bd.billAmount
			updatedActivity := &jobactivitypb.JobActivity{
				Id:         bd.activity.GetId(),
				BillRate:   &billRate,
				BillAmount: &billAmount,
			}
			if _, err := uc.repositories.JobActivity.UpdateJobActivity(txCtx, &jobactivitypb.UpdateJobActivityRequest{
				Data: updatedActivity,
			}); err != nil {
				return fmt.Errorf("BillJobActivities: stamp bill rate on activity %s: %w", bd.activity.GetId(), err)
			}
		}

		// 7b. Create Revenue header.
		revenueDate := today
		revenue := &revenuepb.Revenue{
			Id:                 revenueID,
			Name:               revenueName,
			ClientId:           req.ClientID,
			TotalAmount:        totalAmount,
			Currency:           req.Currency,
			RevenueDate:        &revenueDate,
			Status:             "draft",
			Active:             true,
			DateCreated:        &dc,
			DateCreatedString:  &dcs,
			DateModified:       &dc,
			DateModifiedString: &dcs,
		}
		if _, err := uc.repositories.Revenue.CreateRevenue(txCtx, &revenuepb.CreateRevenueRequest{
			Data: revenue,
		}); err != nil {
			return fmt.Errorf("BillJobActivities: create revenue: %w", err)
		}

		// 7c. Create RevenueLineItem + JobSettlement for each activity.
		for _, bd := range billData {
			lineItemID := generateID()

			description := bd.activity.GetDescription()
			if description == "" {
				description = fmt.Sprintf("Activity — %s", bd.activity.GetEntryDateString())
			}

			costPrice := bd.activity.GetUnitCost()
			activityID := bd.activity.GetId()

			lineItem := &revenuelineitempb.RevenueLineItem{
				Id:                 lineItemID,
				RevenueId:          revenueID,
				Description:        description,
				Quantity:           bd.activity.GetQuantity(),
				UnitPrice:          bd.billRate,
				TotalPrice:         bd.billAmount,
				CostPrice:          &costPrice,
				LineItemType:       "item",
				JobActivityId:      &activityID,
				Active:             true,
				DateCreated:        &dc,
				DateCreatedString:  &dcs,
				DateModified:       &dc,
				DateModifiedString: &dcs,
			}
			if bd.productID != "" {
				lineItem.ProductId = &bd.productID
			}

			if _, err := uc.repositories.RevenueLineItem.CreateRevenueLineItem(txCtx, &revenuelineitempb.CreateRevenueLineItemRequest{
				Data: lineItem,
			}); err != nil {
				return fmt.Errorf("BillJobActivities: create line item for activity %s: %w", bd.activity.GetId(), err)
			}

			// 7d. Create JobSettlement.
			settlementID := generateID()
			settlement := &jobsettlementpb.JobSettlement{
				Id:              settlementID,
				JobActivityId:   bd.activity.GetId(),
				TargetType:      jobsettlementpb.SettlementTargetType_SETTLEMENT_TARGET_TYPE_INVOICE_LINE,
				TargetId:        lineItemID,
				AllocatedAmount: bd.billAmount,
				Status:          jobsettlementpb.SettlementStatus_SETTLEMENT_STATUS_SETTLED,
				Active:          true,
				DateCreated:     &dc,
				DateCreatedString: &dcs,
			}
			if _, err := uc.repositories.JobSettlement.CreateJobSettlement(txCtx, &jobsettlementpb.CreateJobSettlementRequest{
				Data: settlement,
			}); err != nil {
				return fmt.Errorf("BillJobActivities: create settlement for activity %s: %w", bd.activity.GetId(), err)
			}
		}

		return nil
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		if err := uc.services.TransactionService.ExecuteInTransaction(ctx, writeFunc); err != nil {
			return nil, err
		}
	} else {
		if err := writeFunc(ctx); err != nil {
			return nil, err
		}
	}

	return result, nil
}
