// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	oauth2ModuleID       = "auth-module-123"
	oauth2ModuleName     = "Test OAuth2 Module"
	oauth2ClientID       = "my-client-id"
	oauth2ClientSecret   = "my-client-secret"
	oauth2ServerURL      = "https://idp.example.com"
	oauth2TokenURL       = "https://idp.example.com/token" //nolint:gosec // G101: test constant, not a hardcoded credential
	oauth2UserInfoURL    = "https://idp.example.com/userinfo"
	oauth2Scope          = "openid email profile"
	oauth2IDPLogoutURL   = "https://idp.example.com/logout"
	oauth2RedirectURI    = "https://hub.example.com/hub/api/rest/oauth2/auth"
	oauth2UserIDPath     = "sub"
	oauth2UserEmailPath  = "email"
	oauth2UserNamePath   = "preferred_username"
	oauth2FullNamePath   = "name"
	oauth2UserGroupsPath = "groups"
)

func newMinimalOAuth2Model() oauth2AuthModuleResourceModel {
	return oauth2AuthModuleResourceModel{
		ID:                     types.StringValue(oauth2ModuleID),
		Name:                   types.StringValue(oauth2ModuleName),
		Disabled:               types.BoolValue(false),
		ClientID:               types.StringValue(oauth2ClientID),
		ClientSecret:           types.StringValue(oauth2ClientSecret),
		ServerURL:              types.StringValue(oauth2ServerURL),
		TokenURL:               types.StringValue(oauth2TokenURL),
		Scope:                  types.StringNull(),
		UserInfoURL:            types.StringNull(),
		RedirectURI:            types.StringNull(),
		FormClientAuth:         types.BoolValue(false),
		EmailVerifiedByDefault: types.BoolValue(false),
		AllowedCreateNewUsers:  types.BoolValue(false),
		BackgroundSyncEnabled:  types.BoolValue(false),
		IDPLogoutURL:           types.StringNull(),
		UserIDPath:             types.StringNull(),
		UserEmailPath:          types.StringNull(),
		UserEmailVerifiedPath:  types.StringNull(),
		UserNamePath:           types.StringNull(),
		FullNamePath:           types.StringNull(),
		UserEmailURL:           types.StringNull(),
		UserAvatarURL:          types.StringNull(),
		UserPictureIDPath:      types.StringNull(),
		UserPictureURLPattern:  types.StringNull(),
		UserGroupsPath:         types.StringNull(),
		IconURL:                types.StringNull(),
		ExtensionGrantType:     types.StringNull(),
		ConnectionTimeout:      types.Int64Null(),
		ReadTimeout:            types.Int64Null(),
		SyncInterval:           types.StringNull(),
	}
}

func TestOAuth2AuthModuleModelToAPIModel(t *testing.T) {
	tests := []struct {
		name  string
		model oauth2AuthModuleResourceModel
		want  youtrack.OAuth2AuthModule
	}{
		{
			name: "full model converts correctly",
			model: func() oauth2AuthModuleResourceModel {
				m := newMinimalOAuth2Model()
				m.Scope = types.StringValue(oauth2Scope)
				m.UserInfoURL = types.StringValue(oauth2UserInfoURL)
				m.RedirectURI = types.StringValue(oauth2RedirectURI)
				m.EmailVerifiedByDefault = types.BoolValue(true)
				m.AllowedCreateNewUsers = types.BoolValue(true)
				m.IDPLogoutURL = types.StringValue(oauth2IDPLogoutURL)
				m.UserIDPath = types.StringValue(oauth2UserIDPath)
				m.UserEmailPath = types.StringValue(oauth2UserEmailPath)
				m.UserNamePath = types.StringValue(oauth2UserNamePath)
				m.FullNamePath = types.StringValue(oauth2FullNamePath)
				m.UserGroupsPath = types.StringValue(oauth2UserGroupsPath)
				m.ConnectionTimeout = types.Int64Value(5000)
				m.ReadTimeout = types.Int64Value(10000)
				return m
			}(),
			want: youtrack.OAuth2AuthModule{
				Name:                   oauth2ModuleName,
				Disabled:               false,
				ClientID:               oauth2ClientID,
				ClientSecret:           oauth2ClientSecret,
				ServerURL:              oauth2ServerURL,
				TokenURL:               oauth2TokenURL,
				Scope:                  oauth2Scope,
				UserInfoURL:            oauth2UserInfoURL,
				RedirectURI:            oauth2RedirectURI,
				FormClientAuth:         false,
				EmailVerifiedByDefault: true,
				AllowedCreateNewUsers:  true,
				BackgroundSyncEnabled:  false,
				IDPLogoutURL:           oauth2IDPLogoutURL,
				UserIDPath:             oauth2UserIDPath,
				UserEmailPath:          oauth2UserEmailPath,
				UserNamePath:           oauth2UserNamePath,
				FullNamePath:           oauth2FullNamePath,
				UserGroupsPath:         oauth2UserGroupsPath,
				ConnectionTimeout:      5000,
				ReadTimeout:            10000,
			},
		},
		{
			name:  "minimal model with required fields only",
			model: newMinimalOAuth2Model(),
			want: youtrack.OAuth2AuthModule{
				Name:         oauth2ModuleName,
				ClientID:     oauth2ClientID,
				ClientSecret: oauth2ClientSecret,
				ServerURL:    oauth2ServerURL,
				TokenURL:     oauth2TokenURL,
			},
		},
		{
			name: "disabled module",
			model: func() oauth2AuthModuleResourceModel {
				m := newMinimalOAuth2Model()
				m.Disabled = types.BoolValue(true)
				return m
			}(),
			want: youtrack.OAuth2AuthModule{
				Name:         oauth2ModuleName,
				Disabled:     true,
				ClientID:     oauth2ClientID,
				ClientSecret: oauth2ClientSecret,
				ServerURL:    oauth2ServerURL,
				TokenURL:     oauth2TokenURL,
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
			helpers.AssertFieldEqual(t, "ServerURL", got.ServerURL, tt.want.ServerURL)
			helpers.AssertFieldEqual(t, "TokenURL", got.TokenURL, tt.want.TokenURL)
			helpers.AssertFieldEqual(t, "Scope", got.Scope, tt.want.Scope)
			helpers.AssertFieldEqual(t, "UserInfoURL", got.UserInfoURL, tt.want.UserInfoURL)
			helpers.AssertFieldEqual(t, "RedirectURI", got.RedirectURI, tt.want.RedirectURI)
			helpers.AssertFieldEqual(t, "FormClientAuth", got.FormClientAuth, tt.want.FormClientAuth)
			helpers.AssertFieldEqual(t, "EmailVerifiedByDefault", got.EmailVerifiedByDefault, tt.want.EmailVerifiedByDefault)
			helpers.AssertFieldEqual(t, "AllowedCreateNewUsers", got.AllowedCreateNewUsers, tt.want.AllowedCreateNewUsers)
			helpers.AssertFieldEqual(t, "BackgroundSyncEnabled", got.BackgroundSyncEnabled, tt.want.BackgroundSyncEnabled)
			helpers.AssertFieldEqual(t, "IDPLogoutURL", got.IDPLogoutURL, tt.want.IDPLogoutURL)
			helpers.AssertFieldEqual(t, "UserIDPath", got.UserIDPath, tt.want.UserIDPath)
			helpers.AssertFieldEqual(t, "UserEmailPath", got.UserEmailPath, tt.want.UserEmailPath)
			helpers.AssertFieldEqual(t, "UserNamePath", got.UserNamePath, tt.want.UserNamePath)
			helpers.AssertFieldEqual(t, "FullNamePath", got.FullNamePath, tt.want.FullNamePath)
			helpers.AssertFieldEqual(t, "UserGroupsPath", got.UserGroupsPath, tt.want.UserGroupsPath)
			helpers.AssertFieldEqual(t, "ConnectionTimeout", got.ConnectionTimeout, tt.want.ConnectionTimeout)
			helpers.AssertFieldEqual(t, "ReadTimeout", got.ReadTimeout, tt.want.ReadTimeout)
		})
	}
}

func TestOAuth2AuthModuleFromAPIModel(t *testing.T) {
	tests := []struct {
		name          string
		apiModule     youtrack.OAuth2AuthModule
		wantID        string
		wantName      string
		wantDisabled  bool
		wantClientID  string
		wantRedirect  bool // true if redirect_uri should be set
		wantScopeNull bool
	}{
		{
			name: "full api response populates model",
			apiModule: youtrack.OAuth2AuthModule{
				ID:                     oauth2ModuleID,
				Name:                   oauth2ModuleName,
				Disabled:               false,
				ClientID:               oauth2ClientID,
				ServerURL:              oauth2ServerURL,
				TokenURL:               oauth2TokenURL,
				Scope:                  oauth2Scope,
				UserInfoURL:            oauth2UserInfoURL,
				RedirectURI:            oauth2RedirectURI,
				FormClientAuth:         false,
				EmailVerifiedByDefault: true,
				AllowedCreateNewUsers:  true,
				BackgroundSyncEnabled:  false,
				UserIDPath:             oauth2UserIDPath,
				UserEmailPath:          oauth2UserEmailPath,
				UserNamePath:           oauth2UserNamePath,
			},
			wantID:        oauth2ModuleID,
			wantName:      oauth2ModuleName,
			wantDisabled:  false,
			wantClientID:  oauth2ClientID,
			wantRedirect:  true,
			wantScopeNull: false,
		},
		{
			name: "api response with empty scope sets null",
			apiModule: youtrack.OAuth2AuthModule{
				ID:        oauth2ModuleID,
				Name:      oauth2ModuleName,
				ClientID:  oauth2ClientID,
				ServerURL: oauth2ServerURL,
				TokenURL:  oauth2TokenURL,
			},
			wantID:        oauth2ModuleID,
			wantName:      oauth2ModuleName,
			wantDisabled:  false,
			wantClientID:  oauth2ClientID,
			wantRedirect:  false,
			wantScopeNull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var model oauth2AuthModuleResourceModel
			model.fromAPIModel(&tt.apiModule)

			helpers.AssertFieldEqual(t, "ID", model.ID.ValueString(), tt.wantID)
			helpers.AssertFieldEqual(t, "Name", model.Name.ValueString(), tt.wantName)
			helpers.AssertFieldEqual(t, "Disabled", model.Disabled.ValueBool(), tt.wantDisabled)
			helpers.AssertFieldEqual(t, "ClientID", model.ClientID.ValueString(), tt.wantClientID)
			helpers.AssertFieldEqual(t, "Scope.IsNull", model.Scope.IsNull(), tt.wantScopeNull)

			if tt.wantRedirect {
				helpers.AssertFieldEqual(t, "RedirectURI", model.RedirectURI.ValueString(), oauth2RedirectURI)
			} else {
				helpers.AssertFieldEqual(t, "RedirectURI.IsNull", model.RedirectURI.IsNull(), true)
			}
		})
	}
}
