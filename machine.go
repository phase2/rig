package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"errors"
	"github.com/bitly/go-simplejson"
	"github.com/hashicorp/go-version"
)

type Machine struct {
	Name        string
	inspectData *simplejson.Json
}

func (m *Machine) Create(driver string, cpuCount string, memSize string, diskSize string) {
	out.Info.Printf("Creating a %s machine named '%s' with CPU(%s) MEM(%s) DISK(%s)", driver, m.Name, cpuCount, memSize, diskSize)

	boot2dockerUrl := "https://github.com/boot2docker/boot2docker/releases/download/v" + GetCurrentDockerVersion().String() + "/boot2docker.iso"

	var create *exec.Cmd

	switch driver {
	case "virtualbox":
		create = exec.Command(
			"docker-machine",
			"create", m.Name,
			"--driver=virtualbox",
			"--virtualbox-boot2docker-url="+boot2dockerUrl,
			"--virtualbox-memory="+memSize,
			"--virtualbox-cpu-count="+cpuCount,
			"--virtualbox-disk-size="+diskSize,
			"--virtualbox-host-dns-resolver=true",
			"--engine-opt", "dns=172.17.0.1",
		)
	case "vmwarefusion":
		create = exec.Command(
			"docker-machine",
			"create", m.Name,
			"--driver=vmwarefusion",
			"--vmwarefusion-boot2docker-url="+boot2dockerUrl,
			"--vmwarefusion-memory-size="+memSize,
			"--vmwarefusion-cpu-count="+cpuCount,
			"--vmwarefusion-disk-size="+diskSize,
			"--engine-opt", "dns=172.17.0.1",
		)
	case "xhyve":
		m.CheckXhyveRequirements()
		create = exec.Command(
			"docker-machine",
			"create", m.Name,
			"--driver=xhyve",
			"--xhyve-boot2docker-url="+boot2dockerUrl,
			"--xhyve-memory-size="+memSize,
			"--xhyve-cpu-count="+cpuCount,
			"--xhyve-disk-size="+diskSize,
			"--engine-opt", "dns=172.17.0.1",
		)
	}

	if err := StreamCommand(create); err != nil {
		out.Error.Fatalf("Error creating machine '%s': %s", m.Name, err)
	}

	out.Info.Printf("Created docker-machine named '%s'...", m.Name)
}

func (m Machine) CheckXhyveRequirements() {
	// Is xhyve installed locally
	if err := exec.Command("which", "xhyve").Run(); err != nil {
		out.Error.Fatal("xhyve is not installed. Install it with 'brew install xhyve'")
	}

	// Is docker-machine-driver-xhyve installed locally
	if err := exec.Command("which", "docker-machine-driver-xhyve").Run(); err != nil {
		out.Error.Fatal("docker-machine-driver-xhyve is not installed. Install it with 'brew install docker-machine-driver-xhyve'")
	}
}

func (m Machine) Start() {
	if !m.IsRunning() {
		out.Info.Printf("The machine '%s' is not running, starting...", m.Name)

		if err := StreamCommand(exec.Command("docker-machine", "start", m.Name)); err != nil {
			out.Error.Fatalf("Error starting machine '%s': %s", m.Name, err)
		}

		m.WaitForDev()
	}
}

func (m Machine) Stop() {
	StreamCommand(exec.Command("docker-machine", "stop", m.Name))
}

func (m Machine) Remove() {
	StreamCommand(exec.Command("docker-machine", "rm", "-y", m.Name))
}

// Wait a period of time for communication with the docker daemon to be established
func (m Machine) WaitForDev() {
	maxTries := 10
	sleepSecs := 3

	for i := 1; i <= maxTries; i++ {
		m.SetEnv()
		if err := exec.Command("docker", "ps").Run(); err == nil {
			out.Verbose.Printf("Machine '%s' has started", m.Name)
			return
		} else {
			out.Warning.Printf("Docker daemon not running! Trying again in %d seconds.  Try %d of %d. \n", sleepSecs, i, maxTries)
			time.Sleep(time.Duration(sleepSecs) * time.Second)
		}
	}
	out.Error.Fatal("Docker daemon failed to start!")
}

// Set the Docker proxy variables that determine which machine the docker command communicates
func (m Machine) SetEnv() {
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

// Remove the Docker proxy variables
func (m Machine) UnsetEnv() {
	os.Unsetenv("DOCKER_TLS_VERIFY")
	os.Unsetenv("DOCKER_HOST")
	os.Unsetenv("DOCKER_CERT_PATH")
	os.Unsetenv("DOCKER_MACHINE_NAME")
}

// Does the Docker Machine exist
func (m Machine) Exists() bool {
	if err := exec.Command("docker-machine", "status", m.Name).Run(); err != nil {
		return false
	}
	return true
}

// Is the Docker Machine running
func (m Machine) IsRunning() bool {
	if err := exec.Command("docker-machine", "env", m.Name).Run(); err != nil {
		return false
	}
	return true
}

// Inspect the Docker Machine and return the parsed JSON describing the machine
func (m *Machine) GetData() *simplejson.Json {
	if m.inspectData != nil {
		return m.inspectData
	}

	if inspect, inspectErr := exec.Command("docker-machine", "inspect", m.Name).Output(); inspectErr == nil {
		if js, jsonErr := simplejson.NewJson(inspect); jsonErr != nil {
			out.Error.Fatalf("Failed to parse '%s' JSON: %s", m.Name, jsonErr)
		} else {
			m.inspectData = js
			return m.inspectData
		}
	}
	return nil
}

// Return the IP address for the Docker Machine
func (m Machine) GetIP() string {
	return m.GetData().Get("Driver").Get("IPAddress").MustString()
}

// Check if the VirtualBox host DNS resolver is working. This should work okay
// for VMware or other machines without the option, too.
func (m Machine) GetHostDNSResolver() bool {
	return m.GetData().Get("Driver").Get("HostDNSResolver").MustBool(false)
}

// Return the Bridge IP by looking for a bip= option
func (m Machine) GetBridgeIP() string {
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

func (m Machine) GetDockerVersion() (*version.Version, error) {
	if b2dOutput, err := exec.Command("docker-machine", "version", m.Name).CombinedOutput(); err == nil {
		b2dVersion := strings.TrimSpace(string(b2dOutput))
		return version.Must(version.NewVersion(b2dVersion)), nil
	} else {
		return nil, errors.New(strings.TrimSpace(string(b2dOutput)))
	}
}

func (m Machine) GetDriver() string {
	return m.GetData().Get("DriverName").MustString()
}

func (m Machine) IsXhyve() bool {
	return m.GetDriver() == "xhyve"
}

func (m Machine) GetCPU() int {
	return m.GetData().Get("Driver").Get("CPU").MustInt()
}

func (m Machine) GetMemory() int {
	return m.GetData().Get("Driver").Get("Memory").MustInt()
}

// Returns the disk size in MB
func (m Machine) GetDisk() int {
	return m.GetData().Get("Driver").Get("DiskSize").MustInt()
}

// Returns the disk size in GB
func (m Machine) GetDiskInGB() int {
	return m.GetDisk() / 1000
}
