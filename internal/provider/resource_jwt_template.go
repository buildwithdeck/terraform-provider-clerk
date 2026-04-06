package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwttemplate"
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
	_ resource.Resource                = &JWTTemplateResource{}
	_ resource.ResourceWithConfigure   = &JWTTemplateResource{}
	_ resource.ResourceWithImportState = &JWTTemplateResource{}
)

func NewJWTTemplateResource() resource.Resource {
	return &JWTTemplateResource{}
}

type JWTTemplateResource struct {
	client *jwttemplate.Client
}

type JWTTemplateResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Claims           types.String `tfsdk:"claims"`
	Lifetime         types.Int64  `tfsdk:"lifetime"`
	AllowedClockSkew types.Int64  `tfsdk:"allowed_clock_skew"`
	CustomSigningKey types.Bool   `tfsdk:"custom_signing_key"`
	SigningKey       types.String `tfsdk:"signing_key"`
	SigningAlgorithm types.String `tfsdk:"signing_algorithm"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *JWTTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The clerk_jwt_template resource requires an api_key. Set it in the provider configuration or via the CLERK_API_KEY environment variable.",
		)
		return
	}
	r.client = jwttemplate.NewClient(&clerkgo.ClientConfig{
		BackendConfig: clerkgo.BackendConfig{
			Key: clerkgo.String(data.APIKey),
		},
	})
}

func (r *JWTTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jwt_template"
}

func (r *JWTTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk JWT template.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the JWT template.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the JWT template.",
			},
			"claims": schema.StringAttribute{
				Required:    true,
				Description: "JSON string of custom claims for the JWT.",
			},
			"lifetime": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Token lifetime in seconds.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"allowed_clock_skew": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Allowed clock skew in seconds.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"custom_signing_key": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to use a custom signing key.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"signing_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Custom signing key. Write-only: not returned by the API on reads.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"signing_algorithm": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Signing algorithm (e.g. RS256, HS256).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the template was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the template was last updated.",
			},
		},
	}
}

func (r *JWTTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan JWTTemplateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &jwttemplate.CreateParams{
		Name:   clerkgo.String(plan.Name.ValueString()),
		Claims: json.RawMessage(plan.Claims.ValueString()),
	}

	if !plan.Lifetime.IsNull() && !plan.Lifetime.IsUnknown() {
		params.Lifetime = clerkgo.Int64(plan.Lifetime.ValueInt64())
	}
	if !plan.AllowedClockSkew.IsNull() && !plan.AllowedClockSkew.IsUnknown() {
		params.AllowedClockSkew = clerkgo.Int64(plan.AllowedClockSkew.ValueInt64())
	}
	if !plan.CustomSigningKey.IsNull() && !plan.CustomSigningKey.IsUnknown() {
		params.CustomSigningKey = clerkgo.Bool(plan.CustomSigningKey.ValueBool())
	}
	if !plan.SigningKey.IsNull() && !plan.SigningKey.IsUnknown() {
		params.SigningKey = clerkgo.String(plan.SigningKey.ValueString())
	}
	if !plan.SigningAlgorithm.IsNull() && !plan.SigningAlgorithm.IsUnknown() {
		params.SigningAlgorithm = clerkgo.String(plan.SigningAlgorithm.ValueString())
	}

	tmpl, err := r.client.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create JWT template", err.Error())
		return
	}

	mapResponseToModel(tmpl, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created JWT template", map[string]any{"id": tmpl.ID})
}

func (r *JWTTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state JWTTemplateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tmpl, err := r.client.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read JWT template",
			fmt.Sprintf("Could not read JWT template ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapResponseToModel(tmpl, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *JWTTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan JWTTemplateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state JWTTemplateResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &jwttemplate.UpdateParams{
		Name:   clerkgo.String(plan.Name.ValueString()),
		Claims: json.RawMessage(plan.Claims.ValueString()),
	}

	if !plan.Lifetime.IsNull() && !plan.Lifetime.IsUnknown() {
		params.Lifetime = clerkgo.Int64(plan.Lifetime.ValueInt64())
	}
	if !plan.AllowedClockSkew.IsNull() && !plan.AllowedClockSkew.IsUnknown() {
		params.AllowedClockSkew = clerkgo.Int64(plan.AllowedClockSkew.ValueInt64())
	}
	if !plan.CustomSigningKey.IsNull() && !plan.CustomSigningKey.IsUnknown() {
		params.CustomSigningKey = clerkgo.Bool(plan.CustomSigningKey.ValueBool())
	}
	if !plan.SigningKey.IsNull() && !plan.SigningKey.IsUnknown() {
		params.SigningKey = clerkgo.String(plan.SigningKey.ValueString())
	}
	if !plan.SigningAlgorithm.IsNull() && !plan.SigningAlgorithm.IsUnknown() {
		params.SigningAlgorithm = clerkgo.String(plan.SigningAlgorithm.ValueString())
	}

	tmpl, err := r.client.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update JWT template", err.Error())
		return
	}

	mapResponseToModel(tmpl, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated JWT template", map[string]any{"id": tmpl.ID})
}

func (r *JWTTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state JWTTemplateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete JWT template",
			fmt.Sprintf("Could not delete JWT template ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted JWT template", map[string]any{"id": state.ID.ValueString()})
}

func (r *JWTTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapResponseToModel(tmpl *clerkgo.JWTTemplate, model *JWTTemplateResourceModel) {
	model.ID = types.StringValue(tmpl.ID)
	model.Name = types.StringValue(tmpl.Name)
	model.Claims = types.StringValue(normalizeJSON(string(tmpl.Claims)))
	model.Lifetime = types.Int64Value(tmpl.Lifetime)
	model.AllowedClockSkew = types.Int64Value(tmpl.AllowedClockSkew)
	model.CustomSigningKey = types.BoolValue(tmpl.CustomSigningKey)
	model.SigningAlgorithm = types.StringValue(tmpl.SigningAlgorithm)
	model.CreatedAt = types.StringValue(time.UnixMilli(tmpl.CreatedAt).UTC().Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(time.UnixMilli(tmpl.UpdatedAt).UTC().Format(time.RFC3339))
	// signing_key is write-only — preserve whatever is in state
}
