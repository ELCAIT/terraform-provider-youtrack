package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	accMailboxResource = "youtrack_mailbox.test"
	accMailboxPassword = "tf-acc-password" //nolint:gosec // test fixture, not a credential
)

// testAccIMAPMailboxConfig is an IMAPS mailbox. YouTrack does not connect to
// the server when it saves a mailbox, so the host does not need to exist.
func testAccIMAPMailboxConfig(host string, port int, login string, timeout int) string {
	return fmt.Sprintf(`
resource "youtrack_mailbox" "test" {
  protocol           = "IMAPS"
  host               = %q
  port               = %d
  login              = %q
  password           = %q
  connection_timeout = %d
}
`, host, port, login, accMailboxPassword, timeout)
}

func TestAccMailbox(t *testing.T) {
	skipUnlessAcc(t)

	login := fmt.Sprintf("tf-acc-%d@example.com", time.Now().UnixMilli())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock() + testAccIMAPMailboxConfig("imap.example.com", 993, login, 30),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(accMailboxResource, "id"),
					resource.TestCheckResourceAttr(accMailboxResource, "protocol", "IMAPS"),
					resource.TestCheckResourceAttr(accMailboxResource, "host", "imap.example.com"),
					resource.TestCheckResourceAttr(accMailboxResource, "port", "993"),
					resource.TestCheckResourceAttr(accMailboxResource, "login", login),
					resource.TestCheckResourceAttr(accMailboxResource, "socket_timeout", "30"),
					resource.TestCheckResourceAttr(accMailboxResource, "exchange_compatibility_mode", "false"),
				),
			},
			{
				// Changing the host and port needs the password in the same update.
				Config: providerBlock() + testAccIMAPMailboxConfig("mail.example.com", 143, login, 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accMailboxResource, "host", "mail.example.com"),
					resource.TestCheckResourceAttr(accMailboxResource, "port", "143"),
					resource.TestCheckResourceAttr(accMailboxResource, "connection_timeout", "60"),
				),
			},
			{
				ResourceName:            accMailboxResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func TestAccMailboxOffice365(t *testing.T) {
	skipUnlessAcc(t)

	address := fmt.Sprintf("tf-acc-%d@example.com", time.Now().UnixMilli())
	config := func(tenant string) string {
		return providerBlock() + fmt.Sprintf(`
resource "youtrack_mailbox" "test" {
  protocol                  = "OFFICE365"
  microsoft_mailbox_address = %q
  microsoft_client_id       = "tf-acc-client"
  microsoft_tenant_id       = %q
  microsoft_client_secret   = "tf-acc-secret"
}
`, address, tenant)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("11111111-1111-1111-1111-111111111111"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accMailboxResource, "microsoft_mailbox_address", address),
					resource.TestCheckNoResourceAttr(accMailboxResource, "host"),
				),
			},
			{
				// Changing the tenant needs the client secret in the same update.
				Config: config("22222222-2222-2222-2222-222222222222"),
				Check:  resource.TestCheckResourceAttr(accMailboxResource, "microsoft_tenant_id", "22222222-2222-2222-2222-222222222222"),
			},
			{
				ResourceName:            accMailboxResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"microsoft_client_secret"},
			},
		},
	})
}
