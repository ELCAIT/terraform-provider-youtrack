// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
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
	_ resource.Resource                   = &projectAppResource{}
	_ resource.ResourceWithConfigure      = &projectAppResource{}
	_ resource.ResourceWithImportState    = &projectAppResource{}
	_ resource.ResourceWithValidateConfig = &projectAppResource{}
)

// NewProjectAppResource is a helper function to simplify the provider implementation.
func NewProjectAppResource() resource.Resource {
	return &projectAppResource{}
}

type projectAppResource struct {
	client *youtrack.Client
}

type projectAppResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	AppName   types.String `tfsdk:"app_name"`
	AppID     types.String `tfsdk:"app_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
}

func (r *projectAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_app"
}

func (r *projectAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack project app resource. Attaches an installed app to a project and enables or disables it there. " +
			"Workflows are apps too, so this is also how a workflow is attached to a project. " +
			"Attaching an app to a template project makes projects created from that template inherit it. " +
			"An app that is already attached to the project is adopted instead of attached a second time, " +
			"and destroying the resource detaches it whether or not Terraform attached it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The entity ID of the app usage, the record that attaches the app to the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			projectAppAttrProjectID: schema.StringAttribute{
				Required:    true,
				Description: "The entity ID of the project to attach the app to (for example `0-1`), not its short name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			projectAppAttrAppName: schema.StringAttribute{
				Required: true,
				Description: "The name of the installed app, as listed by `GET /api/admin/apps` (for example " +
					"`@jetbrains/youtrack-workflow-due-date`). This is the package name, not the title shown in the UI.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"app_id": schema.StringAttribute{
				Computed:    true,
				Description: "The entity ID of the app. It changes when the app is uninstalled and installed again.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the app is enabled in the project. Defaults to `true`; an app disabled outside of Terraform is enabled again on the next apply.",
			},
		},
	}
}

func (r *projectAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *projectAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectAppResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(errCreatingProjectApp, fmt.Sprintf("Could not attach app to project: %v", err))
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *projectAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectAppResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ProjectID, &resp.Diagnostics, errMissingProjectAppProjectID, errProjectAppProjectIDRequired) {
		return
	}

	appID, usage, err := r.findUsage(ctx, state)
	if err != nil {
		if youtrack.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(errReadingProjectApp, fmt.Sprintf("Could not read project app: %v", err))
		return
	}

	if usage == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.fromAPIModel(appID, usage)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *projectAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectAppResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	if !helpers.ValidateResourceID(plan.ProjectID, &resp.Diagnostics, errMissingProjectAppProjectID, errProjectAppProjectIDRequired) {
		return
	}

	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(errUpdatingProjectApp, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "project app", err))
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *projectAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectAppResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ProjectID) {
		return
	}

	if err := r.detach(ctx, state); err != nil {
		resp.Diagnostics.AddError(errDeletingProjectApp, fmt.Sprintf("Could not detach app from project: %v", err))
	}
}

// ValidateConfig rejects a blank project_id or app_name before any API call is made.
func (r *projectAppResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config projectAppResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for attribute, value := range map[string]types.String{projectAppAttrProjectID: config.ProjectID, projectAppAttrAppName: config.AppName} {
		if !value.IsNull() && !value.IsUnknown() && strings.TrimSpace(value.ValueString()) == "" {
			resp.Diagnostics.AddAttributeError(path.Root(attribute), errInvalidProjectAppConfig, fmt.Sprintf(errProjectAppAttributeEmptyFmt, attribute))
		}
	}
}

func (r *projectAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectID, appName, ok := parseProjectAppImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(errMissingProjectAppProjectID, errProjectAppImportIDInvalid)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(projectAppAttrProjectID), projectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(projectAppAttrAppName), appName)...)
}
