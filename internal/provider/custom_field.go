package provider

import (
	"context"
	"fmt"
	"strings"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &customFieldResource{}
	_ resource.ResourceWithConfigure   = &customFieldResource{}
	_ resource.ResourceWithImportState = &customFieldResource{}
)

const (
	errCreatingCustomField  = "Error creating custom field"
	errReadingCustomField   = "Error reading custom field"
	errUpdatingCustomField  = "Error updating custom field"
	errDeletingCustomField  = "Error deleting custom field"
	errCustomFieldType      = "Custom field type is immutable"
	errMissingCustomFieldID = "Missing custom field ID"
	errCustomFieldIDReq     = "Custom field ID is required"
	errBundleFallbackCreate = "failed fallback create without bundle defaults"
	errBundleFallbackUpdate = "failed fallback update with bundle defaults"
	errBundleFallbackDelete = "failed cleanup delete after fallback update error"

	fieldTypePrefixEnum       = "enum"
	fieldTypePrefixState      = "state"
	fieldTypePrefixOwnedField = "ownedField"
	fieldTypePrefixBuild      = "build"
	fieldTypePrefixVersion    = "version"

	bundleTypeEnum  = "EnumBundle"
	bundleTypeState = "StateBundle"
	bundleTypeOwned = "OwnedBundle"
	bundleTypeBuild = "BuildBundle"
	bundleTypeVer   = "VersionBundle"

	defaultsTypeEnum  = "EnumBundleCustomFieldDefaults"
	defaultsTypeState = "StateBundleCustomFieldDefaults"
	defaultsTypeOwned = "OwnedBundleCustomFieldDefaults"
	defaultsTypeBuild = "BuildBundleCustomFieldDefaults"
	defaultsTypeVer   = "VersionBundleCustomFieldDefaults"
)

func NewCustomFieldResource() resource.Resource {
	return &customFieldResource{}
}

type customFieldResource struct {
	client *youtrack.Client
}

type customFieldDefaultsModel struct {
	CanBeEmpty     types.Bool   `tfsdk:"can_be_empty"`
	EmptyFieldText types.String `tfsdk:"empty_field_text"`
	IsPublic       types.Bool   `tfsdk:"is_public"`
	BundleID       types.String `tfsdk:"bundle_id"`
	BundleName     types.String `tfsdk:"bundle_name"`
}

type customFieldResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	LocalizedName          types.String `tfsdk:"localized_name"`
	Aliases                types.String `tfsdk:"aliases"`
	FieldTypeID            types.String `tfsdk:"field_type_id"`
	FieldTypePresentation  types.String `tfsdk:"field_type_presentation"`
	IsAutoAttached         types.Bool   `tfsdk:"is_auto_attached"`
	IsDisplayedInIssueList types.Bool   `tfsdk:"is_displayed_in_issue_list"`
	Ordinal                types.Int64  `tfsdk:"ordinal"`
	IsUpdateable           types.Bool   `tfsdk:"is_updateable"`
	HasRunningJob          types.Bool   `tfsdk:"has_running_job"`
	FieldDefaults          types.Object `tfsdk:"field_defaults"`
}

func (r *customFieldResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_field"
}

func (r *customFieldResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack custom field resource. This resource manages system custom fields.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the custom field.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the custom field.",
			},
			"localized_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Localized name used in UI.",
			},
			"aliases": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Comma-separated aliases used in search and commands.",
			},
			"field_type_id": schema.StringAttribute{
				Required:    true,
				Description: "YouTrack field type identifier (for example: enum[1], state[1], string). This value is immutable after creation.",
			},
			"field_type_presentation": schema.StringAttribute{
				Computed:    true,
				Description: "Field type presentation returned by YouTrack.",
			},
			"is_auto_attached": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the field is attached automatically to new projects.",
			},
			"is_displayed_in_issue_list": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the field is displayed in the issue list by default.",
			},
			"ordinal": schema.Int64Attribute{
				Computed:    true,
				Description: "The numeric order of the field.",
			},
			"is_updateable": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether current user can update this custom field.",
			},
			"has_running_job": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether a background job is running for this field.",
			},
			"field_defaults": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Default project-related settings for the custom field.",
				Attributes: map[string]schema.Attribute{
					"can_be_empty": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether issues may keep this field empty.",
					},
					"empty_field_text": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Placeholder text shown when the field is empty.",
					},
					"is_public": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether the field is public.",
					},
					"bundle_id": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Referenced default bundle ID for bundle-based field types.",
					},
					"bundle_name": schema.StringAttribute{
						Computed:    true,
						Description: "Referenced default bundle name.",
					},
				},
			},
		},
	}
}

func (r *customFieldResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *customFieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customFieldResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	apiModel := plan.toAPIModel()
	created, err := r.client.CreateCustomField(ctx, apiModel)
	if err != nil && customFieldRequestHasBundle(apiModel) {
		created, err = r.createWithBundleFallback(ctx, apiModel)
	}

	if err != nil {
		resp.Diagnostics.AddError(errCreatingCustomField, fmt.Sprintf("Could not create custom field: %v", err))
		return
	}

	plan.fromAPIModel(created)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *customFieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customFieldResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() || strings.TrimSpace(state.ID.ValueString()) == "" {
		resp.Diagnostics.AddError(errMissingCustomFieldID, errCustomFieldIDReq)
		return
	}

	apiModel, err := r.client.GetCustomFieldByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(errReadingCustomField, fmt.Sprintf("Could not read custom field: %v", err))
		return
	}

	state.fromAPIModel(apiModel)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *customFieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customFieldResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	var state customFieldResourceModel
	stateDiags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.IsNull() || plan.ID.IsUnknown() || strings.TrimSpace(plan.ID.ValueString()) == "" {
		resp.Diagnostics.AddError(errMissingCustomFieldID, errCustomFieldIDReq)
		return
	}

	if customFieldTypeChangeRequested(state.FieldTypeID, plan.FieldTypeID) {
		resp.Diagnostics.AddError(
			errCustomFieldType,
			fmt.Sprintf(
				"Changing field_type_id from %q to %q is forbidden to avoid destructive side effects in YouTrack. Create a new custom field instead.",
				state.FieldTypeID.ValueString(),
				plan.FieldTypeID.ValueString(),
			),
		)
		return
	}

	updated, err := r.client.UpdateCustomField(ctx, plan.ID.ValueString(), plan.toAPIModel())
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingCustomField, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "custom field", err))
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *customFieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customFieldResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() || strings.TrimSpace(state.ID.ValueString()) == "" {
		return
	}

	err := r.client.DeleteCustomField(ctx, state.ID.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(errDeletingCustomField, fmt.Sprintf("Could not delete custom field: %v", err))
	}
}

func (r *customFieldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func customFieldTypeChangeRequested(currentType, plannedType types.String) bool {
	if currentType.IsNull() || currentType.IsUnknown() || plannedType.IsNull() || plannedType.IsUnknown() {
		return false
	}

	return currentType.ValueString() != plannedType.ValueString()
}
