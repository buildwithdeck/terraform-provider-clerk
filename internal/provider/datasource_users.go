package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &UsersDataSource{}
	_ datasource.DataSourceWithConfigure = &UsersDataSource{}
)

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

type UsersDataSource struct {
	client *PlatformClient
}

type UsersDataSourceModel struct {
	ApplicationID types.String `tfsdk:"application_id"`
	InstanceID    types.String `tfsdk:"instance_id"`
	Query         types.String `tfsdk:"query"`
	OrderBy       types.String `tfsdk:"order_by"`
	Limit         types.Int64  `tfsdk:"limit"`
	Offset        types.Int64  `tfsdk:"offset"`
	TotalCount    types.Int64  `tfsdk:"total_count"`
	Users         types.List   `tfsdk:"users"`
}

var userEmailAddressAttrTypes = map[string]attr.Type{
	"id":            types.StringType,
	"email_address": types.StringType,
}

var userAttrTypes = map[string]attr.Type{
	"id":         types.StringType,
	"username":   types.StringType,
	"first_name": types.StringType,
	"last_name":  types.StringType,
	"image_url":  types.StringType,
	"email_addresses": types.ListType{
		ElemType: types.ObjectType{AttrTypes: userEmailAddressAttrTypes},
	},
	"banned":     types.BoolType,
	"created_at": types.StringType,
	"updated_at": types.StringType,
}

func (d *UsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			"Expected ProviderData, got something else. Please report this issue.",
		)
		return
	}
	if data.PlatformAPIKey == "" {
		resp.Diagnostics.AddError(
			"Missing Platform API Key",
			"The clerk_users data source requires a platform_api_key. Set it in the provider configuration or via the CLERK_PLATFORM_API_KEY environment variable. The Platform API is a beta feature that must be enabled by Clerk — contact Clerk support or visit your dashboard to request access.",
		)
		return
	}
	d.client = NewPlatformClient(data.PlatformAPIKey)
}

func (d *UsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *UsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists users for a Clerk application instance via the Platform API.",
		Attributes: map[string]schema.Attribute{
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Clerk application.",
			},
			"instance_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the Clerk instance.",
			},
			"query": schema.StringAttribute{
				Optional:    true,
				Description: "Filter users by name or email address.",
			},
			"order_by": schema.StringAttribute{
				Optional:    true,
				Description: "Order results by a field (e.g. -created_at, +created_at). Defaults to -created_at.",
			},
			"limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of users to return (1-500).",
			},
			"offset": schema.Int64Attribute{
				Optional:    true,
				Description: "Number of users to skip before returning results.",
			},
			"total_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Total number of users matching the query.",
			},
			"users": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of users.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Unique identifier of the user.",
						},
						"username": schema.StringAttribute{
							Computed:    true,
							Description: "Username of the user.",
						},
						"first_name": schema.StringAttribute{
							Computed:    true,
							Description: "First name of the user.",
						},
						"last_name": schema.StringAttribute{
							Computed:    true,
							Description: "Last name of the user.",
						},
						"image_url": schema.StringAttribute{
							Computed:    true,
							Description: "URL of the user's profile image.",
						},
						"email_addresses": schema.ListNestedAttribute{
							Computed:    true,
							Description: "Email addresses associated with the user.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:    true,
										Description: "Unique identifier of the email address.",
									},
									"email_address": schema.StringAttribute{
										Computed:    true,
										Description: "The email address.",
									},
								},
							},
						},
						"banned": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the user is banned.",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							Description: "Timestamp when the user was created (RFC3339).",
						},
						"updated_at": schema.StringAttribute{
							Computed:    true,
							Description: "Timestamp when the user was last updated (RFC3339).",
						},
					},
				},
			},
		},
	}
}

func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model UsersDataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &ListUsersParams{}
	if !model.Query.IsNull() && !model.Query.IsUnknown() {
		params.Query = model.Query.ValueString()
	}
	if !model.OrderBy.IsNull() && !model.OrderBy.IsUnknown() {
		params.OrderBy = model.OrderBy.ValueString()
	}
	if !model.Limit.IsNull() && !model.Limit.IsUnknown() {
		params.Limit = model.Limit.ValueInt64()
	}
	if !model.Offset.IsNull() && !model.Offset.IsUnknown() {
		params.Offset = model.Offset.ValueInt64()
	}

	result, err := d.client.ListInstanceUsers(ctx, model.ApplicationID.ValueString(), model.InstanceID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list users",
			fmt.Sprintf("Could not list users for application %s instance %s: %s",
				model.ApplicationID.ValueString(), model.InstanceID.ValueString(), err.Error()),
		)
		return
	}

	model.TotalCount = types.Int64Value(result.TotalCount)

	emailAddressObjectType := types.ObjectType{AttrTypes: userEmailAddressAttrTypes}
	userObjectType := types.ObjectType{AttrTypes: userAttrTypes}

	users := make([]attr.Value, len(result.Data))
	for i, u := range result.Data {
		// Build email addresses list.
		emailAddrs := make([]attr.Value, len(u.EmailAddresses))
		for j, ea := range u.EmailAddresses {
			emailAddrs[j], _ = types.ObjectValue(userEmailAddressAttrTypes, map[string]attr.Value{
				"id":            types.StringValue(ea.ID),
				"email_address": types.StringValue(ea.EmailAddress),
			})
		}
		emailAddrsList, _ := types.ListValue(emailAddressObjectType, emailAddrs)

		// Handle nullable string fields.
		username := types.StringNull()
		if u.Username != nil {
			username = types.StringValue(*u.Username)
		}
		firstName := types.StringNull()
		if u.FirstName != nil {
			firstName = types.StringValue(*u.FirstName)
		}
		lastName := types.StringNull()
		if u.LastName != nil {
			lastName = types.StringValue(*u.LastName)
		}

		users[i], _ = types.ObjectValue(userAttrTypes, map[string]attr.Value{
			"id":              types.StringValue(u.ID),
			"username":        username,
			"first_name":      firstName,
			"last_name":       lastName,
			"image_url":       types.StringValue(u.ImageURL),
			"email_addresses": emailAddrsList,
			"banned":          types.BoolValue(u.Banned),
			"created_at":      types.StringValue(time.UnixMilli(u.CreatedAt).UTC().Format(time.RFC3339)),
			"updated_at":      types.StringValue(time.UnixMilli(u.UpdatedAt).UTC().Format(time.RFC3339)),
		})
	}

	model.Users, _ = types.ListValue(userObjectType, users)

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)

	tflog.Debug(ctx, "Read users data source", map[string]any{
		"total_count": result.TotalCount,
		"count":       len(result.Data),
	})
}
