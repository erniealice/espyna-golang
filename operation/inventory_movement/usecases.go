package inventory_movement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/internal/application/usecases/authcheck"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/inventory_movement"
)

// --- Repositories & Services ---

// InventoryMovementRepositories groups all repository dependencies
type InventoryMovementRepositories struct {
	InventoryMovement pb.InventoryMovementDomainServiceServer
}

// InventoryMovementServices groups all business service dependencies
type InventoryMovementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

// --- UseCases Aggregate ---

// UseCases contains all inventory_movement-related use cases
type UseCases struct {
	CreateInventoryMovement          *CreateInventoryMovementUseCase
	ReadInventoryMovement            *ReadInventoryMovementUseCase
	UpdateInventoryMovement          *UpdateInventoryMovementUseCase
	DeleteInventoryMovement          *DeleteInventoryMovementUseCase
	ListInventoryMovements           *ListInventoryMovementsUseCase
	GetInventoryMovementListPageData *GetInventoryMovementListPageDataUseCase
	GetInventoryMovementItemPageData *GetInventoryMovementItemPageDataUseCase
	ListByJob                        *ListByJobUseCase
	ListByProduct                    *ListByProductUseCase
}

// NewUseCases creates a new collection of inventory_movement use cases
func NewUseCases(
	repositories InventoryMovementRepositories,
	services InventoryMovementServices,
) *UseCases {
	return &UseCases{
		CreateInventoryMovement: NewCreateInventoryMovementUseCase(
			CreateInventoryMovementRepositories{InventoryMovement: repositories.InventoryMovement},
			CreateInventoryMovementServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService, IDService: services.IDService},
		),
		ReadInventoryMovement: NewReadInventoryMovementUseCase(
			ReadInventoryMovementRepositories{InventoryMovement: repositories.InventoryMovement},
			ReadInventoryMovementServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		UpdateInventoryMovement: NewUpdateInventoryMovementUseCase(
			UpdateInventoryMovementRepositories{InventoryMovement: repositories.InventoryMovement},
			UpdateInventoryMovementServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		DeleteInventoryMovement: NewDeleteInventoryMovementUseCase(
			DeleteInventoryMovementRepositories{InventoryMovement: repositories.InventoryMovement},
			DeleteInventoryMovementServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		ListInventoryMovements: NewListInventoryMovementsUseCase(
			ListInventoryMovementsRepositories{InventoryMovement: repositories.InventoryMovement},
			ListInventoryMovementsServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		GetInventoryMovementListPageData: NewGetInventoryMovementListPageDataUseCase(
			GetInventoryMovementListPageDataRepositories{InventoryMovement: repositories.InventoryMovement},
			GetInventoryMovementListPageDataServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		GetInventoryMovementItemPageData: NewGetInventoryMovementItemPageDataUseCase(
			GetInventoryMovementItemPageDataRepositories{InventoryMovement: repositories.InventoryMovement},
			GetInventoryMovementItemPageDataServices{AuthorizationService: services.AuthorizationService, TransactionService: services.TransactionService, TranslationService: services.TranslationService},
		),
		ListByJob: NewListByJobUseCase(
			ListByJobRepositories{InventoryMovement: repositories.InventoryMovement},
			ListByJobServices{AuthorizationService: services.AuthorizationService, TranslationService: services.TranslationService},
		),
		ListByProduct: NewListByProductUseCase(
			ListByProductRepositories{InventoryMovement: repositories.InventoryMovement},
			ListByProductServices{AuthorizationService: services.AuthorizationService, TranslationService: services.TranslationService},
		),
	}
}

// ============================================================
// Create
// ============================================================

type CreateInventoryMovementRepositories struct {
	InventoryMovement pb.InventoryMovementDomainServiceServer
}

type CreateInventoryMovementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}

type CreateInventoryMovementUseCase struct {
	repositories CreateInventoryMovementRepositories
	services     CreateInventoryMovementServices
}

func NewCreateInventoryMovementUseCase(repos CreateInventoryMovementRepositories, svcs CreateInventoryMovementServices) *CreateInventoryMovementUseCase {
	return &CreateInventoryMovementUseCase{repositories: repos, services: svcs}
}

func (uc *CreateInventoryMovementUseCase) Execute(ctx context.Context, req *pb.CreateInventoryMovementRequest) (*pb.CreateInventoryMovementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"inventory_movement", ports.ActionCreate); err != nil {
		return nil, err
	}

	if req == nil || req.Data == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.data_required", "[ERR-DEFAULT] Inventory movement data is required"))
	}

	if err := uc.validateBusinessRules(ctx, req.Data); err != nil {
		return nil, err
	}

	enriched := uc.applyBusinessLogic(req.Data)

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.CreateInventoryMovementResponse
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

func (uc *CreateInventoryMovementUseCase) executeCore(ctx context.Context, data *pb.InventoryMovement) (*pb.CreateInventoryMovementResponse, error) {
	resp, err := uc.repositories.InventoryMovement.CreateInventoryMovement(ctx, &pb.CreateInventoryMovementRequest{Data: data})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.creation_failed", "[ERR-DEFAULT] Inventory movement creation failed"))
	}
	return resp, nil
}

func (uc *CreateInventoryMovementUseCase) applyBusinessLogic(movement *pb.InventoryMovement) *pb.InventoryMovement {
	now := time.Now()
	if movement.Id == "" {
		movement.Id = uc.services.IDService.GenerateID()
	}
	movement.Active = true
	movement.DateCreated = &[]int64{now.UnixMilli()}[0]
	movement.DateCreatedString = &[]string{now.Format(time.RFC3339)}[0]
	return movement
}

func (uc *CreateInventoryMovementUseCase) validateBusinessRules(ctx context.Context, m *pb.InventoryMovement) error {
	if m.WorkspaceId == nil || *m.WorkspaceId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.workspace_id_required", "[ERR-DEFAULT] Workspace ID is required"))
	}
	if m.ProductId == "" {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.product_id_required", "[ERR-DEFAULT] Product ID is required"))
	}
	if m.MovementType == enumspb.MovementType_MOVEMENT_TYPE_UNSPECIFIED {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.movement_type_required", "[ERR-DEFAULT] Movement type is required"))
	}
	if m.Quantity <= 0 {
		return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.quantity_positive", "[ERR-DEFAULT] Quantity must be positive"))
	}
	// Transfer movements must have both from and to locations
	if m.MovementType == enumspb.MovementType_MOVEMENT_TYPE_TRANSFER {
		if m.FromLocationId == nil || *m.FromLocationId == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.from_location_required", "[ERR-DEFAULT] From location is required for transfers"))
		}
		if m.ToLocationId == nil || *m.ToLocationId == "" {
			return errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.to_location_required", "[ERR-DEFAULT] To location is required for transfers"))
		}
	}
	return nil
}

// ============================================================
// Read
// ============================================================

type ReadInventoryMovementRepositories struct {
	InventoryMovement pb.InventoryMovementDomainServiceServer
}

type ReadInventoryMovementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ReadInventoryMovementUseCase struct {
	repositories ReadInventoryMovementRepositories
	services     ReadInventoryMovementServices
}

func NewReadInventoryMovementUseCase(repos ReadInventoryMovementRepositories, svcs ReadInventoryMovementServices) *ReadInventoryMovementUseCase {
	return &ReadInventoryMovementUseCase{repositories: repos, services: svcs}
}

func (uc *ReadInventoryMovementUseCase) Execute(ctx context.Context, req *pb.ReadInventoryMovementRequest) (*pb.ReadInventoryMovementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"inventory_movement", ports.ActionRead); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.id_required", "[ERR-DEFAULT] Inventory movement ID is required"))
	}

	resp, err := uc.repositories.InventoryMovement.ReadInventoryMovement(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.not_found", "[ERR-DEFAULT] Inventory movement not found"))
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.not_found", "[ERR-DEFAULT] Inventory movement not found"))
	}
	return resp, nil
}

// ============================================================
// Update
// ============================================================

type UpdateInventoryMovementRepositories struct {
	InventoryMovement pb.InventoryMovementDomainServiceServer
}

type UpdateInventoryMovementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type UpdateInventoryMovementUseCase struct {
	repositories UpdateInventoryMovementRepositories
	services     UpdateInventoryMovementServices
}

func NewUpdateInventoryMovementUseCase(repos UpdateInventoryMovementRepositories, svcs UpdateInventoryMovementServices) *UpdateInventoryMovementUseCase {
	return &UpdateInventoryMovementUseCase{repositories: repos, services: svcs}
}

func (uc *UpdateInventoryMovementUseCase) Execute(ctx context.Context, req *pb.UpdateInventoryMovementRequest) (*pb.UpdateInventoryMovementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"inventory_movement", ports.ActionUpdate); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.id_required", "[ERR-DEFAULT] Inventory movement ID is required"))
	}

	_, err := uc.repositories.InventoryMovement.ReadInventoryMovement(ctx, &pb.ReadInventoryMovementRequest{Data: &pb.InventoryMovement{Id: req.Data.Id}})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.not_found", "[ERR-DEFAULT] Inventory movement not found"))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.UpdateInventoryMovementResponse
		txErr := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.InventoryMovement.UpdateInventoryMovement(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if txErr != nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.update_failed", "[ERR-DEFAULT] Inventory movement update failed"))
		}
		return result, nil
	}

	resp, err := uc.repositories.InventoryMovement.UpdateInventoryMovement(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.update_failed", "[ERR-DEFAULT] Inventory movement update failed"))
	}
	return resp, nil
}

// ============================================================
// Delete
// ============================================================

type DeleteInventoryMovementRepositories struct {
	InventoryMovement pb.InventoryMovementDomainServiceServer
}

type DeleteInventoryMovementServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type DeleteInventoryMovementUseCase struct {
	repositories DeleteInventoryMovementRepositories
	services     DeleteInventoryMovementServices
}

func NewDeleteInventoryMovementUseCase(repos DeleteInventoryMovementRepositories, svcs DeleteInventoryMovementServices) *DeleteInventoryMovementUseCase {
	return &DeleteInventoryMovementUseCase{repositories: repos, services: svcs}
}

func (uc *DeleteInventoryMovementUseCase) Execute(ctx context.Context, req *pb.DeleteInventoryMovementRequest) (*pb.DeleteInventoryMovementResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"inventory_movement", ports.ActionDelete); err != nil {
		return nil, err
	}
	if req == nil || req.Data == nil || req.Data.Id == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.id_required", "[ERR-DEFAULT] Inventory movement ID is required"))
	}

	_, err := uc.repositories.InventoryMovement.ReadInventoryMovement(ctx, &pb.ReadInventoryMovementRequest{Data: &pb.InventoryMovement{Id: req.Data.Id}})
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.not_found", "[ERR-DEFAULT] Inventory movement not found"))
	}

	if uc.services.TransactionService != nil && uc.services.TransactionService.SupportsTransactions() {
		var result *pb.DeleteInventoryMovementResponse
		txErr := uc.services.TransactionService.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
			res, err := uc.repositories.InventoryMovement.DeleteInventoryMovement(txCtx, req)
			if err != nil {
				return err
			}
			result = res
			return nil
		})
		if txErr != nil {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.deletion_failed", "[ERR-DEFAULT] Inventory movement deletion failed"))
		}
		return result, nil
	}

	resp, err := uc.repositories.InventoryMovement.DeleteInventoryMovement(ctx, req)
	if err != nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.deletion_failed", "[ERR-DEFAULT] Inventory movement deletion failed"))
	}
	return resp, nil
}

// ============================================================
// List
// ============================================================

type ListInventoryMovementsRepositories struct {
	InventoryMovement pb.InventoryMovementDomainServiceServer
}

type ListInventoryMovementsServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type ListInventoryMovementsUseCase struct {
	repositories ListInventoryMovementsRepositories
	services     ListInventoryMovementsServices
}

func NewListInventoryMovementsUseCase(repos ListInventoryMovementsRepositories, svcs ListInventoryMovementsServices) *ListInventoryMovementsUseCase {
	return &ListInventoryMovementsUseCase{repositories: repos, services: svcs}
}

func (uc *ListInventoryMovementsUseCase) Execute(ctx context.Context, req *pb.ListInventoryMovementsRequest) (*pb.ListInventoryMovementsResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"inventory_movement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}

	resp, err := uc.repositories.InventoryMovement.ListInventoryMovements(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.list_failed", "inventory movement listing failed: %w"), err)
	}
	return resp, nil
}

// ============================================================
// GetListPageData
// ============================================================

type GetInventoryMovementListPageDataRepositories struct {
	InventoryMovement pb.InventoryMovementDomainServiceServer
}

type GetInventoryMovementListPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetInventoryMovementListPageDataUseCase struct {
	repositories GetInventoryMovementListPageDataRepositories
	services     GetInventoryMovementListPageDataServices
}

func NewGetInventoryMovementListPageDataUseCase(repos GetInventoryMovementListPageDataRepositories, svcs GetInventoryMovementListPageDataServices) *GetInventoryMovementListPageDataUseCase {
	return &GetInventoryMovementListPageDataUseCase{repositories: repos, services: svcs}
}

func (uc *GetInventoryMovementListPageDataUseCase) Execute(ctx context.Context, req *pb.GetInventoryMovementListPageDataRequest) (*pb.GetInventoryMovementListPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"inventory_movement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.request_required", "[ERR-DEFAULT] Request is required"))
	}
	if req.Pagination != nil {
		if req.Pagination.Limit < 0 || req.Pagination.Limit > 100 {
			return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.invalid_limit", "pagination limit must be between 1 and 100"))
		}
	}

	resp, err := uc.repositories.InventoryMovement.GetInventoryMovementListPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.list_page_data_failed", "failed to retrieve inventory movement list page data: %w"), err)
	}
	return resp, nil
}

// ============================================================
// GetItemPageData
// ============================================================

type GetInventoryMovementItemPageDataRepositories struct {
	InventoryMovement pb.InventoryMovementDomainServiceServer
}

type GetInventoryMovementItemPageDataServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
}

type GetInventoryMovementItemPageDataUseCase struct {
	repositories GetInventoryMovementItemPageDataRepositories
	services     GetInventoryMovementItemPageDataServices
}

func NewGetInventoryMovementItemPageDataUseCase(repos GetInventoryMovementItemPageDataRepositories, svcs GetInventoryMovementItemPageDataServices) *GetInventoryMovementItemPageDataUseCase {
	return &GetInventoryMovementItemPageDataUseCase{repositories: repos, services: svcs}
}

func (uc *GetInventoryMovementItemPageDataUseCase) Execute(ctx context.Context, req *pb.GetInventoryMovementItemPageDataRequest) (*pb.GetInventoryMovementItemPageDataResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"inventory_movement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.InventoryMovementId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.id_required", "[ERR-DEFAULT] Inventory movement ID is required"))
	}

	resp, err := uc.repositories.InventoryMovement.GetInventoryMovementItemPageData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.item_page_data_failed", "failed to retrieve inventory movement item page data: %w"), err)
	}
	return resp, nil
}

// ============================================================
// ListByJob (Custom)
// ============================================================

type ListByJobRepositories struct {
	InventoryMovement pb.InventoryMovementDomainServiceServer
}

type ListByJobServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

type ListByJobUseCase struct {
	repositories ListByJobRepositories
	services     ListByJobServices
}

func NewListByJobUseCase(repos ListByJobRepositories, svcs ListByJobServices) *ListByJobUseCase {
	return &ListByJobUseCase{repositories: repos, services: svcs}
}

func (uc *ListByJobUseCase) Execute(ctx context.Context, req *pb.ListInventoryMovementsByJobRequest) (*pb.ListInventoryMovementsByJobResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"inventory_movement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.JobId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.job_id_required", "[ERR-DEFAULT] Job ID is required"))
	}

	resp, err := uc.repositories.InventoryMovement.ListByJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.list_by_job_failed", "failed to list movements by job: %w"), err)
	}
	return resp, nil
}

// ============================================================
// ListByProduct (Custom)
// ============================================================

type ListByProductRepositories struct {
	InventoryMovement pb.InventoryMovementDomainServiceServer
}

type ListByProductServices struct {
	AuthorizationService ports.AuthorizationService
	TranslationService   ports.TranslationService
}

type ListByProductUseCase struct {
	repositories ListByProductRepositories
	services     ListByProductServices
}

func NewListByProductUseCase(repos ListByProductRepositories, svcs ListByProductServices) *ListByProductUseCase {
	return &ListByProductUseCase{repositories: repos, services: svcs}
}

func (uc *ListByProductUseCase) Execute(ctx context.Context, req *pb.ListInventoryMovementsByProductRequest) (*pb.ListInventoryMovementsByProductResponse, error) {
	if err := authcheck.Check(ctx, uc.services.AuthorizationService, uc.services.TranslationService,
		"inventory_movement", ports.ActionList); err != nil {
		return nil, err
	}
	if req == nil || req.ProductId == "" {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.validation.product_id_required", "[ERR-DEFAULT] Product ID is required"))
	}

	resp, err := uc.repositories.InventoryMovement.ListByProduct(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.TranslationService, "inventory_movement.errors.list_by_product_failed", "failed to list movements by product: %w"), err)
	}
	return resp, nil
}

// Ensure commonpb import is used
var _ *commonpb.PaginationRequest
