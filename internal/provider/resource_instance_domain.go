package provider

import (
	"context"
	"fmt"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/domain"
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
	_ resource.Resource                = &InstanceDomainResource{}
	_ resource.ResourceWithConfigure   = &InstanceDomainResource{}
	_ resource.ResourceWithImportState = &InstanceDomainResource{}
)

func NewInstanceDomainResource() resource.Resource {
	return &InstanceDomainResource{}
}

type InstanceDomainResource struct {
	configured bool
}

type InstanceDomainResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	IsSatellite       types.Bool   `tfsdk:"is_satellite"`
	ProxyURL          types.String `tfsdk:"proxy_url"`
	FrontendAPIURL    types.String `tfsdk:"frontend_api_url"`
	AccountsPortalURL types.String `tfsdk:"accounts_portal_url"`
}

func (r *InstanceDomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *InstanceDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_domain"
}

func (r *InstanceDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a domain on a Clerk instance via the Instance API (Clerk Go SDK). For managing domains via the Platform API, use clerk_domain instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The domain name (e.g. app.example.com).",
			},
			"is_satellite": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this is a satellite domain. Create-only; changing forces replacement.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
					boolplanmodifier.RequiresReplace(),
				},
			},
			"proxy_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The proxy URL for the domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"frontend_api_url": schema.StringAttribute{
				Computed:    true,
				Description: "The frontend API URL for this domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"accounts_portal_url": schema.StringAttribute{
				Computed:    true,
				Description: "The accounts portal URL for this domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *InstanceDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceDomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &domain.CreateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
	}

	if !plan.IsSatellite.IsNull() && !plan.IsSatellite.IsUnknown() {
		params.IsSatellite = clerkgo.Bool(plan.IsSatellite.ValueBool())
	}
	if !plan.ProxyURL.IsNull() && !plan.ProxyURL.IsUnknown() {
		params.ProxyURL = clerkgo.String(plan.ProxyURL.ValueString())
	}

	d, err := domain.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create instance domain", err.Error())
		return
	}

	mapInstanceDomainResponseToModel(d, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created instance domain", map[string]any{"id": d.ID})
}

func (r *InstanceDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceDomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The Instance API has no Get endpoint for domains; use List and filter.
	list, err := domain.List(ctx, &domain.ListParams{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read instance domain",
			fmt.Sprintf("Could not list domains: %s", err.Error()),
		)
		return
	}

	var found *clerkgo.Domain
	for _, d := range list.Domains {
		if d.ID == state.ID.ValueString() {
			found = d
			break
		}
	}

	if found == nil {
		// Domain no longer exists; remove from state.
		resp.State.RemoveResource(ctx)
		return
	}

	mapInstanceDomainResponseToModel(found, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *InstanceDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InstanceDomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state InstanceDomainResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &domain.UpdateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
	}

	if !plan.ProxyURL.IsNull() && !plan.ProxyURL.IsUnknown() {
		params.ProxyURL = clerkgo.String(plan.ProxyURL.ValueString())
	}

	d, err := domain.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update instance domain", err.Error())
		return
	}

	mapInstanceDomainResponseToModel(d, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated instance domain", map[string]any{"id": d.ID})
}

func (r *InstanceDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceDomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := domain.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete instance domain",
			fmt.Sprintf("Could not delete domain ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted instance domain", map[string]any{"id": state.ID.ValueString()})
}

func (r *InstanceDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapInstanceDomainResponseToModel(d *clerkgo.Domain, model *InstanceDomainResourceModel) {
	model.ID = types.StringValue(d.ID)
	model.Name = types.StringValue(d.Name)
	model.IsSatellite = types.BoolValue(d.IsSatellite)
	model.FrontendAPIURL = types.StringValue(d.FrontendAPIURL)

	if d.ProxyURL != nil {
		model.ProxyURL = types.StringValue(*d.ProxyURL)
	} else {
		model.ProxyURL = types.StringNull()
	}

	if d.AccountPortalURL != nil {
		model.AccountsPortalURL = types.StringValue(*d.AccountPortalURL)
	} else {
		model.AccountsPortalURL = types.StringNull()
	}
}
