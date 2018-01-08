package commands

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"errors"
	"github.com/bitly/go-simplejson"
	"github.com/hashicorp/go-version"
	"github.com/phase2/rig/util"
)

// Machine is the struct for encapsulating operations on a Docker Machine
type Machine struct {
	Name        string
	out         *util.RigLogger
	inspectData *simplejson.Json
}

// Create will generate a new Docker Machine configured according to user specification
func (m *Machine) Create(driver string, cpuCount string, memSize string, diskSize string) error {
	m.out.Info("Creating a %s machine named '%s' with CPU(%s) MEM(%s) DISK(%s)...", driver, m.Name, cpuCount, memSize, diskSize)

	boot2dockerURL := "https://github.com/boot2docker/boot2docker/releases/download/v" + util.GetRawCurrentDockerVersion() + "/boot2docker.iso"

	var create util.Executor

	switch driver {
	case util.VirtualBox:
		create = util.Command(
			"docker-machine",
			"create", m.Name,
			"--driver=virtualbox",
			"--virtualbox-boot2docker-url="+boot2dockerURL,
			"--virtualbox-memory="+memSize,
			"--virtualbox-cpu-count="+cpuCount,
			"--virtualbox-disk-size="+diskSize,
			"--virtualbox-host-dns-resolver=true",
			"--engine-opt", "dns=172.17.0.1",
		)
	case util.VMWare:
		create = util.Command(
			"docker-machine",
			"create", m.Name,
			"--driver=vmwarefusion",
			"--vmwarefusion-boot2docker-url="+boot2dockerURL,
			"--vmwarefusion-memory-size="+memSize,
			"--vmwarefusion-cpu-count="+cpuCount,
			"--vmwarefusion-disk-size="+diskSize,
			"--engine-opt", "dns=172.17.0.1",
		)
	case util.Xhyve:
		if err := m.CheckXhyveRequirements(); err != nil {
			return err
		}
		create = util.Command(
			"docker-machine",
			"create", m.Name,
			"--driver=xhyve",
			"--xhyve-boot2docker-url="+boot2dockerURL,
			"--xhyve-memory-size="+memSize,
			"--xhyve-cpu-count="+cpuCount,
			"--xhyve-disk-size="+diskSize,
			"--engine-opt", "dns=172.17.0.1",
		)
	}

	if err := create.Execute(false); err != nil {
		return fmt.Errorf("error creating machine '%s': %s", m.Name, err)
	}

	m.out.Info("Created docker-machine named '%s'...", m.Name)
	return nil
}

// CheckXhyveRequirements verifies that the correct xhyve environment exists
func (m *Machine) CheckXhyveRequirements() error {
	// Is xhyve installed locally
	if err := util.Command("which", "xhyve").Run(); err != nil {
		return fmt.Errorf("xhyve is not installed. Install it with 'brew install xhyve'")
	}

	// Is docker-machine-driver-xhyve installed locally
	if err := util.Command("which", "docker-machine-driver-xhyve").Run(); err != nil {
		return fmt.Errorf("docker-machine-driver-xhyve is not installed. Install it with 'brew install docker-machine-driver-xhyve'")
	}

	return nil
}

// Start boots the Docker Machine
func (m *Machine) Start() error {
	if !m.IsRunning() {
		m.out.Verbose("The machine '%s' is not running, starting...", m.Name)

		if err := util.StreamCommand("docker-machine", "start", m.Name); err != nil {
			return fmt.Errorf("error starting machine '%s': %s", m.Name, err)
		}

		return m.WaitForDev()
	}

	return nil
}

// Stop halts the Docker Machine
func (m *Machine) Stop() error {
	if m.IsRunning() {
		return util.StreamCommand("docker-machine", "stop", m.Name)
	}
	return nil
}

// Remove deleted the Docker Machine
func (m *Machine) Remove() error {
	return util.StreamCommand("docker-machine", "rm", "-y", m.Name)
}

// WaitForDev will wait a period of time for communication with the docker daemon to be established
func (m *Machine) WaitForDev() error {
	maxTries := 10
	sleepSecs := 3

	for i := 1; i <= maxTries; i++ {
		m.SetEnv()
		if err := util.Command("docker", "ps").Run(); err == nil {
			m.out.Verbose("Machine '%s' has started", m.Name)
			return nil
		}
		m.out.Warning("Docker daemon not running! Trying again in %d seconds.  Try %d of %d. \n", sleepSecs, i, maxTries)
		time.Sleep(time.Duration(sleepSecs) * time.Second)
	}

	return fmt.Errorf("docker daemon failed to start")
}

// SetEnv will set the Docker proxy variables that determine which machine the docker command communicates
func (m *Machine) SetEnv() {
	if js := m.GetData(); js != nil {
		tlsVerify := 0
		if js.Get("HostOptions").Get("EngineOptions").Get("TlsVerify").MustBool() {
			tlsVerify = 1
		}
		os.Setenv("DOCKER_TLS_VERIFY", fmt.Sprintf("%d", tlsVerify))
		os.Setenv("DOCKER_HOST", fmt.Sprintf("tcp://%s:2376", js.Get("Driver").Get("IPAddress").MustString()))
		os.Setenv("DOCKER_MACHINE_NAME", js.Get("Driver").Get("MachineName").MustString())
		os.Setenv("DOCKER_CERT_PATH", js.Get("HostOptions").Get("AuthOptions").Get("StorePath").MustString())
	}
}

// UnsetEnv will remove the Docker proxy variables
func (m *Machine) UnsetEnv() {
	os.Unsetenv("DOCKER_TLS_VERIFY")
	os.Unsetenv("DOCKER_HOST")
	os.Unsetenv("DOCKER_CERT_PATH")
	os.Unsetenv("DOCKER_MACHINE_NAME")
}

// Exists determines if the Docker Machine exist
func (m *Machine) Exists() bool {
	if err := util.Command("docker-machine", "status", m.Name).Run(); err != nil {
		return false
	}
	return true
}

// IsRunning returns the Docker Machine running status
func (m *Machine) IsRunning() bool {
	if err := util.Command("docker-machine", "env", m.Name).Run(); err != nil {
		return false
	}
	return true
}

// GetData will inspect the Docker Machine and return the parsed JSON describing the machine
func (m *Machine) GetData() *simplejson.Json {
	if m.inspectData != nil {
		return m.inspectData
	}

	if inspect, inspectErr := util.Command("docker-machine", "inspect", m.Name).Output(); inspectErr == nil {
		if js, jsonErr := simplejson.NewJson(inspect); jsonErr != nil {
			m.out.Channel.Error.Fatalf("Failed to parse '%s' JSON: %s", m.Name, jsonErr)
		} else {
			m.inspectData = js
			return m.inspectData
		}
	}
	return nil
}

// GetIP returns the IP address for the Docker Machine
func (m *Machine) GetIP() string {
	return m.GetData().Get("Driver").Get("IPAddress").MustString()
}

// GetHostDNSResolver checks if the VirtualBox host DNS resolver is working. This should work okay
// for VMware or other machines without the option, too.
func (m *Machine) GetHostDNSResolver() bool {
	return m.GetData().Get("Driver").Get("HostDNSResolver").MustBool(false)
}

// GetBridgeIP returns the Bridge IP by looking for a bip= option
func (m *Machine) GetBridgeIP() string {
	ip := "172.17.0.1"
	r := regexp.MustCompile("bip=([0-9.]+)/[0-9+]")
	var matches []string

	options := m.GetData().Get("HostOptions").Get("EngineOptions").Get("ArbitraryFlags").MustArray()

	for _, option := range options {
		matches = r.FindStringSubmatch(option.(string))
		if len(matches) > 1 {
			ip = matches[1]
		}
	}

	return ip
}

// GetDockerVersion returns the Version of Docker running within Docker Machine
func (m *Machine) GetDockerVersion() (*version.Version, error) {
	b2dOutput, err := util.Command("docker-machine", "version", m.Name).CombinedOutput()
	if err != nil {
		return nil, errors.New(strings.TrimSpace(string(b2dOutput)))
	}
	b2dVersion := strings.TrimSpace(string(b2dOutput))
	return version.Must(version.NewVersion(b2dVersion)), nil
}

// GetDriver returns the virtualization driver name
func (m *Machine) GetDriver() string {
	return m.GetData().Get("DriverName").MustString()
}

// IsXhyve returns if the virt driver is xhyve
func (m *Machine) IsXhyve() bool {
	return m.GetDriver() == util.Xhyve
}

// GetCPU returns the number of configured CPU for this Docker Machine
func (m *Machine) GetCPU() int {
	return m.GetData().Get("Driver").Get("CPU").MustInt()
}

// GetMemory returns the amount of configured memory for this Docker Machine
func (m *Machine) GetMemory() int {
	return m.GetData().Get("Driver").Get("Memory").MustInt()
}

// GetDisk returns the disk size in MB
func (m *Machine) GetDisk() int {
	return m.GetData().Get("Driver").Get("DiskSize").MustInt()
}

// GetDiskInGB returns the disk size in GB
func (m *Machine) GetDiskInGB() int {
	return m.GetDisk() / 1000
}

// GetSysctl returns the configured value for the provided sysctl setting on the Docker Machine
func (m *Machine) GetSysctl(setting string) (string, error) {
	output, err := util.Command("docker-machine", "ssh", m.Name, "sudo", "sysctl", "-n", setting).CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// SetSysctl sets the sysctl setting on the Docker Machine
func (m *Machine) SetSysctl(key string, val string) error {
	cmd := fmt.Sprintf("sudo sysctl -w %s=%s", key, val)
	m.out.Verbose("Modifying Docker Machine kernel settings: %s", cmd)
	_, err := util.Command("docker-machine", "ssh", m.Name, cmd).CombinedOutput()
	return err
}
