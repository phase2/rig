package commands

import (
	"os/exec"
	"runtime"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

type Stop struct {
	BaseCommand
}

func (cmd *Stop) Commands() cli.Command {
	return cli.Command{
		Name:    "stop",
		Aliases: []string{"halt"},
		Usage:   "Stop the docker-machine",
		Before:  cmd.Before,
		Action:  cmd.Run,
	}
}

func (cmd *Stop) Run(c *cli.Context) error {
	cmd.out.Info.Printf("Stopping machine '%s'", cmd.machine.Name)
	cmd.machine.Stop()

	cmd.out.Info.Println("Cleaning up local networking (may require your admin password)")
	if runtime.GOOS == "windows" {
		exec.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.0.0").Run()
		exec.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.42.1").Run()
	} else {
		exec.Command("sudo", "route", "-n", "delete", "-net", "172.17.0.0").Run()
		exec.Command("sudo", "route", "-n", "delete", "-net", "172.17.42.1").Run()
	}
	color.Unset()

	return nil
}
