// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// Error message titles
	errCreatingOAuth2Module  = "Error creating OAuth2 auth module"
	errReadingOAuth2Module   = "Error reading OAuth2 auth module"
	errUpdatingOAuth2Module  = "Error updating OAuth2 auth module"
	errDeletingOAuth2Module  = "Error deleting OAuth2 auth module"
	errImportingOAuth2Module = "Error importing OAuth2 auth module"
	errMissingOAuth2ModuleID = "Missing OAuth2 auth module ID"

	// Error message details
	errOAuth2ModuleIDRequired = "OAuth2 auth module ID is required to read the module"
)

// oauth2AuthModuleResourceModel maps the Terraform resource schema data.
type oauth2AuthModuleResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Disabled               types.Bool   `tfsdk:"disabled"`
	ClientID               types.String `tfsdk:"client_id"`
	ClientSecret           types.String `tfsdk:"client_secret"`
	ServerURL              types.String `tfsdk:"server_url"`
	TokenURL               types.String `tfsdk:"token_url"`
	Scope                  types.String `tfsdk:"scope"`
	UserInfoURL            types.String `tfsdk:"user_info_url"`
	RedirectURI            types.String `tfsdk:"redirect_uri"`
	FormClientAuth         types.Bool   `tfsdk:"form_client_auth"`
	EmailVerifiedByDefault types.Bool   `tfsdk:"email_verified_by_default"`
	AllowedCreateNewUsers  types.Bool   `tfsdk:"allowed_create_new_users"`
	BackgroundSyncEnabled  types.Bool   `tfsdk:"background_sync_enabled"`
	IDPLogoutURL           types.String `tfsdk:"idp_logout_url"`
	UserIDPath             types.String `tfsdk:"user_id_path"`
	UserEmailPath          types.String `tfsdk:"user_email_path"`
	UserEmailVerifiedPath  types.String `tfsdk:"user_email_verified_path"`
	UserNamePath           types.String `tfsdk:"user_name_path"`
	FullNamePath           types.String `tfsdk:"full_name_path"`
	UserEmailURL           types.String `tfsdk:"user_email_url"`
	UserAvatarURL          types.String `tfsdk:"user_avatar_url"`
	UserPictureIDPath      types.String `tfsdk:"user_picture_id_path"`
	UserPictureURLPattern  types.String `tfsdk:"user_picture_url_pattern"`
	UserGroupsPath         types.String `tfsdk:"user_groups_path"`
	IconURL                types.String `tfsdk:"icon_url"`
	ExtensionGrantType     types.String `tfsdk:"extension_grant_type"`
	ConnectionTimeout      types.Int64  `tfsdk:"connection_timeout"`
	ReadTimeout            types.Int64  `tfsdk:"read_timeout"`
	SyncInterval           types.String `tfsdk:"sync_interval"`
	IsDefault              types.Bool   `tfsdk:"is_default"`
}

// toAPIModel converts the Terraform model to the API model.
func (m *oauth2AuthModuleResourceModel) toAPIModel() youtrack.OAuth2AuthModule {
	return youtrack.OAuth2AuthModule{
		Name:                   m.Name.ValueString(),
		Disabled:               m.Disabled.ValueBool(),
		ClientID:               m.ClientID.ValueString(),
		ClientSecret:           m.ClientSecret.ValueString(),
		ServerURL:              m.ServerURL.ValueString(),
		TokenURL:               m.TokenURL.ValueString(),
		Scope:                  m.Scope.ValueString(),
		UserInfoURL:            m.UserInfoURL.ValueString(),
		RedirectURI:            m.RedirectURI.ValueString(),
		FormClientAuth:         m.FormClientAuth.ValueBool(),
		EmailVerifiedByDefault: m.EmailVerifiedByDefault.ValueBool(),
		AllowedCreateNewUsers:  m.AllowedCreateNewUsers.ValueBool(),
		BackgroundSyncEnabled:  m.BackgroundSyncEnabled.ValueBool(),
		IDPLogoutURL:           m.IDPLogoutURL.ValueString(),
		UserIDPath:             m.UserIDPath.ValueString(),
		UserEmailPath:          m.UserEmailPath.ValueString(),
		UserEmailVerifiedPath:  m.UserEmailVerifiedPath.ValueString(),
		UserNamePath:           m.UserNamePath.ValueString(),
		FullNamePath:           m.FullNamePath.ValueString(),
		UserEmailURL:           m.UserEmailURL.ValueString(),
		UserAvatarURL:          m.UserAvatarURL.ValueString(),
		UserPictureIDPath:      m.UserPictureIDPath.ValueString(),
		UserPictureURLPattern:  m.UserPictureURLPattern.ValueString(),
		UserGroupsPath:         m.UserGroupsPath.ValueString(),
		IconURL:                m.IconURL.ValueString(),
		ExtensionGrantType:     m.ExtensionGrantType.ValueString(),
		ConnectionTimeout:      int(m.ConnectionTimeout.ValueInt64()),
		ReadTimeout:            int(m.ReadTimeout.ValueInt64()),
		SyncInterval:           m.SyncInterval.ValueString(),
		IsDefault:              m.IsDefault.ValueBool(),
	}
}

// fromAPIModel populates the Terraform model from the API model, preserving the client secret
// from the prior state since the API does not return it.
func (m *oauth2AuthModuleResourceModel) fromAPIModel(api *youtrack.OAuth2AuthModule) {
	m.ID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.Disabled = types.BoolValue(api.Disabled)
	m.ClientID = types.StringValue(api.ClientID)
	m.FormClientAuth = types.BoolValue(api.FormClientAuth)
	m.EmailVerifiedByDefault = types.BoolValue(api.EmailVerifiedByDefault)
	m.AllowedCreateNewUsers = types.BoolValue(api.AllowedCreateNewUsers)
	m.BackgroundSyncEnabled = types.BoolValue(api.BackgroundSyncEnabled)

	// client_secret is not returned by the API; preserve existing state value.

	m.ServerURL = types.StringValue(api.ServerURL)
	m.TokenURL = types.StringValue(api.TokenURL)
	m.Scope = helpers.StringOrNull(api.Scope)
	m.UserInfoURL = helpers.StringOrNull(api.UserInfoURL)
	m.RedirectURI = helpers.StringOrNull(api.RedirectURI)
	m.IDPLogoutURL = types.StringValue(api.IDPLogoutURL)
	m.UserIDPath = helpers.StringOrNull(api.UserIDPath)
	m.UserEmailPath = helpers.StringOrNull(api.UserEmailPath)
	m.UserEmailVerifiedPath = helpers.StringOrNull(api.UserEmailVerifiedPath)
	m.UserNamePath = helpers.StringOrNull(api.UserNamePath)
	m.FullNamePath = helpers.StringOrNull(api.FullNamePath)
	m.UserEmailURL = helpers.StringOrNull(api.UserEmailURL)
	m.UserAvatarURL = helpers.StringOrNull(api.UserAvatarURL)
	m.UserPictureIDPath = helpers.StringOrNull(api.UserPictureIDPath)
	m.UserPictureURLPattern = helpers.StringOrNull(api.UserPictureURLPattern)
	m.UserGroupsPath = helpers.StringOrNull(api.UserGroupsPath)
	m.IconURL = helpers.StringOrNull(api.IconURL)
	m.ExtensionGrantType = helpers.StringOrNull(api.ExtensionGrantType)

	if api.ConnectionTimeout > 0 {
		m.ConnectionTimeout = types.Int64Value(int64(api.ConnectionTimeout))
	} else {
		m.ConnectionTimeout = types.Int64Null()
	}

	if api.ReadTimeout > 0 {
		m.ReadTimeout = types.Int64Value(int64(api.ReadTimeout))
	} else {
		m.ReadTimeout = types.Int64Null()
	}

	m.SyncInterval = helpers.StringOrNull(api.SyncInterval)
	m.IsDefault = types.BoolValue(api.IsDefault)
}
