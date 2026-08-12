// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// Error message titles
	errCreatingService  = "Error creating service"
	errReadingService   = "Error reading service"
	errUpdatingService  = "Error updating service"
	errDeletingService  = "Error deleting service"
	errImportingService = "Error importing service"
	errMissingServiceID = "Missing service ID"

	// Error message details
	errServiceIDRequired = "Service ID is required to read the service"
)

// serviceResourceModel maps the Terraform resource schema data.
type serviceResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	Name                         types.String `tfsdk:"name"`
	Key                          types.String `tfsdk:"key"`
	HomeURL                      types.String `tfsdk:"home_url"`
	ApplicationName              types.String `tfsdk:"application_name"`
	Description                  types.String `tfsdk:"description"`
	Vendor                       types.String `tfsdk:"vendor"`
	Version                      types.String `tfsdk:"version"`
	RedirectURIs                 types.List   `tfsdk:"redirect_uris"`
	BaseURLs                     types.List   `tfsdk:"base_urls"`
	Trusted                      types.Bool   `tfsdk:"trusted"`
	ConsentRequired              types.Bool   `tfsdk:"consent_required"`
	ClientCredentialsFlowEnabled types.Bool   `tfsdk:"client_credentials_flow_enabled"`
	AuthCodeFlowEnabled          types.Bool   `tfsdk:"auth_code_flow_enabled"`
	PKCERequired                 types.Bool   `tfsdk:"pkce_required"`
	ImplicitFlowEnabled          types.Bool   `tfsdk:"implicit_flow_enabled"`
	ResourceOwnerFlowEnabled     types.Bool   `tfsdk:"resource_owner_flow_enabled"`
	Secret                       types.String `tfsdk:"secret"`
}

// toAPIModel converts the Terraform model to the API model.
//
// RedirectURIs/BaseURLs are passed through as a real Go nil when the
// Terraform list is null/unconfigured (marshals to JSON null, which Hub
// treats as "leave unchanged") rather than an empty-but-non-nil slice
// (which would marshal to JSON [] and clear the list) - see Service's doc
// comment in youtrack-api-client for the confirmed clearing semantics.
func (m *serviceResourceModel) toAPIModel(ctx context.Context) youtrack.Service {
	service := youtrack.Service{
		Name:                         m.Name.ValueString(),
		Key:                          m.Key.ValueString(),
		HomeURL:                      m.HomeURL.ValueString(),
		ApplicationName:              m.ApplicationName.ValueString(),
		Description:                  m.Description.ValueString(),
		Vendor:                       m.Vendor.ValueString(),
		Version:                      m.Version.ValueString(),
		Trusted:                      m.Trusted.ValueBool(),
		ConsentRequired:              m.ConsentRequired.ValueBool(),
		ClientCredentialsFlowEnabled: m.ClientCredentialsFlowEnabled.ValueBool(),
		AuthCodeFlowEnabled:          m.AuthCodeFlowEnabled.ValueBool(),
		PKCERequired:                 m.PKCERequired.ValueBool(),
		ImplicitFlowEnabled:          m.ImplicitFlowEnabled.ValueBool(),
		ResourceOwnerFlowEnabled:     m.ResourceOwnerFlowEnabled.ValueBool(),
		Secret:                       m.Secret.ValueString(),
	}

	if !m.RedirectURIs.IsNull() && !m.RedirectURIs.IsUnknown() {
		if uris, ok := helpers.ListToStringSlice(ctx, m.RedirectURIs); ok {
			service.RedirectURIs = uris
		}
	}

	if !m.BaseURLs.IsNull() && !m.BaseURLs.IsUnknown() {
		if urls, ok := helpers.ListToStringSlice(ctx, m.BaseURLs); ok {
			service.BaseURLs = urls
		}
	}

	return service
}

// fromAPIModel populates the Terraform model from the API model. secret is
// intentionally left untouched by this function since the API never returns
// it; callers must preserve/set it separately.
func (m *serviceResourceModel) fromAPIModel(api *youtrack.Service) {
	m.ID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.Key = types.StringValue(api.Key)
	m.HomeURL = helpers.StringOrNull(api.HomeURL)
	m.ApplicationName = helpers.StringOrNull(api.ApplicationName)
	m.Description = helpers.StringOrNull(api.Description)
	m.Vendor = helpers.StringOrNull(api.Vendor)
	m.Version = helpers.StringOrNull(api.Version)
	m.Trusted = types.BoolValue(api.Trusted)
	m.ConsentRequired = types.BoolValue(api.ConsentRequired)
	m.ClientCredentialsFlowEnabled = types.BoolValue(api.ClientCredentialsFlowEnabled)
	m.AuthCodeFlowEnabled = types.BoolValue(api.AuthCodeFlowEnabled)
	m.PKCERequired = types.BoolValue(api.PKCERequired)
	m.ImplicitFlowEnabled = types.BoolValue(api.ImplicitFlowEnabled)
	m.ResourceOwnerFlowEnabled = types.BoolValue(api.ResourceOwnerFlowEnabled)

	m.RedirectURIs = stringSliceToList(api.RedirectURIs)
	m.BaseURLs = stringSliceToList(api.BaseURLs)
}

// stringSliceToList converts a Go string slice into a Terraform list,
// collapsing an empty/nil slice to null so it round-trips cleanly against an
// unconfigured Optional list attribute.
func stringSliceToList(values []string) types.List {
	if len(values) == 0 {
		return types.ListNull(types.StringType)
	}

	list, diags := types.ListValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}

	return list
}
