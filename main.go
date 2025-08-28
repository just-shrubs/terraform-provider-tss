package main

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/just_shrubs/terraform-provider-tss/v2/internal/provider"
)

func main() {
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

	providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/DelineaXPM/tss",
	})
}
