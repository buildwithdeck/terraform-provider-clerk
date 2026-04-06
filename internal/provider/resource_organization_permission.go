package provider

import (
	"context"
	"fmt"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/organizationpermission"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &OrganizationPermissionResource{}
	_ resource.ResourceWithConfigure   = &OrganizationPermissionResource{}
	_ resource.ResourceWithImportState = &OrganizationPermissionResource{}
)

func NewOrganizationPermissionResource() resource.Resource {
	return &OrganizationPermissionResource{}
}

type OrganizationPermissionResource struct {
	client *organizationpermission.Client
}

type OrganizationPermissionResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Key         types.String `tfsdk:"key"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *OrganizationPermissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	if data.APIKey == "" {
		resp.Diagnostics.AddError(
			"Missing Clerk API Key",
			"The clerk_organization_permission resource requires an api_key. Set it in the provider configuration or via the CLERK_API_KEY environment variable.",
		)
		return
	}
	r.client = organizationpermission.NewClient(&clerkgo.ClientConfig{BackendConfig: clerkgo.BackendConfig{Key: clerkgo.String(data.APIKey)}})
}

func (r *OrganizationPermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_permission"
}

func (r *OrganizationPermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk organization permission.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the organization permission.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the permission.",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "Unique key for the permission (e.g. org:posts:create).",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of what the permission allows.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of the permission (system or user).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the permission was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the permission was last updated.",
			},
		},
	}
}

func (r *OrganizationPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationPermissionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationpermission.CreateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
		Key:  clerkgo.String(plan.Key.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}

	perm, err := r.client.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create organization permission", err.Error())
		return
	}

	mapOrganizationPermissionResponseToModel(perm, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created organization permission", map[string]any{"id": perm.ID})
}

func (r *OrganizationPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationPermissionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	perm, err := r.client.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read organization permission",
			fmt.Sprintf("Could not read organization permission ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapOrganizationPermissionResponseToModel(perm, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OrganizationPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationPermissionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrganizationPermissionResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationpermission.UpdateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
		Key:  clerkgo.String(plan.Key.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}

	perm, err := r.client.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update organization permission", err.Error())
		return
	}

	mapOrganizationPermissionResponseToModel(perm, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated organization permission", map[string]any{"id": perm.ID})
}

func (r *OrganizationPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationPermissionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete organization permission",
			fmt.Sprintf("Could not delete organization permission ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted organization permission", map[string]any{"id": state.ID.ValueString()})
}

func (r *OrganizationPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapOrganizationPermissionResponseToModel(perm *clerkgo.OrganizationPermission, model *OrganizationPermissionResourceModel) {
	model.ID = types.StringValue(perm.ID)
	model.Name = types.StringValue(perm.Name)
	model.Key = types.StringValue(perm.Key)
	model.Type = types.StringValue(perm.Type)
	model.CreatedAt = types.StringValue(millisToRFC3339(perm.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(perm.UpdatedAt))

	if perm.Description != nil {
		model.Description = types.StringValue(*perm.Description)
	}
}
