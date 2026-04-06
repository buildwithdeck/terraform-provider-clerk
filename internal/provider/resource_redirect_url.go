package provider

import (
	"context"
	"fmt"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/redirecturl"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &RedirectURLResource{}
	_ resource.ResourceWithConfigure   = &RedirectURLResource{}
	_ resource.ResourceWithImportState = &RedirectURLResource{}
)

func NewRedirectURLResource() resource.Resource {
	return &RedirectURLResource{}
}

type RedirectURLResource struct {
	configured bool
}

type RedirectURLResourceModel struct {
	ID        types.String `tfsdk:"id"`
	URL       types.String `tfsdk:"url"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *RedirectURLResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The clerk_redirect_url resource requires an api_key. Set it in the provider configuration or via the CLERK_API_KEY environment variable.",
		)
		return
	}
	r.configured = true
}

func (r *RedirectURLResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redirect_url"
}

func (r *RedirectURLResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clerk redirect URL. Redirect URLs are used to whitelist URLs for OAuth and other redirect-based authentication flows.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the redirect URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The URL to whitelist for redirect-based authentication flows.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the redirect URL was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the redirect URL was last updated.",
			},
		},
	}
}

func (r *RedirectURLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RedirectURLResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &redirecturl.CreateParams{
		URL: clerkgo.String(plan.URL.ValueString()),
	}

	result, err := redirecturl.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create redirect URL", err.Error())
		return
	}

	mapRedirectURLResponseToModel(result, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created redirect URL", map[string]any{"id": result.ID})
}

func (r *RedirectURLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RedirectURLResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := redirecturl.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read redirect URL",
			fmt.Sprintf("Could not read redirect URL ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapRedirectURLResponseToModel(result, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *RedirectURLResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"Redirect URLs cannot be updated. All fields require replacement.",
	)
}

func (r *RedirectURLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RedirectURLResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := redirecturl.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete redirect URL",
			fmt.Sprintf("Could not delete redirect URL ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted redirect URL", map[string]any{"id": state.ID.ValueString()})
}

func (r *RedirectURLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapRedirectURLResponseToModel(ru *clerkgo.RedirectURL, model *RedirectURLResourceModel) {
	model.ID = types.StringValue(ru.ID)
	model.URL = types.StringValue(ru.URL)
	model.CreatedAt = types.StringValue(millisToRFC3339(ru.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(ru.UpdatedAt))
}
