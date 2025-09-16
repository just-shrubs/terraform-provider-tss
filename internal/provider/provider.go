package provider

import (
	"context"
	"os"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the provider implements the ProviderWithEphemeralResources interface
var (
	_ provider.Provider                       = &TssProvider{}
	_ provider.ProviderWithEphemeralResources = (*TssProvider)(nil)
)

// Define the provider structure
type TssProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Define the provider schema model
type TssProviderModel struct {
	ServerURL types.String `tfsdk:"server_url"`
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
	Domain    types.String `tfsdk:"domain"`
}

// Metadata returns the provider type name
func (p *TssProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tss"
	tflog.Trace(ctx, "TssProvider metadata configured", map[string]interface{}{
		"type_name": "tss",
		"version":   p.version,
	})
}

// Schema defines the provider-level schema
func (p *TssProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	tflog.Trace(ctx, "Defining schema for TssProvider")

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server_url": schema.StringAttribute{
				Required:    true,
				Description: "The Secret Server base URL e.g. https://localhost/SecretServer",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username of the Secret Server User to connect as",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The password of the Secret Server User",
			},
			"domain": schema.StringAttribute{
				Optional:    true,
				Description: "Domain of the Secret Server user",
			},
		},
	}
}

// Configure initializes the provider with the given configuration
func (p *TssProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring TSS provider")

	serverUrl := os.Getenv("TSS_SERVER_URL")
	username := os.Getenv("TSS_USERNAME")
	password := os.Getenv("TSS_PASSWORD")
	domain := os.Getenv("TSS_DOMAIN")

	tflog.Debug(ctx, "Checking environment variables", map[string]interface{}{
		"has_server_url": serverUrl != "",
		"has_username":   username != "",
		"has_password":   password != "",
		"has_domain":     domain != "",
	})

	var data TssProviderModel

	// Read configuration values into the config struct
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to read provider configuration", map[string]interface{}{
			"error": resp.Diagnostics.Errors(),
		})
		resp.Diagnostics.AddError(
			"Configuration Error",
			"Failed to read provider configuration",
		)
		return
	}

	// Log the configuration values
	tflog.Info(ctx, "Provider configuration values retrieved", map[string]interface{}{
		"server_url": data.ServerURL.ValueString(),
		"username":   data.Username.ValueString(),
	})

	// Check configuration data, which should take precedence over environment variable data, if found.
	if data.ServerURL.ValueString() != "" {
		tflog.Debug(ctx, "Using server URL from provider configuration")
		serverUrl = data.ServerURL.ValueString()
	}
	if data.Username.ValueString() != "" {
		tflog.Debug(ctx, "Using username from provider configuration")
		username = data.Username.ValueString()
	}
	if data.Password.ValueString() != "" {
		tflog.Debug(ctx, "Using password from provider configuration")
		password = data.Password.ValueString()
	}
	if data.Domain.ValueString() != "" {
		tflog.Debug(ctx, "Using domain from provider configuration")
		domain = data.Domain.ValueString()
	}

	if serverUrl == "" {
		tflog.Error(ctx, "Missing server URL configuration")
		resp.Diagnostics.AddError(
			"Missing Server URL Configuration",
			"While configuring the provider, the Server URL was not found in "+
				"the TSS_SEVRER_URL environment variable or provider "+
				"configuration block server_url attribute.",
		)
		// Not returning early allows the logic to collect all errors.
	}

	if username == "" {
		tflog.Error(ctx, "Missing username configuration")
		resp.Diagnostics.AddError(
			"Missing Username Configuration",
			"While configuring the provider, the username was not found in "+
				"the TSS_USERNAME environment variable or provider "+
				"configuration block username attribute.",
		)
		// Not returning early allows the logic to collect all errors.
	}

	if password == "" {
		tflog.Error(ctx, "Missing password configuration")
		resp.Diagnostics.AddError(
			"Missing Password Configuration",
			"While configuring the provider, the password was not found in "+
				"the TSS_PASSWORD environment variable or provider "+
				"configuration block password attribute.",
		)
		// Not returning early allows the logic to collect all errors.
	}

	// Create the server configuration
	serverConfig := &server.Configuration{
		ServerURL: serverUrl,
		Credentials: server.UserCredential{
			Username: username,
			Password: password,
			Domain:   domain,
		},
	}

	tflog.Debug(ctx, "Final configuration values", map[string]interface{}{
		"server_url":   serverUrl,
		"username":     username,
		"has_password": password != "",
		"domain":       domain,
	})

	// Create the server client
	tssClient, err := server.New(*serverConfig)
	if err != nil {
		tflog.Error(ctx, "Failed to create TSS client", map[string]interface{}{
			"error":      err.Error(),
			"server_url": serverUrl,
		})
		resp.Diagnostics.AddError(
			"An unexpected error occurred when creating the tss client",
			"Error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "TSS provider configured successfully", map[string]interface{}{
		"server_url": serverUrl,
		"username":   username,
	})

	resp.DataSourceData = tssClient
	resp.ResourceData = tssClient
	resp.EphemeralResourceData = tssClient
}

// DataSources returns the data sources supported by the provider
func (p *TssProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	tflog.Trace(ctx, "Registering TSS data sources")
	return []func() datasource.DataSource{
		NewTssSecretDataSource,
		NewTssSecretsDataSource,
	}
}

// Resources returns the resources supported by the provider
func (p *TssProvider) Resources(ctx context.Context) []func() resource.Resource {
	tflog.Trace(ctx, "Registering TSS resources")
	return []func() resource.Resource{
		NewTssSecretResource,
	}
}

func (p *TssProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	tflog.Trace(ctx, "Registering TSS ephemeral resources")
	return []func() ephemeral.EphemeralResource{
		NewTssSecretEphemeralResource,
		NewTssSecretsEphemeralResource,
	}
}

// New returns a new instance of the provider
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TssProvider{
			version: version,
		}
	}
}
