package provider

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testMailboxHost     = "imap.example.com"
	testMailboxLogin    = "support@example.com"
	testMailboxPassword = "secret" //nolint:gosec // test fixture, not a credential
)

func testIMAPMailboxModel() mailboxResourceModel {
	return mailboxResourceModel{
		ID:                        types.StringValue("254-1"),
		Protocol:                  types.StringValue(youtrack.MailProtocolIMAPS),
		Host:                      types.StringValue(testMailboxHost),
		Port:                      types.Int64Value(993),
		Login:                     types.StringValue(testMailboxLogin),
		Password:                  types.StringValue(testMailboxPassword),
		SocketTimeout:             types.Int64Unknown(),
		ConnectionTimeout:         types.Int64Value(60),
		ExchangeCompatibilityMode: types.BoolValue(true),
		MicrosoftMailboxAddress:   types.StringNull(),
		MicrosoftClientID:         types.StringNull(),
		MicrosoftTenantID:         types.StringNull(),
		MicrosoftClientSecret:     types.StringNull(),
	}
}

func TestMailboxToPayload(t *testing.T) {
	t.Parallel()

	model := testIMAPMailboxModel()
	payload := model.toPayload()

	helpers.AssertFieldEqual(t, "Protocol", payload.Protocol.ID, youtrack.MailProtocolIMAPS)
	helpers.AssertFieldEqual(t, "Host", *payload.Host, testMailboxHost)
	helpers.AssertFieldEqual(t, "Port", *payload.Port, 993)
	helpers.AssertFieldEqual(t, "Login", *payload.Login, testMailboxLogin)
	helpers.AssertFieldEqual(t, "Password", *payload.Password, testMailboxPassword)
	helpers.AssertFieldEqual(t, "ConnectionTimeOut", *payload.ConnectionTimeOut, 60)
	helpers.AssertFieldEqual(t, "ExchangeCompatibilityMode", *payload.ExchangeCompatibilityMode, true)

	// Unknown and null values are left out, so YouTrack keeps or defaults them.
	if payload.SocketTimeOut != nil {
		t.Errorf("SocketTimeOut: got %d, want nil", *payload.SocketTimeOut)
	}
	if payload.MicrosoftClientSecret != nil || payload.MicrosoftMailboxAddress != nil {
		t.Error("Microsoft fields of an IMAP mailbox must be left out")
	}
}

func TestMailboxFromAPIModelKeepsSecrets(t *testing.T) {
	t.Parallel()

	model := testIMAPMailboxModel()
	model.fromAPIModel(&youtrack.Mailbox{
		ID:                "254-1",
		Protocol:          &youtrack.MailboxIDRef{ID: youtrack.MailProtocolIMAPS},
		Host:              testMailboxHost,
		Port:              993,
		Login:             testMailboxLogin,
		SocketTimeOut:     30,
		ConnectionTimeOut: 60,
	})

	helpers.AssertFieldEqual(t, "Password", model.Password.ValueString(), testMailboxPassword)
	helpers.AssertFieldEqual(t, "SocketTimeout", model.SocketTimeout.ValueInt64(), int64(30))
	helpers.AssertFieldEqual(t, "ExchangeCompatibilityMode", model.ExchangeCompatibilityMode.ValueBool(), false)
	helpers.AssertFieldEqual(t, "MicrosoftMailboxAddress null", model.MicrosoftMailboxAddress.IsNull(), true)
}

func TestValidateMailboxConfig(t *testing.T) {
	t.Parallel()

	office365 := func() mailboxResourceModel {
		model := testIMAPMailboxModel()
		model.Protocol = types.StringValue(youtrack.MailProtocolOffice365)
		model.Host, model.Login, model.Password = types.StringNull(), types.StringNull(), types.StringNull()
		model.Port = types.Int64Null()
		model.MicrosoftMailboxAddress = types.StringValue("helpdesk@example.com")
		model.MicrosoftClientID = types.StringValue("client")
		model.MicrosoftTenantID = types.StringValue("tenant")
		model.MicrosoftClientSecret = types.StringValue("secret")
		return model
	}

	tests := []struct {
		name       string
		mutate     func() mailboxResourceModel
		wantErrors int
	}{
		{"valid IMAP", testIMAPMailboxModel, 0},
		{"valid OFFICE365", office365, 0},
		{"unknown password is accepted", func() mailboxResourceModel {
			m := testIMAPMailboxModel()
			m.Password = types.StringUnknown()
			return m
		}, 0},
		{"unknown protocol value", func() mailboxResourceModel {
			m := testIMAPMailboxModel()
			m.Protocol = types.StringValue("SMTP")
			return m
		}, 1},
		{"IMAP without host and password", func() mailboxResourceModel {
			m := testIMAPMailboxModel()
			m.Host, m.Password = types.StringNull(), types.StringNull()
			return m
		}, 2},
		{"IMAP with a Microsoft tenant", func() mailboxResourceModel {
			m := testIMAPMailboxModel()
			m.MicrosoftTenantID = types.StringValue("tenant")
			return m
		}, 1},
		{"OFFICE365 with a host and without a secret", func() mailboxResourceModel {
			m := office365()
			m.Host = types.StringValue(testMailboxHost)
			m.MicrosoftClientSecret = types.StringNull()
			return m
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var diagnostics diag.Diagnostics
			config := tt.mutate()
			validateMailboxConfig(&config, &diagnostics)
			helpers.AssertFieldEqual(t, "error count", diagnostics.ErrorsCount(), tt.wantErrors)
		})
	}
}
