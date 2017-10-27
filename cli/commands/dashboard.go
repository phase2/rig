package commands

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

const (
	dashboardContainerName = "outrigger-dashboard"
	dashboardImageName     = "outrigger/dashboard:latest"
)

type Dashboard struct {
	BaseCommand
}

func (cmd *Dashboard) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "dashboard",
			Usage:  "Start Dashboard services on the docker-machine",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

func (cmd *Dashboard) Run(ctx *cli.Context) error {
	if cmd.machine.IsRunning() || util.SupportsNativeDocker() {
		cmd.out.Info.Println("Launching Dashboard")
		return cmd.LaunchDashboard(cmd.machine)
	} else {
		return cmd.Error(fmt.Sprintf("Machine '%s' is not running.", cmd.machine.Name), "MACHINE-STOPPED", 12)
	}

	return cmd.Success("")
}

// Launch the dashboard, stopping it first for a clean automatic update.
func (cmd *Dashboard) LaunchDashboard(machine Machine) error {
	machine.SetEnv()

	cmd.StopDashboard()

	// The check for whether the image is older than 30 days is not currently used,
	// except to indicate the age of the image before update in the next section.
	_, seconds, err := util.ImageOlderThan(dashboardImageName, 86400*30)
	if err == nil {
		cmd.out.Verbose.Printf("Local copy of the dashboardImageName '%s' was originally published %0.2f days ago.", dashboardImageName, seconds/86400)
	}

	cmd.out.Verbose.Printf("Attempting to update %s", dashboardImageName)
	if err := util.StreamCommand(exec.Command("docker", "pull", dashboardImageName)); err != nil {
		cmd.out.Verbose.Println("Failed to update dashboard image. Will use local cache if available.")
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
		"--name", dashboardContainerName,
		dashboardImageName,
	}

	util.ForceStreamCommand(exec.Command("docker", args...))

	if runtime.GOOS == "darwin" {
		exec.Command("open", "http://dashboard.outrigger.vm").Run()
	} else if runtime.GOOS == "windows" {
		exec.Command("start", "http://dashboard.outrigger.vm").Run()
	} else {
		cmd.out.Info.Println("Outrigger Dashboard is now available at http://dashboard.outrigger.vm")
	}

	return nil
}

// Stop and remove the dashboard Docker image.
func (cmd *Dashboard) StopDashboard() error {
	exec.Command("docker", "stop", dashboardContainerName).Run()
	exec.Command("docker", "rm", dashboardContainerName).Run()

	return nil
}
