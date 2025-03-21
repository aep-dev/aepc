package main

import (
	"log"

	"github.com/aep-dev/terraform-provider-aep/openapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

// Version specifies the version of the provider (will be set statically at compile time)
const Version = "0.0.1"

// Commit specifies the commit hash of the provider at the time of building the binary (will be set statically at compile time)
const Commit = "none"

// Date specifies the data which the binary was build (will be set statically at compile time)
const Date = "unknown"

const ProviderName = "bookstore"
const ProviderOpenAPIURL = "http://localhost:8081/openapi.json"

func main() {

	log.Printf("[INFO] Running Terraform Provider %s v%s-%s; Released on: %s", ProviderName, Version, Commit, Date)

	log.Printf("[INFO] Initializing OpenAPI Terraform provider '%s' with service provider's OpenAPI document: %s", ProviderName, ProviderOpenAPIURL)

	p := openapi.ProviderOpenAPI{ProviderName: ProviderName}
	serviceProviderConfig := &openapi.ServiceConfigV1{
		SwaggerURL: ProviderOpenAPIURL,
	}

	provider, err := p.CreateSchemaProviderFromServiceConfiguration(serviceProviderConfig)
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize the terraform provider: %s", err)
	}

	plugin.Serve(
		&plugin.ServeOpts{
			ProviderFunc: func() *schema.Provider {
				return provider
			},
		},
	)
}
