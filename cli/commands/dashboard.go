package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Dashboard struct {
	BaseCommand
}

func (cmd *Dashboard) Commands() cli.Command {
	return cli.Command{
		Name:   "dashboard",
		Usage:  "Start Dashboard services on the docker-machine",
		Before: cmd.Before,
		Action: cmd.Run,
	}
}

func (cmd *Dashboard) Run(c *cli.Context) error {
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

	home := os.Getenv("HOME")

	exec.Command("docker", "stop", "outrigger-dashboard").Run()
	exec.Command("docker", "rm", "outrigger-dashboard").Run()

	dockerApiVersion, _ := util.GetDockerServerApiVersion(cmd.machine.Name)

	args := []string{
		"run",
		"-d",
		"--restart=always",
		"-v", fmt.Sprintf("%s:%s", home, home),
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
