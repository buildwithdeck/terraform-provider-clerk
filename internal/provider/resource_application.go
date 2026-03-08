package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ApplicationResource{}
	_ resource.ResourceWithConfigure   = &ApplicationResource{}
	_ resource.ResourceWithImportState = &ApplicationResource{}
)

func NewApplicationResource() resource.Resource {
	return &ApplicationResource{}
}

type ApplicationResource struct {
	client *PlatformClient
}

type ApplicationResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Domain           types.String `tfsdk:"domain"`
	ProxyPath        types.String `tfsdk:"proxy_path"`
	EnvironmentTypes types.List   `tfsdk:"environment_types"`
	Template         types.String `tfsdk:"template"`
	Instances        types.List   `tfsdk:"instances"`
}

type ApplicationInstanceModel struct {
	InstanceID      types.String `tfsdk:"instance_id"`
	EnvironmentType types.String `tfsdk:"environment_type"`
	SecretKey       types.String `tfsdk:"secret_key"`
	PublishableKey  types.String `tfsdk:"publishable_key"`
}

var applicationInstanceAttrTypes = map[string]attr.Type{
	"instance_id":      types.StringType,
	"environment_type": types.StringType,
	"secret_key":       types.StringType,
	"publishable_key":  types.StringType,
}

func (r *ApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The clerk_application resource requires a platform_api_key. Set it in the provider configuration or via the CLERK_PLATFORM_API_KEY environment variable. The Platform API is a beta feature that must be enabled by Clerk — contact Clerk support or visit your dashboard to request access.",
		)
		return
	}
	r.client = NewPlatformClient(data.PlatformAPIKey)
}

func (r *ApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *ApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk application via the Platform API. The Platform API is a beta feature that must be enabled by Clerk — contact Clerk support or visit your dashboard to request access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the application.",
			},
			"domain": schema.StringAttribute{
				Optional:    true,
				Description: "Custom domain for the application. Create-only; changing forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"proxy_path": schema.StringAttribute{
				Optional:    true,
				Description: "Proxy path for the application. Create-only; changing forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_types": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Environment types to create (e.g. [\"development\", \"production\"]). Create-only; changing forces replacement.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"template": schema.StringAttribute{
				Optional:    true,
				Description: "Application template slug. Create-only; changing forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instances": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of instances (environments) created for this application.",
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

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &CreateApplicationParams{
		Name: plan.Name.ValueString(),
	}

	if !plan.Domain.IsNull() && !plan.Domain.IsUnknown() {
		params.Domain = plan.Domain.ValueString()
	}
	if !plan.ProxyPath.IsNull() && !plan.ProxyPath.IsUnknown() {
		params.ProxyPath = plan.ProxyPath.ValueString()
	}
	if !plan.Template.IsNull() && !plan.Template.IsUnknown() {
		params.Template = plan.Template.ValueString()
	}
	if !plan.EnvironmentTypes.IsNull() && !plan.EnvironmentTypes.IsUnknown() {
		var envTypes []string
		diags = plan.EnvironmentTypes.ElementsAs(ctx, &envTypes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.EnvironmentTypes = envTypes
	}

	app, err := r.client.CreateApplication(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create application", err.Error())
		return
	}

	mapApplicationResponseToModel(app, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created application", map[string]any{"id": app.ApplicationID})
}

func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetApplication(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read application",
			fmt.Sprintf("Could not read application ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	// Preserve fields not returned by the API from state.
	name := state.Name
	domain := state.Domain
	proxyPath := state.ProxyPath
	envTypes := state.EnvironmentTypes
	template := state.Template

	mapApplicationResponseToModel(app, &state)

	state.Name = name
	state.Domain = domain
	state.ProxyPath = proxyPath
	state.EnvironmentTypes = envTypes
	state.Template = template

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ApplicationResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &UpdateApplicationParams{
		Name: plan.Name.ValueString(),
	}

	app, err := r.client.UpdateApplication(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update application", err.Error())
		return
	}

	mapApplicationResponseToModel(app, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated application", map[string]any{"id": app.ApplicationID})
}

func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.DeleteApplication(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete application",
			fmt.Sprintf("Could not delete application ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted application", map[string]any{"id": state.ID.ValueString()})
}

func (r *ApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapApplicationResponseToModel(app *PlatformApplicationResponse, model *ApplicationResourceModel) {
	model.ID = types.StringValue(app.ApplicationID)

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
	model.Instances, _ = types.ListValue(instanceType, instances)
}
