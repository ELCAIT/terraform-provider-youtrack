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

// resolveImplicitPermissionNames computes permissions implied by the configured
// state permissions using both impliedPermissions and dependentPermissions
// relations from the permissions catalog.
func resolveImplicitPermissionNames(statePerms []types.String, catalog []youtrack.PermissionGraphEntry) map[string]struct{} {
	result := make(map[string]struct{})
	if len(statePerms) == 0 || len(catalog) == 0 {
		return result
	}

	catalogByName := make(map[string]youtrack.PermissionGraphEntry, len(catalog))
	for _, p := range catalog {
		name := strings.ToLower(strings.TrimSpace(p.Name))
		if name == "" {
			continue
		}
		catalogByName[name] = p
	}

	configured := make(map[string]struct{}, len(statePerms))
	queue := make([]string, 0, len(statePerms))
	for _, statePerm := range statePerms {
		name := strings.ToLower(strings.TrimSpace(statePerm.ValueString()))
		if name == "" {
			continue
		}
		configured[name] = struct{}{}
		queue = append(queue, name)
	}

	visited := make(map[string]struct{}, len(queue))
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		if _, seen := visited[name]; seen {
			continue
		}
		visited[name] = struct{}{}

		perm, ok := catalogByName[name]
		if !ok {
			continue
		}

		for _, implied := range perm.ImpliedPermissions {
			impliedName := strings.ToLower(strings.TrimSpace(implied.Name))
			if impliedName == "" {
				continue
			}
			if _, explicitlyConfigured := configured[impliedName]; !explicitlyConfigured {
				result[impliedName] = struct{}{}
			}
			queue = append(queue, impliedName)
		}
	}

	for _, candidate := range catalog {
		candidateName := strings.ToLower(strings.TrimSpace(candidate.Name))
		if candidateName == "" {
			continue
		}
		for _, dependent := range candidate.DependentPermissions {
			dependentName := strings.ToLower(strings.TrimSpace(dependent.Name))
			if dependentName == "" {
				continue
			}
			if _, configuredDep := configured[dependentName]; configuredDep {
				if _, explicitlyConfigured := configured[candidateName]; !explicitlyConfigured {
					result[candidateName] = struct{}{}
				}
				break
			}
		}
	}

	return result
}
