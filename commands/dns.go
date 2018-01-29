package commands

import (
	"fmt"
	"os"
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
	if !util.SupportsNativeDocker() && !cmd.machine.IsRunning() {
		return cmd.Failure(fmt.Sprintf("Machine '%s' is not running.", cmd.machine.Name), "MACHINE-STOPPED", 12)
	}

	if err := cmd.StartDNS(cmd.machine, c.String("nameservers")); err != nil {
		cmd.out.Error("DNS is ready")
		return cmd.Failure(err.Error(), "DNS-SETUP-FAILED", 13)
	}

	if !util.SupportsNativeDocker() {
		cmd.ConfigureRoutes(cmd.machine)
	}

	return cmd.Success("DNS Services have been started")
}

// ConfigureRoutes will configure routing to allow access to containers on IP addresses
// within the Docker Machine bridge network
func (cmd *DNS) ConfigureRoutes(machine Machine) {
	cmd.out.Spin("Setting up local networking (may require your admin password)")

	if util.IsMac() {
		cmd.configureMacRoutes(machine)
	} else if util.IsWindows() {
		cmd.configureWindowsRoutes(machine)
	}

	cmd.out.Info("Local networking is ready")
}

// ConfigureMac configures DNS resolution and network routing
func (cmd *DNS) configureMacRoutes(machine Machine) {
	machineIP := machine.GetIP()

	if machine.IsXhyve() {
		cmd.removeHostFilter(machineIP)
	}
	util.Command("sudo", "route", "-n", "delete", "-net", "172.17.0.0").Run()
	util.StreamCommand("sudo", "route", "-n", "add", "172.17.0.0/16", machineIP)
	if _, err := os.Stat("/usr/sbin/discoveryutil"); err == nil {
		// Put this here for people running OS X 10.10.0 to 10.10.3 (oy vey.)
		cmd.out.Verbose("Restarting discoveryutil to flush DNS caches")
		util.StreamCommand("sudo", "launchctl", "unload", "-w", "/System/Library/LaunchDaemons/com.apple.discoveryd.plist")
		util.StreamCommand("sudo", "launchctl", "load", "-w", "/System/Library/LaunchDaemons/com.apple.discoveryd.plist")
	} else {
		// Reset DNS cache. We have seen this suddenly make /etc/resolver/vm work.
		cmd.out.Verbose("Restarting mDNSResponder to flush DNS caches")
		util.StreamCommand("sudo", "killall", "-HUP", "mDNSResponder")
	}
}

// removeHostFilter removes the host filter from the xhyve bridge interface
func (cmd *DNS) removeHostFilter(ipAddr string) {
	// #1: route -n get <machineIP> to find the interface name
	routeData, err := util.Command("route", "-n", "get", ipAddr).CombinedOutput()
	if err != nil {
		cmd.out.Warning("Unable to determine bridge interface to remove hostfilter")
		return
	}
	ifaceRegexp := regexp.MustCompile(`interface:\s(\w+)`)
	iface := ifaceRegexp.FindStringSubmatch(string(routeData))[1]

	// #2: ifconfig <interface name> to get the details
	ifaceData, err := util.Command("ifconfig", iface).CombinedOutput()
	if err != nil {
		cmd.out.Warning("Unable to determine member to remove hostfilter")
		return
	}
	memberRegexp := regexp.MustCompile(`member:\s(\w+)\s`)
	member := memberRegexp.FindStringSubmatch(string(ifaceData))[1]

	// #4: ifconfig <bridge> -hostfilter <member>
	util.StreamCommand("sudo", "ifconfig", iface, "-hostfilter", member)
}

// ConfigureWindowsRoutes configures network routing
func (cmd *DNS) configureWindowsRoutes(machine Machine) {
	util.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.0.0").Run()
	util.StreamCommand("runas", "/noprofile", "/user:Administrator", "route", "-p", "ADD", "172.17.0.0/16", machine.GetIP())
}

// StartDNS will start the dnsdock service
func (cmd *DNS) StartDNS(machine Machine, nameservers string) error {
	cmd.out.Spin("Setting up DNS resolver...")
	dnsServers := strings.Split(nameservers, ",")

	bridgeIP, err := util.GetBridgeIP()
	if err != nil {
		return err
	}

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
		"aacebedo/dnsdock:v1.16.4-amd64",
		"--domain=vm",
	}
	for _, server := range dnsServers {
		args = append(args, "--nameserver="+server)
	}

	util.StreamCommand("docker", args...)
	// Configure the resolvers based on platform
	var resolverReturn error
	if util.IsMac() {
		resolverReturn = cmd.configureMacResolver(machine)
	} else if util.IsLinux() {
		resolverReturn = cmd.configureLinuxResolver()
	} else if util.IsWindows() {
		resolverReturn = cmd.configureWindowsResolver(machine)
	}
	cmd.out.Info("DNS resolution is ready")

	return resolverReturn
}

// configureMacResolver configures DNS resolution and network routing
func (cmd *DNS) configureMacResolver(machine Machine) error {
	cmd.out.Verbose("Configuring DNS resolution for macOS")
	bridgeIP := machine.GetBridgeIP()

	if err := util.Command("sudo", "mkdir", "-p", "/etc/resolver").Run(); err != nil {
		return err
	}
	if err := util.Command("bash", "-c", fmt.Sprintf("echo 'nameserver %s' | sudo tee /etc/resolver/vm", bridgeIP)).Run(); err != nil {
		return err
	}
	if _, err := os.Stat("/usr/sbin/discoveryutil"); err == nil {
		// Put this here for people running OS X 10.10.0 to 10.10.3 (oy vey.)
		cmd.out.Verbose("Restarting discoveryutil to flush DNS caches")
		util.StreamCommand("sudo", "launchctl", "unload", "-w", "/System/Library/LaunchDaemons/com.apple.discoveryd.plist")
		util.StreamCommand("sudo", "launchctl", "load", "-w", "/System/Library/LaunchDaemons/com.apple.discoveryd.plist")
	} else {
		// Reset DNS cache. We have seen this suddenly make /etc/resolver/vm work.
		cmd.out.Verbose("Restarting mDNSResponder to flush DNS caches")
		util.StreamCommand("sudo", "killall", "-HUP", "mDNSResponder")
	}
	return nil
}

// configureLinuxResolver configures DNS resolution
func (cmd *DNS) configureLinuxResolver() error {
	cmd.out.Verbose("Configuring DNS resolution for linux")
	bridgeIP, err := util.GetBridgeIP()
	if err != nil {
		return err
	}

	// Is NetworkManager in use
	if _, err := os.Stat("/etc/NetworkManager/dnsmasq.d"); err == nil {
		// Install for NetworkManager/dnsmasq connection to dnsdock
		util.StreamCommand("bash", "-c", fmt.Sprintf("echo 'server=/vm/%s' | sudo tee /etc/NetworkManager/dnsmasq.d/dnsdock.conf", bridgeIP))

		// Restart NetworkManager if it is running
		if err := util.Command("systemctl", "is-active", "NetworkManager").Run(); err != nil {
			util.StreamCommand("sudo", "systemctl", "restart", "NetworkManager")
		}
	}

	// Is libnss-resolver in use
	if _, err := os.Stat("/etc/resolver"); err == nil {
		// Install for libnss-resolver connection to dnsdock
		util.Command("bash", "-c", fmt.Sprintf("echo 'nameserver %s:53' | sudo tee /etc/resolver/vm", bridgeIP)).Run()
	}

	return nil
}

// configureWindowsResolver configures DNS resolution and network routing
func (cmd *DNS) configureWindowsResolver(machine Machine) error {
	// TODO: Figure out Windows resolver configuration
	cmd.out.Verbose("TODO: Configuring DNS resolution for windows")
	return nil
}

// StopDNS stops the dnsdock service and cleans up
func (cmd *DNS) StopDNS() {
	util.Command("docker", "stop", "dnsdock").Run()
	util.Command("docker", "rm", "dnsdock").Run()
}
