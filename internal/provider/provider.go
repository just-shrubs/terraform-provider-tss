package provider

import (
	"context"
	"log"
	"os"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the provider implements the ProviderWithEphemeralResources interface
var _ provider.Provider = &TSSProvider{}
var _ provider.ProviderWithEphemeralResources = (*TSSProvider)(nil)

// Define the provider structure
type TSSProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Define the provider schema model
type TSSProviderModel struct {
	ServerURL types.String `tfsdk:"server_url"`
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
	Domain    types.String `tfsdk:"domain"`
}

// Metadata returns the provider type name
func (p *TSSProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tss"
}

// Schema defines the provider-level schema
func (p *TSSProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
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
func (p *TSSProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	serverUrl := os.Getenv("TSS_SERVER_URL")
	username := os.Getenv("TSS_USERNAME")
	password := os.Getenv("TSS_PASSWORD")
	domain := os.Getenv("TSS_DOMAIN")

	var data TSSProviderModel

	// Log the start of the Configure method
	log.Printf("Starting Configure method")

	// Read configuration values into the config struct
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Configuration Error", "Failed to read provider configuration")
		log.Printf("Failed to read provider configuration", map[string]interface{}{
			"diagnostics": resp.Diagnostics,
		})
		return
	}

	// Log the configuration values
	log.Printf("Provider configuration values retrieved", map[string]interface{}{
		"server_url": data.ServerURL.ValueString(),
		"username":   data.Username.ValueString(),
	})

	// Check configuration data, which should take precedence over environment variable data, if found.
	if data.ServerURL.ValueString() != "" {
		serverUrl = data.ServerURL.ValueString()
	}
	if data.Username.ValueString() != "" {
		username = data.Username.ValueString()
	}
	if data.Password.ValueString() != "" {
		password = data.Password.ValueString()
	}
	if data.Domain.ValueString() != "" {
		domain = data.Domain.ValueString()
	}

	if serverUrl == "" {
		resp.Diagnostics.AddError(
			"Missing Server URL Configuration",
			"While configuring the provider, the Server URL was not found in "+
				"the TSS_SEVRER_URL environment variable or provider "+
				"configuration block server_url attribute.",
		)
		// Not returning early allows the logic to collect all errors.
	}

	if username == "" {
		resp.Diagnostics.AddError(
			"Missing Username Configuration",
			"While configuring the provider, the username was not found in "+
				"the TSS_USERNAME environment variable or provider "+
				"configuration block username attribute.",
		)
		// Not returning early allows the logic to collect all errors.
	}

	if password == "" {
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

	// Pass the server configuration to resources and data sources
	if serverConfig == nil {
		log.Printf("Server configuration is nil")
		resp.Diagnostics.AddError("Configuration Error", "Server configuration is nil")
		return
	}

	// Create the server client
	tssClient, err := server.New(*serverConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"An unexpected error occurred when creating the tss client",
			"Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = tssClient
	resp.ResourceData = tssClient
	resp.EphemeralResourceData = tssClient
}

// DataSources returns the data sources supported by the provider
func (p *TSSProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &TSSSecretDataSource{} },
		func() datasource.DataSource { return &TSSSecretsDataSource{} },
	}
}

// Resources returns the resources supported by the provider
func (p *TSSProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTSSSecretResource,
		func() resource.Resource {
			return &TSSSecretDeletionResource{}
		},
		//For the DEBUG environment, uncomment this line to unit test whether the secret value is being fetched successfully.
		//func() resource.Resource { return &PrintSecretResource{} },
	}
}

func (p *TSSProvider) EphemeralResources(_ context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		func() ephemeral.EphemeralResource {
			return &TSSSecretEphemeralResource{}
		},
		func() ephemeral.EphemeralResource {
			return &TSSSecretsEphemeralResource{}
		},
	}
}

// New returns a new instance of the provider
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TSSProvider{
			version: version,
		}
	}
}
