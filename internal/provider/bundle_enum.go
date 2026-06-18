package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &enumBundleResource{}
	_ resource.ResourceWithConfigure   = &enumBundleResource{}
	_ resource.ResourceWithImportState = &enumBundleResource{}
)

const (
	errCreatingEnumBundle  = "Error creating enum bundle"
	errReadingEnumBundle   = "Error reading enum bundle"
	errUpdatingEnumBundle  = "Error updating enum bundle"
	errDeletingEnumBundle  = "Error deleting enum bundle"
	errMissingEnumBundleID = "Missing enum bundle ID"
	errEnumBundleIDReq     = "Enum bundle ID is required"
	errRemovingEnumValues  = "Removing enum values is not supported"
)

func NewEnumBundleResource() resource.Resource {
	return &enumBundleResource{}
}

type enumBundleResource struct {
	client *youtrack.Client
}

type enumBundleValueModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	LocalizedName types.String `tfsdk:"localized_name"`
	Description   types.String `tfsdk:"description"`
	Archived      types.Bool   `tfsdk:"archived"`
	Ordinal       types.Int64  `tfsdk:"ordinal"`
}

type enumBundleResourceModel struct {
	ID           types.String           `tfsdk:"id"`
	Name         types.String           `tfsdk:"name"`
	IsUpdateable types.Bool             `tfsdk:"is_updateable"`
	Values       []enumBundleValueModel `tfsdk:"values"`
}

func (r *enumBundleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_enum_bundle"
}

func (r *enumBundleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	valueAttributes := bundleCommonValueAttributes("enum")

	resp.Schema = schema.Schema{
		Description: "YouTrack enum bundle resource. This resource manages sets of enum values.",
		Attributes:  bundleCommonAttributes("enum", valueAttributes),
	}
}

func (r *enumBundleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *enumBundleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan enumBundleResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	created, err := r.client.CreateEnumBundle(ctx, plan.toAPIModel())
	if err != nil {
		resp.Diagnostics.AddError(errCreatingEnumBundle, fmt.Sprintf("Could not create enum bundle: %v", err))
		return
	}

	plan.fromAPIModel(created)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *enumBundleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state enumBundleResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingEnumBundleID, errEnumBundleIDReq) {
		return
	}

	apiModel, err := r.client.GetEnumBundleByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			reboundByName, reboundErr := r.client.GetEnumBundleByName(ctx, state.Name.ValueString())
			if reboundErr != nil {
				if youtrack.IsEnumBundleNotFoundError(reboundErr) {
					resp.State.RemoveResource(ctx)
					return
				}

				resp.Diagnostics.AddError(errReadingEnumBundle, fmt.Sprintf("Could not recover enum bundle by name: %v", reboundErr))
				return
			}

			state.fromAPIModel(reboundByName)
			helpers.SetStateAndCheckError(ctx, resp, &state)
			return
		}
		resp.Diagnostics.AddError(errReadingEnumBundle, fmt.Sprintf("Could not read enum bundle: %v", err))
		return
	}

	state.fromAPIModel(apiModel)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *enumBundleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan enumBundleResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	var state enumBundleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !helpers.ValidateResourceID(plan.ID, &resp.Diagnostics, errMissingEnumBundleID, errEnumBundleIDReq) {
		return
	}

	removedValues := removedEnumBundleValues(state.Values, plan.Values)
	if len(removedValues) > 0 {
		resp.Diagnostics.AddError(
			errRemovingEnumValues,
			fmt.Sprintf(
				"YouTrack keeps existing enum values when they are omitted from bundle updates. Removed values: %s. Mark values with archived = true instead of deleting them from configuration.",
				strings.Join(removedValues, ", "),
			),
		)
		return
	}

	updated, err := r.client.UpdateEnumBundle(ctx, plan.ID.ValueString(), plan.toAPIModel())
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingEnumBundle, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "enum bundle", err))
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *enumBundleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state enumBundleResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	err := r.client.DeleteEnumBundle(ctx, state.ID.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(errDeletingEnumBundle, fmt.Sprintf("Could not delete enum bundle: %v", err))
	}
}

func (r *enumBundleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *enumBundleResourceModel) toAPIModel() youtrack.EnumBundle {
	values := make([]youtrack.EnumBundleElement, 0, len(m.Values))
	for _, value := range m.Values {
		item := youtrack.EnumBundleElement{
			Name:     value.Name.ValueString(),
			Archived: helpers.BoolFromOptional(value.Archived),
		}
		item.ID = helpers.StringFromOptional(value.ID)
		item.Description = helpers.StringFromOptional(value.Description)
		item.LocalizedName = helpers.StringFromOptional(value.LocalizedName)
		values = append(values, item)
	}

	return youtrack.EnumBundle{
		Name:   m.Name.ValueString(),
		Values: values,
	}
}

func (m *enumBundleResourceModel) fromAPIModel(apiModel *youtrack.EnumBundle) {
	m.ID = types.StringValue(apiModel.ID)
	m.Name = types.StringValue(apiModel.Name)
	m.IsUpdateable = types.BoolValue(apiModel.IsUpdateable)

	values := make([]enumBundleValueModel, 0, len(apiModel.Values))
	for _, value := range apiModel.Values {
		values = append(values, enumBundleValueModel{
			ID:            types.StringValue(value.ID),
			Name:          types.StringValue(value.Name),
			LocalizedName: helpers.StringOrNull(value.LocalizedName),
			Description:   helpers.StringOrNull(value.Description),
			Archived:      types.BoolValue(value.Archived),
			Ordinal:       types.Int64Value(int64(value.Ordinal)),
		})
	}
	m.Values = values
}

func removedEnumBundleValues(stateValues, planValues []enumBundleValueModel) []string {
	plannedByID := buildIDMap(planValues)
	plannedByName := buildNameMap(planValues)

	removed := make([]string, 0)
	for _, value := range stateValues {
		if !isValuePlanned(value, plannedByID, plannedByName) {
			removed = append(removed, getValueLabel(value))
		}
	}

	sort.Strings(removed)
	return removed
}

func buildIDMap(values []enumBundleValueModel) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !value.ID.IsNull() && !value.ID.IsUnknown() {
			result[value.ID.ValueString()] = struct{}{}
		}
	}
	return result
}

func buildNameMap(values []enumBundleValueModel) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !value.Name.IsNull() && !value.Name.IsUnknown() {
			result[value.Name.ValueString()] = struct{}{}
		}
	}
	return result
}

func isValuePlanned(value enumBundleValueModel, plannedByID, plannedByName map[string]struct{}) bool {
	if !value.ID.IsNull() && !value.ID.IsUnknown() {
		if _, exists := plannedByID[value.ID.ValueString()]; exists {
			return true
		}
	}

	if !value.Name.IsNull() && !value.Name.IsUnknown() {
		_, exists := plannedByName[value.Name.ValueString()]
		return exists
	}

	return false
}

func getValueLabel(value enumBundleValueModel) string {
	if !value.Name.IsNull() && !value.Name.IsUnknown() {
		return value.Name.ValueString()
	}
	return "<unknown>"
}
