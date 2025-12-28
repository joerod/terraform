package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/example/terraform-provider-scvmm/internal/provider"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "github.com/example/scvmm",
	})
	if err != nil {
		log.Fatal(err)
	}
}
