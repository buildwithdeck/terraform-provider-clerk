package provider

import (
	"context"
	"fmt"
	"strings"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/organizationdomain"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &OrganizationDomainResource{}
	_ resource.ResourceWithConfigure   = &OrganizationDomainResource{}
	_ resource.ResourceWithImportState = &OrganizationDomainResource{}
)

func NewOrganizationDomainResource() resource.Resource {
	return &OrganizationDomainResource{}
}

type OrganizationDomainResource struct {
	client *organizationdomain.Client
}

type OrganizationDomainResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	OrganizationID          types.String `tfsdk:"organization_id"`
	Name                    types.String `tfsdk:"name"`
	EnrollmentMode          types.String `tfsdk:"enrollment_mode"`
	Verified                types.Bool   `tfsdk:"verified"`
	AffiliationEmailAddress types.String `tfsdk:"affiliation_email_address"`
	VerificationStatus      types.String `tfsdk:"verification_status"`
	CreatedAt               types.String `tfsdk:"created_at"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
}

func (r *OrganizationDomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The clerk_organization_domain resource requires an api_key. Set it in the provider configuration or via the CLERK_API_KEY environment variable.",
		)
		return
	}
	r.client = organizationdomain.NewClient(&clerkgo.ClientConfig{
		BackendConfig: clerkgo.BackendConfig{
			Key: clerkgo.String(data.APIKey),
		},
	})
}

func (r *OrganizationDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_domain"
}

func (r *OrganizationDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk organization domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the organization domain.",
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
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The domain name (e.g. example.com).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enrollment_mode": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enrollment mode for the domain (e.g. automatic_invitation, automatic_suggestion, manual_invitation).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"verified": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether the domain should be verified. Only settable at creation time.",
			},
			"affiliation_email_address": schema.StringAttribute{
				Computed:    true,
				Description: "Email address used for domain affiliation.",
			},
			"verification_status": schema.StringAttribute{
				Computed:    true,
				Description: "Current verification status of the domain.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the domain was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the domain was last updated.",
			},
		},
	}
}

func (r *OrganizationDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationDomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationdomain.CreateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
	}

	if !plan.EnrollmentMode.IsNull() && !plan.EnrollmentMode.IsUnknown() {
		params.EnrollmentMode = clerkgo.String(plan.EnrollmentMode.ValueString())
	}

	if !plan.Verified.IsNull() && !plan.Verified.IsUnknown() {
		params.Verified = clerkgo.Bool(plan.Verified.ValueBool())
	}

	domain, err := r.client.Create(ctx, plan.OrganizationID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create organization domain", err.Error())
		return
	}

	mapOrganizationDomainResponseToModel(domain, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created organization domain", map[string]any{"id": domain.ID})
}

func (r *OrganizationDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrganizationDomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No Get endpoint — use List and filter by ID.
	list, err := r.client.List(ctx, state.OrganizationID.ValueString(), &organizationdomain.ListParams{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read organization domain",
			fmt.Sprintf("Could not list domains for org %s: %s", state.OrganizationID.ValueString(), err.Error()),
		)
		return
	}

	var found *clerkgo.OrganizationDomain
	for _, d := range list.OrganizationDomains {
		if d.ID == state.ID.ValueString() {
			found = d
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapOrganizationDomainResponseToModel(found, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OrganizationDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationDomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrganizationDomainResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &organizationdomain.UpdateParams{
		OrganizationID: state.OrganizationID.ValueString(),
		DomainID:       state.ID.ValueString(),
	}

	if !plan.EnrollmentMode.IsNull() && !plan.EnrollmentMode.IsUnknown() {
		params.EnrollmentMode = clerkgo.String(plan.EnrollmentMode.ValueString())
	}

	domain, err := r.client.Update(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update organization domain", err.Error())
		return
	}

	mapOrganizationDomainResponseToModel(domain, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated organization domain", map[string]any{"id": domain.ID})
}

func (r *OrganizationDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationDomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, &organizationdomain.DeleteParams{
		OrganizationID: state.OrganizationID.ValueString(),
		DomainID:       state.ID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete organization domain",
			fmt.Sprintf("Could not delete domain %s in org %s: %s",
				state.ID.ValueString(), state.OrganizationID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted organization domain", map[string]any{"id": state.ID.ValueString()})
}

func (r *OrganizationDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format: organization_id/domain_id",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func mapOrganizationDomainResponseToModel(d *clerkgo.OrganizationDomain, model *OrganizationDomainResourceModel) {
	model.ID = types.StringValue(d.ID)
	model.OrganizationID = types.StringValue(d.OrganizationID)
	model.Name = types.StringValue(d.Name)
	model.EnrollmentMode = types.StringValue(d.EnrollmentMode)
	model.CreatedAt = types.StringValue(millisToRFC3339(d.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(d.UpdatedAt))

	if d.AffiliationEmailAddress != nil {
		model.AffiliationEmailAddress = types.StringValue(*d.AffiliationEmailAddress)
	}

	if d.Verification != nil {
		model.VerificationStatus = types.StringValue(d.Verification.Status)
	}
}
