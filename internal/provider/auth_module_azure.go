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
	_ resource.Resource                = &azureAuthModuleResource{}
	_ resource.ResourceWithConfigure   = &azureAuthModuleResource{}
	_ resource.ResourceWithImportState = &azureAuthModuleResource{}
)

// NewAzureAuthModuleResource is a helper function to simplify the provider implementation.
func NewAzureAuthModuleResource() resource.Resource {
	return &azureAuthModuleResource{}
}

// azureAuthModuleResource is the resource implementation.
type azureAuthModuleResource struct {
	client *youtrack.Client
}

// Metadata returns the resource type name.
func (r *azureAuthModuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_module_azure"
}

// Schema defines the schema for the resource.
func (r *azureAuthModuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hub Microsoft Entra ID (formerly Azure AD) authentication module. " +
			"This resource allows configuring Microsoft Entra ID as an external identity provider for Hub.",
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
				Description: "Application (client) ID of the Entra ID app registration.",
			},
			"client_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Client secret of the Entra ID app registration. This value is write-only and is not returned by the API.",
			},
			"tenant": schema.StringAttribute{
				Optional:    true,
				Description: "Entra ID tenant (directory) ID, a GUID. Leave unset to allow sign-in from any Microsoft tenant (\"common\").",
			},
			"server_url": schema.StringAttribute{
				Computed:    true,
				Description: "Authorization endpoint URL, derived by Hub from tenant (computed).",
			},
			"redirect_uri": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "OAuth 2.0 redirect URI. When omitted, Hub sets this automatically.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allowed_create_new_users": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether new users may be created on first login via this module. Defaults to false.",
			},
			"background_sync_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether background synchronisation with Entra ID is enabled. Defaults to false.",
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
				Description: "Connection timeout in milliseconds when contacting Entra ID.",
			},
			"read_timeout": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Read timeout in milliseconds when contacting Entra ID.",
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
			"request_group_permission": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Hub requests Microsoft Graph group-read permissions for group synchronisation. Defaults to false.",
			},
			"request_id_token": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether Hub requests an ID token from Entra ID. Defaults to false.",
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *azureAuthModuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := helpers.GetClientFromConfigure(req, resp)
	if !ok {
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *azureAuthModuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan azureAuthModuleResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	apiModule := plan.toAPIModel()
	created, err := r.client.CreateAzureAuthModule(ctx, apiModule)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingAzureModule,
			fmt.Sprintf("Could not create Azure auth module: %v", err),
		)
		return
	}

	// Hub always creates Azure auth modules with allowedCreateNewUsers=true,
	// regardless of what was requested (same behavior as the OAuth2 module);
	// correct it with a follow-up update when the plan asked for something
	// else, otherwise Terraform reports "provider produced inconsistent
	// result after apply".
	if created.AllowedCreateNewUsers != apiModule.AllowedCreateNewUsers {
		created.AllowedCreateNewUsers = apiModule.AllowedCreateNewUsers

		created, err = r.client.UpdateAzureAuthModule(ctx, created.ID, *created)
		if err != nil {
			resp.Diagnostics.AddError(
				errCreatingAzureModule,
				fmt.Sprintf("Could not enforce allowed_create_new_users after create: %v", err),
			)
			return
		}
	}

	// Preserve client_secret from plan since the API does not return it.
	plan.fromAPIModel(created)
	plan.ClientSecret = types.StringValue(apiModule.ClientSecret)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *azureAuthModuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state azureAuthModuleResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingAzureModuleID, errAzureModuleIDRequired) {
		return
	}

	apiModule, err := r.client.GetAzureAuthModuleByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			errReadingAzureModule,
			fmt.Sprintf("Could not read Azure auth module: %v", err),
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
func (r *azureAuthModuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan azureAuthModuleResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	moduleID := plan.ID.ValueString()
	apiModule := plan.toAPIModel()

	updated, err := r.client.UpdateAzureAuthModule(ctx, moduleID, apiModule)
	if err != nil {
		resp.Diagnostics.AddError(
			errUpdatingAzureModule,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "azure auth module", err),
		)
		return
	}

	// Preserve client_secret from plan since the API does not return it.
	plan.fromAPIModel(updated)
	plan.ClientSecret = types.StringValue(apiModule.ClientSecret)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *azureAuthModuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state azureAuthModuleResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	err := r.client.DeleteAzureAuthModule(ctx, state.ID.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			errDeletingAzureModule,
			fmt.Sprintf("Could not delete Azure auth module: %v", err),
		)
		return
	}
}

// ImportState imports the resource state by module ID.
func (r *azureAuthModuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	moduleID := strings.TrimSpace(req.ID)

	apiModule, err := r.client.GetAzureAuthModuleByID(ctx, moduleID)
	if err != nil {
		resp.Diagnostics.AddError(
			errImportingAzureModule,
			fmt.Sprintf("Could not read Azure auth module with ID '%s': %v", moduleID, err),
		)
		return
	}

	var state azureAuthModuleResourceModel
	state.fromAPIModel(apiModule)
	// client_secret cannot be imported; set to empty string as a placeholder.
	state.ClientSecret = types.StringValue("")

	helpers.SetStateAndCheckError(ctx, resp, &state)
}
