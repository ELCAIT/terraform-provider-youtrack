package provider

import (
	"reflect"
	"testing"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	testMailRuleMaintainerID = "2-1"
	testMailRuleReporterID   = "2-7"
	testMailRuleProjectID    = "0-0"
)

func testMailRuleModel() mailRuleResourceModel {
	return mailRuleResourceModel{
		ID:                           types.StringValue("416-1"),
		MailboxID:                    types.StringValue("254-1"),
		ProjectID:                    types.StringValue(testMailRuleProjectID),
		MaintainerLogin:              types.StringValue("admin"),
		ReporterPolicy:               types.StringValue(youtrack.MailRulePolicySenderOrNewUser),
		DefaultReporterLogin:         types.StringNull(),
		Disabled:                     types.BoolValue(false),
		Folder:                       types.StringNull(),
		To:                           types.StringValue("support@example.com"),
		ReplyTo:                      types.StringNull(),
		SpamFilterMode:               types.StringValue(youtrack.MailRuleSpamFilterBlock),
		FilterAutoReplies:            types.BoolValue(true),
		NewIssueCommand:              types.StringValue("Type Bug"),
		NewCommentCommand:            types.StringNull(),
		DoNotDiscardBracketsContents: types.BoolValue(false),
		DeleteAfterFetch:             types.BoolValue(false),
		OriginalTimestamps:           types.BoolValue(true),
		PlainTextDescription:         types.BoolValue(false),
		FetchOldEmails:               types.BoolValue(false),
	}
}

func TestMailRuleToPayload(t *testing.T) {
	t.Parallel()

	model := testMailRuleModel()
	model.ReporterPolicy = types.StringValue(youtrack.MailRulePolicyAlwaysPredefined)
	model.Filters = []mailRuleFilterModel{{
		Scope:         types.StringValue(youtrack.MailRuleFilterScopeSubject),
		Patterns:      []types.String{types.StringValue("newsletter"), types.StringValue("unsubscribe")},
		AcceptMatched: types.BoolValue(false),
	}}

	payload := model.toPayload(testMailRuleMaintainerID, testMailRuleReporterID)

	helpers.AssertFieldEqual(t, "Mailbox", payload.Mailbox == nil, true)
	helpers.AssertFieldEqual(t, "Project", payload.Project.ID, testMailRuleProjectID)
	helpers.AssertFieldEqual(t, "Maintainer", payload.Maintainer.ID, testMailRuleMaintainerID)
	helpers.AssertFieldEqual(t, "Policy", payload.Policy.Policy, youtrack.MailRulePolicyAlwaysPredefined)
	helpers.AssertFieldEqual(t, "Policy user", payload.Policy.User.ID, testMailRuleReporterID)
	helpers.AssertFieldEqual(t, "SpamFilterMode", payload.SpamFilterMode.ID, youtrack.MailRuleSpamFilterBlock)
	helpers.AssertFieldEqual(t, "To", *payload.To, "support@example.com")
	// A null string is sent as "" so that removing it from the configuration clears it.
	helpers.AssertFieldEqual(t, "Folder", *payload.Folder, "")
	helpers.AssertFieldEqual(t, "OriginalTimestamps", *payload.OriginalTimestamps, true)

	wantFilters := []youtrack.MailRuleFilter{{
		Scope:         &youtrack.MailboxIDRef{ID: youtrack.MailRuleFilterScopeSubject},
		Patterns:      "newsletter\nunsubscribe",
		AcceptMatched: false,
	}}
	if !reflect.DeepEqual(*payload.Filters, wantFilters) {
		t.Errorf("Filters: got %+v, want %+v", *payload.Filters, wantFilters)
	}
}

func TestMailRuleToPayloadWithoutFiltersClearsThem(t *testing.T) {
	t.Parallel()

	model := testMailRuleModel()
	payload := model.toPayload(testMailRuleMaintainerID, "")

	if payload.Filters == nil || len(*payload.Filters) != 0 {
		t.Errorf("Filters: got %v, want a pointer to an empty slice", payload.Filters)
	}
	if payload.Policy.User != nil {
		t.Errorf("Policy user: got %+v, want nil", payload.Policy.User)
	}
}

func TestMailRuleFromAPIModel(t *testing.T) {
	t.Parallel()

	var model mailRuleResourceModel
	model.fromAPIModel(&youtrack.MailRule{
		ID:         "416-1",
		Mailbox:    &youtrack.MailboxIDRef{ID: "254-1"},
		Project:    &youtrack.MailRuleProjectRef{ID: testMailRuleProjectID, ShortName: "DEMO"},
		Maintainer: &youtrack.UserRef{ID: testMailRuleMaintainerID, Login: "admin"},
		Policy: &youtrack.MailRulePolicy{
			Policy: youtrack.MailRulePolicySenderOrDefault,
			User:   &youtrack.UserRef{ID: testMailRuleReporterID, Login: "reporter"},
		},
		SpamFilterMode: &youtrack.MailboxIDRef{ID: youtrack.MailRuleSpamFilterMark},
		Filters: []youtrack.MailRuleFilter{{
			Scope:         &youtrack.MailboxIDRef{ID: youtrack.MailRuleFilterScopeBody},
			Patterns:      "a\r\nb",
			AcceptMatched: true,
		}},
		Folder:            "",
		FilterAutoReplies: true,
	})

	helpers.AssertFieldEqual(t, "MailboxID", model.MailboxID.ValueString(), "254-1")
	helpers.AssertFieldEqual(t, "ProjectID", model.ProjectID.ValueString(), testMailRuleProjectID)
	helpers.AssertFieldEqual(t, "MaintainerLogin", model.MaintainerLogin.ValueString(), "admin")
	helpers.AssertFieldEqual(t, "ReporterPolicy", model.ReporterPolicy.ValueString(), youtrack.MailRulePolicySenderOrDefault)
	helpers.AssertFieldEqual(t, "DefaultReporterLogin", model.DefaultReporterLogin.ValueString(), "reporter")
	helpers.AssertFieldEqual(t, "SpamFilterMode", model.SpamFilterMode.ValueString(), youtrack.MailRuleSpamFilterMark)
	helpers.AssertFieldEqual(t, "Folder null", model.Folder.IsNull(), true)
	helpers.AssertFieldEqual(t, "FilterAutoReplies", model.FilterAutoReplies.ValueBool(), true)

	wantFilters := []mailRuleFilterModel{{
		Scope:         types.StringValue(youtrack.MailRuleFilterScopeBody),
		Patterns:      []types.String{types.StringValue("a"), types.StringValue("b")},
		AcceptMatched: types.BoolValue(true),
	}}
	if !reflect.DeepEqual(model.Filters, wantFilters) {
		t.Errorf("Filters: got %+v, want %+v", model.Filters, wantFilters)
	}
}

func TestMailRuleFromAPIModelKeepsEmptyFilterList(t *testing.T) {
	t.Parallel()

	model := mailRuleResourceModel{Filters: []mailRuleFilterModel{}}
	model.fromAPIFilters(nil)
	if model.Filters == nil {
		t.Error("an empty filter list in the configuration must not become null")
	}

	model = mailRuleResourceModel{}
	model.fromAPIFilters(nil)
	if model.Filters != nil {
		t.Errorf("Filters: got %v, want nil", model.Filters)
	}
}

func TestValidateMailRuleConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mutate     func(*mailRuleResourceModel)
		wantErrors int
	}{
		{"valid", func(*mailRuleResourceModel) {}, 0},
		{"null policy is the default", func(m *mailRuleResourceModel) { m.ReporterPolicy = types.StringNull() }, 0},
		{"predefined reporter set", func(m *mailRuleResourceModel) {
			m.ReporterPolicy = types.StringValue(youtrack.MailRulePolicyAlwaysPredefined)
			m.DefaultReporterLogin = types.StringValue("reporter")
		}, 0},
		{"predefined reporter missing", func(m *mailRuleResourceModel) {
			m.ReporterPolicy = types.StringValue(youtrack.MailRulePolicySenderOrDefault)
		}, 1},
		{"predefined reporter not used", func(m *mailRuleResourceModel) {
			m.DefaultReporterLogin = types.StringValue("reporter")
		}, 1},
		{"invalid enumerated values", func(m *mailRuleResourceModel) {
			m.SpamFilterMode = types.StringValue("DROP")
			m.Filters = []mailRuleFilterModel{{Scope: types.StringValue("HEADER"), AcceptMatched: types.BoolValue(true)}}
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := testMailRuleModel()
			tt.mutate(&config)
			var diagnostics diag.Diagnostics
			validateMailRuleConfig(&config, &diagnostics)
			helpers.AssertFieldEqual(t, "error count", diagnostics.ErrorsCount(), tt.wantErrors)
		})
	}
}
