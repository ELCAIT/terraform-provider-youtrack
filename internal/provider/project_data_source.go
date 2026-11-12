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
	_ datasource.DataSource              = &projectDataSource{}
	_ datasource.DataSourceWithConfigure = &projectDataSource{}
)

// NewProjectDataSource is a helper function to simplify the provider implementation.
func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

// projectDataSource is the data source implementation.
type projectDataSource struct {
	client *youtrack.Client
}

// projectDataSourceModel maps the data source schema data.
type projectDataSourceModel struct {
	ShortName   types.String `tfsdk:"short_name"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	LeaderLogin types.String `tfsdk:"leader_login"`
	Archived    types.Bool   `tfsdk:"archived"`
	Template    types.Bool   `tfsdk:"template"`
}

const (
	errReadingProjectDataSource = "Unable to read YouTrack project"
)

// Metadata returns the data source type name.
func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema defines the schema for the data source.
func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves an existing YouTrack project by its short name (the issue ID prefix, such as `DEMO`), " +
			"for instance to reference a project that is not managed by Terraform.",
		Attributes: map[string]schema.Attribute{
			"short_name": schema.StringAttribute{
				Required:    true,
				Description: "Short name of the project, the prefix of its issue IDs.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Entity ID of the project, such as `0-0`.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the project.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description of the project.",
			},
			"leader_login": schema.StringAttribute{
				Computed:    true,
				Description: "Login of the project owner.",
			},
			"archived": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the project is archived.",
			},
			"template": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the project is a template.",
			},
		},
	}
}

// Read fetches the project from YouTrack and sets the state. YouTrack accepts
// a project's short name wherever it expects its ID.
func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, err := d.client.GetProject(ctx, state.ShortName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(errReadingProjectDataSource,
			fmt.Sprintf("Could not find project with short name %q: %v", state.ShortName.ValueString(), err))
		return
	}

	state.fromAPIModel(project)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// fromAPIModel populates the data source model from the API model.
func (m *projectDataSourceModel) fromAPIModel(project *youtrack.Project) {
	m.ID = types.StringValue(project.ID)
	m.ShortName = types.StringValue(project.ShortName)
	m.Name = types.StringValue(project.Name)
	m.Description = helpers.StringOrNull(project.Description)
	m.Archived = types.BoolValue(project.Archived)
	m.Template = types.BoolValue(project.Template)
	m.LeaderLogin = types.StringNull()
	if project.Leader != nil {
		m.LeaderLogin = types.StringValue(project.Leader.Login)
	}
}

// Configure adds the provider configured client to the data source.
func (d *projectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
