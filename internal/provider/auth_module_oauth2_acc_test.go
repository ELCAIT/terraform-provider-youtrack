package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccOAuth2Config(name, clientID, clientSecret, serverURL, tokenURL, userInfoURL, userIDPath string) string {
	return providerBlock() + fmt.Sprintf(`
resource "youtrack_auth_module_oauth2" "test" {
  name          = %q
  client_id     = %q
  client_secret = %q
  server_url    = %q
  token_url     = %q
  user_info_url = %q
  user_id_path  = %q
  is_default    = false
}
`, name, clientID, clientSecret, serverURL, tokenURL, userInfoURL, userIDPath)
}

const accOAuth2ModuleAddr = "youtrack_auth_module_oauth2.test"

func TestAccOAuth2AuthModule(t *testing.T) {
	skipUnlessAcc(t)

	if os.Getenv(envOAuth2ClientID) == "" || os.Getenv(envOAuth2ClientSecret) == "" ||
		os.Getenv(envOAuth2ServerURL) == "" || os.Getenv(envOAuth2TokenURL) == "" ||
		os.Getenv(envOAuth2UserInfoURL) == "" || os.Getenv(envOAuth2UserIDPath) == "" {
		t.Skip("set YOUTRACK_OAUTH2_CLIENT_ID, YOUTRACK_OAUTH2_CLIENT_SECRET, YOUTRACK_OAUTH2_SERVER_URL, YOUTRACK_OAUTH2_TOKEN_URL, YOUTRACK_OAUTH2_USER_INFO_URL, YOUTRACK_OAUTH2_USER_ID_PATH to run OAuth2 acceptance tests")
	}

	suffix := time.Now().UnixMilli()
	name := fmt.Sprintf("tf-acc-oauth2-%d", suffix)
	nameUpdated := fmt.Sprintf("tf-acc-oauth2-updated-%d", suffix)

	clientID := os.Getenv(envOAuth2ClientID)
	clientSecret := os.Getenv(envOAuth2ClientSecret)
	serverURL := os.Getenv(envOAuth2ServerURL)
	tokenURL := os.Getenv(envOAuth2TokenURL)
	userInfoURL := os.Getenv(envOAuth2UserInfoURL)
	userIDPath := os.Getenv(envOAuth2UserIDPath)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccOAuth2Config(name, clientID, clientSecret, serverURL, tokenURL, userInfoURL, userIDPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accOAuth2ModuleAddr, "name", name),
					resource.TestCheckResourceAttr(accOAuth2ModuleAddr, "client_id", clientID),
					resource.TestCheckResourceAttr(accOAuth2ModuleAddr, "server_url", serverURL),
					resource.TestCheckResourceAttr(accOAuth2ModuleAddr, "is_default", "false"),
					resource.TestCheckResourceAttrSet(accOAuth2ModuleAddr, "id"),
				),
			},
			{
				Config: testAccOAuth2Config(nameUpdated, clientID, clientSecret, serverURL, tokenURL, userInfoURL, userIDPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(accOAuth2ModuleAddr, "name", nameUpdated),
				),
			},
			{
				ResourceName:            accOAuth2ModuleAddr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret"},
			},
		},
	})
}
