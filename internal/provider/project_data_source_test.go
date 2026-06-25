// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/id-unibe-ch/terraform-provider-switchcloud/internal/provider/testserver"
)

func TestAccProjectDataSource(t *testing.T) {
	const knownID = "0faaecfb-d154-4f8f-bdc8-fccd630ddb39"
	name := "test-datasource-project"

	srv := testserver.New()
	defer srv.Close()
	srv.SeedProject(testserver.Project{
		Id:   knownID,
		Name: name,
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesWithServer(t, srv),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDataSourceConfig(knownID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.switchcloud_project.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact(knownID),
					),
					statecheck.ExpectKnownValue(
						"data.switchcloud_project.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
				},
			},
		},
	})
}

func testAccProjectDataSourceConfig(id string) string {
	return `data "switchcloud_project" "test" { id = "` + id + `" }`
}
