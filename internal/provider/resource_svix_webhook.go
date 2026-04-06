package provider

import (
	"context"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/svixwebhook"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource              = &SvixWebhookResource{}
	_ resource.ResourceWithConfigure = &SvixWebhookResource{}
)

func NewSvixWebhookResource() resource.Resource {
	return &SvixWebhookResource{}
}

type SvixWebhookResource struct {
	client *svixwebhook.Client
}

type SvixWebhookResourceModel struct {
	ID      types.String `tfsdk:"id"`
	SvixURL types.String `tfsdk:"svix_url"`
}

func (r *SvixWebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The clerk_svix_webhook resource requires an api_key. Set it in the provider configuration or via the CLERK_API_KEY environment variable.",
		)
		return
	}
	r.client = svixwebhook.NewClient(&clerkgo.ClientConfig{
		BackendConfig: clerkgo.BackendConfig{
			Key: clerkgo.String(data.APIKey),
		},
	})
}

func (r *SvixWebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_svix_webhook"
}

func (r *SvixWebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the Svix webhook integration for a Clerk instance. This is a singleton resource — only one can exist per instance.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Synthetic identifier for this singleton resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"svix_url": schema.StringAttribute{
				Computed:    true,
				Description: "The Svix dashboard URL for managing webhooks.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SvixWebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SvixWebhookResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	webhook, err := r.client.Create(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Svix webhook integration", err.Error())
		return
	}

	plan.ID = types.StringValue("svix_webhook")
	plan.SvixURL = types.StringValue(webhook.SvixURL)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created Svix webhook integration")
}

func (r *SvixWebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SvixWebhookResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// There is no Get endpoint for Svix webhooks. Preserve current state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *SvixWebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SvixWebhookResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No updatable fields; preserve state.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *SvixWebhookResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	_, err := r.client.Delete(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete Svix webhook integration", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted Svix webhook integration")
}
