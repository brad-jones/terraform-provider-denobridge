package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAction(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("TF_LOG", "DEBUG")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: `
					resource "terraform_data" "test" {
						input = "fake-string"

						lifecycle {
							action_trigger {
								events  = [before_create]
								actions = [action.denobridge_action.test]
							}
						}
					}

					action "denobridge_action" "test" {
						config {
							path = "./action_test.ts"
							props = {
								path = "./action_test.txt"
								content = "hello"
							}
							permissions = {
								all = true
							}
						}
					}
				`,
				PostApplyFunc: func() {
					// Test the results of the action operation by verifying
					// the file was created with the expected content
					expectedContent := "hello"
					filePath := "./action_test.txt"

					resultContent, err := os.ReadFile(filePath)
					if err != nil {
						t.Errorf("Error occurred while reading file at path: %s, error: %s", filePath, err)
						return
					}

					if string(resultContent) != expectedContent {
						t.Errorf("Expected file content %q, got: %q", expectedContent, string(resultContent))
					}

					// Clean up the test file
					_ = os.Remove(filePath)
				},
			},
		},
	})
}

func TestActionWithZod(t *testing.T) {
	t.Setenv("TF_ACC", "1")
	t.Setenv("TF_LOG", "DEBUG")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: `
					resource "terraform_data" "test" {
						input = "fake-string"

						lifecycle {
							action_trigger {
								events  = [before_create]
								actions = [action.denobridge_action.test_zod]
							}
						}
					}

					action "denobridge_action" "test_zod" {
						config {
							path = "./action_zod_test.ts"
							props = {
								path = "./action_zod_test.txt"
								content = "hello from zod"
								args = ["a", "b", "c"]
							}
							permissions = {
								all = true
							}
						}
					}
				`,
				PostApplyFunc: func() {
					// Test the results of the action operation by verifying
					// the file was created with the expected content
					expectedContent := "hello from zod"
					filePath := "./action_zod_test.txt"

					resultContent, err := os.ReadFile(filePath)
					if err != nil {
						t.Errorf("Error occurred while reading file at path: %s, error: %s", filePath, err)
						return
					}

					if string(resultContent) != expectedContent {
						t.Errorf("Expected file content %q, got: %q", expectedContent, string(resultContent))
					}

					// Clean up the test file
					_ = os.Remove(filePath)
				},
			},
		},
	})
}
