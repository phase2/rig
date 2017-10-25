package util

import (
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
)

func GetRawCurrentDockerVersion() string {
	output, _ := exec.Command("docker", "--version").Output()
	re := regexp.MustCompile("Docker version (.*),")
	return re.FindAllStringSubmatch(string(output), -1)[0][1]
}

func GetCurrentDockerVersion() *version.Version {
	versionNumber := GetRawCurrentDockerVersion()
	return version.Must(version.NewVersion(versionNumber))
}

func GetDockerClientApiVersion() *version.Version {
	output, _ := exec.Command("docker", "version", "--format", "{{.Client.APIVersion}}").Output()
	re := regexp.MustCompile("^([\\d|\\.]+)")
	versionNumber := re.FindAllStringSubmatch(string(output), -1)[0][1]
	return version.Must(version.NewVersion(versionNumber))
}

func GetDockerServerApiVersion(name string) (*version.Version, error) {
	output, err := exec.Command("docker-machine", "ssh", name, "docker version --format {{.Server.APIVersion}}").Output()
	if err != nil {
		return nil, err
	}
	return version.Must(version.NewVersion(strings.TrimSpace(string(output)))), nil
}

func GetDockerServerMinApiVersion(name string) (*version.Version, error) {
	output, err := exec.Command("docker-machine", "ssh", name, "docker version --format {{.Server.MinAPIVersion}}").Output()
	if err != nil {
		return nil, err
	}
	return version.Must(version.NewVersion(strings.TrimSpace(string(output)))), nil
}

// Determine the age of the Docker Image and whether the image is older than the designated timestamp.
func ImageOlderThan(image string, elapsed_seconds float64) (bool, float64, error) {
	output, err := exec.Command("docker", "inspect", "--format", "{{.Created}}", image).Output()
	if err != nil {
		return false, 0, err
	}

	datestring := strings.TrimSpace(string(output))
	datetime, err := time.Parse(time.RFC3339, datestring)
	if err != nil {
		return false, 0, err
	}

	seconds := time.Since(datetime).Seconds()
	return seconds > elapsed_seconds, seconds, nil
}
