// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testProjectRoleAssignmentId = "145-1"
	testProjectId               = "0-1"
	testGroupName               = "developers"
)

func projectScopedAssignment(holder youtrack.Holder) *youtrack.AssignedRole {
	return &youtrack.AssignedRole{
		Id:     testProjectRoleAssignmentId,
		Role:   youtrack.Role{Id: testRoleId, Name: testRoleName},
		Scope:  youtrack.Scope{Type: projectScopeType, Project: &youtrack.Project{ID: testProjectId}},
		Holder: holder,
	}
}

func TestMapAPIToProjectModel(t *testing.T) {
	userHolder := youtrack.Holder{Id: testHolderId, Login: testHolderLogin, Name: testHolderName, Type: assignedRoleHolderTypeUser}
	userWithoutLogin := youtrack.Holder{Id: testHolderId, Name: testHolderName, Type: assignedRoleHolderTypeUser}
	groupHolder := youtrack.Holder{Id: testHolderId, Name: testGroupName, Type: assignedRoleHolderTypeGroup}

	tests := []struct {
		name      string
		holder    youtrack.Holder
		current   types.String
		wantLogin string
		wantType  string
	}{
		{
			name:      "user holder takes the login the API reports",
			holder:    userHolder,
			current:   types.StringNull(),
			wantLogin: testHolderLogin,
			wantType:  holderTypeUser,
		},
		{
			name:      "user holder keeps the configured login when it only differs in case",
			holder:    userHolder,
			current:   types.StringValue("John.Doe"),
			wantLogin: "John.Doe",
			wantType:  holderTypeUser,
		},
		{
			name:      "user holder takes the API login when it differs from the configured one",
			holder:    userHolder,
			current:   types.StringValue("jane.doe"),
			wantLogin: testHolderLogin,
			wantType:  holderTypeUser,
		},
		{
			name:      "user holder without a login falls back to an empty login after an import",
			holder:    userWithoutLogin,
			current:   types.StringNull(),
			wantLogin: "",
			wantType:  holderTypeUser,
		},
		{
			name:      "user holder without a login keeps the configured login",
			holder:    userWithoutLogin,
			current:   types.StringValue(testHolderLogin),
			wantLogin: testHolderLogin,
			wantType:  holderTypeUser,
		},
		{
			name:      "group holder has no login, so its name stands in for it after an import",
			holder:    groupHolder,
			current:   types.StringNull(),
			wantLogin: testGroupName,
			wantType:  holderTypeGroup,
		},
		{
			name:      "group holder keeps the configured name when it only differs in case",
			holder:    groupHolder,
			current:   types.StringValue("Developers"),
			wantLogin: "Developers",
			wantType:  holderTypeGroup,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := projectRoleAssignmentResourceModel{HolderLogin: tt.current}
			mapAPIToProjectModel(projectScopedAssignment(tt.holder), &model)

			helpers.AssertFieldEqual(t, "id", model.Id.ValueString(), testProjectRoleAssignmentId)
			helpers.AssertFieldEqual(t, "project_id", model.ProjectId.ValueString(), testProjectId)
			helpers.AssertFieldEqual(t, "role_id", model.RoleId.ValueString(), testRoleId)
			helpers.AssertFieldEqual(t, "role_name", model.RoleName.ValueString(), testRoleName)
			helpers.AssertFieldEqual(t, "holder_id", model.HolderId.ValueString(), testHolderId)
			helpers.AssertFieldEqual(t, "holder_login", model.HolderLogin.ValueString(), tt.wantLogin)
			helpers.AssertFieldEqual(t, "holder_type", model.HolderType.ValueString(), tt.wantType)
		})
	}
}

func TestMapAPIToProjectModelKeepsProjectWhenResponseOmitsIt(t *testing.T) {
	api := projectScopedAssignment(youtrack.Holder{Id: testHolderId, Login: testHolderLogin, Type: assignedRoleHolderTypeUser})
	api.Scope.Project = nil
	model := projectRoleAssignmentResourceModel{ProjectId: types.StringValue(testProjectId)}

	mapAPIToProjectModel(api, &model)

	helpers.AssertFieldEqual(t, "project_id", model.ProjectId.ValueString(), testProjectId)
}

func TestNewProjectAssignedRole(t *testing.T) {
	tests := []struct {
		name       string
		holder     youtrack.Holder
		requested  types.String
		wantHolder string
	}{
		{
			name:       "user holder",
			holder:     youtrack.Holder{Id: testHolderId, Type: assignedRoleHolderTypeUser},
			requested:  types.StringValue(holderTypeUser),
			wantHolder: assignedRoleHolderTypeUser,
		},
		{
			name:       "group holder",
			holder:     youtrack.Holder{Id: testHolderId, Type: assignedRoleHolderTypeGroup},
			requested:  types.StringValue(holderTypeGroup),
			wantHolder: assignedRoleHolderTypeGroup,
		},
		{
			name:       "holder without an API type follows the requested one",
			holder:     youtrack.Holder{Id: testHolderId},
			requested:  types.StringValue(holderTypeGroup),
			wantHolder: assignedRoleHolderTypeGroup,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newProjectAssignedRole(testRoleId, testProjectId, &tt.holder, tt.requested)

			helpers.AssertFieldEqual(t, "role", got.Role.Id, testRoleId)
			helpers.AssertFieldEqual(t, "scope type", got.Scope.Type, projectScopeType)
			if got.Scope.Project == nil {
				t.Fatal("scope project is nil, want the project")
			}
			helpers.AssertFieldEqual(t, "scope project", got.Scope.Project.ID, testProjectId)
			helpers.AssertFieldEqual(t, "holder id", got.Holder.Id, testHolderId)
			helpers.AssertFieldEqual(t, "holder type", got.Holder.Type, tt.wantHolder)
		})
	}
}

func TestCheckProjectScope(t *testing.T) {
	tests := []struct {
		name  string
		scope youtrack.Scope
		want  bool
	}{
		{name: "project scope", scope: youtrack.Scope{Type: projectScopeType, Project: &youtrack.Project{ID: testProjectId}}, want: true},
		{name: "project scope without a project in the response", scope: youtrack.Scope{Type: projectScopeType}, want: true},
		{name: "global scope", scope: youtrack.Scope{Type: globalScopeType}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diagnostics diag.Diagnostics
			got := checkProjectScope(&youtrack.AssignedRole{Scope: tt.scope}, &diagnostics)

			helpers.AssertFieldEqual(t, "checkProjectScope", got, tt.want)
			helpers.AssertFieldEqual(t, "HasError", diagnostics.HasError(), !tt.want)
		})
	}
}

func TestValidateHolderType(t *testing.T) {
	tests := []struct {
		name       string
		holderType types.String
		wantError  bool
	}{
		{name: "not set", holderType: types.StringNull()},
		{name: "unknown", holderType: types.StringUnknown()},
		{name: "user", holderType: types.StringValue(holderTypeUser)},
		{name: "group", holderType: types.StringValue(holderTypeGroup)},
		{name: "capitalised user", holderType: types.StringValue("User"), wantError: true},
		{name: "unsupported type", holderType: types.StringValue("team"), wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diagnostics diag.Diagnostics
			validateHolderType(tt.holderType, &diagnostics)

			helpers.AssertFieldEqual(t, "HasError", diagnostics.HasError(), tt.wantError)
		})
	}
}
