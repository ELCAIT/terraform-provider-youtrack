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

func permissionsFromAPI(apiPerms []youtrack.Permission) []types.String {
	perms := make([]types.String, 0, len(apiPerms))
	for _, perm := range apiPerms {
		if perm.Name != "" {
			perms = append(perms, types.StringValue(perm.Name))
		}
	}
	return perms
}

func buildAPIPermissionSet(apiPerms []youtrack.Permission) map[string]struct{} {
	m := make(map[string]struct{}, len(apiPerms))
	for _, perm := range apiPerms {
		if perm.Name != "" {
			m[normalizePermissionName(perm.Name)] = struct{}{}
		}
	}
	return m
}

func collectPreservedStatePerms(statePerms []types.String, apiPermSet map[string]struct{}) ([]types.String, map[string]struct{}) {
	perms := make([]types.String, 0, len(statePerms))
	preserved := make(map[string]struct{}, len(statePerms))
	for _, statePerm := range statePerms {
		key := normalizePermissionName(statePerm.ValueString())
		if key == "" {
			continue
		}
		if _, exists := apiPermSet[key]; exists {
			perms = append(perms, statePerm)
			preserved[key] = struct{}{}
		}
	}
	return perms, preserved
}

func appendAPIOnlyPerms(perms []types.String, apiPerms []youtrack.Permission, preserved, implicitPermNames map[string]struct{}) []types.String {
	for _, perm := range apiPerms {
		key := normalizePermissionName(perm.Name)
		if key == "" {
			continue
		}
		_, isPreserved := preserved[key]
		_, isImplicit := implicitPermNames[key]
		if !isPreserved && !isImplicit {
			perms = append(perms, types.StringValue(perm.Name))
		}
	}
	return perms
}

func normalizePermissionName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func buildConfiguredPermissions(statePerms []types.String) (map[string]struct{}, []string) {
	configured := make(map[string]struct{}, len(statePerms))
	queue := make([]string, 0, len(statePerms))

	for _, statePerm := range statePerms {
		name := normalizePermissionName(statePerm.ValueString())
		if name == "" {
			continue
		}
		configured[name] = struct{}{}
		queue = append(queue, name)
	}

	return configured, queue
}

func buildPermissionCatalogByName(catalog []youtrack.PermissionGraphEntry) map[string]youtrack.PermissionGraphEntry {
	catalogByName := make(map[string]youtrack.PermissionGraphEntry, len(catalog))

	for _, p := range catalog {
		name := normalizePermissionName(p.Name)
		if name == "" {
			continue
		}
		catalogByName[name] = p
	}

	return catalogByName
}

func collectImpliedPermissions(result map[string]struct{}, configured map[string]struct{}, queue []string, catalogByName map[string]youtrack.PermissionGraphEntry) {
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
			impliedName := normalizePermissionName(implied.Name)
			if impliedName == "" {
				continue
			}
			if _, explicitlyConfigured := configured[impliedName]; !explicitlyConfigured {
				result[impliedName] = struct{}{}
			}
			queue = append(queue, impliedName)
		}
	}
}

func collectDependentPermissions(result map[string]struct{}, configured map[string]struct{}, catalog []youtrack.PermissionGraphEntry) {
	for _, candidate := range catalog {
		candidateName := normalizePermissionName(candidate.Name)
		if candidateName == "" {
			continue
		}

		for _, dependent := range candidate.DependentPermissions {
			dependentName := normalizePermissionName(dependent.Name)
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
}

// resolveImplicitPermissionNames computes permissions implied by the configured
// state permissions using both impliedPermissions and dependentPermissions
// relations from the permissions catalog.
func resolveImplicitPermissionNames(statePerms []types.String, catalog []youtrack.PermissionGraphEntry) map[string]struct{} {
	result := make(map[string]struct{})
	if len(statePerms) == 0 || len(catalog) == 0 {
		return result
	}

	catalogByName := buildPermissionCatalogByName(catalog)
	configured, queue := buildConfiguredPermissions(statePerms)

	collectImpliedPermissions(result, configured, queue, catalogByName)
	collectDependentPermissions(result, configured, catalog)

	return result
}
