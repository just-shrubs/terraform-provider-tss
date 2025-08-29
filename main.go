package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/just_shrubs/terraform-provider-tss/v2/internal/provider"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	if len(os.Args) >= 2 {
		action := os.Args[1]
		stateFile := os.Args[2]

		passphrase := os.Getenv("TFSTATE_PASSPHRASE")
		if passphrase == "" {
			log.Println("Passphrase not set in TFSTATE_PASSPHRASE environment variable")
			return
		}

		switch action {
		case "encrypt":
			err := provider.EncryptFile(passphrase, stateFile)
			if err != nil {
				log.Printf("[DEBUG] Error encrypting file: %v\n", err)
			}
		case "decrypt":
			err := provider.DecryptFile(passphrase, stateFile)
			if err != nil {
				log.Printf("[DEBUG] Error decrypting file: %v\n", err)
			}
		default:
			log.Println("[DEBUG] Invalid action. Use 'encrypt' or 'decrypt'.")
		}
		return
	}

	opts := providerserver.ServeOpts{
		// TODO: Update this string with the published name of your provider.
		Address: "github.com/just_shrubs/etcdv2",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
