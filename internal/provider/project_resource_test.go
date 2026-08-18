// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/id-unibe-ch/terraform-provider-switchcloud/internal/provider/testserver"
)

func TestAccProjectResource(t *testing.T) {
	srv := testserver.New()
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithServer(t, srv),
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("switchcloud_project.test", plancheck.ResourceActionUpdate),
					},
				},
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
					statecheck.ExpectKnownValue(
						"switchcloud_project.test",
						tfjsonpath.New("billing_reference"),
						knownvalue.StringExact("REF-1234"),
					),
				},
			},
		},
	})
}

func TestAccProjectResource_CreateServerError(t *testing.T) {
	srv := testserver.New()
	defer srv.Close()
	srv.Override("POST", "/api/v1/projects",
		testserver.RespondWith(http.StatusInternalServerError, `{"error":"internal server error"}`))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithServer(t, srv),
		Steps: []resource.TestStep{
			{
				Config:      testAccProjectResourceConfig,
				ExpectError: regexp.MustCompile("API returned status 500"),
			},
		},
	})
}

func TestAccProjectResource_ReadNotFound(t *testing.T) {
	srv := testserver.New()
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithServer(t, srv),
		Steps: []resource.TestStep{
			// Create the project normally.
			{
				Config: testAccProjectResourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"switchcloud_project.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Simulate the project disappearing on the server side;
			// Terraform should detect the drift and recreate the resource.
			{
				PreConfig: func() {
					srv.Reset()
				},
				Config: testAccProjectResourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"switchcloud_project.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
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
  billing_reference = "REF-1234"
}
`
