package provider

import (
	"context"
	"fmt"
	"time"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &nestedGroupResource{}
	_ resource.ResourceWithConfigure   = &nestedGroupResource{}
	_ resource.ResourceWithImportState = &nestedGroupResource{}
)

const (
	errCreatingNestedGroup = "Error creating nested group"
	errReadingNestedGroup  = "Error reading nested group"
	errUpdatingNestedGroup = "Error updating nested group"
	errDeletingNestedGroup = "Error deleting nested group"
	errMissingNestedID     = "Missing nested group ID"
	errNestedIDRequired    = "Nested group ID is required"
	errResolvingUser       = "Error resolving user"
	errResolvingGroup      = "Error resolving subgroup"
	errSelfSubgroup        = "Invalid subgroup reference"

	subGroupResolveMaxAttempts = 6
	subGroupResolveRetryDelay  = 250 * time.Millisecond
	errNotFoundFragment        = "not found"
	errHTTPNotFound            = "404"
)

// NewNestedGroupResource creates the nested-group resource.
func NewNestedGroupResource() resource.Resource {
	return &nestedGroupResource{}
}

type nestedGroupResource struct {
	client *youtrack.Client
}

type nestedGroupResourceModel struct {
	ID                             types.String `tfsdk:"id"`
	Name                           types.String `tfsdk:"name"`
	Description                    types.String `tfsdk:"description"`
	OwnUserLogins                  types.Set    `tfsdk:"own_user_logins"`
	SubGroupNames                  types.Set    `tfsdk:"sub_group_names"`
	RequireTwoFactorAuthentication types.Bool   `tfsdk:"require_two_factor_authentication"`
	Viewers                        types.Set    `tfsdk:"viewers"`
	Updaters                       types.Set    `tfsdk:"updaters"`
	AutoJoin                       types.Bool   `tfsdk:"auto_join"`
	AutoJoinDomain                 types.String `tfsdk:"auto_join_domain"`
}

func (r *nestedGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nested_group"
}

func (r *nestedGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack nested group resource. Manages a group and its direct membership via user logins and subgroup names.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "YouTrack ID of the group (computed)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Group name.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Group description.",
			},
			"own_user_logins": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Direct users in the group, identified by login.",
			},
			"sub_group_names": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Direct subgroups of this group, identified by group name.",
			},
			"require_two_factor_authentication": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether users in this group must use two-factor authentication.",
			},
			"viewers": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Description: "Users or groups that can view this group. Values are resolved by user login first, then group name.",
			},
			"updaters": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Description: "Users or groups that can update this group. Values are resolved by user login first, then group name.",
			},
			"auto_join": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether matching users are auto-joined to this group.",
			},
			"auto_join_domain": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Domain used for automatic joining. Typically used with auto_join.",
			},
		},
	}
}

func (r *nestedGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *nestedGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nestedGroupResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	created, err := r.client.CreateGroup(ctx, newNestedGroupPayload(plan.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(errCreatingNestedGroup, fmt.Sprintf("Could not create nested group: %v", err))
		return
	}

	resolved, ok := r.resolveMembership(ctx, &plan, created.ID, &resp.Diagnostics)
	if !ok {
		return
	}

	updated, err := r.client.UpdateGroup(ctx, created.ID, *resolved)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingNestedGroup, fmt.Sprintf("Could not set nested group membership: %v", err))
		return
	}

	r.mapAPIToModel(ctx, updated, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *nestedGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state nestedGroupResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingNestedID, errNestedIDRequired) {
		return
	}

	apiGroup, err := r.client.GetGroupByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(errReadingNestedGroup, fmt.Sprintf("Could not read nested group: %v", err))
		return
	}

	r.mapAPIToModel(ctx, apiGroup, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *nestedGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan nestedGroupResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	if !helpers.ValidateResourceID(plan.ID, &resp.Diagnostics, errMissingNestedID, errNestedIDRequired) {
		return
	}

	resolved, ok := r.resolveMembership(ctx, &plan, plan.ID.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	updated, err := r.client.UpdateGroup(ctx, plan.ID.ValueString(), *resolved)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingNestedGroup, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "nested group", err))
		return
	}

	r.mapAPIToModel(ctx, updated, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *nestedGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nestedGroupResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	allUsers, err := r.client.GetAllUsersGroup(ctx)
	if err != nil {
		resp.Diagnostics.AddError(errDeletingNestedGroup, fmt.Sprintf("Could not resolve successor group: %v", err))
		return
	}

	err = r.client.DeleteGroup(ctx, state.ID.ValueString(), allUsers.ID)
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(errDeletingNestedGroup, fmt.Sprintf("Could not delete nested group: %v", err))
	}
}

func (r *nestedGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
