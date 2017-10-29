package commands

import (
	"fmt"
	"os/exec"

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
			Usage:  "Kill the docker-machine",
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
		return cmd.Error(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), "MACHINE-NOT-FOUND", 12)
	}

	// First stop it (and cleanup)
	stop := Stop{BaseCommand{machine: cmd.machine, out: cmd.out}}
	if err := stop.Run(c); err != nil {
		return err
	}

	cmd.out.Info.Printf("Killing machine '%s'", cmd.machine.Name)
	util.StreamCommand(exec.Command("docker-machine", "kill", cmd.machine.Name))

	// Ensure the underlying virtualization has stopped
	driver := cmd.machine.GetDriver()
	switch driver {
	case util.VirtualBox:
		util.StreamCommand(exec.Command("controlvm", cmd.machine.Name, "poweroff"))
	case util.VMWare:
		cmd.out.Warning.Println("Add vmrun suspend command.")
	case util.Xhyve:
		cmd.out.Warning.Println("Add equivalent xhyve kill command.")
	default:
		cmd.out.Warning.Printf("Driver not recognized: %s\n", driver)
	}

	return cmd.Success(fmt.Sprintf("Machine '%s' killed", cmd.machine.Name))
}
