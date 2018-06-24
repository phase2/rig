package commands

import (
	"fmt"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// Kill is the command killing a Docker Machine
type Kill struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *Kill) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "kill",
			Usage:  "Kill the docker-machine. Useful when stop does not appear to be working",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig kill` command
func (cmd *Kill) Run(c *cli.Context) error {
	if util.SupportsNativeDocker() {
		return cmd.Success("Kill is not needed on Linux")
	}

	if !cmd.machine.Exists() {
		return cmd.Failure(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), "MACHINE-NOT-FOUND", 12)
	}

	// First stop it (and cleanup)
	stop := Stop{cmd.BaseCommand}
	if err := stop.Run(c); err != nil {
		return err
	}

	cmd.out.Spin(fmt.Sprintf("Killing machine '%s'...", cmd.machine.Name))
	util.StreamCommand("docker-machine", "kill", cmd.machine.Name)

	// Ensure the underlying virtualization has stopped
	driver := cmd.machine.GetDriver()
	switch driver {
	case util.VirtualBox:
		util.StreamCommand("controlvm", cmd.machine.Name, "poweroff")
	case util.VMWare:
		cmd.out.Warning("Add vmrun suspend command.")
	case util.Xhyve:
		cmd.out.Warning("Add equivalent xhyve kill command.")
	default:
		cmd.out.Warning("Driver not recognized: %s\n", driver)
	}

	return cmd.Success(fmt.Sprintf("Machine '%s' killed", cmd.machine.Name))
}
