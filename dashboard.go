package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/urfave/cli"
)

type Dashboard struct{}

func (cmd *Dashboard) Commands() cli.Command {
	return cli.Command{
		Name:   "dashboard",
		Usage:  "Start Dashboard services on the docker-machine",
		Action: cmd.Run,
	}
}

func (cmd *Dashboard) Run(c *cli.Context) error {
	if machine.IsRunning() {
		out.Info.Println("Launching Dashboard")
		cmd.LaunchDashboard(machine)
	} else {
		out.Error.Fatalf("Machine '%s' is not running.", machine.Name)
	}

	return nil
}

func (cmd Dashboard) LaunchDashboard(machine Machine) {
	machine.SetEnv()

	home := os.Getenv("HOME")

	StreamCommand(exec.Command("docker", "stop", "outrigger-dashboard"))
	StreamCommand(exec.Command("docker", "rm", "outrigger-dashboard"))

	dockerApiVersion, _ := GetDockerServerApiVersion()

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

	StreamCommand(exec.Command("docker", args...))

	if runtime.GOOS == "darwin" {
		StreamCommand(exec.Command("open", "http://dashboard.outrigger.vm"))
	} else if runtime.GOOS == "windows" {
		StreamCommand(exec.Command("start", "http://dashboard.outrigger.vm"))
	} else {
		out.Info.Println("Outrigger Dashboard is now available at http://dashboard.outrigger.vm")
	}
}
