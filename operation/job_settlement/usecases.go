package job_settlement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	jobactivitypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_settlement"
)

// --- Repositories & Services ---

// JobSettlementRepositories groups all repository dependencies
type JobSettlementRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
	JobActivity   jobactivitypb.JobActivityDomainServiceServer
}

// JobSettlementServices groups all business service dependencies
type JobSettlementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// --- UseCases Aggregate ---

// UseCases contains all job_settlement-related use cases
type UseCases struct {
	CreateJobSettlement          *CreateJobSettlementUseCase
	ReadJobSettlement            *ReadJobSettlementUseCase
	UpdateJobSettlement          *UpdateJobSettlementUseCase
	DeleteJobSettlement          *DeleteJobSettlementUseCase
	ListJobSettlements           *ListJobSettlementsUseCase
	GetJobSettlementListPageData *GetJobSettlementListPageDataUseCase
	GetJobSettlementItemPageData *GetJobSettlementItemPageDataUseCase
	ListByActivity               *ListByActivityUseCase
	ListByTarget                 *ListByTargetUseCase
	GetSettlementSummary         *GetSettlementSummaryUseCase
}

// NewUseCases creates a new collection of job_settlement use cases
func NewUseCases(
	repositories JobSettlementRepositories,
	services JobSettlementServices,
) *UseCases {
	return &UseCases{
		CreateJobSettlement: NewCreateJobSettlementUseCase(
			CreateJobSettlementRepositories{JobSettlement: repositories.JobSettlement, JobActivity: repositories.JobActivity},
			CreateJobSettlementServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService, IDService: services.IDService},
		),
		ReadJobSettlement: NewReadJobSettlementUseCase(
			ReadJobSettlementRepositories{JobSettlement: repositories.JobSettlement},
			ReadJobSettlementServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		UpdateJobSettlement: NewUpdateJobSettlementUseCase(
			UpdateJobSettlementRepositories{JobSettlement: repositories.JobSettlement},
			UpdateJobSettlementServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		DeleteJobSettlement: NewDeleteJobSettlementUseCase(
			DeleteJobSettlementRepositories{JobSettlement: repositories.JobSettlement},
			DeleteJobSettlementServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		ListJobSettlements: NewListJobSettlementsUseCase(
			ListJobSettlementsRepositories{JobSettlement: repositories.JobSettlement},
			ListJobSettlementsServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		GetJobSettlementListPageData: NewGetJobSettlementListPageDataUseCase(
			GetJobSettlementListPageDataRepositories{JobSettlement: repositories.JobSettlement},
			GetJobSettlementListPageDataServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		GetJobSettlementItemPageData: NewGetJobSettlementItemPageDataUseCase(
			GetJobSettlementItemPageDataRepositories{JobSettlement: repositories.JobSettlement},
			GetJobSettlementItemPageDataServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		ListByActivity: NewListByActivityUseCase(
			ListByActivityRepositories{JobSettlement: repositories.JobSettlement},
			ListByActivityServices{AuthorizationService: services.AuthorizationService, TranslationService: services.TranslationService},
		),
		ListByTarget: NewListByTargetUseCase(
			ListByTargetRepositories{JobSettlement: repositories.JobSettlement},
			ListByTargetServices{AuthorizationService: services.AuthorizationService, TranslationService: services.TranslationService},
		),
		GetSettlementSummary: NewGetSettlementSummaryUseCase(
			GetSettlementSummaryRepositories{JobSettlement: repositories.JobSettlement},
			GetSettlementSummaryServices{AuthorizationService: services.AuthorizationService, TranslationService: services.TranslationService},
		),
	}
}

// ============================================================
// Create
// ============================================================

type CreateJobSettlementRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
	JobActivity   jobactivitypb.JobActivityDomainServiceServer
}

type CreateJobSettlementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

type CreateJobSettlementUseCase struct {
	repositories CreateJobSettlementRepositories
	services     CreateJobSettlementServices
}

func NewCreateJobSettlementUseCase(repos CreateJobSettlementRepositories, svcs CreateJobSettlementServices) *CreateJobSettlementUseCase {
	return &CreateJobSettlementUseCase{repositories: repos, services: svcs}
}

func (uc *CreateJobSettlementUseCase) Execute(ctx context.Context, req *pb.CreateJobSettlementRequest) (*pb.CreateJobSettlementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_settlement", ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.data_required", "[ERR-DEFAULT] Job settlement data is required"))
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	// Validate allocation sum: existing allocations for this activity + new must not exceed activity total_cost
	if err := uc.validateAllocationSum(ctx, req.Data); err != nil {
		return nil, err
	}

	enriched := uc.applyBusinessLogic(req.Data)

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.CreateJobSettlementResponse
		err := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.executeCore(txCtx, enriched)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return uc.executeCore(ctx, enriched)
}

func (uc *CreateJobSettlementUseCase) executeCore(ctx context.Context, data *pb.JobSettlement) (*pb.CreateJobSettlementResponse, error) {
	resp, err := uc.repositories.JobSettlement.CreateJobSettlement(ctx, &pb.CreateJobSettlementRequest{Data: data})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.creation_failed", "[ERR-DEFAULT] Job settlement creation failed"))
	}
	return resp, nil
}

func (uc *CreateJobSettlementUseCase) applyBusinessLogic(settlement *pb.JobSettlement) *pb.JobSettlement {
	now := time.Now()
	if settlement.Id == "" {
		settlement.Id = uc.services.IDService.GenerateID()
	}
	settlement.Active = true
	settlement.DateCreated = &[]int64{now.UnixMilli()}[0]
	settlement.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	if settlement.Status == pb.SettlementStatus_SETTLEMENT_STATUS_UNSPECIFIED {
		settlement.Status = pb.SettlementStatus_SETTLEMENT_STATUS_PENDING
	}
	return settlement
}

func (uc *CreateJobSettlementUseCase) validateBusinessRules(ctx context.Context, s *pb.JobSettlement) error {
	if s.JobActivityId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.activity_id_required", "[ERR-DEFAULT] Job activity ID is required"))
	}
	if s.TargetType == pb.SettlementTargetType_SETTLEMENT_TARGET_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.target_type_required", "[ERR-DEFAULT] Settlement target type is required"))
	}
	if s.AllocatedAmount <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.amount_positive", "[ERR-DEFAULT] Allocated amount must be positive"))
	}
	return nil
}

// validateAllocationSum checks that sum of allocated_amounts for this activity does not exceed the activity's total_cost
func (uc *CreateJobSettlementUseCase) validateAllocationSum(ctx context.Context, s *pb.JobSettlement) error {
	// Fetch existing settlements for this activity
	existingResp, err := uc.repositories.JobSettlement.ListByActivity(ctx, &pb.ListJobSettlementsByActivityRequest{
		JobActivityId: s.JobActivityId,
	})
	if err != nil {
		return fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.allocation_check_failed", "[ERR-DEFAULT] Failed to check existing allocations: %w"), err)
	}

	var existingSum float64
	if existingResp != nil {
		for _, existing := range existingResp.JobSettlements {
			if existing.Active && existing.Status != pb.SettlementStatus_SETTLEMENT_STATUS_REVERSED {
				existingSum += existing.AllocatedAmount
			}
		}
	}

	// Fetch the activity to get total_cost
	if uc.repositories.JobActivity != nil {
		activityResp, err := uc.repositories.JobActivity.ReadJobActivity(ctx, &jobactivitypb.ReadJobActivityRequest{
			Data: &jobactivitypb.JobActivity{Id: s.JobActivityId},
		})
		if err != nil {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.activity_not_found", "[ERR-DEFAULT] Job activity not found"))
		}
		if activityResp != nil && len(activityResp.Data) > 0 {
			totalCost := activityResp.Data[0].TotalCost
			if existingSum+s.AllocatedAmount > totalCost {
				return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.allocation_exceeds_total", "[ERR-DEFAULT] Allocation sum exceeds activity total cost"))
			}
		}
	}

	return nil
}

// ============================================================
// Read
// ============================================================

type ReadJobSettlementRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
}

type ReadJobSettlementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ReadJobSettlementUseCase struct {
	repositories ReadJobSettlementRepositories
	services     ReadJobSettlementServices
}

func NewReadJobSettlementUseCase(repos ReadJobSettlementRepositories, svcs ReadJobSettlementServices) *ReadJobSettlementUseCase {
	return &ReadJobSettlementUseCase{repositories: repos, services: svcs}
}

func (uc *ReadJobSettlementUseCase) Execute(ctx context.Context, req *pb.ReadJobSettlementRequest) (*pb.ReadJobSettlementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_settlement", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.id_required", "[ERR-DEFAULT] Job settlement ID is required"))
	}

	resp, err := uc.repositories.JobSettlement.ReadJobSettlement(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.not_found", "[ERR-DEFAULT] Job settlement not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.not_found", "[ERR-DEFAULT] Job settlement not found"))
	}
	return resp, nil
}

// ============================================================
// Update
// ============================================================

type UpdateJobSettlementRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
}

type UpdateJobSettlementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type UpdateJobSettlementUseCase struct {
	repositories UpdateJobSettlementRepositories
	services     UpdateJobSettlementServices
}

func NewUpdateJobSettlementUseCase(repos UpdateJobSettlementRepositories, svcs UpdateJobSettlementServices) *UpdateJobSettlementUseCase {
	return &UpdateJobSettlementUseCase{repositories: repos, services: svcs}
}

func (uc *UpdateJobSettlementUseCase) Execute(ctx context.Context, req *pb.UpdateJobSettlementRequest) (*pb.UpdateJobSettlementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_settlement", ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.id_required", "[ERR-DEFAULT] Job settlement ID is required"))
	}

	// Check existence
	_, err := uc.repositories.JobSettlement.ReadJobSettlement(ctx, &pb.ReadJobSettlementRequest{Data: &pb.JobSettlement{Id: req.Data.Id}})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.not_found", "[ERR-DEFAULT] Job settlement not found"))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.UpdateJobSettlementResponse
		txErr := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.JobSettlement.UpdateJobSettlement(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if txErr != nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.update_failed", "[ERR-DEFAULT] Job settlement update failed"))
		}
		return result, nil
	}

	resp, err := uc.repositories.JobSettlement.UpdateJobSettlement(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.update_failed", "[ERR-DEFAULT] Job settlement update failed"))
	}
	return resp, nil
}

// ============================================================
// Delete
// ============================================================

type DeleteJobSettlementRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
}

type DeleteJobSettlementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type DeleteJobSettlementUseCase struct {
	repositories DeleteJobSettlementRepositories
	services     DeleteJobSettlementServices
}

func NewDeleteJobSettlementUseCase(repos DeleteJobSettlementRepositories, svcs DeleteJobSettlementServices) *DeleteJobSettlementUseCase {
	return &DeleteJobSettlementUseCase{repositories: repos, services: svcs}
}

func (uc *DeleteJobSettlementUseCase) Execute(ctx context.Context, req *pb.DeleteJobSettlementRequest) (*pb.DeleteJobSettlementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_settlement", ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.id_required", "[ERR-DEFAULT] Job settlement ID is required"))
	}

	_, err := uc.repositories.JobSettlement.ReadJobSettlement(ctx, &pb.ReadJobSettlementRequest{Data: &pb.JobSettlement{Id: req.Data.Id}})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.not_found", "[ERR-DEFAULT] Job settlement not found"))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.DeleteJobSettlementResponse
		txErr := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.JobSettlement.DeleteJobSettlement(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if txErr != nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.deletion_failed", "[ERR-DEFAULT] Job settlement deletion failed"))
		}
		return result, nil
	}

	resp, err := uc.repositories.JobSettlement.DeleteJobSettlement(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.deletion_failed", "[ERR-DEFAULT] Job settlement deletion failed"))
	}
	return resp, nil
}

// ============================================================
// List
// ============================================================

type ListJobSettlementsRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
}

type ListJobSettlementsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ListJobSettlementsUseCase struct {
	repositories ListJobSettlementsRepositories
	services     ListJobSettlementsServices
}

func NewListJobSettlementsUseCase(repos ListJobSettlementsRepositories, svcs ListJobSettlementsServices) *ListJobSettlementsUseCase {
	return &ListJobSettlementsUseCase{repositories: repos, services: svcs}
}

func (uc *ListJobSettlementsUseCase) Execute(ctx context.Context, req *pb.ListJobSettlementsRequest) (*pb.ListJobSettlementsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_settlement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	resp, err := uc.repositories.JobSettlement.ListJobSettlements(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.list_failed", "job settlement listing failed: %w"), err)
	}
	return resp, nil
}

// ============================================================
// GetListPageData
// ============================================================

type GetJobSettlementListPageDataRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
}

type GetJobSettlementListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetJobSettlementListPageDataUseCase struct {
	repositories GetJobSettlementListPageDataRepositories
	services     GetJobSettlementListPageDataServices
}

func NewGetJobSettlementListPageDataUseCase(repos GetJobSettlementListPageDataRepositories, svcs GetJobSettlementListPageDataServices) *GetJobSettlementListPageDataUseCase {
	return &GetJobSettlementListPageDataUseCase{repositories: repos, services: svcs}
}

func (uc *GetJobSettlementListPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobSettlementListPageDataRequest) (*pb.GetJobSettlementListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_settlement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 || req.Pagination.Limit > 100 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.invalid_limit", "pagination limit must be between 1 and 100"))
		}
	}

	resp, err := uc.repositories.JobSettlement.GetJobSettlementListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.list_page_data_failed", "failed to retrieve job settlement list page data: %w"), err)
	}
	return resp, nil
}

// ============================================================
// GetItemPageData
// ============================================================

type GetJobSettlementItemPageDataRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
}

type GetJobSettlementItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetJobSettlementItemPageDataUseCase struct {
	repositories GetJobSettlementItemPageDataRepositories
	services     GetJobSettlementItemPageDataServices
}

func NewGetJobSettlementItemPageDataUseCase(repos GetJobSettlementItemPageDataRepositories, svcs GetJobSettlementItemPageDataServices) *GetJobSettlementItemPageDataUseCase {
	return &GetJobSettlementItemPageDataUseCase{repositories: repos, services: svcs}
}

func (uc *GetJobSettlementItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobSettlementItemPageDataRequest) (*pb.GetJobSettlementItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_settlement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.JobSettlementId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.id_required", "[ERR-DEFAULT] Job settlement ID is required"))
	}

	resp, err := uc.repositories.JobSettlement.GetJobSettlementItemPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.item_page_data_failed", "failed to retrieve job settlement item page data: %w"), err)
	}
	return resp, nil
}

// ============================================================
// ListByActivity (Custom)
// ============================================================

type ListByActivityRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
}

type ListByActivityServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

type ListByActivityUseCase struct {
	repositories ListByActivityRepositories
	services     ListByActivityServices
}

func NewListByActivityUseCase(repos ListByActivityRepositories, svcs ListByActivityServices) *ListByActivityUseCase {
	return &ListByActivityUseCase{repositories: repos, services: svcs}
}

func (uc *ListByActivityUseCase) Execute(ctx context.Context, req *pb.ListJobSettlementsByActivityRequest) (*pb.ListJobSettlementsByActivityResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_settlement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.JobActivityId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.activity_id_required", "[ERR-DEFAULT] Job activity ID is required"))
	}

	resp, err := uc.repositories.JobSettlement.ListByActivity(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.list_by_activity_failed", "failed to list settlements by activity: %w"), err)
	}
	return resp, nil
}

// ============================================================
// ListByTarget (Custom)
// ============================================================

type ListByTargetRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
}

type ListByTargetServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

type ListByTargetUseCase struct {
	repositories ListByTargetRepositories
	services     ListByTargetServices
}

func NewListByTargetUseCase(repos ListByTargetRepositories, svcs ListByTargetServices) *ListByTargetUseCase {
	return &ListByTargetUseCase{repositories: repos, services: svcs}
}

func (uc *ListByTargetUseCase) Execute(ctx context.Context, req *pb.ListJobSettlementsByTargetRequest) (*pb.ListJobSettlementsByTargetResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_settlement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.TargetId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.target_id_required", "[ERR-DEFAULT] Target ID is required"))
	}
	if req.TargetType == pb.SettlementTargetType_SETTLEMENT_TARGET_TYPE_UNSPECIFIED {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.target_type_required", "[ERR-DEFAULT] Target type is required"))
	}

	resp, err := uc.repositories.JobSettlement.ListByTarget(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.list_by_target_failed", "failed to list settlements by target: %w"), err)
	}
	return resp, nil
}

// ============================================================
// GetSettlementSummary (Custom)
// ============================================================

type GetSettlementSummaryRepositories struct {
	JobSettlement pb.JobSettlementDomainServiceServer
}

type GetSettlementSummaryServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

type GetSettlementSummaryUseCase struct {
	repositories GetSettlementSummaryRepositories
	services     GetSettlementSummaryServices
}

func NewGetSettlementSummaryUseCase(repos GetSettlementSummaryRepositories, svcs GetSettlementSummaryServices) *GetSettlementSummaryUseCase {
	return &GetSettlementSummaryUseCase{repositories: repos, services: svcs}
}

func (uc *GetSettlementSummaryUseCase) Execute(ctx context.Context, req *pb.GetSettlementSummaryRequest) (*pb.GetSettlementSummaryResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"job_settlement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.JobId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.validation.job_id_required", "[ERR-DEFAULT] Job ID is required"))
	}

	resp, err := uc.repositories.JobSettlement.GetSettlementSummary(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "job_settlement.errors.summary_failed", "failed to retrieve settlement summary: %w"), err)
	}
	return resp, nil
}

// Ensure commonpb import is used
var _ *commonpb.PaginationRequest
