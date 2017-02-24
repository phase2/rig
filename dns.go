package main

import (
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/urfave/cli"
)

type Dns struct{}

func (cmd *Dns) Commands() cli.Command {
	return cli.Command{
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
		Action: cmd.Run,
	}
}

func (cmd *Dns) Run(c *cli.Context) error {
	if machine.IsRunning() {
		out.Info.Println("Configuring DNS")
		cmd.ConfigureDns(machine, c.String("nameservers"))
		cmd.ConfigureRoutes(machine)
	} else {
		out.Error.Fatalf("Machine '%s' is not running.", machine.Name)
	}

	return nil
}

// Remove the host filter from the xhyve bridge interface
func (cmd Dns) RemoveHostFilter(ipAddr string) {
	// #1: route -n get <machineIP> to find the interface name
	routeData, err := exec.Command("route", "-n", "get", ipAddr).CombinedOutput()
	if err != nil {
		out.Warning.Println("Unable to determine bridge interface to remove hostfilter")
		return
	}
	ifaceRegexp := regexp.MustCompile(`interface:\s(\w+)`)
	iface := ifaceRegexp.FindStringSubmatch(string(routeData))[1]

	// #2: ifconfig <interface name> to get the details
	ifaceData, err := exec.Command("ifconfig", iface).CombinedOutput()
	if err != nil {
		out.Warning.Println("Unable to determine member to remove hostfilter")
		return
	}
	memberRegexp := regexp.MustCompile(`member:\s(\w+)\s`)
	member := memberRegexp.FindStringSubmatch(string(ifaceData))[1]

	// #4: ifconfig <bridge> -hostfilter <member>
	StreamCommand(exec.Command("sudo", "ifconfig", iface, "-hostfilter", member))
}

func (cmd Dns) ConfigureRoutes(machine Machine) {
	out.Info.Println("Setting up local networking (may require your admin password)")

	machineIp := machine.GetIP()
	bridgeIp := machine.GetBridgeIP()
	if runtime.GOOS == "windows" {
		StreamCommand(exec.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.0.0"))
		StreamCommand(exec.Command("runas", "/noprofile", "/user:Administrator", "route", "-p", "ADD", "172.17.0.0/16", machineIp))

		// Delete this in version > 0.4.x
		StreamCommand(exec.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.42.1"))
	} else {
		if machine.IsXhyve() {
			cmd.RemoveHostFilter(machine.GetIP())
		}
		exec.Command("sudo", "mkdir", "-p", "/etc/resolver").Run()
		exec.Command("bash", "-c", "echo \"nameserver "+bridgeIp+"\" | sudo tee /etc/resolver/vm").Run()
		StreamCommand(exec.Command("sudo", "route", "-n", "delete", "-net", "172.17.0.0"))
		StreamCommand(exec.Command("sudo", "route", "-n", "add", "172.17.0.0/16", machineIp))

		// Delete this in version > 0.4.x
		StreamCommand(exec.Command("sudo", "route", "-n", "delete", "-net", "172.17.42.1"))

		if _, err := os.Stat("/usr/sbin/discoveryutil"); err == nil {
			// Put this here for people running OS X 10.10.0 to 10.10.3 (oy vey.)
			out.Info.Println("Restarting discoveryutil to flush DNS caches")
			StreamCommand(exec.Command("sudo", "launchctl", "unload", "-w", "/System/Library/LaunchDaemons/com.apple.discoveryd.plist"))
			StreamCommand(exec.Command("sudo", "launchctl", "load", "-w", "/System/Library/LaunchDaemons/com.apple.discoveryd.plist"))
		} else {
			// Reset DNS. We have seen this suddenly make /etc/resolver/vm work.
			out.Verbose.Println("Restarting mDNSResponder to flush DNS caches")
			StreamCommand(exec.Command("sudo", "launchctl", "unload", "-w", "/System/Library/LaunchDaemons/com.apple.mDNSResponder.plist"))
			StreamCommand(exec.Command("sudo", "launchctl", "load", "-w", "/System/Library/LaunchDaemons/com.apple.mDNSResponder.plist"))
		}
	}
}

func (cmd Dns) ConfigureDns(machine Machine, nameservers string) {
	dnsServers := strings.Split(nameservers, ",")

	machine.SetEnv()
	bridgeIp := machine.GetBridgeIP()

	// Start dnsdock
	StreamCommand(exec.Command("docker", "stop", "dnsdock"))
	StreamCommand(exec.Command("docker", "rm", "dnsdock"))

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
	StreamCommand(exec.Command("docker", args...))
}
