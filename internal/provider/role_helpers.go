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
	errCreatingRole        = "Error creating role"
	errReadingRole         = "Error reading role"
	errUpdatingRole        = "Error updating role"
	errDeletingRole        = "Error deleting role"
	errMissingRoleID       = "Missing role ID"
	errInvalidRoleName     = "Invalid Role Name"
	errEmptyPermissionList = "Empty Permissions List"

	errRoleIDRequired       = "Role ID is required to read the role"
	errRoleNameEmpty        = "Role name cannot be empty"
	errPermissionsListEmpty = "Permissions list must contain at least one permission"
)

// validateAndConvertPermissions resolves permission names to Permission objects.
func (r *roleResource) validateAndConvertPermissions(ctx context.Context, permissionNames []types.String, diagnostics *diag.Diagnostics) []youtrack.Permission {
	allPermissions, err := r.client.GetAllPermissions(ctx)
	if err != nil {
		diagnostics.AddError(
			"Error fetching permissions",
			fmt.Sprintf("Could not fetch available permissions: %v", err),
		)
		return nil
	}

	permMap := make(map[string]youtrack.Permission)
	for _, perm := range allPermissions {
		permMap[strings.ToLower(perm.Name)] = perm
	}

	var permissions []youtrack.Permission
	for _, permName := range permissionNames {
		permNameStr := permName.ValueString()
		if perm, found := permMap[strings.ToLower(permNameStr)]; found {
			permissions = append(permissions, perm)
		} else {
			diagnostics.AddError(
				"Unknown permission",
				fmt.Sprintf("Permission '%s' not found in available permissions", permNameStr),
			)
			return nil
		}
	}

	return permissions
}

// buildAPIRole builds a Role from the resource model.
func (r *roleResource) buildAPIRole(model *roleResourceModel, permissions []youtrack.Permission) youtrack.Role {
	apiRole := youtrack.Role{
		Name:        model.Name.ValueString(),
		Permissions: permissions,
	}

	if !model.Id.IsNull() {
		apiRole.Id = model.Id.ValueString()
	}

	if !model.Description.IsNull() {
		apiRole.Description = model.Description.ValueString()
	}

	return apiRole
}
