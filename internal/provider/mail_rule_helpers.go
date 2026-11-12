// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
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
	errCreatingMailRule      = "Error creating mail rule"
	errReadingMailRule       = "Error reading mail rule"
	errUpdatingMailRule      = "Error updating mail rule"
	errDeletingMailRule      = "Error deleting mail rule"
	errMissingMailRuleID     = "Missing mail rule ID"
	errInvalidMailRuleConfig = "Invalid mail rule configuration"

	// Error message details
	errMailRuleIDRequired = "Mail rule ID is required to read the mail rule"
	errMailRuleUserFmt    = "Could not find user with login %q: %v"
	errMailRuleOneOfFmt   = "%s must be one of %s, got %q."

	// Attribute names
	attrMailRuleMailboxID            = "mailbox_id"
	attrMailRuleProjectID            = "project_id"
	attrMailRuleMaintainerLogin      = "maintainer_login"
	attrMailRuleReporterPolicy       = "reporter_policy"
	attrMailRuleDefaultReporterLogin = "default_reporter_login"
	attrMailRuleDisabled             = "disabled"
	attrMailRuleFolder               = "folder"
	attrMailRuleTo                   = "to"
	attrMailRuleReplyTo              = "reply_to"
	attrMailRuleSpamFilterMode       = "spam_filter_mode"
	attrMailRuleFilters              = "filters"
	attrMailRuleFilterScope          = "scope"
	attrMailRuleFilterPatterns       = "patterns"
	attrMailRuleFilterAcceptMatched  = "accept_matched"
	attrMailRuleFilterAutoReplies    = "filter_auto_replies"
	attrMailRuleNewIssueCommand      = "new_issue_command"
	attrMailRuleNewCommentCommand    = "new_comment_command"
	attrMailRuleKeepBrackets         = "do_not_discard_brackets_contents"
	attrMailRuleDeleteAfterFetch     = "delete_after_fetch"
	attrMailRuleOriginalTimestamps   = "original_timestamps"
	attrMailRulePlainTextDescription = "plain_text_description"
	attrMailRuleFetchOldEmails       = "fetch_old_emails"

	// mailRulePatternSeparator separates the patterns of a filter in the API.
	mailRulePatternSeparator = "\n"
)

var (
	// mailRuleReporterPolicies are the reporter policies YouTrack accepts.
	mailRuleReporterPolicies = []string{
		youtrack.MailRulePolicySenderOrNewUser,
		youtrack.MailRulePolicySenderOrDefault,
		youtrack.MailRulePolicyAlwaysPredefined,
	}

	// mailRuleSpamFilterModes are the spam filter modes YouTrack accepts.
	mailRuleSpamFilterModes = []string{
		youtrack.MailRuleSpamFilterNone,
		youtrack.MailRuleSpamFilterMark,
		youtrack.MailRuleSpamFilterBlock,
	}

	// mailRuleFilterScopes are the filter scopes YouTrack accepts.
	mailRuleFilterScopes = []string{
		youtrack.MailRuleFilterScopeSubject,
		youtrack.MailRuleFilterScopeBody,
		youtrack.MailRuleFilterScopeSubjectAndBody,
	}
)

// mailRuleResourceModel maps the Terraform resource schema data.
type mailRuleResourceModel struct {
	ID                           types.String          `tfsdk:"id"`
	MailboxID                    types.String          `tfsdk:"mailbox_id"`
	ProjectID                    types.String          `tfsdk:"project_id"`
	MaintainerLogin              types.String          `tfsdk:"maintainer_login"`
	ReporterPolicy               types.String          `tfsdk:"reporter_policy"`
	DefaultReporterLogin         types.String          `tfsdk:"default_reporter_login"`
	Disabled                     types.Bool            `tfsdk:"disabled"`
	Folder                       types.String          `tfsdk:"folder"`
	To                           types.String          `tfsdk:"to"`
	ReplyTo                      types.String          `tfsdk:"reply_to"`
	SpamFilterMode               types.String          `tfsdk:"spam_filter_mode"`
	Filters                      []mailRuleFilterModel `tfsdk:"filters"`
	FilterAutoReplies            types.Bool            `tfsdk:"filter_auto_replies"`
	NewIssueCommand              types.String          `tfsdk:"new_issue_command"`
	NewCommentCommand            types.String          `tfsdk:"new_comment_command"`
	DoNotDiscardBracketsContents types.Bool            `tfsdk:"do_not_discard_brackets_contents"`
	DeleteAfterFetch             types.Bool            `tfsdk:"delete_after_fetch"`
	OriginalTimestamps           types.Bool            `tfsdk:"original_timestamps"`
	PlainTextDescription         types.Bool            `tfsdk:"plain_text_description"`
	FetchOldEmails               types.Bool            `tfsdk:"fetch_old_emails"`
}

// mailRuleFilterModel maps one element of the filters attribute.
type mailRuleFilterModel struct {
	Scope         types.String   `tfsdk:"scope"`
	Patterns      []types.String `tfsdk:"patterns"`
	AcceptMatched types.Bool     `tfsdk:"accept_matched"`
}

// validateOneOf reports an attribute error when a known value is not one of allowed.
func validateOneOf(value types.String, attributePath path.Path, allowed []string, diagnostics *diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() || slices.Contains(allowed, value.ValueString()) {
		return
	}
	diagnostics.AddAttributeError(attributePath, errInvalidMailRuleConfig,
		fmt.Sprintf(errMailRuleOneOfFmt, attributePath, strings.Join(allowed, ", "), value.ValueString()))
}

// validateMailRuleConfig checks the enumerated values, and that a predefined
// reporter is set exactly when the reporter policy uses one.
func validateMailRuleConfig(config *mailRuleResourceModel, diagnostics *diag.Diagnostics) {
	validateOneOf(config.ReporterPolicy, path.Root(attrMailRuleReporterPolicy), mailRuleReporterPolicies, diagnostics)
	validateOneOf(config.SpamFilterMode, path.Root(attrMailRuleSpamFilterMode), mailRuleSpamFilterModes, diagnostics)
	for i, filter := range config.Filters {
		scopePath := path.Root(attrMailRuleFilters).AtListIndex(i).AtName(attrMailRuleFilterScope)
		validateOneOf(filter.Scope, scopePath, mailRuleFilterScopes, diagnostics)
	}

	if config.ReporterPolicy.IsUnknown() || config.DefaultReporterLogin.IsUnknown() {
		return
	}

	// A null policy is the default, SENDER_OR_NEW_USER.
	policy := config.ReporterPolicy.ValueString()
	needsReporter := policy == youtrack.MailRulePolicySenderOrDefault || policy == youtrack.MailRulePolicyAlwaysPredefined
	hasReporter := !config.DefaultReporterLogin.IsNull()
	reporterPath := path.Root(attrMailRuleDefaultReporterLogin)

	switch {
	case needsReporter && !hasReporter:
		diagnostics.AddAttributeError(reporterPath, errInvalidMailRuleConfig,
			fmt.Sprintf("%s is required when %s is %s.", attrMailRuleDefaultReporterLogin, attrMailRuleReporterPolicy, policy))
	case !needsReporter && hasReporter:
		diagnostics.AddAttributeError(reporterPath, errInvalidMailRuleConfig,
			fmt.Sprintf("%s is only used when %s is %s or %s.", attrMailRuleDefaultReporterLogin, attrMailRuleReporterPolicy,
				youtrack.MailRulePolicySenderOrDefault, youtrack.MailRulePolicyAlwaysPredefined))
	}
}

// buildPayload resolves the maintainer and the predefined reporter, then
// converts the plan to the request body of a create or an update. errSummary
// titles the diagnostic of a login that does not resolve.
func (r *mailRuleResource) buildPayload(ctx context.Context, plan *mailRuleResourceModel, errSummary string, diagnostics *diag.Diagnostics) (youtrack.MailRulePayload, bool) {
	maintainerID, ok := r.resolveUserID(ctx, plan.MaintainerLogin, errSummary, diagnostics)
	if !ok {
		return youtrack.MailRulePayload{}, false
	}

	reporterID := ""
	if !plan.DefaultReporterLogin.IsNull() {
		if reporterID, ok = r.resolveUserID(ctx, plan.DefaultReporterLogin, errSummary, diagnostics); !ok {
			return youtrack.MailRulePayload{}, false
		}
	}

	return plan.toPayload(maintainerID, reporterID), true
}

// resolveUserID returns the ID of the user with the given login.
func (r *mailRuleResource) resolveUserID(ctx context.Context, login types.String, errSummary string, diagnostics *diag.Diagnostics) (string, bool) {
	user, err := r.client.GetUserByLogin(ctx, login.ValueString())
	if err != nil {
		diagnostics.AddError(errSummary, fmt.Sprintf(errMailRuleUserFmt, login.ValueString(), err))
		return "", false
	}

	return user.Id, true
}

// stringOrClear returns a pointer to the value, or to "" when it is null: an
// empty string clears the field in YouTrack, while a field left out of an
// update keeps its value.
func stringOrClear(value types.String) *string {
	s := value.ValueString()
	return &s
}

// toPayload converts the Terraform model to the request body of a create or an
// update. Every field is sent, so an attribute removed from the configuration
// is cleared rather than left as it was. The mailbox is left out: it cannot
// change, and Create sets it.
func (m *mailRuleResourceModel) toPayload(maintainerID, reporterID string) youtrack.MailRulePayload {
	policy := &youtrack.MailRulePolicy{Policy: m.ReporterPolicy.ValueString()}
	if reporterID != "" {
		policy.User = &youtrack.UserRef{ID: reporterID}
	}

	filters := make([]youtrack.MailRuleFilter, 0, len(m.Filters))
	for _, filter := range m.Filters {
		patterns := make([]string, 0, len(filter.Patterns))
		for _, pattern := range filter.Patterns {
			patterns = append(patterns, pattern.ValueString())
		}
		filters = append(filters, youtrack.MailRuleFilter{
			Scope:         &youtrack.MailboxIDRef{ID: filter.Scope.ValueString()},
			Patterns:      strings.Join(patterns, mailRulePatternSeparator),
			AcceptMatched: filter.AcceptMatched.ValueBool(),
		})
	}

	return youtrack.MailRulePayload{
		Project:                      &youtrack.MailboxIDRef{ID: m.ProjectID.ValueString()},
		Maintainer:                   &youtrack.UserRef{ID: maintainerID},
		Disabled:                     optionalBool(m.Disabled),
		Folder:                       stringOrClear(m.Folder),
		To:                           stringOrClear(m.To),
		ReplyTo:                      stringOrClear(m.ReplyTo),
		Policy:                       policy,
		SpamFilterMode:               &youtrack.MailboxIDRef{ID: m.SpamFilterMode.ValueString()},
		Filters:                      &filters,
		FilterAutoReplies:            optionalBool(m.FilterAutoReplies),
		NewIssueCommand:              stringOrClear(m.NewIssueCommand),
		NewCommentCommand:            stringOrClear(m.NewCommentCommand),
		DoNotDiscardBracketsContents: optionalBool(m.DoNotDiscardBracketsContents),
		DeleteAfterFetch:             optionalBool(m.DeleteAfterFetch),
		OriginalTimestamps:           optionalBool(m.OriginalTimestamps),
		PlainTextDescription:         optionalBool(m.PlainTextDescription),
		FetchOldEmails:               optionalBool(m.FetchOldEmails),
	}
}

// fromAPIModel populates the Terraform model from the API model.
func (m *mailRuleResourceModel) fromAPIModel(api *youtrack.MailRule) {
	m.ID = types.StringValue(api.ID)
	if api.Mailbox != nil {
		m.MailboxID = types.StringValue(api.Mailbox.ID)
	}
	if api.Project != nil {
		m.ProjectID = types.StringValue(api.Project.ID)
	}
	if api.Maintainer != nil {
		m.MaintainerLogin = types.StringValue(api.Maintainer.Login)
	}
	m.fromAPIPolicy(api.Policy)
	if api.SpamFilterMode != nil {
		m.SpamFilterMode = types.StringValue(api.SpamFilterMode.ID)
	}
	m.fromAPIFilters(api.Filters)

	m.Disabled = types.BoolValue(api.Disabled)
	m.Folder = helpers.StringOrNull(api.Folder)
	m.To = helpers.StringOrNull(api.To)
	m.ReplyTo = helpers.StringOrNull(api.ReplyTo)
	m.FilterAutoReplies = types.BoolValue(api.FilterAutoReplies)
	m.NewIssueCommand = helpers.StringOrNull(api.NewIssueCommand)
	m.NewCommentCommand = helpers.StringOrNull(api.NewCommentCommand)
	m.DoNotDiscardBracketsContents = types.BoolValue(api.DoNotDiscardBracketsContents)
	m.DeleteAfterFetch = types.BoolValue(api.DeleteAfterFetch)
	m.OriginalTimestamps = types.BoolValue(api.OriginalTimestamps)
	m.PlainTextDescription = types.BoolValue(api.PlainTextDescription)
	m.FetchOldEmails = types.BoolValue(api.FetchOldEmails)
}

// fromAPIPolicy populates the reporter policy and the predefined reporter.
func (m *mailRuleResourceModel) fromAPIPolicy(policy *youtrack.MailRulePolicy) {
	if policy == nil {
		return
	}

	m.ReporterPolicy = types.StringValue(policy.Policy)
	m.DefaultReporterLogin = types.StringNull()
	if policy.User != nil {
		m.DefaultReporterLogin = types.StringValue(policy.User.Login)
	}
}

// fromAPIFilters populates the filters. A rule without filters keeps an empty
// list when the configuration has one, so `filters = []` does not drift to null.
func (m *mailRuleResourceModel) fromAPIFilters(filters []youtrack.MailRuleFilter) {
	if len(filters) == 0 {
		if m.Filters != nil {
			m.Filters = []mailRuleFilterModel{}
		}
		return
	}

	m.Filters = make([]mailRuleFilterModel, 0, len(filters))
	for _, filter := range filters {
		model := mailRuleFilterModel{
			Patterns:      splitMailRulePatterns(filter.Patterns),
			AcceptMatched: types.BoolValue(filter.AcceptMatched),
		}
		if filter.Scope != nil {
			model.Scope = types.StringValue(filter.Scope.ID)
		}
		m.Filters = append(m.Filters, model)
	}
}

// splitMailRulePatterns splits the patterns of a filter, one per line.
func splitMailRulePatterns(patterns string) []types.String {
	lines := strings.Split(strings.ReplaceAll(patterns, "\r\n", mailRulePatternSeparator), mailRulePatternSeparator)
	result := make([]types.String, 0, len(lines))
	for _, line := range lines {
		result = append(result, types.StringValue(line))
	}

	return result
}
