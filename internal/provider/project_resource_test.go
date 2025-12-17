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

// Placeholder integration test
// Resource cant be deleted by api call so cant be fully tested

func TestAccProjectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectResourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"switchcloud_project.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"switchcloud_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Project"),
					),
				},
			},
			{
				ResourceName:      "switchcloud_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccProjectResourceUpdateConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"switchcloud_project.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"switchcloud_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("Test Project"),
					),
					statecheck.ExpectKnownValue(
						"switchcloud_project.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("This is a test project description."),
					),
				},
			},
		},
	})
}

const testAccProjectResourceConfig = `
resource "switchcloud_project" "test" {
  name = "Test Project"
}
`

const testAccProjectResourceUpdateConfig = `
resource "switchcloud_project" "test" {
  name = "Test Project"
  description = "This is a test project description."
}
`
