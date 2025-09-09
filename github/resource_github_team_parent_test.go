package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubTeamParent(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("creates a team configured with defaults", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_team" "parent" {
			  name    = "tf-parent-%s"
			  privacy = "closed"
			}
			resource "github_team" "child" {
			  name    = "tf-child-%s"
			  privacy = "closed"
			  lifecycle {
				ignore_changes = [parent_team_id]
			  }
			}

			resource "github_team_parent" "test" {
			  team_id  = github_team.child.id
			  parent_team_id = github_team.parent.id
			}
		`, randomID, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet("github_team_parent.test", "parent_team_id"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:  func() { skipUnlessMode(t, mode) },
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
