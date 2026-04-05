package provider

import (
	"context"
	"fmt"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/oauthapplication"
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
	_ resource.Resource                = &OAuthApplicationResource{}
	_ resource.ResourceWithConfigure   = &OAuthApplicationResource{}
	_ resource.ResourceWithImportState = &OAuthApplicationResource{}
)

func NewOAuthApplicationResource() resource.Resource {
	return &OAuthApplicationResource{}
}

type OAuthApplicationResource struct {
	configured bool
}

type OAuthApplicationResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	CallbackURL          types.String `tfsdk:"callback_url"`
	Scopes               types.String `tfsdk:"scopes"`
	Public               types.Bool   `tfsdk:"public"`
	ConsentScreenEnabled types.Bool   `tfsdk:"consent_screen_enabled"`
	ClientID             types.String `tfsdk:"client_id"`
	ClientSecret         types.String `tfsdk:"client_secret"`
	DiscoveryURL         types.String `tfsdk:"discovery_url"`
	AuthorizeURL         types.String `tfsdk:"authorize_url"`
	TokenFetchURL        types.String `tfsdk:"token_fetch_url"`
	UserInfoURL          types.String `tfsdk:"user_info_url"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func (r *OAuthApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OAuthApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_application"
}

func (r *OAuthApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk OAuth application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the OAuth application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the OAuth application.",
			},
			"callback_url": schema.StringAttribute{
				Required:    true,
				Description: "Callback URL for the OAuth application.",
			},
			"scopes": schema.StringAttribute{
				Required:    true,
				Description: "Space-separated OAuth scopes for the application.",
			},
			"public": schema.BoolAttribute{
				Required:    true,
				Description: "Whether this is a public OAuth application. Cannot be changed after creation.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"consent_screen_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the consent screen is enabled for this OAuth application.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"client_id": schema.StringAttribute{
				Computed:    true,
				Description: "OAuth client ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_secret": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "OAuth client secret. Write-only: only returned on creation, not on subsequent reads.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"discovery_url": schema.StringAttribute{
				Computed:    true,
				Description: "OpenID Connect discovery URL.",
			},
			"authorize_url": schema.StringAttribute{
				Computed:    true,
				Description: "OAuth authorize URL.",
			},
			"token_fetch_url": schema.StringAttribute{
				Computed:    true,
				Description: "OAuth token fetch URL.",
			},
			"user_info_url": schema.StringAttribute{
				Computed:    true,
				Description: "OAuth user info URL.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the OAuth application was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the OAuth application was last updated.",
			},
		},
	}
}

func (r *OAuthApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OAuthApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &oauthapplication.CreateParams{
		Name:        plan.Name.ValueString(),
		CallbackURL: plan.CallbackURL.ValueString(),
		Scopes:      plan.Scopes.ValueString(),
		Public:      plan.Public.ValueBool(),
	}

	if !plan.ConsentScreenEnabled.IsNull() && !plan.ConsentScreenEnabled.IsUnknown() {
		params.ConsentScreenEnabled = clerkgo.Bool(plan.ConsentScreenEnabled.ValueBool())
	}

	app, err := oauthapplication.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create OAuth application", err.Error())
		return
	}

	mapOAuthApplicationResponseToModel(app, &plan)

	// client_secret is only returned on creation — set it from the response
	if app.ClientSecret != nil {
		plan.ClientSecret = types.StringValue(*app.ClientSecret)
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created OAuth application", map[string]any{"id": app.ID})
}

func (r *OAuthApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OAuthApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := oauthapplication.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read OAuth application",
			fmt.Sprintf("Could not read OAuth application ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapOAuthApplicationResponseToModel(app, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *OAuthApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OAuthApplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OAuthApplicationResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &oauthapplication.UpdateParams{
		Name:        clerkgo.String(plan.Name.ValueString()),
		CallbackURL: clerkgo.String(plan.CallbackURL.ValueString()),
		Scopes:      clerkgo.String(plan.Scopes.ValueString()),
	}

	if !plan.ConsentScreenEnabled.IsNull() && !plan.ConsentScreenEnabled.IsUnknown() {
		params.ConsentScreenEnabled = clerkgo.Bool(plan.ConsentScreenEnabled.ValueBool())
	}

	app, err := oauthapplication.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update OAuth application", err.Error())
		return
	}

	mapOAuthApplicationResponseToModel(app, &plan)

	// Preserve client_secret from state — API does not return it on update
	plan.ClientSecret = state.ClientSecret

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated OAuth application", map[string]any{"id": app.ID})
}

func (r *OAuthApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OAuthApplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := oauthapplication.DeleteOAuthApplication(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete OAuth application",
			fmt.Sprintf("Could not delete OAuth application ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted OAuth application", map[string]any{"id": state.ID.ValueString()})
}

func (r *OAuthApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapOAuthApplicationResponseToModel(app *clerkgo.OAuthApplication, model *OAuthApplicationResourceModel) {
	model.ID = types.StringValue(app.ID)
	model.Name = types.StringValue(app.Name)
	model.CallbackURL = types.StringValue(app.CallbackURL)
	// Scopes: the API may reorder and add implicit scopes (e.g. offline_access).
	// Preserve the user's configured value to avoid inconsistent result errors.
	// Only set from API when model has no value (e.g. during import).
	if model.Scopes.IsNull() || model.Scopes.IsUnknown() {
		model.Scopes = types.StringValue(app.Scopes)
	}
	model.Public = types.BoolValue(app.Public)
	model.ConsentScreenEnabled = types.BoolValue(app.ConsentScreenEnabled)
	model.ClientID = types.StringValue(app.ClientID)
	model.DiscoveryURL = types.StringValue(app.DiscoveryURL)
	model.AuthorizeURL = types.StringValue(app.AuthorizeURL)
	model.TokenFetchURL = types.StringValue(app.TokenFetchURL)
	model.UserInfoURL = types.StringValue(app.UserInfoURL)
	model.CreatedAt = types.StringValue(millisToRFC3339(app.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(app.UpdatedAt))
	// NOTE: client_secret is intentionally NOT set here.
	// It is only returned on creation and must be preserved from state.
}
