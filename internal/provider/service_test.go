// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	serviceTestID              = "service-123"
	serviceTestName            = "Test Service"
	serviceTestKey             = "test-service-key"
	serviceTestHomeURL         = "https://service.example.com"
	serviceTestApplicationName = "Test Application"
	serviceTestDescription     = "A test service"
	serviceTestVendor          = "ELCA"
	serviceTestVersion         = "1.0.0"
	serviceTestRedirectURI     = "https://service.example.com/callback"
	serviceTestBaseURL         = "https://service.example.com"
	serviceTestSecret          = "my-service-secret" //nolint:gosec // G101: test constant, not a hardcoded credential
)

func newMinimalServiceModel() serviceResourceModel {
	return serviceResourceModel{
		ID:                           types.StringValue(serviceTestID),
		Name:                         types.StringValue(serviceTestName),
		Key:                          types.StringValue(serviceTestKey),
		HomeURL:                      types.StringNull(),
		ApplicationName:              types.StringNull(),
		Description:                  types.StringNull(),
		Vendor:                       types.StringNull(),
		Version:                      types.StringNull(),
		RedirectURIs:                 types.ListNull(types.StringType),
		BaseURLs:                     types.ListNull(types.StringType),
		Trusted:                      types.BoolValue(false),
		ConsentRequired:              types.BoolValue(true),
		ClientCredentialsFlowEnabled: types.BoolValue(true),
		AuthCodeFlowEnabled:          types.BoolValue(true),
		PKCERequired:                 types.BoolValue(false),
		ImplicitFlowEnabled:          types.BoolValue(true),
		ResourceOwnerFlowEnabled:     types.BoolValue(true),
		Secret:                       types.StringNull(),
	}
}

func TestServiceModelToAPIModel(t *testing.T) {
	tests := []struct {
		name  string
		model serviceResourceModel
		want  youtrack.Service
	}{
		{
			name:  "minimal model with required fields only sends nil lists",
			model: newMinimalServiceModel(),
			want: youtrack.Service{
				Name:                         serviceTestName,
				Key:                          serviceTestKey,
				Trusted:                      false,
				ConsentRequired:              true,
				ClientCredentialsFlowEnabled: true,
				AuthCodeFlowEnabled:          true,
				PKCERequired:                 false,
				ImplicitFlowEnabled:          true,
				ResourceOwnerFlowEnabled:     true,
			},
		},
		{
			name: "full model converts correctly",
			model: func() serviceResourceModel {
				m := newMinimalServiceModel()
				m.HomeURL = types.StringValue(serviceTestHomeURL)
				m.ApplicationName = types.StringValue(serviceTestApplicationName)
				m.Description = types.StringValue(serviceTestDescription)
				m.Vendor = types.StringValue(serviceTestVendor)
				m.Version = types.StringValue(serviceTestVersion)
				m.Trusted = types.BoolValue(true)
				m.ConsentRequired = types.BoolValue(false)
				m.ClientCredentialsFlowEnabled = types.BoolValue(false)
				m.AuthCodeFlowEnabled = types.BoolValue(false)
				m.PKCERequired = types.BoolValue(true)
				m.ImplicitFlowEnabled = types.BoolValue(false)
				m.ResourceOwnerFlowEnabled = types.BoolValue(false)
				m.Secret = types.StringValue(serviceTestSecret)
				m.RedirectURIs = mustStringList(t, []string{serviceTestRedirectURI}...)
				m.BaseURLs = mustStringList(t, []string{serviceTestBaseURL}...)
				return m
			}(),
			want: youtrack.Service{
				Name:                         serviceTestName,
				Key:                          serviceTestKey,
				HomeURL:                      serviceTestHomeURL,
				ApplicationName:              serviceTestApplicationName,
				Description:                  serviceTestDescription,
				Vendor:                       serviceTestVendor,
				Version:                      serviceTestVersion,
				Trusted:                      true,
				ConsentRequired:              false,
				ClientCredentialsFlowEnabled: false,
				AuthCodeFlowEnabled:          false,
				PKCERequired:                 true,
				ImplicitFlowEnabled:          false,
				ResourceOwnerFlowEnabled:     false,
				Secret:                       serviceTestSecret,
				RedirectURIs:                 []string{serviceTestRedirectURI},
				BaseURLs:                     []string{serviceTestBaseURL},
			},
		},
		{
			name: "configured-but-empty lists are sent as empty, not nil, to allow clearing",
			model: func() serviceResourceModel {
				m := newMinimalServiceModel()
				m.RedirectURIs = mustStringList(t, []string{}...)
				m.BaseURLs = mustStringList(t, []string{}...)
				return m
			}(),
			want: youtrack.Service{
				Name:                         serviceTestName,
				Key:                          serviceTestKey,
				Trusted:                      false,
				ConsentRequired:              true,
				ClientCredentialsFlowEnabled: true,
				AuthCodeFlowEnabled:          true,
				PKCERequired:                 false,
				ImplicitFlowEnabled:          true,
				ResourceOwnerFlowEnabled:     true,
				RedirectURIs:                 []string{},
				BaseURLs:                     []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.model.toAPIModel(context.Background())
			helpers.AssertFieldEqual(t, "Name", got.Name, tt.want.Name)
			helpers.AssertFieldEqual(t, "Key", got.Key, tt.want.Key)
			helpers.AssertFieldEqual(t, "HomeURL", got.HomeURL, tt.want.HomeURL)
			helpers.AssertFieldEqual(t, "ApplicationName", got.ApplicationName, tt.want.ApplicationName)
			helpers.AssertFieldEqual(t, "Description", got.Description, tt.want.Description)
			helpers.AssertFieldEqual(t, "Vendor", got.Vendor, tt.want.Vendor)
			helpers.AssertFieldEqual(t, "Version", got.Version, tt.want.Version)
			helpers.AssertFieldEqual(t, "Trusted", got.Trusted, tt.want.Trusted)
			helpers.AssertFieldEqual(t, "ConsentRequired", got.ConsentRequired, tt.want.ConsentRequired)
			helpers.AssertFieldEqual(t, "ClientCredentialsFlowEnabled", got.ClientCredentialsFlowEnabled, tt.want.ClientCredentialsFlowEnabled)
			helpers.AssertFieldEqual(t, "AuthCodeFlowEnabled", got.AuthCodeFlowEnabled, tt.want.AuthCodeFlowEnabled)
			helpers.AssertFieldEqual(t, "PKCERequired", got.PKCERequired, tt.want.PKCERequired)
			helpers.AssertFieldEqual(t, "ImplicitFlowEnabled", got.ImplicitFlowEnabled, tt.want.ImplicitFlowEnabled)
			helpers.AssertFieldEqual(t, "ResourceOwnerFlowEnabled", got.ResourceOwnerFlowEnabled, tt.want.ResourceOwnerFlowEnabled)
			helpers.AssertFieldEqual(t, "Secret", got.Secret, tt.want.Secret)

			if (got.RedirectURIs == nil) != (tt.want.RedirectURIs == nil) {
				t.Fatalf("RedirectURIs nil-ness mismatch: got %v, want %v", got.RedirectURIs, tt.want.RedirectURIs)
			}
			if len(got.RedirectURIs) != len(tt.want.RedirectURIs) {
				t.Fatalf("RedirectURIs length mismatch: got %v, want %v", got.RedirectURIs, tt.want.RedirectURIs)
			}

			if (got.BaseURLs == nil) != (tt.want.BaseURLs == nil) {
				t.Fatalf("BaseURLs nil-ness mismatch: got %v, want %v", got.BaseURLs, tt.want.BaseURLs)
			}
			if len(got.BaseURLs) != len(tt.want.BaseURLs) {
				t.Fatalf("BaseURLs length mismatch: got %v, want %v", got.BaseURLs, tt.want.BaseURLs)
			}
		})
	}
}

func TestServiceModelFromAPIModel(t *testing.T) {
	tests := []struct {
		name           string
		apiService     youtrack.Service
		wantHomeURL    bool
		wantRedirect   bool
		wantRedirectAt string
	}{
		{
			name: "full api response populates model",
			apiService: youtrack.Service{
				ID:                           serviceTestID,
				Name:                         serviceTestName,
				Key:                          serviceTestKey,
				HomeURL:                      serviceTestHomeURL,
				ApplicationName:              serviceTestApplicationName,
				Description:                  serviceTestDescription,
				Vendor:                       serviceTestVendor,
				Version:                      serviceTestVersion,
				Trusted:                      true,
				ConsentRequired:              false,
				ClientCredentialsFlowEnabled: true,
				AuthCodeFlowEnabled:          true,
				PKCERequired:                 true,
				ImplicitFlowEnabled:          false,
				ResourceOwnerFlowEnabled:     false,
				RedirectURIs:                 []string{serviceTestRedirectURI},
				BaseURLs:                     []string{serviceTestBaseURL},
			},
			wantHomeURL:    true,
			wantRedirect:   true,
			wantRedirectAt: serviceTestRedirectURI,
		},
		{
			name: "api response with absent optional fields sets null, not empty string",
			apiService: youtrack.Service{
				ID:   serviceTestID,
				Name: serviceTestName,
				Key:  serviceTestKey,
			},
			wantHomeURL:  false,
			wantRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var model serviceResourceModel
			model.fromAPIModel(&tt.apiService)

			helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), tt.apiService.ID)
			helpers.AssertFieldEqual(t, "Name", model.Name.ValueString(), tt.apiService.Name)
			helpers.AssertFieldEqual(t, "Key", model.Key.ValueString(), tt.apiService.Key)
			helpers.AssertFieldEqual(t, "Trusted", model.Trusted.ValueBool(), tt.apiService.Trusted)
			helpers.AssertFieldEqual(t, "ConsentRequired", model.ConsentRequired.ValueBool(), tt.apiService.ConsentRequired)
			helpers.AssertFieldEqual(t, "ClientCredentialsFlowEnabled", model.ClientCredentialsFlowEnabled.ValueBool(), tt.apiService.ClientCredentialsFlowEnabled)
			helpers.AssertFieldEqual(t, "AuthCodeFlowEnabled", model.AuthCodeFlowEnabled.ValueBool(), tt.apiService.AuthCodeFlowEnabled)
			helpers.AssertFieldEqual(t, "PKCERequired", model.PKCERequired.ValueBool(), tt.apiService.PKCERequired)
			helpers.AssertFieldEqual(t, "ImplicitFlowEnabled", model.ImplicitFlowEnabled.ValueBool(), tt.apiService.ImplicitFlowEnabled)
			helpers.AssertFieldEqual(t, "ResourceOwnerFlowEnabled", model.ResourceOwnerFlowEnabled.ValueBool(), tt.apiService.ResourceOwnerFlowEnabled)

			if tt.wantHomeURL {
				helpers.AssertFieldEqual(t, "HomeURL", model.HomeURL.ValueString(), serviceTestHomeURL)
			} else {
				helpers.AssertFieldEqual(t, "HomeURL.IsNull", model.HomeURL.IsNull(), true)
			}

			if tt.wantRedirect {
				helpers.AssertFieldEqual(t, "RedirectURIs.IsNull", model.RedirectURIs.IsNull(), false)
				uris, ok := helpers.ListToStringSlice(context.Background(), model.RedirectURIs)
				if !ok || len(uris) != 1 || uris[0] != tt.wantRedirectAt {
					t.Fatalf("unexpected RedirectURIs: %v", uris)
				}
			} else {
				helpers.AssertFieldEqual(t, "RedirectURIs.IsNull", model.RedirectURIs.IsNull(), true)
			}
		})
	}
}

func mustStringList(t *testing.T, values ...string) types.List {
	t.Helper()

	list, diags := types.ListValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatalf("failed to build string list: %v", diags)
	}

	return list
}
