// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"context"
	"fmt"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	errReadingMailboxIntegrationSettings  = "Error reading mailbox integration settings"
	errUpdatingMailboxIntegrationSettings = "Error updating mailbox integration settings"

	// mailboxIntegrationSettingsID is the ID of the singleton in Terraform state.
	mailboxIntegrationSettingsID = "global"

	// defaultMailboxCronExpression is the schedule of a fresh YouTrack instance:
	// every two minutes.
	defaultMailboxCronExpression = "0 0/2 * * * ?"
)

// readCron reads the mailbox poll schedule.
func (r *mailboxIntegrationSettingsResource) readCron(ctx context.Context, diagnostics *diag.Diagnostics) (*youtrack.MailboxCronSettings, bool) {
	settings, err := r.client.GetMailboxCronSettings(ctx)
	if err != nil {
		diagnostics.AddError(errReadingMailboxIntegrationSettings,
			fmt.Sprintf("Could not read mailbox integration settings: %v", err))
		return nil, false
	}

	return settings, true
}

// updateCron sets the mailbox poll schedule.
func (r *mailboxIntegrationSettingsResource) updateCron(ctx context.Context, cronExpression string, diagnostics *diag.Diagnostics) (*youtrack.MailboxCronSettings, bool) {
	settings, err := r.client.UpdateMailboxCronSettings(ctx, cronExpression)
	if err != nil {
		diagnostics.AddError(errUpdatingMailboxIntegrationSettings,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "mailbox integration settings", err))
		return nil, false
	}

	return settings, true
}

// fromAPIModel populates the Terraform model from the API model.
func (m *mailboxIntegrationSettingsResourceModel) fromAPIModel(settings *youtrack.MailboxCronSettings) {
	m.ID = types.StringValue(mailboxIntegrationSettingsID)
	m.CronExpression = types.StringValue(settings.CronExpression)
}
