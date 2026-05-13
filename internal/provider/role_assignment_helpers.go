package provider

import (
	"context"
	"fmt"
	"strings"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	holderTypeUser  = "user"
	holderTypeGroup = "group"

	assignedRoleHolderTypeUser  = "User"
	assignedRoleHolderTypeGroup = "UserGroup"

	globalScopeType         = "GlobalScope"
	errExpectedScopeTypeFmt = "Expected scope type '%s', got '%s'. This resource only supports global scope assignments."

	errCreatingRoleAssignment  = "Error creating role assignment"
	errReadingRoleAssignment   = "Error reading role assignment"
	errUpdatingRoleAssignment  = "Error updating role assignment"
	errDeletingRoleAssignment  = "Error deleting role assignment"
	errImportingRoleAssignment = "Error importing role assignment"
	errMissingRoleAssignmentID = "Missing role assignment ID"
	errInvalidScopeType        = "Invalid scope type"
	errLookupRole              = "Error looking up role"
	errLookupHolder            = "Error looking up holder"

	errRoleAssignmentIDRequired = "Role assignment ID is required to read the role assignment"
)

// lookupHolderByLogin looks up a user or group by login/name and returns the holder.
func (r *roleAssignmentResource) lookupHolderByLogin(ctx context.Context, login string, holderType types.String, diagnostics *diag.Diagnostics) *youtrack.Holder {
	if !holderType.IsNull() && !holderType.IsUnknown() {
		return r.lookupHolderByExplicitType(ctx, login, strings.ToLower(holderType.ValueString()), diagnostics)
	}

	return r.lookupHolderAuto(ctx, login, diagnostics)
}

func (r *roleAssignmentResource) lookupHolderByExplicitType(ctx context.Context, login, holderTypeValue string, diagnostics *diag.Diagnostics) *youtrack.Holder {
	var holder *youtrack.Holder
	var err error

	switch holderTypeValue {
	case holderTypeUser:
		holder, err = r.client.GetUserByLogin(ctx, login)
		if err != nil {
			diagnostics.AddError(
				errLookupHolder,
				fmt.Sprintf("Could not find user with login '%s': %v", login, err),
			)
			return nil
		}
	case holderTypeGroup:
		holder, err = r.client.GetUserGroupByName(ctx, login)
		if err != nil {
			diagnostics.AddError(
				errLookupHolder,
				fmt.Sprintf("Could not find group with name '%s': %v", login, err),
			)
			return nil
		}
	default:
		diagnostics.AddError(
			errLookupHolder,
			fmt.Sprintf("Invalid holder_type '%s'. Must be 'user' or 'group'.", holderTypeValue),
		)
		return nil
	}

	return holder
}

func (r *roleAssignmentResource) lookupHolderAuto(ctx context.Context, login string, diagnostics *diag.Diagnostics) *youtrack.Holder {
	holder, err := r.client.GetUserByLogin(ctx, login)
	if err == nil {
		return holder
	}

	holder, err = r.client.GetUserGroupByName(ctx, login)
	if err != nil {
		diagnostics.AddError(
			errLookupHolder,
			fmt.Sprintf("Could not find user or group with login/name '%s': %v. Consider specifying holder_type.", login, err),
		)
		return nil
	}

	return holder
}

// buildAPIRoleAssignment creates a youtrack.AssignedRoles from the model data.
func (r *roleAssignmentResource) buildAPIRoleAssignment(ctx context.Context, model *roleAssignmentResourceModel, diagnostics *diag.Diagnostics) *youtrack.AssignedRoles {
	holder := r.lookupHolderByLogin(ctx, model.HolderLogin.ValueString(), model.HolderType, diagnostics)
	if holder == nil {
		return nil
	}

	apiAssignment := youtrack.AssignedRoles{
		Type: "AssignedRole",
		Role: youtrack.Role{
			Id: model.RoleId.ValueString(),
		},
		Scope: youtrack.Scope{
			Type: "GlobalScope",
		},
		Holder: youtrack.Holder{
			Id:   holder.Id,
			Type: mapHolderTypeForAssignedRole(holder.Type, model.HolderType),
		},
	}

	if !model.Id.IsNull() {
		apiAssignment.Id = model.Id.ValueString()
	}

	return &apiAssignment
}

// mapAPIToModel maps the API response to the Terraform model.
func (r *roleAssignmentResource) mapAPIToModel(apiAssignment *youtrack.AssignedRoles, model *roleAssignmentResourceModel) {
	model.Id = types.StringValue(apiAssignment.Id)
	model.RoleId = types.StringValue(apiAssignment.Role.Id)
	model.RoleName = types.StringValue(apiAssignment.Role.Name)
	model.HolderId = types.StringValue(apiAssignment.Holder.Id)
	model.HolderName = types.StringValue(apiAssignment.Holder.Name)
	if apiAssignment.Holder.Login != "" {
		model.HolderLogin = types.StringValue(apiAssignment.Holder.Login)
	} else if model.HolderLogin.IsNull() || model.HolderLogin.IsUnknown() {
		model.HolderLogin = types.StringValue("")
	}
	model.HolderType = types.StringValue(normalizeHolderType(apiAssignment.Holder.Type))
}

// normalizeHolderType converts API holder types to our simplified types.
func normalizeHolderType(apiType string) string {
	if strings.Contains(strings.ToLower(apiType), "user") && !strings.Contains(strings.ToLower(apiType), "group") {
		return holderTypeUser
	}
	return holderTypeGroup
}

// mapHolderTypeForAssignedRole maps holder type values from Hub/API responses
// to values accepted by YouTrack AssignedRole holder payload.
func mapHolderTypeForAssignedRole(holderAPIType string, requestedHolderType types.String) string {
	holderTypeLower := strings.ToLower(holderAPIType)
	if strings.Contains(holderTypeLower, "group") {
		return assignedRoleHolderTypeGroup
	}
	if strings.Contains(holderTypeLower, "user") {
		return assignedRoleHolderTypeUser
	}

	if !requestedHolderType.IsNull() && !requestedHolderType.IsUnknown() {
		if strings.EqualFold(requestedHolderType.ValueString(), holderTypeGroup) {
			return assignedRoleHolderTypeGroup
		}
	}

	return assignedRoleHolderTypeUser
}
