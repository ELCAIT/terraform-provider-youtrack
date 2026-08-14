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
	errCreatingAzureModule  = "Error creating Azure auth module"
	errReadingAzureModule   = "Error reading Azure auth module"
	errUpdatingAzureModule  = "Error updating Azure auth module"
	errDeletingAzureModule  = "Error deleting Azure auth module"
	errImportingAzureModule = "Error importing Azure auth module"
	errMissingAzureModuleID = "Missing Azure auth module ID"

	// Error message details
	errAzureModuleIDRequired = "Azure auth module ID is required to read the module"
)

// ops describes the Azure auth module to the shared secret-backed resource CRUD.
func (r *azureAuthModuleResource) ops() secretBackedResourceOps[azureAuthModuleResourceModel, youtrack.AzureAuthModule] {
	return secretBackedResourceOps[azureAuthModuleResourceModel, youtrack.AzureAuthModule]{
		label:         "Azure auth module",
		updateLabel:   "azure auth module",
		errCreating:   errCreatingAzureModule,
		errReading:    errReadingAzureModule,
		errUpdating:   errUpdatingAzureModule,
		errDeleting:   errDeletingAzureModule,
		errImporting:  errImportingAzureModule,
		errMissingID:  errMissingAzureModuleID,
		errIDRequired: errAzureModuleIDRequired,

		toAPI: func(_ context.Context, model *azureAuthModuleResourceModel) youtrack.AzureAuthModule {
			return model.toAPIModel()
		},
		fromAPI: func(model *azureAuthModuleResourceModel, api *youtrack.AzureAuthModule) {
			model.fromAPIModel(api)
		},
		modelID:     func(model *azureAuthModuleResourceModel) types.String { return model.ID },
		modelSecret: func(model *azureAuthModuleResourceModel) types.String { return model.ClientSecret },
		setSecret: func(model *azureAuthModuleResourceModel, secret types.String) {
			model.ClientSecret = secret
		},
		apiID:     func(api *youtrack.AzureAuthModule) string { return api.ID },
		apiSecret: func(api *youtrack.AzureAuthModule) string { return api.ClientSecret },

		enforcedAttribute: attrAllowedCreateNewUsers,
		enforcedValue:     func(api *youtrack.AzureAuthModule) *bool { return &api.AllowedCreateNewUsers },

		create: r.client.CreateAzureAuthModule,
		read:   r.client.GetAzureAuthModuleByID,
		update: r.client.UpdateAzureAuthModule,
		delete: r.client.DeleteAzureAuthModule,
	}
}

// azureAuthModuleResourceModel maps the Terraform resource schema data.
type azureAuthModuleResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Disabled               types.Bool   `tfsdk:"disabled"`
	ClientID               types.String `tfsdk:"client_id"`
	ClientSecret           types.String `tfsdk:"client_secret"`
	Tenant                 types.String `tfsdk:"tenant"`
	ServerURL              types.String `tfsdk:"server_url"`
	RedirectURI            types.String `tfsdk:"redirect_uri"`
	AllowedCreateNewUsers  types.Bool   `tfsdk:"allowed_create_new_users"`
	BackgroundSyncEnabled  types.Bool   `tfsdk:"background_sync_enabled"`
	IconURL                types.String `tfsdk:"icon_url"`
	ExtensionGrantType     types.String `tfsdk:"extension_grant_type"`
	ConnectionTimeout      types.Int64  `tfsdk:"connection_timeout"`
	ReadTimeout            types.Int64  `tfsdk:"read_timeout"`
	SyncInterval           types.String `tfsdk:"sync_interval"`
	IsDefault              types.Bool   `tfsdk:"is_default"`
	RequestGroupPermission types.Bool   `tfsdk:"request_group_permission"`
	RequestIDToken         types.Bool   `tfsdk:"request_id_token"`
}

// toAPIModel converts the Terraform model to the API model.
func (m *azureAuthModuleResourceModel) toAPIModel() youtrack.AzureAuthModule {
	return youtrack.AzureAuthModule{
		Name:                   m.Name.ValueString(),
		Disabled:               m.Disabled.ValueBool(),
		ClientID:               m.ClientID.ValueString(),
		ClientSecret:           m.ClientSecret.ValueString(),
		Tenant:                 m.Tenant.ValueString(),
		RedirectURI:            m.RedirectURI.ValueString(),
		AllowedCreateNewUsers:  m.AllowedCreateNewUsers.ValueBool(),
		BackgroundSyncEnabled:  m.BackgroundSyncEnabled.ValueBool(),
		IconURL:                m.IconURL.ValueString(),
		ExtensionGrantType:     m.ExtensionGrantType.ValueString(),
		ConnectionTimeout:      int(m.ConnectionTimeout.ValueInt64()),
		ReadTimeout:            int(m.ReadTimeout.ValueInt64()),
		SyncInterval:           m.SyncInterval.ValueString(),
		IsDefault:              m.IsDefault.ValueBool(),
		RequestGroupPermission: m.RequestGroupPermission.ValueBool(),
		RequestIDToken:         m.RequestIDToken.ValueBool(),
	}
}

// fromAPIModel populates the Terraform model from the API model, preserving the client secret
// from the prior state since the API does not return it.
func (m *azureAuthModuleResourceModel) fromAPIModel(api *youtrack.AzureAuthModule) {
	m.ID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.Disabled = types.BoolValue(api.Disabled)
	m.ClientID = types.StringValue(api.ClientID)
	m.AllowedCreateNewUsers = types.BoolValue(api.AllowedCreateNewUsers)
	m.BackgroundSyncEnabled = types.BoolValue(api.BackgroundSyncEnabled)

	// client_secret is not returned by the API; preserve existing state value.

	m.Tenant = helpers.StringOrNull(api.Tenant)
	m.ServerURL = types.StringValue(api.ServerURL)
	m.RedirectURI = helpers.StringOrNull(api.RedirectURI)
	m.IconURL = helpers.StringOrNull(api.IconURL)
	m.ExtensionGrantType = helpers.StringOrNull(api.ExtensionGrantType)

	m.ConnectionTimeout = helpers.Int64OrNull(api.ConnectionTimeout)
	m.ReadTimeout = helpers.Int64OrNull(api.ReadTimeout)

	m.SyncInterval = helpers.StringOrNull(api.SyncInterval)
	m.IsDefault = types.BoolValue(api.IsDefault)
	m.RequestGroupPermission = types.BoolValue(api.RequestGroupPermission)
	m.RequestIDToken = types.BoolValue(api.RequestIDToken)
}
