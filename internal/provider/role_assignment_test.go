// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testRoleAssignmentId = "assignment-123"
	testRoleKey          = "developer"
	testRoleId           = "role-456"
	testRoleName         = "Developer"
	testHolderLogin      = "john.doe"
	testHolderId         = "holder-789"
	testHolderName       = "John Doe"
	testHolderTypeUser   = "user"
	testHolderTypeGroup  = "group"
)

func TestRoleAssignmentResourceModelConversion(t *testing.T) {
	tests := []struct {
		name  string
		model roleAssignmentResourceModel
		want  roleAssignmentResourceModel
	}{
		{
			name: "model with all fields for user",
			model: roleAssignmentResourceModel{
				Id:          types.StringValue(testRoleAssignmentId),
				RoleId:      types.StringValue(testRoleId),
				RoleName:    types.StringValue(testRoleName),
				HolderLogin: types.StringValue(testHolderLogin),
				HolderId:    types.StringValue(testHolderId),
				HolderName:  types.StringValue(testHolderName),
				HolderType:  types.StringValue(testHolderTypeUser),
			},
			want: roleAssignmentResourceModel{
				Id:          types.StringValue(testRoleAssignmentId),
				RoleId:      types.StringValue(testRoleId),
				RoleName:    types.StringValue(testRoleName),
				HolderLogin: types.StringValue(testHolderLogin),
				HolderId:    types.StringValue(testHolderId),
				HolderName:  types.StringValue(testHolderName),
				HolderType:  types.StringValue(testHolderTypeUser),
			},
		},
		{
			name: "model with all fields for group",
			model: roleAssignmentResourceModel{
				Id:          types.StringValue(testRoleAssignmentId),
				RoleId:      types.StringValue(testRoleId),
				RoleName:    types.StringValue(testRoleName),
				HolderLogin: types.StringValue("Administrators"),
				HolderId:    types.StringValue(testHolderId),
				HolderName:  types.StringValue("Administrators"),
				HolderType:  types.StringValue(testHolderTypeGroup),
			},
			want: roleAssignmentResourceModel{
				Id:          types.StringValue(testRoleAssignmentId),
				RoleId:      types.StringValue(testRoleId),
				RoleName:    types.StringValue(testRoleName),
				HolderLogin: types.StringValue("Administrators"),
				HolderId:    types.StringValue(testHolderId),
				HolderName:  types.StringValue("Administrators"),
				HolderType:  types.StringValue(testHolderTypeGroup),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.AssertFieldEqual(t, "Id", tt.model.Id.ValueString(), tt.want.Id.ValueString())
			helpers.AssertFieldEqual(t, "RoleId", tt.model.RoleId.ValueString(), tt.want.RoleId.ValueString())
			helpers.AssertFieldEqual(t, "RoleName", tt.model.RoleName.ValueString(), tt.want.RoleName.ValueString())
			helpers.AssertFieldEqual(t, "HolderLogin", tt.model.HolderLogin.ValueString(), tt.want.HolderLogin.ValueString())
			helpers.AssertFieldEqual(t, "HolderId", tt.model.HolderId.ValueString(), tt.want.HolderId.ValueString())
			helpers.AssertFieldEqual(t, "HolderName", tt.model.HolderName.ValueString(), tt.want.HolderName.ValueString())
			helpers.AssertFieldEqual(t, "HolderType", tt.model.HolderType.ValueString(), tt.want.HolderType.ValueString())
		})
	}
}

func TestNormalizeHolderType(t *testing.T) {
	tests := []struct {
		name     string
		apiType  string
		expected string
	}{
		{
			name:     "User type",
			apiType:  "User",
			expected: holderTypeUser,
		},
		{
			name:     "UserGroup type",
			apiType:  "UserGroup",
			expected: holderTypeGroup,
		},
		{
			name:     "user lowercase",
			apiType:  "user",
			expected: holderTypeUser,
		},
		{
			name:     "group lowercase",
			apiType:  "usergroup",
			expected: holderTypeGroup,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeHolderType(tt.apiType)
			helpers.AssertFieldEqual(t, "normalized type", got, tt.expected)
		})
	}
}
