package license

import (
	"leapfor.xyz/espyna/internal/application/ports"
	licensehistory "leapfor.xyz/espyna/internal/application/usecases/subscription/license_history"
	clientpb "leapfor.xyz/esqyma/golang/v1/domain/entity/client"
	licensepb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license"
	licensehistorypb "leapfor.xyz/esqyma/golang/v1/domain/subscription/license_history"
	planpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/plan"
	subscriptionpb "leapfor.xyz/esqyma/golang/v1/domain/subscription/subscription"
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
	AuthorizationService ports.AuthorizationService // RBAC and permissions
	TransactionService   ports.TransactionService   // Database transactions
	TranslationService   ports.TranslationService   // i18n error messages
	IDService            ports.IDService            // UUID generation for licenses
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
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	// Read License
	readRepos := ReadLicenseRepositories{
		License: repositories.License,
	}
	readServices := ReadLicenseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	// Update License
	updateRepos := UpdateLicenseRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
	}
	updateServices := UpdateLicenseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	// Delete License
	deleteRepos := DeleteLicenseRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
	}
	deleteServices := DeleteLicenseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	// List Licenses
	listRepos := ListLicensesRepositories{
		License: repositories.License,
	}
	listServices := ListLicensesServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	// Get License List Page Data
	getListPageDataRepos := GetLicenseListPageDataRepositories{
		License: repositories.License,
	}
	getListPageDataServices := GetLicenseListPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	// Get License Item Page Data
	getItemPageDataRepos := GetLicenseItemPageDataRepositories{
		License: repositories.License,
	}
	getItemPageDataServices := GetLicenseItemPageDataServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
	}

	// Create Licenses From Plan
	createFromPlanRepos := CreateLicensesFromPlanRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
		Plan:         repositories.Plan,
	}
	createFromPlanServices := CreateLicensesFromPlanServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
		IDService:            services.IDService,
	}

	// Assign License
	assignRepos := AssignLicenseRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
		Client:       repositories.Client,
	}
	assignServices := AssignLicenseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	// Revoke License Assignment
	revokeRepos := RevokeLicenseAssignmentRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
	}
	revokeServices := RevokeLicenseAssignmentServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	// Reassign License
	reassignRepos := ReassignLicenseRepositories{
		License:      repositories.License,
		Subscription: repositories.Subscription,
		Client:       repositories.Client,
	}
	reassignServices := ReassignLicenseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	// Suspend License
	suspendRepos := SuspendLicenseRepositories{
		License: repositories.License,
	}
	suspendServices := SuspendLicenseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	// Reactivate License
	reactivateRepos := ReactivateLicenseRepositories{
		License: repositories.License,
	}
	reactivateServices := ReactivateLicenseServices{
		AuthorizationService: services.AuthorizationService,
		TransactionService:   services.TransactionService,
		TranslationService:   services.TranslationService,
	}

	// Validate License Access
	validateRepos := ValidateLicenseAccessRepositories{
		License: repositories.License,
	}
	validateServices := ValidateLicenseAccessServices{
		TransactionService: services.TransactionService,
		TranslationService: services.TranslationService,
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
