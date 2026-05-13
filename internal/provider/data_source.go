package provider

import (
	"context"
	"fmt"

	helpers "github.com/elcait/terraform-provider-youtrack/internal/helpers"

	youtrack "github.com/elcait/youtrack-api-client/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

// NewUserDataSource is a helper function to simplify the provider implementation.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// userDataSource is the data source implementation.
type userDataSource struct {
	client *youtrack.Client
}

// userDataSourceModel maps the data source schema data.
type userDataSourceModel struct {
	Login    types.String `tfsdk:"login"`
	ID       types.String `tfsdk:"id"`
	FullName types.String `tfsdk:"full_name"`
}

const (
	errReadingUser = "Unable to read YouTrack user"
)

// Metadata returns the data source type name.
func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the data source.
func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a YouTrack user by login.",
		Attributes: map[string]schema.Attribute{
			"login": schema.StringAttribute{
				Required:    true,
				Description: "The login (username) of the YouTrack user.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Hub ID of the user.",
			},
			"full_name": schema.StringAttribute{
				Computed:    true,
				Description: "The display name of the user.",
			},
		},
	}
}

// Read fetches the user data from YouTrack and sets the state.
func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state userDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := d.client.GetUserByLogin(ctx, state.Login.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(errReadingUser, err.Error())
		return
	}

	state.ID = types.StringValue(user.Id)
	state.FullName = types.StringValue(user.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Configure adds the provider configured client to the data source.
func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*youtrack.Client)
	if !ok {
		resp.Diagnostics.AddError(
			helpers.ErrUnexpectedResourceConfigType,
			fmt.Sprintf(helpers.ErrUnexpectedConfigureType, req.ProviderData),
		)
		return
	}

	d.client = client
}
