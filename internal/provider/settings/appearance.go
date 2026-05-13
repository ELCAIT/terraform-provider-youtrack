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
	// Default appearance settings values
	defaultDateFormatID = "youtrack.datefieldformat.iso8601"
	defaultTimeZoneID   = "UTC"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &appearanceSettingsResource{}
	_ resource.ResourceWithConfigure   = &appearanceSettingsResource{}
	_ resource.ResourceWithImportState = &appearanceSettingsResource{}
)

// NewAppearanceSettingsResource is a helper function to simplify the provider implementation.
func NewAppearanceSettingsResource() resource.Resource {
	return &appearanceSettingsResource{}
}

// appearanceSettingsResource is the resource implementation.
type appearanceSettingsResource struct {
	client *youtrack.Client
}

// appearanceSettingsResourceModel maps the resource schema data.
type appearanceSettingsResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	DateFormatID           types.String `tfsdk:"date_format_id"`
	DateFormatPresentation types.String `tfsdk:"date_format_presentation"`
	DateFormatPattern      types.String `tfsdk:"date_format_pattern"`
	DateFormatDatePattern  types.String `tfsdk:"date_format_date_pattern"`
	TimeZoneID             types.String `tfsdk:"time_zone_id"`
	TimeZonePresentation   types.String `tfsdk:"time_zone_presentation"`
	TimeZoneOffset         types.Int64  `tfsdk:"time_zone_offset"`
	LastUpdated            types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *appearanceSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_appearance_settings"
}

// Schema defines the schema for the resource.
func (r *appearanceSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack appearance settings configuration",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Appearance settings identifier",
				Computed:    true,
			},
			"date_format_id": schema.StringAttribute{
				Description: "Date format identifier",
				Required:    true,
			},
			"date_format_presentation": schema.StringAttribute{
				Description: "Date format presentation string",
				Computed:    true,
			},
			"date_format_pattern": schema.StringAttribute{
				Description: "Date format pattern (e.g., 'yyyy-MM-dd', 'dd/MM/yyyy HH:mm', 'MM/dd/yyyy')",
				Computed:    true,
			},
			"date_format_date_pattern": schema.StringAttribute{
				Description: "Date format date pattern",
				Computed:    true,
			},
			"time_zone_id": schema.StringAttribute{
				Description: "Time zone identifier (e.g., 'Europe/Zurich', 'America/New_York')",
				Required:    true,
			},
			"time_zone_presentation": schema.StringAttribute{
				Description: "Time zone presentation string",
				Computed:    true,
			},
			"time_zone_offset": schema.Int64Attribute{
				Description: "Time zone offset in milliseconds",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last update",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *appearanceSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appearanceSettingsResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	as := convertModelToAppearanceSettings(plan)

	appearanceSettings, ok := r.updateAndHandleError(ctx, as, &resp.Diagnostics)
	if !ok {
		return
	}

	updateAppearanceSettingsModelWithTimestamp(appearanceSettings, &plan)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *appearanceSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appearanceSettingsResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	appearanceSettings, ok := r.getAppearanceSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	updateAppearanceSettingsModelWithTimestamp(appearanceSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *appearanceSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appearanceSettingsResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	as := convertModelToAppearanceSettings(plan)

	appearanceSettings, ok := r.updateAndHandleError(ctx, as, &resp.Diagnostics)
	if !ok {
		return
	}

	// Map response body to model and update timestamp
	updateAppearanceSettingsModelWithTimestamp(appearanceSettings, &plan)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *appearanceSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appearanceSettingsResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	var as = youtrack.AppearanceSettings{
		DateFormat: youtrack.DateFormatDescriptor{
			ID: defaultDateFormatID,
		},
		TimeZone: youtrack.TimeZoneDescriptor{
			ID: defaultTimeZoneID,
		},
	}

	appearanceSettings, ok := r.updateAndHandleError(ctx, as, &resp.Diagnostics)
	if !ok {
		return
	}

	updateAppearanceSettingsModelWithTimestamp(appearanceSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// ImportState imports the resource state.
func (r *appearanceSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	appearanceSettings, ok := r.getAppearanceSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	var state appearanceSettingsResourceModel
	updateAppearanceSettingsModelWithTimestamp(appearanceSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Configure adds the provider configured client to the resource.
func (r *appearanceSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}
