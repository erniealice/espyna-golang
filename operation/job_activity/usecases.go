package job_activity

import (
	"context"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/job_activity"
)

// JobActivityRepositories groups all repository dependencies
type JobActivityRepositories struct {
	JobActivity pb.JobActivityDomainServiceServer
}

// JobActivityServices groups all business service dependencies
type JobActivityServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// UseCases contains all job activity use cases
type UseCases struct {
	CreateJobActivity          *CreateJobActivityUseCase
	ReadJobActivity            *ReadJobActivityUseCase
	UpdateJobActivity          *UpdateJobActivityUseCase
	DeleteJobActivity          *DeleteJobActivityUseCase
	ListJobActivities          *ListJobActivitiesUseCase
	GetJobActivityListPageData *GetJobActivityListPageDataUseCase
	GetJobActivityItemPageData *GetJobActivityItemPageDataUseCase
	ListByJob                  *ListByJobUseCase
	GetJobActivityRollup       *GetJobActivityRollupUseCase
	SubmitForApproval          *SubmitForApprovalUseCase
	ApproveActivity            *ApproveActivityUseCase
	RejectActivity             *RejectActivityUseCase
	PostActivity               *PostActivityUseCase
	ReverseActivity            *ReverseActivityUseCase
}

// NewUseCases creates a new collection of job activity use cases
func NewUseCases(
	repositories JobActivityRepositories,
	services JobActivityServices,
) *UseCases {
	return &UseCases{
		CreateJobActivity: &CreateJobActivityUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
			IDSvc:   services.IDService,
		},
		ReadJobActivity: &ReadJobActivityUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
		},
		UpdateJobActivity: &UpdateJobActivityUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		DeleteJobActivity: &DeleteJobActivityUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		ListJobActivities: &ListJobActivitiesUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
		},
		GetJobActivityListPageData: &GetJobActivityListPageDataUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
		},
		GetJobActivityItemPageData: &GetJobActivityItemPageDataUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
		},
		ListByJob: &ListByJobUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
		},
		GetJobActivityRollup: &GetJobActivityRollupUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
		},
		SubmitForApproval: &SubmitForApprovalUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		ApproveActivity: &ApproveActivityUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		RejectActivity: &RejectActivityUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		PostActivity: &PostActivityUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
		},
		ReverseActivity: &ReverseActivityUseCase{
			Repo:    repositories.JobActivity,
			AuthSvc: services.AuthorizationService,
			TxSvc:   services.TransactionService,
			I18nSvc: services.TranslationService,
			IDSvc:   services.IDService,
		},
	}
}

// =============================================================================
// Create
// =============================================================================

// CreateJobActivityUseCase handles creating a new job activity
type CreateJobActivityUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
	IDSvc   ports.IDService
}

// Execute creates a new job activity
func (uc *CreateJobActivityUseCase) Execute(ctx context.Context, req *pb.CreateJobActivityRequest) (*pb.CreateJobActivityResponse, error) {
	if req.Data == nil {
		return nil, fmt.Errorf("job activity data is required")
	}

	// Generate ID if not provided
	if req.Data.Id == "" && uc.IDSvc != nil {
		req.Data.Id = uc.IDSvc.GenerateID()
	}

	// Set defaults
	if req.Data.ApprovalStatus == pb.ActivityApprovalStatus_ACTIVITY_APPROVAL_STATUS_UNSPECIFIED {
		req.Data.ApprovalStatus = pb.ActivityApprovalStatus_ACTIVITY_APPROVAL_STATUS_DRAFT
	}
	if req.Data.PostingStatus == pb.ActivityPostingStatus_ACTIVITY_POSTING_STATUS_UNSPECIFIED {
		req.Data.PostingStatus = pb.ActivityPostingStatus_ACTIVITY_POSTING_STATUS_UNPOSTED
	}
	if req.Data.BillableStatus == pb.BillableStatus_BILLABLE_STATUS_UNSPECIFIED {
		req.Data.BillableStatus = pb.BillableStatus_BILLABLE_STATUS_BILLABLE
	}

	return uc.Repo.CreateJobActivity(ctx, req)
}

// =============================================================================
// Read
// =============================================================================

// ReadJobActivityUseCase handles reading a single job activity
type ReadJobActivityUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute reads a job activity by ID
func (uc *ReadJobActivityUseCase) Execute(ctx context.Context, req *pb.ReadJobActivityRequest) (*pb.ReadJobActivityResponse, error) {
	return uc.Repo.ReadJobActivity(ctx, req)
}

// =============================================================================
// Update
// =============================================================================

// UpdateJobActivityUseCase handles updating a job activity
type UpdateJobActivityUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute updates a job activity
func (uc *UpdateJobActivityUseCase) Execute(ctx context.Context, req *pb.UpdateJobActivityRequest) (*pb.UpdateJobActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job activity ID is required")
	}
	return uc.Repo.UpdateJobActivity(ctx, req)
}

// =============================================================================
// Delete
// =============================================================================

// DeleteJobActivityUseCase handles deleting a job activity
type DeleteJobActivityUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute deletes a job activity (soft delete)
func (uc *DeleteJobActivityUseCase) Execute(ctx context.Context, req *pb.DeleteJobActivityRequest) (*pb.DeleteJobActivityResponse, error) {
	if req.Data == nil || req.Data.Id == "" {
		return nil, fmt.Errorf("job activity ID is required")
	}
	return uc.Repo.DeleteJobActivity(ctx, req)
}

// =============================================================================
// List
// =============================================================================

// ListJobActivitiesUseCase handles listing job activities
type ListJobActivitiesUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute lists job activities with optional filters
func (uc *ListJobActivitiesUseCase) Execute(ctx context.Context, req *pb.ListJobActivitiesRequest) (*pb.ListJobActivitiesResponse, error) {
	return uc.Repo.ListJobActivities(ctx, req)
}

// =============================================================================
// GetJobActivityListPageData
// =============================================================================

// GetJobActivityListPageDataUseCase handles paginated list page data
type GetJobActivityListPageDataUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute retrieves paginated job activity list page data
func (uc *GetJobActivityListPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobActivityListPageDataRequest) (*pb.GetJobActivityListPageDataResponse, error) {
	return uc.Repo.GetJobActivityListPageData(ctx, req)
}

// =============================================================================
// GetJobActivityItemPageData
// =============================================================================

// GetJobActivityItemPageDataUseCase handles single item page data
type GetJobActivityItemPageDataUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute retrieves a single job activity with all related data
func (uc *GetJobActivityItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetJobActivityItemPageDataRequest) (*pb.GetJobActivityItemPageDataResponse, error) {
	return uc.Repo.GetJobActivityItemPageData(ctx, req)
}

// =============================================================================
// ListByJob
// =============================================================================

// ListByJobUseCase handles listing activities for a specific job
type ListByJobUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute lists all activities for a given job
func (uc *ListByJobUseCase) Execute(ctx context.Context, req *pb.ListJobActivitiesByJobRequest) (*pb.ListJobActivitiesByJobResponse, error) {
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}
	return uc.Repo.ListByJob(ctx, req)
}

// =============================================================================
// GetJobActivityRollup
// =============================================================================

// GetJobActivityRollupUseCase handles aggregated cost rollup by entry type
type GetJobActivityRollupUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
}

// Execute returns aggregated costs grouped by entry_type for a job
func (uc *GetJobActivityRollupUseCase) Execute(ctx context.Context, req *pb.GetJobActivityRollupRequest) (*pb.GetJobActivityRollupResponse, error) {
	if req.JobId == "" {
		return nil, fmt.Errorf("job ID is required")
	}
	return uc.Repo.GetJobActivityRollup(ctx, req)
}

// =============================================================================
// SubmitForApproval
// =============================================================================

// SubmitForApprovalUseCase handles transitioning activity from DRAFT to SUBMITTED
type SubmitForApprovalUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute transitions activity from DRAFT to SUBMITTED
func (uc *SubmitForApprovalUseCase) Execute(ctx context.Context, req *pb.SubmitForApprovalRequest) (*pb.SubmitForApprovalResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	// Read current activity to validate status
	readResp, err := uc.Repo.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{
		Data: &pb.JobActivity{Id: req.ActivityId},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read activity: %w", err)
	}
	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("activity not found: %s", req.ActivityId)
	}

	activity := readResp.Data[0]
	if activity.ApprovalStatus != pb.ActivityApprovalStatus_ACTIVITY_APPROVAL_STATUS_DRAFT {
		return nil, fmt.Errorf("cannot submit activity: current status is %s, must be DRAFT", activity.ApprovalStatus.String())
	}

	return uc.Repo.SubmitForApproval(ctx, req)
}

// =============================================================================
// ApproveActivity
// =============================================================================

// ApproveActivityUseCase handles transitioning activity from SUBMITTED to APPROVED
type ApproveActivityUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute transitions activity from SUBMITTED to APPROVED
func (uc *ApproveActivityUseCase) Execute(ctx context.Context, req *pb.ApproveJobActivityRequest) (*pb.ApproveJobActivityResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	// Read current activity to validate status
	readResp, err := uc.Repo.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{
		Data: &pb.JobActivity{Id: req.ActivityId},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read activity: %w", err)
	}
	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("activity not found: %s", req.ActivityId)
	}

	activity := readResp.Data[0]
	if activity.ApprovalStatus != pb.ActivityApprovalStatus_ACTIVITY_APPROVAL_STATUS_SUBMITTED {
		return nil, fmt.Errorf("cannot approve activity: current status is %s, must be SUBMITTED", activity.ApprovalStatus.String())
	}

	return uc.Repo.ApproveActivity(ctx, req)
}

// =============================================================================
// RejectActivity
// =============================================================================

// RejectActivityUseCase handles transitioning activity from SUBMITTED to REJECTED
type RejectActivityUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute transitions activity from SUBMITTED to REJECTED with reason
func (uc *RejectActivityUseCase) Execute(ctx context.Context, req *pb.RejectJobActivityRequest) (*pb.RejectJobActivityResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}
	if req.Reason == "" {
		return nil, fmt.Errorf("rejection reason is required")
	}

	// Read current activity to validate status
	readResp, err := uc.Repo.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{
		Data: &pb.JobActivity{Id: req.ActivityId},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read activity: %w", err)
	}
	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("activity not found: %s", req.ActivityId)
	}

	activity := readResp.Data[0]
	if activity.ApprovalStatus != pb.ActivityApprovalStatus_ACTIVITY_APPROVAL_STATUS_SUBMITTED {
		return nil, fmt.Errorf("cannot reject activity: current status is %s, must be SUBMITTED", activity.ApprovalStatus.String())
	}

	return uc.Repo.RejectActivity(ctx, req)
}

// =============================================================================
// PostActivity
// =============================================================================

// PostActivityUseCase handles transitioning posting_status from UNPOSTED to POSTED
type PostActivityUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
}

// Execute transitions posting_status from UNPOSTED to POSTED
// This must run within a transaction as it creates settlement entries atomically
func (uc *PostActivityUseCase) Execute(ctx context.Context, req *pb.PostJobActivityRequest) (*pb.PostJobActivityResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}

	// Read current activity to validate statuses
	readResp, err := uc.Repo.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{
		Data: &pb.JobActivity{Id: req.ActivityId},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read activity: %w", err)
	}
	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("activity not found: %s", req.ActivityId)
	}

	activity := readResp.Data[0]

	// Must be APPROVED before posting
	if activity.ApprovalStatus != pb.ActivityApprovalStatus_ACTIVITY_APPROVAL_STATUS_APPROVED {
		return nil, fmt.Errorf("cannot post activity: approval status is %s, must be APPROVED", activity.ApprovalStatus.String())
	}
	// Must be UNPOSTED
	if activity.PostingStatus != pb.ActivityPostingStatus_ACTIVITY_POSTING_STATUS_UNPOSTED {
		return nil, fmt.Errorf("cannot post activity: posting status is %s, must be UNPOSTED", activity.PostingStatus.String())
	}

	// Execute posting within a transaction
	var resp *pb.PostJobActivityResponse
	if uc.TxSvc != nil && uc.TxSvc.SupportsTransactions() {
		err = uc.TxSvc.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			var txErr error
			resp, txErr = uc.Repo.PostActivity(txCtx, req)
			return txErr
		})
		if err != nil {
			return nil, fmt.Errorf("failed to post activity in transaction: %w", err)
		}
		return resp, nil
	}

	return uc.Repo.PostActivity(ctx, req)
}

// =============================================================================
// ReverseActivity
// =============================================================================

// ReverseActivityUseCase handles creating a reversal entry and marking original as REVERSED
type ReverseActivityUseCase struct {
	Repo    pb.JobActivityDomainServiceServer
	AuthSvc ports.AuthorizationService
	TxSvc   ports.TransactionService
	I18nSvc ports.TranslationService
	IDSvc   ports.IDService
}

// Execute creates a reversal entry and marks original as REVERSED
// This must run within a transaction for atomicity
func (uc *ReverseActivityUseCase) Execute(ctx context.Context, req *pb.ReverseJobActivityRequest) (*pb.ReverseJobActivityResponse, error) {
	if req.ActivityId == "" {
		return nil, fmt.Errorf("activity ID is required")
	}
	if req.Reason == "" {
		return nil, fmt.Errorf("reversal reason is required")
	}

	// Read current activity to validate status
	readResp, err := uc.Repo.ReadJobActivity(ctx, &pb.ReadJobActivityRequest{
		Data: &pb.JobActivity{Id: req.ActivityId},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read activity: %w", err)
	}
	if len(readResp.Data) == 0 {
		return nil, fmt.Errorf("activity not found: %s", req.ActivityId)
	}

	activity := readResp.Data[0]

	// Must be POSTED to reverse
	if activity.PostingStatus != pb.ActivityPostingStatus_ACTIVITY_POSTING_STATUS_POSTED {
		return nil, fmt.Errorf("cannot reverse activity: posting status is %s, must be POSTED", activity.PostingStatus.String())
	}

	// Execute reversal within a transaction
	var resp *pb.ReverseJobActivityResponse
	if uc.TxSvc != nil && uc.TxSvc.SupportsTransactions() {
		err = uc.TxSvc.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			// Create reversal entry (negative of original)
			reversalID := ""
			if uc.IDSvc != nil {
				reversalID = uc.IDSvc.GenerateID()
			}
			now := time.Now().Unix()
			reversalOfID := req.ActivityId
			description := fmt.Sprintf("Reversal of %s: %s", req.ActivityId, req.Reason)

			_, txErr := uc.Repo.CreateJobActivity(txCtx, &pb.CreateJobActivityRequest{
				Data: &pb.JobActivity{
					Id:             reversalID,
					JobId:          activity.JobId,
					JobTaskId:      activity.JobTaskId,
					EntryType:      activity.EntryType,
					Quantity:        -activity.Quantity,
					UnitCost:       activity.UnitCost,
					TotalCost:      -activity.TotalCost,
					Currency:       activity.Currency,
					Description:    &description,
					BillableStatus: activity.BillableStatus,
					ApprovalStatus: pb.ActivityApprovalStatus_ACTIVITY_APPROVAL_STATUS_APPROVED,
					PostingStatus:  pb.ActivityPostingStatus_ACTIVITY_POSTING_STATUS_POSTED,
					PostedBy:       &req.ReversedBy,
					DatePosted:     &now,
					ReversalOfId:   &reversalOfID,
					CreatedBy:      &req.ReversedBy,
					Active:         true,
				},
			})
			if txErr != nil {
				return fmt.Errorf("failed to create reversal entry: %w", txErr)
			}

			// Mark original as REVERSED
			resp, txErr = uc.Repo.ReverseActivity(txCtx, req)
			return txErr
		})
		if err != nil {
			return nil, fmt.Errorf("failed to reverse activity in transaction: %w", err)
		}
		return resp, nil
	}

	return uc.Repo.ReverseActivity(ctx, req)
}
