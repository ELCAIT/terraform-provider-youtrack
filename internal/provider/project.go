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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

const (
	errCreatingProject  = "Error creating project"
	errReadingProject   = "Error reading project"
	errUpdatingProject  = "Error updating project"
	errDeletingProject  = "Error deleting project"
	errMissingProjectID = "Missing project ID"

	errProjectIDRequired = "Project ID is required"
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	client *youtrack.Client
}

type projectResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ShortName    types.String `tfsdk:"short_name"`
	Description  types.String `tfsdk:"description"`
	LeaderLogin  types.String `tfsdk:"leader_login"`
	Archived     types.Bool   `tfsdk:"archived"`
	Template     types.Bool   `tfsdk:"template"`
	FromEmail    types.String `tfsdk:"from_email"`
	ReplyToEmail types.String `tfsdk:"reply_to_email"`
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack project resource. Manages a YouTrack project including its settings.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The entity ID of the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the project.",
			},
			"short_name": schema.StringAttribute{
				Required:    true,
				Description: "The short name (prefix) of the project. Used as the prefix for issue IDs. Cannot be changed after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A description of the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"leader_login": schema.StringAttribute{
				Required:    true,
				Description: "The login (username) of the user to set as the project owner.",
			},
			"archived": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the project is archived.",
			},
			"template": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether this project is a template.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"from_email": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The email address used to send notifications for the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reply_to_email": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The reply-to email address for project notifications.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	leader, err := r.client.GetUserByLogin(ctx, plan.LeaderLogin.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingProject,
			fmt.Sprintf("Could not find user with login %q: %v", plan.LeaderLogin.ValueString(), err),
		)
		return
	}

	payload := plan.toCreatePayload(leader.Id)
	created, err := r.client.CreateProject(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingProject,
			fmt.Sprintf("Could not create project: %v", err),
		)
		return
	}

	plan.fromAPIModel(created)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingProjectID, errProjectIDRequired) {
		return
	}

	project, err := r.client.GetProject(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			errReadingProject,
			fmt.Sprintf("Could not read project: %v", err),
		)
		return
	}

	state.fromAPIModel(project)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	if !helpers.ValidateResourceID(plan.ID, &resp.Diagnostics, errMissingProjectID, errProjectIDRequired) {
		return
	}

	leader, err := r.client.GetUserByLogin(ctx, plan.LeaderLogin.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			errUpdatingProject,
			fmt.Sprintf("Could not find user with login %q: %v", plan.LeaderLogin.ValueString(), err),
		)
		return
	}

	payload := plan.toUpdatePayload(leader.Id)
	updated, err := r.client.UpdateProject(ctx, plan.ID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			errUpdatingProject,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "project", err),
		)
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	err := r.client.DeleteProject(ctx, state.ID.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			errDeletingProject,
			fmt.Sprintf("Could not delete project: %v", err),
		)
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *projectResourceModel) toCreatePayload(leaderID string) youtrack.ProjectCreatePayload {
	payload := youtrack.ProjectCreatePayload{
		Name:      m.Name.ValueString(),
		ShortName: m.ShortName.ValueString(),
		Leader:    &youtrack.UserRef{ID: leaderID},
	}

	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		payload.Description = m.Description.ValueString()
	}

	if !m.Template.IsNull() && !m.Template.IsUnknown() {
		v := m.Template.ValueBool()
		payload.Template = &v
	}

	return payload
}

func (m *projectResourceModel) toUpdatePayload(leaderID string) youtrack.ProjectUpdatePayload {
	payload := youtrack.ProjectUpdatePayload{
		Name:   m.Name.ValueString(),
		Leader: &youtrack.UserRef{ID: leaderID},
	}

	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		payload.Description = m.Description.ValueString()
	}

	if !m.Archived.IsNull() && !m.Archived.IsUnknown() {
		v := m.Archived.ValueBool()
		payload.Archived = &v
	}

	if !m.FromEmail.IsNull() && !m.FromEmail.IsUnknown() {
		payload.FromEmail = m.FromEmail.ValueString()
	}

	if !m.ReplyToEmail.IsNull() && !m.ReplyToEmail.IsUnknown() {
		payload.ReplyToEmail = m.ReplyToEmail.ValueString()
	}

	return payload
}

func (m *projectResourceModel) fromAPIModel(p *youtrack.Project) {
	m.ID = types.StringValue(p.ID)
	m.Name = types.StringValue(p.Name)
	m.ShortName = types.StringValue(p.ShortName)
	m.Description = helpers.StringOrNull(p.Description)
	m.Archived = types.BoolValue(p.Archived)
	m.Template = types.BoolValue(p.Template)
	m.FromEmail = helpers.StringOrNull(p.FromEmail)
	m.ReplyToEmail = helpers.StringOrNull(p.ReplyToEmail)

	if p.Leader != nil {
		m.LeaderLogin = types.StringValue(p.Leader.Login)
	} else {
		m.LeaderLogin = types.StringNull()
	}
}
