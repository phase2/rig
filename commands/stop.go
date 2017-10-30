package commands

import (
	"fmt"
	"os/exec"

	"github.com/fatih/color"
	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// Stop is the command for shutting down the Docker Machine and core Outrigger services
type Stop struct {
	BaseCommand
}

// Commands returns the operations supported by this command
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

// Run executes the `rig stop` command
func (cmd *Stop) Run(c *cli.Context) error {
	if util.SupportsNativeDocker() {
		return cmd.StopMinimal()
	}

	return cmd.StopOutrigger()
}

// StopMinimal will stop "minimal" Outrigger operations, which refers to environments where
// a virtual machine and networking are not required or managed by Outrigger.
func (cmd *Stop) StopMinimal() error {
	cmd.out.Verbose.Printf("Skipping Step: Linux does not have a docker-machine to stop.")
	cmd.out.Verbose.Printf("Skipping Step: Outrigger does not manage Linux networking.")

	dash := Dashboard{BaseCommand{machine: cmd.machine, out: cmd.out}}
	dash.StopDashboard()

	dns := DNS{BaseCommand{machine: cmd.machine, out: cmd.out}}
	dns.StopDNS()

	return cmd.Success("")
}

// StopOutrigger will halt all Outrigger and Docker-related operations.
func (cmd *Stop) StopOutrigger() error {
	cmd.out.Info.Printf("Stopping machine '%s'", cmd.machine.Name)
	if err := cmd.machine.Stop(); err != nil {
		return cmd.Error(err.Error(), "MACHINE-STOP-FAILED", 12)
	}

	cmd.out.Info.Println("Cleaning up local networking (may require your admin password)")
	if util.IsWindows() {
		exec.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.0.0").Run()
		exec.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.42.1").Run()
	} else {
		exec.Command("sudo", "route", "-n", "delete", "-net", "172.17.0.0").Run()
		exec.Command("sudo", "route", "-n", "delete", "-net", "172.17.42.1").Run()
	}
	color.Unset()

	return cmd.Success(fmt.Sprintf("Machine '%s' stopped", cmd.machine.Name))
}
