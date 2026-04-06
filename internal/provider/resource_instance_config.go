package provider

import (
	"context"
	"encoding/json"
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
	_ resource.Resource                = &InstanceConfigResource{}
	_ resource.ResourceWithConfigure   = &InstanceConfigResource{}
	_ resource.ResourceWithImportState = &InstanceConfigResource{}
)

func NewInstanceConfigResource() resource.Resource {
	return &InstanceConfigResource{}
}

type InstanceConfigResource struct {
	client *PlatformClient
}

type InstanceConfigResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	InstanceID    types.String `tfsdk:"instance_id"`
	Config        types.String `tfsdk:"config"`
	ConfigVersion types.String `tfsdk:"config_version"`
}

func (r *InstanceConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The clerk_instance_config resource requires a platform_api_key. Set it in the provider configuration or via the CLERK_PLATFORM_API_KEY environment variable. The Platform API is a beta feature that must be enabled by Clerk — contact Clerk support or visit your dashboard to request access.",
		)
		return
	}
	r.client = NewPlatformClient(data.PlatformAPIKey)
}

func (r *InstanceConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_config"
}

func (r *InstanceConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the configuration of a Clerk instance via the Platform API. This is a singleton resource — deleting it from state does not reset the instance configuration. The Platform API is a beta feature that must be enabled by Clerk.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource identifier in the format application_id/instance_id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Clerk application.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"instance_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Clerk instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.StringAttribute{
				Required:    true,
				Description: "JSON string containing instance configuration key-value pairs. Use jsonencode() to construct this value.",
			},
			"config_version": schema.StringAttribute{
				Computed:    true,
				Description: "The current version of the instance configuration, used for optimistic concurrency control.",
			},
		},
	}
}

func (r *InstanceConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(plan.Config.ValueString()), &configMap); err != nil {
		resp.Diagnostics.AddError(
			"Invalid config JSON",
			fmt.Sprintf("Could not parse config JSON: %s", err.Error()),
		)
		return
	}

	appID := plan.ApplicationID.ValueString()
	instID := plan.InstanceID.ValueString()

	newVersion, err := r.client.UpdateInstanceConfig(ctx, appID, instID, configMap)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create instance config", err.Error())
		return
	}

	plan.ID = types.StringValue(appID + "/" + instID)
	plan.ConfigVersion = types.StringValue(newVersion)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created instance config", map[string]any{"id": plan.ID.ValueString()})
}

func (r *InstanceConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := state.ApplicationID.ValueString()
	instID := state.InstanceID.ValueString()

	fullConfig, configVersion, err := r.client.GetInstanceConfig(ctx, appID, instID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read instance config",
			fmt.Sprintf("Could not read instance config for %s/%s: %s", appID, instID, err.Error()),
		)
		return
	}

	// Only keep the keys that the user originally specified in their config
	// to avoid storing the entire API response and causing perpetual diffs.
	var managedKeys map[string]interface{}
	if err := json.Unmarshal([]byte(state.Config.ValueString()), &managedKeys); err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse state config",
			fmt.Sprintf("Could not parse existing config from state: %s", err.Error()),
		)
		return
	}

	filtered := filterConfigKeys(managedKeys, fullConfig)

	configJSON, err := json.Marshal(filtered)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to serialize instance config",
			fmt.Sprintf("Could not serialize config to JSON: %s", err.Error()),
		)
		return
	}

	state.Config = types.StringValue(string(configJSON))
	state.ConfigVersion = types.StringValue(configVersion)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *InstanceConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InstanceConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state InstanceConfigResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(plan.Config.ValueString()), &configMap); err != nil {
		resp.Diagnostics.AddError(
			"Invalid config JSON",
			fmt.Sprintf("Could not parse config JSON: %s", err.Error()),
		)
		return
	}

	appID := plan.ApplicationID.ValueString()
	instID := plan.InstanceID.ValueString()

	newVersion, err := r.client.UpdateInstanceConfig(ctx, appID, instID, configMap)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update instance config", err.Error())
		return
	}

	plan.ID = state.ID
	plan.ConfigVersion = types.StringValue(newVersion)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated instance config", map[string]any{"id": plan.ID.ValueString()})
}

func (r *InstanceConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Removing instance config from state (no-op, singleton resource)", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *InstanceConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: application_id/instance_id, got: %s", req.ID),
		)
		return
	}

	resp.State.SetAttribute(ctx, path.Root("application_id"), types.StringValue(parts[0]))
	resp.State.SetAttribute(ctx, path.Root("instance_id"), types.StringValue(parts[1]))
	resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))
}

// filterConfigKeys recursively filters apiConfig to only include the keys
// present in managed. This prevents the API's full config from causing
// perpetual diffs when the user only manages a subset of keys.
func filterConfigKeys(managed, apiConfig map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, managedVal := range managed {
		apiVal, ok := apiConfig[key]
		if !ok {
			continue
		}
		managedMap, managedIsMap := managedVal.(map[string]interface{})
		apiMap, apiIsMap := apiVal.(map[string]interface{})
		if managedIsMap && apiIsMap {
			result[key] = filterConfigKeys(managedMap, apiMap)
		} else {
			result[key] = apiVal
		}
	}
	return result
}
