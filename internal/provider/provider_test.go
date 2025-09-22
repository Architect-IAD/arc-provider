// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
)

// IMPORTANT: the map key must match your provider's type name (resp.TypeName from Provider.Metadata).
// If your provider's Metadata sets TypeName = "arc", use "arc" here.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"arc": providerserver.NewProtocol6WithError(New("test")()), // if New returns func() provider.Provider
	// If your New signature is: func New() provider.Provider  -> use: providerserver.NewProtocol6WithError(New())
}

// Same for the factories that include the echo provider
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"arc":  providerserver.NewProtocol6WithError(New("test")()),
	"echo": echoprovider.NewProviderServer(),
}

func testAccPreCheck(t *testing.T) {
	// Add environment checks here if you need AWS creds/region for acc tests.
	// e.g., skip when running without required env:
	// if os.Getenv("AWS_REGION") == "" { t.Skip("AWS_REGION not set") }
}