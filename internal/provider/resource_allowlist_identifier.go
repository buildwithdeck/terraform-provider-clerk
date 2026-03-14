package provider

import (
	"context"
	"fmt"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/allowlistidentifier"
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
	_ resource.Resource                = &AllowlistIdentifierResource{}
	_ resource.ResourceWithConfigure   = &AllowlistIdentifierResource{}
	_ resource.ResourceWithImportState = &AllowlistIdentifierResource{}
)

func NewAllowlistIdentifierResource() resource.Resource {
	return &AllowlistIdentifierResource{}
}

type AllowlistIdentifierResource struct {
	configured bool
}

type AllowlistIdentifierResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Identifier types.String `tfsdk:"identifier"`
	Notify     types.Bool   `tfsdk:"notify"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func (r *AllowlistIdentifierResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AllowlistIdentifierResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allowlist_identifier"
}

func (r *AllowlistIdentifierResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk allowlist identifier. Allowlist identifiers restrict sign-ups to only the specified email addresses, phone numbers, or web3 wallets.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the allowlist entry.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identifier": schema.StringAttribute{
				Required:    true,
				Description: "The identifier to add to the allowlist. Can be an email address, phone number, or web3 wallet address. Supports wildcard patterns for email addresses (e.g. *@example.com).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"notify": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to send an invitation email to the identifier upon creation. Only applies to email address identifiers.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the allowlist entry was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the allowlist entry was last updated.",
			},
		},
	}
}

func (r *AllowlistIdentifierResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AllowlistIdentifierResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &allowlistidentifier.CreateParams{
		Identifier: clerkgo.String(plan.Identifier.ValueString()),
	}

	if !plan.Notify.IsNull() && !plan.Notify.IsUnknown() {
		params.Notify = clerkgo.Bool(plan.Notify.ValueBool())
	}

	entry, err := allowlistidentifier.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create allowlist identifier", err.Error())
		return
	}

	mapAllowlistIdentifierResponseToModel(entry, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created allowlist identifier", map[string]any{"id": entry.ID})
}

func (r *AllowlistIdentifierResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AllowlistIdentifierResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, err := allowlistidentifier.List(ctx, &allowlistidentifier.ListParams{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read allowlist identifiers",
			fmt.Sprintf("Could not list allowlist identifiers: %s", err.Error()),
		)
		return
	}

	var found *clerkgo.AllowlistIdentifier
	for _, entry := range list.AllowlistIdentifiers {
		if entry.ID == state.ID.ValueString() {
			found = entry
			break
		}
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapAllowlistIdentifierResponseToModel(found, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *AllowlistIdentifierResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Allowlist identifiers do not support updates. All attribute changes require resource replacement.",
	)
}

func (r *AllowlistIdentifierResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AllowlistIdentifierResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := allowlistidentifier.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete allowlist identifier",
			fmt.Sprintf("Could not delete allowlist identifier ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted allowlist identifier", map[string]any{"id": state.ID.ValueString()})
}

func (r *AllowlistIdentifierResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapAllowlistIdentifierResponseToModel(entry *clerkgo.AllowlistIdentifier, model *AllowlistIdentifierResourceModel) {
	model.ID = types.StringValue(entry.ID)
	model.Identifier = types.StringValue(entry.Identifier)
	model.CreatedAt = types.StringValue(millisToRFC3339(entry.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(entry.UpdatedAt))
	// notify is write-only — not returned by the API, preserve whatever is in state
}
