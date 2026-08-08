//go:build postgresql

package entity

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/shared/identity"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	workspaceuserpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/workspace_user"
)

func TestWorkspaceUserRepositoryRejectsPreselectionIdentityBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()

	repo := NewPostgresWorkspaceUserRepository(&userSecurityFixture{}, "workspace_user")
	ctx := identity.WithRequestIdentity(context.Background(), &identity.RequestIdentity{UserID: "user-1"})

	tests := []struct {
		name string
		call func() error
	}{
		{name: "create", call: func() error {
			_, err := repo.CreateWorkspaceUser(ctx, &workspaceuserpb.CreateWorkspaceUserRequest{Data: &workspaceuserpb.WorkspaceUser{UserId: "user-2"}})
			return err
		}},
		{name: "read", call: func() error {
			_, err := repo.ReadWorkspaceUser(ctx, &workspaceuserpb.ReadWorkspaceUserRequest{Data: &workspaceuserpb.WorkspaceUser{Id: "membership-1"}})
			return err
		}},
		{name: "update", call: func() error {
			_, err := repo.UpdateWorkspaceUser(ctx, &workspaceuserpb.UpdateWorkspaceUserRequest{Data: &workspaceuserpb.WorkspaceUser{Id: "membership-1"}})
			return err
		}},
		{name: "delete", call: func() error {
			_, err := repo.DeleteWorkspaceUser(ctx, &workspaceuserpb.DeleteWorkspaceUserRequest{Data: &workspaceuserpb.WorkspaceUser{Id: "membership-1"}})
			return err
		}},
		{name: "list", call: func() error {
			_, err := repo.ListWorkspaceUsers(ctx, &workspaceuserpb.ListWorkspaceUsersRequest{})
			return err
		}},
		{name: "list page", call: func() error {
			_, err := repo.GetWorkspaceUserListPageData(ctx, &workspaceuserpb.GetWorkspaceUserListPageDataRequest{})
			return err
		}},
		{name: "item page", call: func() error {
			_, err := repo.GetWorkspaceUserItemPageData(ctx, &workspaceuserpb.GetWorkspaceUserItemPageDataRequest{WorkspaceUserId: "membership-1"})
			return err
		}},
		{name: "batch workspaces", call: func() error {
			_, err := repo.ListWorkspacesForUsers(ctx, &workspaceuserpb.ListWorkspacesForUsersRequest{})
			return err
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, identity.ErrWorkspaceNotSelected) {
				t.Fatalf("error = %v, want ErrWorkspaceNotSelected", err)
			}
		})
	}
}

func TestWorkspaceUserFiltersForScope(t *testing.T) {
	t.Parallel()

	matchingWorkspace := &commonpb.TypedFilter{
		Field: "workspace_id",
		FilterType: &commonpb.TypedFilter_StringFilter{StringFilter: &commonpb.StringFilter{
			Value:         "workspace-1",
			Operator:      commonpb.StringOperator_STRING_EQUALS,
			CaseSensitive: true,
		}},
	}
	statusFilter := &commonpb.TypedFilter{
		Field: "active",
		FilterType: &commonpb.TypedFilter_BooleanFilter{BooleanFilter: &commonpb.BooleanFilter{
			Value: true,
		}},
	}

	filtered, err := workspaceUserFiltersForScope(&commonpb.FilterRequest{
		Logic:   commonpb.FilterLogic_AND,
		Filters: []*commonpb.TypedFilter{matchingWorkspace, statusFilter},
	}, "workspace-1")
	if err != nil {
		t.Fatalf("matching workspace filter: %v", err)
	}
	if len(filtered.GetFilters()) != 1 || filtered.GetFilters()[0].GetField() != "active" {
		t.Fatalf("filtered request = %#v, want only active filter", filtered)
	}

	matchingWorkspace.GetStringFilter().Value = "workspace-2"
	if _, err := workspaceUserFiltersForScope(&commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{matchingWorkspace}}, "workspace-1"); err == nil {
		t.Fatal("mismatched workspace filter was accepted")
	}

	matchingWorkspace.GetStringFilter().Value = "workspace-1"
	matchingWorkspace.GetStringFilter().Operator = commonpb.StringOperator_STRING_NOT_EQUALS
	if _, err := workspaceUserFiltersForScope(&commonpb.FilterRequest{Filters: []*commonpb.TypedFilter{matchingWorkspace}}, "workspace-1"); err == nil {
		t.Fatal("non-equality workspace filter was accepted")
	}
}
