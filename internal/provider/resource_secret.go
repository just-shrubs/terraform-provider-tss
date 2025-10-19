package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &TssSecretResource{}
	_ resource.ResourceWithConfigure   = &TssSecretResource{}
	_ resource.ResourceWithImportState = &TssSecretResource{}
)

// NewTssecretResource is a helper function to simplify the provider implementation.
func NewTssSecretResource() resource.Resource {
	return &TssSecretResource{}
}

// TssSecretResource defines the resource implementation
type TssSecretResource struct {
	client *server.Server
}

// SecretResourceState defines the state structure for the secret resource
type SecretResourceState struct {
	ID                               types.String  `tfsdk:"id"`
	Name                             types.String  `tfsdk:"name"`
	FolderID                         types.String  `tfsdk:"folderid"`
	SiteID                           types.String  `tfsdk:"siteid"`
	SecretTemplateID                 types.String  `tfsdk:"secrettemplateid"`
	Fields                           []SecretField `tfsdk:"fields"`
	SshKeyArgs                       *SshKeyArgs   `tfsdk:"sshkeyargs"`
	Active                           types.Bool    `tfsdk:"active"`
	SecretPolicyID                   types.Int64   `tfsdk:"secretpolicyid"`
	PasswordTypeWebScriptID          types.Int64   `tfsdk:"passwordtypewebscriptid"`
	LauncherConnectAsSecretID        types.Int64   `tfsdk:"launcherconnectassecretid"`
	CheckOutIntervalMinutes          types.Int64   `tfsdk:"checkoutintervalminutes"`
	CheckedOut                       types.Bool    `tfsdk:"checkedout"`
	CheckOutEnabled                  types.Bool    `tfsdk:"checkoutenabled"`
	AutoChangeEnabled                types.Bool    `tfsdk:"autochangenabled"`
	CheckOutChangePasswordEnabled    types.Bool    `tfsdk:"checkoutchangepasswordenabled"`
	DelayIndexing                    types.Bool    `tfsdk:"delayindexing"`
	EnableInheritPermissions         types.Bool    `tfsdk:"enableinheritpermissions"`
	EnableInheritSecretPolicy        types.Bool    `tfsdk:"enableinheritsecretpolicy"`
	ProxyEnabled                     types.Bool    `tfsdk:"proxyenabled"`
	RequiresComment                  types.Bool    `tfsdk:"requirescomment"`
	SessionRecordingEnabled          types.Bool    `tfsdk:"sessionrecordingenabled"`
	WebLauncherRequiresIncognitoMode types.Bool    `tfsdk:"weblauncherrequiresincognitomode"`
}

type SecretField struct {
	FieldName        types.String `tfsdk:"fieldname"`
	ItemValue        types.String `tfsdk:"itemvalue"`
	ItemID           types.Int64  `tfsdk:"itemid"`
	FieldID          types.Int64  `tfsdk:"fieldid"`
	FileAttachmentID types.Int64  `tfsdk:"fileattachmentid"`
	Slug             types.String `tfsdk:"slug"`
	FieldDescription types.String `tfsdk:"fielddescription"`
	Filename         types.String `tfsdk:"filename"`
	IsFile           types.Bool   `tfsdk:"isfile"`
	IsNotes          types.Bool   `tfsdk:"isnotes"`
	IsPassword       types.Bool   `tfsdk:"ispassword"`
	IsList           types.Bool   `tfsdk:"islist"`
	ListType         types.String `tfsdk:"listtype"`
}

type SshKeyArgs struct {
	GeneratePassphrase types.Bool `tfsdk:"generatepassphrase"`
	GenerateSshKeys    types.Bool `tfsdk:"generatesshkeys"`
}

// Metadata provides the resource type name
func (r *TssSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "dept-tss_resource_secret"
	tflog.Trace(ctx, "TssSecretResource metadata configured", map[string]interface{}{
		"type_name": resp.TypeName,
	})
}

// Schema defines the schema for the resource
func (r *TssSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	tflog.Trace(ctx, "Defining schema for TssSecretResource")

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "The ID of the secret.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the secret.",
			},
			"folderid": schema.StringAttribute{ // Changed to string for backward compatibility
				Required:    true,
				Description: "The folder ID of the secret.",
			},
			"siteid": schema.StringAttribute{ // Changed to string for backward compatibility
				Required:    true,
				Description: "The site ID where the secret will be created.",
			},
			"secrettemplateid": schema.StringAttribute{ // Changed to string for backward compatibility
				Required:    true,
				Description: "The template ID in which the secret will be created.",
			},
			"secretpolicyid": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the secret policy.",
			},
			"passwordtypewebscriptid": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the password type web script.",
			},
			"launcherconnectassecretid": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the launcher connect-as secret.",
			},
			"checkoutintervalminutes": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The checkout interval in minutes.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the secret is active.",
			},
			"checkedout": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the secret is checked out.",
			},
			"checkoutenabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether checkout is enabled for the secret.",
			},
			"autochangenabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether auto-change is enabled for the secret.",
			},
			"checkoutchangepasswordenabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether checkout change password is enabled.",
			},
			"delayindexing": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether delay indexing is enabled.",
			},
			"enableinheritpermissions": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether inherit permissions is enabled.",
			},
			"enableinheritsecretpolicy": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether inherit secret policy is enabled.",
			},
			"proxyenabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether proxy is enabled.",
			},
			"requirescomment": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether a comment is required.",
			},
			"sessionrecordingenabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether session recording is enabled.",
			},
			"weblauncherrequiresincognitomode": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the web launcher requires incognito mode.",
			},
		},
		Blocks: map[string]schema.Block{
			"fields": schema.ListNestedBlock{
				Description: "List of fields for the secret.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"fieldname": schema.StringAttribute{
							Optional: true,
						},
						"itemvalue": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Sensitive:   true,
							Description: "The value of the field. For SSH key generation, this will be computed by the server.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
								sshKeyFieldPlanModifier{},
								passwordFieldPlanModifier{},
							},
						},
						"itemid": schema.Int64Attribute{
							Optional: true,
							Computed: true,
						},
						"fieldid": schema.Int64Attribute{
							Optional: true,
							Computed: true,
						},
						"fileattachmentid": schema.Int64Attribute{
							Optional: true,
							Computed: true,
						},
						"slug": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"fielddescription": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"filename": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"isfile": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"isnotes": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"ispassword": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"islist": schema.BoolAttribute{
							Optional: true,
							Computed: true,
						},
						"listtype": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"sshkeyargs": schema.SingleNestedBlock{
				Description: "SSH key generation arguments.",
				Attributes: map[string]schema.Attribute{
					"generatepassphrase": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether to generate a passphrase for the SSH key.",
					},
					"generatesshkeys": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Whether to generate SSH keys.",
					},
				},
			},
		},
	}
	tflog.Debug(ctx, "Schema definition complete for TssSecretResource")
}

// Configure initializes the resource with the provider configuration
func (r *TssSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	tflog.Trace(ctx, "Configuring TssSecretResource")
	if req.ProviderData == nil {
		tflog.Debug(ctx, "Provider data is nil, skipping configuration")
		return
	}

	tflog.Debug(ctx, "Attempting to cast provider data to *server.Server")
	client, ok := req.ProviderData.(*server.Server)

	if !ok {
		tflog.Error(ctx, "Failed to cast provider data", map[string]interface{}{
			"expected_type": "*server.Server",
			"actual_type":   fmt.Sprintf("%T", req.ProviderData),
		})
		resp.Diagnostics.AddError("Configuration Error", "Failed to retrieve provider configuration")
		return
	}

	// Store the provider configuration in the resource
	r.client = client
	tflog.Info(ctx, "Configuring TssSecretResource completed successfully")
}

// Create creates the resource
func (r *TssSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating TssSecretResource")
	var plan SecretResourceState

	// Read the configuration
	tflog.Debug(ctx, "Reading plan configuration")
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan configuration", map[string]interface{}{
			"diagnostics": resp.Diagnostics.Errors(),
		})
		return
	}

	// Log plan details
	tflog.Debug(ctx, "Plan configuration read successfully", map[string]interface{}{
		"name":             plan.Name.ValueString(),
		"folder_id":        plan.FolderID.ValueString(),
		"site_id":          plan.SiteID.ValueString(),
		"template_id":      plan.SecretTemplateID.ValueString(),
		"field_count":      len(plan.Fields),
		"has_ssh_key_args": plan.SshKeyArgs != nil,
	})

	// Ensure the client configuration is set
	if r.client == nil {
		tflog.Error(ctx, "TSS client is not configured")
		resp.Diagnostics.AddError("Client Error", "The server client is not configured")
		return
	}

	// Get the secret data
	tflog.Debug(ctx, "Preparing secret data for creation")
	newSecret, err := r.generatePassword(ctx, &plan, r.client)
	if err != nil {
		tflog.Error(ctx, "Failed to prepare secret data", map[string]interface{}{
			"error": err.Error(),
			"name":  plan.Name.ValueString(),
		})
		resp.Diagnostics.AddError("Secret Data Error", fmt.Sprintf("Failed to prepare secret data: %s", err))
		return
	}

	tflog.Info(ctx, "Creating secret in TSS", map[string]interface{}{
		"name":        newSecret.Name,
		"folder_id":   newSecret.FolderID,
		"site_id":     newSecret.SiteID,
		"template_id": newSecret.SecretTemplateID,
	})

	// Use the client to create the secret
	createdSecret, err := r.client.CreateSecret(*newSecret)
	if err != nil {
		tflog.Error(ctx, "Failed to create secret in TSS", map[string]interface{}{
			"error":       err.Error(),
			"name":        newSecret.Name,
			"folder_id":   newSecret.FolderID,
			"template_id": newSecret.SecretTemplateID,
		})
		resp.Diagnostics.AddError("Secret Creation Error", fmt.Sprintf("Failed to create secret: %s", err))
		return
	}

	stringCreatedSecret := strconv.Itoa(createdSecret.ID)
	tflog.Info(ctx, "Secret created successfully in TSS", map[string]interface{}{
		"id":   stringCreatedSecret,
		"name": createdSecret.Name,
	})

	// Refresh state - let Terraform accept the computed values from the server
	tflog.Debug(ctx, "Refreshing state with created secret data")
	newState, readDiags := r.readSecretByID(ctx, stringCreatedSecret)
	resp.Diagnostics.Append(readDiags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to refresh state after creation", map[string]interface{}{
			"id":          stringCreatedSecret,
			"diagnostics": resp.Diagnostics.Errors(),
		})
		return
	}

	tflog.Debug(ctx, "Reordering fields to match original state order")
	newState.Fields = r.reorderFieldsToMatchPlan(ctx, plan.Fields, newState.Fields)

	// Preserve the SSH key args from the plan since the server doesn't return them
	if plan.SshKeyArgs != nil {
		newState.SshKeyArgs = plan.SshKeyArgs
		tflog.Debug(ctx, "Preserved SSH key arguments from plan", map[string]interface{}{
			"generate_ssh_keys":   plan.SshKeyArgs.GenerateSshKeys.ValueBool(),
			"generate_passphrase": plan.SshKeyArgs.GeneratePassphrase.ValueBool(),
		})
	}

	// Preserve file attachment information for file fields
	for i, field := range newState.Fields {
		if field.IsFile.ValueBool() {
			// Find the matching field in the plan
			for _, planField := range plan.Fields {
				if planField.FieldName.ValueString() == field.FieldName.ValueString() && planField.IsFile.ValueBool() {
					// Preserve FileAttachmentID and Filename
					newState.Fields[i].FileAttachmentID = planField.FileAttachmentID
					newState.Fields[i].Filename = planField.Filename
					tflog.Trace(ctx, "Preserved file attachment info", map[string]interface{}{
						"field":              field.FieldName.ValueString(),
						"file_attachment_id": planField.FileAttachmentID.ValueInt64(),
						"filename":           planField.Filename.ValueString(),
					})
					break
				}
			}
		}
	}

	// Set the state
	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set final state", map[string]interface{}{
			"diagnostics": resp.Diagnostics.Errors(),
		})
		return
	}

	tflog.Info(ctx, "Creating TssSecretResource completed successfully", map[string]interface{}{
		"id":   stringCreatedSecret,
		"name": createdSecret.Name,
	})
}

func (r *TssSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading TssSecretResource")
	var state SecretResourceState

	// Read the state
	tflog.Trace(ctx, "Reading current state")
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read current state", map[string]interface{}{
			"diagnostics": resp.Diagnostics.Errors(),
		})
		return
	}

	secretID := state.ID.ValueString()
	tflog.Debug(ctx, "Current state read successfully", map[string]interface{}{
		"id":   secretID,
		"name": state.Name.ValueString(),
	})

	// Store the original field order from the current state
	originalFields := state.Fields

	// Ensure the client configuration is set
	if r.client == nil {
		tflog.Error(ctx, "TSS client is not configured")
		resp.Diagnostics.AddError("Client Error", "The server client is not configured")
		return
	}

	tflog.Info(ctx, "Reading secret from TSS", map[string]interface{}{
		"id": secretID,
	})

	// Retrieve the secret
	newState, readDiags := r.readSecretByID(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(readDiags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read secret from TSS", map[string]interface{}{
			"id":          secretID,
			"diagnostics": resp.Diagnostics.Errors(),
		})
		return
	}

	tflog.Debug(ctx, "Secret retrieved from TSS", map[string]interface{}{
		"id":          secretID,
		"name":        newState.Name.ValueString(),
		"field_count": len(newState.Fields),
	})

	tflog.Debug(ctx, "Reordering fields to match original state order")
	newState.Fields = r.reorderFieldsToMatchPlan(ctx, originalFields, newState.Fields)

	// Preserve the SSH key args from the current state since the server doesn't return them
	if state.SshKeyArgs != nil {
		tflog.Debug(ctx, "Preserved SSH key arguments from state", map[string]interface{}{
			"generate_ssh_keys":   state.SshKeyArgs.GenerateSshKeys.ValueBool(),
			"generate_passphrase": state.SshKeyArgs.GeneratePassphrase.ValueBool(),
		})
		newState.SshKeyArgs = state.SshKeyArgs
	}

	// Determine if this secret was created with SSH key generation
	hasSshKeyArgs := false
	if state.SshKeyArgs != nil &&
		(state.SshKeyArgs.GenerateSshKeys.ValueBool() ||
			state.SshKeyArgs.GeneratePassphrase.ValueBool()) {
		hasSshKeyArgs = true
		tflog.Debug(ctx, "Secret has SSH key generation arguments")
	}

	// Preserve file attachment information for file fields and SSH key fields
	for i, field := range newState.Fields {
		fieldName := field.FieldName.ValueString()
		isSSHKeyField := hasSshKeyArgs && (strings.Contains(strings.ToLower(fieldName), "key") ||
			strings.Contains(strings.ToLower(fieldName), "passphrase"))

		if field.IsFile.ValueBool() || isSSHKeyField {
			// Find the matching field in the old state
			for _, oldField := range state.Fields {
				if oldField.FieldName.ValueString() == fieldName {
					// Preserve FileAttachmentID and Filename
					if !oldField.FileAttachmentID.IsNull() {
						newState.Fields[i].FileAttachmentID = oldField.FileAttachmentID
					}
					if !oldField.Filename.IsNull() && oldField.Filename.ValueString() != "" {
						newState.Fields[i].Filename = oldField.Filename
						tflog.Trace(ctx, "Preserved filename after update", map[string]interface{}{
							"field":    fieldName,
							"filename": oldField.Filename.ValueString(),
						})
					}
					break
				}
			}
		}
	}

	// Set the state
	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource
func (r *TssSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating TssSecretResource")
	var plan SecretResourceState
	var state SecretResourceState

	// Read the plan
	tflog.Debug(ctx, "Reading plan configuration")
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read plan or state", map[string]interface{}{
			"diagnostics": resp.Diagnostics.Errors(),
		})
		return
	}

	secretID := state.ID.ValueString()
	tflog.Debug(ctx, "Update configuration", map[string]interface{}{
		"id":           secretID,
		"name":         plan.Name.ValueString(),
		"current_name": state.Name.ValueString(),
		"field_count":  len(plan.Fields),
	})

	// Ensure the client configuration is set
	if r.client == nil {
		tflog.Error(ctx, "TSS client is not configured")
		resp.Diagnostics.AddError("Client Error", "The server client is not configured")
		return
	}

	// Get the secret data
	// During update, we shouldn't send SSH key generation parameters
	// because the server doesn't support SSH key generation during update
	updatePlan := plan

	// Check if SSH key generation was requested in the original creation
	hasSshKeyArgs := false
	if state.SshKeyArgs != nil &&
		(state.SshKeyArgs.GenerateSshKeys.ValueBool() ||
			state.SshKeyArgs.GeneratePassphrase.ValueBool()) {
		hasSshKeyArgs = true
		tflog.Debug(ctx, "Secret has SSH key arguments, will preserve during update")
	}

	// Don't send SSH key args during update - they're only for creation
	updatePlan.SshKeyArgs = nil

	// Prepare the updated secret data
	tflog.Debug(ctx, "Preparing updated secret data")
	updatedSecret, err := r.getSecretData(ctx, &updatePlan, r.client)
	if err != nil {
		tflog.Error(ctx, "Failed to prepare secret data for update", map[string]interface{}{
			"error": err.Error(),
		})
		resp.Diagnostics.AddError("Secret Data Error", fmt.Sprintf("Failed to prepare secret data: %s", err))
		return
	}

	// If we have SSH key fields, preserve the existing values from the current state
	for i, field := range updatedSecret.Fields {
		fieldName := field.FieldName

		isSSHKeyField := hasSshKeyArgs && (strings.Contains(strings.ToLower(fieldName), "key") ||
			strings.Contains(strings.ToLower(fieldName), "passphrase"))

		isPasswordField := false
		// For secrets with SSH keys, preserve the server-generated values
		for _, stateField := range state.Fields {
			if strings.EqualFold(stateField.FieldName.ValueString(), fieldName) {
				if !stateField.IsPassword.IsNull() && stateField.IsPassword.ValueBool() {
					isPasswordField = true
				}
				break
			}
		}

		if isSSHKeyField || isPasswordField {
			for _, stateField := range state.Fields {
				if strings.EqualFold(stateField.FieldName.ValueString(), fieldName) {
					// Check if the plan specifically wants to update this field
					// If not, preserve the existing state value
					fieldFound := false
					for _, planField := range plan.Fields {
						if strings.EqualFold(planField.FieldName.ValueString(), fieldName) {
							fieldFound = true
							if planField.ItemValue.IsNull() || planField.ItemValue.ValueString() == "" {
								// Plan is not updating this field, preserve state
								updatedSecret.Fields[i].ItemValue = stateField.ItemValue.ValueString()
								tflog.Trace(ctx, "Preserving SSH field value", map[string]interface{}{
									"field": fieldName,
								})
							} else if !isPasswordField || planField.ItemValue.ValueString() != "" {
								// Plan is updating this field, use new value
								tflog.Debug(ctx, "Updating field with new value", map[string]interface{}{
									"field": fieldName,
								})
							}
							break
						}
					}

					if !fieldFound {
						// Field not found in plan, preserve state value
						updatedSecret.Fields[i].ItemValue = stateField.ItemValue.ValueString()
						tflog.Trace(ctx, "Preserving SSH field value (not in plan)", map[string]interface{}{
							"field": fieldName,
						})
					}

					// Also preserve the filename for key fields regardless
					if !stateField.Filename.IsNull() && stateField.Filename.ValueString() != "" {
						updatedSecret.Fields[i].Filename = stateField.Filename.ValueString()
						tflog.Debug(ctx, "Preserving filename for field", map[string]interface{}{
							"filename": field.Filename,
							"field":    fieldName,
						})
					}
					break
				}
			}
		}
	}

	us := state.ID.ValueString()
	ustoi, err := strconv.Atoi(us)
	if err != nil {
		tflog.Error(ctx, "Failed to convert secret ID to integer", map[string]interface{}{
			"id":    secretID,
			"error": err.Error(),
		})
		resp.Diagnostics.AddError("Error converting ID from string to int", fmt.Sprintf("Failed to update secret: %s", err))
		return
	}

	// Update the secret
	updatedSecret.ID = ustoi
	tflog.Info(ctx, "Updating secret in TSS", map[string]interface{}{
		"id":   ustoi,
		"name": updatedSecret.Name,
	})

	_, err = r.client.UpdateSecret(*updatedSecret)
	if err != nil {
		tflog.Error(ctx, "Failed to update secret in TSS", map[string]interface{}{
			"id":    ustoi,
			"name":  updatedSecret.Name,
			"error": err.Error(),
		})
		resp.Diagnostics.AddError("Secret Update Error", fmt.Sprintf("Failed to update secret: %s", err))
		return
	}

	tflog.Info(ctx, "Secret updated successfully in TSS", map[string]interface{}{
		"id":   ustoi,
		"name": updatedSecret.Name,
	})

	// Refresh state
	newState, readDiags := r.readSecretByID(ctx, us)
	resp.Diagnostics.Append(readDiags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to refresh state after update", map[string]interface{}{
			"id":          secretID,
			"diagnostics": resp.Diagnostics.Errors(),
		})
		return
	}

	tflog.Debug(ctx, "Reordering fields to match original state order")
	newState.Fields = r.reorderFieldsToMatchPlan(ctx, plan.Fields, newState.Fields)

	// Preserve the SSH key args from the plan since the server doesn't return them
	if plan.SshKeyArgs != nil {
		newState.SshKeyArgs = plan.SshKeyArgs
		tflog.Debug(ctx, "Preserved SSH key args for update")
	}

	// Preserve file attachment information for file fields and SSH key fields
	for i, field := range newState.Fields {
		fieldName := field.FieldName.ValueString()
		isSSHKeyField := hasSshKeyArgs && (strings.Contains(strings.ToLower(fieldName), "key") ||
			strings.Contains(strings.ToLower(fieldName), "passphrase"))

		// Handle both regular file fields and SSH key fields
		if field.IsFile.ValueBool() || isSSHKeyField {
			// First check the state (higher priority for existing secrets)
			for _, stateField := range state.Fields {
				if stateField.FieldName.ValueString() == fieldName {
					// Preserve FileAttachmentID and Filename from state
					if !stateField.FileAttachmentID.IsNull() {
						newState.Fields[i].FileAttachmentID = stateField.FileAttachmentID
					}
					if !stateField.Filename.IsNull() && stateField.Filename.ValueString() != "" {
						newState.Fields[i].Filename = stateField.Filename
						tflog.Debug(ctx, "Preserved filename for field from state", map[string]interface{}{
							"field":    fieldName,
							"filename": stateField.Filename.ValueString(),
						})
					}
					break
				}
			}

			// If filename still empty, check plan
			if newState.Fields[i].Filename.IsNull() || newState.Fields[i].Filename.ValueString() == "" {
				for _, planField := range plan.Fields {
					if planField.FieldName.ValueString() == fieldName {
						if !planField.Filename.IsNull() && planField.Filename.ValueString() != "" {
							newState.Fields[i].Filename = planField.Filename
						}
						break
					}
				}
			}
		}
	}

	// Set the state
	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource
func (r *TssSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting TSS secret")
	var state SecretResourceState

	// Read the state
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read state for deletion", map[string]interface{}{
			"diagnostics": resp.Diagnostics.Errors(),
		})
		return
	}

	id := state.ID.ValueString()
	name := state.Name.ValueString()
	tflog.Debug(ctx, "State read for deletion", map[string]interface{}{
		"id":   id,
		"name": name,
	})

	// Ensure the client configuration is set
	if r.client == nil {
		tflog.Error(ctx, "TSS client is not configured")
		resp.Diagnostics.AddError("Client Error", "The server client is not configured")
		return
	}

	idtoi, err := strconv.Atoi(id)
	if err != nil {
		tflog.Error(ctx, "Failed to convert ID for deletion", map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		})
	}

	tflog.Info(ctx, "Deleting secret from TSS", map[string]interface{}{
		"id":   idtoi,
		"name": name,
	})

	// Delete the secret
	err = r.client.DeleteSecret(idtoi)
	if err != nil {
		tflog.Error(ctx, "Failed to delete secret from TSS", map[string]interface{}{
			"id":    idtoi,
			"name":  name,
			"error": err.Error(),
		})
		resp.Diagnostics.AddError("Secret Deletion Error", fmt.Sprintf("Failed to delete secret: %s", err))
		return
	}

	tflog.Info(ctx, "TssSecretResource.Delete completed successfully", map[string]interface{}{
		"id":   idtoi,
		"name": name,
	})
}

// reorderFieldsToMatchPlan reorders the fields from the server response
// This prevents "inconsistent result" errors in workflows.
func (r *TssSecretResource) reorderFieldsToMatchPlan(ctx context.Context, planFields []SecretField, stateFields []SecretField) []SecretField {
	tflog.Debug(ctx, "Reordering fields to match plan")

	// Create a map of state fields by field name for quick lookup
	stateFieldMap := make(map[string]SecretField)
	for _, field := range stateFields {
		stateFieldMap[strings.ToLower(field.FieldName.ValueString())] = field
	}

	// Create result slice in the same order as plan
	reorderedFields := make([]SecretField, 0, len(planFields))

	for _, planField := range planFields {
		fieldName := strings.ToLower(planField.FieldName.ValueString())
		if stateField, exists := stateFieldMap[fieldName]; exists {
			reorderedFields = append(reorderedFields, stateField)
			tflog.Trace(ctx, "Matched field from state", map[string]interface{}{
				"field": planField.FieldName.ValueString(),
			})
		} else {
			tflog.Warn(ctx, "Field from plan not found in state", map[string]interface{}{
				"field": planField.FieldName.ValueString(),
			})
		}
	}

	// Add any fields from state that weren't in the plan (shouldn't normally happen)
	for _, stateField := range stateFields {
		found := false
		for _, reorderedField := range reorderedFields {
			if strings.EqualFold(stateField.FieldName.ValueString(), reorderedField.FieldName.ValueString()) {
				found = true
				break
			}
		}
		if !found {
			tflog.Warn(ctx, "Field from state not in plan, appending", map[string]interface{}{
				"field": stateField.FieldName.ValueString(),
			})
			reorderedFields = append(reorderedFields, stateField)
		}
	}

	return reorderedFields
}

// Support import of Secret Resources via ID
func (r *TssSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Trace(ctx, "Starting ImportState", map[string]interface{}{
		"import id": req.ID,
	})

	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *TssSecretResource) generatePassword(ctx context.Context, state *SecretResourceState, client *server.Server) (*server.Secret, error) {
	tflog.Debug(ctx, "Preparing secret data with password generation")

	secret, err := r.getSecretData(ctx, state, client)
	if err != nil {
		return nil, err
	}

	templateID, err := strconv.Atoi(state.SecretTemplateID.ValueString())
	if err != nil {
		return nil, fmt.Errorf("invalid Template ID: %w", err)
	}

	template, err := client.SecretTemplate(templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secret template: %w", err)
	}

	for i, field := range secret.Fields {
		var templateField *server.SecretTemplateField
		for _, tf := range template.Fields {
			// Only match by FieldID if it's non-zero
			if (field.FieldID > 0 && tf.SecretTemplateFieldID == field.FieldID) ||
				strings.EqualFold(tf.Name, field.FieldName) ||
				strings.EqualFold(tf.FieldSlugName, field.FieldName) {
				templateField = &tf
				break
			}
		}

		if templateField != nil && templateField.IsPassword {
			if field.ItemValue == "" {
				generatedPassword, err := client.GeneratePassword(templateField.FieldSlugName, template)
				if err != nil {
					tflog.Error(ctx, "Failed to generate password", map[string]interface{}{
						"field": field.FieldName,
						"error": err.Error(),
					})
					return nil, fmt.Errorf("failed to generate password for field %s: %w", field.FieldName, err)
				}

				secret.Fields[i].ItemValue = generatedPassword
				tflog.Debug(ctx, "Generated password for field", map[string]interface{}{
					"field": field.FieldName,
				})
			} else {
				tflog.Debug(ctx, "Using provided password for field", map[string]interface{}{
					"field": field.FieldName,
				})
			}
		}
	}

	return secret, nil
}

func (r *TssSecretResource) readSecretByID(ctx context.Context, id string) (*SecretResourceState, diag.Diagnostics) {
	tflog.Debug(ctx, "Reading secret by ID", map[string]interface{}{
		"id": id,
	})

	secretID, err := strconv.Atoi(id)
	if err != nil {
		tflog.Error(ctx, "Invalid secret ID format", map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		})
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic("Secret Conversion Error", fmt.Sprintf("invalid secret ID: %s", err)),
		}
	}

	// Retrieve the secret using the provided client
	secret, err := r.client.Secret(secretID)
	if err != nil {
		tflog.Error(ctx, "Failed to retrieve secret", map[string]interface{}{
			"id":    secretID,
			"error": err.Error(),
		})
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic("Secret Retrieval Error", fmt.Sprintf("Failed to retrieve secret: %s", err)),
		}
	}

	tflog.Debug(ctx, "Successfully retrieved secret", map[string]interface{}{
		"id":   secretID,
		"name": secret.Name,
	})

	state, err := flattenSecret(secret)
	if err != nil {
		tflog.Error(ctx, "Failed to flatten secret", map[string]interface{}{
			"id":    secretID,
			"error": err.Error(),
		})
		return nil, diag.Diagnostics{
			diag.NewErrorDiagnostic("State Error", fmt.Sprintf("Failed to flatten secret: %s", err)),
		}
	}

	return state, nil
}

func (r *TssSecretResource) getSecretData(ctx context.Context, state *SecretResourceState, client *server.Server) (*server.Secret, error) {
	tflog.Debug(ctx, "Preparing secret data from state")

	// Convert string attributes to integers
	folderID, err := strconv.Atoi(state.FolderID.ValueString())
	if err != nil {
		tflog.Error(ctx, "Invalid folder ID", map[string]interface{}{
			"folder_id": state.FolderID.ValueString(),
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("invalid Folder ID: %w", err)
	}

	siteID, err := strconv.Atoi(state.SiteID.ValueString())
	if err != nil {
		tflog.Error(ctx, "Invalid site ID", map[string]interface{}{
			"site_id": state.SiteID.ValueString(),
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("invalid Site ID: %w", err)
	}

	templateID, err := strconv.Atoi(state.SecretTemplateID.ValueString())
	if err != nil {
		tflog.Error(ctx, "Invalid template ID", map[string]interface{}{
			"template_id": state.SecretTemplateID.ValueString(),
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("invalid Template ID: %w", err)
	}

	tflog.Debug(ctx, "Fetching secret template", map[string]interface{}{
		"template_id": templateID,
	})

	// Fetch the secret template
	template, err := client.SecretTemplate(templateID)
	if err != nil {
		tflog.Error(ctx, "Failed to retrieve secret template", map[string]interface{}{
			"template_id": templateID,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("failed to retrieve secret template: %w", err)
	}

	// Construct the fields dynamically
	var fields []server.SecretField
	for _, field := range state.Fields {
		fieldName := field.FieldName.ValueString()

		// Find the matching template field
		var templateField server.SecretTemplateField
		foundField := false

		for _, record := range template.Fields {
			if strings.EqualFold(record.Name, fieldName) || strings.EqualFold(record.FieldSlugName, fieldName) {
				templateField = record // Not &record, just record
				foundField = true
				tflog.Trace(ctx, "Matched field with template", map[string]interface{}{
					"field":             fieldName,
					"template_field_id": record.SecretTemplateFieldID,
				})
				break
			}
		}

		// Validate that we found a matching template field
		if !foundField {
			tflog.Error(ctx, "Field not found in template", map[string]interface{}{
				"field": fieldName,
				"available_fields": func() []string {
					names := make([]string, len(template.Fields))
					for i, f := range template.Fields {
						names[i] = fmt.Sprintf("%s (slug: %s, id: %d)", f.Name, f.FieldSlugName, f.SecretTemplateFieldID)
					}
					return names
				}(),
			})
			return nil, fmt.Errorf("field '%s' not found in secret template", fieldName)
		}

		// Handle field values appropriately - all optional fields should accept null or empty values
		var itemValue string

		// All fields can accept null or empty values (they're all optional in Terraform schema)
		if field.ItemValue.IsNull() {
			// For null values, use empty string
			itemValue = ""
			tflog.Trace(ctx, "Field has null value, using empty string instead", map[string]interface{}{
				"field": fieldName,
			})
		} else {
			// Otherwise use the actual value
			itemValue = field.ItemValue.ValueString()

			// Log empty strings but keep them as valid values
			if itemValue == "" {
				tflog.Trace(ctx, "Field has explicitly set empty string value", map[string]interface{}{
					"field": fieldName,
				})
			}
		}

		// Populate the field object
		secretField := server.SecretField{
			FieldDescription: templateField.Description,
			FieldID:          templateField.SecretTemplateFieldID,
			FieldName:        templateField.Name,
			FileAttachmentID: 0,
			IsFile:           templateField.IsFile,
			IsNotes:          templateField.IsNotes,
			IsPassword:       templateField.IsPassword,
			ItemValue:        itemValue,
			Slug:             templateField.FieldSlugName,
		}

		// For file attachments, preserve the FileAttachmentID and Filename
		if templateField.IsFile || (!field.IsFile.IsNull() && field.IsFile.ValueBool()) {
			if !field.FileAttachmentID.IsNull() {
				secretField.FileAttachmentID = int(field.FileAttachmentID.ValueInt64())
			}

			if !field.Filename.IsNull() {
				secretField.Filename = field.Filename.ValueString()
			}

			tflog.Trace(ctx, "Preserved file attachment info", map[string]interface{}{
				"field":    fieldName,
				"filename": secretField.Filename,
			})
		}

		fields = append(fields, secretField)
	}

	// Populate the secret object
	secret := &server.Secret{
		Name:             state.Name.ValueString(),
		FolderID:         folderID,
		SiteID:           siteID,
		SecretTemplateID: templateID,
		Fields:           fields,
		Active:           state.Active.ValueBool(),
	}

	// Handle SSH key args if provided - only during create operations
	// (We ensure this is nil during updates in the Update method)
	if state.SshKeyArgs != nil {
		secret.SshKeyArgs = &server.SshKeyArgs{
			GeneratePassphrase: state.SshKeyArgs.GeneratePassphrase.ValueBool(),
			GenerateSshKeys:    state.SshKeyArgs.GenerateSshKeys.ValueBool(),
		}
		tflog.Debug(ctx, "Added SSH key generation arguments", map[string]interface{}{
			"generate_keys":       secret.SshKeyArgs.GenerateSshKeys,
			"generate_passphrase": secret.SshKeyArgs.GeneratePassphrase,
		})
	}

	// Handle optional attributes
	if !state.SecretPolicyID.IsNull() {
		secret.SecretPolicyID = int(state.SecretPolicyID.ValueInt64())
	}
	if !state.PasswordTypeWebScriptID.IsNull() {
		secret.PasswordTypeWebScriptID = int(state.PasswordTypeWebScriptID.ValueInt64())
	}
	if !state.LauncherConnectAsSecretID.IsNull() {
		secret.LauncherConnectAsSecretID = int(state.LauncherConnectAsSecretID.ValueInt64())
	}
	if !state.CheckOutIntervalMinutes.IsNull() {
		secret.CheckOutIntervalMinutes = int(state.CheckOutIntervalMinutes.ValueInt64())
	}
	if !state.CheckedOut.IsNull() {
		secret.CheckedOut = state.CheckedOut.ValueBool()
	}
	if !state.CheckOutEnabled.IsNull() {
		secret.CheckOutEnabled = state.CheckOutEnabled.ValueBool()
	}
	if !state.AutoChangeEnabled.IsNull() {
		secret.AutoChangeEnabled = state.AutoChangeEnabled.ValueBool()
	}
	if !state.CheckOutChangePasswordEnabled.IsNull() {
		secret.CheckOutChangePasswordEnabled = state.CheckOutChangePasswordEnabled.ValueBool()
	}
	if !state.DelayIndexing.IsNull() {
		secret.DelayIndexing = state.DelayIndexing.ValueBool()
	}
	if !state.EnableInheritPermissions.IsNull() {
		secret.EnableInheritPermissions = state.EnableInheritPermissions.ValueBool()
	}
	if !state.EnableInheritSecretPolicy.IsNull() {
		secret.EnableInheritSecretPolicy = state.EnableInheritSecretPolicy.ValueBool()
	}
	if !state.ProxyEnabled.IsNull() {
		secret.ProxyEnabled = state.ProxyEnabled.ValueBool()
	}
	if !state.RequiresComment.IsNull() {
		secret.RequiresComment = state.RequiresComment.ValueBool()
	}
	if !state.SessionRecordingEnabled.IsNull() {
		secret.SessionRecordingEnabled = state.SessionRecordingEnabled.ValueBool()
	}
	if !state.WebLauncherRequiresIncognitoMode.IsNull() {
		secret.WebLauncherRequiresIncognitoMode = state.WebLauncherRequiresIncognitoMode.ValueBool()
	}

	tflog.Debug(ctx, "Prepared secret data", map[string]interface{}{
		"name":        secret.Name,
		"folder_id":   secret.FolderID,
		"template_id": secret.SecretTemplateID,
		"field_count": len(secret.Fields),
	})

	return secret, nil
}

func flattenSecret(secret *server.Secret) (*SecretResourceState, error) {
	ctx := context.Background()
	tflog.Debug(ctx, "Flattening secret to state", map[string]interface{}{
		"id":   secret.ID,
		"name": secret.Name,
	})

	var fields []SecretField

	for _, f := range secret.Fields {
		// Handle ItemValue consistently for all fields - all fields can have empty values
		var itemValue types.String

		// All fields should use StringValue even for empty strings
		// This ensures Terraform treats empty strings as valid values rather than null
		itemValue = types.StringValue(f.ItemValue)

		// Add debug logging for empty values
		if f.ItemValue == "" {
			tflog.Trace(ctx, "Field has empty value", map[string]interface{}{
				"field": f.FieldName,
			})
		}

		field := SecretField{
			FieldName:        types.StringValue(f.FieldName),
			ItemValue:        itemValue,
			ItemID:           types.Int64Value(int64(f.ItemID)),
			FieldID:          types.Int64Value(int64(f.FieldID)),
			FileAttachmentID: types.Int64Value(int64(f.FileAttachmentID)),
			Slug:             types.StringValue(f.Slug),
			FieldDescription: types.StringValue(f.FieldDescription),
			Filename:         types.StringValue(f.Filename),
			IsFile:           types.BoolValue(f.IsFile),
			IsNotes:          types.BoolValue(f.IsNotes),
			IsPassword:       types.BoolValue(f.IsPassword),
		}

		// Handle file fields and potential SSH key fields
		if f.IsFile {
			field.FileAttachmentID = types.Int64Value(int64(f.FileAttachmentID))
			if f.Filename != "" {
				field.Filename = types.StringValue(f.Filename)
			}
		}

		// Special handling for SSH key fields - ensure they have filename if provided by server
		isSSHKeyField := strings.Contains(strings.ToLower(f.FieldName), "key") ||
			strings.Contains(strings.ToLower(f.FieldName), "passphrase")

		if isSSHKeyField && f.Filename != "" {
			field.Filename = types.StringValue(f.Filename)
			tflog.Trace(ctx, "Found SSH key field with filename", map[string]interface{}{
				"field":    f.FieldName,
				"filename": f.Filename,
			})
		}

		fields = append(fields, field)
	}

	state := &SecretResourceState{
		Name:             types.StringValue(secret.Name),
		ID:               types.StringValue(strconv.Itoa(secret.ID)),
		FolderID:         types.StringValue(strconv.Itoa(secret.FolderID)),
		SiteID:           types.StringValue(strconv.Itoa(secret.SiteID)),
		SecretTemplateID: types.StringValue(strconv.Itoa(secret.SecretTemplateID)),
		Fields:           fields,
		Active:           types.BoolValue(secret.Active),
	}

	// Handle SSH key args if present
	if secret.SshKeyArgs != nil {
		state.SshKeyArgs = &SshKeyArgs{
			GeneratePassphrase: types.BoolValue(secret.SshKeyArgs.GeneratePassphrase),
			GenerateSshKeys:    types.BoolValue(secret.SshKeyArgs.GenerateSshKeys),
		}
		tflog.Debug(ctx, "Preserved SSH key args in state")
	}

	// Optional fields
	if secret.SecretPolicyID != 0 {
		state.SecretPolicyID = types.Int64Value(int64(secret.SecretPolicyID))
	}
	if secret.PasswordTypeWebScriptID != 0 {
		state.PasswordTypeWebScriptID = types.Int64Value(int64(secret.PasswordTypeWebScriptID))
	}
	if secret.LauncherConnectAsSecretID != 0 {
		state.LauncherConnectAsSecretID = types.Int64Value(int64(secret.LauncherConnectAsSecretID))
	}
	if secret.CheckOutIntervalMinutes != 0 {
		state.CheckOutIntervalMinutes = types.Int64Value(int64(secret.CheckOutIntervalMinutes))
	}
	state.CheckedOut = types.BoolValue(secret.CheckedOut)
	state.CheckOutEnabled = types.BoolValue(secret.CheckOutEnabled)
	state.AutoChangeEnabled = types.BoolValue(secret.AutoChangeEnabled)
	state.CheckOutChangePasswordEnabled = types.BoolValue(secret.CheckOutChangePasswordEnabled)
	state.DelayIndexing = types.BoolValue(secret.DelayIndexing)
	state.EnableInheritPermissions = types.BoolValue(secret.EnableInheritPermissions)
	state.EnableInheritSecretPolicy = types.BoolValue(secret.EnableInheritSecretPolicy)
	state.ProxyEnabled = types.BoolValue(secret.ProxyEnabled)
	state.RequiresComment = types.BoolValue(secret.RequiresComment)
	state.SessionRecordingEnabled = types.BoolValue(secret.SessionRecordingEnabled)
	state.WebLauncherRequiresIncognitoMode = types.BoolValue(secret.WebLauncherRequiresIncognitoMode)

	tflog.Debug(ctx, "Successfully flattened secret", map[string]interface{}{
		"id":          secret.ID,
		"field_count": len(fields),
	})

	return state, nil
}

// sshKeyFieldPlanModifier is a custom plan modifier for SSH key fields
type sshKeyFieldPlanModifier struct{}

func (m sshKeyFieldPlanModifier) Description(ctx context.Context) string {
	return "If SSH key generation is enabled and the value is empty, mark as unknown so it can be computed."
}

func (m sshKeyFieldPlanModifier) MarkdownDescription(ctx context.Context) string {
	return "If SSH key generation is enabled and the value is empty, mark as unknown so it can be computed."
}

func (m sshKeyFieldPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Log the plan values for debugging
	tflog.Trace(ctx, "Running SSH key field plan modifier")

	// If user explicitly set a value (including empty string) in the config, respect it
	if !req.ConfigValue.IsNull() {
		tflog.Trace(ctx, "Using explicit config value for field")
		resp.PlanValue = req.ConfigValue
		return
	}

	if !req.StateValue.IsNull() && req.StateValue.ValueString() != "" {
		tflog.Trace(ctx, "Preserving existing value from state (import or update)")
		resp.PlanValue = req.StateValue
		return
	}

	// For creation with potentially computed values
	if req.State.Raw.IsNull() && (req.PlanValue.IsNull() || req.PlanValue.ValueString() == "") {
		// Determine if this value should be computed by SSH key generation
		if shouldComputeSshKeyValue(req) {
			tflog.Debug(ctx, "Marking field as computed for SSH key generation")
			resp.PlanValue = types.StringUnknown()
			return
		}
	}

	// For null values in the plan, convert to empty string for consistency
	if req.PlanValue.IsNull() {
		tflog.Trace(ctx, "Converting null plan value to empty string")
		resp.PlanValue = types.StringValue("")
		return
	}

	// Otherwise, use the planned value as is
	resp.PlanValue = req.PlanValue
}

// Helper function to determine if a field value should be computed by SSH key generation
func shouldComputeSshKeyValue(req planmodifier.StringRequest) bool {
	ctx := context.Background()
	// Only mark values as computed during creation for SSH key fields when SSH key generation is enabled

	// Check if this is a create operation (state is null)
	if !req.State.Raw.IsNull() {
		// This is an update, not a creation, so don't compute
		tflog.Trace(ctx, "Not a create operation, won't compute SSH key value")
		return false
	}

	// Check if the user explicitly set an empty string in the config
	// If they did, we should respect that and not compute a value
	if req.ConfigValue.IsNull() == false && req.ConfigValue.ValueString() == "" {
		// User explicitly set an empty string, preserve it
		tflog.Trace(ctx, "User explicitly set empty string, preserving it")
		return false
	}

	// If we've reached here, it's a create operation and the field might need to be computed

	// Check if the path contains a field reference
	pathSteps := req.Path.Steps()
	if len(pathSteps) < 3 {
		return false
	}

	// Check if this is the "itemvalue" attribute within a "fields" block
	if pathSteps[0].String() != "fields" || pathSteps[len(pathSteps)-1].String() != "itemvalue" {
		return false
	}

	// At this point, we would ideally check:
	// 1. If this field is an SSH key field (by name)
	// 2. If SSH key generation is enabled in the plan
	//
	// However, without easy access to the field name here,
	// and since we don't have access to other parts of the plan,
	// we'll assume any null/empty field during create could be computed

	// For create operations with empty values that haven't been explicitly set,
	// mark as computed
	return req.PlanValue.ValueString() == ""
}

// passwordFieldPlanModifier is a custom plan modifier for password fields
type passwordFieldPlanModifier struct{}

func (m passwordFieldPlanModifier) Description(ctx context.Context) string {
	return "If the field is a password and no value is provided, mark as unknown so it can be computed by the server."
}

func (m passwordFieldPlanModifier) MarkdownDescription(ctx context.Context) string {
	return "If the field is a password and no value is provided, mark as unknown so it can be computed by the server."
}

func (m passwordFieldPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	tflog.Trace(ctx, "Running password field plan modifier")

	if !req.ConfigValue.IsNull() && req.ConfigValue.ValueString() != "" {
		tflog.Debug(ctx, "Using explicit config value for password field")
		resp.PlanValue = req.ConfigValue
		return
	}

	if !req.StateValue.IsNull() && req.StateValue.ValueString() != "" {
		if req.ConfigValue.IsNull() || req.ConfigValue.ValueString() == "" {
			tflog.Debug(ctx, "Preserving existing password from state (import or update)")
			resp.PlanValue = req.StateValue
			return
		}
	}

	if req.State.Raw.IsNull() && (req.PlanValue.IsNull() || req.PlanValue.ValueString() == "") {
		if shouldComputePasswordValue(req) {
			tflog.Debug(ctx, "Marking password field as computed for generation")
			resp.PlanValue = types.StringUnknown()
			return
		}
	}

	resp.PlanValue = req.PlanValue
}

func shouldComputePasswordValue(req planmodifier.StringRequest) bool {
	ctx := context.Background()

	if !req.State.Raw.IsNull() {
		tflog.Trace(ctx, "Not a create operation, won't compute password value")
		return false
	}

	if req.ConfigValue.IsNull() == false && req.ConfigValue.ValueString() == "" {
		tflog.Trace(ctx, "User explicitly set empty password, not generating")
		return false
	}

	return true
}
