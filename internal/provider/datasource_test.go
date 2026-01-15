package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestDataSource(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("TF_LOG", "DEBUG")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "denobridge_datasource" "test" {
						path = "./datasource_test.ts"
						props = {
							value = "Hello World"
						}
					}
				`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.denobridge_datasource.test",
						tfjsonpath.New("path"),
						knownvalue.StringExact("./datasource_test.ts"),
					),
					statecheck.ExpectKnownValue(
						"data.denobridge_datasource.test",
						tfjsonpath.New("props").AtMapKey("value"),
						knownvalue.StringExact("Hello World"),
					),
					statecheck.ExpectKnownValue(
						"data.denobridge_datasource.test",
						tfjsonpath.New("result").AtMapKey("hashedValue"),
						knownvalue.StringExact("a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e"),
					),
				},
			},
		},
	})
}
