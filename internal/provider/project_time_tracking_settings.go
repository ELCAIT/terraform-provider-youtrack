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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &projectTimeTrackingSettingsResource{}
	_ resource.ResourceWithConfigure   = &projectTimeTrackingSettingsResource{}
	_ resource.ResourceWithImportState = &projectTimeTrackingSettingsResource{}
)

const (
	errCreatingProjectTimeTracking  = "Error creating project time tracking settings"
	errReadingProjectTimeTracking   = "Error reading project time tracking settings"
	errUpdatingProjectTimeTracking  = "Error updating project time tracking settings"
	errMissingProjectTimeTrackingID = "Missing project ID for time tracking settings"

	errProjectTimeTrackingIDRequired = "Project ID is required for time tracking settings"
)

// NewProjectTimeTrackingSettingsResource is a helper function to simplify the provider implementation.
func NewProjectTimeTrackingSettingsResource() resource.Resource {
	return &projectTimeTrackingSettingsResource{}
}

type projectTimeTrackingSettingsResource struct {
	client *youtrack.Client
}

type projectTimeTrackingSettingsResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	EstimateFieldName  types.String `tfsdk:"estimate_field_name"`
	TimeSpentFieldName types.String `tfsdk:"time_spent_field_name"`
}

func (r *projectTimeTrackingSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_time_tracking_settings"
}

func (r *projectTimeTrackingSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack project time tracking settings resource. Manages time tracking configuration for a project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The entity ID of the project time tracking settings.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The entity ID of the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Required:    true,
				Description: "Whether time tracking is enabled for this project.",
			},
			"estimate_field_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the custom field attached to the project to use for estimation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"time_spent_field_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the custom field attached to the project to use for tracking spent time.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *projectTimeTrackingSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *projectTimeTrackingSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectTimeTrackingSettingsResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	payload, err := r.toUpdatePayload(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(errCreatingProjectTimeTracking, err.Error())
		return
	}

	updated, err := r.client.UpdateProjectTimeTrackingSettings(ctx, plan.ProjectID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingProjectTimeTracking,
			fmt.Sprintf("Could not configure project time tracking settings: %v", err),
		)
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *projectTimeTrackingSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectTimeTrackingSettingsResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ProjectID, &resp.Diagnostics, errMissingProjectTimeTrackingID, errProjectTimeTrackingIDRequired) {
		return
	}

	settings, err := r.client.GetProjectTimeTrackingSettings(ctx, state.ProjectID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			errReadingProjectTimeTracking,
			fmt.Sprintf("Could not read project time tracking settings: %v", err),
		)
		return
	}

	state.fromAPIModel(settings)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *projectTimeTrackingSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectTimeTrackingSettingsResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	if !helpers.ValidateResourceID(plan.ProjectID, &resp.Diagnostics, errMissingProjectTimeTrackingID, errProjectTimeTrackingIDRequired) {
		return
	}

	payload, err := r.toUpdatePayload(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingProjectTimeTracking, err.Error())
		return
	}

	updated, err := r.client.UpdateProjectTimeTrackingSettings(ctx, plan.ProjectID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			errUpdatingProjectTimeTracking,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "project time tracking settings", err),
		)
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

// Delete resets time tracking to disabled since the settings object is a singleton per project.
func (r *projectTimeTrackingSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectTimeTrackingSettingsResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ProjectID) {
		return
	}

	payload := youtrack.ProjectTimeTrackingUpdatePayload{Enabled: false}
	_, err := r.client.UpdateProjectTimeTrackingSettings(ctx, state.ProjectID.ValueString(), payload)
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Error resetting project time tracking settings",
			fmt.Sprintf("Could not disable project time tracking: %v", err),
		)
	}
}

func (r *projectTimeTrackingSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), req.ID)...)
}

func (r *projectTimeTrackingSettingsResource) toUpdatePayload(ctx context.Context, m projectTimeTrackingSettingsResourceModel) (youtrack.ProjectTimeTrackingUpdatePayload, error) {
	payload := youtrack.ProjectTimeTrackingUpdatePayload{
		Enabled: m.Enabled.ValueBool(),
	}

	if !m.EstimateFieldName.IsNull() && !m.EstimateFieldName.IsUnknown() && m.EstimateFieldName.ValueString() != "" {
		f, err := r.client.GetProjectCustomFieldByName(ctx, m.ProjectID.ValueString(), m.EstimateFieldName.ValueString())
		if err != nil {
			return payload, fmt.Errorf("could not find estimate field %q in project: %w", m.EstimateFieldName.ValueString(), err)
		}
		payload.Estimate = &youtrack.ProjectCustomFieldTimeRef{ID: f.ID}
	}

	if !m.TimeSpentFieldName.IsNull() && !m.TimeSpentFieldName.IsUnknown() && m.TimeSpentFieldName.ValueString() != "" {
		f, err := r.client.GetProjectCustomFieldByName(ctx, m.ProjectID.ValueString(), m.TimeSpentFieldName.ValueString())
		if err != nil {
			return payload, fmt.Errorf("could not find time spent field %q in project: %w", m.TimeSpentFieldName.ValueString(), err)
		}
		payload.TimeSpent = &youtrack.ProjectCustomFieldTimeRef{ID: f.ID}
	}

	return payload, nil
}

func (m *projectTimeTrackingSettingsResourceModel) fromAPIModel(s *youtrack.ProjectTimeTrackingSettings) {
	m.ID = types.StringValue(s.ID)
	m.Enabled = types.BoolValue(s.Enabled)

	if s.Estimate != nil && s.Estimate.Field != nil {
		m.EstimateFieldName = types.StringValue(s.Estimate.Field.Name)
	} else {
		m.EstimateFieldName = types.StringNull()
	}

	if s.TimeSpent != nil && s.TimeSpent.Field != nil {
		m.TimeSpentFieldName = types.StringValue(s.TimeSpent.Field.Name)
	} else {
		m.TimeSpentFieldName = types.StringNull()
	}
}
