// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"slices"
	"strings"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// Error message titles
	errCreatingMailbox      = "Error creating mailbox"
	errReadingMailbox       = "Error reading mailbox"
	errUpdatingMailbox      = "Error updating mailbox"
	errDeletingMailbox      = "Error deleting mailbox"
	errMissingMailboxID     = "Missing mailbox ID"
	errInvalidMailboxConfig = "Invalid mailbox configuration"

	// Error message details
	errMailboxIDRequired = "Mailbox ID is required to read the mailbox"

	// Attribute names
	attrID                               = "id"
	attrMailboxProtocol                  = "protocol"
	attrMailboxHost                      = "host"
	attrMailboxPort                      = "port"
	attrMailboxLogin                     = "login"
	attrMailboxPassword                  = "password"
	attrMailboxSocketTimeout             = "socket_timeout"
	attrMailboxConnectionTimeout         = "connection_timeout"
	attrMailboxExchangeCompatibilityMode = "exchange_compatibility_mode"
	attrMailboxMicrosoftAddress          = "microsoft_mailbox_address"
	attrMailboxMicrosoftClientID         = "microsoft_client_id"
	attrMailboxMicrosoftTenantID         = "microsoft_tenant_id"
	attrMailboxMicrosoftClientSecret     = "microsoft_client_secret" //nolint:gosec // attribute name, not a credential
)

// mailboxProtocols are the protocols YouTrack accepts for a mailbox.
var mailboxProtocols = []string{
	youtrack.MailProtocolIMAP,
	youtrack.MailProtocolIMAPS,
	youtrack.MailProtocolOffice365,
	youtrack.MailProtocolPOP,
	youtrack.MailProtocolPOPS,
}

// mailboxResourceModel maps the Terraform resource schema data.
type mailboxResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Protocol                  types.String `tfsdk:"protocol"`
	Host                      types.String `tfsdk:"host"`
	Port                      types.Int64  `tfsdk:"port"`
	Login                     types.String `tfsdk:"login"`
	Password                  types.String `tfsdk:"password"`
	SocketTimeout             types.Int64  `tfsdk:"socket_timeout"`
	ConnectionTimeout         types.Int64  `tfsdk:"connection_timeout"`
	ExchangeCompatibilityMode types.Bool   `tfsdk:"exchange_compatibility_mode"`
	MicrosoftMailboxAddress   types.String `tfsdk:"microsoft_mailbox_address"`
	MicrosoftClientID         types.String `tfsdk:"microsoft_client_id"`
	MicrosoftTenantID         types.String `tfsdk:"microsoft_tenant_id"`
	MicrosoftClientSecret     types.String `tfsdk:"microsoft_client_secret"`
}

// mailboxAttributeValue is one string-like attribute of a mailbox, as
// validateMailboxConfig checks it.
type mailboxAttributeValue struct {
	name  string
	value attrValue
}

// attrValue is the part of a framework value validateMailboxConfig needs.
type attrValue interface {
	IsNull() bool
	IsUnknown() bool
}

// serverAttributes are the attributes of an IMAP(S)/POP(S) mailbox.
func (m *mailboxResourceModel) serverAttributes() []mailboxAttributeValue {
	return []mailboxAttributeValue{
		{attrMailboxHost, m.Host},
		{attrMailboxPort, m.Port},
		{attrMailboxLogin, m.Login},
		{attrMailboxPassword, m.Password},
	}
}

// microsoftAttributes are the attributes of an OFFICE365 mailbox.
func (m *mailboxResourceModel) microsoftAttributes() []mailboxAttributeValue {
	return []mailboxAttributeValue{
		{attrMailboxMicrosoftAddress, m.MicrosoftMailboxAddress},
		{attrMailboxMicrosoftClientID, m.MicrosoftClientID},
		{attrMailboxMicrosoftTenantID, m.MicrosoftTenantID},
		{attrMailboxMicrosoftClientSecret, m.MicrosoftClientSecret},
	}
}

// validateMailboxConfig requires the attributes the protocol needs, and rejects
// those of the other kind of mailbox, which YouTrack would keep and report back.
func validateMailboxConfig(config *mailboxResourceModel, diagnostics *diag.Diagnostics) {
	if config.Protocol.IsNull() || config.Protocol.IsUnknown() {
		return
	}

	protocol := config.Protocol.ValueString()
	if !slices.Contains(mailboxProtocols, protocol) {
		diagnostics.AddAttributeError(path.Root(attrMailboxProtocol), errInvalidMailboxConfig,
			fmt.Sprintf("protocol must be one of %s, got %q.", strings.Join(mailboxProtocols, ", "), protocol))
		return
	}

	required, forbidden := config.serverAttributes(), config.microsoftAttributes()
	if protocol == youtrack.MailProtocolOffice365 {
		required, forbidden = forbidden, required
	}

	for _, attribute := range required {
		if attribute.value.IsNull() {
			diagnostics.AddAttributeError(path.Root(attribute.name), errInvalidMailboxConfig,
				fmt.Sprintf("%s is required for a %s mailbox.", attribute.name, protocol))
		}
	}
	for _, attribute := range forbidden {
		if !attribute.value.IsNull() && !attribute.value.IsUnknown() {
			diagnostics.AddAttributeError(path.Root(attribute.name), errInvalidMailboxConfig,
				fmt.Sprintf("%s is not used by a %s mailbox.", attribute.name, protocol))
		}
	}
}

// optionalString returns a pointer to the value, or nil when it is null or
// unknown, so the payload leaves it out.
func optionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	s := value.ValueString()
	return &s
}

// optionalInt returns a pointer to the value, or nil when it is null or
// unknown, so the payload leaves it out.
func optionalInt(value types.Int64) *int {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	i := int(value.ValueInt64())
	return &i
}

// optionalBool returns a pointer to the value, or nil when it is null or
// unknown, so the payload leaves it out.
func optionalBool(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	b := value.ValueBool()
	return &b
}

// toPayload converts the Terraform model to the request body of a create or an
// update. The password and the Microsoft client secret are sent on every update:
// YouTrack refuses to change the host, port, login or tenant without them.
func (m *mailboxResourceModel) toPayload() youtrack.MailboxPayload {
	return youtrack.MailboxPayload{
		Protocol:                  &youtrack.MailboxIDRef{ID: m.Protocol.ValueString()},
		Host:                      optionalString(m.Host),
		Port:                      optionalInt(m.Port),
		Login:                     optionalString(m.Login),
		Password:                  optionalString(m.Password),
		SocketTimeOut:             optionalInt(m.SocketTimeout),
		ConnectionTimeOut:         optionalInt(m.ConnectionTimeout),
		ExchangeCompatibilityMode: optionalBool(m.ExchangeCompatibilityMode),
		MicrosoftMailboxAddress:   optionalString(m.MicrosoftMailboxAddress),
		MicrosoftClientID:         optionalString(m.MicrosoftClientID),
		MicrosoftTenantID:         optionalString(m.MicrosoftTenantID),
		MicrosoftClientSecret:     optionalString(m.MicrosoftClientSecret),
	}
}

// fromAPIModel populates the Terraform model from the API model. The password
// and the Microsoft client secret are write-only, so they keep their prior value.
func (m *mailboxResourceModel) fromAPIModel(api *youtrack.Mailbox) {
	m.ID = types.StringValue(api.ID)
	if api.Protocol != nil {
		m.Protocol = types.StringValue(api.Protocol.ID)
	}
	m.Host = helpers.StringOrNull(api.Host)
	m.Port = types.Int64Value(int64(api.Port))
	m.Login = helpers.StringOrNull(api.Login)
	m.SocketTimeout = types.Int64Value(int64(api.SocketTimeOut))
	m.ConnectionTimeout = types.Int64Value(int64(api.ConnectionTimeOut))
	m.ExchangeCompatibilityMode = types.BoolValue(api.ExchangeCompatibilityMode)
	m.MicrosoftMailboxAddress = helpers.StringOrNull(api.MicrosoftMailboxAddress)
	m.MicrosoftClientID = helpers.StringOrNull(api.MicrosoftClientID)
	m.MicrosoftTenantID = helpers.StringOrNull(api.MicrosoftTenantID)
}
