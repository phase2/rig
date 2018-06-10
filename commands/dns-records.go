package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/urfave/cli"

	"github.com/phase2/rig/util"
)

// DNSRecords is the command for exporting all DNS Records in Outrigger DNS in `hosts` file format
type DNSRecords struct {
	BaseCommand
}

// DNSRecord is the struct for a single DNS entry
type DNSRecord struct {
	Id      string // nolint
	Name    string
	Image   string
	IPs     []string
	TTL     int64
	Aliases []string
}

// DNSRecordsList is an array of DNSRecords
type DNSRecordsList []*DNSRecord

const (
	unixHostsPreamble  = "##+++ added by rig"
	unixHostsPostamble = "##--- end rig additions"
)

func (record *DNSRecord) String() string {
	result := ""
	for _, ip := range record.IPs {
		result += fmt.Sprintf("%s\t%s.%s.%s\n", ip, record.Name, record.Image, "vm")
		// attach any aliases too
		for _, a := range record.Aliases {
			result += fmt.Sprintf("%s\t%s\n", ip, a)
		}
	}
	return result
}

// String converts a list of DNSRecords to a formatted string
func (hosts DNSRecordsList) String() string {
	result := ""
	for _, host := range hosts {
		result += host.String()
	}
	return result
}

// Commands returns the operations supported by this command
func (cmd *DNSRecords) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:  "dns-records",
			Usage: "List all DNS records for running containers",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "save",
					Usage: "Save the DNS records to /etc/hosts or {FIX insert Windows desscription}",
				},
				cli.BoolFlag{
					Name:  "remove",
					Usage: "Remove the DNS records from /etc/hosts or {FIX insert Windows desscription}",
				},
			},
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig dns-records` command
func (cmd *DNSRecords) Run(c *cli.Context) error {
	// Don't require rig to be started to remove records
	if c.Bool("remove") {
		if c.Bool("save") {
			return cmd.Failure("--remove and --save are mutually exclusive", "COMMAND-ERROR", 13)
		}
		// TODO The VM might have to be up for Windows
		return cmd.removeDNSRecords()
	}

	records, err := cmd.LoadRecords()
	if err != nil {
		return cmd.Failure(err.Error(), "COMMAND-ERROR", 13)
	}

	if c.Bool("save") {
		return cmd.saveDNSRecords(records)
	}

	printDNSRecords(records)

	return cmd.Success("")
}

// LoadRecords retrieves the records from DNSDock and processes/return them
func (cmd *DNSRecords) LoadRecords() ([]*DNSRecord, error) {
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
	records := make([]*DNSRecord, 0, 20)
	for id, rawValue := range dnsdockMap {
		// Cast rawValue to its actual type
		value := rawValue.(map[string]interface{})
		ttl, _ := value["TTL"].(json.Number).Int64()
		record := &DNSRecord{
			Id:    id,
			Name:  value["Name"].(string),
			Image: value["Image"].(string),
			TTL:   ttl,
		}
		record.IPs = make([]string, 0, 10)
		for _, ip := range value["IPs"].([]interface{}) {
			record.IPs = append(record.IPs, ip.(string))
		}
		record.Aliases = make([]string, 0, 10)
		for _, alias := range value["Aliases"].([]interface{}) {
			record.Aliases = append(record.Aliases, alias.(string))
		}
		records = append(records, record)
	}
	return records, nil
}

func printDNSRecords(records []*DNSRecord) {
	for _, record := range records {
		fmt.Print(record)
	}
}

// Write the records to /etc/hosts or FIX Windows?
func (cmd *DNSRecords) saveDNSRecords(records []*DNSRecord) error {
	if util.IsMac() || util.IsLinux() {
		return cmd.saveDNSRecordsUnix(records)
	} else if util.IsWindows() {
		return cmd.saveDNSRecordsWindows(records)
	}
	return cmd.Success("Not implemented")
}

// ⚠ Administrative privileges needed...

func (cmd *DNSRecords) saveDNSRecordsUnix(records []*DNSRecord) error {
	// Both of these are []string
	oldHostEntries := util.LoadFile("/etc/hosts")
	newHostEntries := stripDNS(oldHostEntries)
	// records.String does the formatting, so convert both to a string
	oldHosts := strings.Join(oldHostEntries, "\n")
	newHosts := strings.Join(newHostEntries, "\n") + "\n" +
		unixHostsPreamble + "\n" +
		DNSRecordsList(records).String() +
		unixHostsPostamble + "\n"
	if oldHosts == newHosts {
		return cmd.Success("No changes made")
	}
	return cmd.writeEtcHosts(newHosts)
}

func (cmd *DNSRecords) saveDNSRecordsWindows(records []*DNSRecord) error {
	return cmd.Failure("Not Implemented", "COMMAND-ERROR", 13)
}

func (cmd *DNSRecords) removeDNSRecords() error {
	if util.IsMac() || util.IsLinux() {
		return cmd.removeDNSRecordsUnix()
	} else if util.IsWindows() {
		return cmd.removeDNSRecordsWindows()
	}
	return cmd.Success("Not implemented")
}

func (cmd *DNSRecords) removeDNSRecordsUnix() error {
	oldHostsEntries := util.LoadFile("/etc/hosts")
	newHostsEntries := stripDNS(oldHostsEntries)
	oldHosts := strings.Join(oldHostsEntries, "\n")
	newHosts := strings.Join(newHostsEntries, "\n")
	if oldHosts == newHosts {
		return cmd.Success("No changes made")
	}
	return cmd.writeEtcHosts(newHosts)
}

func (cmd *DNSRecords) removeDNSRecordsWindows() error {
	return cmd.Failure("Not Implemented", "COMMAND-ERROR", 13)
}

// Save a new version of /etc/hosts, arg is the full text to save
func (cmd *DNSRecords) writeEtcHosts(hostsText string) error {
	// Make sure it ends in a newline
	if hostsText[len(hostsText)-1] != '\n' {
		hostsText += "\n"
	}
	// Write new version to a temp file
	tmpfile, err := ioutil.TempFile("", "rig-hosts")
	if err != nil {
		return cmd.Failure("Unable to create hosts tempfile: "+err.Error(), "COMMAND-ERROR", 13)
	}
	tmpname := tmpfile.Name()
	defer os.Remove(tmpname)
	if _, err := tmpfile.Write([]byte(hostsText)); err != nil {
		return cmd.Failure("Unable to write hosts tempfile: ("+tmpname+") "+err.Error(), "COMMAND-ERROR", 13)
	}
	if err := tmpfile.Close(); err != nil {
		return cmd.Failure("Unable to close hosts tempfile: ("+tmpname+") "+err.Error(), "COMMAND-ERROR", 13)
	}
	// mv it into place. This is safer than trying to write /etc/hosts on the fly.
	if err := util.EscalatePrivilege(); err != nil {
		return cmd.Failure("Unable to obtain privileges to replace: "+err.Error(), "COMMAND-ERROR", 13)
	}
	if err := util.Command("sudo", "mv", "-f", tmpname, "/etc/hosts").Run(); err != nil {
		return cmd.Failure("Unable to replace /etc/hosts: "+err.Error(), "COMMAND-ERROR", 13)
	}
	return cmd.Success("/etc/hosts updated")
}

// Remove a section of the hosts file we previously added
func stripDNS(hosts []string) []string {
	const (
		looking = iota
		found
	)
	results := make([]string, 0, 1000)
	state := looking
	for _, host := range hosts {
		switch state {
		case looking:
			if host == unixHostsPreamble {
				state = found
			} else {
				results = append(results, host)
			}
		case found:
			if host == unixHostsPostamble {
				state = looking
			}
		}
	}
	return results
}
