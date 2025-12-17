// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccProjectDataSourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.switchcloud_project.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("0faaecfb-d154-4f8f-bdc8-fccd630ddb39"),
					),
					statecheck.ExpectKnownValue(
						"data.switchcloud_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test1"),
					),
				},
			},
		},
	})
}

const testAccProjectDataSourceConfig = `
data "switchcloud_project" "test" {
  id = "0faaecfb-d154-4f8f-bdc8-fccd630ddb39"
}
`
