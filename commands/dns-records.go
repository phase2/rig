package commands

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/urfave/cli"
)

// DNSRecords is the command for exporting all DNS Records in Outrigger DNS in `hosts` file format
type DNSRecords struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *DNSRecords) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "dns-records",
			Usage:  "List all DNS records for running containers",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig dns-records` command
func (cmd *DNSRecords) Run(c *cli.Context) error {

	records, err := cmd.LoadRecords()
	if err != nil {
		return cmd.Error(err.Error(), "COMMAND-ERROR", 13)
	}

	for _, record := range records {
		for _, ip := range record["IPs"].([]interface{}) {
			fmt.Printf("%s\t%s.%s.%s\n", ip, record["Name"], record["Image"], "vm")
			// attach any aliases too
			for _, a := range record["Aliases"].([]interface{}) {
				fmt.Printf("%s\t%s\n", ip, a)
			}
		}
	}

	return cmd.Success("")
}

// LoadRecords retrieves the records from DNSDock and processes/return them
// nolint: golint
func (cmd *DNSRecords) LoadRecords() ([]map[string]interface{}, error) {
	if ip, ipErr := exec.Command("docker", "inspect", "--format", "{{.NetworkSettings.IPAddress}}", "dnsdock").Output(); ipErr == nil {
		if response, httpErr := http.Get(fmt.Sprintf("http://%s/services", strings.Trim(string(ip), "\n"))); httpErr == nil && response.StatusCode == 200 {
			defer response.Body.Close()
			if body, readErr := ioutil.ReadAll(response.Body); readErr == nil {
				if js, jsonErr := simplejson.NewJson(body); jsonErr == nil {
					dnsdockMap, _ := js.Map()
					records := []map[string]interface{}{}
					for id, value := range dnsdockMap {
						record := value.(map[string]interface{})
						record["Id"] = id
						records = append(records, record)
					}
					return records, nil
				} else {
					return nil, fmt.Errorf("Failed to parse dnsdock JSON: %s", jsonErr)
				}
			} else {
				return nil, fmt.Errorf("Unable to get response from dnsdock. %s", readErr)
			}
		} else {
			return nil, fmt.Errorf("Response from dnsdock: %s", httpErr)
		}
	} else {
		return nil, fmt.Errorf("Failed to discover dnsdock IP address: %s", ipErr)
	}
}
