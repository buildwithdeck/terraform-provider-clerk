package provider

import (
	"context"
	"os"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = &ClerkProvider{}

type ClerkProvider struct {
	version string
}

type ClerkProviderModel struct {
	APIKey         types.String `tfsdk:"api_key"`
	PlatformAPIKey types.String `tfsdk:"platform_api_key"`
}

// ProviderData is passed to resources via req.ProviderData so each resource
// can access the keys it needs.
type ProviderData struct {
	APIKey         string
	PlatformAPIKey string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ClerkProvider{
			version: version,
		}
	}
}

func (p *ClerkProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "clerk"
	resp.Version = p.version
}

func (p *ClerkProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Clerk resources. This provider uses two APIs: the Instance API (for jwt_template, organization, application_settings) requires an instance secret key (api_key), and the Platform API (for application, domain, instance_config) requires a Platform API beta key (platform_api_key).",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Clerk instance secret key (starts with sk_test_ or sk_live_). Required for clerk_jwt_template, clerk_organization, and clerk_application_settings resources. Can also be set via the CLERK_API_KEY environment variable.",
			},
			"platform_api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Clerk Platform API key (starts with ak_). Required for clerk_application, clerk_domain, and clerk_instance_config resources. The Platform API is a beta feature — contact Clerk support or visit your dashboard to request access. Can also be set via the CLERK_PLATFORM_API_KEY environment variable.",
			},
		},
	}
}

func (p *ClerkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Clerk provider")

	var config ClerkProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown Clerk API Key",
			"The provider cannot create the Clerk client as there is an unknown configuration value for the API key.",
		)
		return
	}

	apiKey := os.Getenv("CLERK_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Clerk API Key",
			"Set the api_key in the provider configuration or via the CLERK_API_KEY environment variable.",
		)
		return
	}

	clerkgo.SetKey(apiKey)

	// Resolve optional Platform API key.
	platformAPIKey := os.Getenv("CLERK_PLATFORM_API_KEY")
	if !config.PlatformAPIKey.IsNull() && !config.PlatformAPIKey.IsUnknown() {
		platformAPIKey = config.PlatformAPIKey.ValueString()
	}

	data := ProviderData{
		APIKey:         apiKey,
		PlatformAPIKey: platformAPIKey,
	}

	resp.DataSourceData = data
	resp.ResourceData = data

	tflog.Info(ctx, "Configured Clerk provider")
}

func (p *ClerkProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewJWTTemplateResource,
		NewOrganizationResource,
		NewApplicationSettingsResource,
		NewApplicationResource,
		NewDomainResource,
		NewInstanceConfigResource,
	}
}

func (p *ClerkProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
