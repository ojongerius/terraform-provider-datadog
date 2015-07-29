package datadog

import (
	"fmt"
	"testing"
	"strconv"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zorkian/go-datadog-api"
)

func TestAccDatadogDashboard_Basic(t *testing.T) {
	var resp datadog.Dashboard

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckDatadogDashboardDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCheckDatadogDashboardConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDatadogDashboardExists("datadog_dashboard.foo", &resp),
					testAccCheckDatadogDashboardAttributes(&resp),
					resource.TestCheckResourceAttr(
						"datadog_dashboard.foo", "name", "terraform_example_dashboard"),
					resource.TestCheckResourceAttr(
						"datadog_dashboard.foo", "title", "bar"),
					resource.TestCheckResourceAttr(
						"datadog_dashboard.foo", "description", "baz"),
					resource.TestCheckResourceAttr(
						"datadog_dashboard.foo", "graphs", ""),
				),
			},
		},
	})
}

func testAccCheckDatadogDashboardDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*datadog.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "datadog_dashboard" {
			continue
		}

		IdInt, int_err := strconv.Atoi(rs.Primary.ID)
		if int_err == nil {
			return int_err
		}

		_, err := client.GetDashboard(IdInt)

		if err == nil {
			return fmt.Errorf("Dashboard still exists")
		}
	}

	return nil
}

func testAccCheckDatadogDashboardAttributes(DashboardResp *datadog.Dashboard) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if DashboardResp.Title != "bar" {
			return fmt.Errorf("Bad dashboard_title: %s", DashboardResp.Title)
		}

		if DashboardResp.Description != "baz" {
			return fmt.Errorf("Bad dashboard_description: %s", DashboardResp.Title)
		}

		// TODO: should be a list
		if len(DashboardResp.Graphs) != 0 {
			return fmt.Errorf("Bad dashboard_graphs : %s", DashboardResp.Title)
		}

		return nil
	}
}

func testAccCheckDatadogDashboardExists(n string, DashboardResp *datadog.Dashboard) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Domain ID is set")
		}

		client := testAccProvider.Meta().(*datadog.Client)

		IntId, int_err := strconv.Atoi(rs.Primary.ID)

		if int_err != nil {
			return int_err
		}

		resp, err := client.GetDashboard(IntId)

		if err != nil {
			return err
		}

		// TODO: fix this one.
		//if resp.Dashboard.name != rs.Primary.ID {
			//return fmt.Errorf("Domain not found")
		//}

		DashboardResp = resp

		return nil
	}
}

const testAccCheckDatadogDashboardConfig_basic = `
resource "datadog_dashboard" "foo" {
    name = "terraform_example_dashboard"
    title = "bar"
    description = "baz"
    graphs = ""
}`
