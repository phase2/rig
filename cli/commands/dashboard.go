package commands

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Dashboard struct {
	BaseCommand
}

func (cmd *Dashboard) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:  "dashboard",
			Usage: "Start Dashboard services on the docker-machine",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

func (cmd *Dashboard) Run(ctx *cli.Context) error {
	if cmd.machine.IsRunning() {
		cmd.out.Info.Println("Launching Dashboard")
		cmd.LaunchDashboard(cmd.machine)
	} else {
		cmd.out.Error.Fatalf("Machine '%s' is not running.", cmd.machine.Name)
	}

	return nil
}

func (cmd *Dashboard) LaunchDashboard(machine Machine) {
	machine.SetEnv()

	exec.Command("docker", "stop", "outrigger-dashboard").Run()
	exec.Command("docker", "rm", "outrigger-dashboard").Run()

	image := "outrigger/dashboard:latest"

	// The check for whether the image is older than 30 days is not currently used.
	_, seconds, err := util.ImageOlderThan(image, 86400*30)
	if err == nil {
		cmd.out.Verbose.Printf("Local copy of the image '%s' was originally published %0.2f days ago.", image, seconds/86400)
	}

	// If there was an error it implies no previous instance of the image is available
	// or that docker operations failed and things will likely go wrong anyway.
	if err == nil {
		cmd.out.Verbose.Printf("Attempting to update %s", image)
		if err := util.StreamCommand(exec.Command("docker", "pull", image)); err != nil {
			cmd.out.Verbose.Println("Failed to update dashboard image. Will use local cache if available.")
		}
	}

	dockerApiVersion, _ := util.GetDockerServerApiVersion(cmd.machine.Name)
	args := []string{
		"run",
		"-d",
		"--restart=always",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-l", "com.dnsdock.name=dashboard",
		"-l", "com.dnsdock.image=outrigger",
		"-e", fmt.Sprintf("DOCKER_API_VERSION=%s", dockerApiVersion),
		"--name", "outrigger-dashboard",
		"outrigger/dashboard:latest",
	}

	util.ForceStreamCommand(exec.Command("docker", args...))

	if runtime.GOOS == "darwin" {
		exec.Command("open", "http://dashboard.outrigger.vm").Run()
	} else if runtime.GOOS == "windows" {
		exec.Command("start", "http://dashboard.outrigger.vm").Run()
	} else {
		cmd.out.Info.Println("Outrigger Dashboard is now available at http://dashboard.outrigger.vm")
	}
}
