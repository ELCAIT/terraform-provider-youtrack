package settings

import (
	"context"

	helpers "github.com/elcait/youtrack-provider/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// Port validation constants
	minValidPort = 1
	maxValidPort = 65535

	// Default mail server settings
	defaultMailPort = 25
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &mailServerResource{}
	_ resource.ResourceWithConfigure      = &mailServerResource{}
	_ resource.ResourceWithImportState    = &mailServerResource{}
	_ resource.ResourceWithValidateConfig = &mailServerResource{}
)

// NewMailServerResource is a helper function to simplify the provider implementation.
func NewMailServerResource() resource.Resource {
	return &mailServerResource{}
}

// mailServerResource is the resource implementation.
type mailServerResource struct {
	client *youtrack.Client
}

// mailServerResourceModel maps the resource schema data.
type mailServerResourceModel struct {
	ID           types.String `tfsdk:"id"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	MailProtocol types.String `tfsdk:"mail_protocol"`
	Host         types.String `tfsdk:"host"`
	Port         types.Int64  `tfsdk:"port"`
	Anonymous    types.Bool   `tfsdk:"anonymous"`
	Login        types.String `tfsdk:"login"`
	From         types.String `tfsdk:"from"`
	ReplyTo      types.String `tfsdk:"reply_to"`
	LastUpdated  types.String `tfsdk:"last_updated"`
}

// Metadata returns the resource type name.
func (r *mailServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_settings"
}

// Schema defines the schema for the resource.
func (r *mailServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "YouTrack mail server configuration",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Notification settings identifier",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Description: "Enable/disable the notification",
				Required:    true,
			},
			"mail_protocol": schema.StringAttribute{
				Description: "Mail protocol to use (e.g., SMTP, IMAP)",
				Required:    true,
			},
			"host": schema.StringAttribute{
				Description: "Mail server DNS name or IP address",
				Required:    true,
			},
			"port": schema.Int64Attribute{
				Description: "Mail server port",
				Required:    true,
			},
			"anonymous": schema.BoolAttribute{
				Description: "Enable/disable anonymous access",
				Required:    true,
			},
			"login": schema.StringAttribute{
				Description: "Login for mail server authentication",
				Optional:    true,
				Computed:    true,
			},
			"from": schema.StringAttribute{
				Description: "Email address to use in the From field",
				Required:    true,
			},
			"reply_to": schema.StringAttribute{
				Description: "Email address to use in the Reply-To field",
				Optional:    true,
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last update",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *mailServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mailServerResourceModel
	if !helpers.GetPlanAndCheckError(ctx, req, resp, &plan) {
		return
	}

	ms := convertModelToMailServer(plan)

	mailServer, ok := r.updateAndHandleError(ctx, ms, &resp.Diagnostics)
	if !ok {
		return
	}

	updateMailServerModelWithTimestamp(mailServer, &plan)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Read refreshes the Terraform state with the latest data.
func (r *mailServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mailServerResourceModel
	if !helpers.GetStateAndCheckError(ctx, req, resp, &state) {
		return
	}

	mailServer, ok := r.getMailServerAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	updateMailServerModelWithTimestamp(mailServer, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mailServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mailServerResourceModel
	if !helpers.GetPlanAndCheckErrorUpdate(ctx, req, resp, &plan) {
		return
	}

	ms := convertModelToMailServer(plan)

	mailServer, ok := r.updateAndHandleError(ctx, ms, &resp.Diagnostics)
	if !ok {
		return
	}

	// Map response body to model and update timestamp
	updateMailServerModelWithTimestamp(mailServer, &plan)

	helpers.SetStateAndCheckError(ctx, resp, plan)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mailServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mailServerResourceModel
	if !helpers.GetStateAndCheckErrorDelete(ctx, req, resp, &state) {
		return
	}

	var ms = youtrack.MailServer{
		IsEnabled:    false,
		MailProtocol: "",
		Host:         "",
		Port:         defaultMailPort,
		Anonymous:    false,
		Login:        "",
		From:         "",
		ReplyTo:      "",
	}

	mailServer, ok := r.updateAndHandleError(ctx, ms, &resp.Diagnostics)
	if !ok {
		return
	}

	updateMailServerModelWithTimestamp(mailServer, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

func (r *mailServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	mailServer, ok := r.getMailServerAndHandleError(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	var state mailServerResourceModel
	updateMailServerModelWithTimestamp(mailServer, &state)

	helpers.SetStateAndCheckError(ctx, resp, &state)
}

// ValidateConfig validates the resource configuration.
func (r *mailServerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config mailServerResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate port is in valid range
	validatePortField(config.Port, &resp.Diagnostics)

	// Validate email fields
	helpers.ValidateEmailField(config.Login, path.Root("login"), "login", &resp.Diagnostics)
	helpers.ValidateEmailField(config.From, path.Root("from"), "from field", &resp.Diagnostics)
	helpers.ValidateEmailField(config.ReplyTo, path.Root("reply_to"), "reply_to field", &resp.Diagnostics)

	// Validate required fields when mail server is enabled
	validateRequiredFieldWhenEnabled(config.IsEnabled, config.Host, "host", "host", &resp.Diagnostics)
	validateRequiredFieldWhenEnabled(config.IsEnabled, config.MailProtocol, "mail_protocol", "mail_protocol", &resp.Diagnostics)
}

// Configure adds the provider configured client to the resource.
func (r *mailServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client, ok := helpers.GetClientFromConfigure(req, resp); ok {
		r.client = client
	}
}
