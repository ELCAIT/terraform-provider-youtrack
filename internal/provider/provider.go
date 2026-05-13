// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/elcait/youtrack-provider/internal/provider/settings"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure YouTrackProvider satisfies various provider interfaces.
var _ provider.Provider = &youtrackProvider{}

// YouTrackProvider defines the provider implementation.
type youtrackProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// youtrackProviderModel maps provider schema data to a Go type.
type youtrackProviderModel struct {
	BaseUrl types.String `tfsdk:"base_url"`
	Token   types.String `tfsdk:"token"`
}

func (p *youtrackProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "youtrack"
	resp.Version = p.version
}

func (p *youtrackProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Youtrack base configuration for provider",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Description: "YouTrack instance URL. Can also be set via TF_VAR_YOUTRACK_URL environment variable.",
				Required:    true,
			},
			"token": schema.StringAttribute{
				Description: "YouTrack API token. Can also be set via TF_VAR_YOUTRACK_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *youtrackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config youtrackProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.BaseUrl.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Unknown Youtrack API URL",
			"The provider cannot create the Youtrack API client as there is an unknown configuration value for the Youtrack API URL. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the YOUTRACK_URL environment variable.",
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown Youtrack API Token",
			"The provider cannot create the Youtrack API client as there is an unknown configuration value for the Youtrack API token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the YOUTRACK_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	url := os.Getenv("TF_VAR_YOUTRACK_URL")
	token := os.Getenv("TF_VAR_YOUTRACK_TOKEN")

	if !config.BaseUrl.IsNull() {
		url = config.BaseUrl.ValueString()
	}

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Missing Youtrack API URL",
			"The provider cannot create the Youtrack API client as there is a missing or empty value for the Youtrack API URL. "+
				"Set the URL value in the configuration or use the YOUTRACK_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Youtrack API Token",
			"The provider cannot create the Youtrack API client as there is a missing or empty value for the Youtrack API token. "+
				"Set the token value in the configuration or use the YOUTRACK_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := youtrack.NewClient(url, token)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Youtrack API Client",
			fmt.Sprintf("An unexpected error occurred when creating the Youtrack API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Youtrack Client Error: %v", err),
		)
		return
	}

	// Make the Youtrack client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources defines the resources implemented in the provider.
func (p *youtrackProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCustomFieldResource,
		NewEnumBundleResource,
		settings.NewAppearanceSettingsResource,
		settings.NewBackupSettingsResource,
		settings.NewGlobalSettingsResource,
		settings.NewGlobalTimeTrackingSettingsResource,
		NewIssueLinkTypeResource,
		settings.NewLocaleSettingsResource,
		settings.NewMailServerResource,
		NewNestedGroupResource,
		NewOAuth2AuthModuleResource,
		settings.NewRestSettingsResource,
		NewRoleResource,
		NewRoleAssignmentResource,
		NewProjectResource,
		NewProjectCustomFieldResource,
		NewProjectTimeTrackingSettingsResource,
		NewStateBundleResource,
		settings.NewSystemSettingsResource,
	}
}

func (p *youtrackProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
		NewGroupDataSource,
	}
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &youtrackProvider{
			version: version,
		}
	}
}
