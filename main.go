package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/virak-cloud/terraform-provider-virak/internal/provider"
)

// Provider documentation generation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name virakcloud

var (
	// goreleaser can pass this information in through ldflags
	version = "dev"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/virak-cloud/virakcloud",
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
