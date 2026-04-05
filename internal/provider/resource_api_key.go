package provider

import (
	"context"
	"encoding/json"
	"fmt"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/apikey"
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
	_ resource.Resource                = &APIKeyResource{}
	_ resource.ResourceWithConfigure   = &APIKeyResource{}
	_ resource.ResourceWithImportState = &APIKeyResource{}
)

func NewAPIKeyResource() resource.Resource {
	return &APIKeyResource{}
}

type APIKeyResource struct {
	configured bool
}

type APIKeyResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Type                   types.String `tfsdk:"type"`
	Subject                types.String `tfsdk:"subject"`
	Description            types.String `tfsdk:"description"`
	Claims                 types.String `tfsdk:"claims"`
	Scopes                 types.List   `tfsdk:"scopes"`
	CreatedBy              types.String `tfsdk:"created_by"`
	SecondsUntilExpiration types.Int64  `tfsdk:"seconds_until_expiration"`
	Secret                 types.String `tfsdk:"secret"`
	Revoked                types.Bool   `tfsdk:"revoked"`
	Expired                types.Bool   `tfsdk:"expired"`
	Expiration             types.String `tfsdk:"expiration"`
	LastUsedAt             types.String `tfsdk:"last_used_at"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

func (r *APIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *APIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the API key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the API key.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Type of the API key. Cannot be changed after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subject": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Subject of the API key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Description of the API key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"claims": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "JSON string of custom claims for the API key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"scopes": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of scopes for the API key.",
			},
			"created_by": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Identifier of the entity that created the API key. Set on create only.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"seconds_until_expiration": schema.Int64Attribute{
				Optional:    true,
				Description: "Number of seconds until the API key expires. Write-only: used on create/update, the API returns an expiration timestamp instead.",
			},
			"secret": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The API key secret. Write-only: only returned on creation, not on subsequent reads.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"revoked": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the API key has been revoked.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"expired": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the API key has expired.",
			},
			"expiration": schema.StringAttribute{
				Computed:    true,
				Description: "Expiration timestamp in RFC3339 format.",
			},
			"last_used_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the API key was last used, in RFC3339 format.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the API key was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the API key was last updated.",
			},
		},
	}
}

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan APIKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &apikey.CreateParams{
		Name: clerkgo.String(plan.Name.ValueString()),
	}

	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		params.Type = clerkgo.String(plan.Type.ValueString())
	}
	if !plan.Subject.IsNull() && !plan.Subject.IsUnknown() {
		params.Subject = clerkgo.String(plan.Subject.ValueString())
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}
	if !plan.Claims.IsNull() && !plan.Claims.IsUnknown() {
		params.Claims = json.RawMessage(plan.Claims.ValueString())
	}
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		var scopes []string
		diags = plan.Scopes.ElementsAs(ctx, &scopes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.Scopes = scopes
	}
	if !plan.CreatedBy.IsNull() && !plan.CreatedBy.IsUnknown() {
		params.CreatedBy = clerkgo.String(plan.CreatedBy.ValueString())
	}
	if !plan.SecondsUntilExpiration.IsNull() && !plan.SecondsUntilExpiration.IsUnknown() {
		params.SecondsUntilExpiration = clerkgo.Int64(plan.SecondsUntilExpiration.ValueInt64())
	}

	key, err := apikey.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create API key", err.Error())
		return
	}

	mapAPIKeyResponseToModel(ctx, &key.APIKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// secret is only returned on creation
	plan.Secret = types.StringValue(key.Secret)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created API key", map[string]any{"id": key.ID})
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state APIKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := apikey.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read API key",
			fmt.Sprintf("Could not read API key ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapAPIKeyResponseToModel(ctx, key, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan APIKeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state APIKeyResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &apikey.UpdateParams{}

	if !plan.Subject.IsNull() && !plan.Subject.IsUnknown() {
		params.Subject = clerkgo.String(plan.Subject.ValueString())
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerkgo.String(plan.Description.ValueString())
	}
	if !plan.Claims.IsNull() && !plan.Claims.IsUnknown() {
		params.Claims = json.RawMessage(plan.Claims.ValueString())
	}
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		var scopes []string
		diags = plan.Scopes.ElementsAs(ctx, &scopes, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.Scopes = scopes
	}
	if !plan.SecondsUntilExpiration.IsNull() && !plan.SecondsUntilExpiration.IsUnknown() {
		params.SecondsUntilExpiration = clerkgo.Int64(plan.SecondsUntilExpiration.ValueInt64())
	}

	key, err := apikey.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update API key", err.Error())
		return
	}

	mapAPIKeyResponseToModel(ctx, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve secret from state — API does not return it on update
	plan.Secret = state.Secret

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated API key", map[string]any{"id": key.ID})
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state APIKeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := apikey.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete API key",
			fmt.Sprintf("Could not delete API key ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted API key", map[string]any{"id": state.ID.ValueString()})
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapAPIKeyResponseToModel(ctx context.Context, key *clerkgo.APIKey, model *APIKeyResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(key.ID)
	model.Name = types.StringValue(key.Name)
	model.Type = types.StringValue(key.Type)
	model.Subject = types.StringValue(key.Subject)

	if key.Description != nil {
		model.Description = types.StringValue(*key.Description)
	}

	if len(key.Claims) > 0 {
		normalized := normalizeJSON(string(key.Claims))
		model.Claims = types.StringValue(normalized)
	}

	if len(key.Scopes) > 0 {
		scopeList, d := types.ListValueFrom(ctx, types.StringType, key.Scopes)
		diags.Append(d...)
		model.Scopes = scopeList
	} else if !model.Scopes.IsNull() {
		model.Scopes, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	model.Revoked = types.BoolValue(key.Revoked)
	model.Expired = types.BoolValue(key.Expired)

	if key.Expiration != nil {
		model.Expiration = types.StringValue(millisToRFC3339(*key.Expiration))
	} else {
		model.Expiration = types.StringNull()
	}

	if key.CreatedBy != nil {
		model.CreatedBy = types.StringValue(*key.CreatedBy)
	}

	if key.LastUsedAt != nil {
		model.LastUsedAt = types.StringValue(millisToRFC3339(*key.LastUsedAt))
	} else {
		model.LastUsedAt = types.StringNull()
	}

	model.CreatedAt = types.StringValue(millisToRFC3339(key.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(key.UpdatedAt))
	// NOTE: secret is intentionally NOT set here.
	// It is only returned on creation and must be preserved from state.
}
