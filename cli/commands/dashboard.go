package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/urfave/cli"
	"github.com/phase2/rig/cli/util"
)

type Dashboard struct{
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

	util.StreamCommand(exec.Command("docker", "stop", "outrigger-dashboard"))
	util.StreamCommand(exec.Command("docker", "rm", "outrigger-dashboard"))

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

	util.StreamCommand(exec.Command("docker", args...))

	if runtime.GOOS == "darwin" {
		util.StreamCommand(exec.Command("open", "http://dashboard.outrigger.vm"))
	} else if runtime.GOOS == "windows" {
		util.StreamCommand(exec.Command("start", "http://dashboard.outrigger.vm"))
	} else {
		cmd.out.Info.Println("Outrigger Dashboard is now available at http://dashboard.outrigger.vm")
	}
}
