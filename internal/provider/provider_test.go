// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/id-unibe-ch/terraform-provider-switchcloud/internal/provider/testserver"
)

// testAccProtoV6ProviderFactoriesWithServer returns a provider factory map that
// points the provider at the given testserver instance. It also sets the
// SWITCHCLOUD_ENDPOINT environment variable for the duration of the test so
// that the provider's Configure step picks up the mock server URL.
func testAccProtoV6ProviderFactoriesWithServer(t *testing.T, srv *testserver.Server) map[string]func() (tfprotov6.ProviderServer, error) {
	t.Helper()
	t.Setenv("SWITCHCLOUD_ENDPOINT", srv.URL())
	return map[string]func() (tfprotov6.ProviderServer, error){
		"switchcloud": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}
