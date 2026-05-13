package settings

import (
	"context"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// Default system settings values
	defaultMaxExportItems    = 500
	defaultMaxUploadFileSize = 10 * 1024 * 1024 // 10MB in bytes
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &systemSettingsResource{}
	_ resource.ResourceWithConfigure      = &systemSettingsResource{}
	_ resource.ResourceWithImportState    = &systemSettingsResource{}
	_ resource.ResourceWithValidateConfig = &systemSettingsResource{}
)

// NewSystemSettingsResource is a helper function to simplify the provider implementation.
func NewSystemSettingsResource() resource.Resource {
	return &systemSettingsResource{}
}

// systemSettingsResource is the resource implementation.
type systemSettingsResource struct {
	client *youtrack.Client
}

type systemSettingsResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	AdministratorEmail        types.String `tfsdk:"administrator_email"`
	MaxExportItems            types.Int64  `tfsdk:"max_export_items"`
	MaxUploadFileSize         types.Int64  `tfsdk:"max_upload_file_size"`
	AllowStatisticsCollection types.Bool   `tfsdk:"allow_statistics_collection"`
	IsApplicationReadOnly     types.Bool   `tfsdk:"is_application_read_only"`
	BaseURL                   types.String `tfsdk:"base_url"`
	LastUpdated               types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *systemSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_settings"
}

// Schema defines the schema for the resource.
func (r *systemSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack system settings resource. This resource manages the global system settings in YouTrack, such as administrator email, maximum export items, and other global configurations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "System settings identifier (always 'global').",
				Computed:    true,
			},
			"administrator_email": schema.StringAttribute{
				Description: "Email address of the YouTrack administrator.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"max_export_items": schema.Int64Attribute{
				Description: "Maximum number of items that can be exported at once. Default is 1000.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(defaultMaxExportItems),
			},
			"max_upload_file_size": schema.Int64Attribute{
				Description: "Maximum file size (in bytes) that can be uploaded to YouTrack. Default is 10485760 (10MB).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(defaultMaxUploadFileSize),
			},
			"allow_statistics_collection": schema.BoolAttribute{
				Description: "Indicates whether YouTrack is allowed to collect usage statistics. Default is false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"is_application_read_only": schema.BoolAttribute{
				Description: "Indicates whether the YouTrack application is in read-only mode. Default is false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"base_url": schema.StringAttribute{
				Description: "Base URL of the YouTrack instance.",
				Required:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last update",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *systemSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemSettingsResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	ss := convertModelToSystemSettings(plan)

	systemSettings, ok := r.updateAndHandleError(ctx, ss, &resp.Diagnostics)
	if !ok {
		return
	}

	updateSystemSettingsModelWithTimestamp(systemSettings, &plan)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *systemSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemSettingsResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	systemSettings, ok := r.getSystemSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	updateSystemSettingsModelWithTimestamp(systemSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *systemSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemSettingsResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	ss := convertModelToSystemSettings(plan)

	systemSettings, ok := r.updateAndHandleError(ctx, ss, &resp.Diagnostics)
	if !ok {
		return
	}

	// Map response body to model and update timestamp
	updateSystemSettingsModelWithTimestamp(systemSettings, &plan)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *systemSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemSettingsResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	var ss = youtrack.SystemSettings{
		AdministratorEmail:        "",
		MaxExportItems:            defaultMaxExportItems,
		MaxUploadFileSize:         defaultMaxUploadFileSize,
		AllowStatisticsCollection: false,
		IsApplicationReadOnly:     false,
		BaseUrl:                   state.BaseURL.ValueString(),
	}

	systemSettings, ok := r.updateAndHandleError(ctx, ss, &resp.Diagnostics)
	if !ok {
		return
	}

	updateSystemSettingsModelWithTimestamp(systemSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// ImportState imports the resource state.
func (r *systemSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	systemSettings, ok := r.getSystemSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	var state systemSettingsResourceModel
	updateSystemSettingsModelWithTimestamp(systemSettings, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// ValidateConfig validates the resource configuration.
func (r *systemSettingsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config systemSettingsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate administrator_email format if provided
	helpers.ValidateEmailField(config.AdministratorEmail, path.Root("administrator_email"), "administrator_email", &resp.Diagnostics)

	// Validate max_export_items is positive
	validatePositiveInt64(config.MaxExportItems, path.Root("max_export_items"), "max_export_items", &resp.Diagnostics)

	// Validate max_upload_file_size is positive
	validatePositiveInt64(config.MaxUploadFileSize, path.Root("max_upload_file_size"), "max_upload_file_size", &resp.Diagnostics)

	// Validate base_url is a valid URL
	validateURLField(config.BaseURL, path.Root("base_url"), "base_url", &resp.Diagnostics)
}

// Configure adds the provider configured client to the resource.
func (r *systemSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}
