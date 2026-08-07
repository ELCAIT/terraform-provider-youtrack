// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	azureModuleID       = "azure-module-123"
	azureModuleName     = "Test Azure Module"
	azureClientID       = "my-client-id"
	azureClientSecret   = "my-client-secret" //nolint:gosec // G101: test constant, not a hardcoded credential
	azureTenant         = "11111111-1111-1111-1111-111111111111"
	azureServerURL      = "https://login.microsoftonline.com/11111111-1111-1111-1111-111111111111/oauth2/v2.0/authorize"
	azureRedirectURI    = "https://hub.example.com/hub/api/rest/oauth2/auth"
	azureExtensionGrant = "custom-grant"
)

func newMinimalAzureModel() azureAuthModuleResourceModel {
	return azureAuthModuleResourceModel{
		ID:                     types.StringValue(azureModuleID),
		Name:                   types.StringValue(azureModuleName),
		Disabled:               types.BoolValue(false),
		ClientID:               types.StringValue(azureClientID),
		ClientSecret:           types.StringValue(azureClientSecret),
		Tenant:                 types.StringNull(),
		RedirectURI:            types.StringNull(),
		AllowedCreateNewUsers:  types.BoolValue(false),
		BackgroundSyncEnabled:  types.BoolValue(false),
		IconURL:                types.StringNull(),
		ExtensionGrantType:     types.StringNull(),
		ConnectionTimeout:      types.Int64Null(),
		ReadTimeout:            types.Int64Null(),
		SyncInterval:           types.StringNull(),
		RequestGroupPermission: types.BoolValue(false),
		RequestIDToken:         types.BoolValue(false),
	}
}

func TestAzureAuthModuleModelToAPIModel(t *testing.T) {
	tests := []struct {
		name  string
		model azureAuthModuleResourceModel
		want  youtrack.AzureAuthModule
	}{
		{
			name: "full model converts correctly",
			model: func() azureAuthModuleResourceModel {
				m := newMinimalAzureModel()
				m.Tenant = types.StringValue(azureTenant)
				m.RedirectURI = types.StringValue(azureRedirectURI)
				m.AllowedCreateNewUsers = types.BoolValue(true)
				m.ExtensionGrantType = types.StringValue(azureExtensionGrant)
				m.ConnectionTimeout = types.Int64Value(5000)
				m.ReadTimeout = types.Int64Value(10000)
				m.RequestGroupPermission = types.BoolValue(true)
				m.RequestIDToken = types.BoolValue(true)
				return m
			}(),
			want: youtrack.AzureAuthModule{
				Name:                   azureModuleName,
				Disabled:               false,
				ClientID:               azureClientID,
				ClientSecret:           azureClientSecret,
				Tenant:                 azureTenant,
				RedirectURI:            azureRedirectURI,
				AllowedCreateNewUsers:  true,
				BackgroundSyncEnabled:  false,
				ExtensionGrantType:     azureExtensionGrant,
				ConnectionTimeout:      5000,
				ReadTimeout:            10000,
				RequestGroupPermission: true,
				RequestIDToken:         true,
			},
		},
		{
			name:  "minimal model with required fields only",
			model: newMinimalAzureModel(),
			want: youtrack.AzureAuthModule{
				Name:         azureModuleName,
				ClientID:     azureClientID,
				ClientSecret: azureClientSecret,
			},
		},
		{
			name: "disabled module",
			model: func() azureAuthModuleResourceModel {
				m := newMinimalAzureModel()
				m.Disabled = types.BoolValue(true)
				return m
			}(),
			want: youtrack.AzureAuthModule{
				Name:         azureModuleName,
				Disabled:     true,
				ClientID:     azureClientID,
				ClientSecret: azureClientSecret,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.toAPIModel()
			helpers.AssertFieldEqual(t, "Name", got.Name, tt.want.Name)
			helpers.AssertFieldEqual(t, "Disabled", got.Disabled, tt.want.Disabled)
			helpers.AssertFieldEqual(t, "ClientID", got.ClientID, tt.want.ClientID)
			helpers.AssertFieldEqual(t, "ClientSecret", got.ClientSecret, tt.want.ClientSecret)
			helpers.AssertFieldEqual(t, "Tenant", got.Tenant, tt.want.Tenant)
			helpers.AssertFieldEqual(t, "RedirectURI", got.RedirectURI, tt.want.RedirectURI)
			helpers.AssertFieldEqual(t, "AllowedCreateNewUsers", got.AllowedCreateNewUsers, tt.want.AllowedCreateNewUsers)
			helpers.AssertFieldEqual(t, "BackgroundSyncEnabled", got.BackgroundSyncEnabled, tt.want.BackgroundSyncEnabled)
			helpers.AssertFieldEqual(t, "ExtensionGrantType", got.ExtensionGrantType, tt.want.ExtensionGrantType)
			helpers.AssertFieldEqual(t, "ConnectionTimeout", got.ConnectionTimeout, tt.want.ConnectionTimeout)
			helpers.AssertFieldEqual(t, "ReadTimeout", got.ReadTimeout, tt.want.ReadTimeout)
			helpers.AssertFieldEqual(t, "RequestGroupPermission", got.RequestGroupPermission, tt.want.RequestGroupPermission)
			helpers.AssertFieldEqual(t, "RequestIDToken", got.RequestIDToken, tt.want.RequestIDToken)
		})
	}
}

func TestAzureAuthModuleFromAPIModel(t *testing.T) {
	tests := []struct {
		name           string
		apiModule      youtrack.AzureAuthModule
		wantID         string
		wantName       string
		wantDisabled   bool
		wantClientID   string
		wantRedirect   bool // true if redirect_uri should be set
		wantTenantNull bool
	}{
		{
			name: "full api response populates model",
			apiModule: youtrack.AzureAuthModule{
				ID:                    azureModuleID,
				Name:                  azureModuleName,
				Disabled:              false,
				ClientID:              azureClientID,
				Tenant:                azureTenant,
				ServerURL:             azureServerURL,
				RedirectURI:           azureRedirectURI,
				AllowedCreateNewUsers: true,
			},
			wantID:         azureModuleID,
			wantName:       azureModuleName,
			wantDisabled:   false,
			wantClientID:   azureClientID,
			wantRedirect:   true,
			wantTenantNull: false,
		},
		{
			name: "api response with empty tenant sets null (multi-tenant)",
			apiModule: youtrack.AzureAuthModule{
				ID:       azureModuleID,
				Name:     azureModuleName,
				ClientID: azureClientID,
			},
			wantID:         azureModuleID,
			wantName:       azureModuleName,
			wantDisabled:   false,
			wantClientID:   azureClientID,
			wantRedirect:   false,
			wantTenantNull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var model azureAuthModuleResourceModel
			model.fromAPIModel(&tt.apiModule)

			helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), tt.wantID)
			helpers.AssertFieldEqual(t, "Name", model.Name.ValueString(), tt.wantName)
			helpers.AssertFieldEqual(t, "Disabled", model.Disabled.ValueBool(), tt.wantDisabled)
			helpers.AssertFieldEqual(t, "ClientID", model.ClientID.ValueString(), tt.wantClientID)
			helpers.AssertFieldEqual(t, "Tenant.IsNull", model.Tenant.IsNull(), tt.wantTenantNull)

			if tt.wantRedirect {
				helpers.AssertFieldEqual(t, "RedirectURI", model.RedirectURI.ValueString(), azureRedirectURI)
			} else {
				helpers.AssertFieldEqual(t, "RedirectURI.IsNull", model.RedirectURI.IsNull(), true)
			}
		})
	}
}
