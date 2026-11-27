// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	projectScopeType = "ProjectScope"

	errExpectedProjectScopeFmt = "Expected scope type '%s', got '%s'. This resource only supports project scope assignments."

	errCreatingProjectRoleAssignment  = "Error creating project role assignment"
	errReadingProjectRoleAssignment   = "Error reading project role assignment"
	errUpdatingProjectRoleAssignment  = "Error updating project role assignment"
	errDeletingProjectRoleAssignment  = "Error deleting project role assignment"
	errMissingProjectRoleAssignmentID = "Missing project role assignment ID"
	errInvalidHolderType              = "Invalid holder_type"

	errProjectRoleAssignmentIDRequired = "Project role assignment ID is required to read the project role assignment"
	errProjectRoleAssignmentImmutable  = "Every attribute of a project role assignment forces a replacement, so an in-place update is never expected."
	errInvalidHolderTypeFmt            = "holder_type must be '%s' or '%s', got '%s'."
)

// buildAPIProjectRoleAssignment creates a youtrack.AssignedRole scoped to the project from the model data.
func (r *projectRoleAssignmentResource) buildAPIProjectRoleAssignment(ctx context.Context, model *projectRoleAssignmentResourceModel, diagnostics *diag.Diagnostics) *youtrack.AssignedRole {
	holder := lookupHolderByLogin(ctx, r.client, model.HolderLogin.ValueString(), model.HolderType, diagnostics)
	if holder == nil {
		return nil
	}

	assignment := newProjectAssignedRole(model.RoleId.ValueString(), model.ProjectId.ValueString(), holder, model.HolderType)
	return &assignment
}

// newProjectAssignedRole builds the AssignedRole payload that gives a role to a holder within one project.
func newProjectAssignedRole(roleId, projectId string, holder *youtrack.Holder, requestedHolderType types.String) youtrack.AssignedRole {
	return youtrack.AssignedRole{
		Role: youtrack.Role{Id: roleId},
		Scope: youtrack.Scope{
			Type:    projectScopeType,
			Project: &youtrack.Project{ID: projectId},
		},
		Holder: youtrack.Holder{
			Id:   holder.Id,
			Type: mapHolderTypeForAssignedRole(holder.Type, requestedHolderType),
		},
	}
}

// mapAPIToProjectModel maps the API response to the Terraform model.
func mapAPIToProjectModel(apiAssignment *youtrack.AssignedRole, model *projectRoleAssignmentResourceModel) {
	model.Id = types.StringValue(apiAssignment.Id)
	model.RoleId = types.StringValue(apiAssignment.Role.Id)
	model.RoleName = types.StringValue(apiAssignment.Role.Name)
	model.HolderId = types.StringValue(apiAssignment.Holder.Id)
	model.HolderName = types.StringValue(apiAssignment.Holder.Name)
	model.HolderType = types.StringValue(normalizeHolderType(apiAssignment.Holder.Type))
	model.HolderLogin = resolveProjectHolderLogin(model.HolderLogin, apiAssignment.Holder)
	if apiAssignment.Scope.Project != nil {
		model.ProjectId = types.StringValue(apiAssignment.Scope.Project.ID)
	}
}

// resolveProjectHolderLogin returns the holder_login to store for an API holder. A group has no login, so
// its name stands in for it, which is also what the configuration holds and what an import has to
// reproduce. The current value is kept when it only differs in case, because YouTrack matches logins and
// group names case-insensitively and holder_login forces a replacement when it changes.
func resolveProjectHolderLogin(current types.String, holder youtrack.Holder) types.String {
	login := holder.Login
	if login == "" && normalizeHolderType(holder.Type) == holderTypeGroup {
		login = holder.Name
	}

	hasCurrent := !current.IsNull() && !current.IsUnknown()
	switch {
	case login == "" && hasCurrent:
		return current
	case login == "":
		return types.StringValue("")
	case hasCurrent && strings.EqualFold(current.ValueString(), login):
		return current
	default:
		return types.StringValue(login)
	}
}

// checkProjectScope reports a diagnostic when the assignment is not scoped to a project. The project itself
// may be missing from the response, for instance when it cannot be read, and the state then keeps its ID.
func checkProjectScope(apiAssignment *youtrack.AssignedRole, diagnostics *diag.Diagnostics) bool {
	if apiAssignment.Scope.Type == projectScopeType {
		return true
	}

	diagnostics.AddError(
		errInvalidScopeType,
		fmt.Sprintf(errExpectedProjectScopeFmt, projectScopeType, apiAssignment.Scope.Type),
	)
	return false
}

// validateHolderType reports a diagnostic when a configured holder_type is not 'user' or 'group'. The value
// is kept as written in the state, so a different spelling would never match what the API returns.
func validateHolderType(holderType types.String, diagnostics *diag.Diagnostics) {
	if holderType.IsNull() || holderType.IsUnknown() {
		return
	}

	if value := holderType.ValueString(); value != holderTypeUser && value != holderTypeGroup {
		diagnostics.AddAttributeError(
			path.Root("holder_type"),
			errInvalidHolderType,
			fmt.Sprintf(errInvalidHolderTypeFmt, holderTypeUser, holderTypeGroup, value),
		)
	}
}
