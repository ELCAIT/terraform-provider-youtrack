package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const accMailRuleResource = "youtrack_mail_rule.test"

// testAccMailRuleDependencies are the two projects and the mailbox a mail rule
// test needs.
func testAccMailRuleDependencies(suffix int64, leaderLogin string) string {
	return providerBlock() + testAccIMAPMailboxConfig("imap.example.com", 993, fmt.Sprintf("tf-acc-%d@example.com", suffix), 30) +
		fmt.Sprintf(`
resource "youtrack_project" "first" {
  name         = "tf-acc-mail-rule-%[1]d"
  short_name   = "MR%[1]d"
  leader_login = %[2]q
}

resource "youtrack_project" "second" {
  name         = "tf-acc-mail-rule-2-%[1]d"
  short_name   = "MRB%[1]d"
  leader_login = %[2]q
}
`, suffix%1000000, leaderLogin)
}

func TestAccMailRule(t *testing.T) {
	skipUnlessAcc(t)

	leaderLogin := testProjectLeaderLogin(t)
	suffix := time.Now().UnixMilli()
	dependencies := testAccMailRuleDependencies(suffix, leaderLogin)

	minimal := dependencies + fmt.Sprintf(`
resource "youtrack_mail_rule" "test" {
  mailbox_id       = youtrack_mailbox.test.id
  project_id       = youtrack_project.first.id
  maintainer_login = %q
}
`, leaderLogin)

	full := dependencies + fmt.Sprintf(`
resource "youtrack_mail_rule" "test" {
  mailbox_id             = youtrack_mailbox.test.id
  project_id             = youtrack_project.second.id
  maintainer_login       = %[1]q
  reporter_policy        = "ALWAYS_PREDEFINED"
  default_reporter_login = %[1]q
  disabled               = true
  folder                 = "INBOX"
  to                     = "support@example.com"
  spam_filter_mode       = "BLOCK_TICKETS"
  new_issue_command      = "tag mail"
  filter_auto_replies    = false
  original_timestamps    = false
  delete_after_fetch     = true

  filters = [
    {
      scope          = "SUBJECT"
      patterns       = ["newsletter", "unsubscribe"]
      accept_matched = false
    },
  ]
}
`, leaderLogin)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: minimal,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accMailRuleResource, "id"),
					resource.TestCheckResourceAttrPair(accMailRuleResource, "mailbox_id", accMailboxResource, "id"),
					resource.TestCheckResourceAttrPair(accMailRuleResource, "project_id", "youtrack_project.first", "id"),
					resource.TestCheckResourceAttr(accMailRuleResource, "maintainer_login", leaderLogin),
					resource.TestCheckResourceAttr(accMailRuleResource, "reporter_policy", "SENDER_OR_NEW_USER"),
					resource.TestCheckResourceAttr(accMailRuleResource, "spam_filter_mode", "MARK_TICKETS"),
					resource.TestCheckResourceAttr(accMailRuleResource, "filter_auto_replies", "true"),
					resource.TestCheckResourceAttr(accMailRuleResource, "original_timestamps", "true"),
					resource.TestCheckNoResourceAttr(accMailRuleResource, "filters"),
				),
			},
			{
				Config: full,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(accMailRuleResource, "project_id", "youtrack_project.second", "id"),
					resource.TestCheckResourceAttr(accMailRuleResource, "default_reporter_login", leaderLogin),
					resource.TestCheckResourceAttr(accMailRuleResource, "disabled", "true"),
					resource.TestCheckResourceAttr(accMailRuleResource, "folder", "INBOX"),
					resource.TestCheckResourceAttr(accMailRuleResource, "spam_filter_mode", "BLOCK_TICKETS"),
					resource.TestCheckResourceAttr(accMailRuleResource, "filters.#", "1"),
					resource.TestCheckResourceAttr(accMailRuleResource, "filters.0.patterns.#", "2"),
					resource.TestCheckResourceAttr(accMailRuleResource, "filters.0.patterns.1", "unsubscribe"),
				),
			},
			{
				ResourceName:      accMailRuleResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Removing the optional attributes resets them.
				Config: minimal,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accMailRuleResource, "reporter_policy", "SENDER_OR_NEW_USER"),
					resource.TestCheckNoResourceAttr(accMailRuleResource, "default_reporter_login"),
					resource.TestCheckNoResourceAttr(accMailRuleResource, "folder"),
					resource.TestCheckNoResourceAttr(accMailRuleResource, "filters"),
					resource.TestCheckResourceAttr(accMailRuleResource, "disabled", "false"),
				),
			},
		},
	})
}
