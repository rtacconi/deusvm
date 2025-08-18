package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	deusprov "github.com/riccardotacconi/deusvm/terraform/provider"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run provider in debug mode")
	flag.Parse()

	if err := providerserver.Serve(context.Background(), func() provider.Provider { return &deusprov.DeusProvider{} }, providerserver.ServeOpts{Address: "registry.terraform.io/deusvm/deusvm"}); err != nil {
		log.Fatal(err)
	}
}
