package commands

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/urfave/cli"
)

type DnsRecords struct {
	BaseCommand
}

func (cmd *DnsRecords) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "dns-records",
			Usage:  "List all DNS records for running containers",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

func (cmd *DnsRecords) Run(c *cli.Context) error {

	records, err := cmd.LoadRecords()
	if err != nil {
		cmd.out.Error.Fatalln(err)
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

	return nil
}

// Get the records from DNSDock and process/return them
func (cmd *DnsRecords) LoadRecords() ([]map[string]interface{}, error) {
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
					return nil, errors.New(fmt.Sprintf("Failed to parse dnsdock JSON: %s", jsonErr))
				}
			} else {
				return nil, errors.New(fmt.Sprintf("Unable to get response from dnsdock. %s", readErr))
			}
		} else {
			return nil, errors.New(fmt.Sprintf("Response from dnsdock: %s", httpErr))
		}
	} else {
		return nil, errors.New(fmt.Sprintf("Failed to discover dnsdock IP address: %s", ipErr))
	}
}
