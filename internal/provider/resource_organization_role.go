package provider

import (
	"context"
	"fmt"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/organizationrole"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &OrganizationRoleResource{}
	_ resource.ResourceWithConfigure   = &OrganizationRoleResource{}
	_ resource.ResourceWithImportState = &OrganizationRoleResource{}
)

func NewOrganizationRoleResource() resource.Resource {
	return &OrganizationRoleResource{}
}

type OrganizationRoleResource struct {
	configured bool
}

type OrganizationRoleResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Key                types.String `tfsdk:"key"`
	Description        types.String `tfsdk:"description"`
	Permissions        types.List   `tfsdk:"permissions"`
	IsCreatorEligible  types.Bool   `tfsdk:"is_creator_eligible"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *OrganizationRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_role"
}

func (r *OrganizationRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk organization role.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the organization role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the role.",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "Unique key for the role (e.g. org:editor).",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the role.",
			},
			"permissions": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of permission IDs to assign to this role.",
			},
			"is_creator_eligible": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the role is eligible to be assigned to organization creators.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role was last updated.",
			},
		},
	}
}

func (r *OrganizationRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationRoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationrole.CreateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
		Key:  clerkgo.String(plan.Key.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var permIDs []string
		diags = plan.Permissions.ElementsAs(ctx, &permIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.Permissions = &permIDs
	}

	role, err := organizationrole.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create organization role", err.Error())
		return
	}

	mapOrganizationRoleResponseToModel(ctx, role, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created organization role", map[string]any{"id": role.ID})
}

func (r *OrganizationRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationRoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := organizationrole.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read organization role",
			fmt.Sprintf("Could not read organization role ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapOrganizationRoleResponseToModel(ctx, role, &state, &resp.Diagnostics)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OrganizationRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationRoleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrganizationRoleResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationrole.UpdateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
		Key:  clerkgo.String(plan.Key.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var permIDs []string
		diags = plan.Permissions.ElementsAs(ctx, &permIDs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.Permissions = &permIDs
	}

	role, err := organizationrole.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update organization role", err.Error())
		return
	}

	mapOrganizationRoleResponseToModel(ctx, role, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated organization role", map[string]any{"id": role.ID})
}

func (r *OrganizationRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationRoleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := organizationrole.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete organization role",
			fmt.Sprintf("Could not delete organization role ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted organization role", map[string]any{"id": state.ID.ValueString()})
}

func (r *OrganizationRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapOrganizationRoleResponseToModel(ctx context.Context, role *clerkgo.OrganizationRole, model *OrganizationRoleResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(role.ID)
	model.Name = types.StringValue(role.Name)
	model.Key = types.StringValue(role.Key)
	model.IsCreatorEligible = types.BoolValue(role.IsCreatorEligible)
	model.CreatedAt = types.StringValue(millisToRFC3339(role.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(role.UpdatedAt))

	if role.Description != nil {
		model.Description = types.StringValue(*role.Description)
	}

	// Map permission objects to list of IDs
	if len(role.Permissions) > 0 {
		permIDs := make([]string, len(role.Permissions))
		for i, p := range role.Permissions {
			permIDs[i] = p.ID
		}
		permList, d := types.ListValueFrom(ctx, types.StringType, permIDs)
		diags.Append(d...)
		model.Permissions = permList
	} else if !model.Permissions.IsNull() {
		// Preserve empty list if it was explicitly set
		model.Permissions, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}
}
