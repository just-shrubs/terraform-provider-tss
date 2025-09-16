package provider

import (
	"context"
	"fmt"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// With the datasource.DataSource implementation
func TSSSecretsDataSource() datasource.DataSource {
	return &TSSSecretsDataSource{}
}

// TSSSecretsDataSource defines the data source implementation
type TSSSecretsDataSource struct {
	client *server.Server // Store the provider configuration
}

// Metadata provides the data source type name
func (d *TSSSecretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "tss_secrets"
	tflog.Trace(ctx, "TSSSecretsDataSource metadata configured", map[string]interface{}{
		"type_name": "tss_secrets",
	})
}

// Schema defines the schema for the data source
func (d *TSSSecretsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	tflog.Trace(ctx, "Defining schema for TSSSecretsDataSource")

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"ids": schema.ListAttribute{
				ElementType: types.Int64Type,
				Required:    true,
				Description: "A list of IDs of the secrets",
			},
			"field": schema.StringAttribute{
				Required:    true,
				Description: "The field to extract from the secrets",
			},
			"secrets": schema.ListNestedAttribute{
				Computed:    true,
				Description: "A list of secrets with their field values",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed:    true,
							Description: "The ID of the secret",
						},
						"value": schema.StringAttribute{
							Computed:    true,
							Sensitive:   true,
							Description: "The ephemeral value of the field of the secret",
						},
					},
				},
			},
		},
	}
}

// Configure initializes the data source with the provider configuration
func (d *TSSSecretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "Configuring TSSSecretsDataSource")

	if req.ProviderData == nil {
		// IMPORTANT: This method is called MULTIPLE times. An initial call might not have configured the Provider yet, so we need
		// to handle this gracefully. It will eventually be called with a configured provider.
		tflog.Debug(ctx, "Provider data is nil, waiting for provider configuration")
		return
	}

	// Log the received ProviderData
	tflog.Debug(ctx, "Provider data received, attempting to configure")

	// Retrieve the provider configuration
	client, ok := req.ProviderData.(*server.Server)
	if !ok {
		tflog.Error(ctx, "Invalid provider data type", map[string]interface{}{
			"expected": "*server.Configuration",
			"actual":   fmt.Sprintf("%T", req.ProviderData),
		})
		resp.Diagnostics.AddError("Configuration Error", "Failed to retrieve provider configuration")
		return
	}

	// Store the provider configuration in the data source
	d.client = client
	tflog.Debug(ctx, "Successfully configured TSSSecretsDataSource")
}

func (d *TSSSecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "Reading TSSSecretsDataSource")

	var state struct {
		IDs     []types.Int64 `tfsdk:"ids"`
		Field   types.String  `tfsdk:"field"`
		Secrets []struct {
			ID    types.Int64  `tfsdk:"id"`
			Value types.String `tfsdk:"value"`
		} `tfsdk:"secrets"`
	}

	// Read the configuration
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

	tflog.Info(ctx, "Fetching multiple secrets from TSS", map[string]interface{}{
		"count": len(state.IDs),
		"field": state.Field.ValueString(),
	})

	// Fetch secrets
	var results []struct {
		ID    types.Int64  `tfsdk:"id"`
		Value types.String `tfsdk:"value"`
	}

	for _, id := range state.IDs {
		secretID := int(id.ValueInt64())

		tflog.Debug(ctx, "Fetching secret", map[string]interface{}{
			"secret_id": secretID,
		})

		// Fetch the secret
		secret, err := d.client.Secret(secretID)
		if err != nil {
			tflog.Warn(ctx, "Failed to fetch secret, skipping", map[string]interface{}{
				"secret_id": secretID,
				"error":     err.Error(),
			})
			resp.Diagnostics.AddWarning("Secret Fetch Warning", fmt.Sprintf("Failed to fetch secret with ID %d: %s", secretID, err))
			continue // Skip this ID and continue with the rest
		}

		// Get the field name dynamically
		fieldName := state.Field.ValueString()

		tflog.Debug(ctx, "Extracting field from secret", map[string]interface{}{
			"secret_id": secretID,
			"field":     fieldName,
		})

		// Extract the field value
		fieldValue, ok := secret.Field(fieldName)
		if !ok {
			tflog.Error(ctx, "Field not found in secret", map[string]interface{}{
				"secret_id": secretID,
				"field":     fieldName,
			})
			resp.Diagnostics.AddError("Field Not Found", fmt.Sprintf("The secret does not contain the field '%s'", fieldName))
			continue
		}

		tflog.Trace(ctx, "Successfully extracted field from secret", map[string]interface{}{
			"secret_id": secretID,
			"field":     fieldName,
		})

		// Save the secret value in the state
		results = append(results, struct {
			ID    types.Int64  `tfsdk:"id"`
			Value types.String `tfsdk:"value"`
		}{
			ID:    types.Int64Value(int64(secretID)),
			Value: types.StringValue(fieldValue),
		})
	}

	tflog.Info(ctx, "Completed fetching secrets", map[string]interface{}{
		"requested":  len(state.IDs),
		"successful": successCount,
		"failed":     failedCount,
	})

	// Set the state
	state.Secrets = results
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to set state", map[string]interface{}{
			"error": resp.Diagnostics.Errors(),
		})
		return
	}

	tflog.Debug(ctx, "TSSSecretsDataSource read completed successfully")
}
