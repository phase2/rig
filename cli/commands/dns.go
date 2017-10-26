package commands

import (
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"fmt"
	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Dns struct {
	BaseCommand
}

func (cmd *Dns) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:  "dns",
			Usage: "Start DNS services on the docker-machine",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "nameservers",
					Value:  "8.8.8.8:53",
					Usage:  "Comma separated list of fallback names servers.",
					EnvVar: "RIG_NAMESERVERS",
				},
			},
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

func (cmd *Dns) Run(c *cli.Context) error {
	if cmd.machine.IsRunning() {
		cmd.out.Info.Println("Configuring DNS")
		cmd.ConfigureDns(cmd.machine, c.String("nameservers"))
		cmd.ConfigureRoutes(cmd.machine)
	} else {
		return cmd.Error(fmt.Sprintf("Machine '%s' is not running.", cmd.machine.Name), "MACHINE-STOPPED", 12)
	}

	return cmd.Success("DNS Services have been started")
}

// Remove the host filter from the xhyve bridge interface
func (cmd *Dns) RemoveHostFilter(ipAddr string) {
	// #1: route -n get <machineIP> to find the interface name
	routeData, err := exec.Command("route", "-n", "get", ipAddr).CombinedOutput()
	if err != nil {
		cmd.out.Warning.Println("Unable to determine bridge interface to remove hostfilter")
		return
	}
	ifaceRegexp := regexp.MustCompile(`interface:\s(\w+)`)
	iface := ifaceRegexp.FindStringSubmatch(string(routeData))[1]

	// #2: ifconfig <interface name> to get the details
	ifaceData, err := exec.Command("ifconfig", iface).CombinedOutput()
	if err != nil {
		cmd.out.Warning.Println("Unable to determine member to remove hostfilter")
		return
	}
	memberRegexp := regexp.MustCompile(`member:\s(\w+)\s`)
	member := memberRegexp.FindStringSubmatch(string(ifaceData))[1]

	// #4: ifconfig <bridge> -hostfilter <member>
	util.StreamCommand(exec.Command("sudo", "ifconfig", iface, "-hostfilter", member))
}

func (cmd *Dns) ConfigureRoutes(machine Machine) {
	cmd.out.Info.Println("Setting up local networking (may require your admin password)")

	machineIp := machine.GetIP()
	bridgeIp := machine.GetBridgeIP()
	if runtime.GOOS == "windows" {
		exec.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.0.0").Run()
		util.StreamCommand(exec.Command("runas", "/noprofile", "/user:Administrator", "route", "-p", "ADD", "172.17.0.0/16", machineIp))
	} else {
		if machine.IsXhyve() {
			cmd.RemoveHostFilter(machine.GetIP())
		}
		exec.Command("sudo", "mkdir", "-p", "/etc/resolver").Run()
		exec.Command("bash", "-c", "echo \"nameserver "+bridgeIp+"\" | sudo tee /etc/resolver/vm").Run()
		exec.Command("sudo", "route", "-n", "delete", "-net", "172.17.0.0").Run()
		util.StreamCommand(exec.Command("sudo", "route", "-n", "add", "172.17.0.0/16", machineIp))

		if _, err := os.Stat("/usr/sbin/discoveryutil"); err == nil {
			// Put this here for people running OS X 10.10.0 to 10.10.3 (oy vey.)
			cmd.out.Verbose.Println("Restarting discoveryutil to flush DNS caches")
			util.StreamCommand(exec.Command("sudo", "launchctl", "unload", "-w", "/System/Library/LaunchDaemons/com.apple.discoveryd.plist"))
			util.StreamCommand(exec.Command("sudo", "launchctl", "load", "-w", "/System/Library/LaunchDaemons/com.apple.discoveryd.plist"))
		} else {
			// Reset DNS cache. We have seen this suddenly make /etc/resolver/vm work.
			cmd.out.Verbose.Println("Restarting mDNSResponder to flush DNS caches")
			util.StreamCommand(exec.Command("sudo", "killall", "-HUP", "mDNSResponder"))
		}
	}
}

func (cmd *Dns) ConfigureDns(machine Machine, nameservers string) {
	dnsServers := strings.Split(nameservers, ",")

	machine.SetEnv()
	bridgeIp := machine.GetBridgeIP()

	// Start dnsdock
	exec.Command("docker", "stop", "dnsdock").Run()
	exec.Command("docker", "rm", "dnsdock").Run()

	args := []string{
		"run",
		"-d",
		"--restart=always",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-l", "com.dnsdock.name=dnsdock",
		"-l", "com.dnsdock.image=outrigger",
		"--name", "dnsdock",
		"-p", bridgeIp + ":53:53/udp",
		"aacebedo/dnsdock:v1.16.1-amd64",
		"--domain=vm",
	}
	for _, server := range dnsServers {
		args = append(args, "--nameserver="+server)
	}
	util.ForceStreamCommand(exec.Command("docker", args...))
}
