package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	accEnabled              = "1"
	envAccEnabled           = "TF_ACC"
	envAccURL               = "YOUTRACK_URL"
	envAccToken             = "YOUTRACK_TOKEN"
	envAccUser              = "YOUTRACK_TEST_USER"
	envAccPermission        = "YOUTRACK_TEST_PERMISSION"
	envAccGroupName         = "YOUTRACK_TEST_GROUP_NAME"
	envAccCustomFieldTypeID = "YOUTRACK_TEST_CUSTOM_FIELD_TYPE_ID"

	envOAuth2ClientID     = "YOUTRACK_OAUTH2_CLIENT_ID"
	envOAuth2ClientSecret = "YOUTRACK_OAUTH2_CLIENT_SECRET" //nolint:gosec // env var name, not a credential
	envOAuth2ServerURL    = "YOUTRACK_OAUTH2_SERVER_URL"
	envOAuth2TokenURL     = "YOUTRACK_OAUTH2_TOKEN_URL" //nolint:gosec // env var name, not a credential
	envOAuth2UserInfoURL  = "YOUTRACK_OAUTH2_USER_INFO_URL"
	envOAuth2UserIDPath   = "YOUTRACK_OAUTH2_USER_ID_PATH"
)

func testProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"youtrack": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func skipUnlessAcc(t *testing.T) {
	t.Helper()

	if os.Getenv(envAccEnabled) != accEnabled {
		t.Skip("acceptance tests skipped unless TF_ACC=1")
	}

	if os.Getenv(envAccURL) == "" || os.Getenv(envAccToken) == "" {
		t.Skip("set YOUTRACK_URL and YOUTRACK_TOKEN to run acceptance tests")
	}
}

func providerBlock() string {
	return `
provider "youtrack" {
  base_url = "` + os.Getenv(envAccURL) + `"
  token    = "` + os.Getenv(envAccToken) + `"
}
`
}

// importStateID returns an ImportStateIdFunc that reads the id attribute
// from the named resource in the current Terraform state.
func importStateID(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %q not found in state", resourceName)
		}
		val, ok := rs.Primary.Attributes["id"]
		if !ok {
			return "", fmt.Errorf("attribute %q not found on resource %q", "id", resourceName)
		}
		return val, nil
	}
}
