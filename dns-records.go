package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/bitly/go-simplejson"
	"github.com/urfave/cli"
)

type DnsRecords struct{}

func (cmd *DnsRecords) Commands() cli.Command {
	return cli.Command{
		Name:   "dns-records",
		Usage:  "List all DNS records for running containers",
		Action: cmd.Run,
	}
}

func (cmd *DnsRecords) Run(c *cli.Context) error {

	records, err := cmd.LoadRecords()
	if err != nil {
		out.Error.Fatalln(err)
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
	if response, httpErr := http.Get("http://dnsdock.outrigger.vm/services"); httpErr == nil && response.StatusCode == 200 {
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
}
