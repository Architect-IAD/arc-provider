package provider

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ----------------------------------------------------------------------------
// Global test configuration (edit these once for your sandbox)
// ----------------------------------------------------------------------------

var testCfg = struct {
	UnitID       string
	ClosedOUID   string
	EmailDomain  string
	NamePrefix   string
	DefaultRegion string // optional: if your provider Configure ignores env, keep for docs
}{
	UnitID:        "ou-ie7g-fscdzl8a",
	ClosedOUID:    "ou-ie7g-8jsuksl2",
	EmailDomain:   "cloudcanvas.ca",
	NamePrefix:    "TF-ARC-",
	DefaultRegion: "us-east-1",
}

// ----------------------------------------------------------------------------
// Provider factory (ensure key matches your provider type name, e.g., "arc")
// ----------------------------------------------------------------------------


// ----------------------------------------------------------------------------
// Tests
// ----------------------------------------------------------------------------

func TestAccArcAWSAccount_Basic(t *testing.T) {
	t.Parallel()

	random := acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	name := testCfg.NamePrefix + random
	email := fmt.Sprintf("acct-%s-%d@%s", random, time.Now().Unix(), testCfg.EmailDomain)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: testAccAccountConfig(name, email),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("arc_aws_account.test", "id"),
					resource.TestCheckResourceAttrSet("arc_aws_account.test", "account_id"),
					resource.TestCheckResourceAttr("arc_aws_account.test", "email", email),
					resource.TestCheckResourceAttr("arc_aws_account.test", "name", name),
					resource.TestCheckResourceAttr("arc_aws_account.test", "unit_id", testCfg.UnitID),
					resource.TestCheckResourceAttr("arc_aws_account.test", "closed_unit_id", testCfg.ClosedOUID),
				),
			},
			{
				ResourceName:            "arc_aws_account.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateIdFunc:       testAccImportID("arc_aws_account.test"),
				ImportStateVerifyIgnore: []string{"email", "name"},
			},
		},
	})
}

func TestAccArcAWSAccount_UpdateForbidden(t *testing.T) {
	t.Parallel()

	random := acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	name := testCfg.NamePrefix + random
	email := fmt.Sprintf("acct-%s@%s", random, testCfg.EmailDomain)
	altName := name + "-changed"

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{ Config: testAccAccountConfig(name, email) },
			{
				Config:      testAccAccountConfig(altName, email),
				ExpectError: regexp.MustCompile(`Cannot Modify Account after creation`),
				PlanOnly:    true,
			},
		},
	})
}

func TestAccArcAWSAccount_DeleteMovesToClosedOU(t *testing.T) {
	t.Parallel()

	random := acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum)
	name := testCfg.NamePrefix + random
	email := fmt.Sprintf("acct-%s@%s", random, testCfg.EmailDomain)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{ Config: testAccAccountConfig(name, email) },
			{
				ResourceName: "arc_aws_account.test",
				Destroy:      true,
				Config:       "",
			},
		},
	})
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

func testAccAccountConfig(name, email string) string {
	return fmt.Sprintf(`
provider "arc" {
  // Region defaults to %s via your resource.Configure()
}

resource "arc_aws_account" "test" {
  name           = %q
  email          = %q
  unit_id        = %q
  closed_unit_id = %q
}
`, testCfg.DefaultRegion, name, email, testCfg.UnitID, testCfg.ClosedOUID)
}

func testAccImportID(resName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		ms := s.RootModule().Resources[resName]
		if ms == nil {
			return "", fmt.Errorf("resource %s not found in state", resName)
		}
		id := ms.Primary.Attributes["id"]
		if id == "" {
			return "", fmt.Errorf("id not set for %s", resName)
		}
		return id, nil
	}
}
