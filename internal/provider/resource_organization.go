package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/organization"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	_ resource.Resource                = &OrganizationResource{}
	_ resource.ResourceWithConfigure   = &OrganizationResource{}
	_ resource.ResourceWithImportState = &OrganizationResource{}
)

func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

type OrganizationResource struct {
	configured bool
}

type OrganizationResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Slug                  types.String `tfsdk:"slug"`
	CreatedBy             types.String `tfsdk:"created_by"`
	MaxAllowedMemberships types.Int64  `tfsdk:"max_allowed_memberships"`
	AdminDeleteEnabled    types.Bool   `tfsdk:"admin_delete_enabled"`
	PublicMetadata        types.String `tfsdk:"public_metadata"`
	PrivateMetadata       types.String `tfsdk:"private_metadata"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
}

func (r *OrganizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *OrganizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the organization.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the organization.",
			},
			"slug": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "URL-friendly identifier for the organization. Auto-generated from name if not provided. Requires \"Enable organization slugs\" to be turned on in the Clerk Dashboard under Configure > Organization settings.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Optional:    true,
				Description: "ID of the user who creates the organization. The user becomes an admin member. Only settable at creation time.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"max_allowed_memberships": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Maximum number of memberships allowed for the organization.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"admin_delete_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether administrators can delete the organization.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"public_metadata": schema.StringAttribute{
				Optional:    true,
				Description: "Public metadata as a JSON string. Accessible from both the frontend and backend.",
			},
			"private_metadata": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Private metadata as a JSON string. Only accessible from the backend.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the organization was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the organization was last updated.",
			},
		},
	}
}

func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organization.CreateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
	}

	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		params.Slug = clerkgo.String(plan.Slug.ValueString())
	}
	if !plan.CreatedBy.IsNull() && !plan.CreatedBy.IsUnknown() {
		params.CreatedBy = clerkgo.String(plan.CreatedBy.ValueString())
	}
	if !plan.MaxAllowedMemberships.IsNull() && !plan.MaxAllowedMemberships.IsUnknown() {
		params.MaxAllowedMemberships = clerkgo.Int64(plan.MaxAllowedMemberships.ValueInt64())
	}
	if !plan.PublicMetadata.IsNull() && !plan.PublicMetadata.IsUnknown() {
		raw := json.RawMessage(plan.PublicMetadata.ValueString())
		params.PublicMetadata = &raw
	}
	if !plan.PrivateMetadata.IsNull() && !plan.PrivateMetadata.IsUnknown() {
		raw := json.RawMessage(plan.PrivateMetadata.ValueString())
		params.PrivateMetadata = &raw
	}

	org, err := organization.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create organization", err.Error())
		return
	}

	mapOrgResponseToModel(org, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created organization", map[string]any{"id": org.ID})
}

func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := organization.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read organization",
			fmt.Sprintf("Could not read organization ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapOrgResponseToModel(org, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrganizationResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organization.UpdateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
	}

	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		params.Slug = clerkgo.String(plan.Slug.ValueString())
	}
	if !plan.MaxAllowedMemberships.IsNull() && !plan.MaxAllowedMemberships.IsUnknown() {
		params.MaxAllowedMemberships = clerkgo.Int64(plan.MaxAllowedMemberships.ValueInt64())
	}
	if !plan.AdminDeleteEnabled.IsNull() && !plan.AdminDeleteEnabled.IsUnknown() {
		params.AdminDeleteEnabled = clerkgo.Bool(plan.AdminDeleteEnabled.ValueBool())
	}
	if !plan.PublicMetadata.IsNull() && !plan.PublicMetadata.IsUnknown() {
		raw := json.RawMessage(plan.PublicMetadata.ValueString())
		params.PublicMetadata = &raw
	}
	if !plan.PrivateMetadata.IsNull() && !plan.PrivateMetadata.IsUnknown() {
		raw := json.RawMessage(plan.PrivateMetadata.ValueString())
		params.PrivateMetadata = &raw
	}

	org, err := organization.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update organization", err.Error())
		return
	}

	mapOrgResponseToModel(org, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated organization", map[string]any{"id": org.ID})
}

func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := organization.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete organization",
			fmt.Sprintf("Could not delete organization ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted organization", map[string]any{"id": state.ID.ValueString()})
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapOrgResponseToModel(org *clerkgo.Organization, model *OrganizationResourceModel) {
	model.ID = types.StringValue(org.ID)
	model.Name = types.StringValue(org.Name)
	model.Slug = types.StringValue(org.Slug)
	model.MaxAllowedMemberships = types.Int64Value(org.MaxAllowedMemberships)
	model.AdminDeleteEnabled = types.BoolValue(org.AdminDeleteEnabled)
	model.CreatedAt = types.StringValue(time.UnixMilli(org.CreatedAt).UTC().Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(time.UnixMilli(org.UpdatedAt).UTC().Format(time.RFC3339))

	if org.CreatedBy != "" {
		model.CreatedBy = types.StringValue(org.CreatedBy)
	}

	if len(org.PublicMetadata) > 0 && string(org.PublicMetadata) != "{}" {
		model.PublicMetadata = types.StringValue(normalizeJSON(string(org.PublicMetadata)))
	}
	if len(org.PrivateMetadata) > 0 && string(org.PrivateMetadata) != "{}" {
		model.PrivateMetadata = types.StringValue(normalizeJSON(string(org.PrivateMetadata)))
	}
}
