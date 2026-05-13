// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package settings

import (
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	host         = "smtp.example.com"
	loginUser    = "user@example.com"
	fromEmail    = "noreply@example.com"
	replyToEmail = "support@example.com"

	// Port numbers
	smtpPortTLS      = 587
	smtpPortStandard = 25
)

// mailServerTestConfig holds configuration for creating test mail server instances.
type mailServerTestConfig struct {
	enabled   bool
	protocol  string
	host      string
	login     string
	from      string
	replyTo   string
	port      int
	anonymous bool
}

// Shared test configurations to avoid duplication
var mailServerTestConfigs = []struct {
	name   string
	config mailServerTestConfig
}{
	{
		name: "converts all fields correctly",
		config: mailServerTestConfig{
			enabled:   true,
			protocol:  "SMTP",
			host:      host,
			login:     loginUser,
			from:      fromEmail,
			replyTo:   replyToEmail,
			port:      smtpPortTLS,
			anonymous: false,
		},
	},
	{
		name: "handles disabled mail server",
		config: mailServerTestConfig{
			enabled:   false,
			protocol:  "",
			host:      "",
			login:     "",
			from:      "",
			replyTo:   "",
			port:      smtpPortStandard,
			anonymous: true,
		},
	},
}

// Helper functions for test data creation
func makeMailModel(cfg mailServerTestConfig) mailServerResourceModel {
	return mailServerResourceModel{
		IsEnabled:    types.BoolValue(cfg.enabled),
		MailProtocol: types.StringValue(cfg.protocol),
		Host:         types.StringValue(cfg.host),
		Port:         types.Int64Value(int64(cfg.port)),
		Anonymous:    types.BoolValue(cfg.anonymous),
		Login:        types.StringValue(cfg.login),
		From:         types.StringValue(cfg.from),
		ReplyTo:      types.StringValue(cfg.replyTo),
	}
}

func makeMailServer(cfg mailServerTestConfig) youtrack.MailServer {
	return youtrack.MailServer{
		IsEnabled:    cfg.enabled,
		MailProtocol: cfg.protocol,
		Host:         cfg.host,
		Port:         cfg.port,
		Anonymous:    cfg.anonymous,
		Login:        cfg.login,
		From:         cfg.from,
		ReplyTo:      cfg.replyTo,
	}
}

// assertMailServerFields verifies all fields match between two MailServer instances.
func assertMailServerFields(t *testing.T, got, want youtrack.MailServer) {
	t.Helper()
	helpers.AssertFieldEqual(t, "IsEnabled", got.IsEnabled, want.IsEnabled)
	helpers.AssertFieldEqual(t, "MailProtocol", got.MailProtocol, want.MailProtocol)
	helpers.AssertFieldEqual(t, "Host", got.Host, want.Host)
	helpers.AssertFieldEqual(t, "Port", got.Port, want.Port)
	helpers.AssertFieldEqual(t, "Anonymous", got.Anonymous, want.Anonymous)
	helpers.AssertFieldEqual(t, "Login", got.Login, want.Login)
	helpers.AssertFieldEqual(t, "From", got.From, want.From)
	helpers.AssertFieldEqual(t, "ReplyTo", got.ReplyTo, want.ReplyTo)
}

// assertMailServerModelFields verifies all fields match between model and config.
func assertMailServerModelFields(t *testing.T, got mailServerResourceModel, cfg mailServerTestConfig) {
	t.Helper()
	helpers.AssertFieldEqual(t, "IsEnabled", got.IsEnabled.ValueBool(), cfg.enabled)
	helpers.AssertFieldEqual(t, "MailProtocol", got.MailProtocol.ValueString(), cfg.protocol)
	helpers.AssertFieldEqual(t, "Host", got.Host.ValueString(), cfg.host)
	helpers.AssertFieldEqual(t, "Port", got.Port.ValueInt64(), int64(cfg.port))
	helpers.AssertFieldEqual(t, "Anonymous", got.Anonymous.ValueBool(), cfg.anonymous)
	helpers.AssertFieldEqual(t, "Login", got.Login.ValueString(), cfg.login)
	helpers.AssertFieldEqual(t, "From", got.From.ValueString(), cfg.from)
	helpers.AssertFieldEqual(t, "ReplyTo", got.ReplyTo.ValueString(), cfg.replyTo)
}

func TestConvertModelToMailServer(t *testing.T) {
	for _, tt := range mailServerTestConfigs {
		t.Run(tt.name, func(t *testing.T) {
			model := makeMailModel(tt.config)
			want := makeMailServer(tt.config)
			got := convertModelToMailServer(model)

			assertMailServerFields(t, got, want)
		})
	}
}

func TestConvertMailServerToModel(t *testing.T) {
	for _, tt := range mailServerTestConfigs {
		t.Run(tt.name, func(t *testing.T) {
			ms := makeMailServer(tt.config)
			got := convertMailServerToModel(ms)

			assertMailServerModelFields(t, got, tt.config)
		})
	}
}

func TestUpdateModelFromResponse(t *testing.T) {
	t.Run("updates model when response has data", func(t *testing.T) {
		mailServer := makeMailServer(mailServerTestConfigs[0].config)
		model := convertMailServerToModel(mailServer)

		helpers.AssertFieldEqual(t, "Host", model.Host.ValueString(), mailServer.Host)
	})

	t.Run("doesn't update model when response is empty", func(t *testing.T) {
		mailServer := youtrack.MailServer{}
		model := convertMailServerToModel(mailServer)

		if !model.Host.IsNull() && model.Host.ValueString() != "" {
			t.Error("Model should not be updated when response is empty")
		}
	})
}

func TestUpdateModelWithTimestamp(t *testing.T) {
	mailServer := makeMailServer(mailServerTestConfigs[0].config)
	resourceModel := mailServerResourceModel{}
	updateMailServerModelWithTimestamp(mailServer, &resourceModel)

	if resourceModel.LastUpdated.IsNull() {
		t.Error("LastUpdated should be set")
	}

	helpers.AssertFieldEqual(t, "Host", resourceModel.Host.ValueString(), mailServer.Host)
}
