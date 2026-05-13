// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &oauth2AuthModuleResource{}
	_ resource.ResourceWithConfigure   = &oauth2AuthModuleResource{}
	_ resource.ResourceWithImportState = &oauth2AuthModuleResource{}
)

// NewOAuth2AuthModuleResource is a helper function to simplify the provider implementation.
func NewOAuth2AuthModuleResource() resource.Resource {
	return &oauth2AuthModuleResource{}
}

// oauth2AuthModuleResource is the resource implementation.
type oauth2AuthModuleResource struct {
	client *youtrack.Client
}

// Metadata returns the resource type name.
func (r *oauth2AuthModuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_module_oauth2"
}

// Schema defines the schema for the resource.
func (r *oauth2AuthModuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hub OAuth 2.0 authentication module. " +
			"This resource allows configuring an external OAuth 2.0 identity provider for Hub.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Hub ID of the auth module (computed).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the auth module.",
			},
			"disabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the auth module is disabled. Defaults to false.",
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "OAuth 2.0 client ID registered with the identity provider.",
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "OAuth 2.0 client secret. This value is write-only and is not returned by the API.",
			},
			"server_url": schema.StringAttribute{
				Required:    true,
				Description: "Base URL of the OAuth 2.0 authorization server (identity provider).",
			},
			"token_url": schema.StringAttribute{
				Required:    true,
				Description: "URL of the token endpoint at the identity provider.",
			},
			"scope": schema.StringAttribute{
				Optional:    true,
				Description: "Space-separated list of OAuth 2.0 scopes to request.",
			},
			"user_info_url": schema.StringAttribute{
				Optional:    true,
				Description: "URL of the userinfo endpoint at the identity provider.",
			},
			"redirect_uri": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "OAuth 2.0 redirect URI. When omitted, Hub sets this automatically.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"form_client_auth": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to send client credentials in the request body instead of the Authorization header. Defaults to false.",
			},
			"email_verified_by_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether email addresses from this provider are considered verified by default. Defaults to false.",
			},
			"allowed_create_new_users": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether new users may be created on first login via this module. Defaults to false.",
			},
			"background_sync_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether background synchronisation with the identity provider is enabled. Defaults to false.",
			},
			"idp_logout_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "URL of the identity provider logout endpoint.",
			},
			"user_id_path": schema.StringAttribute{
				Required:    true,
				Description: "JSON path expression to the user ID claim in the identity provider response.",
			},
			"user_email_path": schema.StringAttribute{
				Optional:    true,
				Description: "JSON path expression to the email claim in the identity provider response.",
			},
			"user_email_verified_path": schema.StringAttribute{
				Optional:    true,
				Description: "JSON path expression to the email-verified claim in the identity provider response.",
			},
			"user_name_path": schema.StringAttribute{
				Optional:    true,
				Description: "JSON path expression to the username claim in the identity provider response.",
			},
			"full_name_path": schema.StringAttribute{
				Optional:    true,
				Description: "JSON path expression to the full name claim in the identity provider response.",
			},
			"user_email_url": schema.StringAttribute{
				Optional:    true,
				Description: "URL used by Hub to retrieve the user's email address from the identity provider.",
			},
			"user_avatar_url": schema.StringAttribute{
				Optional:    true,
				Description: "URL used by Hub to retrieve the user's avatar from the identity provider.",
			},
			"user_picture_id_path": schema.StringAttribute{
				Optional:    true,
				Description: "JSON path expression to the picture ID claim in the identity provider response.",
			},
			"user_picture_url_pattern": schema.StringAttribute{
				Optional:    true,
				Description: "URL pattern used to build the picture URL from the picture ID.",
			},
			"user_groups_path": schema.StringAttribute{
				Optional:    true,
				Description: "JSON path expression to the groups claim in the identity provider response.",
			},
			"icon_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "URL of a custom icon to display for this auth module.",
			},
			"extension_grant_type": schema.StringAttribute{
				Optional:    true,
				Description: "Custom OAuth 2.0 extension grant type.",
			},
			"connection_timeout": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Connection timeout in milliseconds when contacting the identity provider.",
			},
			"read_timeout": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Read timeout in milliseconds when contacting the identity provider.",
			},
			"sync_interval": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Cron expression that controls background synchronisation frequency.",
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this auth module is the default login method. Defaults to false.",
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *oauth2AuthModuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := helpers.GetClientFromConfigure(req, resp)
	if !ok {
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *oauth2AuthModuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan oauth2AuthModuleResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	apiModule := plan.toAPIModel()
	created, err := r.client.CreateOAuth2AuthModule(ctx, apiModule)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingOAuth2Module,
			fmt.Sprintf("Could not create OAuth2 auth module: %v", err),
		)
		return
	}

	// Preserve client_secret from plan since the API does not return it.
	plan.fromAPIModel(created)
	plan.ClientSecret = types.StringValue(apiModule.ClientSecret)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *oauth2AuthModuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state oauth2AuthModuleResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingOAuth2ModuleID, errOAuth2ModuleIDRequired) {
		return
	}

	apiModule, err := r.client.GetOAuth2AuthModuleByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			errReadingOAuth2Module,
			fmt.Sprintf("Could not read OAuth2 auth module: %v", err),
		)
		return
	}

	// Preserve client_secret from existing state; the API never returns it.
	existingSecret := state.ClientSecret
	state.fromAPIModel(apiModule)
	state.ClientSecret = existingSecret

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *oauth2AuthModuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan oauth2AuthModuleResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	moduleID := plan.ID.ValueString()
	apiModule := plan.toAPIModel()

	updated, err := r.client.UpdateOAuth2AuthModule(ctx, moduleID, apiModule)
	if err != nil {
		resp.Diagnostics.AddError(
			errUpdatingOAuth2Module,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "oauth2 auth module", err),
		)
		return
	}

	// Preserve client_secret from plan since the API does not return it.
	plan.fromAPIModel(updated)
	plan.ClientSecret = types.StringValue(apiModule.ClientSecret)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *oauth2AuthModuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state oauth2AuthModuleResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	err := r.client.DeleteOAuth2AuthModule(ctx, state.ID.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			errDeletingOAuth2Module,
			fmt.Sprintf("Could not delete OAuth2 auth module: %v", err),
		)
		return
	}
}

// ImportState imports the resource state by module ID.
func (r *oauth2AuthModuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	moduleID := strings.TrimSpace(req.ID)

	apiModule, err := r.client.GetOAuth2AuthModuleByID(ctx, moduleID)
	if err != nil {
		resp.Diagnostics.AddError(
			errImportingOAuth2Module,
			fmt.Sprintf("Could not read OAuth2 auth module with ID '%s': %v", moduleID, err),
		)
		return
	}

	var state oauth2AuthModuleResourceModel
	state.fromAPIModel(apiModule)
	// client_secret cannot be imported; set to empty string as a placeholder.
	state.ClientSecret = types.StringValue("")

	helpers.SetStateAndCheckError(ctx, resp, &state)
}
