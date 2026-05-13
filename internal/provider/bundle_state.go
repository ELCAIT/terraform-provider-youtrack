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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &stateBundleResource{}
	_ resource.ResourceWithConfigure   = &stateBundleResource{}
	_ resource.ResourceWithImportState = &stateBundleResource{}
)

const (
	errCreatingStateBundle  = "Error creating state bundle"
	errReadingStateBundle   = "Error reading state bundle"
	errUpdatingStateBundle  = "Error updating state bundle"
	errDeletingStateBundle  = "Error deleting state bundle"
	errMissingStateBundleID = "Missing state bundle ID"
	errStateBundleIDReq     = "State bundle ID is required"
)

func NewStateBundleResource() resource.Resource {
	return &stateBundleResource{}
}

type stateBundleResource struct {
	client *youtrack.Client
}

type stateBundleValueModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	LocalizedName types.String `tfsdk:"localized_name"`
	Description   types.String `tfsdk:"description"`
	IsResolved    types.Bool   `tfsdk:"is_resolved"`
	Archived      types.Bool   `tfsdk:"archived"`
	Ordinal       types.Int64  `tfsdk:"ordinal"`
}

type stateBundleResourceModel struct {
	ID           types.String            `tfsdk:"id"`
	Name         types.String            `tfsdk:"name"`
	IsUpdateable types.Bool              `tfsdk:"is_updateable"`
	Values       []stateBundleValueModel `tfsdk:"values"`
}

func (r *stateBundleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_state_bundle"
}

func (r *stateBundleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	valueAttributes := bundleCommonValueAttributes("state")
	valueAttributes["is_resolved"] = schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(false),
		Description: "Whether issues in this state are considered resolved.",
	}

	resp.Schema = schema.Schema{
		Description: "YouTrack state bundle resource. This resource manages sets of state values.",
		Attributes:  bundleCommonAttributes("state", valueAttributes),
	}
}

func (r *stateBundleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *stateBundleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stateBundleResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	created, err := r.client.CreateStateBundle(ctx, plan.toAPIModel())
	if err != nil {
		resp.Diagnostics.AddError(errCreatingStateBundle, fmt.Sprintf("Could not create state bundle: %v", err))
		return
	}

	plan.fromAPIModel(created)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *stateBundleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateBundleResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingStateBundleID, errStateBundleIDReq) {
		return
	}

	apiModel, err := r.client.GetStateBundleByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			reboundByName, reboundErr := r.client.GetStateBundleByName(ctx, state.Name.ValueString())
			if reboundErr != nil {
				if youtrack.IsStateBundleNotFoundError(reboundErr) {
					resp.State.RemoveResource(ctx)
					return
				}

				resp.Diagnostics.AddError(errReadingStateBundle, fmt.Sprintf("Could not recover state bundle by name: %v", reboundErr))
				return
			}

			state.fromAPIModel(reboundByName)
			helpers.SetStateAndCheckError(ctx, resp, &state)
			return
		}
		resp.Diagnostics.AddError(errReadingStateBundle, fmt.Sprintf("Could not read state bundle: %v", err))
		return
	}

	state.fromAPIModel(apiModel)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *stateBundleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stateBundleResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	if !helpers.ValidateResourceID(plan.ID, &resp.Diagnostics, errMissingStateBundleID, errStateBundleIDReq) {
		return
	}

	updated, err := r.client.UpdateStateBundle(ctx, plan.ID.ValueString(), plan.toAPIModel())
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingStateBundle, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "state bundle", err))
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *stateBundleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateBundleResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	err := r.client.DeleteStateBundle(ctx, state.ID.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(errDeletingStateBundle, fmt.Sprintf("Could not delete state bundle: %v", err))
	}
}

func (r *stateBundleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *stateBundleResourceModel) toAPIModel() youtrack.StateBundle {
	values := make([]youtrack.StateBundleElement, 0, len(m.Values))
	for _, value := range m.Values {
		item := youtrack.StateBundleElement{
			Name:       value.Name.ValueString(),
			IsResolved: helpers.BoolFromOptional(value.IsResolved),
			Archived:   helpers.BoolFromOptional(value.Archived),
		}
		item.ID = helpers.StringFromOptional(value.ID)
		item.Description = helpers.StringFromOptional(value.Description)
		item.LocalizedName = helpers.StringFromOptional(value.LocalizedName)
		values = append(values, item)
	}

	return youtrack.StateBundle{
		Name:   m.Name.ValueString(),
		Values: values,
	}
}

func (m *stateBundleResourceModel) fromAPIModel(apiModel *youtrack.StateBundle) {
	m.ID = types.StringValue(apiModel.ID)
	m.Name = types.StringValue(apiModel.Name)
	m.IsUpdateable = types.BoolValue(apiModel.IsUpdateable)

	values := make([]stateBundleValueModel, 0, len(apiModel.Values))
	for _, value := range apiModel.Values {
		values = append(values, stateBundleValueModel{
			ID:            types.StringValue(value.ID),
			Name:          types.StringValue(value.Name),
			LocalizedName: helpers.StringOrNull(value.LocalizedName),
			Description:   helpers.StringOrNull(value.Description),
			IsResolved:    types.BoolValue(value.IsResolved),
			Archived:      types.BoolValue(value.Archived),
			Ordinal:       types.Int64Value(int64(value.Ordinal)),
		})
	}
	m.Values = values
}
