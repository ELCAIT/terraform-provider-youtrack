package settings

import (
	"context"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &backupSettingsResource{}
	_ resource.ResourceWithConfigure   = &backupSettingsResource{}
	_ resource.ResourceWithImportState = &backupSettingsResource{}
)

const (
	errConvertingBackupSettings = "Error converting backup settings"
	errUpdatingBackupSettings   = "Error updating backup settings"
	errLoginToUserIDResolution  = "Failed to resolve notified_users logins to user IDs"
)

func NewBackupSettingsResource() resource.Resource {
	return &backupSettingsResource{}
}

type backupSettingsResource struct {
	client *youtrack.Client
}

type backupSettingsResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Location       types.String `tfsdk:"location"`
	FilesToKeep    types.Int64  `tfsdk:"files_to_keep"`
	CronExpression types.String `tfsdk:"cron_expression"`
	ArchiveFormat  types.String `tfsdk:"archive_format"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	NotifiedUsers  types.List   `tfsdk:"notified_users"`
	LastUpdated    types.String `tfsdk:"last_updated"`
}

func (r *backupSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_backup_settings"
}

func (r *backupSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack database backup settings configuration",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Backup settings identifier",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"location": schema.StringAttribute{
				Description: "Backup storage location",
				Required:    true,
			},
			"files_to_keep": schema.Int64Attribute{
				Description: "Number of backup files to keep",
				Required:    true,
			},
			"cron_expression": schema.StringAttribute{
				Description: "Cron expression for backup schedule",
				Required:    true,
			},
			"archive_format": schema.StringAttribute{
				Description: "Archive format for backups (ZIP or TAR_GZ)",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether scheduled backup is enabled",
				Required:    true,
			},
			"notified_users": schema.ListAttribute{
				Description: "List of YouTrack user logins notified about backup status",
				ElementType: types.StringType,
				Required:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last update",
				Computed:    true,
			},
		},
	}
}

func (r *backupSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan backupSettingsResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	bs, ok := r.convertModelToBackupSettings(ctx, plan, &resp.Diagnostics)
	if !ok {
		resp.Diagnostics.AddError(errConvertingBackupSettings, errLoginToUserIDResolution)
		return
	}

	backupSettings, ok := r.updateAndHandleError(ctx, bs, &resp.Diagnostics)
	if !ok {
		return
	}

	if !updateBackupSettingsModelWithTimestamp(ctx, backupSettings, &plan) {
		resp.Diagnostics.AddError(errUpdatingBackupSettings, errStringSliceToListConversion)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

func (r *backupSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state backupSettingsResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	backupSettings, ok := r.getBackupSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	if !updateBackupSettingsModelWithTimestamp(ctx, backupSettings, &state) {
		resp.Diagnostics.AddError(errUpdatingBackupSettings, errStringSliceToListConversion)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *backupSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan backupSettingsResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	bs, ok := r.convertModelToBackupSettings(ctx, plan, &resp.Diagnostics)
	if !ok {
		resp.Diagnostics.AddError(errConvertingBackupSettings, errLoginToUserIDResolution)
		return
	}

	backupSettings, ok := r.updateAndHandleError(ctx, bs, &resp.Diagnostics)
	if !ok {
		return
	}

	if !updateBackupSettingsModelWithTimestamp(ctx, backupSettings, &plan) {
		resp.Diagnostics.AddError(errUpdatingBackupSettings, errStringSliceToListConversion)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

func (r *backupSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state backupSettingsResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}
}

func (r *backupSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	backupSettings, ok := r.getBackupSettingsAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	var state backupSettingsResourceModel
	if !updateBackupSettingsModelWithTimestamp(ctx, backupSettings, &state) {
		resp.Diagnostics.AddError(errUpdatingBackupSettings, errStringSliceToListConversion)
		return
	}

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *backupSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}
