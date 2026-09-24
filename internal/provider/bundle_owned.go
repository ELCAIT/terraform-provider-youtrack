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
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ownedBundleResource{}
	_ resource.ResourceWithConfigure   = &ownedBundleResource{}
	_ resource.ResourceWithImportState = &ownedBundleResource{}
)

const (
	errCreatingOwnedBundle  = "Error creating owned bundle"
	errReadingOwnedBundle   = "Error reading owned bundle"
	errUpdatingOwnedBundle  = "Error updating owned bundle"
	errDeletingOwnedBundle  = "Error deleting owned bundle"
	errMissingOwnedBundleID = "Missing owned bundle ID"
	errOwnedBundleIDReq     = "Owned bundle ID is required"

	ownedBundleValueKind = "owned"
)

func NewOwnedBundleResource() resource.Resource {
	return &ownedBundleResource{}
}

type ownedBundleResource struct {
	client *youtrack.Client
}

type ownedBundleValueModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Archived    types.Bool   `tfsdk:"archived"`
	Ordinal     types.Int64  `tfsdk:"ordinal"`
	OwnerLogin  types.String `tfsdk:"owner_login"`
}

type ownedBundleResourceModel struct {
	ID           types.String            `tfsdk:"id"`
	Name         types.String            `tfsdk:"name"`
	IsUpdateable types.Bool              `tfsdk:"is_updateable"`
	Values       []ownedBundleValueModel `tfsdk:"values"`
}

func (r *ownedBundleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_owned_bundle"
}

func (r *ownedBundleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	valueAttributes := bundleCommonValueAttributes(ownedBundleValueKind)
	// Owned values are plain bundle elements in YouTrack: they carry no localized name.
	delete(valueAttributes, "localized_name")
	valueAttributes["owner_login"] = schema.StringAttribute{
		Optional:    true,
		Description: "Login of the user who owns this value. Leave unset for a value without an owner.",
	}

	resp.Schema = schema.Schema{
		Description: "YouTrack owned field bundle resource. This resource manages the sets of values, each optionally owned by a user, that back ownedField[1] custom fields. " +
			"Values are matched to existing ones by name, so renaming a value replaces it with a new one.",
		Attributes: bundleCommonAttributes(ownedBundleValueKind, valueAttributes),
	}
}

func (r *ownedBundleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

func (r *ownedBundleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ownedBundleResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	owners, err := r.resolveOwners(ctx, plan.Values)
	if err != nil {
		resp.Diagnostics.AddError(errCreatingOwnedBundle, fmt.Sprintf("Could not create owned bundle: %v", err))
		return
	}

	created, err := r.client.CreateOwnedBundle(ctx, plan.toAPIModel(owners))
	if err != nil {
		resp.Diagnostics.AddError(errCreatingOwnedBundle, fmt.Sprintf("Could not create owned bundle: %v", err))
		return
	}

	plan.fromAPIModel(created, plan.Values)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *ownedBundleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ownedBundleResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingOwnedBundleID, errOwnedBundleIDReq) {
		return
	}

	apiModel, err := r.client.GetOwnedBundleByID(ctx, state.ID.ValueString())
	if err != nil && youtrack.IsNotFoundError(err) {
		apiModel, err = r.client.GetOwnedBundleByName(ctx, state.Name.ValueString())
		if youtrack.IsOwnedBundleNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
	}
	if err != nil {
		resp.Diagnostics.AddError(errReadingOwnedBundle, fmt.Sprintf("Could not read owned bundle: %v", err))
		return
	}

	state.fromAPIModel(apiModel, state.Values)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *ownedBundleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ownedBundleResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	if !helpers.ValidateResourceID(plan.ID, &resp.Diagnostics, errMissingOwnedBundleID, errOwnedBundleIDReq) {
		return
	}

	updated, err := r.reconcile(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingOwnedBundle, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "owned bundle", err))
		return
	}

	if unexpected := unexpectedOwnedValueNames(plan, updated); len(unexpected) > 0 {
		resp.Diagnostics.AddError(errUpdatingOwnedBundle, fmt.Sprintf(errKeptBundleValuesFmt, strings.Join(unexpected, ", ")))
		return
	}

	plan.fromAPIModel(updated, plan.Values)
	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

func (r *ownedBundleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ownedBundleResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	err := r.client.DeleteOwnedBundle(ctx, state.ID.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(errDeletingOwnedBundle, fmt.Sprintf("Could not delete owned bundle: %v", err))
	}
}

func (r *ownedBundleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
