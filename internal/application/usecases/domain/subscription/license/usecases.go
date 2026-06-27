package license

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	licensehistory "github.com/erniealice/espyna-golang/internal/application/usecases/domain/subscription/license_history"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	licensepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license"
	licensehistorypb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/license_history"
	planpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/plan"
	subscriptionpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/subscription/subscription"
)

// LicenseRepositories groups all repository dependencies for license use cases
type LicenseRepositories struct {
	License        licensepb.LicenseDomainServiceServer               // Primary entity repository
	LicenseHistory licensehistorypb.LicenseHistoryDomainServiceServer // For creating history entries
	Subscription   subscriptionpb.SubscriptionDomainServiceServer     // For FK validation and assigned_count updates
	Client         clientpb.ClientDomainServiceServer                 // For assignee validation
	Plan           planpb.PlanDomainServiceServer                     // For plan-based license creation
}

// LicenseServices groups all business service dependencies for license use cases
type LicenseServices struct {
	Authorizer       ports.Authorizer             // RBAC and permissions
	Transactor       ports.Transactor             // Database transactions
	Translator       ports.Translator             // i18n error messages
	IDGenerator      ports.IDGenerator            // UUID generation for licenses
	ActionGatekeeper *actiongate.ActionGatekeeper // Action-level gate checks
}

// UseCases contains all license-related use cases
type UseCases struct {
	// Standard CRUD
	CreateLicense *CreateLicenseUseCase
	ReadLicense   *ReadLicenseUseCase
	UpdateLicense *UpdateLicenseUseCase
	DeleteLicense *DeleteLicenseUseCase
	ListLicenses  *ListLicensesUseCase

	// Page data
	GetLicenseListPageData *GetLicenseListPageDataUseCase
	GetLicenseItemPageData *GetLicenseItemPageDataUseCase

	// Bulk operations
	CreateLicensesFromPlan *CreateLicensesFromPlanUseCase

	// Assignment operations
	AssignLicense           *AssignLicenseUseCase
	RevokeLicenseAssignment *RevokeLicenseAssignmentUseCase
	ReassignLicense         *ReassignLicenseUseCase
	SuspendLicense          *SuspendLicenseUseCase
	ReactivateLicense       *ReactivateLicenseUseCase

	// Validation
	ValidateLicenseAccess *ValidateLicenseAccessUseCase
}

// NewUseCases creates a new collection of license use cases
func NewUseCases(
	repositories LicenseRepositories,
	services LicenseServices,
	licenseHistoryUseCases *licensehistory.UseCases,
) *UseCases {
	// Build individual grouped parameters for each use case

	// Create License
	createRepos := CreateLicenseRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
	}
	createServices := CreateLicenseServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		IDGenerator:      services.IDGenerator,
	}

	// Read License
	readRepos := ReadLicenseRepositories{
		License: repositories.License,
	}
	readServices := ReadLicenseServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// Update License
	updateRepos := UpdateLicenseRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
	}
	updateServices := UpdateLicenseServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// Delete License
	deleteRepos := DeleteLicenseRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
	}
	deleteServices := DeleteLicenseServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// List Licenses
	listRepos := ListLicensesRepositories{
		License: repositories.License,
	}
	listServices := ListLicensesServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// Get License List Page Data
	getListPageDataRepos := GetLicenseListPageDataRepositories{
		License: repositories.License,
	}
	getListPageDataServices := GetLicenseListPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// Get License Item Page Data
	getItemPageDataRepos := GetLicenseItemPageDataRepositories{
		License: repositories.License,
	}
	getItemPageDataServices := GetLicenseItemPageDataServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// Create Licenses From Plan
	createFromPlanRepos := CreateLicensesFromPlanRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
		Plan:         repositories.Plan,
	}
	createFromPlanServices := CreateLicensesFromPlanServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
		IDGenerator:      services.IDGenerator,
	}

	// Assign License
	assignRepos := AssignLicenseRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
		Client:       repositories.Client,
	}
	assignServices := AssignLicenseServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// Revoke License Assignment
	revokeRepos := RevokeLicenseAssignmentRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
	}
	revokeServices := RevokeLicenseAssignmentServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// Reassign License
	reassignRepos := ReassignLicenseRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
		Client:       repositories.Client,
	}
	reassignServices := ReassignLicenseServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// Suspend License
	suspendRepos := SuspendLicenseRepositories{
		License: repositories.License,
	}
	suspendServices := SuspendLicenseServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// Reactivate License
	reactivateRepos := ReactivateLicenseRepositories{
		License: repositories.License,
	}
	reactivateServices := ReactivateLicenseServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Authorizer:       services.Authorizer,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	// Validate License Access
	validateRepos := ValidateLicenseAccessRepositories{
		License: repositories.License,
	}
	validateServices := ValidateLicenseAccessServices{
		ActionGatekeeper: services.ActionGatekeeper,
		Transactor:       services.Transactor,
		Translator:       services.Translator,
	}

	return &UseCases{
		// Standard CRUD
		CreateLicense: NewCreateLicenseUseCase(createRepos, createServices, licenseHistoryUseCases.CreateLicenseHistory),
		ReadLicense:   NewReadLicenseUseCase(readRepos, readServices),
		UpdateLicense: NewUpdateLicenseUseCase(updateRepos, updateServices),
		DeleteLicense: NewDeleteLicenseUseCase(deleteRepos, deleteServices, licenseHistoryUseCases.CreateLicenseHistory),
		ListLicenses:  NewListLicensesUseCase(listRepos, listServices),

		// Page data
		GetLicenseListPageData: NewGetLicenseListPageDataUseCase(getListPageDataRepos, getListPageDataServices),
		GetLicenseItemPageData: NewGetLicenseItemPageDataUseCase(getItemPageDataRepos, getItemPageDataServices),

		// Bulk operations
		CreateLicensesFromPlan: NewCreateLicensesFromPlanUseCase(createFromPlanRepos, createFromPlanServices, licenseHistoryUseCases.CreateLicenseHistory),

		// Assignment operations
		AssignLicense:           NewAssignLicenseUseCase(assignRepos, assignServices, licenseHistoryUseCases.CreateLicenseHistory),
		RevokeLicenseAssignment: NewRevokeLicenseAssignmentUseCase(revokeRepos, revokeServices, licenseHistoryUseCases.CreateLicenseHistory),
		ReassignLicense:         NewReassignLicenseUseCase(reassignRepos, reassignServices, licenseHistoryUseCases.CreateLicenseHistory),
		SuspendLicense:          NewSuspendLicenseUseCase(suspendRepos, suspendServices, licenseHistoryUseCases.CreateLicenseHistory),
		ReactivateLicense:       NewReactivateLicenseUseCase(reactivateRepos, reactivateServices, licenseHistoryUseCases.CreateLicenseHistory),

		// Validation
		ValidateLicenseAccess: NewValidateLicenseAccessUseCase(validateRepos, validateServices),
	}
}
