// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &serviceResource{}
	_ resource.ResourceWithConfigure   = &serviceResource{}
	_ resource.ResourceWithImportState = &serviceResource{}
)

// NewServiceResource is a helper function to simplify the provider implementation.
func NewServiceResource() resource.Resource {
	return &serviceResource{}
}

// serviceResource is the resource implementation.
type serviceResource struct {
	client *youtrack.Client
}

// preserveBoolPlanModifiers is shared by every Optional+Computed+Default bool attribute in this
// package that can change out-of-band (e.g. directly in Hub or the YouTrack UI), so an unrelated
// apply doesn't silently revert it back to the schema default. See helpers.PreserveStateWhenUnconfiguredBool.
var preserveBoolPlanModifiers = []planmodifier.Bool{helpers.PreserveStateWhenUnconfiguredBool}

// Metadata returns the resource type name.
func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

// Schema defines the schema for the resource.
func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hub service: an external application registered for authentication/authorization " +
			"(for example an OAuth client used to integrate an MCP server or another service with Hub).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Hub ID of the service (computed).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the service.",
			},
			"key": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Unique key identifying the service. Defaults to the service name when omitted.",
			},
			"home_url": schema.StringAttribute{
				Optional:    true,
				Description: "Home URL of the service.",
			},
			"application_name": schema.StringAttribute{
				Optional: true,
				Description: "Name of the application this service represents. Can only be set at creation; " +
					"Hub only allows the service itself to change this afterward.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the service.",
			},
			"vendor": schema.StringAttribute{
				Optional: true,
				Description: "Vendor of the service. Can only be set at creation; Hub only allows the service " +
					"itself to change this afterward.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Optional: true,
				Description: "Version of the service. Can only be set at creation; Hub only allows the service " +
					"itself to change this afterward.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"redirect_uris": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "OAuth 2.0 redirect URIs allowed for this service.",
			},
			"base_urls": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Base URLs from which the service is allowed to make requests.",
			},
			"trusted": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Whether the service is trusted. Trusted services skip the user consent screen. Defaults to false.",
				PlanModifiers: preserveBoolPlanModifiers,
			},
			"consent_required": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(true),
				Description:   "Whether Hub must ask the user for consent before granting this service access. Defaults to true.",
				PlanModifiers: preserveBoolPlanModifiers,
			},
			"client_credentials_flow_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(true),
				Description:   "Whether the OAuth 2.0 client credentials grant flow is enabled for this service. Defaults to true.",
				PlanModifiers: preserveBoolPlanModifiers,
			},
			"auth_code_flow_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(true),
				Description:   "Whether the OAuth 2.0 authorization code grant flow is enabled for this service. Defaults to true.",
				PlanModifiers: preserveBoolPlanModifiers,
			},
			"pkce_required": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Whether PKCE is required for the authorization code grant flow. Defaults to false.",
				PlanModifiers: preserveBoolPlanModifiers,
			},
			"implicit_flow_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(true),
				Description:   "Whether the OAuth 2.0 implicit grant flow is enabled for this service. Defaults to true.",
				PlanModifiers: preserveBoolPlanModifiers,
			},
			"resource_owner_flow_enabled": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(true),
				Description:   "Whether the OAuth 2.0 resource owner password credentials grant flow is enabled for this service. Defaults to true.",
				PlanModifiers: preserveBoolPlanModifiers,
			},
			"secret": schema.StringAttribute{
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				Description: "Client secret for the service. When omitted, Hub generates one automatically. " +
					"This value is write-only and is not returned by the API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, ok := helpers.GetClientFromConfigure(req, resp)
	if !ok {
		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.ops().Create(ctx, req, resp)
}

// Read refreshes the Terraform state with the latest data.
func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.ops().Read(ctx, req, resp)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.ops().Update(ctx, req, resp)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.ops().Delete(ctx, req, resp)
}

// ImportState imports the resource state by service ID.
func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.ops().ImportState(ctx, req, resp)
}
