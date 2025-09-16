package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// TSSSecretsEphemeralResource implements the ephemeral resource for fetching multiple secrets.
// Ephemeral resources are used for sensitive data that should not be persisted in state.
type TSSSecretsEphemeralResource struct {
	client *server.Server // Store the provider configuration
}

// TSSSecretsEphemeralResourceModel represents the data model for the ephemeral resource.
// This structure maps directly to the Terraform schema.
type TSSSecretsEphemeralResourceModel struct {
	IDs     []types.Int64 `tfsdk:"ids"`
	Field   types.String  `tfsdk:"field"`
	Secrets []SecretModel `tfsdk:"secrets"`
}

// SecretModel represents a single secret's extracted data
type SecretModel struct {
	ID    types.Int64  `tfsdk:"id"`
	Value types.String `tfsdk:"value"`
}

// Define private data structure (optional)
// TSSSecretsPrivateData stores data between resource lifecycle operations.
// This is used during renewal to avoid re-reading configuration.
type TSSSecretsPrivateData struct {
	IDs     []types.Int64 `tfsdk:"ids"`
	Field   string        `json:"field"`
	Secrets []SecretModel `tfsdk:"secrets"`
}

func (r *TSSSecretsEphemeralResource) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	tflog.Trace(ctx, "TSSSecretsEphemeralResource metadata configured", map[string]interface{}{
		"type_name": "tss_secrets",
	})
	resp.TypeName = "tss_secrets"
}

func (r *TSSSecretsEphemeralResource) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	tflog.Trace(ctx, "Defining schema for TSSSecretsEphemeralResource")

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
							Description: "The ephemeral value of the field of the secret",
						},
					},
				},
			},
		},
	}
}

func (r *TSSSecretsEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	tflog.Debug(ctx, "Opening TSSSecretsEphemeralResource")

	// Create a model to hold the input configuration
	var data TSSSecretsEphemeralResourceModel

	// Read the Terraform config data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read configuration", map[string]interface{}{
			"error": resp.Diagnostics.Errors(),
		})
		return
	}

	if r.client == nil {
		tflog.Error(ctx, "TSS server is nil")
		resp.Diagnostics.AddError("Provider not configured", "Cannot fetch secrets because the provider is not configured.")
		return
	}

	// Check for required fields in the model (secret_ids and field)
	if len(data.IDs) == 0 || data.Field.IsNull() {
		tflog.Error(ctx, "Missing required fields", map[string]interface{}{
			"has_ids":   data.IDs != nil && len(data.IDs) > 0,
			"has_field": !data.Field.IsNull(),
		})
		resp.Diagnostics.AddError("Missing Required Field", "Both secret_ids and field are required")
		return
	}

	tflog.Info(ctx, "Fetching secrets", map[string]interface{}{
		"count": len(data.IDs),
		"field": data.Field.ValueString(),
	})

	// Fetch secrets
	var results []SecretModel

	for _, id := range data.IDs {
		secretID := int(id.ValueInt64())

		tflog.Debug(ctx, "Fetching secret", map[string]interface{}{
			"secret_id": secretID,
		})

		// Fetch the secret
		secret, err := r.client.Secret(secretID)
		if err != nil {
			tflog.Warn(ctx, "Failed to fetch secret", map[string]interface{}{
				"secret_id": secretID,
				"error":     err.Error(),
			})
			resp.Diagnostics.AddWarning("Secret Fetch Warning", fmt.Sprintf("Failed to fetch secret with ID %d: %s", secretID, err))
			continue // Skip this ID and continue with the rest
		}

		tflog.Debug(ctx, "Using field of secret with id", map[string]interface{}{
			"field":     data.Field.ValueString(),
			"secret id": secretID,
		})

		// Extract the requested field value (assuming Field() method is available)
		fieldValue, ok := secret.Field(data.Field.ValueString())
		if !ok {
			tflog.Error(ctx, "Field not found in secret", map[string]interface{}{
				"secret_id": secretID,
				"field":     data.Field.ValueString(),
			})
			resp.Diagnostics.AddError("Field Not Found", fmt.Sprintf("Field %s not found in the secret", data.Field.ValueString()))
			continue
		}

		tflog.Trace(ctx, "Successfully extracted field from secret", map[string]interface{}{
			"secret_id": secretID,
			"field":     data.Field.ValueString(),
		})

		// Save the secret value in the state
		results = append(results, struct {
			ID    types.Int64  `tfsdk:"id"`
			Value types.String `tfsdk:"value"`
		}{
			ID:    types.Int64Value(int64(secretID)),
			Value: types.StringValue(fieldValue),
		})

		tflog.Info(ctx, "Successfully fetched secrets", map[string]interface{}{
			"requested": len(data.IDs),
			"retrieved": len(results),
		})
	}

	// Set the secret value in the result
	data.Secrets = results

	// Save the data into the ephemeral result state
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)

	// Set a renewal time for the resource
	resp.RenewAt = time.Now().Add(5 * time.Minute)
	tflog.Debug(ctx, "Set renewal time", map[string]interface{}{
		"renew_at": resp.RenewAt.Format(time.RFC3339),
	})

	// Store private data for use during renewal
	privateData, _ := json.Marshal(TSSSecretsPrivateData{
		IDs:     data.IDs,
		Field:   data.Field.ValueString(),
		Secrets: data.Secrets,
	})
	resp.Private.SetKey(ctx, "tss_secrets_data", privateData)
	tflog.Trace(ctx, "Stored private data for renewal")
}

func (r *TSSSecretsEphemeralResource) Renew(ctx context.Context, req ephemeral.RenewRequest, resp *ephemeral.RenewResponse) {
	tflog.Debug(ctx, "Renewing TSSSecretsEphemeralResource")

	// Retrieve the private data that was stored during Open
	privateBytes, _ := req.Private.GetKey(ctx, "tss_secrets_data")
	if privateBytes == nil {
		tflog.Error(ctx, "Private data not found for renewal")
		resp.Diagnostics.AddError("Missing Private Data", "Private data was not found for renewal.")
		return
	}

	// Unmarshal private data
	var privateData TSSSecretsPrivateData
	if err := json.Unmarshal(privateBytes, &privateData); err != nil {
		tflog.Error(ctx, "Failed to unmarshal private data", map[string]interface{}{
			"error": err.Error(),
		})
		resp.Diagnostics.AddError("Invalid Private Data", "Failed to unmarshal private data.")
		return
	}

	// Ensure that secret_id and field are available in the private data
	if len(privateData.IDs) == 0 || privateData.Field == "" {
		tflog.Error(ctx, "Incomplete private data for renewal", map[string]interface{}{
			"has_ids":   privateData.IDs != nil && len(privateData.IDs) > 0,
			"has_field": privateData.Field != "",
		})
		resp.Diagnostics.AddError("Missing Private Data Fields", "Secret ID and field are required.")
		return
	}

	tflog.Info(ctx, "Renewing secrets", map[string]interface{}{
		"count": len(privateData.IDs),
		"field": privateData.Field,
	})

	// Fetch secrets
	var results []SecretModel

	for _, id := range privateData.IDs {
		secretID := int(id.ValueInt64())

		tflog.Debug(ctx, "Renewing secret", map[string]interface{}{
			"secret_id": secretID,
		})

		// Fetch the secret
		secret, err := r.client.Secret(secretID)
		if err != nil {
			tflog.Warn(ctx, "Failed to fetch secret during renewal", map[string]interface{}{
				"secret_id": secretID,
				"error":     err.Error(),
			})
			resp.Diagnostics.AddWarning("Secret Fetch Warning", fmt.Sprintf("Failed to fetch secret with ID %d: %s", secretID, err))
			continue // Skip this ID and continue with the rest
		}

		tflog.Debug(ctx, "Using field of secret to renew data", map[string]interface{}{
			"secret id": secretID,
			"field":     privateData.Field,
		})

		// Extract the requested field value (assuming Field() method is available)
		fieldValue, ok := secret.Field(privateData.Field)
		if !ok {
			tflog.Error(ctx, "Field not found during renewal", map[string]interface{}{
				"secret_id": secretID,
				"field":     privateData.Field,
			})
			resp.Diagnostics.AddError("Field Not Found", fmt.Sprintf("Field %s not found in the secret", privateData.Field))
			continue
		}

		tflog.Trace(ctx, "Successfully renewed secret", map[string]interface{}{
			"secret_id": secretID,
			"field":     privateData.Field,
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

	tflog.Info(ctx, "Successfully renewed secrets", map[string]interface{}{
		"requested": len(privateData.IDs),
		"retrieved": len(results),
	})

	// Update the private data with the new secret value
	privateData.Secrets = results

	// Store the updated private data for the next renewal
	privateDataBytes, _ := json.Marshal(privateData)
	resp.Private.SetKey(ctx, "tss_secrets_data", privateDataBytes)

	// Set the renewal time (e.g., 5 minutes from now)
	resp.RenewAt = time.Now().Add(5 * time.Minute)
	tflog.Debug(ctx, "Set next renewal time", map[string]interface{}{
		"renew_at": resp.RenewAt.Format(time.RFC3339),
	})
}

func (r *TSSSecretsEphemeralResource) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	tflog.Debug(ctx, "Closing TSSSecretsEphemeralResource")
	// No cleanup needed for this resource
}

func (r *TSSSecretsEphemeralResource) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	tflog.Trace(ctx, "Configuring TSSSecretsEphemeralResource")

	if req.ProviderData == nil {
		tflog.Debug(ctx, "Provider data is nil, skipping configuration")
		return
	}

	client, ok := req.ProviderData.(*server.Server)
	if !ok {
		tflog.Error(ctx, "Invalid provider data type", map[string]interface{}{
			"expected": "*server.Server",
			"actual":   fmt.Sprintf("%T", req.ProviderData),
		})
		resp.Diagnostics.AddError("Invalid Provider Data", "Expected provider data of type *server.Configuration")
		return
	}

	log.Printf("DEBUG: Successfully retrieved provider configuration")

	r.client = client
}
