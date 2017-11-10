package util

import (
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
)

// GetRawCurrentDockerVersion returns the entire semver string from the docker version cli
func GetRawCurrentDockerVersion() string {
	output, _ := Command("docker", "--version").Output()
	re := regexp.MustCompile("Docker version (.*),")
	return re.FindAllStringSubmatch(string(output), -1)[0][1]
}

// GetCurrentDockerVersion returns a Version based in the Docker semver
func GetCurrentDockerVersion() *version.Version {
	versionNumber := GetRawCurrentDockerVersion()
	return version.Must(version.NewVersion(versionNumber))
}

// GetDockerClientAPIVersion returns a Version for the docker client API version
func GetDockerClientAPIVersion() *version.Version {
	output, _ := Command("docker", "version", "--format", "{{.Client.APIVersion}}").Output()
	re := regexp.MustCompile(`^([\d|\.]+)`)
	versionNumber := re.FindAllStringSubmatch(string(output), -1)[0][1]
	return version.Must(version.NewVersion(versionNumber))
}

// GetDockerServerAPIVersion returns a Version for the docker server API version
func GetDockerServerAPIVersion() (*version.Version, error) {
	output, err := Command("docker", "version", "--format", "{{.Server.APIVersion}}").Output()

	if err != nil {
		return nil, err
	}
	return version.Must(version.NewVersion(strings.TrimSpace(string(output)))), nil
}

// GetDockerServerMinAPIVersion returns the minimum compatability version for the docker server
func GetDockerServerMinAPIVersion() (*version.Version, error) {
	output, err := Command("docker", "version", "--format", "{{.Server.MinAPIVersion}}").Output()

	if err != nil {
		return nil, err
	}
	return version.Must(version.NewVersion(strings.TrimSpace(string(output)))), nil
}

// ImageOlderThan determines the age of the Docker Image and whether the image is older than the designated timestamp.
func ImageOlderThan(image string, elapsedSeconds float64) (bool, float64, error) {
	output, err := Command("docker", "inspect", "--format", "{{.Created}}", image).Output()
	if err != nil {
		return false, 0, err
	}

	datestring := strings.TrimSpace(string(output))
	datetime, err := time.Parse(time.RFC3339, datestring)
	if err != nil {
		return false, 0, err
	}

	seconds := time.Since(datetime).Seconds()
	return seconds > elapsedSeconds, seconds, nil
}

// GetBridgeIP returns the IP address of the Docker bridge network gateway
func GetBridgeIP() (string, error) {
	output, err := Command("docker", "network", "inspect", "bridge", "--format", "{{(index .IPAM.Config 0).Gateway}}").Output()
	if err != nil {
		return "", err
	}

	bip := strings.Trim(string(output), "\n")
	if bip == "" {
		bip = "172.17.0.1"
	}
	return bip, nil
}
