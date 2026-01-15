package main

import (
	"context"
	"log"

	"github.com/brad-jones/deno-tf-bridge/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New("0.0.0"), providerserver.ServeOpts{
		Address: "example.registry.local/brad-jones/denobridge",
	})
	if err != nil {
		log.Fatal(err.Error())
	}
}
