package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &ApplicationTransferResource{}
	_ resource.ResourceWithConfigure   = &ApplicationTransferResource{}
	_ resource.ResourceWithImportState = &ApplicationTransferResource{}
)

func NewApplicationTransferResource() resource.Resource {
	return &ApplicationTransferResource{}
}

type ApplicationTransferResource struct {
	client *PlatformClient
}

type ApplicationTransferResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	Status        types.String `tfsdk:"status"`
	Code          types.String `tfsdk:"code"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
	CreatedAt     types.String `tfsdk:"created_at"`
	CanceledAt    types.String `tfsdk:"canceled_at"`
	CompletedAt   types.String `tfsdk:"completed_at"`
}

func (r *ApplicationTransferResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The clerk_application_transfer resource requires a platform_api_key. Set it in the provider configuration or via the CLERK_PLATFORM_API_KEY environment variable. The Platform API is a beta feature that must be enabled by Clerk — contact Clerk support or visit your dashboard to request access.",
		)
		return
	}
	r.client = NewPlatformClient(data.PlatformAPIKey)
}

func (r *ApplicationTransferResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_transfer"
}

func (r *ApplicationTransferResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Initiates a transfer of a Clerk application to another owner via the Platform API. The Platform API is a beta feature that must be enabled by Clerk — contact Clerk support or visit your dashboard to request access.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the transfer.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the application to transfer. Changing forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Current status of the transfer (pending, completed, canceled, expired).",
			},
			"code": schema.StringAttribute{
				Computed:    true,
				Description: "The transfer code used to complete the transfer.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"expires_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the transfer expires.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the transfer was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"canceled_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the transfer was canceled, if applicable.",
			},
			"completed_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the transfer was completed, if applicable.",
			},
		},
	}
}

func (r *ApplicationTransferResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationTransferResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	transfer, err := r.client.CreateTransfer(ctx, plan.ApplicationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create application transfer", err.Error())
		return
	}

	mapTransferResponseToModel(transfer, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created application transfer", map[string]any{"id": transfer.ID})
}

func (r *ApplicationTransferResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationTransferResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	transfer, err := r.client.GetTransfer(ctx, state.ApplicationID.ValueString(), state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*PlatformAPIError); ok && apiErr.StatusCode == 404 {
			tflog.Debug(ctx, "Application transfer not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to read application transfer",
			fmt.Sprintf("Could not read transfer ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	// If the transfer has expired, remove it from state.
	if transfer.Status == "expired" {
		tflog.Debug(ctx, "Application transfer expired, removing from state", map[string]any{"id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	// Preserve application_id from state.
	appID := state.ApplicationID
	mapTransferResponseToModel(transfer, &state)
	state.ApplicationID = appID

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *ApplicationTransferResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Application transfers cannot be updated. All attribute changes require replacement.",
	)
}

func (r *ApplicationTransferResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationTransferResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.CancelTransfer(ctx, state.ApplicationID.ValueString(), state.ID.ValueString())
	if err != nil {
		if apiErr, ok := err.(*PlatformAPIError); ok {
			// Treat 404 as already canceled/expired.
			if apiErr.StatusCode == 404 {
				tflog.Debug(ctx, "Application transfer already gone", map[string]any{"id": state.ID.ValueString()})
				return
			}
		}
		resp.Diagnostics.AddError(
			"Unable to cancel application transfer",
			fmt.Sprintf("Could not cancel transfer ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Canceled application transfer", map[string]any{"id": state.ID.ValueString()})
}

func (r *ApplicationTransferResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: application_id/transfer_id, got: %s", req.ID),
		)
		return
	}

	resp.State.SetAttribute(ctx, path.Root("application_id"), types.StringValue(parts[0]))
	resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(parts[1]))
}

func mapTransferResponseToModel(resp *PlatformTransferResponse, model *ApplicationTransferResourceModel) {
	model.ID = types.StringValue(resp.ID)
	model.ApplicationID = types.StringValue(resp.ApplicationID)
	model.Status = types.StringValue(resp.Status)
	model.Code = types.StringValue(resp.Code)
	model.ExpiresAt = types.StringValue(resp.ExpiresAt)
	model.CreatedAt = types.StringValue(resp.CreatedAt)

	if resp.CanceledAt != nil {
		model.CanceledAt = types.StringValue(*resp.CanceledAt)
	} else {
		model.CanceledAt = types.StringNull()
	}

	if resp.CompletedAt != nil {
		model.CompletedAt = types.StringValue(*resp.CompletedAt)
	} else {
		model.CompletedAt = types.StringNull()
	}
}
