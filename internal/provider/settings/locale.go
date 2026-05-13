// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"context"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// Default locale settings values
	defaultLocaleID   = "en_US"
	defaultLocaleCode = "en"
	defaultLocaleName = "English"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &localeSettingsResource{}
	_ resource.ResourceWithConfigure   = &localeSettingsResource{}
	_ resource.ResourceWithImportState = &localeSettingsResource{}
)

// NewLocaleSettingsResource is a helper function to simplify the provider implementation.
func NewLocaleSettingsResource() resource.Resource {
	return &localeSettingsResource{}
}

// localeSettingsResource is the resource implementation.
type localeSettingsResource struct {
	client *youtrack.Client
}

// localeSettingsResourceModel maps the resource schema data.
type localeSettingsResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Locale      types.String `tfsdk:"locale"`
	Language    types.String `tfsdk:"language"`
	Community   types.Bool   `tfsdk:"community"`
	Name        types.String `tfsdk:"name"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *localeSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_locale_settings"
}

// Schema defines the schema for the resource.
func (r *localeSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack locale settings configuration",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Locale identifier (e.g., 'en_US', 'fr_FR')",
				Required:    true,
			},
			"locale": schema.StringAttribute{
				Description: "Locale code (e.g., 'en_US', 'fr_FR')",
				Required:    true,
			},
			"language": schema.StringAttribute{
				Description: "Language code (e.g., 'en', 'fr')",
				Required:    true,
			},
			"community": schema.BoolAttribute{
				Description: "Whether this is a community-contributed locale",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Display name of the locale (e.g., 'English', 'French')",
				Required:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last update",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *localeSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan localeSettingsResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	ls := convertModelToLocaleSettings(plan)

	localeSettings, ok := r.updateAndHandleError(ctx, ls, &resp.Diagnostics)
	if !ok {
		return
	}

	updateLocaleSettingsModelWithTimestamp(localeSettings, &plan)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *localeSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state localeSettingsResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	localeSettings, ok := r.getLocaleSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	updateLocaleSettingsModelWithTimestamp(localeSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *localeSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan localeSettingsResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	ls := convertModelToLocaleSettings(plan)

	localeSettings, ok := r.updateAndHandleError(ctx, ls, &resp.Diagnostics)
	if !ok {
		return
	}

	// Map response body to model and update timestamp
	updateLocaleSettingsModelWithTimestamp(localeSettings, &plan)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *localeSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state localeSettingsResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	var ls = youtrack.LocaleSettings{
		Locale: youtrack.LocaleDescriptor{
			ID:        defaultLocaleID,
			Locale:    defaultLocaleID,
			Language:  defaultLocaleCode,
			Community: false,
			Name:      defaultLocaleName,
		},
	}

	localeSettings, ok := r.updateAndHandleError(ctx, ls, &resp.Diagnostics)
	if !ok {
		return
	}

	updateLocaleSettingsModelWithTimestamp(localeSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// ImportState imports the resource state.
func (r *localeSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	localeSettings, ok := r.getLocaleSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	var state localeSettingsResourceModel
	updateLocaleSettingsModelWithTimestamp(localeSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Configure adds the provider configured client to the resource.
func (r *localeSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}
