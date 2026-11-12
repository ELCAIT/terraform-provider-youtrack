// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &mailRuleResource{}
	_ resource.ResourceWithConfigure      = &mailRuleResource{}
	_ resource.ResourceWithImportState    = &mailRuleResource{}
	_ resource.ResourceWithValidateConfig = &mailRuleResource{}
)

// NewMailRuleResource is a helper function to simplify the provider implementation.
func NewMailRuleResource() resource.Resource {
	return &mailRuleResource{}
}

// mailRuleResource is the resource implementation.
type mailRuleResource struct {
	client *youtrack.Client
}

// Metadata returns the resource type name.
func (r *mailRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mail_rule"
}

// optionalBoolAttribute is an Optional+Computed bool attribute with a static default.
func optionalBoolAttribute(defaultValue bool, description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(defaultValue),
		Description: fmt.Sprintf("%s Defaults to %t.", description, defaultValue),
	}
}

// Schema defines the schema for the resource.
func (r *mailRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a mail rule of the YouTrack mailbox integration: which emails of a mailbox " +
			"(`youtrack_mailbox`) YouTrack turns into issues and comments of a project, and how.",
		Attributes: map[string]schema.Attribute{
			attrID: schema.StringAttribute{
				Computed:    true,
				Description: "Mail rule ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attrMailRuleMailboxID: schema.StringAttribute{
				Required:    true,
				Description: "ID of the mailbox the rule reads emails from. Changing it replaces the rule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			attrMailRuleProjectID: schema.StringAttribute{
				Required:    true,
				Description: "ID of the project the rule creates issues in.",
			},
			attrMailRuleMaintainerLogin: schema.StringAttribute{
				Required:    true,
				Description: "Login of the user YouTrack notifies when it cannot process an email.",
			},
			attrMailRuleReporterPolicy: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(youtrack.MailRulePolicySenderOrNewUser),
				Description: "Who becomes the reporter of an issue created from an email: `SENDER_OR_NEW_USER` " +
					"(the sender, registered as a new user if unknown), `SENDER_OR_DEFAULT` (the sender, or " +
					"`default_reporter_login` if unknown) or `ALWAYS_PREDEFINED` (always `default_reporter_login`). " +
					"Defaults to `SENDER_OR_NEW_USER`.",
			},
			attrMailRuleDefaultReporterLogin: schema.StringAttribute{
				Optional:    true,
				Description: "Login of the predefined reporter. Required for `SENDER_OR_DEFAULT` and `ALWAYS_PREDEFINED`, not allowed otherwise.",
			},
			attrMailRuleDisabled: optionalBoolAttribute(false, "Whether the rule is disabled."),
			attrMailRuleFolder: schema.StringAttribute{
				Optional:    true,
				Description: "Mailbox folder to read emails from. When unset, YouTrack reads the inbox.",
			},
			attrMailRuleTo: schema.StringAttribute{
				Optional:    true,
				Description: "Only process emails sent to this address.",
			},
			attrMailRuleReplyTo: schema.StringAttribute{
				Optional:    true,
				Description: "Reply-To address of the notifications YouTrack sends about the issues this rule creates.",
			},
			attrMailRuleSpamFilterMode: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(youtrack.MailRuleSpamFilterMark),
				Description: "What to do with emails the spam filter flags: `NONE` (ignore the spam filter), " +
					"`MARK_TICKETS` (create the issue and mark it as spam) or `BLOCK_TICKETS` (drop the email). " +
					"Defaults to `MARK_TICKETS`.",
			},
			attrMailRuleFilters: schema.ListNestedAttribute{
				Optional:    true,
				Description: "Filters that accept or reject emails by their subject or body.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						attrMailRuleFilterScope: schema.StringAttribute{
							Required:    true,
							Description: "What the patterns are matched against: `SUBJECT`, `BODY` or `SUBJECT_AND_BODY`.",
						},
						attrMailRuleFilterPatterns: schema.ListAttribute{
							Required:    true,
							ElementType: types.StringType,
							Description: "Patterns to match.",
						},
						attrMailRuleFilterAcceptMatched: schema.BoolAttribute{
							Required:    true,
							Description: "true to process only the emails that match, false to reject the emails that match.",
						},
					},
				},
			},
			attrMailRuleFilterAutoReplies: optionalBoolAttribute(true, "Whether to ignore automatic replies, such as out-of-office messages."),
			attrMailRuleNewIssueCommand: schema.StringAttribute{
				Optional:    true,
				Description: "Command applied to every issue the rule creates, such as `Type Bug`.",
			},
			attrMailRuleNewCommentCommand: schema.StringAttribute{
				Optional:    true,
				Description: "Command applied to the issue of every comment the rule creates.",
			},
			attrMailRuleKeepBrackets:         optionalBoolAttribute(false, "Whether to keep the text in square brackets of the subject in the issue summary."),
			attrMailRuleDeleteAfterFetch:     optionalBoolAttribute(false, "Whether to delete emails from the mailbox once processed."),
			attrMailRuleOriginalTimestamps:   optionalBoolAttribute(true, "Whether issues and comments get the date of the email instead of the date it was processed."),
			attrMailRulePlainTextDescription: optionalBoolAttribute(false, "Whether to use the plain text part of the email as issue description instead of the HTML part."),
			attrMailRuleFetchOldEmails:       optionalBoolAttribute(false, "Whether to process emails that were already in the mailbox, including read ones."),
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *mailRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

// ValidateConfig checks the enumerated values and the predefined reporter.
func (r *mailRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config mailRuleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateMailRuleConfig(&config, &resp.Diagnostics)
}

// Create creates the resource and sets the initial Terraform state.
func (r *mailRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mailRuleResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	payload, ok := r.buildPayload(ctx, &plan, errCreatingMailRule, &resp.Diagnostics)
	if !ok {
		return
	}
	payload.Mailbox = &youtrack.MailboxIDRef{ID: plan.MailboxID.ValueString()}

	created, err := r.client.CreateMailRule(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError(errCreatingMailRule, fmt.Sprintf("Could not create mail rule: %v", err))
		return
	}

	plan.fromAPIModel(created)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *mailRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mailRuleResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}
	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingMailRuleID, errMailRuleIDRequired) {
		return
	}

	rule, err := r.client.GetMailRuleByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(errReadingMailRule, fmt.Sprintf("Could not read mail rule: %v", err))
		return
	}

	state.fromAPIModel(rule)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mailRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mailRuleResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}
	if !helpers.ValidateResourceID(plan.ID, &resp.Diagnostics, errMissingMailRuleID, errMailRuleIDRequired) {
		return
	}

	payload, ok := r.buildPayload(ctx, &plan, errUpdatingMailRule, &resp.Diagnostics)
	if !ok {
		return
	}

	updated, err := r.client.UpdateMailRule(ctx, plan.ID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingMailRule, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "mail rule", err))
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mailRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mailRuleResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}
	if !helpers.HasResourceID(state.ID) {
		return
	}

	if err := r.client.DeleteMailRule(ctx, state.ID.ValueString()); err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(errDeletingMailRule, fmt.Sprintf("Could not delete mail rule: %v", err))
	}
}

// ImportState imports the resource state by mail rule ID.
func (r *mailRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(attrID), req, resp)
}
