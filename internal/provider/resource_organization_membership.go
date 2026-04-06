package provider

import (
	"context"
	"fmt"
	"strings"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/organizationmembership"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &OrganizationMembershipResource{}
	_ resource.ResourceWithConfigure   = &OrganizationMembershipResource{}
	_ resource.ResourceWithImportState = &OrganizationMembershipResource{}
)

func NewOrganizationMembershipResource() resource.Resource {
	return &OrganizationMembershipResource{}
}

type OrganizationMembershipResource struct {
	configured bool
}

type OrganizationMembershipResourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	UserID         types.String `tfsdk:"user_id"`
	Role           types.String `tfsdk:"role"`
	RoleName       types.String `tfsdk:"role_name"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *OrganizationMembershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The clerk_organization_membership resource requires an api_key. Set it in the provider configuration or via the CLERK_API_KEY environment variable.",
		)
		return
	}
	r.configured = true
}

func (r *OrganizationMembershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_membership"
}

func (r *OrganizationMembershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk organization membership.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the organization membership.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the organization.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the user to add as a member.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Required:    true,
				Description: "Role key to assign to the member (e.g. org:admin, org:member).",
			},
			"role_name": schema.StringAttribute{
				Computed:    true,
				Description: "Display name of the assigned role.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the membership was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the membership was last updated.",
			},
		},
	}
}

func (r *OrganizationMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationMembershipResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationmembership.CreateParams{
		OrganizationID: plan.OrganizationID.ValueString(),
		UserID:         clerkgo.String(plan.UserID.ValueString()),
		Role:           clerkgo.String(plan.Role.ValueString()),
	}

	membership, err := organizationmembership.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create organization membership", err.Error())
		return
	}

	mapOrganizationMembershipResponseToModel(membership, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created organization membership", map[string]any{"id": membership.ID})
}

func (r *OrganizationMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationMembershipResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No Get endpoint — use List with UserIDs filter.
	list, err := organizationmembership.List(ctx, &organizationmembership.ListParams{
		OrganizationID: state.OrganizationID.ValueString(),
		UserIDs:        []string{state.UserID.ValueString()},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read organization membership",
			fmt.Sprintf("Could not list memberships for org %s: %s", state.OrganizationID.ValueString(), err.Error()),
		)
		return
	}

	var found *clerkgo.OrganizationMembership
	for _, m := range list.OrganizationMemberships {
		if m.PublicUserData != nil && m.PublicUserData.UserID == state.UserID.ValueString() {
			found = m
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapOrganizationMembershipResponseToModel(found, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OrganizationMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationMembershipResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrganizationMembershipResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationmembership.UpdateParams{
		OrganizationID: state.OrganizationID.ValueString(),
		UserID:         state.UserID.ValueString(),
		Role:           clerkgo.String(plan.Role.ValueString()),
	}

	membership, err := organizationmembership.Update(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update organization membership", err.Error())
		return
	}

	mapOrganizationMembershipResponseToModel(membership, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated organization membership", map[string]any{"id": membership.ID})
}

func (r *OrganizationMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationMembershipResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := organizationmembership.Delete(ctx, &organizationmembership.DeleteParams{
		OrganizationID: state.OrganizationID.ValueString(),
		UserID:         state.UserID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete organization membership",
			fmt.Sprintf("Could not delete membership for user %s in org %s: %s",
				state.UserID.ValueString(), state.OrganizationID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted organization membership", map[string]any{"id": state.ID.ValueString()})
}

func (r *OrganizationMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format: organization_id/user_id",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
}

func mapOrganizationMembershipResponseToModel(m *clerkgo.OrganizationMembership, model *OrganizationMembershipResourceModel) {
	model.ID = types.StringValue(m.ID)
	model.Role = types.StringValue(m.Role)
	model.RoleName = types.StringValue(m.RoleName)
	model.CreatedAt = types.StringValue(millisToRFC3339(m.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(m.UpdatedAt))

	if m.Organization != nil {
		model.OrganizationID = types.StringValue(m.Organization.ID)
	}
	if m.PublicUserData != nil {
		model.UserID = types.StringValue(m.PublicUserData.UserID)
	}
}
