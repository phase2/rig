package util

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/go-version"
	"github.com/kardianos/osext"
	"io/ioutil"
)

// Set up the output streams (and colors) to stream command output
func StreamCommand(cmd *exec.Cmd) error {
	cmd.Stderr = os.Stderr
	if Logger().IsVerbose {
		cmd.Stdout = os.Stdout
	} else {
		cmd.Stdout = ioutil.Discard
	}

	color.Set(color.FgCyan)
	err := cmd.Run()
	color.Unset()
	return err
}

// Get the directory of this binary
func GetExecutableDir() (string, error) {
	return osext.ExecutableFolder()
}

// Ask the user a yes/no question
// Return true if they answered yes, false otherwise
func AskYesNo(question string) bool {

	fmt.Printf("%s? [y/N]: ", question)

	var response string
	fmt.Scanln(&response)

	yesResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	for _, elem := range yesResponses {
		if elem == response {
			return true
		}
	}
	return false

}

func GetCurrentDockerVersion() *version.Version {
	output, _ := exec.Command("docker", "--version").Output()
	re := regexp.MustCompile("Docker version ([\\d|\\.]+)")
	versionNumber := re.FindAllStringSubmatch(string(output), -1)[0][1]
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
