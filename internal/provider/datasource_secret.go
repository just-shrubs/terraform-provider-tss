package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// With the datasource.DataSource implementation
func TSSSecretDataSource() datasource.DataSource {
	return &TSSSecretDataSource{}
}

// TSSSecretDataSource defines the data source implementation
type TSSSecretDataSource struct {
	client *server.Server // Store the provider configuration
}

// Metadata provides the data source type name
func (d *TSSSecretDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "tss_secret"
	tflog.Trace(ctx, "TSSSecretDataSource metadata configured", map[string]interface{}{
		"type_name": "tss_secret",
	})
}

// Schema defines the schema for the data source
func (d *TSSSecretDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	tflog.Trace(ctx, "Defining schema for TSSSecretDataSource")

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the secret to retrieve.",
			},
			"field": schema.StringAttribute{
				Required:    true,
				Description: "The field to extract from the secret.",
			},
			"value": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The value of the requested field from the secret.",
			},
		},
	}
}

// Configure initializes the data source with the provider configuration
func (d *TSSSecretDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "Configuring TSSSecretDataSource")

	if req.ProviderData == nil {
		// IMPORTANT: This method is called MULTIPLE times. An initial call might not have configured the Provider yet, so we need
		// to handle this gracefully. It will eventually be called with a configured provider.
		tflog.Debug(ctx, "Provider data is nil, waiting for provider configuration")
		return
	}

	// Log the received ProviderData
	tflog.Debug(ctx, "Provider data received, attempting to configure")

	client, ok := req.ProviderData.(*server.Server)
	if !ok || config == nil {
		tflog.Error(ctx, "Invalid provider data type", map[string]interface{}{
			"expected": "*server.Configuration",
			"actual":   fmt.Sprintf("%T", req.ProviderData),
		})
		resp.Diagnostics.AddError("Configuration Error", "Failed to retrieve provider configuration")
		return
	}

	// Log the successfully retrieved configuration
	tflog.Debug(ctx, "Successfully configured TSSSecretDataSource")

	d.client = client
}

// Read retrieves the data for the data source
func (d *TSSSecretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading TSSSecretDataSource")

	// Define the state structure
	var state struct {
		SecretID    types.String `tfsdk:"id"`
		Field       types.String `tfsdk:"field"`
		SecretValue types.String `tfsdk:"value"`
	}

	// Read the configuration from the request
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read configuration", map[string]interface{}{
			"error": resp.Diagnostics.Errors(),
		})
		return
	}

	// Ensure the client configuration is set
	if d.client == nil {
		tflog.Error(ctx, "Client configuration is nil")
		resp.Diagnostics.AddError("Client Error", "The server client is not configured")
		return
	}

	// Convert SecretID to int
	secretID, err := strconv.Atoi(state.SecretID.ValueString())
	if err != nil {
		tflog.Error(ctx, "Invalid secret ID format", map[string]interface{}{
			"secret_id": secretID,
			"error":     err.Error(),
		})
		resp.Diagnostics.AddError("Invalid Secret ID", "Secret ID must be an integer")
		return
	}

	tflog.Info(ctx, "Fetching secret from TSS", map[string]interface{}{
		"secret_id": secretID,
		"field":     state.Field.ValueString(),
	})

	// Fetch the secret
	secret, err := d.client.Secret(secretID)
	if err != nil {
		tflog.Error(ctx, "Failed to fetch secret", map[string]interface{}{
			"secret_id": secretID,
			"error":     err.Error(),
		})
		resp.Diagnostics.AddError("Secret Fetch Error", fmt.Sprintf("Failed to fetch secret: %s", err))
		return
	}

	// Get the field name dynamically
	fieldName := state.Field.ValueString()
	tflog.Debug(ctx, "Extracting field from secret", map[string]interface{}{
		"secret_id": secretID,
		"field":     fieldName,
	})

	// Extract the secret value
	fieldValue, ok := secret.Field(fieldName)
	if !ok {
		tflog.Error(ctx, "Field not found in secret", map[string]interface{}{
			"secret_id": secretID,
			"field":     fieldName,
		})
		resp.Diagnostics.AddError("Field Not Found", fmt.Sprintf("The secret does not contain the field '%s'", fieldName))
		return
	}

	tflog.Info(ctx, "Successfully retrieved secret field", map[string]interface{}{
		"secret_id": secretID,
		"field":     fieldName,
		"has_value": fieldValue != "",
	})

	// Set the secret value in the state
	state.SecretValue = types.StringValue(fieldValue)

	// Set the state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
