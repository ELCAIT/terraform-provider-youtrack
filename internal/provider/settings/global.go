package settings

import (
	"context"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &globalSettingsResource{}
	_ resource.ResourceWithConfigure   = &globalSettingsResource{}
	_ resource.ResourceWithImportState = &globalSettingsResource{}
)

// NewGlobalSettingsResource is a helper function to simplify the provider implementation.
func NewGlobalSettingsResource() resource.Resource {
	return &globalSettingsResource{}
}

// globalSettingsResource is the resource implementation.
type globalSettingsResource struct {
	client *youtrack.Client
}

type globalSettingsResourceModel struct {
	ID          types.String `tfsdk:"id"`
	License     types.String `tfsdk:"license"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *globalSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_settings"
}

// Schema defines the schema for the resource.
func (r *globalSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack global settings resource. This resource manages the global settings in YouTrack, specifically the license configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the global settings configuration.",
				Computed:    true,
			},
			"license": schema.StringAttribute{
				Description: "The license key for the YouTrack instance. Can be null if no license is configured.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last update",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *globalSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan globalSettingsResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	gs, ok := r.buildGlobalSettings(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}

	globalSettings, ok := r.updateAndHandleError(ctx, gs, &resp.Diagnostics)
	if !ok {
		return
	}

	updateGlobalSettingsModelWithTimestamp(globalSettings, &plan)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *globalSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state globalSettingsResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	globalSettings, ok := r.getGlobalSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	updateGlobalSettingsModelWithTimestamp(globalSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *globalSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan globalSettingsResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	gs, ok := r.buildGlobalSettings(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}

	globalSettings, ok := r.updateAndHandleError(ctx, gs, &resp.Diagnostics)
	if !ok {
		return
	}

	// Map response body to model and update timestamp
	updateGlobalSettingsModelWithTimestamp(globalSettings, &plan)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *globalSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state globalSettingsResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	gs := youtrack.GlobalSettings{
		License: convertModelToGlobalSettings(state).License,
	}

	globalSettings, ok := r.updateAndHandleError(ctx, gs, &resp.Diagnostics)
	if !ok {
		return
	}

	updateGlobalSettingsModelWithTimestamp(globalSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// ImportState imports the resource state.
func (r *globalSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	globalSettings, ok := r.getGlobalSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	var state globalSettingsResourceModel
	updateGlobalSettingsModelWithTimestamp(globalSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Configure adds the provider configured client to the resource.
func (r *globalSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}
