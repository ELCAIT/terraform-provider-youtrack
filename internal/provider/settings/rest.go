// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"context"

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
	_ resource.Resource                = &restSettingsResource{}
	_ resource.ResourceWithConfigure   = &restSettingsResource{}
	_ resource.ResourceWithImportState = &restSettingsResource{}
)

// Error message constants for REST settings operations
const (
	errConvertingRESTSettings      = "Error converting REST settings"
	errUpdatingRESTSettings        = "Error updating REST settings"
	errListToStringSliceConversion = "Failed to convert list to string slice"
	errStringSliceToListConversion = "Failed to convert string slice to list"
)

// NewRestSettingsResource is a helper function to simplify the provider implementation.
func NewRestSettingsResource() resource.Resource {
	return &restSettingsResource{}
}

// restSettingsResource is the resource implementation.
type restSettingsResource struct {
	client *youtrack.Client
}

// restSettingsResourceModel maps the resource schema data.
type restSettingsResourceModel struct {
	ID              types.String `tfsdk:"id"`
	AllowAllOrigins types.Bool   `tfsdk:"allow_all_origins"`
	AllowedOrigins  types.List   `tfsdk:"allowed_origins"`
	LastUpdated     types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *restSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rest_settings"
}

// Schema defines the schema for the resource.
func (r *restSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack REST API settings configuration",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "REST settings identifier",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_all_origins": schema.BoolAttribute{
				Description: "Whether to allow CORS requests from all origins",
				Required:    true,
			},
			"allowed_origins": schema.ListAttribute{
				Description: "List of allowed origins for CORS requests",
				ElementType: types.StringType,
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
func (r *restSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan restSettingsResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	rs, ok := convertModelToRestSettings(plan)
	if !ok {
		resp.Diagnostics.AddError(
			errConvertingRESTSettings,
			errListToStringSliceConversion,
		)
		return
	}

	restSettings, ok := r.updateAndHandleError(ctx, rs, &resp.Diagnostics)
	if !ok {
		return
	}

	if !updateRestSettingsModelWithTimestamp(ctx, restSettings, &plan) {
		resp.Diagnostics.AddError(
			errUpdatingRESTSettings,
			errStringSliceToListConversion,
		)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *restSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state restSettingsResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	restSettings, ok := r.getRestSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	if !updateRestSettingsModelWithTimestamp(ctx, restSettings, &state) {
		resp.Diagnostics.AddError(
			errUpdatingRESTSettings,
			errStringSliceToListConversion,
		)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *restSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan restSettingsResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	rs, ok := convertModelToRestSettings(plan)
	if !ok {
		resp.Diagnostics.AddError(
			errConvertingRESTSettings,
			errListToStringSliceConversion,
		)
		return
	}

	restSettings, ok := r.updateAndHandleError(ctx, rs, &resp.Diagnostics)
	if !ok {
		return
	}

	if !updateRestSettingsModelWithTimestamp(ctx, restSettings, &plan) {
		resp.Diagnostics.AddError(
			errUpdatingRESTSettings,
			errStringSliceToListConversion,
		)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *restSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state restSettingsResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	// Reset to default settings: allow all origins disabled and empty list
	rs := youtrack.RestSettings{
		AllowAllOrigins: false,
		AllowedOrigins:  []string{},
	}

	restSettings, ok := r.updateAndHandleError(ctx, rs, &resp.Diagnostics)
	if !ok {
		return
	}

	if !updateRestSettingsModelWithTimestamp(ctx, restSettings, &state) {
		resp.Diagnostics.AddError(
			errUpdatingRESTSettings,
			errStringSliceToListConversion,
		)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// ImportState imports the resource state.
func (r *restSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	restSettings, ok := r.getRestSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	var state restSettingsResourceModel
	if !updateRestSettingsModelWithTimestamp(ctx, restSettings, &state) {
		resp.Diagnostics.AddError(
			errUpdatingRESTSettings,
			errStringSliceToListConversion,
		)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Configure adds the provider configured client to the resource.
func (r *restSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}
