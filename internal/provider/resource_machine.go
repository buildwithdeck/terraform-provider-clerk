package provider

import (
	"context"
	"fmt"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/machine"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &MachineResource{}
	_ resource.ResourceWithConfigure   = &MachineResource{}
	_ resource.ResourceWithImportState = &MachineResource{}
)

func NewMachineResource() resource.Resource {
	return &MachineResource{}
}

type MachineResource struct {
	configured bool
}

type MachineResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	DefaultTokenTTL types.Int64  `tfsdk:"default_token_ttl"`
	SecretKey       types.String `tfsdk:"secret_key"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

func (r *MachineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MachineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (r *MachineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk machine.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the machine.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the machine.",
			},
			"default_token_ttl": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Default token TTL in seconds for this machine.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"secret_key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Machine secret key. Write-only: only returned on creation, not on subsequent reads.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the machine was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the machine was last updated.",
			},
		},
	}
}

func (r *MachineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MachineResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &machine.CreateParams{
		Name: plan.Name.ValueString(),
	}

	if !plan.DefaultTokenTTL.IsNull() && !plan.DefaultTokenTTL.IsUnknown() {
		params.DefaultTokenTTL = clerkgo.Int64(plan.DefaultTokenTTL.ValueInt64())
	}

	result, err := machine.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create machine", err.Error())
		return
	}

	mapMachineResponseToModel(&result.MachineWithScopedMachines, &plan)

	// secret_key is only returned on creation — set it from the response
	plan.SecretKey = types.StringValue(result.SecretKey)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created machine", map[string]any{"id": result.ID})
}

func (r *MachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MachineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := machine.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read machine",
			fmt.Sprintf("Could not read machine ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapMachineResponseToModel(result, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *MachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MachineResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state MachineResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &machine.UpdateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
	}

	if !plan.DefaultTokenTTL.IsNull() && !plan.DefaultTokenTTL.IsUnknown() {
		params.DefaultTokenTTL = clerkgo.Int64(plan.DefaultTokenTTL.ValueInt64())
	}

	result, err := machine.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update machine", err.Error())
		return
	}

	mapMachineResponseToModel(result, &plan)

	// Preserve secret_key from state — API does not return it on update
	plan.SecretKey = state.SecretKey

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated machine", map[string]any{"id": result.ID})
}

func (r *MachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MachineResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := machine.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete machine",
			fmt.Sprintf("Could not delete machine ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted machine", map[string]any{"id": state.ID.ValueString()})
}

func (r *MachineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapMachineResponseToModel(m *clerkgo.MachineWithScopedMachines, model *MachineResourceModel) {
	model.ID = types.StringValue(m.ID)
	model.Name = types.StringValue(m.Name)
	model.DefaultTokenTTL = types.Int64Value(m.DefaultTokenTTL)
	model.CreatedAt = types.StringValue(millisToRFC3339(m.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(m.UpdatedAt))
	// NOTE: secret_key is intentionally NOT set here.
	// It is only returned on creation and must be preserved from state.
}
