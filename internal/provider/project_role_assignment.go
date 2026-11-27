// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &projectRoleAssignmentResource{}
	_ resource.ResourceWithConfigure   = &projectRoleAssignmentResource{}
	_ resource.ResourceWithImportState = &projectRoleAssignmentResource{}

	_ resource.ResourceWithValidateConfig = &projectRoleAssignmentResource{}
)

// NewProjectRoleAssignmentResource is a helper function to simplify the provider implementation.
func NewProjectRoleAssignmentResource() resource.Resource {
	return &projectRoleAssignmentResource{}
}

// projectRoleAssignmentResource is the resource implementation.
type projectRoleAssignmentResource struct {
	client *youtrack.Client
}

// projectRoleAssignmentResourceModel maps the resource schema data.
type projectRoleAssignmentResourceModel struct {
	Id          types.String `tfsdk:"id"`
	RoleId      types.String `tfsdk:"role_id"`
	RoleName    types.String `tfsdk:"role_name"`
	ProjectId   types.String `tfsdk:"project_id"`
	HolderLogin types.String `tfsdk:"holder_login"`
	HolderId    types.String `tfsdk:"holder_id"`
	HolderName  types.String `tfsdk:"holder_name"`
	HolderType  types.String `tfsdk:"holder_type"`
}

// Metadata returns the resource type name.
func (r *projectRoleAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_role_assignment"
}

// Schema defines the schema for the resource.
func (r *projectRoleAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack project role assignment resource. This resource manages role assignments with ONLY project scope in YouTrack, which assigns a role to a user or group within one project. " +
			"Attached to a template project, the assignment is inherited by every project created from that template.",
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
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the project the role is assigned in (e.g. youtrack_project.my_project.id)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *projectRoleAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

// ValidateConfig rejects a holder_type other than 'user' or 'group'.
func (r *projectRoleAssignmentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var holderType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("holder_type"), &holderType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateHolderType(holderType, &resp.Diagnostics)
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectRoleAssignmentResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	apiAssignment := r.buildAPIProjectRoleAssignment(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAssignedRole(ctx, *apiAssignment)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingProjectRoleAssignment,
			fmt.Sprintf("Could not create project role assignment: %v", err),
		)
		return
	}

	if !checkProjectScope(created, &resp.Diagnostics) {
		return
	}

	mapAPIToProjectModel(created, &plan)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *projectRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectRoleAssignmentResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.Id, &resp.Diagnostics, errMissingProjectRoleAssignmentID, errProjectRoleAssignmentIDRequired) {
		return
	}

	apiAssignment, err := r.client.GetAssignedRoleById(ctx, state.Id.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			errReadingProjectRoleAssignment,
			fmt.Sprintf("Could not read project role assignment: %v", err),
		)
		return
	}

	if !checkProjectScope(apiAssignment, &resp.Diagnostics) {
		return
	}

	mapAPIToProjectModel(apiAssignment, &state)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update is never reached with a change: every configurable attribute forces a replacement.
func (r *projectRoleAssignmentResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(errUpdatingProjectRoleAssignment, errProjectRoleAssignmentImmutable)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectRoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectRoleAssignmentResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.Id) {
		return
	}

	err := r.client.DeleteAssignedRole(ctx, state.Id.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			errDeletingProjectRoleAssignment,
			fmt.Sprintf("Could not delete project role assignment: %v", err),
		)
	}
}

// ImportState imports the resource state using the role assignment ID, which Read then resolves.
func (r *projectRoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
