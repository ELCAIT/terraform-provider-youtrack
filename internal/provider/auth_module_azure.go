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
	_ resource.Resource                = &azureAuthModuleResource{}
	_ resource.ResourceWithConfigure   = &azureAuthModuleResource{}
	_ resource.ResourceWithImportState = &azureAuthModuleResource{}
)

// NewAzureAuthModuleResource is a helper function to simplify the provider implementation.
func NewAzureAuthModuleResource() resource.Resource {
	return &azureAuthModuleResource{}
}

// azureAuthModuleResource is the resource implementation.
type azureAuthModuleResource struct {
	client *youtrack.Client
}

// Metadata returns the resource type name.
func (r *azureAuthModuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_module_azure"
}

// Schema defines the schema for the resource.
func (r *azureAuthModuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := authModuleCommonAttributes(azureProviderName)

	attributes["client_id"] = schema.StringAttribute{
		Required:    true,
		Description: "Application (client) ID of the Entra ID app registration.",
	}
	attributes["client_secret"] = schema.StringAttribute{
		Required:    true,
		Sensitive:   true,
		Description: "Client secret of the Entra ID app registration. This value is write-only and is not returned by the API.",
	}
	attributes["tenant"] = schema.StringAttribute{
		Optional:    true,
		Description: "Entra ID tenant (directory) ID, a GUID. Leave unset to allow sign-in from any Microsoft tenant (\"common\").",
	}
	attributes["server_url"] = schema.StringAttribute{
		Computed:    true,
		Description: "Authorization endpoint URL, derived by Hub from tenant (computed).",
	}
	attributes["request_group_permission"] = schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Whether Hub requests Microsoft Graph group-read permissions for group synchronisation. Defaults to false.",
	}
	attributes["request_id_token"] = schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Whether Hub requests an ID token from Entra ID. Defaults to false.",
	}

	resp.Schema = schema.Schema{
		Description: "Manages a Hub Microsoft Entra ID (formerly Azure AD) authentication module. " +
			"This resource allows configuring Microsoft Entra ID as an external identity provider for Hub.",
		Attributes: attributes,
	}
}

// Configure adds the provider configured client to the resource.
func (r *azureAuthModuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := helpers.GetClientFromConfigure(req, resp)
	if !ok {
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *azureAuthModuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.ops().Create(ctx, req, resp)
}

// Read refreshes the Terraform state with the latest data.
func (r *azureAuthModuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.ops().Read(ctx, req, resp)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *azureAuthModuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.ops().Update(ctx, req, resp)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *azureAuthModuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.ops().Delete(ctx, req, resp)
}

// ImportState imports the resource state by module ID.
func (r *azureAuthModuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.ops().ImportState(ctx, req, resp)
}
