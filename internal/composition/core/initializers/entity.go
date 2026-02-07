package initializers

import (
	"leapfor.xyz/espyna/internal/application/ports"
	"leapfor.xyz/espyna/internal/application/usecases/entity"
	"leapfor.xyz/espyna/internal/composition/providers/domain"

	// Entity sub-domain use cases
	adminUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/admin"
	clientUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/client"
	clientAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/client_attribute"
	delegateUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/delegate"
	delegateAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/delegate_attribute"
	delegateClientUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/delegate_client"
	groupUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/group"
	groupAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/group_attribute"
	locationUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/location"
	locationAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/location_attribute"
	permissionUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/permission"
	roleUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/role"
	rolePermissionUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/role_permission"
	staffUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/staff"
	staffAttributeUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/staff_attribute"
	userUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/user"
	workspaceUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/workspace"
	workspaceUserUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/workspace_user"
	workspaceUserRoleUseCases "leapfor.xyz/espyna/internal/application/usecases/entity/workspace_user_role"
)

// InitializeEntity creates all entity use cases from provider repositories
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeEntity(
	repos *domain.EntityRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*entity.EntityUseCases, error) {
	// Create individual domain use cases with proper dependency injection
	adminUC := adminUseCases.NewUseCases(
		adminUseCases.AdminRepositories{Admin: repos.Admin},
		adminUseCases.AdminServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	clientUC := clientUseCases.NewUseCases(
		clientUseCases.ClientRepositories{
			Client: repos.Client,
			User:   repos.User,
		},
		clientUseCases.ClientServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	clientAttributeUC := clientAttributeUseCases.NewUseCases(
		clientAttributeUseCases.ClientAttributeRepositories{
			ClientAttribute: repos.ClientAttribute,
			Client:          repos.Client,
			Attribute:       repos.Attribute,
		},
		clientAttributeUseCases.ClientAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	delegateUC := delegateUseCases.NewUseCases(
		delegateUseCases.DelegateRepositories{Delegate: repos.Delegate},
		delegateUseCases.DelegateServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	delegateAttributeUC := delegateAttributeUseCases.NewUseCases(
		delegateAttributeUseCases.DelegateAttributeRepositories{
			DelegateAttribute: repos.DelegateAttribute,
			Delegate:          repos.Delegate,
			Attribute:         repos.Attribute,
		},
		delegateAttributeUseCases.DelegateAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	delegateClientUC := delegateClientUseCases.NewUseCases(
		delegateClientUseCases.DelegateClientRepositories{
			DelegateClient: repos.DelegateClient,
			Delegate:       repos.Delegate,
			Client:         repos.Client,
		},
		delegateClientUseCases.DelegateClientServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	groupUC := groupUseCases.NewUseCases(
		groupUseCases.GroupRepositories{Group: repos.Group},
		groupUseCases.GroupServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	groupAttributeUC := groupAttributeUseCases.NewUseCases(
		groupAttributeUseCases.GroupAttributeRepositories{
			GroupAttribute: repos.GroupAttribute,
			Group:          repos.Group,
			Attribute:      repos.Attribute,
		},
		groupAttributeUseCases.GroupAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	locationUC := locationUseCases.NewUseCases(
		locationUseCases.LocationRepositories{Location: repos.Location},
		locationUseCases.LocationServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	locationAttributeUC := locationAttributeUseCases.NewUseCases(
		locationAttributeUseCases.LocationAttributeRepositories{
			LocationAttribute: repos.LocationAttribute,
			Location:          repos.Location,
			Attribute:         repos.Attribute,
		},
		locationAttributeUseCases.LocationAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	permissionUC := permissionUseCases.NewUseCases(
		permissionUseCases.PermissionRepositories{Permission: repos.Permission},
		permissionUseCases.PermissionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	roleUC := roleUseCases.NewUseCases(
		roleUseCases.RoleRepositories{Role: repos.Role},
		roleUseCases.RoleServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	rolePermissionUC := rolePermissionUseCases.NewUseCases(
		rolePermissionUseCases.RolePermissionRepositories{
			RolePermission: repos.RolePermission,
			Role:           repos.Role,
			Permission:     repos.Permission,
		},
		rolePermissionUseCases.RolePermissionServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	staffUC := staffUseCases.NewUseCases(
		staffUseCases.StaffRepositories{Staff: repos.Staff},
		staffUseCases.StaffServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	staffAttributeUC := staffAttributeUseCases.NewUseCases(
		staffAttributeUseCases.StaffAttributeRepositories{
			StaffAttribute: repos.StaffAttribute,
			Staff:          repos.Staff,
			Attribute:      repos.Attribute,
		},
		staffAttributeUseCases.StaffAttributeServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	userUC := userUseCases.NewUseCases(
		userUseCases.UserRepositories{User: repos.User},
		userUseCases.UserServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	workspaceUC := workspaceUseCases.NewUseCases(
		workspaceUseCases.WorkspaceRepositories{Workspace: repos.Workspace},
		workspaceUseCases.WorkspaceServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	workspaceUserUC := workspaceUserUseCases.NewUseCases(
		workspaceUserUseCases.WorkspaceUserRepositories{
			WorkspaceUser: repos.WorkspaceUser,
			Workspace:     repos.Workspace,
			User:          repos.User,
		},
		workspaceUserUseCases.WorkspaceUserServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	workspaceUserRoleUC := workspaceUserRoleUseCases.NewUseCases(
		workspaceUserRoleUseCases.WorkspaceUserRoleRepositories{
			WorkspaceUserRole: repos.WorkspaceUserRole,
			WorkspaceUser:     repos.WorkspaceUser,
			Role:              repos.Role,
		},
		workspaceUserRoleUseCases.WorkspaceUserRoleServices{
			AuthorizationService: authSvc,
			TransactionService:   txSvc,
			TranslationService:   i18nSvc,
			IDService:            idSvc,
		},
	)

	// Note: Entity domain uses a different pattern - direct aggregation
	// Return aggregated entity use cases
	return &entity.EntityUseCases{
		Admin:             adminUC,
		Client:            clientUC,
		ClientAttribute:   clientAttributeUC,
		Delegate:          delegateUC,
		DelegateAttribute: delegateAttributeUC,
		DelegateClient:    delegateClientUC,
		Group:             groupUC,
		GroupAttribute:    groupAttributeUC,
		Location:          locationUC,
		LocationAttribute: locationAttributeUC,
		Permission:        permissionUC,
		Role:              roleUC,
		RolePermission:    rolePermissionUC,
		Staff:             staffUC,
		StaffAttribute:    staffAttributeUC,
		User:              userUC,
		Workspace:         workspaceUC,
		WorkspaceUser:     workspaceUserUC,
		WorkspaceUserRole: workspaceUserRoleUC,
	}, nil
}
