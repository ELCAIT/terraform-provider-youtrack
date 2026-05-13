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
	_ resource.Resource                = &issueLinkTypeResource{}
	_ resource.ResourceWithConfigure   = &issueLinkTypeResource{}
	_ resource.ResourceWithImportState = &issueLinkTypeResource{}
)

const (
	errCreatingIssueLinkType  = "Error creating issue link type"
	errReadingIssueLinkType   = "Error reading issue link type"
	errUpdatingIssueLinkType  = "Error updating issue link type"
	errDeletingIssueLinkType  = "Error deleting issue link type"
	errMissingIssueLinkTypeID = "Missing issue link type ID"

	errIssueLinkTypeIDRequired = "Issue link type ID is required"
)

func NewIssueLinkTypeResource() resource.Resource {
	return &issueLinkTypeResource{}
}

type issueLinkTypeResource struct {
	client *youtrack.Client
}

type issueLinkTypeResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	SourceToTarget          types.String `tfsdk:"source_to_target"`
	TargetToSource          types.String `tfsdk:"target_to_source"`
	Directed                types.Bool   `tfsdk:"directed"`
	Aggregation             types.Bool   `tfsdk:"aggregation"`
	ReadOnly                types.Bool   `tfsdk:"read_only"`
	LocalizedName           types.String `tfsdk:"localized_name"`
	LocalizedSourceToTarget types.String `tfsdk:"localized_source_to_target"`
	LocalizedTargetToSource types.String `tfsdk:"localized_target_to_source"`
}

func (r *issueLinkTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_issue_link_type"
}

func (r *issueLinkTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack issue link type resource. This resource manages issue link types.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the issue link type.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the issue link type.",
			},
			"source_to_target": schema.StringAttribute{
				Required:    true,
				Description: "The outward name of the issue link type.",
			},
			"target_to_source": schema.StringAttribute{
				Required:    true,
				Description: "The inward name of the issue link type.",
			},
			"directed": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the issue link type is directed.",
			},
			"aggregation": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the issue link type represents an aggregation relation.",
			},
			"read_only": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the issue link type is read-only in YouTrack.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"localized_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Localized name of the issue link type, if available.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"localized_source_to_target": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Localized outward name of the issue link type, if available.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"localized_target_to_source": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Localized inward name of the issue link type, if available.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *issueLinkTypeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *issueLinkTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan issueLinkTypeResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	apiModel := plan.toAPIModel()
	created, err := r.client.CreateIssueLinkType(ctx, apiModel)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingIssueLinkType,
			fmt.Sprintf("Could not create issue link type: %v", err),
		)
		return
	}

	plan.fromAPIModel(created)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *issueLinkTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state issueLinkTypeResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingIssueLinkTypeID, errIssueLinkTypeIDRequired) {
		return
	}

	apiModel, err := r.client.GetIssueLinkTypeByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			errReadingIssueLinkType,
			fmt.Sprintf("Could not read issue link type: %v", err),
		)
		return
	}

	state.fromAPIModel(apiModel)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *issueLinkTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan issueLinkTypeResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	if !helpers.ValidateResourceID(plan.ID, &resp.Diagnostics, errMissingIssueLinkTypeID, errIssueLinkTypeIDRequired) {
		return
	}

	apiModel := plan.toAPIModel()
	updated, err := r.client.UpdateIssueLinkType(ctx, plan.ID.ValueString(), apiModel)
	if err != nil {
		resp.Diagnostics.AddError(
			errUpdatingIssueLinkType,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "issue link type", err),
		)
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *issueLinkTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state issueLinkTypeResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	err := r.client.DeleteIssueLinkType(ctx, state.ID.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			errDeletingIssueLinkType,
			fmt.Sprintf("Could not delete issue link type: %v", err),
		)
		return
	}
}

func (r *issueLinkTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *issueLinkTypeResourceModel) toAPIModel() youtrack.IssueLinkType {
	apiModel := youtrack.IssueLinkType{
		Name:           m.Name.ValueString(),
		SourceToTarget: m.SourceToTarget.ValueString(),
		TargetToSource: m.TargetToSource.ValueString(),
		Directed:       helpers.BoolFromOptional(m.Directed),
		Aggregation:    helpers.BoolFromOptional(m.Aggregation),
	}

	if !m.LocalizedName.IsNull() && !m.LocalizedName.IsUnknown() {
		apiModel.LocalizedName = m.LocalizedName.ValueString()
	}
	if !m.LocalizedSourceToTarget.IsNull() && !m.LocalizedSourceToTarget.IsUnknown() {
		apiModel.LocalizedSourceToTarget = m.LocalizedSourceToTarget.ValueString()
	}
	if !m.LocalizedTargetToSource.IsNull() && !m.LocalizedTargetToSource.IsUnknown() {
		apiModel.LocalizedTargetToSource = m.LocalizedTargetToSource.ValueString()
	}

	return apiModel
}

func (m *issueLinkTypeResourceModel) fromAPIModel(apiModel *youtrack.IssueLinkType) {
	m.ID = types.StringValue(apiModel.ID)
	m.Name = types.StringValue(apiModel.Name)
	m.SourceToTarget = types.StringValue(apiModel.SourceToTarget)
	m.TargetToSource = types.StringValue(apiModel.TargetToSource)
	m.Directed = types.BoolValue(apiModel.Directed)
	m.Aggregation = types.BoolValue(apiModel.Aggregation)
	m.ReadOnly = types.BoolValue(apiModel.ReadOnly)
	m.LocalizedName = helpers.StringOrNull(apiModel.LocalizedName)
	m.LocalizedSourceToTarget = helpers.StringOrNull(apiModel.LocalizedSourceToTarget)
	m.LocalizedTargetToSource = helpers.StringOrNull(apiModel.LocalizedTargetToSource)
}
