package settings

import (
	"context"
	"fmt"
	"strings"
	"time"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	TestBackupPath        = "/opt/youtrack/backups"
	TestCronExpressionOne = "0 0 2 * * ?"
	TestBackupFormatZIP   = "ZIP"
	FilesToKeep           = 7
	errUnableToReadBackup = "Unable to Read YouTrack Backup Settings"
	errUpdatingBackup     = "Error updating backup settings"
)

func (r *backupSettingsResource) getBackupSettingsAndHandleError(ctx context.Context, diagnostics *diag.Diagnostics) (youtrack.BackupSettings, bool) {
	backupSettings, err := r.client.GetBackupSettings(ctx)
	if err != nil {
		diagnostics.AddError(
			errUnableToReadBackup,
			err.Error(),
		)
		return youtrack.BackupSettings{}, false
	}

	return backupSettings, true
}

func updateBackupSettingsModelWithTimestamp(ctx context.Context, backupSettings youtrack.BackupSettings, resourceModel *backupSettingsResourceModel) bool {
	converted := convertBackupSettingsToModel(ctx, backupSettings)
	if converted == nil {
		return false
	}

	*resourceModel = *converted
	resourceModel.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	return true
}

func (r *backupSettingsResource) convertModelToBackupSettings(ctx context.Context, model backupSettingsResourceModel, diagnostics *diag.Diagnostics) (youtrack.BackupSettings, bool) {
	logins, ok := helpers.ListToStringSlice(ctx, model.NotifiedUsers)
	if !ok {
		return youtrack.BackupSettings{}, false
	}

	notifiedUsers := make([]youtrack.User, 0, len(logins))
	for _, login := range logins {
		user, err := r.client.GetUserByLogin(ctx, login)
		if err != nil {
			diagnostics.AddError(
				errConvertingBackupSettings,
				fmt.Sprintf("Could not resolve user login '%s': %v", login, err),
			)
			return youtrack.BackupSettings{}, false
		}

		notifiedUsers = append(notifiedUsers, youtrack.User{ID: user.Id})
	}

	return youtrack.BackupSettings{
		ID:             model.ID.ValueString(),
		Location:       model.Location.ValueString(),
		FilesToKeep:    int(model.FilesToKeep.ValueInt64()),
		CronExpression: model.CronExpression.ValueString(),
		ArchiveFormat:  strings.ToUpper(model.ArchiveFormat.ValueString()),
		Enabled:        model.Enabled.ValueBool(),
		NotifiedUsers:  notifiedUsers,
	}, true
}

func convertBackupSettingsToModel(ctx context.Context, backupSettings youtrack.BackupSettings) *backupSettingsResourceModel {
	userLogins := make([]string, 0, len(backupSettings.NotifiedUsers))
	for _, user := range backupSettings.NotifiedUsers {
		if user.Login != "" {
			userLogins = append(userLogins, user.Login)
		}
	}

	notifiedUsers, diags := types.ListValueFrom(ctx, types.StringType, userLogins)
	if diags.HasError() {
		return nil
	}

	return &backupSettingsResourceModel{
		ID:             types.StringValue(backupSettings.ID),
		Location:       types.StringValue(backupSettings.Location),
		FilesToKeep:    types.Int64Value(int64(backupSettings.FilesToKeep)),
		CronExpression: types.StringValue(backupSettings.CronExpression),
		ArchiveFormat:  types.StringValue(backupSettings.ArchiveFormat),
		Enabled:        types.BoolValue(backupSettings.Enabled),
		NotifiedUsers:  notifiedUsers,
	}
}

func (r *backupSettingsResource) updateAndHandleError(ctx context.Context, backupSettings youtrack.BackupSettings, diagnostics *diag.Diagnostics) (youtrack.BackupSettings, bool) {
	updated, err := r.client.UpdateBackupSettings(ctx, backupSettings)
	if err != nil {
		diagnostics.AddError(
			errUpdatingBackup,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "backup settings", err),
		)
		return youtrack.BackupSettings{}, false
	}

	return updated, true
}
