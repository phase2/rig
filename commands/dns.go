package commands

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// DNS is the command for starting all DNS services and appropriate network routing to access services
type DNS struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *DNS) Commands() []cli.Command {
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

// Run executes the `rig dns` command
func (cmd *DNS) Run(c *cli.Context) error {
	cmd.out.Info.Println("Configuring DNS")

	if util.SupportsNativeDocker() {
		cmd.StartDNS(cmd.machine, c.String("nameservers"))
		return cmd.ConfigureLinux()
	} else if cmd.machine.IsRunning() {
		cmd.StartDNS(cmd.machine, c.String("nameservers"))
		cmd.ConfigureDNS(cmd.machine)
	} else {
		return cmd.Error(fmt.Sprintf("Machine '%s' is not running.", cmd.machine.Name), "MACHINE-STOPPED", 12)
	}

	return cmd.Success("DNS Services have been started")
}

// RemoveHostFilter removs the host filter from the xhyve bridge interface
func (cmd *DNS) RemoveHostFilter(ipAddr string) {
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

// ConfigureDNS will configure routing to allow access to containers on IP addresses
// within the Docker Machine bridge network
func (cmd *DNS) ConfigureDNS(machine Machine) {
	cmd.out.Info.Println("Setting up local networking (may require your admin password)")

	if util.IsWindows() {
		cmd.ConfigureWindows(machine)
	} else if util.IsMac() {
		cmd.ConfigureMac(machine)
	}
}

// ConfigureWindows configures network routing
func (cmd *DNS) ConfigureWindows(machine Machine) {
	exec.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.0.0").Run()
	util.StreamCommand(exec.Command("runas", "/noprofile", "/user:Administrator", "route", "-p", "ADD", "172.17.0.0/16", machine.GetIP()))
}

// ConfigureMac configures DNS resolution and network routing
func (cmd *DNS) ConfigureMac(machine Machine) {
	machineIP := machine.GetIP()
	bridgeIP := machine.GetBridgeIP()

	if machine.IsXhyve() {
		cmd.RemoveHostFilter(machine.GetIP())
	}
	exec.Command("sudo", "mkdir", "-p", "/etc/resolver").Run()
	exec.Command("bash", "-c", "echo \"nameserver "+bridgeIP+"\" | sudo tee /etc/resolver/vm").Run()
	exec.Command("sudo", "route", "-n", "delete", "-net", "172.17.0.0").Run()
	util.StreamCommand(exec.Command("sudo", "route", "-n", "add", "172.17.0.0/16", machineIP))
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

// ConfigureLinux configures network routing and DNS resolution
func (cmd *DNS) ConfigureLinux() error {
	bridgeIP, err := util.GetBridgeIP()
	if err != nil {
		return err
	}

	// Is NetworkManager in use
	if _, err := os.Stat("/etc/NetworkManager/dnsmasq.d"); err == nil {
		// Install for NetworkManager/dnsmasq connection to dnsdock
		util.StreamCommand(exec.Command("bash", "-c", fmt.Sprintf("echo 'server=/vm/%s' | sudo tee /etc/NetworkManager/dnsmasq.d/dnsdock.conf", bridgeIP)))

		// Restart NetworkManager if it is running
		if err := exec.Command("systemctl", "is-active", "NetworkManager").Run(); err != nil {
			util.StreamCommand(exec.Command("sudo", "systemctl", "restart", "NetworkManager"))
		}
	}

	// Is libnss-resolver in use
	if _, err := os.Stat("/etc/resolver"); err == nil {
		// Install for libnss-resolver connection to dnsdock
		exec.Command("bash", "-c", fmt.Sprintf("echo 'nameserver %s' | sudo tee /etc/resolver/vm", bridgeIP)).Run()
	}

	return nil
}


// StartDNS will start the dnsdock service
func (cmd *DNS) StartDNS(machine Machine, nameservers string) {
	dnsServers := strings.Split(nameservers, ",")

	// Linux uses standard bridge IP
	// May need to make this configurable is there are local linux/docker customizations?
	var bridgeIP = "172.17.0.1"
	if !util.SupportsNativeDocker() {
		machine.SetEnv()
		bridgeIP = machine.GetBridgeIP()
	}

	cmd.StopDNS()

	// Start dnsdock
	args := []string{
		"run",
		"-d",
		"--restart=always",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-l", "com.dnsdock.name=dnsdock",
		"-l", "com.dnsdock.image=outrigger",
		"--name", "dnsdock",
		"-p", fmt.Sprintf("%s:53:53/udp", bridgeIP),
		"aacebedo/dnsdock:v1.16.1-amd64",
		"--domain=vm",
	}
	for _, server := range dnsServers {
		args = append(args, "--nameserver="+server)
	}
	util.ForceStreamCommand(exec.Command("docker", args...))
}

// StopDNS stops the dnsdock service and cleans up
func (cmd *DNS) StopDNS() {
	exec.Command("docker", "stop", "dnsdock").Run()
	exec.Command("docker", "rm", "dnsdock").Run()
}
