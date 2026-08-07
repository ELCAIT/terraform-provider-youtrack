// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

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
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the service is trusted. Trusted services skip the user consent screen. Defaults to false.",
			},
			"consent_required": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether Hub must ask the user for consent before granting this service access. Defaults to true.",
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
	var plan serviceResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	apiService := plan.toAPIModel(ctx)
	created, err := r.client.CreateService(ctx, apiService)
	if err != nil {
		resp.Diagnostics.AddError(errCreatingService, fmt.Sprintf("Could not create service: %v", err))
		return
	}

	// Preserve secret from what was sent/generated since the API does not return it on subsequent reads.
	plan.fromAPIModel(created)
	plan.Secret = types.StringValue(created.Secret)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingServiceID, errServiceIDRequired) {
		return
	}

	apiService, err := r.client.GetServiceByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(errReadingService, fmt.Sprintf("Could not read service: %v", err))
		return
	}

	// Preserve secret from existing state; the API never returns it.
	existingSecret := state.Secret
	state.fromAPIModel(apiService)
	state.Secret = existingSecret

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	serviceID := plan.ID.ValueString()
	apiService := plan.toAPIModel(ctx)

	updated, err := r.client.UpdateService(ctx, serviceID, apiService)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingService, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "service", err))
		return
	}

	// Preserve secret from what was sent since the API does not return it.
	plan.fromAPIModel(updated)
	plan.Secret = types.StringValue(apiService.Secret)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	if !helpers.HasResourceID(state.ID) {
		return
	}

	err := r.client.DeleteService(ctx, state.ID.ValueString())
	if err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(errDeletingService, fmt.Sprintf("Could not delete service: %v", err))
	}
}

// ImportState imports the resource state by service ID.
func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serviceID := strings.TrimSpace(req.ID)

	apiService, err := r.client.GetServiceByID(ctx, serviceID)
	if err != nil {
		resp.Diagnostics.AddError(errImportingService, fmt.Sprintf("Could not read service with ID '%s': %v", serviceID, err))
		return
	}

	var state serviceResourceModel
	state.fromAPIModel(apiService)
	// secret cannot be imported; set to empty string as a placeholder.
	state.Secret = types.StringValue("")

	helpers.SetStateAndCheckError(ctx, resp, &state)
}
