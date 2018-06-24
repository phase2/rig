package commands

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/urfave/cli"

	"github.com/phase2/rig/util"
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
		return cmd.Failure(err.Error(), "COMMAND-ERROR", 13)
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
func (cmd *DNSRecords) LoadRecords() ([]map[string]interface{}, error) {
	ip, err := util.Command("docker", "inspect", "--format", "{{.NetworkSettings.IPAddress}}", "dnsdock").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to discover dnsdock IP address: %s", err)
	}

	response, err := http.Get(fmt.Sprintf("http://%s/services", strings.Trim(string(ip), "\n")))
	if err != nil || response.StatusCode != 200 {
		return nil, fmt.Errorf("response from dnsdock: %s", err)
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to get response from dnsdock. %s", err)
	}

	js, err := simplejson.NewJson(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dnsdock JSON: %s", err)
	}

	dnsdockMap, _ := js.Map()
	records := []map[string]interface{}{}
	for id, value := range dnsdockMap {
		record := value.(map[string]interface{})
		record["Id"] = id
		records = append(records, record)
	}
	return records, nil
}
