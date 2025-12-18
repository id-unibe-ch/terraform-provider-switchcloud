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

func TestAccProjectMemberResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectMemberResourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"switchcloud_project_member.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"switchcloud_project_member.test",
						tfjsonpath.New("user_id"),
						knownvalue.StringExact("user-12345"),
					),
					statecheck.ExpectKnownValue(
						"switchcloud_project_member.test",
						tfjsonpath.New("display_name"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"switchcloud_project_member.test",
						tfjsonpath.New("email"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				Config: testAccProjectMemberResourceByEmailConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"switchcloud_project_member.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"switchcloud_project_member.test",
						tfjsonpath.New("user_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"switchcloud_project_member.test",
						tfjsonpath.New("display_name"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"switchcloud_project_member.test",
						tfjsonpath.New("email"),
						knownvalue.StringExact("user@example.com"),
					),
				},
			},
		},
	})
}

const testAccProjectMemberResourceConfig = `
resource "switchcloud_project" "test" {
  name = "Test Project"
}

resource "switchcloud_project_member" "test" {
  project_id = switchcloud_project.test.id
  user_id    = "user-12345"
}
`

const testAccProjectMemberResourceByEmailConfig = `
resource "switchcloud_project" "test2" {
  name = "Test Project"
}

resource "switchcloud_project_member" "test" {
  project_id = switchcloud_project.test2.id
  email      = "user@example.com"
}
`
