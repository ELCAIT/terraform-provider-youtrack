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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &userResource{}
	_ resource.ResourceWithConfigure      = &userResource{}
	_ resource.ResourceWithImportState    = &userResource{}
	_ resource.ResourceWithValidateConfig = &userResource{}
)

func NewUserResource() resource.Resource {
	return &userResource{}
}

type userResource struct {
	client *youtrack.Client
}

type userResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Login    types.String `tfsdk:"login"`
	FullName types.String `tfsdk:"full_name"`
	Email    types.String `tfsdk:"email"`
	Password types.String `tfsdk:"password"`
	Banned   types.Bool   `tfsdk:"banned"`
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a YouTrack user account, for example a service account used by an integration. " +
			"Deleting the resource deletes the account and hands its content over to the guest user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"login": schema.StringAttribute{
				Required:    true,
				Description: "The login (username) of the user.",
			},
			"full_name": schema.StringAttribute{
				Required:    true,
				Description: "The display name of the user.",
			},
			"email": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "The email address of the user. An address cannot be cleared once set, " +
					"so removing this attribute from the configuration leaves the current address in place.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Description: "The password of the user. YouTrack requires one to create an account, even a service " +
					"account that only authenticates with a token. YouTrack never returns it, so Terraform cannot " +
					"detect a password changed outside of Terraform, and the first apply after an import sets it.",
			},
			"banned": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Whether the user is banned. A banned user cannot log in.",
				PlanModifiers: preserveBoolPlanModifiers,
			},
		},
	}
}

func (r *userResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config userResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	helpers.ValidateEmailField(config.Email, path.Root("email"), "email", &resp.Diagnostics)
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	created, err := r.client.CreateUser(ctx, plan.toAPIModel())
	if err != nil {
		resp.Diagnostics.AddError(errCreatingUser, fmt.Sprintf("Could not create user: %v", err))
		return
	}

	plan.fromAPIModel(created)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingUserID, errUserIDRequired) {
		return
	}

	user, err := r.client.GetUserByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(errReadingUserResource, fmt.Sprintf("Could not read user: %v", err))
		return
	}

	state.fromAPIModel(user)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior userResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !helpers.ValidateResourceID(plan.ID, &resp.Diagnostics, errMissingUserID, errUserIDRequired) {
		return
	}

	updated, err := r.applyUserUpdate(ctx, plan, prior)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingUser, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "user", err))
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	if err := r.client.DeleteUser(ctx, state.ID.ValueString()); err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(errDeletingUser, fmt.Sprintf("Could not delete user: %v", err))
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
