// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &roleResource{}
	_ resource.ResourceWithConfigure      = &roleResource{}
	_ resource.ResourceWithImportState    = &roleResource{}
	_ resource.ResourceWithValidateConfig = &roleResource{}
)

// NewRoleResource is a helper function to simplify the provider implementation.
func NewRoleResource() resource.Resource {
	return &roleResource{}
}

// roleResource is the resource implementation.
type roleResource struct {
	client *youtrack.Client
}

// roleResourceModel maps the resource schema data.
type roleResourceModel struct {
	Id          types.String   `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	Permissions []types.String `tfsdk:"permissions"`
}

// Metadata returns the resource type name.
func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the resource.
func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack role resource. This resource manages a single role in YouTrack, which defines a set of permissions that can be assigned to users.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "YouTrack ID of the role (computed)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The display name of the role",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The role description",
			},
			"permissions": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "List of permission names to assign to this role. The permissions are the ones available in YouTrack, e.g. 'Read Issues', 'Create Issues', etc.",
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	permissions := r.validateAndConvertPermissions(ctx, plan.Permissions, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRole := r.buildAPIRole(&plan, permissions)

	createdRole, err := r.client.CreateYoutrackRole(ctx, apiRole)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingRole,
			fmt.Sprintf("Could not create role: %v", err),
		)
		return
	}

	plan.Id = types.StringValue(createdRole.Id)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.Id, &resp.Diagnostics, errMissingRoleID, errRoleIDRequired) {
		return
	}

	apiRole, err := r.client.GetYoutrackRoleById(ctx, state.Id.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			errReadingRole,
			fmt.Sprintf("Could not read role: %v", err),
		)
		return
	}

	state.Name = types.StringValue(apiRole.Name)
	state.Description = normalizeRoleDescription(apiRole.Description, state.Description)
	state.Permissions = reconcileRolePermissions(state.Permissions, apiRole.Permissions)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// normalizeRoleDescription returns the appropriate description state value.
// An empty API description preserves the existing null/empty distinction from state.
func normalizeRoleDescription(apiDesc string, stateDesc types.String) types.String {
	if apiDesc != "" {
		return types.StringValue(apiDesc)
	}
	if stateDesc.IsNull() {
		return types.StringNull()
	}
	return types.StringValue("")
}

// reconcileRolePermissions reconciles the permission list from state with the API response.
// When the API returns a non-empty list: on import, permissions are populated from the API;
// on normal read, state order is preserved and absent permissions are dropped.
func reconcileRolePermissions(statePerms []types.String, apiPerms []youtrack.Permission) []types.String {
	if len(apiPerms) == 0 {
		return statePerms
	}

	if len(statePerms) == 0 {
		var perms []types.String
		for _, perm := range apiPerms {
			if perm.Name != "" {
				perms = append(perms, types.StringValue(perm.Name))
			}
		}
		return perms
	}

	apiPermMap := make(map[string]struct{}, len(apiPerms))
	for _, perm := range apiPerms {
		apiPermMap[strings.ToLower(perm.Name)] = struct{}{}
	}

	var perms []types.String
	for _, statePerm := range statePerms {
		if _, exists := apiPermMap[strings.ToLower(statePerm.ValueString())]; exists {
			perms = append(perms, statePerm)
		}
	}
	return perms
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	permissions := r.validateAndConvertPermissions(ctx, plan.Permissions, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiRole := r.buildAPIRole(&plan, permissions)

	_, err := r.client.UpdateYoutrackRole(ctx, apiRole)
	if err != nil {
		resp.Diagnostics.AddError(
			errUpdatingRole,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "role", err),
		)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.Id) {
		return
	}

	err := r.client.DeleteYoutrackRole(ctx, state.Id.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			errDeletingRole,
			fmt.Sprintf("Could not delete role: %v", err),
		)
		return
	}
}

// ImportState imports the resource state by role ID.
func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ValidateConfig validates the resource configuration.
func (r *roleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config roleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.Name.IsNull() && !config.Name.IsUnknown() {
		name := config.Name.ValueString()
		if strings.TrimSpace(name) == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("name"),
				errInvalidRoleName,
				errRoleNameEmpty,
			)
		}
	}

	if config.Permissions != nil && len(config.Permissions) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("permissions"),
			errEmptyPermissionList,
			errPermissionsListEmpty,
		)
	}
}

// Configure adds the provider configured client to the resource.
func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}
