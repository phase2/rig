package commands

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/fatih/color"
	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Stop struct {
	BaseCommand
}

func (cmd *Stop) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:    "stop",
			Aliases: []string{"halt"},
			Usage:   "Stop the docker-machine",
			Before:  cmd.Before,
			Action:  cmd.Run,
		},
	}
}

func (cmd *Stop) Run(c *cli.Context) error {
	if util.SupportsNativeDocker() {
		return cmd.StopMinimal()
	}

	return cmd.StopOutrigger()
}

// Stop "minimal" Outrigger operations, which refers to Linux environments where
// a virtual machine and networking is not managed by Outrigger.
func (cmd *Stop) StopMinimal() error {
	cmd.out.Verbose.Printf("Skipping Step: Linux does not have a docker-machine to stop.")
	cmd.out.Verbose.Printf("Skipping Step: Outrigger does not manage Linux networking.")

	dash := Dashboard{BaseCommand{machine: cmd.machine, out: cmd.out}}
	dash.StopDashboard()

	dns := Dns{BaseCommand{machine: cmd.machine, out: cmd.out}}
	dns.StopDns()

	return cmd.Success("")
}

// Halt all Outrigger and Docker-related operations.
func (cmd *Stop) StopOutrigger() error {
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

	return cmd.Success(fmt.Sprintf("Machine '%s' stopped", cmd.machine.Name))
}
