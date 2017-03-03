package commands

import (
	"os/exec"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Kill struct {
	BaseCommand
}

func (cmd *Kill) Commands() cli.Command {
	return cli.Command{
		Name:   "kill",
		Usage:  "Kill the docker-machine",
		Before: cmd.Before,
		Action: cmd.Run,
	}
}

func (cmd *Kill) Run(c *cli.Context) error {
	if !cmd.machine.Exists() {
		cmd.out.Error.Fatalf("No machine named '%s' exists.", cmd.machine.Name)
	}

	// First stop it (and cleanup)
	stop := Stop{BaseCommand{machine: cmd.machine, out: cmd.out}}
	stop.Run(c)

	cmd.out.Info.Printf("Killing machine '%s'", cmd.machine.Name)
	util.StreamCommand(exec.Command("docker-machine", "kill", cmd.machine.Name))

	// Ensure the underlying virtualization has stopped
	driver := cmd.machine.GetDriver()
	switch driver {
	case "virtualbox":
		util.StreamCommand(exec.Command("controlvm", cmd.machine.Name, "poweroff"))
	case "vmwarefusion":
		cmd.out.Warning.Println("Add vmrun suspend command.")
	case "xhyve":
		cmd.out.Warning.Println("Add equivalent xhyve kill command.")
	default:
		cmd.out.Warning.Printf("Driver not recognized: %s\n", driver)
	}

	return nil
}
