package provider

import (
	"context"
	"fmt"
	"net/http"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/roleset"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &RoleSetResource{}
	_ resource.ResourceWithConfigure   = &RoleSetResource{}
	_ resource.ResourceWithImportState = &RoleSetResource{}
)

func NewRoleSetResource() resource.Resource {
	return &RoleSetResource{}
}

type RoleSetResource struct {
	configured bool
}

type RoleSetResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Key            types.String `tfsdk:"key"`
	Description    types.String `tfsdk:"description"`
	Type           types.String `tfsdk:"type"`
	DefaultRoleKey types.String `tfsdk:"default_role_key"`
	CreatorRoleKey types.String `tfsdk:"creator_role_key"`
	Roles          types.List   `tfsdk:"roles"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

// roleSetCreateParams extends the SDK's CreateParams with fields the API
// requires but the SDK (v2.5.1) does not yet include.
type roleSetCreateParams struct {
	clerkgo.APIParams
	Name           *string   `json:"name,omitempty"`
	Key            *string   `json:"key,omitempty"`
	Description    *string   `json:"description,omitempty"`
	Type           *string   `json:"type,omitempty"`
	Roles          *[]string `json:"roles,omitempty"`
	DefaultRoleKey *string   `json:"default_role_key,omitempty"`
	CreatorRoleKey *string   `json:"creator_role_key,omitempty"`
}

func (r *RoleSetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_set"
}

func (r *RoleSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk role set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the role set.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name of the role set.",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "Unique key for the role set.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the role set.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Type of the role set (initial or custom).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_role_key": schema.StringAttribute{
				Required:    true,
				Description: "Key of the role automatically assigned to new organization members.",
			},
			"creator_role_key": schema.StringAttribute{
				Required:    true,
				Description: "Key of the role assigned to the creator of an organization.",
			},
			"roles": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "List of role keys to include in this role set. Must include default_role_key and creator_role_key.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role set was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the role set was last updated.",
			},
		},
	}
}

func (r *RoleSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleSetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &roleSetCreateParams{
		Name:           clerkgo.String(plan.Name.ValueString()),
		Key:            clerkgo.String(plan.Key.ValueString()),
		DefaultRoleKey: clerkgo.String(plan.DefaultRoleKey.ValueString()),
		CreatorRoleKey: clerkgo.String(plan.CreatorRoleKey.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}

	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		params.Type = clerkgo.String(plan.Type.ValueString())
	}

	var roleKeys []string
	diags = plan.Roles.ElementsAs(ctx, &roleKeys, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	params.Roles = &roleKeys

	apiReq := clerkgo.NewAPIRequest(http.MethodPost, "/role_sets")
	apiReq.SetParams(params)
	rs := &clerkgo.RoleSet{}
	err := clerkgo.GetBackend().Call(ctx, apiReq, rs)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create role set", err.Error())
		return
	}

	mapRoleSetResponseToModel(ctx, rs, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created role set", map[string]any{"id": rs.ID})
}

func (r *RoleSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleSetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rs, err := roleset.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read role set",
			fmt.Sprintf("Could not read role set ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapRoleSetResponseToModel(ctx, rs, &state, &resp.Diagnostics)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *RoleSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RoleSetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RoleSetResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update scalar fields.
	params := &roleset.UpdateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
		Key:  clerkgo.String(plan.Key.ValueString()),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}

	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		params.Type = clerkgo.String(plan.Type.ValueString())
	}

	_, err := roleset.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update role set", err.Error())
		return
	}

	// Update default/creator role keys if changed.
	if plan.DefaultRoleKey.ValueString() != state.DefaultRoleKey.ValueString() ||
		plan.CreatorRoleKey.ValueString() != state.CreatorRoleKey.ValueString() {
		type roleSetUpdateKeysParams struct {
			clerkgo.APIParams
			DefaultRoleKey *string `json:"default_role_key,omitempty"`
			CreatorRoleKey *string `json:"creator_role_key,omitempty"`
		}
		keysParams := &roleSetUpdateKeysParams{
			DefaultRoleKey: clerkgo.String(plan.DefaultRoleKey.ValueString()),
			CreatorRoleKey: clerkgo.String(plan.CreatorRoleKey.ValueString()),
		}
		updatePath, pathErr := clerkgo.JoinPath("/role_sets", state.ID.ValueString())
		if pathErr != nil {
			resp.Diagnostics.AddError("Unable to build role set update path", pathErr.Error())
			return
		}
		updateReq := clerkgo.NewAPIRequest(http.MethodPatch, updatePath)
		updateReq.SetParams(keysParams)
		rs := &clerkgo.RoleSet{}
		if err := clerkgo.GetBackend().Call(ctx, updateReq, rs); err != nil {
			resp.Diagnostics.AddError("Unable to update role set keys", err.Error())
			return
		}
	}

	// Diff roles and apply AddRoles/RemoveRoles.
	var oldRoles, newRoles []string
	if !state.Roles.IsNull() && !state.Roles.IsUnknown() {
		diags = state.Roles.ElementsAs(ctx, &oldRoles, false)
		resp.Diagnostics.Append(diags...)
	}
	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		diags = plan.Roles.ElementsAs(ctx, &newRoles, false)
		resp.Diagnostics.Append(diags...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	toAdd, toRemove := diffStringSlices(oldRoles, newRoles)

	if len(toAdd) > 0 {
		_, err := roleset.AddRoles(ctx, state.ID.ValueString(), &roleset.AddRolesParams{
			RoleKeys: toAdd,
		})
		if err != nil {
			resp.Diagnostics.AddError("Unable to add roles to role set", err.Error())
			return
		}
	}

	if len(toRemove) > 0 {
		_, err := roleset.RemoveRoles(ctx, state.ID.ValueString(), &roleset.RemoveRolesParams{
			RoleKeys: toRemove,
		})
		if err != nil {
			resp.Diagnostics.AddError("Unable to remove roles from role set", err.Error())
			return
		}
	}

	// Re-read to get final state.
	rs, err := roleset.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read role set after update", err.Error())
		return
	}

	mapRoleSetResponseToModel(ctx, rs, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated role set", map[string]any{"id": rs.ID})
}

func (r *RoleSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleSetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := roleset.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete role set",
			fmt.Sprintf("Could not delete role set ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted role set", map[string]any{"id": state.ID.ValueString()})
}

func (r *RoleSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapRoleSetResponseToModel(ctx context.Context, rs *clerkgo.RoleSet, model *RoleSetResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(rs.ID)
	model.Name = types.StringValue(rs.Name)
	model.Key = types.StringValue(rs.Key)
	model.Type = types.StringValue(rs.Type)
	model.CreatedAt = types.StringValue(millisToRFC3339(rs.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(rs.UpdatedAt))

	if rs.Description != nil {
		model.Description = types.StringValue(*rs.Description)
	}

	if len(rs.Roles) > 0 {
		roleKeys := make([]string, len(rs.Roles))
		for i, role := range rs.Roles {
			roleKeys[i] = role.Key
		}
		roleList, d := types.ListValueFrom(ctx, types.StringType, roleKeys)
		diags.Append(d...)
		model.Roles = roleList
	} else if !model.Roles.IsNull() {
		model.Roles, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}
}

// diffStringSlices returns elements to add (in new but not old) and remove (in old but not new).
func diffStringSlices(old, new []string) (toAdd, toRemove []string) {
	oldSet := make(map[string]bool, len(old))
	for _, s := range old {
		oldSet[s] = true
	}
	newSet := make(map[string]bool, len(new))
	for _, s := range new {
		newSet[s] = true
	}

	for _, s := range new {
		if !oldSet[s] {
			toAdd = append(toAdd, s)
		}
	}
	for _, s := range old {
		if !newSet[s] {
			toRemove = append(toRemove, s)
		}
	}
	return
}
