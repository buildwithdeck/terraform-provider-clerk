package provider

import (
	"context"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/instancesettings"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &ApplicationSettingsResource{}
	_ resource.ResourceWithConfigure = &ApplicationSettingsResource{}
)

func NewApplicationSettingsResource() resource.Resource {
	return &ApplicationSettingsResource{}
}

type ApplicationSettingsResource struct {
	configured bool
}

type ApplicationSettingsResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	EnableOrganizations   types.Bool   `tfsdk:"enable_organizations"`
	MaxAllowedMemberships types.Int64  `tfsdk:"max_allowed_memberships"`
	MaxAllowedRoles       types.Int64  `tfsdk:"max_allowed_roles"`
	MaxAllowedPermissions types.Int64  `tfsdk:"max_allowed_permissions"`
	CreatorRole           types.String `tfsdk:"creator_role"`
	AdminDeleteEnabled    types.Bool   `tfsdk:"admin_delete_enabled"`
	DomainsEnabled        types.Bool   `tfsdk:"domains_enabled"`
	DomainsDefaultRole    types.String `tfsdk:"domains_default_role"`
}

func (r *ApplicationSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	_, ok := req.ProviderData.(ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			"Expected ProviderData, got something else. Please report this issue.",
		)
		return
	}
	r.configured = true
}

func (r *ApplicationSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_settings"
}

func (r *ApplicationSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages application-level settings for a Clerk instance. This is a singleton resource — only one can exist per instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Synthetic identifier for this singleton resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enable_organizations": schema.BoolAttribute{
				Required:    true,
				Description: "Whether organizations are enabled for this instance.",
			},
			"max_allowed_memberships": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Default maximum number of memberships allowed per organization.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_allowed_roles": schema.Int64Attribute{
				Computed:    true,
				Description: "Maximum number of roles allowed per organization.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"max_allowed_permissions": schema.Int64Attribute{
				Computed:    true,
				Description: "Maximum number of permissions allowed per organization.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"creator_role": schema.StringAttribute{
				Computed:    true,
				Description: "The role assigned to the creator of a new organization.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"admin_delete_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether administrators can delete organizations.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"domains_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether domain-based organization enrollment is enabled.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"domains_default_role": schema.StringAttribute{
				Computed:    true,
				Description: "Default role for users added via domain enrollment.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ApplicationSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationSettingsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := instancesettings.UpdateOrganizationSettings(ctx, buildAppSettingsParams(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to update application settings", err.Error())
		return
	}

	mapAppSettingsResponseToModel(settings, &plan)
	plan.ID = types.StringValue("application_settings")

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created application settings")
}

func (r *ApplicationSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationSettingsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The Clerk SDK has no GET endpoint for organization settings.
	// Re-apply current state via Update to get the latest values back.
	settings, err := instancesettings.UpdateOrganizationSettings(ctx, buildAppSettingsParams(&state))
	if err != nil {
		resp.Diagnostics.AddError("Unable to read application settings", err.Error())
		return
	}

	mapAppSettingsResponseToModel(settings, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *ApplicationSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationSettingsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := instancesettings.UpdateOrganizationSettings(ctx, buildAppSettingsParams(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to update application settings", err.Error())
		return
	}

	mapAppSettingsResponseToModel(settings, &plan)
	plan.ID = types.StringValue("application_settings")

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated application settings")
}

func (r *ApplicationSettingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// "Deleting" application settings means disabling organizations.
	params := &instancesettings.UpdateOrganizationSettingsParams{
		Enabled: clerk.Bool(false),
	}
	_, err := instancesettings.UpdateOrganizationSettings(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to disable application settings", err.Error())
		return
	}
	tflog.Debug(ctx, "Disabled application settings (resource deleted)")
}

func buildAppSettingsParams(model *ApplicationSettingsResourceModel) *instancesettings.UpdateOrganizationSettingsParams {
	params := &instancesettings.UpdateOrganizationSettingsParams{
		Enabled: clerk.Bool(model.EnableOrganizations.ValueBool()),
	}
	if !model.MaxAllowedMemberships.IsNull() && !model.MaxAllowedMemberships.IsUnknown() {
		params.MaxAllowedMemberships = clerk.Int64(model.MaxAllowedMemberships.ValueInt64())
	}
	if !model.AdminDeleteEnabled.IsNull() && !model.AdminDeleteEnabled.IsUnknown() {
		params.AdminDeleteEnabled = clerk.Bool(model.AdminDeleteEnabled.ValueBool())
	}
	if !model.DomainsEnabled.IsNull() && !model.DomainsEnabled.IsUnknown() {
		params.DomainsEnabled = clerk.Bool(model.DomainsEnabled.ValueBool())
	}
	return params
}

func mapAppSettingsResponseToModel(settings *clerk.OrganizationSettings, model *ApplicationSettingsResourceModel) {
	model.EnableOrganizations = types.BoolValue(settings.Enabled)
	model.MaxAllowedMemberships = types.Int64Value(settings.MaxAllowedMemberships)
	model.MaxAllowedRoles = types.Int64Value(settings.MaxAllowedRoles)
	model.MaxAllowedPermissions = types.Int64Value(settings.MaxAllowedPermissions)
	model.CreatorRole = types.StringValue(settings.CreatorRole)
	model.AdminDeleteEnabled = types.BoolValue(settings.AdminDeleteEnabled)
	model.DomainsEnabled = types.BoolValue(settings.DomainsEnabled)
	model.DomainsDefaultRole = types.StringValue(settings.DomainsDefaultRole)
}
