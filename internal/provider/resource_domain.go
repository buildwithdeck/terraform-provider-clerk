package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithConfigure   = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

type DomainResource struct {
	client *PlatformClient
}

type DomainResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ApplicationID    types.String `tfsdk:"application_id"`
	Name             types.String `tfsdk:"name"`
	ProxyPath        types.String `tfsdk:"proxy_path"`
	IsSatellite      types.Bool   `tfsdk:"is_satellite"`
	IsProviderDomain types.Bool   `tfsdk:"is_provider_domain"`
	FrontendAPIURL   types.String `tfsdk:"frontend_api_url"`
	CNAMETargets     types.List   `tfsdk:"cname_targets"`
}

var domainCNAMETargetAttrTypes = map[string]attr.Type{
	"host":     types.StringType,
	"value":    types.StringType,
	"required": types.BoolType,
}

func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected ProviderData, got something else. Please report this issue.",
		)
		return
	}
	if data.PlatformAPIKey == "" {
		resp.Diagnostics.AddError(
			"Missing Platform API Key",
			"The clerk_domain resource requires a platform_api_key. Set it in the provider configuration or via the CLERK_PLATFORM_API_KEY environment variable. The Platform API is a beta feature that must be enabled by Clerk — contact Clerk support or visit your dashboard to request access.",
		)
		return
	}
	r.client = NewPlatformClient(data.PlatformAPIKey)
}

func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a domain on a Clerk application via the Platform API. The Platform API is a beta feature that must be enabled by Clerk — contact Clerk support or visit your dashboard to request access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the application this domain belongs to. Changing forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The domain name (e.g. app.example.com). Changing forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"proxy_path": schema.StringAttribute{
				Optional:    true,
				Description: "Proxy path for the domain. Create-only; changing forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_satellite": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a satellite domain.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_provider_domain": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a provider domain.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"frontend_api_url": schema.StringAttribute{
				Computed:    true,
				Description: "The frontend API URL for this domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cname_targets": schema.ListNestedAttribute{
				Computed:    true,
				Description: "CNAME targets for DNS configuration.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"host": schema.StringAttribute{
							Computed:    true,
							Description: "The CNAME host.",
						},
						"value": schema.StringAttribute{
							Computed:    true,
							Description: "The CNAME value.",
						},
						"required": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether this CNAME record is required.",
						},
					},
				},
			},
		},
	}
}

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &CreateDomainParams{
		Name: plan.Name.ValueString(),
	}

	if !plan.ProxyPath.IsNull() && !plan.ProxyPath.IsUnknown() {
		params.ProxyPath = plan.ProxyPath.ValueString()
	}

	domain, err := r.client.CreateDomain(ctx, plan.ApplicationID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create domain", err.Error())
		return
	}

	mapDomainResponseToModel(domain, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created domain", map[string]any{"id": domain.ID})
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := r.client.GetDomain(ctx, state.ApplicationID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read domain",
			fmt.Sprintf("Could not read domain ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	// Preserve application_id (not returned by API).
	appID := state.ApplicationID
	// Preserve proxy_path from state if not returned by API.
	proxyPath := state.ProxyPath

	mapDomainResponseToModel(domain, &state)

	state.ApplicationID = appID
	state.ProxyPath = proxyPath

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *DomainResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The Platform API does not support updating domains. All mutable attributes
	// use RequiresReplace, so this method should never be called.
	resp.Diagnostics.AddError(
		"Update not supported",
		"Domains cannot be updated via the Platform API. All attribute changes require replacement.",
	)
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteDomain(ctx, state.ApplicationID.ValueString(), state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*PlatformAPIError); ok {
			// Treat 404 as already deleted (e.g. parent application was deleted).
			if apiErr.StatusCode == 404 {
				tflog.Debug(ctx, "Domain already deleted", map[string]any{"id": state.ID.ValueString()})
				return
			}
			// Treat 500 as a warning — the Clerk Platform API domain delete
			// endpoint has a known server-side issue. The domain will be
			// cleaned up when the parent application is deleted.
			if apiErr.StatusCode == 500 {
				resp.Diagnostics.AddWarning(
					"Domain deletion returned a server error",
					fmt.Sprintf("The Clerk API returned HTTP 500 when deleting domain %s. The domain will be removed when the parent application is deleted. Error: %s", state.ID.ValueString(), err.Error()),
				)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Unable to delete domain",
			fmt.Sprintf("Could not delete domain ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted domain", map[string]any{"id": state.ID.ValueString()})
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: application_id/domain_id, got: %s", req.ID),
		)
		return
	}

	resp.State.SetAttribute(ctx, path.Root("application_id"), types.StringValue(parts[0]))
	resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(parts[1]))
}

func mapDomainResponseToModel(domain *PlatformDomainResponse, model *DomainResourceModel) {
	model.ID = types.StringValue(domain.ID)
	model.Name = types.StringValue(domain.Name)
	model.IsSatellite = types.BoolValue(domain.IsSatellite)
	model.IsProviderDomain = types.BoolValue(domain.IsProviderDomain)
	model.FrontendAPIURL = types.StringValue(domain.FrontendAPIURL)

	targets := make([]attr.Value, len(domain.CNAMETargets))
	for i, t := range domain.CNAMETargets {
		targets[i], _ = types.ObjectValue(domainCNAMETargetAttrTypes, map[string]attr.Value{
			"host":     types.StringValue(t.Host),
			"value":    types.StringValue(t.Value),
			"required": types.BoolValue(t.Required),
		})
	}

	targetType := types.ObjectType{AttrTypes: domainCNAMETargetAttrTypes}
	model.CNAMETargets, _ = types.ListValue(targetType, targets)
}
