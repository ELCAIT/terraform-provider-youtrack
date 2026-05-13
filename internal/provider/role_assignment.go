// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &roleAssignmentResource{}
	_ resource.ResourceWithConfigure   = &roleAssignmentResource{}
	_ resource.ResourceWithImportState = &roleAssignmentResource{}
)

// NewRoleAssignmentResource is a helper function to simplify the provider implementation.
func NewRoleAssignmentResource() resource.Resource {
	return &roleAssignmentResource{}
}

// roleAssignmentResource is the resource implementation.
type roleAssignmentResource struct {
	client *youtrack.Client
}

// roleAssignmentResourceModel maps the resource schema data.
type roleAssignmentResourceModel struct {
	Id          types.String `tfsdk:"id"`
	RoleId      types.String `tfsdk:"role_id"`
	RoleName    types.String `tfsdk:"role_name"`
	HolderLogin types.String `tfsdk:"holder_login"`
	HolderId    types.String `tfsdk:"holder_id"`
	HolderName  types.String `tfsdk:"holder_name"`
	HolderType  types.String `tfsdk:"holder_type"`
}

// Metadata returns the resource type name.
func (r *roleAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_assignment"
}

// Schema defines the schema for the resource.
func (r *roleAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack role assignment resource. This resource manages role assignments with ONLY global scope in YouTrack, which assigns a role to a user or group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "YouTrack ID of the role assignment (computed)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the role to assign (e.g. youtrack_role.my_role.id)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the assigned role (computed)",
			},
			"holder_login": schema.StringAttribute{
				Required:    true,
				Description: "The login (username) of the user or name of the group to assign the role to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"holder_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the holder (computed)",
			},
			"holder_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the holder (computed)",
			},
			"holder_type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The type of the holder: 'user' or 'group'. If not specified, will try user first, then group.",
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *roleAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *roleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleAssignmentResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	apiRoleAssignment := r.buildAPIRoleAssignment(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	createdAssignment, err := r.client.CreateAssignedRole(ctx, *apiRoleAssignment)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingRoleAssignment,
			fmt.Sprintf("Could not create role assignment: %v", err),
		)
		return
	}

	if createdAssignment.Scope.Type != globalScopeType {
		resp.Diagnostics.AddError(
			errInvalidScopeType,
			fmt.Sprintf(errExpectedScopeTypeFmt, globalScopeType, createdAssignment.Scope.Type),
		)
		return
	}

	r.mapAPIToModel(createdAssignment, &plan)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *roleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleAssignmentResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.Id, &resp.Diagnostics, errMissingRoleAssignmentID, errRoleAssignmentIDRequired) {
		return
	}

	apiAssignment, err := r.client.GetAssignedRoleById(ctx, state.Id.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			errReadingRoleAssignment,
			fmt.Sprintf("Could not read role assignment: %v", err),
		)
		return
	}

	if apiAssignment.Scope.Type != globalScopeType {
		resp.Diagnostics.AddError(
			errInvalidScopeType,
			fmt.Sprintf(errExpectedScopeTypeFmt, globalScopeType, apiAssignment.Scope.Type),
		)
		return
	}

	r.mapAPIToModel(apiAssignment, &state)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *roleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan roleAssignmentResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	roleAssignmentId := plan.Id.ValueString()

	apiRoleAssignment := r.buildAPIRoleAssignment(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedAssignment, err := r.client.UpdateAssignedRole(ctx, roleAssignmentId, *apiRoleAssignment)
	if err != nil {
		resp.Diagnostics.AddError(
			errUpdatingRoleAssignment,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "role assignment", err),
		)
		return
	}

	if updatedAssignment.Scope.Type != globalScopeType {
		resp.Diagnostics.AddError(
			errInvalidScopeType,
			fmt.Sprintf(errExpectedScopeTypeFmt, globalScopeType, updatedAssignment.Scope.Type),
		)
		return
	}

	r.mapAPIToModel(updatedAssignment, &plan)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *roleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleAssignmentResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.Id) {
		return
	}

	err := r.client.DeleteAssignedRole(ctx, state.Id.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			errDeletingRoleAssignment,
			fmt.Sprintf("Could not delete role assignment: %v", err),
		)
		return
	}
}

// ImportState imports the resource state using the role assignment ID.
func (r *roleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	roleAssignmentId := strings.TrimSpace(req.ID)

	apiAssignment, err := r.client.GetAssignedRoleById(ctx, roleAssignmentId)
	if err != nil {
		resp.Diagnostics.AddError(
			errImportingRoleAssignment,
			fmt.Sprintf("Could not read role assignment: %v", err),
		)
		return
	}

	if apiAssignment.Scope.Type != globalScopeType {
		resp.Diagnostics.AddError(
			errInvalidScopeType,
			fmt.Sprintf(errExpectedScopeTypeFmt, globalScopeType, apiAssignment.Scope.Type),
		)
		return
	}

	var state roleAssignmentResourceModel
	r.mapAPIToModel(apiAssignment, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}
