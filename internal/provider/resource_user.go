package provider

import (
	"context"
	"encoding/json"
	"fmt"

	clerkgo "github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithConfigure   = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	configured bool
}

type UserResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	EmailAddresses            types.List   `tfsdk:"email_addresses"`
	PhoneNumbers              types.List   `tfsdk:"phone_numbers"`
	Web3Wallets               types.List   `tfsdk:"web3_wallets"`
	Username                  types.String `tfsdk:"username"`
	Password                  types.String `tfsdk:"password"`
	PasswordDigest            types.String `tfsdk:"password_digest"`
	PasswordHasher            types.String `tfsdk:"password_hasher"`
	SkipPasswordRequirement   types.Bool   `tfsdk:"skip_password_requirement"`
	SkipPasswordChecks        types.Bool   `tfsdk:"skip_password_checks"`
	SkipLegalChecks           types.Bool   `tfsdk:"skip_legal_checks"`
	FirstName                 types.String `tfsdk:"first_name"`
	LastName                  types.String `tfsdk:"last_name"`
	ExternalID                types.String `tfsdk:"external_id"`
	TOTPSecret                types.String `tfsdk:"totp_secret"`
	BackupCodes               types.List   `tfsdk:"backup_codes"`
	DeleteSelfEnabled         types.Bool   `tfsdk:"delete_self_enabled"`
	CreateOrganizationEnabled types.Bool   `tfsdk:"create_organization_enabled"`
	CreateOrganizationsLimit  types.Int64  `tfsdk:"create_organizations_limit"`
	LegalAcceptedAt           types.String `tfsdk:"legal_accepted_at"`
	Locale                    types.String `tfsdk:"locale"`
	PublicMetadata            types.String `tfsdk:"public_metadata"`
	PrivateMetadata           types.String `tfsdk:"private_metadata"`
	UnsafeMetadata            types.String `tfsdk:"unsafe_metadata"`
	PrimaryEmailAddressID     types.String `tfsdk:"primary_email_address_id"`
	PrimaryPhoneNumberID      types.String `tfsdk:"primary_phone_number_id"`
	Banned                    types.Bool   `tfsdk:"banned"`
	Locked                    types.Bool   `tfsdk:"locked"`
	CreatedAt                 types.String `tfsdk:"created_at"`
	UpdatedAt                 types.String `tfsdk:"updated_at"`
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages (seeds) a Clerk user via the Instance API. Useful for provisioning fixture users in non-production instances and for migration-style seeding with pre-hashed passwords.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email_addresses": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Email addresses to associate with the user at creation. The first is treated as primary. Changing this set forces a new user (identifiers are managed via dedicated APIs after creation).",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"phone_numbers": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Phone numbers to associate with the user at creation. Changing this set forces a new user.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"web3_wallets": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Web3 wallet addresses to associate with the user at creation. Changing this set forces a new user.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Username for the user. Requires usernames to be enabled in the Clerk Dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Plaintext password for the user. Write-only; never returned by the API.",
			},
			"password_digest": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Pre-hashed password for migration-style seeding. Pair with password_hasher. Write-only.",
			},
			"password_hasher": schema.StringAttribute{
				Optional:    true,
				Description: "Hashing algorithm used for password_digest (e.g. bcrypt, argon2i, pbkdf2_sha256). Write-only.",
			},
			"skip_password_requirement": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, the user can be created without a password. Write-only.",
			},
			"skip_password_checks": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, password strength validation is skipped. Write-only.",
			},
			"skip_legal_checks": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, legal acceptance validation is skipped. Write-only.",
			},
			"first_name": schema.StringAttribute{
				Optional:    true,
				Description: "First name of the user.",
			},
			"last_name": schema.StringAttribute{
				Optional:    true,
				Description: "Last name of the user.",
			},
			"external_id": schema.StringAttribute{
				Optional:    true,
				Description: "A unique identifier for the user in an external system.",
			},
			"totp_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "TOTP secret to seed for the user. Write-only.",
			},
			"backup_codes": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Sensitive:   true,
				Description: "Backup codes to seed for the user. Write-only; never returned by the API.",
			},
			"delete_self_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the user is allowed to delete their own account.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"create_organization_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the user is allowed to create organizations.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"create_organizations_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "The maximum number of organizations the user can create.",
			},
			"legal_accepted_at": schema.StringAttribute{
				Optional:    true,
				Description: "Timestamp (RFC3339) when the user accepted legal requirements. Write-only.",
			},
			"locale": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The user's locale in BCP-47 format.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_metadata": schema.StringAttribute{
				Optional:    true,
				Description: "Public metadata as a JSON string. Accessible from both the frontend and backend.",
			},
			"private_metadata": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Private metadata as a JSON string. Only accessible from the backend.",
			},
			"unsafe_metadata": schema.StringAttribute{
				Optional:    true,
				Description: "Unsafe metadata as a JSON string. Writable from the frontend; do not store sensitive data here.",
			},
			"primary_email_address_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the user's primary email address.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"primary_phone_number_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the user's primary phone number.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"banned": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the user is banned.",
			},
			"locked": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the user is locked.",
			},
			"created_at": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Timestamp (RFC3339) when the user was created. Set explicitly to backdate a seeded user; backdating forces a new user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp (RFC3339) when the user was last updated.",
			},
		},
	}
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &user.CreateParams{}

	emails, d := stringListToSlice(ctx, plan.EmailAddresses)
	resp.Diagnostics.Append(d...)
	phones, d := stringListToSlice(ctx, plan.PhoneNumbers)
	resp.Diagnostics.Append(d...)
	wallets, d := stringListToSlice(ctx, plan.Web3Wallets)
	resp.Diagnostics.Append(d...)
	backupCodes, d := stringListToSlice(ctx, plan.BackupCodes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if emails != nil {
		params.EmailAddresses = &emails
	}
	if phones != nil {
		params.PhoneNumbers = &phones
	}
	if wallets != nil {
		params.Web3Wallets = &wallets
	}
	if backupCodes != nil {
		params.BackupCodes = &backupCodes
	}

	if !plan.Username.IsNull() && !plan.Username.IsUnknown() {
		params.Username = clerkgo.String(plan.Username.ValueString())
	}
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		params.Password = clerkgo.String(plan.Password.ValueString())
	}
	if !plan.PasswordDigest.IsNull() && !plan.PasswordDigest.IsUnknown() {
		params.PasswordDigest = clerkgo.String(plan.PasswordDigest.ValueString())
	}
	if !plan.PasswordHasher.IsNull() && !plan.PasswordHasher.IsUnknown() {
		params.PasswordHasher = clerkgo.String(plan.PasswordHasher.ValueString())
	}
	if !plan.SkipPasswordRequirement.IsNull() && !plan.SkipPasswordRequirement.IsUnknown() {
		params.SkipPasswordRequirement = clerkgo.Bool(plan.SkipPasswordRequirement.ValueBool())
	}
	if !plan.SkipPasswordChecks.IsNull() && !plan.SkipPasswordChecks.IsUnknown() {
		params.SkipPasswordChecks = clerkgo.Bool(plan.SkipPasswordChecks.ValueBool())
	}
	if !plan.SkipLegalChecks.IsNull() && !plan.SkipLegalChecks.IsUnknown() {
		params.SkipLegalChecks = clerkgo.Bool(plan.SkipLegalChecks.ValueBool())
	}
	if !plan.FirstName.IsNull() && !plan.FirstName.IsUnknown() {
		params.FirstName = clerkgo.String(plan.FirstName.ValueString())
	}
	if !plan.LastName.IsNull() && !plan.LastName.IsUnknown() {
		params.LastName = clerkgo.String(plan.LastName.ValueString())
	}
	if !plan.ExternalID.IsNull() && !plan.ExternalID.IsUnknown() {
		params.ExternalID = clerkgo.String(plan.ExternalID.ValueString())
	}
	if !plan.TOTPSecret.IsNull() && !plan.TOTPSecret.IsUnknown() {
		params.TOTPSecret = clerkgo.String(plan.TOTPSecret.ValueString())
	}
	if !plan.DeleteSelfEnabled.IsNull() && !plan.DeleteSelfEnabled.IsUnknown() {
		params.DeleteSelfEnabled = clerkgo.Bool(plan.DeleteSelfEnabled.ValueBool())
	}
	if !plan.CreateOrganizationEnabled.IsNull() && !plan.CreateOrganizationEnabled.IsUnknown() {
		params.CreateOrganizationEnabled = clerkgo.Bool(plan.CreateOrganizationEnabled.ValueBool())
	}
	if !plan.CreateOrganizationsLimit.IsNull() && !plan.CreateOrganizationsLimit.IsUnknown() {
		limit := int(plan.CreateOrganizationsLimit.ValueInt64())
		params.CreateOrganizationsLimit = &limit
	}
	if !plan.LegalAcceptedAt.IsNull() && !plan.LegalAcceptedAt.IsUnknown() {
		params.LegalAcceptedAt = clerkgo.String(plan.LegalAcceptedAt.ValueString())
	}
	if !plan.Locale.IsNull() && !plan.Locale.IsUnknown() {
		params.Locale = clerkgo.String(plan.Locale.ValueString())
	}
	if !plan.CreatedAt.IsNull() && !plan.CreatedAt.IsUnknown() {
		params.CreatedAt = clerkgo.String(plan.CreatedAt.ValueString())
	}
	if !plan.PublicMetadata.IsNull() && !plan.PublicMetadata.IsUnknown() {
		raw := json.RawMessage(plan.PublicMetadata.ValueString())
		params.PublicMetadata = &raw
	}
	if !plan.PrivateMetadata.IsNull() && !plan.PrivateMetadata.IsUnknown() {
		raw := json.RawMessage(plan.PrivateMetadata.ValueString())
		params.PrivateMetadata = &raw
	}
	if !plan.UnsafeMetadata.IsNull() && !plan.UnsafeMetadata.IsUnknown() {
		raw := json.RawMessage(plan.UnsafeMetadata.ValueString())
		params.UnsafeMetadata = &raw
	}

	usr, err := user.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create user", err.Error())
		return
	}

	mapUserResponseToModel(usr, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Created user", map[string]any{"id": usr.ID})
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	usr, err := user.Get(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read user",
			fmt.Sprintf("Could not read user ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	mapUserResponseToModel(usr, &state)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UserResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := &user.UpdateParams{}

	backupCodes, d := stringListToSlice(ctx, plan.BackupCodes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if backupCodes != nil {
		params.BackupCodes = &backupCodes
	}

	if !plan.Username.IsNull() && !plan.Username.IsUnknown() {
		params.Username = clerkgo.String(plan.Username.ValueString())
	}
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		params.Password = clerkgo.String(plan.Password.ValueString())
	}
	if !plan.PasswordDigest.IsNull() && !plan.PasswordDigest.IsUnknown() {
		params.PasswordDigest = clerkgo.String(plan.PasswordDigest.ValueString())
	}
	if !plan.PasswordHasher.IsNull() && !plan.PasswordHasher.IsUnknown() {
		params.PasswordHasher = clerkgo.String(plan.PasswordHasher.ValueString())
	}
	if !plan.SkipPasswordChecks.IsNull() && !plan.SkipPasswordChecks.IsUnknown() {
		params.SkipPasswordChecks = clerkgo.Bool(plan.SkipPasswordChecks.ValueBool())
	}
	if !plan.SkipLegalChecks.IsNull() && !plan.SkipLegalChecks.IsUnknown() {
		params.SkipLegalChecks = clerkgo.Bool(plan.SkipLegalChecks.ValueBool())
	}
	if !plan.FirstName.IsNull() && !plan.FirstName.IsUnknown() {
		params.FirstName = clerkgo.String(plan.FirstName.ValueString())
	}
	if !plan.LastName.IsNull() && !plan.LastName.IsUnknown() {
		params.LastName = clerkgo.String(plan.LastName.ValueString())
	}
	if !plan.ExternalID.IsNull() && !plan.ExternalID.IsUnknown() {
		params.ExternalID = clerkgo.String(plan.ExternalID.ValueString())
	}
	if !plan.TOTPSecret.IsNull() && !plan.TOTPSecret.IsUnknown() {
		params.TOTPSecret = clerkgo.String(plan.TOTPSecret.ValueString())
	}
	if !plan.DeleteSelfEnabled.IsNull() && !plan.DeleteSelfEnabled.IsUnknown() {
		params.DeleteSelfEnabled = clerkgo.Bool(plan.DeleteSelfEnabled.ValueBool())
	}
	if !plan.CreateOrganizationEnabled.IsNull() && !plan.CreateOrganizationEnabled.IsUnknown() {
		params.CreateOrganizationEnabled = clerkgo.Bool(plan.CreateOrganizationEnabled.ValueBool())
	}
	if !plan.CreateOrganizationsLimit.IsNull() && !plan.CreateOrganizationsLimit.IsUnknown() {
		limit := int(plan.CreateOrganizationsLimit.ValueInt64())
		params.CreateOrganizationsLimit = &limit
	}
	if !plan.LegalAcceptedAt.IsNull() && !plan.LegalAcceptedAt.IsUnknown() {
		params.LegalAcceptedAt = clerkgo.String(plan.LegalAcceptedAt.ValueString())
	}
	if !plan.Locale.IsNull() && !plan.Locale.IsUnknown() {
		params.Locale = clerkgo.String(plan.Locale.ValueString())
	}
	if !plan.PublicMetadata.IsNull() && !plan.PublicMetadata.IsUnknown() {
		raw := json.RawMessage(plan.PublicMetadata.ValueString())
		params.PublicMetadata = &raw
	}
	if !plan.PrivateMetadata.IsNull() && !plan.PrivateMetadata.IsUnknown() {
		raw := json.RawMessage(plan.PrivateMetadata.ValueString())
		params.PrivateMetadata = &raw
	}
	if !plan.UnsafeMetadata.IsNull() && !plan.UnsafeMetadata.IsUnknown() {
		raw := json.RawMessage(plan.UnsafeMetadata.ValueString())
		params.UnsafeMetadata = &raw
	}

	usr, err := user.Update(ctx, state.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update user", err.Error())
		return
	}

	mapUserResponseToModel(usr, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Debug(ctx, "Updated user", map[string]any{"id": usr.ID})
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := user.Delete(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete user",
			fmt.Sprintf("Could not delete user ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Deleted user", map[string]any{"id": state.ID.ValueString()})
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapUserResponseToModel maps a Clerk user API response onto the resource model.
// Write-only fields the API never echoes (password*, skip_*, totp_secret,
// backup_codes, legal_accepted_at) are intentionally left untouched so they
// keep their configured value and do not produce spurious diffs.
func mapUserResponseToModel(usr *clerkgo.User, model *UserResourceModel) {
	model.ID = types.StringValue(usr.ID)
	model.Banned = types.BoolValue(usr.Banned)
	model.Locked = types.BoolValue(usr.Locked)
	model.DeleteSelfEnabled = types.BoolValue(usr.DeleteSelfEnabled)
	model.CreateOrganizationEnabled = types.BoolValue(usr.CreateOrganizationEnabled)
	model.CreatedAt = types.StringValue(millisToRFC3339(usr.CreatedAt))
	model.UpdatedAt = types.StringValue(millisToRFC3339(usr.UpdatedAt))

	model.Username = optionalString(usr.Username)
	model.FirstName = optionalString(usr.FirstName)
	model.LastName = optionalString(usr.LastName)
	model.ExternalID = optionalString(usr.ExternalID)
	model.Locale = optionalString(usr.Locale)
	model.PrimaryEmailAddressID = optionalString(usr.PrimaryEmailAddressID)
	model.PrimaryPhoneNumberID = optionalString(usr.PrimaryPhoneNumberID)

	if usr.CreateOrganizationsLimit != nil {
		model.CreateOrganizationsLimit = types.Int64Value(int64(*usr.CreateOrganizationsLimit))
	}

	emails := make([]string, 0, len(usr.EmailAddresses))
	for _, e := range usr.EmailAddresses {
		if e != nil {
			emails = append(emails, e.EmailAddress)
		}
	}
	model.EmailAddresses = stringSliceToList(emails)

	phones := make([]string, 0, len(usr.PhoneNumbers))
	for _, p := range usr.PhoneNumbers {
		if p != nil {
			phones = append(phones, p.PhoneNumber)
		}
	}
	model.PhoneNumbers = stringSliceToList(phones)

	wallets := make([]string, 0, len(usr.Web3Wallets))
	for _, w := range usr.Web3Wallets {
		if w != nil {
			wallets = append(wallets, w.Web3Wallet)
		}
	}
	model.Web3Wallets = stringSliceToList(wallets)

	model.PublicMetadata = optionalMetadata(usr.PublicMetadata)
	model.PrivateMetadata = optionalMetadata(usr.PrivateMetadata)
	model.UnsafeMetadata = optionalMetadata(usr.UnsafeMetadata)
}

// optionalString converts a Clerk *string field to a Terraform value, mapping
// nil to null.
func optionalString(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// optionalMetadata normalizes a JSON metadata field, mapping empty / "{}" to
// null so an unset metadata block does not show as a diff.
func optionalMetadata(raw json.RawMessage) types.String {
	if len(raw) == 0 || string(raw) == "{}" {
		return types.StringNull()
	}
	return types.StringValue(normalizeJSON(string(raw)))
}

// stringListToSlice extracts a []string from a Terraform list. A null/unknown
// list yields a nil slice and no diagnostics. Conversion errors are returned
// for the caller to append.
func stringListToSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	out := make([]string, 0, len(list.Elements()))
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}

// stringSliceToList builds a Terraform string list from a slice, mapping an
// empty slice to null (keeps an unset identifier list out of state).
func stringSliceToList(in []string) types.List {
	if len(in) == 0 {
		return types.ListNull(types.StringType)
	}
	elems := make([]attr.Value, len(in))
	for i, s := range in {
		elems[i] = types.StringValue(s)
	}
	list, _ := types.ListValue(types.StringType, elems)
	return list
}
