// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &oauth2AuthModuleResource{}
	_ resource.ResourceWithConfigure   = &oauth2AuthModuleResource{}
	_ resource.ResourceWithImportState = &oauth2AuthModuleResource{}
)

// NewOAuth2AuthModuleResource is a helper function to simplify the provider implementation.
func NewOAuth2AuthModuleResource() resource.Resource {
	return &oauth2AuthModuleResource{}
}

// oauth2AuthModuleResource is the resource implementation.
type oauth2AuthModuleResource struct {
	client *youtrack.Client
}

// Metadata returns the resource type name.
func (r *oauth2AuthModuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_module_oauth2"
}

// Schema defines the schema for the resource.
func (r *oauth2AuthModuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := authModuleCommonAttributes(oauth2ProviderName)

	attributes["client_id"] = schema.StringAttribute{
		Required:    true,
		Description: "OAuth 2.0 client ID registered with the identity provider.",
	}
	attributes["client_secret"] = schema.StringAttribute{
		Required:    true,
		Sensitive:   true,
		Description: "OAuth 2.0 client secret. This value is write-only and is not returned by the API.",
	}
	attributes["server_url"] = schema.StringAttribute{
		Required:    true,
		Description: "Base URL of the OAuth 2.0 authorization server (identity provider).",
	}
	attributes["token_url"] = schema.StringAttribute{
		Required:    true,
		Description: "URL of the token endpoint at the identity provider.",
	}
	attributes["scope"] = schema.StringAttribute{
		Optional:    true,
		Description: "Space-separated list of OAuth 2.0 scopes to request.",
	}
	attributes["user_info_url"] = schema.StringAttribute{
		Optional:    true,
		Description: "URL of the userinfo endpoint at the identity provider.",
	}
	attributes["form_client_auth"] = schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Whether to send client credentials in the request body instead of the Authorization header. Defaults to false.",
	}
	attributes["email_verified_by_default"] = schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Whether email addresses from this provider are considered verified by default. Defaults to false.",
	}
	attributes["idp_logout_url"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "URL of the identity provider logout endpoint.",
	}
	attributes["user_id_path"] = schema.StringAttribute{
		Required:    true,
		Description: "JSON path expression to the user ID claim in the identity provider response.",
	}
	attributes["user_email_path"] = schema.StringAttribute{
		Optional:    true,
		Description: "JSON path expression to the email claim in the identity provider response.",
	}
	attributes["user_email_verified_path"] = schema.StringAttribute{
		Optional:    true,
		Description: "JSON path expression to the email-verified claim in the identity provider response.",
	}
	attributes["user_name_path"] = schema.StringAttribute{
		Optional:    true,
		Description: "JSON path expression to the username claim in the identity provider response.",
	}
	attributes["full_name_path"] = schema.StringAttribute{
		Optional:    true,
		Description: "JSON path expression to the full name claim in the identity provider response.",
	}
	attributes["user_email_url"] = schema.StringAttribute{
		Optional:    true,
		Description: "URL used by Hub to retrieve the user's email address from the identity provider.",
	}
	attributes["user_avatar_url"] = schema.StringAttribute{
		Optional:    true,
		Description: "URL used by Hub to retrieve the user's avatar from the identity provider.",
	}
	attributes["user_picture_id_path"] = schema.StringAttribute{
		Optional:    true,
		Description: "JSON path expression to the picture ID claim in the identity provider response.",
	}
	attributes["user_picture_url_pattern"] = schema.StringAttribute{
		Optional:    true,
		Description: "URL pattern used to build the picture URL from the picture ID.",
	}
	attributes["user_groups_path"] = schema.StringAttribute{
		Optional:    true,
		Description: "JSON path expression to the groups claim in the identity provider response.",
	}

	resp.Schema = schema.Schema{
		Description: "Manages a Hub OAuth 2.0 authentication module. " +
			"This resource allows configuring an external OAuth 2.0 identity provider for Hub.",
		Attributes: attributes,
	}
}

// Configure adds the provider configured client to the resource.
func (r *oauth2AuthModuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := helpers.GetClientFromConfigure(req, resp)
	if !ok {
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *oauth2AuthModuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.ops().Create(ctx, req, resp)
}

// Read refreshes the Terraform state with the latest data.
func (r *oauth2AuthModuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.ops().Read(ctx, req, resp)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *oauth2AuthModuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.ops().Update(ctx, req, resp)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *oauth2AuthModuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.ops().Delete(ctx, req, resp)
}

// ImportState imports the resource state by module ID.
func (r *oauth2AuthModuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.ops().ImportState(ctx, req, resp)
}
