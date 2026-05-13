package settings

import (
	"context"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &globalTimeTrackingSettingsResource{}
	_ resource.ResourceWithConfigure   = &globalTimeTrackingSettingsResource{}
	_ resource.ResourceWithImportState = &globalTimeTrackingSettingsResource{}
)

// NewGlobalTimeTrackingSettingsResource is a helper function to simplify the provider implementation.
func NewGlobalTimeTrackingSettingsResource() resource.Resource {
	return &globalTimeTrackingSettingsResource{}
}

// globalTimeTrackingSettingsResource is the resource implementation.
type globalTimeTrackingSettingsResource struct {
	client *youtrack.Client
}

type globalTimeTrackingSettingsResourceModel struct {
	ID                  types.String                `tfsdk:"id"`
	WorkTimeSettings    globalWorkTimeSettingsModel `tfsdk:"work_time_settings"`
	WorkItemTypes       types.Set                   `tfsdk:"work_item_types"`
	AttributePrototypes types.List                  `tfsdk:"attribute_prototypes"`
	LastUpdated         types.String                `tfsdk:"last_updated"`
}

type globalWorkTimeSettingsModel struct {
	ID             types.String `tfsdk:"id"`
	MinutesADay    types.Int64  `tfsdk:"minutes_a_day"`
	WorkDays       types.List   `tfsdk:"work_days"`
	FirstDayOfWeek types.Int64  `tfsdk:"first_day_of_week"`
	DaysAWeek      types.Int64  `tfsdk:"days_a_week"`
}

type globalWorkItemTypeModel struct {
	Name         types.String `tfsdk:"name"`
	AutoAttached types.Bool   `tfsdk:"auto_attached"`
}

type globalWorkItemAttributePrototypeResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Values    types.List   `tfsdk:"values"`
	Instances types.List   `tfsdk:"instances"`
}

type globalWorkItemProjectAttributeModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Ordinal types.Int64  `tfsdk:"ordinal"`
	Values  types.List   `tfsdk:"values"`
}

type globalWorkItemAttributeValueResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	AutoAttach  types.Bool   `tfsdk:"auto_attach"`
}

// Metadata returns the resource type name.
func (r *globalTimeTrackingSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_time_tracking_settings"
}

// Schema defines the schema for the resource.
func (r *globalTimeTrackingSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack global time tracking settings resource. This resource manages the global work time schedule and exposes related global work item type and attribute data.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the global time tracking settings configuration.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"work_time_settings": schema.SingleNestedAttribute{
				Description: "System-wide work schedule settings.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "Work time settings identifier.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"minutes_a_day": schema.Int64Attribute{
						Description: "Number of minutes in a working day.",
						Required:    true,
					},
					"work_days": schema.ListAttribute{
						Description: "Indexes of working days in week. Sunday is 0, Monday is 1, and so on.",
						ElementType: types.Int64Type,
						Required:    true,
					},
					"first_day_of_week": schema.Int64Attribute{
						Description: "Index of the first day of week reported by the server.",
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"days_a_week": schema.Int64Attribute{
						Description: "Number of working days a week reported by the server.",
						Computed:    true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"work_item_types": schema.SetNestedAttribute{
				Description: "Set of work item types to manage. When set, this set is the source of truth: " +
					"types present here are created or updated in YouTrack, types absent here are deleted. " +
					"When omitted, work item types are read from the API and stored as computed state.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Set{
					computedWhenUnconfiguredSetModifier{},
					setplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Name of the work item type.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"auto_attached": schema.BoolAttribute{
							Description: "When true, this type is automatically added to a project when time tracking is enabled.",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"attribute_prototypes": schema.ListNestedAttribute{
				Description: "Read-only list of global work item attribute prototypes and their values.",
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
						"values": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id":          schema.StringAttribute{Computed: true},
									"name":        schema.StringAttribute{Computed: true},
									"description": schema.StringAttribute{Computed: true},
									"auto_attach": schema.BoolAttribute{Computed: true},
								},
							},
						},
						"instances": schema.ListNestedAttribute{
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id":      schema.StringAttribute{Computed: true},
									"name":    schema.StringAttribute{Computed: true},
									"ordinal": schema.Int64Attribute{Computed: true},
									"values": schema.ListNestedAttribute{
										Computed: true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"id":          schema.StringAttribute{Computed: true},
												"name":        schema.StringAttribute{Computed: true},
												"description": schema.StringAttribute{Computed: true},
												"auto_attach": schema.BoolAttribute{Computed: true},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last update",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *globalTimeTrackingSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan globalTimeTrackingSettingsResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	if !r.applyWorkTimeSettingsAndUpdateModel(ctx, &plan, &resp.Diagnostics) {
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *globalTimeTrackingSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state globalTimeTrackingSettingsResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	globalSettings, ok := r.getGlobalTimeTrackingSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	if !updateGlobalTimeTrackingSettingsModelWithTimestamp(ctx, globalSettings, &state) {
		resp.Diagnostics.AddError(errConvertingTimeTracking, errConvertingNestedTimeTracking)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *globalTimeTrackingSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan globalTimeTrackingSettingsResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	if !r.applyWorkTimeSettingsAndUpdateModel(ctx, &plan, &resp.Diagnostics) {
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete resets the work time settings to sane defaults and removes Terraform state on success.
func (r *globalTimeTrackingSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state globalTimeTrackingSettingsResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	defaultSettings := youtrack.WorkTimeSettings{
		MinutesADay: defaultWorkMinutesADay,
		WorkDays:    defaultWorkDays,
	}

	if !r.updateWorkTimeSettingsAndHandleError(ctx, defaultSettings, &resp.Diagnostics) {
		return
	}

	globalSettings, ok := r.getGlobalTimeTrackingSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	if !updateGlobalTimeTrackingSettingsModelWithTimestamp(ctx, globalSettings, &state) {
		resp.Diagnostics.AddError(errConvertingTimeTracking, errConvertingNestedTimeTracking)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// ImportState imports the resource state.
func (r *globalTimeTrackingSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	globalSettings, ok := r.getGlobalTimeTrackingSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	var state globalTimeTrackingSettingsResourceModel
	if !updateGlobalTimeTrackingSettingsModelWithTimestamp(ctx, globalSettings, &state) {
		resp.Diagnostics.AddError(errConvertingTimeTracking, errConvertingNestedTimeTracking)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Configure adds the provider configured client to the resource.
func (r *globalTimeTrackingSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}
