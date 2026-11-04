// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	_ resource.Resource                   = &appGlobalConfigResource{}
	_ resource.ResourceWithConfigure      = &appGlobalConfigResource{}
	_ resource.ResourceWithImportState    = &appGlobalConfigResource{}
	_ resource.ResourceWithValidateConfig = &appGlobalConfigResource{}
)

// NewAppGlobalConfigResource is a helper function to simplify the provider implementation.
func NewAppGlobalConfigResource() resource.Resource {
	return &appGlobalConfigResource{}
}

type appGlobalConfigResource struct {
	client *youtrack.Client
}

type appGlobalConfigResourceModel struct {
	ID                types.String `tfsdk:"id"`
	AppName           types.String `tfsdk:"app_name"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	Settings          types.String `tfsdk:"settings"`
	PermittedGroupIDs types.Set    `tfsdk:"permitted_group_ids"`
}

func (r *appGlobalConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_global_config"
}

func (r *appGlobalConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack app global configuration resource. Enables or disables an installed app, sets its global settings " +
			"(the ones an administrator edits under Administration > Apps) and chooses who may use it. " +
			"The app itself is not installed by this resource. " +
			"Destroying the resource only removes it from Terraform: the app keeps its configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The entity ID of the app. It changes when the app is uninstalled and installed again.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			appGlobalConfigAttrAppName: schema.StringAttribute{
				Required: true,
				Description: "The name of the installed app, as listed by `GET /api/admin/apps` (for example " +
					"`@jetbrains/youtrack-workflow-due-date`). This is the package name, not the title shown in the UI.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the app is enabled. Defaults to `true`; an app disabled outside of Terraform is enabled again on the next apply.",
			},
			appGlobalConfigAttrSettings: schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "The global settings of the app, as a JSON object. Each app declares its own in its `settings.json`, " +
					"so the keys depend on the app; build the value with `jsonencode()`. " +
					"The whole document is written whenever this value changes: a key left out is cleared, a secret included. " +
					"An entity setting, such as a user group, is written as `{ id = \"...\" }`. " +
					"Only the keys written here are compared with YouTrack. " +
					"YouTrack leaves out a setting written as `null` or `\"\"`, which counts as unchanged, and stores a list of entities sorted and without duplicates, so their order does not matter. " +
					"It also drops a key the app does not declare, which then shows as a change on every plan: check its spelling against the app's `settings.json`. " +
					"Settings the app declares as secrets are never returned by YouTrack, so a secret changed outside of Terraform is not detected, " +
					"and after an import the first apply writes the value from the configuration. " +
					"Leave it out to leave the settings of the app alone.",
			},
			"permitted_group_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The entity IDs of the user groups allowed to use the app, for example `6-0` for all registered users. " +
					"Users allowed one by one are left as they are. Leave it out to leave who may use the app alone.",
			},
		},
	}
}

func (r *appGlobalConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *appGlobalConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appGlobalConfigResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	if err := r.apply(ctx, &plan, nil); err != nil {
		resp.Diagnostics.AddError(errCreatingAppGlobalConfig, fmt.Sprintf("Could not configure app: %v", err))
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *appGlobalConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appGlobalConfigResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.AppName, &resp.Diagnostics, errMissingAppGlobalConfig, errAppGlobalConfigAppNameRequired) {
		return
	}

	appID, config, err := r.readConfig(ctx, state)
	if err != nil {
		if youtrack.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(errReadingAppGlobalConfig, fmt.Sprintf(errAppGlobalConfigReadFmt, err))
		return
	}

	if err = state.fromAPIModel(ctx, appID, config); err != nil {
		resp.Diagnostics.AddError(errReadingAppGlobalConfig, fmt.Sprintf(errAppGlobalConfigReadFmt, err))
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *appGlobalConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appGlobalConfigResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	var state appGlobalConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !helpers.ValidateResourceID(plan.AppName, &resp.Diagnostics, errMissingAppGlobalConfig, errAppGlobalConfigAppNameRequired) {
		return
	}

	if err := r.apply(ctx, &plan, &state); err != nil {
		resp.Diagnostics.AddError(errUpdatingAppGlobalConfig, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "app configuration", err))
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

// Delete only removes the resource from state: clearing the settings would also
// clear secrets such as an API token, and the app may be configured elsewhere.
func (r *appGlobalConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ValidateConfig rejects a blank app_name and settings that are not a JSON
// object before any API call is made.
func (r *appGlobalConfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config appGlobalConfigResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	helpers.ValidateNotBlank(map[string]types.String{appGlobalConfigAttrAppName: config.AppName}, errInvalidAppGlobalConfig, &resp.Diagnostics)

	if config.Settings.IsNull() || config.Settings.IsUnknown() {
		return
	}

	var settings map[string]any
	if err := json.Unmarshal([]byte(config.Settings.ValueString()), &settings); err != nil || settings == nil {
		resp.Diagnostics.AddAttributeError(path.Root(appGlobalConfigAttrSettings), errInvalidAppGlobalConfig, errAppGlobalConfigSettingsInvalid)
	}
}

// ImportState takes the name of the app. settings and permitted_group_ids stay
// unset, so the first apply writes the ones from the configuration.
func (r *appGlobalConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	appName := strings.TrimSpace(req.ID)
	if appName == "" {
		resp.Diagnostics.AddError(errInvalidAppGlobalConfig, "Invalid import ID. Expected: {app_name}")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(appGlobalConfigAttrAppName), appName)...)
}
