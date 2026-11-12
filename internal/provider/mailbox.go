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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &mailboxResource{}
	_ resource.ResourceWithConfigure      = &mailboxResource{}
	_ resource.ResourceWithImportState    = &mailboxResource{}
	_ resource.ResourceWithValidateConfig = &mailboxResource{}
)

// NewMailboxResource is a helper function to simplify the provider implementation.
func NewMailboxResource() resource.Resource {
	return &mailboxResource{}
}

// mailboxResource is the resource implementation.
type mailboxResource struct {
	client *youtrack.Client
}

// Metadata returns the resource type name.
func (r *mailboxResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mailbox"
}

// Schema defines the schema for the resource.
func (r *mailboxResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a mailbox of the YouTrack mailbox integration: a mail account YouTrack polls for emails, " +
			"which its mail rules (`youtrack_mail_rule`) turn into issues and comments. " +
			"IMAP(S) and POP(S) mailboxes use `host`, `port`, `login` and `password`; " +
			"Microsoft Exchange Online (`OFFICE365`) mailboxes use the `microsoft_*` attributes. " +
			"Deleting a mailbox also deletes its mail rules.",
		Attributes: map[string]schema.Attribute{
			attrID: schema.StringAttribute{
				Computed:    true,
				Description: "Mailbox ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attrMailboxProtocol: schema.StringAttribute{
				Required: true,
				Description: "Mail protocol: `IMAP`, `IMAPS`, `OFFICE365` (Microsoft Exchange Online), or the deprecated `POP` and `POPS`. " +
					"Changing it replaces the mailbox.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			attrMailboxHost: schema.StringAttribute{
				Optional:    true,
				Description: "Mail server host name. Required for every protocol except `OFFICE365`.",
			},
			attrMailboxPort: schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Mail server port. Required for every protocol except `OFFICE365`.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			attrMailboxLogin: schema.StringAttribute{
				Optional:    true,
				Description: "Login of the mail account. Required for every protocol except `OFFICE365`.",
			},
			attrMailboxPassword: schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Password of the mail account. Required for every protocol except `OFFICE365`. " +
					"This value is write-only and is not returned by the API.",
			},
			attrMailboxSocketTimeout: schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Socket timeout, in seconds (YouTrack defaults to 30, at most 300 on a stock instance).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			attrMailboxConnectionTimeout: schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Connection timeout, in seconds (YouTrack defaults to 30, at most 300 on a stock instance).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			attrMailboxExchangeCompatibilityMode: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to work around the IMAP quirks of Microsoft Exchange servers. Defaults to false.",
			},
			attrMailboxMicrosoftAddress: schema.StringAttribute{
				Optional:    true,
				Description: "Email address of the Exchange Online mailbox. Required for `OFFICE365`.",
			},
			attrMailboxMicrosoftClientID: schema.StringAttribute{
				Optional:    true,
				Description: "Application (client) ID of the Microsoft Entra ID app YouTrack reads the mailbox with. Required for `OFFICE365`.",
			},
			attrMailboxMicrosoftTenantID: schema.StringAttribute{
				Optional:    true,
				Description: "Directory (tenant) ID of the Microsoft Entra ID app. Required for `OFFICE365`.",
			},
			attrMailboxMicrosoftClientSecret: schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Client secret of the Microsoft Entra ID app. Required for `OFFICE365`. " +
					"This value is write-only and is not returned by the API.",
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *mailboxResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}

// ValidateConfig checks that the attributes the protocol needs are set, and that
// the attributes of the other kind of mailbox are not.
func (r *mailboxResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config mailboxResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateMailboxConfig(&config, &resp.Diagnostics)
}

// Create creates the resource and sets the initial Terraform state.
func (r *mailboxResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mailboxResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	created, err := r.client.CreateMailbox(ctx, plan.toPayload())
	if err != nil {
		resp.Diagnostics.AddError(errCreatingMailbox, fmt.Sprintf("Could not create mailbox: %v", err))
		return
	}

	plan.fromAPIModel(created)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *mailboxResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mailboxResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}
	if !helpers.ValidateResourceID(state.ID, &resp.Diagnostics, errMissingMailboxID, errMailboxIDRequired) {
		return
	}

	mailbox, err := r.client.GetMailboxByID(ctx, state.ID.ValueString())
	if err != nil {
		if youtrack.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(errReadingMailbox, fmt.Sprintf("Could not read mailbox: %v", err))
		return
	}

	state.fromAPIModel(mailbox)
	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mailboxResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mailboxResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}
	if !helpers.ValidateResourceID(plan.ID, &resp.Diagnostics, errMissingMailboxID, errMailboxIDRequired) {
		return
	}

	updated, err := r.client.UpdateMailbox(ctx, plan.ID.ValueString(), plan.toPayload())
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingMailbox, fmt.Sprintf(helpers.ErrCouldNotUpdateFmt, "mailbox", err))
		return
	}

	plan.fromAPIModel(updated)
	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the mailbox, and with it its mail rules.
func (r *mailboxResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mailboxResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}
	if !helpers.HasResourceID(state.ID) {
		return
	}

	if err := r.client.DeleteMailbox(ctx, state.ID.ValueString()); err != nil && !youtrack.IsNotFoundError(err) {
		resp.Diagnostics.AddError(errDeletingMailbox, fmt.Sprintf("Could not delete mailbox: %v", err))
	}
}

// ImportState imports the resource state by mailbox ID.
func (r *mailboxResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(attrID), req, resp)
}
