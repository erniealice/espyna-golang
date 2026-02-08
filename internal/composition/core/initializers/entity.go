package initializers

import (
	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/usecases/entity"
	"github.com/erniealice/espyna-golang/internal/composition/providers/domain"

	// Entity sub-domain use cases
	adminUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/admin"
	clientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/client"
	clientAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/client_attribute"
	delegateUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/delegate"
	delegateAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/delegate_attribute"
	delegateClientUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/delegate_client"
	groupUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/group"
	groupAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/group_attribute"
	locationUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/location"
	locationAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/location_attribute"
	permissionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/permission"
	roleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/role"
	rolePermissionUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/role_permission"
	staffUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/staff"
	staffAttributeUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/staff_attribute"
	userUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/user"
	workspaceUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/workspace"
	workspaceUserUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/workspace_user"
	workspaceUserRoleUseCases "github.com/erniealice/espyna-golang/internal/application/usecases/entity/workspace_user_role"
)

// InitializeEntity creates all entity use cases from provider repositories.
// Use cases are only created when their primary repository is available (graceful degradation).
// This is composition logic - it wires infrastructure (providers) to application (use cases)
func InitializeEntity(
	repos *domain.EntityRepositories,
	authSvc ports.AuthorizationService,
	txSvc ports.TransactionService,
	i18nSvc ports.TranslationService,
	idSvc ports.IDService,
) (*entity.EntityUseCases, error) {
	svc := func() entityServices {
		return entityServices{authSvc, txSvc, i18nSvc, idSvc}
	}

	result := &entity.EntityUseCases{}

	if repos.Admin != nil {
		result.Admin = adminUseCases.NewUseCases(
			adminUseCases.AdminRepositories{Admin: repos.Admin},
			adminUseCases.AdminServices(svc()),
		)
	}

	if repos.Client != nil {
		result.Client = clientUseCases.NewUseCases(
			clientUseCases.ClientRepositories{
				Client: repos.Client,
				User:   repos.User,
			},
			clientUseCases.ClientServices(svc()),
		)
	}

	if repos.ClientAttribute != nil {
		result.ClientAttribute = clientAttributeUseCases.NewUseCases(
			clientAttributeUseCases.ClientAttributeRepositories{
				ClientAttribute: repos.ClientAttribute,
				Client:          repos.Client,
				Attribute:       repos.Attribute,
			},
			clientAttributeUseCases.ClientAttributeServices(svc()),
		)
	}

	if repos.Delegate != nil {
		result.Delegate = delegateUseCases.NewUseCases(
			delegateUseCases.DelegateRepositories{Delegate: repos.Delegate},
			delegateUseCases.DelegateServices(svc()),
		)
	}

	if repos.DelegateAttribute != nil {
		result.DelegateAttribute = delegateAttributeUseCases.NewUseCases(
			delegateAttributeUseCases.DelegateAttributeRepositories{
				DelegateAttribute: repos.DelegateAttribute,
				Delegate:          repos.Delegate,
				Attribute:         repos.Attribute,
			},
			delegateAttributeUseCases.DelegateAttributeServices(svc()),
		)
	}

	if repos.DelegateClient != nil {
		result.DelegateClient = delegateClientUseCases.NewUseCases(
			delegateClientUseCases.DelegateClientRepositories{
				DelegateClient: repos.DelegateClient,
				Delegate:       repos.Delegate,
				Client:         repos.Client,
			},
			delegateClientUseCases.DelegateClientServices(svc()),
		)
	}

	if repos.Group != nil {
		result.Group = groupUseCases.NewUseCases(
			groupUseCases.GroupRepositories{Group: repos.Group},
			groupUseCases.GroupServices(svc()),
		)
	}

	if repos.GroupAttribute != nil {
		result.GroupAttribute = groupAttributeUseCases.NewUseCases(
			groupAttributeUseCases.GroupAttributeRepositories{
				GroupAttribute: repos.GroupAttribute,
				Group:          repos.Group,
				Attribute:      repos.Attribute,
			},
			groupAttributeUseCases.GroupAttributeServices(svc()),
		)
	}

	if repos.Location != nil {
		result.Location = locationUseCases.NewUseCases(
			locationUseCases.LocationRepositories{Location: repos.Location},
			locationUseCases.LocationServices(svc()),
		)
	}

	if repos.LocationAttribute != nil {
		result.LocationAttribute = locationAttributeUseCases.NewUseCases(
			locationAttributeUseCases.LocationAttributeRepositories{
				LocationAttribute: repos.LocationAttribute,
				Location:          repos.Location,
				Attribute:         repos.Attribute,
			},
			locationAttributeUseCases.LocationAttributeServices(svc()),
		)
	}

	if repos.Permission != nil {
		result.Permission = permissionUseCases.NewUseCases(
			permissionUseCases.PermissionRepositories{Permission: repos.Permission},
			permissionUseCases.PermissionServices(svc()),
		)
	}

	if repos.Role != nil {
		result.Role = roleUseCases.NewUseCases(
			roleUseCases.RoleRepositories{Role: repos.Role},
			roleUseCases.RoleServices(svc()),
		)
	}

	if repos.RolePermission != nil {
		result.RolePermission = rolePermissionUseCases.NewUseCases(
			rolePermissionUseCases.RolePermissionRepositories{
				RolePermission: repos.RolePermission,
				Role:           repos.Role,
				Permission:     repos.Permission,
			},
			rolePermissionUseCases.RolePermissionServices(svc()),
		)
	}

	if repos.Staff != nil {
		result.Staff = staffUseCases.NewUseCases(
			staffUseCases.StaffRepositories{Staff: repos.Staff},
			staffUseCases.StaffServices(svc()),
		)
	}

	if repos.StaffAttribute != nil {
		result.StaffAttribute = staffAttributeUseCases.NewUseCases(
			staffAttributeUseCases.StaffAttributeRepositories{
				StaffAttribute: repos.StaffAttribute,
				Staff:          repos.Staff,
				Attribute:      repos.Attribute,
			},
			staffAttributeUseCases.StaffAttributeServices(svc()),
		)
	}

	if repos.User != nil {
		result.User = userUseCases.NewUseCases(
			userUseCases.UserRepositories{User: repos.User},
			userUseCases.UserServices(svc()),
		)
	}

	if repos.Workspace != nil {
		result.Workspace = workspaceUseCases.NewUseCases(
			workspaceUseCases.WorkspaceRepositories{Workspace: repos.Workspace},
			workspaceUseCases.WorkspaceServices(svc()),
		)
	}

	if repos.WorkspaceUser != nil {
		result.WorkspaceUser = workspaceUserUseCases.NewUseCases(
			workspaceUserUseCases.WorkspaceUserRepositories{
				WorkspaceUser: repos.WorkspaceUser,
				Workspace:     repos.Workspace,
				User:          repos.User,
			},
			workspaceUserUseCases.WorkspaceUserServices(svc()),
		)
	}

	if repos.WorkspaceUserRole != nil {
		result.WorkspaceUserRole = workspaceUserRoleUseCases.NewUseCases(
			workspaceUserRoleUseCases.WorkspaceUserRoleRepositories{
				WorkspaceUserRole: repos.WorkspaceUserRole,
				WorkspaceUser:     repos.WorkspaceUser,
				Role:              repos.Role,
			},
			workspaceUserRoleUseCases.WorkspaceUserRoleServices(svc()),
		)
	}

	return result, nil
}

// entityServices is a type alias to reduce repetition in InitializeEntity.
// All entity *Services structs share the same layout, so we can convert between them.
type entityServices struct {
	AuthorizationService ports.AuthorizationService
	TransactionService   ports.TransactionService
	TranslationService   ports.TranslationService
	IDService            ports.IDService
}
