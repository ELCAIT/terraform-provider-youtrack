// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// secretBackedResourceOps holds everything that differs between the resources whose
// API never returns their secret back (Hub services, OAuth2/Azure auth modules), so
// their otherwise identical CRUD bodies can live here once.
//
// TModel is the Terraform resource model, TAPI the youtrack-api-client model.
type secretBackedResourceOps[TModel any, TAPI any] struct {
	// label names the resource in diagnostics detail messages, as it reads mid
	// sentence ("service", "Azure auth module").
	label string
	// updateLabel names the resource in helpers.ErrCouldNotUpdateFmt, which
	// already lowercases the surrounding wording.
	updateLabel string

	errCreating   string
	errReading    string
	errUpdating   string
	errDeleting   string
	errImporting  string
	errMissingID  string
	errIDRequired string

	// toAPI/fromAPI wrap the resource's own toAPIModel/fromAPIModel.
	toAPI   func(ctx context.Context, model *TModel) TAPI
	fromAPI func(model *TModel, api *TAPI)

	modelID     func(model *TModel) types.String
	modelSecret func(model *TModel) types.String
	setSecret   func(model *TModel, secret types.String)
	apiID       func(api *TAPI) string
	apiSecret   func(api *TAPI) string

	// secretGeneratedByAPI reports whether the secret to persist after Create comes
	// from the create response (Hub generated it) instead of the request payload.
	secretGeneratedByAPI bool

	// enforcedAttribute/enforcedValue describe a bool attribute that Hub forces to
	// its own value at create time regardless of what was requested. When set,
	// Create re-applies the planned value with a follow-up update, otherwise
	// Terraform reports "provider produced inconsistent result after apply".
	// enforcedValue must return a pointer into the passed API model.
	enforcedAttribute string
	enforcedValue     func(api *TAPI) *bool

	create func(ctx context.Context, api TAPI) (*TAPI, error)
	read   func(ctx context.Context, id string) (*TAPI, error)
	update func(ctx context.Context, id string, api TAPI) (*TAPI, error)
	delete func(ctx context.Context, id string) error
}

// Create creates the resource and sets the initial Terraform state.
func (o secretBackedResourceOps[TModel, TAPI]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	sent := o.toAPI(ctx, &plan)
	created, err := o.create(ctx, sent)
	if err != nil {
		resp.Diagnostics.AddError(o.errCreating, fmt.Sprintf("Could not create %s: %v", o.label, err))
		return
	}

	created, ok := o.enforceCreateOverride(ctx, sent, created, &resp.Diagnostics)
	if !ok {
		return
	}

	// Preserve the secret since the API does not return it on subsequent reads.
	secretSource := &sent
	if o.secretGeneratedByAPI {
		secretSource = created
	}

	o.fromAPI(&plan, created)
	o.setSecret(&plan, types.StringValue(o.apiSecret(secretSource)))

	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

// Read refreshes the Terraform state with the latest data.
func (o secretBackedResourceOps[TModel, TAPI]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	id := o.modelID(&state)
	if !helpers.ValidateResourceID(id, &resp.Diagnostics, o.errMissingID, o.errIDRequired) {
		return
	}

	api, err := o.read(ctx, id.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(o.errReading, fmt.Sprintf("Could not read %s: %v", o.label, err))
		return
	}

	// Preserve the secret from existing state; the API never returns it.
	existingSecret := o.modelSecret(&state)
	o.fromAPI(&state, api)
	o.setSecret(&state, existingSecret)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (o secretBackedResourceOps[TModel, TAPI]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	sent := o.toAPI(ctx, &plan)

	updated, err := o.update(ctx, o.modelID(&plan).ValueString(), sent)
	if err != nil {
		resp.Diagnostics.AddError(o.errUpdating, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, o.updateLabel, err))
		return
	}

	// Preserve the secret from what was sent since the API does not return it.
	o.fromAPI(&plan, updated)
	o.setSecret(&plan, types.StringValue(o.apiSecret(&sent)))

	helpers.SetStateAndCheckError(ctx, resp, &plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (o secretBackedResourceOps[TModel, TAPI]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	id := o.modelID(&state)
	if !helpers.HasResourceID(id) {
		return
	}

	err := o.delete(ctx, id.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(o.errDeleting, fmt.Sprintf("Could not delete %s: %v", o.label, err))
	}
}

// ImportState imports the resource state by ID.
func (o secretBackedResourceOps[TModel, TAPI]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)

	api, err := o.read(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			o.errImporting,
			fmt.Sprintf("Could not read %s with ID '%s': %v", o.label, id, err),
		)
		return
	}

	var state TModel
	o.fromAPI(&state, api)
	// The secret cannot be imported; set to empty string as a placeholder.
	o.setSecret(&state, types.StringValue(""))

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// enforceCreateOverride re-applies an attribute Hub overrode during create, if the
// resource declares one. It returns the API model to build state from and whether
// Create may continue.
func (o secretBackedResourceOps[TModel, TAPI]) enforceCreateOverride(
	ctx context.Context,
	sent TAPI,
	created *TAPI,
	diagnostics *diag.Diagnostics,
) (*TAPI, bool) {
	if o.enforcedValue == nil {
		return created, true
	}

	planned := *o.enforcedValue(&sent)
	actual := o.enforcedValue(created)
	if *actual == planned {
		return created, true
	}

	*actual = planned

	updated, err := o.update(ctx, o.apiID(created), *created)
	if err != nil {
		diagnostics.AddError(
			o.errCreating,
			fmt.Sprintf("Could not enforce %s after create: %v", o.enforcedAttribute, err),
		)
		return nil, false
	}

	return updated, true
}
