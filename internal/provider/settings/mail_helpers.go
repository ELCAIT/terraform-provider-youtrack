package settings

import (
	"context"
	"fmt"
	"time"

	youtrack "github.com/elcait/youtrack-api-client/client"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	errUnableToReadMailServer = "Unable to Read YouTrack Mail Server Settings"
	errUpdatingMailServer     = "Error updating mail server settings"
)

// getMailServerAndHandleError fetches the mail server via API and handles errors.
func (r *mailServerResource) getMailServerAndHandleError(ctx context.Context, diagnostics *diag.Diagnostics) (youtrack.MailServer, bool) {
	mailServer, err := r.client.GetMailServer(ctx)
	if err != nil {
		diagnostics.AddError(
			errUnableToReadMailServer,
			err.Error(),
		)
		return youtrack.MailServer{}, false
	}
	return mailServer, true
}

// updateMailServerModelWithTimestamp updates the model from response and sets timestamp.
func updateMailServerModelWithTimestamp(mailServer youtrack.MailServer, resourceModel *mailServerResourceModel) {
	*resourceModel = convertMailServerToModel(mailServer)
	resourceModel.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
}

// convertModelToMailServer converts a mailServerResourceModel to youtrack.MailServer.
func convertModelToMailServer(model mailServerResourceModel) youtrack.MailServer {
	return youtrack.MailServer{
		IsEnabled:    model.IsEnabled.ValueBool(),
		MailProtocol: model.MailProtocol.ValueString(),
		Host:         model.Host.ValueString(),
		Port:         int(model.Port.ValueInt64()),
		Anonymous:    model.Anonymous.ValueBool(),
		Login:        model.Login.ValueString(),
		From:         model.From.ValueString(),
		ReplyTo:      model.ReplyTo.ValueString(),
	}
}

// convertMailServerToModel converts a youtrack.MailServer to mailServerResourceModel.
func convertMailServerToModel(ms youtrack.MailServer) mailServerResourceModel {
	return mailServerResourceModel{
		ID:           types.StringValue("global"),
		IsEnabled:    types.BoolValue(ms.IsEnabled),
		MailProtocol: types.StringValue(ms.MailProtocol),
		Host:         types.StringValue(ms.Host),
		Port:         types.Int64Value(int64(ms.Port)),
		Anonymous:    types.BoolValue(ms.Anonymous),
		Login:        types.StringValue(ms.Login),
		From:         types.StringValue(ms.From),
		ReplyTo:      types.StringValue(ms.ReplyTo),
	}
}

// updateMailServerAndHandleError updates the mail server via API and handles errors.
func (r *mailServerResource) updateAndHandleError(ctx context.Context, ms youtrack.MailServer, diagnostics *diag.Diagnostics) (youtrack.MailServer, bool) {
	mailServer, err := r.client.UpdateMailServer(ctx, ms)
	if err != nil {
		diagnostics.AddError(
			errUpdatingMailServer,
			fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "mail server", err),
		)
		return youtrack.MailServer{}, false
	}
	return mailServer, true
}

// Validation helper functions to reduce cognitive complexity

// validatePortField validates that a port number is within the valid range.
func validatePortField(value types.Int64, diagnostics *diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return
	}

	port := value.ValueInt64()
	if port < minValidPort || port > maxValidPort {
		diagnostics.AddAttributeError(
			path.Root("port"),
			helpers.ErrInvalidPortNumber,
			fmt.Sprintf("Port must be between %d and %d, got: %d", minValidPort, maxValidPort, port),
		)
	}
}

// validateRequiredFieldWhenEnabled validates that a required field is set when the mail server is enabled.
func validateRequiredFieldWhenEnabled(isEnabled types.Bool, field types.String, fieldName, fieldDescription string, diagnostics *diag.Diagnostics) {
	if isEnabled.IsNull() || isEnabled.IsUnknown() || !isEnabled.ValueBool() {
		return
	}

	if field.IsNull() || field.ValueString() == "" {
		diagnostics.AddAttributeError(
			path.Root(fieldName),
			"Missing Required Field",
			fmt.Sprintf("The %s field is required when is_enabled is true.", fieldDescription),
		)
	}
}
