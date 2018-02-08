package commands

import (
	"fmt"

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
	cmd.out.Channel.Verbose.Printf("Skipping Step: Linux does not have a docker-machine to stop.")
	cmd.out.Channel.Verbose.Printf("Skipping Step: Outrigger does not manage Linux networking.")

	dash := Dashboard{cmd.BaseCommand}
	dash.StopDashboard()

	dns := DNS{cmd.BaseCommand}
	dns.StopDNS()

	return cmd.Success("")
}

// StopOutrigger will halt all Outrigger and Docker-related operations.
func (cmd *Stop) StopOutrigger() error {
	cmd.out.Spin(fmt.Sprintf("Stopping machine '%s'...", cmd.machine.Name))
	if err := cmd.machine.Stop(); err != nil {
		return cmd.Failure(err.Error(), "MACHINE-STOP-FAILED", 12)
	}
	cmd.out.Info("Stopped machine '%s'", cmd.machine.Name)

	cmd.out.Spin("Cleaning up local networking...")
	if util.IsWindows() {
		util.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.0.0").Run()
		util.Command("runas", "/noprofile", "/user:Administrator", "route", "DELETE", "172.17.42.1").Run()
	} else {
		util.EscalatePrivilege()
		util.Command("sudo", "route", "-n", "delete", "-net", "172.17.0.0").Run()
		util.Command("sudo", "route", "-n", "delete", "-net", "172.17.42.1").Run()
	}
	cmd.out.Info("Networking cleanup completed")

	return cmd.Success(fmt.Sprintf("Machine '%s' stopped", cmd.machine.Name))
}
