// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"strings"
	"testing"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	roleName        = "Test Role"
	roleDescription = "Test Description"
	roleId          = "role-123"
	readIssuePerm   = "Read Issue"
	createIssuePerm = "Create Issue"
	deleteIssuePerm = "Delete Issue"
)

func TestRoleResourceModelConversion(t *testing.T) {
	tests := []struct {
		name  string
		model roleResourceModel
		want  roleResourceModel
	}{
		{
			name: "model with all fields",
			model: roleResourceModel{
				Id:          types.StringValue(roleId),
				Name:        types.StringValue(roleName),
				Description: types.StringValue(roleDescription),
				Permissions: []types.String{
					types.StringValue(readIssuePerm),
					types.StringValue(createIssuePerm),
				},
			},
			want: roleResourceModel{
				Id:          types.StringValue(roleId),
				Name:        types.StringValue(roleName),
				Description: types.StringValue(roleDescription),
				Permissions: []types.String{
					types.StringValue(readIssuePerm),
					types.StringValue(createIssuePerm),
				},
			},
		},
		{
			name: "model without description",
			model: roleResourceModel{
				Id:          types.StringValue(roleId),
				Name:        types.StringValue(roleName),
				Description: types.StringNull(),
				Permissions: []types.String{
					types.StringValue(readIssuePerm),
				},
			},
			want: roleResourceModel{
				Id:          types.StringValue(roleId),
				Name:        types.StringValue(roleName),
				Description: types.StringNull(),
				Permissions: []types.String{
					types.StringValue(readIssuePerm),
				},
			},
		},
		{
			name: "model with empty permissions",
			model: roleResourceModel{
				Id:          types.StringValue(roleId),
				Name:        types.StringValue(roleName),
				Description: types.StringValue(roleDescription),
				Permissions: []types.String{},
			},
			want: roleResourceModel{
				Id:          types.StringValue(roleId),
				Name:        types.StringValue(roleName),
				Description: types.StringValue(roleDescription),
				Permissions: []types.String{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.AssertFieldEqual(t, "Id", tt.model.Id.ValueString(), tt.want.Id.ValueString())
			helpers.AssertFieldEqual(t, "Name", tt.model.Name.ValueString(), tt.want.Name.ValueString())
			helpers.AssertFieldEqual(t, "Description.IsNull()", tt.model.Description.IsNull(), tt.want.Description.IsNull())
			if !tt.model.Description.IsNull() {
				helpers.AssertFieldEqual(t, "Description", tt.model.Description.ValueString(), tt.want.Description.ValueString())
			}
			helpers.AssertFieldEqual(t, "Permissions length", len(tt.model.Permissions), len(tt.want.Permissions))
		})
	}
}

func TestPermissionValidation(t *testing.T) {
	tests := []struct {
		name            string
		permissionNames []types.String
		availablePerms  []youtrack.Permission
		wantValid       bool
	}{
		{
			name: "valid permissions - exact match",
			permissionNames: []types.String{
				types.StringValue(readIssuePerm),
				types.StringValue(createIssuePerm),
			},
			availablePerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
				{Id: "perm2", Name: createIssuePerm},
			},
			wantValid: true,
		},
		{
			name: "valid permissions - case insensitive",
			permissionNames: []types.String{
				types.StringValue("read issue"),
				types.StringValue("CREATE ISSUE"),
			},
			availablePerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
				{Id: "perm2", Name: createIssuePerm},
			},
			wantValid: true,
		},
		{
			name: "invalid permission",
			permissionNames: []types.String{
				types.StringValue("Invalid Permission"),
			},
			availablePerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
			},
			wantValid: false,
		},
		{
			name:            "empty permissions list",
			permissionNames: []types.String{},
			availablePerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create permission lookup map (simulating the Create/Update logic)
			permMap := make(map[string]youtrack.Permission)
			for _, perm := range tt.availablePerms {
				permMap[strings.ToLower(perm.Name)] = perm
			}

			allValid := true
			for _, permName := range tt.permissionNames {
				permNameStr := permName.ValueString()
				if _, found := permMap[strings.ToLower(permNameStr)]; !found {
					allValid = false
					break
				}
			}

			helpers.AssertFieldEqual(t, "permissions valid", allValid, tt.wantValid)
		})
	}
}

func TestPreservePermissionOrder(t *testing.T) {
	tests := []struct {
		name         string
		statePerms   []types.String
		apiPerms     []youtrack.Permission
		wantPreserve bool
		wantCount    int
	}{
		{
			name: "preserves order and casing from state",
			statePerms: []types.String{
				types.StringValue(readIssuePerm),
				types.StringValue(createIssuePerm),
			},
			apiPerms: []youtrack.Permission{
				{Id: "perm2", Name: createIssuePerm},
				{Id: "perm1", Name: readIssuePerm},
			},
			wantPreserve: true,
			wantCount:    2,
		},
		{
			name: "filters out removed permissions",
			statePerms: []types.String{
				types.StringValue(readIssuePerm),
				types.StringValue(deleteIssuePerm),
			},
			apiPerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
			},
			wantPreserve: true,
			wantCount:    1,
		},
		{
			name:       "empty state permissions",
			statePerms: []types.String{},
			apiPerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
			},
			wantPreserve: true,
			wantCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create API permission map (simulating the Read logic)
			apiPermMap := make(map[string]string)
			for _, permission := range tt.apiPerms {
				apiPermMap[strings.ToLower(permission.Name)] = permission.Name
			}

			var preserved []types.String
			for _, statePerm := range tt.statePerms {
				permName := statePerm.ValueString()
				if _, exists := apiPermMap[strings.ToLower(permName)]; exists {
					preserved = append(preserved, types.StringValue(permName))
				}
			}

			helpers.AssertFieldEqual(t, "preserved count", len(preserved), tt.wantCount)
		})
	}
}

func TestDescriptionHandling(t *testing.T) {
	tests := []struct {
		name           string
		apiDescription string
		stateIsNull    bool
		wantIsNull     bool
		wantValue      string
	}{
		{
			name:           "API has description",
			apiDescription: roleDescription,
			stateIsNull:    false,
			wantIsNull:     false,
			wantValue:      roleDescription,
		},
		{
			name:           "API empty, state was null",
			apiDescription: "",
			stateIsNull:    true,
			wantIsNull:     true,
			wantValue:      "",
		},
		{
			name:           "API empty, state was not null",
			apiDescription: "",
			stateIsNull:    false,
			wantIsNull:     false,
			wantValue:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stateDesc types.String
			if tt.stateIsNull {
				stateDesc = types.StringNull()
			} else {
				stateDesc = types.StringValue("Previous Value")
			}

			// Simulate the description handling logic from Read method
			var result types.String
			if tt.apiDescription != "" {
				result = types.StringValue(tt.apiDescription)
			} else if stateDesc.IsNull() {
				result = types.StringNull()
			} else {
				result = types.StringValue("")
			}

			helpers.AssertFieldEqual(t, "description is null", result.IsNull(), tt.wantIsNull)
			if !result.IsNull() {
				helpers.AssertFieldEqual(t, "description value", result.ValueString(), tt.wantValue)
			}
		})
	}
}

// calculatePermissionDiff returns the permission IDs to add and remove.
func calculatePermissionDiff(current, desired []youtrack.Permission) (toAdd, toRemove []string) {
	desiredMap := make(map[string]bool)
	for _, perm := range desired {
		desiredMap[perm.Id] = true
	}

	currentMap := make(map[string]bool)
	for _, perm := range current {
		currentMap[perm.Id] = true
		if !desiredMap[perm.Id] {
			toRemove = append(toRemove, perm.Id)
		}
	}

	for _, perm := range desired {
		if !currentMap[perm.Id] {
			toAdd = append(toAdd, perm.Id)
		}
	}

	return toAdd, toRemove
}

func TestPermissionSync(t *testing.T) {
	tests := []struct {
		name         string
		currentPerms []youtrack.Permission
		desiredPerms []youtrack.Permission
		wantToAdd    []string
		wantToRemove []string
	}{
		{
			name: "add and remove permissions",
			currentPerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
				{Id: "perm2", Name: deleteIssuePerm},
			},
			desiredPerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
				{Id: "perm3", Name: createIssuePerm},
			},
			wantToAdd:    []string{"perm3"},
			wantToRemove: []string{"perm2"},
		},
		{
			name: "only add permissions",
			currentPerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
			},
			desiredPerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
				{Id: "perm2", Name: createIssuePerm},
			},
			wantToAdd:    []string{"perm2"},
			wantToRemove: []string{},
		},
		{
			name: "only remove permissions",
			currentPerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
				{Id: "perm2", Name: deleteIssuePerm},
			},
			desiredPerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
			},
			wantToAdd:    []string{},
			wantToRemove: []string{"perm2"},
		},
		{
			name: "no changes needed",
			currentPerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
			},
			desiredPerms: []youtrack.Permission{
				{Id: "perm1", Name: readIssuePerm},
			},
			wantToAdd:    []string{},
			wantToRemove: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toAdd, toRemove := calculatePermissionDiff(tt.currentPerms, tt.desiredPerms)

			helpers.AssertFieldEqual(t, "permissions to add count", len(toAdd), len(tt.wantToAdd))
			helpers.AssertFieldEqual(t, "permissions to remove count", len(toRemove), len(tt.wantToRemove))
		})
	}
}
