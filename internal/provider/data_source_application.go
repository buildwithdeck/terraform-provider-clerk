package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &ApplicationDataSource{}
	_ datasource.DataSourceWithConfigure = &ApplicationDataSource{}
)

func NewApplicationDataSource() datasource.DataSource {
	return &ApplicationDataSource{}
}

type ApplicationDataSource struct {
	client *PlatformClient
}

type ApplicationDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Instances types.List   `tfsdk:"instances"`
}

func (d *ApplicationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (d *ApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Clerk application by ID via the Platform API. Use this to look up application details and instance keys without remote state coupling.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The application ID to look up.",
			},
			"instances": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of instances (environments) for this application.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"instance_id": schema.StringAttribute{
							Computed:    true,
							Description: "Unique identifier of the instance.",
						},
						"environment_type": schema.StringAttribute{
							Computed:    true,
							Description: "Environment type (e.g. development, production).",
						},
						"secret_key": schema.StringAttribute{
							Computed:    true,
							Sensitive:   true,
							Description: "Secret key for this instance.",
						},
						"publishable_key": schema.StringAttribute{
							Computed:    true,
							Description: "Publishable key for this instance.",
						},
					},
				},
			},
		},
	}
}

func (d *ApplicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"Expected ProviderData, got something else. Please report this issue.",
		)
		return
	}
	if data.PlatformAPIKey == "" {
		resp.Diagnostics.AddError(
			"Missing Platform API Key",
			"The clerk_application data source requires a platform_api_key. Set it in the provider configuration or via the CLERK_PLATFORM_API_KEY environment variable.",
		)
		return
	}
	d.client = NewPlatformClient(data.PlatformAPIKey)
}

func (d *ApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ApplicationDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := d.client.GetApplication(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read application",
			fmt.Sprintf("Could not read application ID %s: %s", config.ID.ValueString(), err.Error()),
		)
		return
	}

	config.ID = types.StringValue(app.ApplicationID)

	instances := make([]attr.Value, len(app.Instances))
	for i, inst := range app.Instances {
		instances[i], _ = types.ObjectValue(applicationInstanceAttrTypes, map[string]attr.Value{
			"instance_id":      types.StringValue(inst.InstanceID),
			"environment_type": types.StringValue(inst.EnvironmentType),
			"secret_key":       types.StringValue(inst.SecretKey),
			"publishable_key":  types.StringValue(inst.PublishableKey),
		})
	}

	instanceType := types.ObjectType{AttrTypes: applicationInstanceAttrTypes}
	config.Instances, _ = types.ListValue(instanceType, instances)

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
