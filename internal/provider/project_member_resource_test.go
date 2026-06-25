// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/id-unibe-ch/terraform-provider-switchcloud/internal/provider/testserver"
)

func TestAccProjectMemberResource(t *testing.T) {
	srv := testserver.New()
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithServer(t, srv),
		Steps: []resource.TestStep{
			// Create member by user_id
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
		},
	})
}

func TestAccProjectMemberResource_ByEmail(t *testing.T) {
	srv := testserver.New()
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithServer(t, srv),
		Steps: []resource.TestStep{
			// Create member by email
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

func TestAccProjectMemberResource_CreateServerError(t *testing.T) {
	srv := testserver.New()
	defer srv.Close()
	srv.Override("POST", "/api/v1/projects/{project_id}/members",
		testserver.RespondWith(http.StatusUnprocessableEntity, `{"error":"member already exists"}`))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithServer(t, srv),
		Steps: []resource.TestStep{
			{
				Config:      testAccProjectMemberResourceConfig,
				ExpectError: regexp.MustCompile("API returned status 422"),
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
resource "switchcloud_project" "test" {
  name = "Test Project"
}

resource "switchcloud_project_member" "test" {
  project_id = switchcloud_project.test.id
  email      = "user@example.com"
}
`
