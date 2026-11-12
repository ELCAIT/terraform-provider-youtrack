// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"context"

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
	_ resource.Resource                = &mailboxIntegrationSettingsResource{}
	_ resource.ResourceWithConfigure   = &mailboxIntegrationSettingsResource{}
	_ resource.ResourceWithImportState = &mailboxIntegrationSettingsResource{}
)

// NewMailboxIntegrationSettingsResource is a helper function to simplify the provider implementation.
func NewMailboxIntegrationSettingsResource() resource.Resource {
	return &mailboxIntegrationSettingsResource{}
}

// mailboxIntegrationSettingsResource is the resource implementation.
type mailboxIntegrationSettingsResource struct {
	client *youtrack.Client
}

// mailboxIntegrationSettingsResourceModel maps the resource schema data.
type mailboxIntegrationSettingsResourceModel struct {
	ID             types.String `tfsdk:"id"`
	CronExpression types.String `tfsdk:"cron_expression"`
}

// Metadata returns the resource type name.
func (r *mailboxIntegrationSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mailbox_integration_settings"
}

// Schema defines the schema for the resource.
func (r *mailboxIntegrationSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the instance-wide settings of the YouTrack mailbox integration: the schedule on which " +
			"YouTrack polls every mailbox. This is a singleton; destroying it restores the default schedule (" +
			"`" + defaultMailboxCronExpression + "`, every two minutes).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Mailbox integration settings identifier, always `global`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cron_expression": schema.StringAttribute{
				Description: "Quartz cron expression on which YouTrack polls every mailbox, such as `0 0/5 * * * ?` for every five minutes.",
				Required:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *mailboxIntegrationSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

// Create applies the schedule and sets the initial Terraform state.
func (r *mailboxIntegrationSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mailboxIntegrationSettingsResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	settings, ok := r.updateCron(ctx, plan.CronExpression.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	plan.fromAPIModel(settings)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *mailboxIntegrationSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mailboxIntegrationSettingsResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	settings, ok := r.readCron(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	state.fromAPIModel(settings)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update applies the schedule and sets the updated Terraform state on success.
func (r *mailboxIntegrationSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mailboxIntegrationSettingsResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	settings, ok := r.updateCron(ctx, plan.CronExpression.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	plan.fromAPIModel(settings)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete restores YouTrack's default schedule.
func (r *mailboxIntegrationSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mailboxIntegrationSettingsResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	r.updateCron(ctx, defaultMailboxCronExpression, &resp.Diagnostics)
}

// ImportState imports the singleton; the import ID is ignored.
func (r *mailboxIntegrationSettingsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	settings, ok := r.readCron(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	var state mailboxIntegrationSettingsResourceModel
	state.fromAPIModel(settings)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}
